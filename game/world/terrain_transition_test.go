package world

import (
	"image"
	"testing"
)

func TestTransitionSpriteBank(t *testing.T) {
	tests := []struct {
		terrain TileType
		sheet   int
		flat    int
	}{
		{TileWater, 1, 0},
		{TileMarsh, 1, 28},
		{TileForest, 1, 56},
		{TileSand, 2, 0},
		{TileIce, 2, 28},
	}
	for _, tt := range tests {
		bank, ok := transitionSpriteBankFor(tt.terrain)
		if !ok {
			t.Fatalf("missing sprite bank for %s", tt.terrain)
		}
		if bank.sheet != tt.sheet || bank.flatOffset != tt.flat {
			t.Errorf("%s bank = %+v, want sheet %d flat offset %d", tt.terrain, bank, tt.sheet, tt.flat)
		}
	}
}

func TestCornerTransitionPatternsMatchOriginalBitExtraction(t *testing.T) {
	tests := []struct {
		mask uint8
		want [4]int
	}{
		{mask: 0, want: [4]int{0, 0, 0, 0}},
		{mask: 1, want: [4]int{1, 0, 4, 0}},
		{mask: 0b00010101, want: [4]int{5, 1, 4, 5}},
		{mask: 0xff, want: [4]int{7, 7, 7, 7}},
	}
	for _, tt := range tests {
		if got := cornerTransitionPatterns(tt.mask); got != tt.want {
			t.Errorf("mask %08b: patterns = %v, want %v", tt.mask, got, tt.want)
		}
	}
}

func TestCornerTransitionPlanUsesOriginalFlatSpriteOrder(t *testing.T) {
	got := cornerTransitionPlan(TileMarsh, 1)
	if len(got) != 2 {
		t.Fatalf("plan length = %d, want 2", len(got))
	}
	want := []positionedTransitionRef{
		{transitionSpriteRef: transitionSpriteRef{Row: 7, Col: 0}, Sheet: 1, OffsetX: 52, OffsetY: 0},
		{transitionSpriteRef: transitionSpriteRef{Row: 11, Col: 1}, Sheet: 1, OffsetX: -51, OffsetY: 50},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("piece %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestOriginalTerrainTransitionMasksDoNotUseForestCstline(t *testing.T) {
	level := transitionTestLevel(TerrainBandPlains)
	center := image.Point{X: 2, Y: 2}
	neighbor := center.Add(originalTerrainDirections()[0])
	level.Tile(neighbor).TerrainBand = TerrainBandForest
	level.Tile(neighbor).TerrainType = TerrainForest

	waterMask, marshMask := originalTerrainTransitionMasks(level, center)
	if waterMask != 0 || marshMask != 0 {
		t.Fatalf("forest neighbor produced masks water=%08b marsh=%08b", waterMask, marshMask)
	}
}

func TestOriginalTerrainDirectionsUseTheSourceMapNeighborhood(t *testing.T) {
	want := [8]image.Point{
		{X: 0, Y: -1},
		{X: 1, Y: -1},
		{X: 1, Y: 0},
		{X: 1, Y: 1},
		{X: 0, Y: 1},
		{X: -1, Y: 1},
		{X: -1, Y: 0},
		{X: -1, Y: -1},
	}
	if got := originalTerrainDirections(); got != want {
		t.Fatalf("directions = %v, want %v", got, want)
	}
}

func TestOriginalTerrainTransitionMasksDrawCoastOnLandSide(t *testing.T) {
	level := transitionTestLevel(TerrainBandWater)
	center := image.Point{X: 2, Y: 2}
	level.Tile(center).TerrainBand = TerrainBandSand
	level.Tile(center).TerrainType = TerrainSand

	waterMask, _ := originalTerrainTransitionMasks(level, center)
	if waterMask != 0xff {
		t.Fatalf("sand tile water mask = %08b, want 11111111", waterMask)
	}

	for _, band := range []TerrainBand{TerrainBandWater, TerrainBandSandMarsh} {
		level.Tile(center).TerrainBand = band
		level.Tile(center).TerrainType = band.properties().terrain
		waterMask, marshMask := originalTerrainTransitionMasks(level, center)
		if waterMask != 0 || marshMask != 0 {
			t.Errorf("%v tile produced masks water=%08b marsh=%08b", band, waterMask, marshMask)
		}
	}
}

func TestOriginalTerrainTransitionMasksUseFixedNeighborBits(t *testing.T) {
	level := transitionTestLevel(TerrainBandPlains)
	center := image.Point{X: 2, Y: 2}
	for index, direction := range originalTerrainDirections() {
		neighbor := center.Add(direction)
		level.Tile(neighbor).TerrainBand = TerrainBandWater
		level.Tile(neighbor).TerrainType = TerrainWater

		waterMask, _ := originalTerrainTransitionMasks(level, center)
		if waterMask != 1<<index {
			t.Errorf("direction %v produced mask %08b, want %08b", direction, waterMask, uint8(1<<index))
		}

		level.Tile(neighbor).TerrainBand = TerrainBandPlains
		level.Tile(neighbor).TerrainType = TerrainPlains
	}
}

func transitionTestLevel(band TerrainBand) *Level {
	level := createTestLevel(5, 5)
	for y := 0; y < level.H; y++ {
		for x := 0; x < level.W; x++ {
			tile := level.Tile(image.Point{X: x, Y: y})
			tile.TerrainBand = band
			tile.TerrainType = band.properties().terrain
		}
	}
	return level
}
