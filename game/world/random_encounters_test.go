package world

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestUpdateEncountersWithMultipleEncountersInTriggerRangeDoesNotPanic(t *testing.T) {
	level := createTestLevel(1, 1)
	level.TileWidth = 200
	level.TileHeight = 100
	level.Player = &domain.Player{}

	encounterTile := image.Point{X: 0, Y: 0}
	level.Player.SetLoc(level.TileToPixel(encounterTile))
	level.RandomEncounters = []RandomEncounter{
		{Tile: encounterTile, SpriteIndex: 1, TerrainType: TerrainPlains},
		{Tile: encounterTile, SpriteIndex: 1, TerrainType: TerrainPlains},
	}
	level.totalTicks = 1

	level.UpdateEncounters()
}
