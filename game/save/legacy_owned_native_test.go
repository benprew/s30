//go:build !js

package save

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestVersionOneFixtureLoadsRebuildsAndResavesWithoutLosingCards(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test-save-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var legacy struct {
		World struct {
			Player struct {
				Gold, Food, Life, X, Y, ActiveDeck int
				CardCollection                     []struct {
					CardID     string `json:"card_id"`
					Count      int    `json:"count"`
					DeckCounts []int  `json:"deck_counts"`
				}
			}
		}
	}
	if err = json.Unmarshal(data, &legacy); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	oldPath := filepath.Join(root, "legacy.json")
	if err = os.WriteFile(oldPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	SetSaveDir(filepath.Join(root, "new-saves"))
	t.Cleanup(func() { SetSaveDir("") })
	level, err := LoadGame(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	player := level.Player
	oldPlayer := legacy.World.Player
	if player.Gold != oldPlayer.Gold || player.Food != oldPlayer.Food || player.Life != oldPlayer.Life || player.X != oldPlayer.X || player.Y != oldPlayer.Y || player.ActiveDeck != oldPlayer.ActiveDeck {
		t.Fatal("old save lost player resources or position")
	}
	if player.CardCollection.NumCards() != 43 {
		t.Fatalf("old save has %d player cards, want 43", player.CardCollection.NumCards())
	}
	expectedArt := make(map[string]int)
	expectedDecks := make(map[int]int)
	for _, item := range oldPlayer.CardCollection {
		expectedArt[item.CardID] += item.Count
		for deckIndex, count := range item.DeckCounts {
			if count > 0 {
				expectedDecks[deckIndex] += count
			}
		}
	}
	actualArt := make(map[string]int)
	actualDecks := make(map[int]int)
	for _, item := range player.CardCollection {
		for _, owned := range item.Instances {
			actualArt[owned.ArtworkID]++
			if owned.DeckIndex >= 0 {
				actualDecks[owned.DeckIndex]++
			}
		}
	}
	if !reflect.DeepEqual(expectedArt, actualArt) || !reflect.DeepEqual(expectedDecks, actualDecks) {
		t.Fatal("old save lost printings or deck membership")
	}
	before := ownedSaveSnapshot(t, level)
	if err = level.RebuildSprites(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, ownedSaveSnapshot(t, level)) {
		t.Fatal("rebuilding the world replaced owned cards")
	}
	path, err := SaveGame(level)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadGame(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = reloaded.RebuildSprites(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, ownedSaveSnapshot(t, reloaded)) {
		t.Fatal("resaving lost owned card identity, artwork, or deck membership")
	}
	unchanged, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, unchanged) {
		t.Fatal("loading changed the original save file")
	}
}
