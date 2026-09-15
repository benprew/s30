package save

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestInvalidSaveVersion(t *testing.T) {
	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "invalid_version.json")

	saveData := &SaveData{
		Version: 999,
	}

	jsonData, err := json.MarshalIndent(saveData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal save data: %v", err)
	}

	if writeErr := os.WriteFile(savePath, jsonData, 0644); writeErr != nil {
		t.Fatalf("Failed to write save file: %v", writeErr)
	}

	_, err = LoadGame(savePath)
	if err == nil {
		t.Error("Expected error for unsupported version, got nil")
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := LoadGame("/nonexistent/path/to/save.json")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadCorruptedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "corrupted.json")

	if err := os.WriteFile(savePath, []byte("invalid json{{{"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	_, err := LoadGame(savePath)
	if err == nil {
		t.Error("Expected error for corrupted JSON, got nil")
	}
}

func TestGetSaveFilePath(t *testing.T) {
	savePath, err := getSaveFilePath("test")
	if err != nil {
		t.Fatalf("Failed to get save file path: %v", err)
	}

	if savePath == "" {
		t.Error("Expected non-empty save path")
	}

	dir := filepath.Dir(savePath)
	if dir == "" {
		t.Error("Expected non-empty directory path")
	}

	filename := filepath.Base(savePath)
	if filename == "" {
		t.Error("Expected non-empty filename")
	}
}

func TestDeserializeSaveWithOldCardVersions(t *testing.T) {
	saveJSON := []byte(`{
		"name": "test_game",
		"game_id": "test_game_123",
		"version": 1,
		"world": {
			"Player": {
				"CardCollection": [
					{"card_id": "4ed-999-lightning-bolt", "card_name": "Lightning Bolt", "count": 4, "deck_counts": [4]}
				],
				"BonusDuelCards": [
					{"CardID": "4ed-999-lightning-bolt"}
				]
			},
			"Tiles": [
				[
					{
						"City": {
							"Name": "Test City",
							"CardsForSale": [
								{"CardID": "4ed-999-lightning-bolt"}
							]
						}
					}
				]
			]
		}
	}`)

	saveData, err := deserializeSave(saveJSON)
	if err != nil {
		t.Fatalf("failed to deserialize save with old card IDs: %v", err)
	}

	if saveData.World == nil || saveData.World.Player == nil {
		t.Fatal("expected non-nil world and player")
	}
	if len(saveData.World.Player.CardCollection) != 1 {
		t.Fatalf("expected 1 card in collection, got %d", len(saveData.World.Player.CardCollection))
	}
	if len(saveData.World.Player.BonusDuelCards) != 1 || saveData.World.Player.BonusDuelCards[0].Name() != "Lightning Bolt" {
		t.Fatalf("expected BonusDuelCards to contain Lightning Bolt, got %v", saveData.World.Player.BonusDuelCards)
	}
	if len(saveData.World.Tiles) != 1 || len(saveData.World.Tiles[0]) != 1 || saveData.World.Tiles[0][0].City == nil {
		t.Fatal("expected non-nil tile city")
	}
	cityCards := saveData.World.Tiles[0][0].City.CardsForSale
	if len(cityCards) != 1 || cityCards[0].Name() != "Lightning Bolt" {
		t.Fatalf("expected CardsForSale to contain Lightning Bolt, got %v", cityCards)
	}
}

func TestDeserializeSaveSkipsRemovedCards(t *testing.T) {
	saveJSON := []byte(`{
		"name": "test_game",
		"game_id": "test_game_123",
		"version": 1,
		"world": {
			"Player": {
				"CardCollection": [
					{"card_id": "2ed-136-word-of-command", "card_name": "Word of Command", "count": 2, "deck_counts": [2]},
					{"card_id": "2ed-161-lightning-bolt", "card_name": "Lightning Bolt", "count": 4, "deck_counts": [4]}
				],
				"BonusDuelCards": [
					{"CardID": "2ed-136-word-of-command"},
					{"CardID": "2ed-161-lightning-bolt"}
				]
			},
			"Tiles": [
				[
					{
						"City": {
							"Name": "Test City",
							"CardsForSale": [
								{"CardID": "2ed-136-word-of-command"},
								{"CardID": "2ed-161-lightning-bolt"}
							]
						}
					}
				]
			]
		}
	}`)

	saveData, err := deserializeSave(saveJSON)
	if err != nil {
		t.Fatalf("failed to deserialize save with removed cards: %v", err)
	}

	if saveData.World == nil || saveData.World.Player == nil {
		t.Fatal("expected non-nil world and player")
	}
	if len(saveData.World.Player.CardCollection) != 1 {
		t.Fatalf("expected exactly 1 card in collection, got %d", len(saveData.World.Player.CardCollection))
	}
	bolt := domain.FindCardByName("Lightning Bolt")
	if saveData.World.Player.CardCollection.GetTotalCount(bolt) != 4 {
		t.Errorf("expected 4 Lightning Bolts, got %d", saveData.World.Player.CardCollection.GetTotalCount(bolt))
	}
	if len(saveData.World.Player.BonusDuelCards) != 1 || saveData.World.Player.BonusDuelCards[0].Name() != "Lightning Bolt" {
		t.Fatalf("expected BonusDuelCards to only contain Lightning Bolt, got %v", saveData.World.Player.BonusDuelCards)
	}
	cityCards := saveData.World.Tiles[0][0].City.CardsForSale
	if len(cityCards) != 1 || cityCards[0].Name() != "Lightning Bolt" {
		t.Fatalf("expected CardsForSale to only contain Lightning Bolt, got %v", cityCards)
	}
}

func TestLoadGameWithoutCamera(t *testing.T) {
	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "test_no_camera.json")

	saveJSON := `{
		"Name": "Test",
		"GameID": "test-123",
		"Version": 1,
		"SavedAt": "2026-09-14T18:00:00Z",
		"World": {
			"W": 1,
			"H": 1,
			"Tiles": [[{}]],
			"Player": {
				"X": 100,
				"Y": 200
			}
		}
	}`

	if err := os.WriteFile(savePath, []byte(saveJSON), 0644); err != nil {
		t.Fatalf("Failed to write save file: %v", err)
	}

	loadedLevel, err := LoadGame(savePath)
	if err != nil {
		t.Fatalf("Failed to load game: %v", err)
	}

	if err := loadedLevel.RebuildSprites(); err != nil {
		t.Fatalf("Failed to rebuild sprites: %v", err)
	}

	if loadedLevel.Camera == nil {
		t.Fatal("expected Camera to be initialized after RebuildSprites")
	}

	loc := loadedLevel.Camera.Loc()
	if loc.X != 100 || loc.Y != 200 {
		t.Errorf("expected Camera loc (100, 200), got (%d, %d)", loc.X, loc.Y)
	}
}

func TestLoadOlderSaveFile(t *testing.T) {
	savePath := filepath.Join("testdata", "test-save-v1.json")
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Fatalf("older save file not found at %s", savePath)
	}

	loadedLevel, err := LoadGame(savePath)
	if err != nil {
		t.Fatalf("Failed to load older save: %v", err)
	}

	if err := loadedLevel.RebuildSprites(); err != nil {
		t.Fatalf("Failed to rebuild sprites: %v", err)
	}

	if loadedLevel.Player == nil {
		t.Fatal("expected Player to be non-nil")
	}

	if loadedLevel.Camera == nil {
		t.Fatal("expected Camera to be initialized")
	}

	loc := loadedLevel.Camera.Loc()
	playerLoc := loadedLevel.Player.Loc()
	if loc != playerLoc {
		t.Errorf("expected Camera loc %v to match Player loc %v", loc, playerLoc)
	}
}
