package screens

import (
	"encoding/json"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestDungeonSpritesLoadFromLegacySave(t *testing.T) {
	rogue := domain.Rogues["Crusader"]
	if rogue == nil {
		t.Fatal("missing test rogue")
	}
	data, err := json.Marshal(rogue)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	fields["Visage"] = json.RawMessage(`{}`)
	fields["WalkingSprite"] = json.RawMessage(`[[{}]]`)
	fields["ShadowSprite"] = json.RawMessage(`[[{}]]`)
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	var enemy domain.Character
	if err = json.Unmarshal(data, &enemy); err != nil {
		t.Fatal(err)
	}
	dungeon := &domain.Dungeon{Grid: [][]domain.DungeonTile{{{Type: domain.DungeonTileEnemy, Enemy: &enemy}}}}
	player := &domain.Player{DungeonState: &domain.DungeonState{CurrentDungeon: dungeon}}
	for range 2 {
		screen := NewDungeonScreen(player, nil)
		frames := screen.enemyFrames[&enemy]
		for name, sheet := range map[string][][]*ebiten.Image{"sprite": frames.sprite, "shadow": frames.shadow} {
			if len(sheet) != domain.SpriteRows || len(sheet[0]) != domain.SpriteCols {
				t.Fatalf("%s frames did not load", name)
			}
			if sheet[0][0] == nil || sheet[0][0].Bounds().Empty() {
				t.Fatalf("%s frame is empty", name)
			}
		}
	}
}
