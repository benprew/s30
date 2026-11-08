package world

import (
	"encoding/json"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestRebuildDungeonEnemiesRepairsAllSavedCopies(t *testing.T) {
	rogue := domain.Rogues["Crusader"]
	if rogue == nil {
		t.Fatal("missing test rogue")
	}
	dungeon := &domain.Dungeon{Grid: [][]domain.DungeonTile{{{Type: domain.DungeonTileEnemy, Enemy: rogue}}}}
	level := &Level{
		Dungeons: []*domain.Dungeon{nil, dungeon},
		Tiles: [][]*Tile{{nil, {IsDungeon: true, Dungeon: dungeon}}},
		Player: &domain.Player{DungeonState: &domain.DungeonState{CurrentDungeon: dungeon}},
	}
	data, err := json.Marshal(level)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Level
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	loaded.rebuildDungeonEnemies()
	for name, d := range map[string]*domain.Dungeon{
		"list": loaded.Dungeons[1], "entrance": loaded.Tiles[0][1].Dungeon, "active": loaded.Player.DungeonState.CurrentDungeon,
	} {
		if d.Grid[0][0].Enemy != rogue {
			t.Errorf("%s dungeon still has a detached enemy", name)
		}
	}
}
