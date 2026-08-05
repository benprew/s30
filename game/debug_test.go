package game

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestApplyRuntimeOptionsShowsOpponentHand(t *testing.T) {
	original := interactive.RevealOpponentHand
	t.Cleanup(func() {
		interactive.RevealOpponentHand = original
	})

	applyRuntimeOptions(Options{ShowOpponentHand: true})

	if !interactive.RevealOpponentHand {
		t.Fatal("opponent hand is hidden, want revealed")
	}
}

func TestApplyRuntimeOptionsHidesOpponentHandByDefault(t *testing.T) {
	original := interactive.RevealOpponentHand
	t.Cleanup(func() {
		interactive.RevealOpponentHand = original
	})
	interactive.RevealOpponentHand = true

	applyRuntimeOptions(Options{})

	if interactive.RevealOpponentHand {
		t.Fatal("opponent hand is revealed, want hidden")
	}
}

func TestApplyDebugOptionsInstallsBurnDeckAndEnemyLifeOverride(t *testing.T) {
	player := &domain.Player{
		Character:   domain.Character{CardCollection: domain.NewCardCollection()},
		MinDeckSize: 40,
	}
	level := &world.Level{Player: player}

	if err := applyDebugOptions(level, Options{Debug: true}); err != nil {
		t.Fatalf("applyDebugOptions: %v", err)
	}

	deck := player.CardCollection.GetDeck(player.ActiveDeck)
	if got := deck[domain.FindCardByName("Lightning Bolt")]; got != 20 {
		t.Errorf("Lightning Bolt count = %d, want 20", got)
	}
	if got := deck[domain.FindCardByName("Mountain")]; got != 20 {
		t.Errorf("Mountain count = %d, want 20", got)
	}
	if len(deck) != 2 {
		t.Errorf("debug deck has %d unique cards, want 2", len(deck))
	}
	if level.EnemyStartingLife != 1 {
		t.Errorf("enemy starting life = %d, want 1", level.EnemyStartingLife)
	}
}

func TestApplyDebugOptionsDisabledLeavesGameUnchanged(t *testing.T) {
	collection := domain.NewCardCollection()
	forest := domain.FindCardByName("Forest")
	collection.AddCardToDeck(forest, 0, 10)
	player := &domain.Player{Character: domain.Character{CardCollection: collection}}
	level := &world.Level{Player: player}

	if err := applyDebugOptions(level, Options{}); err != nil {
		t.Fatalf("applyDebugOptions: %v", err)
	}

	if got := player.CardCollection.GetDeck(0)[forest]; got != 10 {
		t.Errorf("Forest count = %d, want 10", got)
	}
	if level.EnemyStartingLife != 0 {
		t.Errorf("enemy starting life = %d, want default override 0", level.EnemyStartingLife)
	}
}
