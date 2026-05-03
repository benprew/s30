package world

import (
	"image"
	"testing"
)

func TestPlaceDungeonsRespectsCount(t *testing.T) {
	l := createTestLevel(20, 20)
	l.placeDungeons(5, 3, 1, nil)
	if got := len(l.Dungeons); got != 5 {
		t.Fatalf("expected 5 dungeons, got %d", got)
	}
}

func TestPlaceDungeonsAvoidsCitiesAndWater(t *testing.T) {
	l := createTestLevel(20, 20)
	l.Tile(image.Point{5, 5}).IsCity = true
	l.Tile(image.Point{6, 5}).TerrainType = TerrainWater
	l.Tile(image.Point{7, 5}).TerrainType = TerrainWater

	l.placeDungeons(5, 3, 42, nil)

	for _, d := range l.Dungeons {
		tile := l.Tile(d.MapTile)
		if tile.IsCity {
			t.Errorf("dungeon placed on city at %v", d.MapTile)
		}
		if tile.TerrainType == TerrainWater {
			t.Errorf("dungeon placed on water at %v", d.MapTile)
		}
		if !tile.IsDungeon {
			t.Errorf("tile at %v not marked IsDungeon", d.MapTile)
		}
		if tile.Dungeon != d {
			t.Errorf("tile.Dungeon at %v does not point back to dungeon", d.MapTile)
		}
	}
}

func TestPlaceDungeonsHonorsMinDistance(t *testing.T) {
	l := createTestLevel(30, 30)
	const minDist = 4
	l.placeDungeons(8, minDist, 7, nil)

	for i, a := range l.Dungeons {
		for j, b := range l.Dungeons {
			if i == j {
				continue
			}
			d := absInt(a.MapTile.X-b.MapTile.X) + absInt(a.MapTile.Y-b.MapTile.Y)
			if d <= minDist {
				t.Errorf("dungeons %d and %d only %d apart (min %d)", i, j, d, minDist)
			}
		}
	}
}

func TestPlaceDungeonsAssignsDistinctColorsRoundRobin(t *testing.T) {
	l := createTestLevel(20, 20)
	l.placeDungeons(5, 2, 1, nil)

	seen := map[int]bool{}
	for _, d := range l.Dungeons {
		seen[int(d.Color)] = true
	}
	if len(seen) != 5 {
		t.Fatalf("expected 5 distinct dungeon colors, got %d (%v)", len(seen), seen)
	}
}
