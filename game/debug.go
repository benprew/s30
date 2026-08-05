package game

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

var debugBurnDeck = map[string]int{
	"Lightning Bolt": 20,
	"Mountain":       20,
}

func applyRuntimeOptions(options Options) {
	interactive.RevealOpponentHand = options.ShowOpponentHand
}

func applyDebugOptions(level *world.Level, options Options) error {
	if !options.Debug {
		return nil
	}

	collection := domain.NewCardCollection()
	for name, count := range debugBurnDeck {
		card := domain.FindCardByName(name)
		if card == nil {
			return fmt.Errorf("debug deck card %q not found", name)
		}
		collection.AddCardToDeck(card, level.Player.ActiveDeck, count)
	}
	level.Player.CardCollection = collection
	level.EnemyStartingLife = 1
	return nil
}
