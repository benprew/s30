package bugreport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBugReportSerialization(t *testing.T) {
	report := &BugReport{
		ID:        "report_2026-08-20_123456",
		Timestamp: time.Date(2026, 8, 20, 12, 34, 56, 0, time.UTC),
		UserNotes: "Lightning Bolt dealt no damage to opposing creature",
		Environment: EnvironmentInfo{
			OS:           "darwin",
			Arch:         "arm64",
			GameVersion:  "1.0.0",
			ActiveScreen: "Duel",
			FPS:          60.0,
			TPS:          60.0,
		},
		ActiveScreen: "Duel",
		DuelState: &DuelReportState{
			OpponentName: "Sea Troll",
			Turn:         3,
			Step:         "Precombat Main",
			ActivePlayer: "You",
			History: []string{
				"Turn 1: Play Mountain",
				"Turn 2: Play Mountain, Cast Ironclaw Orcs",
				"Turn 3: Cast Lightning Bolt targeting Sea Troll",
			},
		},
	}

	data, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed BugReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed.ID != report.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, report.ID)
	}
	if parsed.UserNotes != report.UserNotes {
		t.Errorf("UserNotes = %q, want %q", parsed.UserNotes, report.UserNotes)
	}
	if parsed.DuelState == nil || parsed.DuelState.OpponentName != "Sea Troll" {
		t.Errorf("DuelState.OpponentName = %v, want Sea Troll", parsed.DuelState)
	}
	if len(parsed.DuelState.History) != 3 {
		t.Errorf("History len = %d, want 3", len(parsed.DuelState.History))
	}
}

func TestBugReportMarkdownFormatting(t *testing.T) {
	report := &BugReport{
		ID:        "report_2026-08-20_123456",
		Timestamp: time.Date(2026, 8, 20, 12, 34, 56, 0, time.UTC),
		UserNotes: "Game froze when casting Fireball",
		Environment: EnvironmentInfo{
			OS:           "darwin",
			Arch:         "arm64",
			GameVersion:  "1.0.0",
			ActiveScreen: "Duel",
			FPS:          59.8,
			TPS:          60.0,
		},
		ActiveScreen: "Duel",
		DuelState: &DuelReportState{
			OpponentName: "Arzakon",
			Turn:         5,
			Step:         "Declare Attackers",
			ActivePlayer: "Arzakon",
		},
	}

	md := report.ToMarkdown()

	if !strings.Contains(md, "Game froze when casting Fireball") {
		t.Errorf("Markdown missing user notes: %s", md)
	}
	if !strings.Contains(md, "darwin/arm64") {
		t.Errorf("Markdown missing OS/Arch: %s", md)
	}
	if !strings.Contains(md, "Arzakon") {
		t.Errorf("Markdown missing opponent name: %s", md)
	}
	if !strings.Contains(md, "<details>") || !strings.Contains(md, "</details>") {
		t.Errorf("Markdown missing collapsible JSON details block: %s", md)
	}
}

func TestCrashReportMarkdownFormatting(t *testing.T) {
	report := &BugReport{
		ID:           "crash_2026-08-20_654321",
		Timestamp:    time.Date(2026, 8, 20, 12, 34, 56, 0, time.UTC),
		IsCrash:      true,
		CrashMessage: "runtime error: invalid memory address or nil pointer dereference",
		StackTrace:   "goroutine 1 [running]:\nmain.main()\n\t/src/s30/main.go:48",
		Environment: EnvironmentInfo{
			OS:           "linux",
			Arch:         "amd64",
			ActiveScreen: "World",
		},
		ActiveScreen: "World",
	}

	md := report.ToMarkdown()

	if !strings.Contains(md, "CRASH REPORT") {
		t.Errorf("Markdown missing crash header: %s", md)
	}
	if !strings.Contains(md, "invalid memory address") {
		t.Errorf("Markdown missing crash message: %s", md)
	}
	if !strings.Contains(md, "goroutine 1 [running]") {
		t.Errorf("Markdown missing stack trace: %s", md)
	}

	title := report.IssueTitle()
	if !strings.HasPrefix(title, "[Crash]") {
		t.Errorf("Crash title = %q, want prefix [Crash]", title)
	}
}
