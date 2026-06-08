package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func dungeonWithTile(tile domain.DungeonTile) (*domain.Player, *DungeonScreen) {
	d := &domain.Dungeon{Grid: [][]domain.DungeonTile{{tile}}}
	p := &domain.Player{}
	p.DungeonState = &domain.DungeonState{CurrentDungeon: d, DungeonLife: 10}
	return p, &DungeonScreen{Player: p}
}

func TestOpenDiceOverlayAppliesLifeAndShowsOverlay(t *testing.T) {
	p, s := dungeonWithTile(domain.DungeonTile{
		Type: domain.DungeonTileDice,
		Dice: &domain.DiceEffect{Type: domain.DiceAdvantage, LifeMod: 3},
	})

	s.openDiceOverlay(image.Point{X: 0, Y: 0})

	if p.BonusDuelLife != 3 {
		t.Errorf("expected dice life bonus 3, got %d", p.BonusDuelLife)
	}
	if !s.overlayActive {
		t.Error("expected dice overlay to be active")
	}
	if s.overlayBody == "" {
		t.Error("expected overlay body text describing the effect")
	}
	if len(s.overlayBtns) != 1 {
		t.Errorf("expected a single continue button, got %d", len(s.overlayBtns))
	}
	if tile := p.DungeonState.CurrentDungeon.Tile(image.Point{}); tile.Type != domain.DungeonTileEmpty {
		t.Errorf("expected dice tile cleared after applying, got %v", tile.Type)
	}
}

func TestOpenDiceOverlayQueuesCardGrant(t *testing.T) {
	card := &domain.Card{CardName: "Serra Angel"}
	p, s := dungeonWithTile(domain.DungeonTile{
		Type: domain.DungeonTileDice,
		Dice: &domain.DiceEffect{Type: domain.DiceAdvantage, Card: card},
	})

	s.openDiceOverlay(image.Point{X: 0, Y: 0})

	if len(p.BonusDuelCards) != 1 || p.BonusDuelCards[0] != card {
		t.Errorf("expected card queued for next duel, got %v", p.BonusDuelCards)
	}
}
