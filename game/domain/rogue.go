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

type DeckEntry struct {
	Count int
	Name  string
}

type Rogue struct {
	Name                  string            `toml:"name"`
	Visage                *ebiten.Image     // rogues headshot, seen at start of duel
	VisageFn              string            `toml:"image"` // filename only, lazy-loaded later
	WalkingSprite         [][]*ebiten.Image // sprites for walking animation
	ShadowSprite          [][]*ebiten.Image // sprites for shadow animation
	WalkingSpriteFn       string            `toml:"walking_sprite"`        // filename only, lazy-loaded later
	WalkingShadowSpriteFn string            `toml:"walking_shadow_sprite"` // filename only, lazy-loaded later
	Life                  int               `toml:"life"`
	Catchphrases          []string          `toml:"catchphrases"`
	DeckRaw               [][]string        `toml:"main_cards"`
	Deck                  []DeckEntry
	SideboardRaw          [][]string `toml:"sideboard_cards"`
	Sideboard             []DeckEntry
}

var Rogues map[string]*Rogue

// RoguesBySprite maps walking sprite filename -> Rogue for quick lookup
var RoguesBySprite map[string]*Rogue

func (e *Rogue) LoadImages() error {
	if e.Visage == nil {
		data, err := assets.RogueVisageFS.ReadFile("art/sprites/rogues/" + e.VisageFn)
		if err != nil {
			return err
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return err
		}
		e.Visage = ebiten.NewImageFromImage(img)
		return nil
	}
	if e.WalkingSprite == nil {
		data, err := assets.RogueSpriteFS.ReadFile("art/sprites/world/characters/" + e.WalkingSpriteFn)
		if err != nil {
			return err
		}
		spr, err := sprites.LoadSpriteSheet(5, 8, data)
		if err != nil {
			return err
		}
		e.WalkingSprite = spr
	}
	if e.ShadowSprite == nil {
		data, err := assets.RogueSpriteFS.ReadFile("art/sprites/world/characters/" + e.WalkingShadowSpriteFn)
		if err != nil {
			return err
		}
		spr, err := sprites.LoadSpriteSheet(5, 8, data)
		if err != nil {
			return err
		}
		e.ShadowSprite = spr
	}
	return nil
}

func LoadRogues() (map[string]*Rogue, error) {
	rogues := make(map[string]*Rogue)
	configDir := "configs/rogues"

	files, err := assets.RogueCfgFS.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}

		data, err := assets.RogueCfgFS.ReadFile(filepath.Join("configs/rogues", f.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading embedded %s: %w", f.Name(), err)
		}
		var r Rogue
		if _, err := toml.Decode(string(data), &r); err != nil {
			return nil, fmt.Errorf("error decoding embedded %s: %w", f.Name(), err)
		}

		// Convert deckRaw into DeckEntries
		for _, entry := range r.DeckRaw {
			if len(entry) != 2 {
				return nil, fmt.Errorf("invalid deck entry format: %v", entry)
			}
			count, err := strconv.Atoi(entry[0])
			if err != nil {
				return nil, fmt.Errorf("invalid deck entry count: %v", entry[0])
			}
			name := entry[1]
			fmt.Println(count, name)
			r.Deck = append(r.Deck, DeckEntry{Count: count, Name: name})
		}

		rogues[r.Name] = &r
	}

	// Build RoguesBySprite for reverse lookup by walking sprite filename
	RoguesBySprite = make(map[string]*Rogue)
	for _, r := range rogues {
		if r.WalkingSpriteFn != "" {
			RoguesBySprite[r.WalkingSpriteFn] = r
		}
	}

	return rogues, nil
}
