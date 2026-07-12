package screens

import (
	"strings"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestQuestScrollOverlayFlags(t *testing.T) {
	s := NewQuestScrollScreen(&domain.Player{})
	if !s.IsFramed() {
		t.Errorf("quest scroll should be framed (HUD stays visible)")
	}
	if !s.IsOverlay() {
		t.Errorf("quest scroll should be an overlay")
	}
}

func TestQuestPanelLinesEmpty(t *testing.T) {
	s := NewQuestScrollScreen(&domain.Player{})
	lines := strings.Join(s.questPanelLines(), "\n")
	if !strings.Contains(lines, "No active quests.") {
		t.Errorf("empty quest panel should prompt for quests, got:\n%s", lines)
	}
}

func TestQuestPanelLinesActionTracker(t *testing.T) {
	p := &domain.Player{ActiveQuests: []*domain.Quest{{
		Type:          domain.QuestTypeActionTracker,
		Title:         "Burn It Down",
		Description:   "Deal damage",
		Progress:      3,
		Target:        10,
		DaysRemaining: 7,
	}}}
	lines := strings.Join(NewQuestScrollScreen(p).questPanelLines(), "\n")
	for _, want := range []string{"Burn It Down", "Progress: 3 / 10", "7 days left"} {
		if !strings.Contains(lines, want) {
			t.Errorf("quest panel missing %q, got:\n%s", want, lines)
		}
	}
}

func TestQuestScrollDismissesForPointerClick(t *testing.T) {
	if !questScrollDismissed(false, true) {
		t.Fatal("escape should dismiss the quest scroll")
	}
	if !questScrollDismissed(true, false) {
		t.Fatal("a pointer click should dismiss the quest scroll")
	}
	if questScrollDismissed(false, false) {
		t.Fatal("quest scroll should remain without input")
	}
}
