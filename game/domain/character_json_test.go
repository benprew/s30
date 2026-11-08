package domain

import (
	"encoding/json"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCharacterJSONOmitsImages(t *testing.T) {
	c := Character{
		Name: "saved enemy", VisageFn: "face.png", WalkingSpriteFn: "walk.png", WalkingShadowSpriteFn: "shadow.png",
		Visage: &ebiten.Image{}, WalkingSprite: [][]*ebiten.Image{{{}}}, ShadowSprite: [][]*ebiten.Image{{{}}},
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Visage", "WalkingSprite", "ShadowSprite"} {
		if _, ok := fields[name]; ok {
			t.Errorf("save contains runtime image field %s", name)
		}
	}
	var loaded Character
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Name != c.Name || loaded.VisageFn != c.VisageFn || loaded.WalkingSpriteFn != c.WalkingSpriteFn || loaded.WalkingShadowSpriteFn != c.WalkingShadowSpriteFn {
		t.Fatal("save lost character identity or image filenames")
	}
}

func TestCharacterJSONIgnoresLegacyImages(t *testing.T) {
	var c Character
	if err := json.Unmarshal([]byte(`{"Name":"saved enemy","Visage":{},"WalkingSprite":[[{}]],"ShadowSprite":[[{}]]}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Visage != nil || c.WalkingSprite != nil || c.ShadowSprite != nil {
		t.Fatal("legacy image objects must remain nil so LoadImages can reload them")
	}
}
