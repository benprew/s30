package domain

import (
	"image"
	"math/rand"
	"testing"
)

func TestGenerateDungeonProducesWalkableEntrance(t *testing.T) {
	d := GenerateDungeon(DungeonGenOptions{
		Name: "Test",
		Seed: 1,
	})

	entrance := d.Tile(d.Entrance)
	if entrance == nil {
		t.Fatalf("entrance tile out of bounds: %v", d.Entrance)
	}
	if entrance.Type != DungeonTileEntrance {
		t.Fatalf("expected entrance tile type, got %v", entrance.Type)
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
			Difficulty:      DungeonDifficultyEasy,
			RestrictedCards: cards,
			NumGoldChests:   2,
			Seed:            seed,
		})
		if err := d.AllRestrictedCardsReachable(); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
	}
}

func TestGenerateDungeonPlacesRequestedEntities(t *testing.T) {
	d := GenerateDungeon(DungeonGenOptions{
		Difficulty:      DungeonDifficultyMedium,
		RestrictedCards: []*Card{{CardName: "X"}, {CardName: "Y"}},
		NumGoldChests:   2,
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
	if counts[DungeonTileEnemy] < 7 || counts[DungeonTileEnemy] > 10 {
		t.Errorf("expected 7-10 enemies, got %d", counts[DungeonTileEnemy])
	}
	if counts[DungeonTileDice] < 3 || counts[DungeonTileDice] > 5 {
		t.Errorf("expected 3-5 dice, got %d", counts[DungeonTileDice])
	}
	if counts[DungeonTileScroll] < 2 || counts[DungeonTileScroll] > 4 {
		t.Errorf("expected 2-4 scrolls, got %d", counts[DungeonTileScroll])
	}
}

func TestGenerateDungeonDeterministicForSeed(t *testing.T) {
	d1 := GenerateDungeon(DungeonGenOptions{Difficulty: DungeonDifficultyEasy, Seed: 99})
	d2 := GenerateDungeon(DungeonGenOptions{Difficulty: DungeonDifficultyEasy, Seed: 99})
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
		Difficulty:      DungeonDifficultyEasy,
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
				t.Errorf("restricted-card chest at (%d,%d) has %d open neighbors, expected 1 (dead end)", x, y, open)
			}
		}
	}
}

func TestGenerateDungeonPlacesFinalEnemyAtFarthestDeadEnd(t *testing.T) {
	boss := &Character{Name: "Wizard"}
	for seed := int64(1); seed <= 20; seed++ {
		d := GenerateDungeon(DungeonGenOptions{
			Difficulty: DungeonDifficultyHard, NumGoldChests: 2, FinalEnemy: boss, Seed: seed,
		})
		dist := distancesFrom(d.Grid, d.Entrance)
		deadEnds := findDeadEndsIncludingEvents(d.Grid)
		maxDistance := -1
		for _, p := range deadEnds {
			maxDistance = max(maxDistance, dist[p.Y][p.X])
		}
		bossCount := 0
		for y := range d.Grid {
			for x := range d.Grid[y] {
				tile := &d.Grid[y][x]
				if !tile.Boss {
					continue
				}
				bossCount++
				if tile.Type != DungeonTileEnemy || tile.Enemy != boss {
					t.Fatalf("seed %d: boss tile was displaced", seed)
				}
				if dist[y][x] != maxDistance {
					t.Fatalf("seed %d: boss distance %d, farthest dead end %d", seed, dist[y][x], maxDistance)
				}
			}
		}
		if bossCount != 1 {
			t.Fatalf("seed %d: got %d bosses, want 1", seed, bossCount)
		}
	}
}

func TestGenerateDungeonPlacesFinalEnemyWhenGridIsTooSmall(t *testing.T) {
	boss := &Character{Name: "Wizard"}
	d := GenerateDungeon(DungeonGenOptions{Difficulty: DungeonDifficultyHard, FinalEnemy: boss, Seed: 1})

	bossCount := 0
	for y := range d.Grid {
		for x := range d.Grid[y] {
			if d.Grid[y][x].Boss && d.Grid[y][x].Enemy == boss {
				bossCount++
			}
		}
	}
	if bossCount != 1 {
		t.Fatalf("got %d bosses, want 1", bossCount)
	}
}

func TestCastleDungeonGenOptionsUsesCastleDefaults(t *testing.T) {
	boss := &Character{Name: "Wizard"}
	opts := CastleDungeonGenOptions("Castle", ColorRed, boss, nil, 7)

	if opts.Theme != DungeonThemeCastle || opts.FinalEnemy != boss {
		t.Fatal("castle identity was not configured")
	}
	if opts.Difficulty != DungeonDifficultyHard || opts.NumGoldChests != 2 {
		t.Fatalf("unexpected castle defaults: %+v", opts)
	}
}

func TestDungeonDifficultyProfilesStayWithinConfiguredRanges(t *testing.T) {
	tests := []struct {
		difficulty             DungeonDifficulty
		minEnemies, maxEnemies int
		minDice, maxDice       int
		minScrolls, maxScrolls int
	}{
		{DungeonDifficultyEasy, 4, 6, 5, 7, 3, 5},
		{DungeonDifficultyMedium, 7, 10, 3, 5, 2, 4},
		{DungeonDifficultyHard, 11, 15, 1, 3, 1, 3},
	}
	for _, test := range tests {
		for seed := int64(1); seed <= 50; seed++ {
			profile := rollDungeonDifficulty(test.difficulty, rand.New(rand.NewSource(seed)))
			if profile.enemies < test.minEnemies || profile.enemies > test.maxEnemies ||
				profile.dice < test.minDice || profile.dice > test.maxDice ||
				profile.scrolls < test.minScrolls || profile.scrolls > test.maxScrolls {
				t.Fatalf("difficulty %d produced out-of-range profile: %+v", test.difficulty, profile)
			}
		}
	}
}

func findDeadEndsIncludingEvents(g [][]DungeonTile) []image.Point {
	copyGrid := make([][]DungeonTile, len(g))
	for y := range g {
		copyGrid[y] = append([]DungeonTile(nil), g[y]...)
		for x := range copyGrid[y] {
			if copyGrid[y][x].Type != DungeonTileWall && copyGrid[y][x].Type != DungeonTileEntrance {
				copyGrid[y][x].Type = DungeonTileEmpty
			}
		}
	}
	return findDeadEnds(copyGrid)
}
