package screens

import (
	"testing"

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

func (m *mockSubmitter) Submit(report *bugreport.BugReport) (*bugreport.SubmitResult, error) {
	m.submittedReport = report
	return m.result, m.err
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
