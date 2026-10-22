package screens

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benprew/s30/game/bugreport"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
)

type mockSubmitter struct {
	submittedReport *bugreport.BugReport
	result          *bugreport.SubmitResult
	err             error
}

type blockingBugSubmitter struct {
	reports chan *bugreport.BugReport
	release chan struct{}
}

func (s *blockingBugSubmitter) Submit(report *bugreport.BugReport) (*bugreport.SubmitResult, error) {
	s.reports <- report
	<-s.release
	return &bugreport.SubmitResult{Success: true, Message: "Sent"}, nil
}

func TestBugReportAsyncSubmission(t *testing.T) {
	level := &world.Level{Player: &domain.Player{Gold: 100}}
	scr := NewBugReportScreen(level, screenui.WorldScr, nil)
	submitter := &blockingBugSubmitter{reports: make(chan *bugreport.BugReport, 2), release: make(chan struct{})}
	defer close(submitter.release)
	scr.submitter = submitter
	scr.textInput.SetText("Original notes")
	scr.triggerSubmit()
	if !scr.submitting {
		t.Fatal("submission must start before the background task")
	}
	scr.textInput.SetText("Edited notes")
	level.Player.Gold = 200
	scr.triggerSubmit()
	select {
	case report := <-submitter.reports:
		if report.UserNotes != "Original notes" {
			t.Errorf("submitted notes = %q", report.UserNotes)
		}
		if report.WorldState.World.Player.Gold != 100 {
			t.Error("report shares mutable world state")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("submission did not start")
	}
	select {
	case <-submitter.reports:
		t.Fatal("duplicate submission started")
	default:
	}
	submitter.release <- struct{}{}
	select {
	case outcome := <-scr.submitResults:
		if !scr.submitting || scr.statusMsg != "Submitting report..." {
			t.Fatal("background task changed UI state")
		}
		scr.submitResults <- outcome
	case <-time.After(5 * time.Second):
		t.Fatal("submission did not finish")
	}
	scr.updateSubmission()
	if scr.submitting || scr.statusMsg != "Sent" {
		t.Fatalf("submission state = %v, %q", scr.submitting, scr.statusMsg)
	}
}

func (m *mockSubmitter) Submit(report *bugreport.BugReport) (*bugreport.SubmitResult, error) {
	m.submittedReport = report
	return m.result, m.err
}

func TestBugReportCancelButton(t *testing.T) {
	scr := NewBugReportScreen(nil, screenui.WorldScr, nil)
	button := scr.buttons[len(scr.buttons)-1]
	if button.ButtonText.Text != "Cancel" {
		t.Errorf("dismiss button text = %q, want Cancel", button.ButtonText.Text)
	}
}

func TestBugReportSubmissionFailureLogsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stderr.log")
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stderr
	os.Stderr = output
	t.Cleanup(func() {
		os.Stderr = previous
		_ = output.Close()
	})

	failure := errors.New("report saved locally to backup.json, but could not send: HTTP 503")
	scr := NewBugReportScreen(nil, screenui.WorldScr, nil)
	scr.submitter = &mockSubmitter{err: failure}
	if _, submitErr := scr.SubmitSync(); !errors.Is(submitErr, failure) {
		t.Fatalf("SubmitSync error = %v, want %v", submitErr, failure)
	}
	logged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logged), "Bug report submission failed: "+failure.Error()) {
		t.Errorf("stderr = %q, want submission error", logged)
	}
	if !strings.Contains(scr.statusMsg, failure.Error()) {
		t.Errorf("status = %q, want submission error", scr.statusMsg)
	}
}

func TestBugReportScreenCreationAndSubmission(t *testing.T) {
	player, err := domain.NewPlayer("Hero", nil, false, domain.DifficultyEasy, domain.ColorGreen)
	if err != nil {
		t.Fatalf("NewPlayer failed: %v", err)
	}
	level, err := world.NewLevel(player)
	if err != nil {
		t.Fatalf("NewLevel failed: %v", err)
	}

	scr := NewBugReportScreen(level, screenui.WorldScr, nil)
	if !scr.IsOverlay() {
		t.Error("BugReportScreen should be an overlay")
	}
	if scr.IsFramed() {
		t.Error("BugReportScreen should not be framed")
	}

	mock := &mockSubmitter{
		result: &bugreport.SubmitResult{
			Success:  true,
			IssueURL: "https://github.com/benprew/s30/issues/50",
			Message:  "Submitted to GitHub",
		},
	}
	scr.submitter = mock

	scr.textInput.SetText("Encountered an enemy bug on tile 20, 20")
	_, _ = scr.SubmitSync()

	if mock.submittedReport == nil {
		t.Fatal("Report was not submitted")
	}
	if mock.submittedReport.UserNotes != "Encountered an enemy bug on tile 20, 20" {
		t.Errorf("UserNotes = %q", mock.submittedReport.UserNotes)
	}
	if mock.submittedReport.ActiveScreen != "World" {
		t.Errorf("ActiveScreen = %q, want World", mock.submittedReport.ActiveScreen)
	}
	if scr.statusMsg == "" {
		t.Error("statusMsg was not set after submission")
	}
}
