package save

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/benprew/s30/game/world"
)

func SaveGame(level *world.Level, saveName string) (string, error) {
	jsonData, err := serializeSave(level, saveName)
	if err != nil {
		return "", fmt.Errorf("failed to serialize save data: %w", err)
	}

	savePath, err := getSaveFilePath(saveName)
	if err != nil {
		return "", fmt.Errorf("failed to get save path: %w", err)
	}

	saveDir := filepath.Dir(savePath)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}

	if err := os.WriteFile(savePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write save file: %w", err)
	}

	return savePath, nil
}

func serializeSave(level *world.Level, saveName string) ([]byte, error) {
	saveData := &SaveData{
		Name:    saveName,
		Version: 1,
		SavedAt: time.Now(),
		World:   level,
	}

	return json.MarshalIndent(saveData, "", "  ")
}

func deserializeSave(jsonData []byte) (*SaveData, error) {
	var saveData SaveData
	if err := json.Unmarshal(jsonData, &saveData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	if saveData.Version != 1 {
		return nil, fmt.Errorf("unsupported save version: %d", saveData.Version)
	}

	return &saveData, nil
}

func LoadGame(savePath string) (*world.Level, error) {
	jsonData, err := os.ReadFile(savePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", savePath, err)
	}

	saveData, err := deserializeSave(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize save data: %w", err)
	}

	return saveData.World, nil
}

func getSaveFilePath(saveName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	saveDir := filepath.Join(homeDir, ".s30", "saves")

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.json", saveName, timestamp)

	return filepath.Join(saveDir, filename), nil
}
