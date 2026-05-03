package domain

import (
	"image"
	"testing"
)

func TestGenerateDungeonProducesWalkableEntrance(t *testing.T) {
	d := GenerateDungeon(DungeonGenOptions{
		Name:     "Test",
		GridSize: 11,
		Seed:     1,
	})

	entrance := d.Tile(d.Entrance)
	if entrance == nil {
		t.Fatalf("entrance tile out of bounds: %v", d.Entrance)
	}
	if entrance.Type != DungeonTileEntrance {
		t.Fatalf("expected entrance tile type, got %v", entrance.Type)
	}
}

func TestGenerateDungeonGridDimensionsForcedOdd(t *testing.T) {
	d := GenerateDungeon(DungeonGenOptions{GridSize: 10, Seed: 1})
	if d.Width() != 11 || d.Height() != 11 {
		t.Fatalf("expected 11x11 grid (odd-coerced), got %dx%d", d.Width(), d.Height())
	}
}

func TestGenerateDungeonAllRewardsReachable(t *testing.T) {
	cards := []*Card{
		{CardName: "Reward A"},
		{CardName: "Reward B"},
		{CardName: "Reward C"},
	}
	for seed := int64(1); seed <= 20; seed++ {
		d := GenerateDungeon(DungeonGenOptions{
			Name:            "Test",
			GridSize:        11,
			RestrictedCards: cards,
			NumGoldChests:   2,
			NumEnemies:      3,
			NumDice:         1,
			NumScrolls:      1,
			Seed:            seed,
		})
		if err := d.AllRestrictedCardsReachable(); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
	}
}

func TestGenerateDungeonPlacesRequestedEntities(t *testing.T) {
	d := GenerateDungeon(DungeonGenOptions{
		GridSize:        15,
		RestrictedCards: []*Card{{CardName: "X"}, {CardName: "Y"}},
		NumGoldChests:   2,
		NumEnemies:      3,
		NumDice:         2,
		NumScrolls:      2,
		Seed:            42,
	})

	counts := map[DungeonTileType]int{}
	for y := range d.Grid {
		for x := range d.Grid[y] {
			counts[d.Grid[y][x].Type]++
		}
	}

	if counts[DungeonTileTreasure] != 4 {
		t.Errorf("expected 4 treasures (2 cards + 2 gold), got %d", counts[DungeonTileTreasure])
	}
	if counts[DungeonTileEnemy] != 3 {
		t.Errorf("expected 3 enemies, got %d", counts[DungeonTileEnemy])
	}
	if counts[DungeonTileDice] != 2 {
		t.Errorf("expected 2 dice, got %d", counts[DungeonTileDice])
	}
	if counts[DungeonTileScroll] != 2 {
		t.Errorf("expected 2 scrolls, got %d", counts[DungeonTileScroll])
	}
}

func TestGenerateDungeonDeterministicForSeed(t *testing.T) {
	d1 := GenerateDungeon(DungeonGenOptions{GridSize: 11, NumEnemies: 2, Seed: 99})
	d2 := GenerateDungeon(DungeonGenOptions{GridSize: 11, NumEnemies: 2, Seed: 99})
	for y := range d1.Grid {
		for x := range d1.Grid[y] {
			if d1.Grid[y][x].Type != d2.Grid[y][x].Type {
				t.Fatalf("seeded generation diverged at (%d,%d)", x, y)
			}
		}
	}
}

func TestGenerateDungeonRestrictedCardsAtDeadEnds(t *testing.T) {
	cards := []*Card{{CardName: "Reward"}}
	d := GenerateDungeon(DungeonGenOptions{
		GridSize:        11,
		RestrictedCards: cards,
		Seed:            7,
	})

	for y := range d.Grid {
		for x := range d.Grid[y] {
			t2 := &d.Grid[y][x]
			if t2.Type != DungeonTileTreasure || t2.Reward == nil {
				continue
			}
			if t2.Reward.Type != DungeonRewardRestrictedCard {
				continue
			}
			open := 0
			for _, dir := range [4]image.Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
				ny, nx := y+dir.Y, x+dir.X
				if ny < 0 || ny >= d.Height() || nx < 0 || nx >= d.Width() {
					continue
				}
				if d.Grid[ny][nx].Type != DungeonTileWall {
					open++
				}
			}
			if open != 1 {
				t.Errorf("restricted-card chest at (%d,%d) has %d open neighbours, expected 1 (dead end)", x, y, open)
			}
		}
	}
}
