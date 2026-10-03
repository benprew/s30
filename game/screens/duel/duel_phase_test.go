package duel

import (
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"image"
	"testing"
)

func TestPhaseIndex(t *testing.T) {
	tests := []struct {
		step string
		want int
	}{
		{step: "Untap", want: 0},
		{step: "Upkeep", want: 1},
		{step: "Draw", want: 2},
		{step: stepPrecombatMain, want: 3},
		{step: stepBeginCombat, want: 4},
		{step: stepDeclareAttackers, want: 4},
		{step: stepDeclareBlockers, want: 4},
		{step: stepFirstStrikeDamage, want: 4},
		{step: stepCombatDamage, want: 4},
		{step: stepEndOfCombat, want: 4},
		{step: "Postcombat Main", want: 5},
		{step: "Cleanup", want: 6},
		{step: "End Step", want: 7},
		{step: "", want: -1},
		{step: "Unknown", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.step, func(t *testing.T) {
			if got := phaseIndex(tt.step); got != tt.want {
				t.Errorf("phaseIndex(%q) = %d, want %d", tt.step, got, tt.want)
			}
		})
	}
}

func TestPhaseOverlay(t *testing.T) {
	tests := []struct {
		name       string
		idx        int
		isPlayer   bool
		wantSlot   int
		wantBounds image.Rectangle
	}{
		{
			name:       "opponent untap",
			idx:        0,
			wantSlot:   0,
			wantBounds: image.Rect(44, 2, 79, 42),
		},
		{
			name:       "opponent end step",
			idx:        7,
			wantSlot:   7,
			wantBounds: image.Rect(44, 289, 79, 329),
		},
		{
			name:       "player untap",
			idx:        0,
			isPlayer:   true,
			wantSlot:   8,
			wantBounds: image.Rect(44, 431, 79, 471),
		},
		{
			name:       "player end step",
			idx:        7,
			isPlayer:   true,
			wantSlot:   15,
			wantBounds: image.Rect(44, 718, 79, 758),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := phaseOverlaySlot(tt.idx, tt.isPlayer); got != tt.wantSlot {
				t.Errorf("phaseOverlaySlot(%d, %t) = %d, want %d", tt.idx, tt.isPlayer, got, tt.wantSlot)
			}
			if got := phaseOverlayBounds(tt.idx, tt.isPlayer); got != tt.wantBounds {
				t.Errorf("phaseOverlayBounds(%d, %t) = %v, want %v", tt.idx, tt.isPlayer, got, tt.wantBounds)
			}
		})
	}
}

func TestPhaseStopClicks(t *testing.T) {
	s := &DuelScreen{human: interactive.NewHumanPlayer("You")}
	s.human.EnablePhaseStops()
	for _, own := range []bool{false, true} {
		for idx := range phaseCount {
			bounds := phaseButtonBounds(idx, own)
			p := bounds.Min.Add(image.Pt(5, 5))
			before := s.human.PhaseStop(phaseSteps[idx][0], own)
			if !s.handlePhaseClick(p.X, p.Y) {
				t.Fatal("phase click was not handled")
			}
			if s.human.PhaseStop(phaseSteps[idx][0], own) == before {
				t.Fatal("phase stop did not toggle")
			}
			s.handlePhaseClick(p.X, p.Y)
			if s.human.PhaseStop(phaseSteps[idx][0], own) != before {
				t.Fatal("second click did not restore stop")
			}
		}
	}
	if s.handlePhaseClick(400, 400) {
		t.Fatal("click outside phase panel was handled")
	}
}
