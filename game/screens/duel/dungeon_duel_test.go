package duel

import (
	"strings"
	"testing"

	_ "github.com/benprew/mage-go/cards"
	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"
	"github.com/benprew/s30/game/domain"
)

func TestPutBonusPermanentsInPlayAddsToBattlefield(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	opp := ai.NewAIPlayer("Opp", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(human, opp)

	s := &DuelScreen{human: human, game: g}
	s.putBonusPermanentsInPlay([]*domain.Card{{CardName: "Orcish Oriflamme"}})

	bf := g.AllBattlefield()
	if len(bf) != 1 {
		t.Fatalf("expected 1 permanent on the battlefield, got %d", len(bf))
	}
	if bf[0].Name() != "Orcish Oriflamme" {
		t.Errorf("expected Orcish Oriflamme on the battlefield, got %q", bf[0].Name())
	}
	if bf[0].Controller != human.PlayerID() {
		t.Errorf("expected bonus permanent controlled by the human player, got %s want %s", bf[0].Controller, human.PlayerID())
	}
}

func TestPutBonusPermanentsInPlayProcessesStaticAbilities(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	opp := ai.NewAIPlayer("Opp", heuristic.New(ai.MidrangeWeighted))
	g := mage.NewGame(human, opp)
	bear, err := mage.CreateCard("Mesa Pegasus")
	if err != nil {
		t.Fatalf("CreateCard Mesa Pegasus: %v", err)
	}
	bearPerm := g.PutOnBattlefield(bear, human.PlayerID())

	s := &DuelScreen{human: human, game: g}
	s.putBonusPermanentsInPlay([]*domain.Card{{CardName: "Crusade"}})

	if got := bearPerm.CurrentPower(g); got != 2 {
		t.Errorf("expected Crusade to boost Mesa Pegasus power to 2, got %d", got)
	}
	if got := bearPerm.CurrentToughness(g); got != 2 {
		t.Errorf("expected Crusade to boost Mesa Pegasus toughness to 2, got %d", got)
	}
}

func TestDiceNotice(t *testing.T) {
	if got := diceNotice(0, nil); got != "" {
		t.Errorf("expected empty notice for no effects, got %q", got)
	}

	got := diceNotice(3, []*domain.Card{{CardName: "Serra Angel"}})
	if got == "" {
		t.Fatal("expected a non-empty notice")
	}
	for _, want := range []string{"+3", "Serra Angel"} {
		if !strings.Contains(got, want) {
			t.Errorf("notice %q missing %q", got, want)
		}
	}

	if got := diceNotice(-2, nil); !strings.Contains(got, "-2") {
		t.Errorf("disadvantage notice %q should mention -2", got)
	}
}
