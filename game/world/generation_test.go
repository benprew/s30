package world

import (
	"fmt"
	"image"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestGenerateTerrainBandsIsDeterministic(t *testing.T) {
	first := generateTerrainBands(47, 63, 12345)
	second := generateTerrainBands(47, 63, 12345)

	if !reflect.DeepEqual(first, second) {
		t.Fatal("the same seed produced different terrain")
	}
}

func TestGenerateTerrainBandsKeepsWaterBorder(t *testing.T) {
	bands := generateTerrainBands(47, 63, 12345)
	for y := range bands {
		for x := range bands[y] {
			if x >= 2 && y >= 2 && x < len(bands[y])-2 && y < len(bands)-2 {
				continue
			}
			if bands[y][x] != TerrainBandWater {
				t.Errorf("border tile (%d,%d) = %v, want water", x, y, bands[y][x])
			}
		}
	}
}

func TestGenerateTerrainBandsKeepsOneLandmass(t *testing.T) {
	bands := generateTerrainBands(47, 63, 12345)
	want := countLandBands(bands)
	if want == 0 {
		t.Fatal("generated map contains no land")
	}

	start, ok := firstLandBand(bands)
	if !ok {
		t.Fatal("generated map contains no land")
	}
	got := reachableLandBands(bands, start)
	if got != want {
		t.Fatalf("reachable land = %d, total land = %d", got, want)
	}
}

func TestGenerateTerrainBandsIncludesCastleBiomes(t *testing.T) {
	for seed := range int64(20) {
		bands := generateTerrainBands(47, 63, seed)
		wanted := map[int]bool{
			TerrainSand:      false,
			TerrainMarsh:     false,
			TerrainPlains:    false,
			TerrainForest:    false,
			TerrainMountains: false,
		}
		for _, row := range bands {
			for _, band := range row {
				if _, ok := wanted[band.properties().terrain]; ok {
					wanted[band.properties().terrain] = true
				}
			}
		}
		for terrain, found := range wanted {
			if !found {
				t.Errorf("seed %d generated no terrain type %d", seed, terrain)
			}
		}
	}
}

func TestTerrainDecorationPlanUsesStableTwoObjects(t *testing.T) {
	first := terrainDecorationPlan(12345, image.Point{X: 13, Y: 17}, TerrainBandForest, 206)
	second := terrainDecorationPlan(12345, image.Point{X: 13, Y: 17}, TerrainBandForest, 206)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("decoration plan changed: first=%v second=%v", first, second)
	}
	if len(first) != 2 {
		t.Fatalf("decoration count = %d, want 2", len(first))
	}
	if first[0].Variant < 0 || first[0].Variant >= 11 || first[1].Variant < 0 || first[1].Variant >= 11 {
		t.Fatalf("invalid variants: %v", first)
	}
	if first[0].OffsetX == first[1].OffsetX {
		t.Fatalf("objects should be placed on different sides: %v", first)
	}
}

func TestSandDecorationsStayCentered(t *testing.T) {
	decorations := terrainDecorationPlan(12345, image.Point{X: 13, Y: 17}, TerrainBandSand, 206)
	if len(decorations) != 2 {
		t.Fatalf("decoration count = %d, want 2", len(decorations))
	}
	if decorations[0].OffsetX != 0 || decorations[1].OffsetX != 0 || decorations[0].OffsetY != decorations[1].OffsetY {
		t.Fatalf("sand decorations are not centered: %v", decorations)
	}
}

func TestTerrainBandProperties(t *testing.T) {
	tests := []struct {
		band      TerrainBand
		terrain   int
		secondary bool
		column    int
		base      terrainBase
	}{
		{TerrainBandWater, TerrainWater, false, -1, terrainBaseWater},
		{TerrainBandSand, TerrainSand, false, 1, terrainBaseSand},
		{TerrainBandSandMarsh, TerrainSand, true, 0, terrainBaseWater},
		{TerrainBandMarsh, TerrainMarsh, false, 0, terrainBaseMarsh},
		{TerrainBandMarshPlains, TerrainPlains, true, 1, terrainBasePlains},
		{TerrainBandPlains, TerrainPlains, false, 4, terrainBasePlains},
		{TerrainBandPlainsForest, TerrainPlains, true, 2, terrainBasePlains},
		{TerrainBandForest, TerrainForest, false, 2, terrainBasePlains},
		{TerrainBandForestMountains, TerrainForest, true, 3, terrainBasePlains},
		{TerrainBandMountains, TerrainMountains, false, 3, terrainBasePlains},
	}

	for _, tt := range tests {
		props := tt.band.properties()
		if props.terrain != tt.terrain || props.secondary != tt.secondary || props.column != tt.column || props.base != tt.base {
			t.Errorf("%v properties = %+v", tt.band, props)
		}
	}
}

func TestLandmarkPlacementIsDeterministic(t *testing.T) {
	const seed = int64(98765)
	first := generatedLandmarkSnapshot(seed)
	second := generatedLandmarkSnapshot(seed)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("landmarks differ for the same seed:\nfirst: %v\nsecond: %v", first, second)
	}
}

func generatedLandmarkSnapshot(seed int64) []string {
	level := createTestLevel(47, 63)
	level.GenerationSeed = seed
	validLocations := level.mapTerrainTypes(generateTerrainBands(level.W, level.H, seed))
	level.placeCastles(seed+1, nil, nil, nil, nil, nil)
	level.placeCities(validLocations, nil, 35, 6, rand.New(rand.NewSource(seed+2)))
	level.placeDungeons(5, 3, seed+3, nil)

	snapshot := make([]string, 0, len(level.Castles)+len(level.Dungeons)+35)
	for _, castle := range level.Castles {
		snapshot = append(snapshot, fmt.Sprintf("castle:%d:%d:%d", castle.Color, castle.MapTile.X, castle.MapTile.Y))
	}
	for y := 0; y < level.H; y++ {
		for x := 0; x < level.W; x++ {
			if city := level.Tile(image.Point{X: x, Y: y}).City; city != nil {
				snapshot = append(snapshot, fmt.Sprintf("city:%d:%d:%s:%d", x, y, city.Name, city.AmuletColor))
			}
		}
	}
	for _, dungeon := range level.Dungeons {
		snapshot = append(snapshot, fmt.Sprintf("dungeon:%d:%d:%d", dungeon.MapTile.X, dungeon.MapTile.Y, dungeon.Difficulty))
	}
	sort.Strings(snapshot)
	return snapshot
}

func TestNewLevelWithSeedBuildsTerrainObjectsAndTransitions(t *testing.T) {
	level, err := NewLevelWithSeed(&domain.Player{}, 12345)
	if err != nil {
		t.Fatal(err)
	}
	if len(level.Castles) != 5 || len(level.Dungeons) != 5 {
		t.Fatalf("landmark counts: castles=%d dungeons=%d", len(level.Castles), len(level.Dungeons))
	}

	var cities, roads, transitions, decoratedTiles int
	for y := 0; y < level.H; y++ {
		for x := 0; x < level.W; x++ {
			position := image.Point{X: x, Y: y}
			tile := level.Tile(position)
			if tile.IsCity() {
				cities++
			}
			if tile.IsRoad() {
				roads++
			}
			if len(tile.transitionSprites) > 0 {
				transitions++
			}
			wantTransitionSprites := expectedTerrainTransitionCount(level, position)
			if len(tile.transitionSprites) != wantTransitionSprites {
				t.Fatalf("tile %v has %d transition sprites, want %d", position, len(tile.transitionSprites), wantTransitionSprites)
			}
			if len(tile.terrainSprites) == 4 {
				decoratedTiles++
			}
		}
	}
	if cities != 35 || roads == 0 || transitions == 0 || decoratedTiles == 0 {
		t.Fatalf("generated features: cities=%d roads=%d transitions=%d decorated=%d", cities, roads, transitions, decoratedTiles)
	}
}
