package duel

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/google/uuid"
)

func TestSoundEffectsForDuelTransition(t *testing.T) {
	deadCreature := uuid.New()
	land := uuid.New()
	creature := uuid.New()
	prev := &interactive.GameMsg{State: &interactive.GameState{
		You: interactive.PlayerState{
			Life:        20,
			Battlefield: []interactive.PermanentState{{ID: deadCreature, IsCreature: true}},
		},
		Opponent: interactive.PlayerState{Life: 20},
	}}
	cur := &interactive.GameMsg{State: &interactive.GameState{
		You: interactive.PlayerState{
			Life:        17,
			Graveyard:   []interactive.CardState{{ID: deadCreature}},
			ManaPool:    interactive.ManaPoolState{Green: 1},
			Battlefield: []interactive.PermanentState{{ID: land, IsLand: true}, {ID: creature, IsCreature: true}},
		},
		Opponent:   interactive.PlayerState{Life: 20},
		StackItems: []interactive.StackItemState{{ID: uuid.NewString(), Name: "Giant Growth"}},
	}}

	want := []gameaudio.SFX{
		gameaudio.SFXDamage,
		gameaudio.SFXCreatureDeath,
		gameaudio.SFXLandPlay,
		gameaudio.SFXSummon,
		gameaudio.SFXCast,
		gameaudio.SFXManaball,
	}
	if got := soundEffectsForDuelTransition(prev, cur); !slices.Equal(got, want) {
		t.Fatalf("soundEffectsForDuelTransition() = %v, want %v", got, want)
	}
}

func TestSoundEffectsIgnoreNonCreatureEnteringGraveyard(t *testing.T) {
	spell := uuid.New()
	prev := &interactive.GameMsg{State: &interactive.GameState{You: interactive.PlayerState{Life: 20}}}
	cur := &interactive.GameMsg{State: &interactive.GameState{
		You: interactive.PlayerState{Life: 20, Graveyard: []interactive.CardState{{ID: spell}}},
	}}
	if got := soundEffectsForDuelTransition(prev, cur); len(got) != 0 {
		t.Fatalf("soundEffectsForDuelTransition() = %v, want no sound", got)
	}
}

func TestSoundEffectsDetectCounteredSpell(t *testing.T) {
	prev := &interactive.GameMsg{State: &interactive.GameState{
		You:      interactive.PlayerState{Life: 20},
		Opponent: interactive.PlayerState{Life: 20},
		StackItems: []interactive.StackItemState{
			{ID: uuid.NewString(), Name: "Lightning Bolt"},
			{ID: uuid.NewString(), Name: "Counterspell"},
		},
	}}
	cur := &interactive.GameMsg{State: &interactive.GameState{
		You:      interactive.PlayerState{Life: 20},
		Opponent: interactive.PlayerState{Life: 20},
	}}
	if got := soundEffectsForDuelTransition(prev, cur); !slices.Contains(got, gameaudio.SFXCounter) {
		t.Fatalf("soundEffectsForDuelTransition() = %v, want counter sound", got)
	}
}
