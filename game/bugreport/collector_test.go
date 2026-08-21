package bugreport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

type mockDuelScreen struct {
	state *DuelReportState
}

func (m *mockDuelScreen) DuelReportState() *DuelReportState {
	return m.state
}

func TestCollectReportWithLevelAndDuel(t *testing.T) {
	player, err := domain.NewPlayer("Hero", nil, false, domain.DifficultyEasy, domain.ColorGreen)
	if err != nil {
		t.Fatalf("NewPlayer failed: %v", err)
	}
	level, err := world.NewLevel(player)
	if err != nil {
		t.Fatalf("NewLevel failed: %v", err)
	}
	level.SetIdentity("test_game_123", domain.DifficultyEasy, domain.ColorGreen)

	mockDuel := &mockDuelScreen{
		state: &DuelReportState{
			OpponentName: "Sorceress",
			Turn:         2,
			Step:         "Declare Blockers",
			ActivePlayer: "Hero",
		},
	}

	report := CollectReport(level, "Duel", mockDuel, "Cannot block flyer with ground creature")

	if report.ID == "" {
		t.Error("report.ID is empty")
	}
	if report.ActiveScreen != "Duel" {
		t.Errorf("ActiveScreen = %q, want Duel", report.ActiveScreen)
	}
	if report.UserNotes != "Cannot block flyer with ground creature" {
		t.Errorf("UserNotes = %q", report.UserNotes)
	}
	if report.WorldState == nil || report.WorldState.GameID != "test_game_123" {
		t.Errorf("WorldState not captured properly: %+v", report.WorldState)
	}
	if report.DuelState == nil || report.DuelState.OpponentName != "Sorceress" {
		t.Errorf("DuelState not captured properly: %+v", report.DuelState)
	}
}

func TestCollectCrashReport(t *testing.T) {
	report := CollectCrashReport(nil, "World", nil, "nil pointer dereference", []byte("goroutine 1 [running]..."))

	if !report.IsCrash {
		t.Error("IsCrash should be true")
	}
	if report.CrashMessage != "nil pointer dereference" {
		t.Errorf("CrashMessage = %q", report.CrashMessage)
	}
	if !strings.Contains(report.StackTrace, "goroutine 1") {
		t.Errorf("StackTrace = %q", report.StackTrace)
	}
	if !strings.HasPrefix(report.IssueTitle(), "[Crash]") {
		t.Errorf("IssueTitle = %q, want [Crash] prefix", report.IssueTitle())
	}
}

func TestWorkerSubmitter(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusBadRequest)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":      true,
			"issue_url":    "https://github.com/benprew/s30/issues/99",
			"issue_number": 99,
		})
	}))
	defer server.Close()

	submitter := NewWorkerSubmitter(server.URL)
	report := &BugReport{
		ID:        "rep_123",
		Timestamp: time.Now(),
		UserNotes: "Testing worker submission",
		ActiveScreen: "World",
	}

	result, err := submitter.Submit(report)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if !result.Success {
		t.Error("Expected result.Success == true")
	}
	if result.IssueURL != "https://github.com/benprew/s30/issues/99" {
		t.Errorf("IssueURL = %q, want #99", result.IssueURL)
	}
	if receivedBody["title"] == nil {
		t.Error("Server did not receive title")
	}
}

func TestLocalFileSubmitter(t *testing.T) {
	tempDir := t.TempDir()
	submitter := NewLocalFileSubmitter(tempDir, nil)

	report := &BugReport{
		ID:           "test_report_local",
		Timestamp:    time.Now(),
		UserNotes:    "Local file test",
		ActiveScreen: "Start",
	}

	result, err := submitter.Submit(report)
	if err != nil {
		t.Fatalf("LocalFileSubmitter failed: %v", err)
	}

	if result.LocalFilePath == "" {
		t.Fatal("LocalFilePath is empty")
	}
	if !filepath.IsAbs(result.LocalFilePath) {
		t.Errorf("LocalFilePath is not absolute: %s", result.LocalFilePath)
	}

	content, err := os.ReadFile(result.LocalFilePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if !strings.Contains(string(content), "Local file test") {
		t.Errorf("Saved content missing UserNotes: %s", string(content))
	}
}
