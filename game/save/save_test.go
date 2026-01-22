package save

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

// func TestSerializePlayer(t *testing.T) {
// 	player, err := domain.NewPlayer("TestPlayer", nil, false)
// 	if err != nil {
// 		t.Fatalf("Failed to create player: %v", err)
// 	}
//
// 	player.Gold = 500
// 	player.Food = 20
// 	player.Life = 10
//
// 	pd, err := serializePlayer(player)
// 	if err != nil {
// 		t.Fatalf("Failed to serialize player: %v", err)
// 	}
//
// 	if pd.Name != "TestPlayer" {
// 		t.Errorf("Expected name 'TestPlayer', got '%s'", pd.Name)
// 	}
// 	if pd.Gold != 500 {
// 		t.Errorf("Expected gold 500, got %d", pd.Gold)
// 	}
// 	if pd.Food != 20 {
// 		t.Errorf("Expected food 20, got %d", pd.Food)
// 	}
// 	if pd.Life != 10 {
// 		t.Errorf("Expected life 10, got %d", pd.Life)
// 	}
// }
//
// func TestSerializeWorld(t *testing.T) {
// 	player, err := domain.NewPlayer("TestPlayer", nil, false)
// 	if err != nil {
// 		t.Fatalf("Failed to create player: %v", err)
// 	}
//
// 	level, err := world.NewLevel(player)
// 	if err != nil {
// 		t.Fatalf("Failed to create level: %v", err)
// 	}
//
// 	wd, err := serializeWorld(level)
// 	if err != nil {
// 		t.Fatalf("Failed to serialize world: %v", err)
// 	}
//
// 	if wd.Width != 47 {
// 		t.Errorf("Expected width 47, got %d", wd.Width)
// 	}
// 	if wd.Height != 63 {
// 		t.Errorf("Expected height 63, got %d", wd.Height)
// 	}
// 	if len(wd.Cities) == 0 {
// 		t.Error("Expected at least one city")
// 	}
// }

func TestEnemiesSave(t *testing.T) {
	enemies := make([]domain.Enemy, 0)

	enemy, err := domain.NewEnemy("Sedge Beast")
	if err != nil {
		t.Fatalf("Failed to create enemy: %v", err)
	}
	enemy.X = 100
	enemy.Y = 200
	enemy.MoveSpeed = 7
	enemies = append(enemies, enemy)

	level, err := world.NewLevel(&domain.Player{})
	if err != nil {
		t.Fatalf("failed to make world: %v", err)
	}
	level.SetEnemies(enemies)
	jsonData, err := serializeSave(level, "Test Save")
	if err != nil {
		t.Fatalf("Failed to marshal save data: %v", err)
	}

	sd, err := deserializeSave(jsonData)
	if err != nil {
		t.Fatalf("Failed to serialize enemies: %v", err)
	}

	ed := sd.World.Enemies()

	if len(ed) != 1 {
		t.Fatalf("Expected 1 enemy, got %d", len(ed))
	}
	if ed[0].Name() != "Sedge Beast" {
		t.Errorf("Expected character name 'Sedge Beast', got '%s'", ed[0].Name())
	}
	if ed[0].X != 100 {
		t.Errorf("Expected X 100, got %d", ed[0].X)
	}
	if ed[0].Y != 200 {
		t.Errorf("Expected Y 200, got %d", ed[0].Y)
	}
	if ed[0].MoveSpeed != 7 {
		t.Errorf("Expected MoveSpeed 7, got %d", ed[0].MoveSpeed)
	}
}

func TestPlayerSave(t *testing.T) {
	player, err := domain.NewPlayer("TestPlayer", nil, false)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	player.Gold = 999
	player.Food = 33
	player.Life = 7
	player.X = 1500
	player.Y = 2000

	level, err := world.NewLevel(player)
	if err != nil {
		t.Fatalf("Failed to create level: %v", err)
	}

	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "test_save.json")

	jsonData, err := serializeSave(level, "Test Save")
	if err != nil {
		t.Fatalf("Failed to marshal save data: %v", err)
	}

	if err := os.WriteFile(savePath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write save file: %v", err)
	}

	loadedLevel, err := LoadGame(savePath)
	if err != nil {
		t.Fatalf("Failed to load game: %v", err)
	}
	loadedPlayer := loadedLevel.Player

	if loadedPlayer.Name != player.Name {
		t.Errorf("Player name mismatch: expected '%s', got '%s'", player.Name, loadedPlayer.Name)
	}
	if loadedPlayer.Gold != player.Gold {
		t.Errorf("Player gold mismatch: expected %d, got %d", player.Gold, loadedPlayer.Gold)
	}
	if loadedPlayer.Food != player.Food {
		t.Errorf("Player food mismatch: expected %d, got %d", player.Food, loadedPlayer.Food)
	}
	if loadedPlayer.Life != player.Life {
		t.Errorf("Player life mismatch: expected %d, got %d", player.Life, loadedPlayer.Life)
	}
	if loadedPlayer.X != player.X {
		t.Errorf("Player X mismatch: expected %d, got %d", player.X, loadedPlayer.X)
	}
	if loadedPlayer.Y != player.Y {
		t.Errorf("Player Y mismatch: expected %d, got %d", player.Y, loadedPlayer.Y)
	}

	w, h := loadedLevel.Size()
	if w != 47 || h != 63 {
		t.Errorf("Level size mismatch: expected 47x63, got %dx%d", w, h)
	}
}

// func TestEmptyCardCollection(t *testing.T) {
// 	pd := PlayerData{
// 		Name:           "TestPlayer",
// 		X:              0,
// 		Y:              0,
// 		Gold:           100,
// 		Food:           10,
// 		Life:           8,
// 		ActiveDeck:     0,
// 		Amulets:        make(map[int]int),
// 		WorldMagics:    []string{},
// 		CardCollection: []CardCollectionItem{},
// 	}
//
// 	player, err := deserializePlayer(&pd)
// 	if err != nil {
// 		t.Fatalf("Failed to deserialize player with empty collection: %v", err)
// 	}
//
// 	if len(player.CardCollection) != 0 {
// 		t.Errorf("Expected empty card collection, got %d cards", len(player.CardCollection))
// 	}
// }
//
// func TestNoWorldMagics(t *testing.T) {
// 	pd := PlayerData{
// 		Name:           "TestPlayer",
// 		X:              0,
// 		Y:              0,
// 		Gold:           100,
// 		Food:           10,
// 		Life:           8,
// 		ActiveDeck:     0,
// 		Amulets:        make(map[int]int),
// 		WorldMagics:    []string{},
// 		CardCollection: []CardCollectionItem{},
// 	}
//
// 	player, err := deserializePlayer(&pd)
// 	if err != nil {
// 		t.Fatalf("Failed to deserialize player with no world magics: %v", err)
// 	}
//
// 	if len(player.WorldMagics) != 0 {
// 		t.Errorf("Expected no world magics, got %d", len(player.WorldMagics))
// 	}
// }
//
// func TestNoEnemies(t *testing.T) {
// 	enemies := make([]domain.Enemy, 0)
//
// 	ed, err := serializeEnemies(enemies)
// 	if err != nil {
// 		t.Fatalf("Failed to serialize empty enemies: %v", err)
// 	}
//
// 	if len(ed) != 0 {
// 		t.Errorf("Expected no enemies, got %d", len(ed))
// 	}
// }

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

	if err := os.WriteFile(savePath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write save file: %v", err)
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
