package world

import (
	"encoding/json"
	"image"
	"math/rand"
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

func TestArenaEncounterSurvivesUntilCompletionAndSave(t *testing.T) {
	level := createTestLevel(1, 1)
	level.TileWidth, level.TileHeight = 200, 100
	level.Player = &domain.Player{}
	tile := image.Pt(0, 0)
	level.Player.SetLoc(level.TileToPixel(tile))
	level.RandomEncounters = []RandomEncounter{{Tile: tile, Type: EncounterArena, TerrainType: TerrainPlains}}
	level.totalTicks = 1
	level.UpdateEncounters()
	encounter, ok := level.TakeRandomEncounterDetails()
	if !ok || encounter.Type != EncounterArena || len(level.RandomEncounters) != 1 {
		t.Fatal("arena must remain on the map during its event")
	}
	data, err := json.Marshal(level.RandomEncounters)
	if err != nil {
		t.Fatal(err)
	}
	var loaded []RandomEncounter
	if err := json.Unmarshal(data, &loaded); err != nil || loaded[0].Type != EncounterArena {
		t.Fatal("save did not retain arena encounter type")
	}
	level.CompleteRandomEncounter(tile)
	if len(level.RandomEncounters) != 0 {
		t.Fatal("completed arena remains on map")
	}
}

func TestRandomEncounterTypeDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	counts := map[EncounterType]int{}
	for range 10000 {
		counts[randomEncounterType(rng)]++
	}
	if counts[EncounterArena] < 4800 || counts[EncounterArena] > 5200 || counts[EncounterLand]+counts[EncounterArena] != 10000 {
		t.Fatalf("encounter distribution = %v", counts)
	}
}
