package bugreport

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/save"
)

// EnvironmentInfo captures machine and runtime environment details.
type EnvironmentInfo struct {
	OS           string  `json:"os"`
	Arch         string  `json:"arch"`
	GoVersion    string  `json:"go_version"`
	GameVersion  string  `json:"game_version"`
	ActiveScreen string  `json:"active_screen"`
	FPS          float64 `json:"fps"`
	TPS          float64 `json:"tps"`
	WindowWidth  int     `json:"window_width"`
	WindowHeight int     `json:"window_height"`
}

// DuelReportState captures the full state and history of an in-progress or recently finished duel.
type DuelReportState struct {
	OpponentName   string                 `json:"opponent_name"`
	OpponentRogue  string                 `json:"opponent_rogue,omitempty"`
	Turn           int                    `json:"turn"`
	Step           string                 `json:"step"`
	ActivePlayer   string                 `json:"active_player"`
	PriorityPlayer string                 `json:"priority_player,omitempty"`
	IsDungeonDuel  bool                   `json:"is_dungeon_duel"`
	DiceNotice     string                 `json:"dice_notice,omitempty"`
	AnteHumanCard  string                 `json:"ante_human_card,omitempty"`
	AnteAICard     string                 `json:"ante_ai_card,omitempty"`
	GameState      *interactive.GameState `json:"game_state,omitempty"`
	HumanDeck      []string               `json:"human_deck,omitempty"`
	AIDeck         []string               `json:"ai_deck,omitempty"`
	History        []string               `json:"history,omitempty"`
	EngineLogs     []string               `json:"engine_logs,omitempty"`
}

// BugReport represents a complete bug or crash report.
type BugReport struct {
	ID           string           `json:"id"`
	Timestamp    time.Time        `json:"timestamp"`
	UserNotes    string           `json:"user_notes,omitempty"`
	IsCrash      bool             `json:"is_crash,omitempty"`
	CrashMessage string           `json:"crash_message,omitempty"`
	StackTrace   string           `json:"stack_trace,omitempty"`
	Environment  EnvironmentInfo  `json:"environment"`
	ActiveScreen string           `json:"active_screen"`
	WorldState   *save.SaveData   `json:"world_state,omitempty"`
	DuelState    *DuelReportState `json:"duel_state,omitempty"`
}

// ToJSON serializes the BugReport into formatted JSON.
func (r *BugReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// IssueTitle generates a descriptive GitHub issue title.
func (r *BugReport) IssueTitle() string {
	if r.IsCrash {
		msg := r.CrashMessage
		if len(msg) > 60 {
			msg = msg[:60] + "..."
		}
		return fmt.Sprintf("[Crash] %s on %s", msg, r.ActiveScreen)
	}

	notes := strings.TrimSpace(r.UserNotes)
	if notes == "" {
		return fmt.Sprintf("[Bug] In-game report on %s screen", r.ActiveScreen)
	}

	firstLine := strings.SplitN(notes, "\n", 2)[0]
	if len(firstLine) > 60 {
		firstLine = firstLine[:60] + "..."
	}
	return fmt.Sprintf("[Bug] %s", firstLine)
}

// ToMarkdown formats the report as a comprehensive GitHub Issue Markdown body.
func (r *BugReport) ToMarkdown() string {
	var b strings.Builder

	if r.IsCrash {
		b.WriteString("## 🚨 CRASH REPORT\n\n")
		fmt.Fprintf(&b, "**Error:** `%s`\n\n", r.CrashMessage)
		if r.StackTrace != "" {
			b.WriteString("### Stack Trace\n```\n")
			b.WriteString(r.StackTrace)
			b.WriteString("\n```\n\n")
		}
	} else {
		b.WriteString("## 🐛 In-App Bug Report\n\n")
		if r.UserNotes != "" {
			b.WriteString("### User Description\n")
			b.WriteString(r.UserNotes)
			b.WriteString("\n\n")
		}
	}

	b.WriteString("### Environment\n")
	fmt.Fprintf(&b, "- **Platform:** %s/%s\n", r.Environment.OS, r.Environment.Arch)
	fmt.Fprintf(&b, "- **Active Screen:** %s\n", r.ActiveScreen)
	if r.Environment.GameVersion != "" {
		fmt.Fprintf(&b, "- **Game Version:** %s\n", r.Environment.GameVersion)
	}
	if r.Environment.FPS > 0 {
		fmt.Fprintf(&b, "- **FPS / TPS:** %.1f / %.1f\n", r.Environment.FPS, r.Environment.TPS)
	}
	fmt.Fprintf(&b, "- **Timestamp:** %s\n\n", r.Timestamp.Format(time.RFC3339))

	if r.DuelState != nil {
		b.WriteString("### Active Duel Context\n")
		fmt.Fprintf(&b, "- **Opponent:** %s\n", r.DuelState.OpponentName)
		fmt.Fprintf(&b, "- **Turn:** %d (%s)\n", r.DuelState.Turn, r.DuelState.Step)
		fmt.Fprintf(&b, "- **Active Player:** %s\n", r.DuelState.ActivePlayer)
		if r.DuelState.DiceNotice != "" {
			fmt.Fprintf(&b, "- **Dice:** %s\n", r.DuelState.DiceNotice)
		}
		if len(r.DuelState.History) > 0 {
			b.WriteString("\n<details><summary><b>Recent Duel History (click to expand)</b></summary>\n\n```\n")
			for _, line := range r.DuelState.History {
				b.WriteString(line)
				b.WriteString("\n")
			}
			b.WriteString("```\n</details>\n\n")
		}
	}

	jsonData, err := r.ToJSON()
	if err == nil {
		b.WriteString("<details><summary><b>Full State JSON (click to expand)</b></summary>\n\n```json\n")
		b.WriteString(string(jsonData))
		b.WriteString("\n```\n</details>\n")
	}

	return b.String()
}
