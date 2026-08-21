package save

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/benprew/s30/game/world"
)

const (
	currentSaveVersion         = 1
	legacyMovementSpeedMinimum = 5
	legacyMovementSpeedMaximum = 11
	legacyMovementSpeedScale   = 6
)

// SaveGame writes the level to disk using the game's stable name and keeps only
// the latest save of that game by pruning any earlier ones.
func SaveGame(level *world.Level) (string, error) {
	saveName := level.SaveName()

	jsonData, err := serializeSave(level)
	if err != nil {
		return "", fmt.Errorf("failed to serialize save data: %w", err)
	}

	savePath, err := writeSave(saveName, level.GameID, jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to persist save data: %w", err)
	}
	return savePath, nil
}

func serializeSave(level *world.Level) ([]byte, error) {
	saveData := &SaveData{
		Name:    level.SaveName(),
		GameID:  level.GameID,
		Version: currentSaveVersion,
		SavedAt: time.Now(),
		World:   level,
	}

	return json.Marshal(saveData)
}

func deserializeSave(jsonData []byte) (*SaveData, error) {
	var saveData SaveData
	if err := json.Unmarshal(jsonData, &saveData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	if saveData.Version != currentSaveVersion {
		return nil, fmt.Errorf("unsupported save version: %d", saveData.Version)
	}
	normalizeMovementSpeed(&saveData)

	return &saveData, nil
}

func normalizeMovementSpeed(saveData *SaveData) {
	if saveData.World == nil {
		return
	}
	if saveData.World.Player != nil {
		saveData.World.Player.MoveSpeed = normalizeSavedMovementSpeed(saveData.World.Player.MoveSpeed)
	}
	for i := range saveData.World.Enemies {
		saveData.World.Enemies[i].MoveSpeed = normalizeSavedMovementSpeed(saveData.World.Enemies[i].MoveSpeed)
	}
}

func normalizeSavedMovementSpeed(speed float64) float64 {
	if speed < legacyMovementSpeedMinimum || speed > legacyMovementSpeedMaximum {
		return speed
	}
	return speed / legacyMovementSpeedScale
}

func LoadGame(savePath string) (*world.Level, error) {
	jsonData, err := readSave(savePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read save %s: %w", savePath, err)
	}

	saveData, err := deserializeSave(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize save data: %w", err)
	}

	return saveData.World, nil
}
