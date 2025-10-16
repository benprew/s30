package domain

import (
	"bytes"
	"fmt"
	"image"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
)

// details for the "rogues" in the game (aka your enemies)
var Rogues = loadRogues()

func (c *Character) LoadImages() error {
	if c.Visage == nil {
		data, err := assets.RogueVisageFS.ReadFile("art/sprites/rogues/" + c.VisageFn)
		if err != nil {
			fmt.Printf("ERROR: Loading Visage: %v", err)
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			fmt.Printf("ERROR: Loading Visage: %v", err)
		}
		if err == nil {
			c.Visage = ebiten.NewImageFromImage(img)
		}
	}
	if c.WalkingSprite == nil {
		data, err := assets.RogueSpriteFS.ReadFile("art/sprites/world/characters/" + c.WalkingSpriteFn)
		if err != nil {
			fmt.Printf("ERROR: Loading Walking for %s: %v", c.Name, err)
		}
		spr, err := sprites.LoadSpriteSheet(5, 8, data)
		if err != nil {
			fmt.Printf("ERROR: Loading Walking for %s: %v", c.Name, err)
		}
		c.WalkingSprite = spr
	}
	if c.ShadowSprite == nil {
		data, err := assets.RogueSpriteFS.ReadFile("art/sprites/world/characters/" + c.WalkingShadowSpriteFn)
		if err != nil {
			fmt.Printf("ERROR: Loading Shadow: %v", err)
		}
		spr, err := sprites.LoadSpriteSheet(5, 8, data)
		if err != nil {
			fmt.Printf("ERROR: Loading Shadow: %v", err)
		}
		c.ShadowSprite = spr
	}
	return nil
}

func loadRogues() map[string]*Character {
	rogues := make(map[string]*Character)
	configDir := "configs/rogues"

	files, err := assets.RogueCfgFS.ReadDir(configDir)
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}

		data, err := assets.RogueCfgFS.ReadFile(filepath.Join("configs/rogues", f.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading embedded %s: %w", f.Name(), err))
		}
		var r Character
		if _, err := toml.Decode(string(data), &r); err != nil {
			panic(fmt.Errorf("error decoding embedded %s: %w", f.Name(), err))
		}

		r.Deck = make(Deck)

		for _, entry := range r.DeckRaw {
			if len(entry) != 2 {
				panic(fmt.Errorf("invalid deck entry format: %v", entry))
			}
			count, err := strconv.Atoi(entry[0])
			if err != nil {
				panic(fmt.Errorf("invalid deck entry count: %v", entry[0]))
			}
			name := entry[1]
			card := FindCardByName(name)
			if card == nil {
				panic(fmt.Sprintf("Unable to find card: %s\n", name))
			}
			r.Deck[card] = count
		}

		rogues[r.Name] = &r
	}

	return rogues
}
