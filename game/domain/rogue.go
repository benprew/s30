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
	"github.com/benprew/s30/game/entities"
	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
)

// details for the "rogues" in the game (aka your enemies)

type DeckEntry struct {
	Count int
	Name  string
}

type Rogue struct {
	Name            string            `toml:"name"`
	Visage          *ebiten.Image     // rogues headshot, seen at start of duel
	VisageFn        string            `toml:"image"` // filename only, lazy-loaded later
	WalkingSprite   [][]*ebiten.Image // sprites for walking animation
	ShadowSprite    [][]*ebiten.Image // sprites for shadow animation
	WalkingSpriteFn string            `toml:"world_sprite"` // filename only, lazy-loaded later
	Life            int               `toml:"life"`
	Catchphrases    []string          `toml:"catchphrases"`
	DeckRaw         [][]string        `toml:"main_cards"`
	Deck            []DeckEntry
	SideboardRaw    [][]string `toml:"sideboard_cards"`
	Sideboard       []DeckEntry
}

var Rogues map[string]*Rogue

func (e *Rogue) LoadImages() error {
	if e.Visage == nil {
		data := getEmbeddedFile(e.VisageFn)

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return err
		}
		e.Visage = ebiten.NewImageFromImage(img)
		return nil
	}
	if e.WalkingSprite == nil {
		data := getEmbeddedFile(e.WalkingSpriteFn)
		spr, err := sprites.LoadSpriteSheet(5, 8, data)
		if err != nil {
			return err
		}
		e.WalkingSprite = spr
	}
	if e.ShadowSprite == nil {
		// derive character name from walking sprite filename by trimming at first '.'
		base := e.WalkingSpriteFn
		if i := strings.Index(base, "."); i != -1 {
			base = base[:i]
		}
		charName := entities.CharacterName(base)
		shadowFile := entities.ShadowName(charName) + ".spr.png"
		data := getEmbeddedFile(shadowFile)
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

	return rogues, nil
}

// Helper function to get embedded file bytes
func getEmbeddedFile(filename string) []byte {
	// Try rogue-specific visage FS first (rogue portraits)
	if data, err := assets.RogueVisageFS.ReadFile("art/sprites/rogues/" + filename); err == nil {
		return data
	}

	// Try rogue-specific sprite FS (world character sprites)
	if data, err := assets.RogueSpriteFS.ReadFile("art/sprites/world/characters/" + filename); err == nil {
		return data
	}

	// If all reads failed, log the last error via a best-effort read to obtain
	// an error message for debugging.
	if _, err := assets.CharacterFS.ReadFile("art/sprites/world/characters/" + filename); err != nil {
		fmt.Printf("Error loading sprite file %s: %v\n", filename, err)
	} else {
		// unlikely: second read succeeded
		if data, err := assets.CharacterFS.ReadFile("art/sprites/world/characters/" + filename); err == nil {
			return data
		}
	}

	return nil
}
