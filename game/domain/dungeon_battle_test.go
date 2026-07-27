package domain

import "testing"

func TestNewEnemyFromCharacterWrapsCharacter(t *testing.T) {
	c := &Character{Name: "Goblin", Life: 7}
	e := NewEnemyFromCharacter(c)
	if e.Character != c {
		t.Fatalf("expected enemy to reference the given character")
	}
	if e.Name() != "Goblin" {
		t.Errorf("expected name Goblin, got %q", e.Name())
	}
}

func TestDungeonEnemyPoolIsNonEmptyDeterministicColorAndDifficultyMatched(t *testing.T) {
	pool := DungeonEnemyPool(ColorRed, DungeonDifficultyMedium)
	if len(pool) == 0 {
		t.Fatal("expected a non-empty enemy pool")
	}

	// Every returned rogue should match the requested color, since red rogues
	// exist in the registry.
	for _, c := range pool {
		if c.PrimaryColor != colorNameRed {
			t.Errorf("expected only Red rogues, got %q (%s)", c.PrimaryColor, c.Name)
		}
		if c.Level < 4 || c.Level > 7 {
			t.Errorf("expected medium rogue tier 4-7, got %d (%s)", c.Level, c.Name)
		}
	}

	// Deterministic ordering so seeded dungeon generation is reproducible.
	again := DungeonEnemyPool(ColorRed, DungeonDifficultyMedium)
	if len(again) != len(pool) {
		t.Fatalf("pool size changed between calls: %d vs %d", len(pool), len(again))
	}
	for i := range pool {
		if pool[i].Name != again[i].Name {
			t.Errorf("pool order not deterministic at %d: %q vs %q", i, pool[i].Name, again[i].Name)
		}
	}
}

func TestDungeonEnemyPoolUsesDifficultyTierBands(t *testing.T) {
	tests := []struct {
		difficulty DungeonDifficulty
		min, max   int
	}{
		{DungeonDifficultyEasy, 1, 3},
		{DungeonDifficultyMedium, 4, 7},
		{DungeonDifficultyHard, 8, 10},
	}
	for _, test := range tests {
		pool := DungeonEnemyPool(ColorRed, test.difficulty)
		if len(pool) == 0 {
			t.Fatalf("difficulty %d returned no enemies", test.difficulty)
		}
		for _, enemy := range pool {
			if enemy.Level < test.min || enemy.Level > test.max {
				t.Errorf("difficulty %d included tier %d enemy %s", test.difficulty, enemy.Level, enemy.Name)
			}
		}
	}
}

func TestDefeatEnemyClearsTile(t *testing.T) {
	st := &DungeonState{}
	tile := &DungeonTile{Type: DungeonTileEnemy, Enemy: &Character{Name: "Goblin"}}

	st.DefeatEnemy(tile)

	if tile.Type != DungeonTileEmpty {
		t.Errorf("expected tile cleared to empty, got %v", tile.Type)
	}
	if tile.Enemy != nil {
		t.Errorf("expected enemy removed from tile, got %v", tile.Enemy)
	}
}

func TestDefeatEnemyNoOpOnNonEnemyTile(t *testing.T) {
	st := &DungeonState{}
	tile := &DungeonTile{Type: DungeonTileTreasure}
	st.DefeatEnemy(tile)
	if tile.Type != DungeonTileTreasure {
		t.Errorf("non-enemy tile should be unchanged, got %v", tile.Type)
	}
	st.DefeatEnemy(nil) // must not panic
}

func TestExitDungeonRestoresLifeAndClearsState(t *testing.T) {
	p := newTestPlayer()
	p.Life = 5
	p.DungeonState = &DungeonState{DungeonLife: 18}

	p.ExitDungeon()

	if p.Life != 18 {
		t.Errorf("expected life restored to dungeon life 18, got %d", p.Life)
	}
	if p.DungeonState != nil {
		t.Errorf("expected dungeon state cleared")
	}
}

func TestExitDungeonNoStateIsNoOp(t *testing.T) {
	p := newTestPlayer()
	p.Life = 5
	p.ExitDungeon()
	if p.Life != 5 || p.DungeonState != nil {
		t.Errorf("exit with no dungeon state should not change life")
	}
}
