package domain

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/imageutil"
)

const MaxRandomEnemyLevel = 10 // this excludes the color wizards and Arzakon

// details for the "rogues" in the game (aka your enemies)
var Rogues = loadRogues()

func (c *Character) LoadImages() error {
	if c.Visage == nil {
		data, err := assets.RogueVisageFS.ReadFile("art/rogues/" + c.VisageFn)
		if err != nil {
			fmt.Printf("ERROR: Loading Visage: %v", err)
		}
		img, err := imageutil.LoadImage(data)
		if err != nil {
			fmt.Printf("ERROR: Loading Visage: %v", err)
		}
		c.Visage = img
	}
	if c.WalkingSprite == nil {
		data, err := assets.RogueSpriteFS.ReadFile("art/screens/world/characters/" + c.WalkingSpriteFn)
		if err != nil {
			fmt.Printf("ERROR: Loading Walking for %s: %v", c.Name, err)
		}
		spr, err := imageutil.LoadSpriteSheet(5, 8, data)
		if err != nil {
			fmt.Printf("ERROR: Loading Walking for %s: %v", c.Name, err)
		}
		c.WalkingSprite = spr
	}
	if c.ShadowSprite == nil {
		data, err := assets.RogueSpriteFS.ReadFile("art/screens/world/characters/" + c.WalkingShadowSpriteFn)
		if err != nil {
			fmt.Printf("ERROR: Loading Shadow: %v", err)
		}
		spr, err := imageutil.LoadSpriteSheet(5, 8, data)
		if err != nil {
			fmt.Printf("ERROR: Loading Shadow: %v", err)
		}
		c.ShadowSprite = spr
	}
	return nil
}

// DungeonEnemyPool returns color-matched rogues from the difficulty's tier band.
func DungeonEnemyPool(color ColorMask, difficulty DungeonDifficulty) []*Character {
	difficulty = difficulty.normalized()
	target := ColorMaskToString(color)
	var matched, all []*Character
	for _, c := range Rogues {
		if !difficulty.includesEnemyTier(c.Level) {
			continue
		}
		all = append(all, c)
		if c.PrimaryColor == target {
			matched = append(matched, c)
		}
	}
	pool := matched
	if len(pool) == 0 {
		pool = all
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].Name < pool[j].Name })
	return pool
}

func (d DungeonDifficulty) includesEnemyTier(tier int) bool {
	switch d {
	case DungeonDifficultyMedium:
		return tier >= 4 && tier <= 7
	case DungeonDifficultyHard:
		return tier >= 8 && tier <= 10
	default:
		return tier >= 1 && tier <= 3
	}
}

func analyzeColors(collection CardCollection) (string, []string) {
	colorCounts := map[string]int{
		"W": 0, // White
		"U": 0, // Blue
		"B": 0, // Black
		"R": 0, // Red
		"G": 0, // Green
	}

	colorSet := make(map[string]bool)

	for card, item := range collection {
		// Add colors from ColorIdentity (includes lands and mana costs)
		for _, color := range card.ColorIdentity {
			colorCounts[color] += item.Count
			colorSet[color] = true
		}
	}

	// Convert color set to slice
	var colors []string
	for color := range colorSet {
		colors = append(colors, color)
	}

	// Find primary color (most frequent)
	primaryColor := ""
	maxCount := 0
	for color, count := range colorCounts {
		if count > maxCount {
			maxCount = count
			primaryColor = color
		}
	}

	// Map single letter colors to full names
	colorMap := map[string]string{
		"W": colorNameWhite,
		"U": colorNameBlue,
		"B": colorNameBlack,
		"R": colorNameRed,
		"G": colorNameGreen,
	}

	if fullColor, ok := colorMap[primaryColor]; ok {
		return fullColor, colors
	}

	// Default to colorless for no colors found
	return colorNameColorless, colors
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

		data, err := assets.RogueCfgFS.ReadFile(path.Join("configs/rogues", f.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading embedded %s: %w", f.Name(), err))
		}
		var r Character
		if _, err := toml.Decode(string(data), &r); err != nil {
			panic(fmt.Errorf("error decoding embedded %s: %w", f.Name(), err))
		}

		r.CardCollection = NewCardCollection()

		// Add deck cards to collection and to deck 0
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
			r.CardCollection.AddCardToDeck(card, 0, count)
		}

		// Add sideboard cards to collection only (not to any deck)
		for _, entry := range r.SideboardRaw {
			if len(entry) != 2 {
				panic(fmt.Errorf("invalid sideboard entry format: %v", entry))
			}
			count, err := strconv.Atoi(entry[0])
			if err != nil {
				panic(fmt.Errorf("invalid sideboard entry count: %v", entry[0]))
			}
			name := entry[1]
			card := FindCardByName(name)
			if card == nil {
				panic(fmt.Sprintf("Unable to find card: %s\n", name))
			}
			r.CardCollection.AddCard(card, count)
		}

		// Analyze collection colors and set color fields
		primaryColor, colorIdentity := analyzeColors(r.CardCollection)
		r.PrimaryColor = primaryColor
		r.ColorIdentity = colorIdentity

		r.Life = r.calculateLifeFromLevel()

		rogues[r.Name] = &r
	}

	return rogues
}
