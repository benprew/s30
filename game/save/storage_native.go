//go:build !js

package save

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var saveDirConfig struct {
	sync.RWMutex
	path string
}

// SetSaveDir configures an app-private save directory. Passing an empty path
// restores the desktop default.
func SetSaveDir(path string) {
	saveDirConfig.Lock()
	defer saveDirConfig.Unlock()
	saveDirConfig.path = path
}

// SaveDir returns the directory used for save files.
func SaveDir() (string, error) {
	saveDirConfig.RLock()
	configured := saveDirConfig.path
	saveDirConfig.RUnlock()
	if configured != "" {
		return configured, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".s30", "saves"), nil
}

func getSaveFilePath(saveName string) (string, error) {
	saveDir, err := SaveDir()
	if err != nil {
		return "", err
	}
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.json", saveName, timestamp)
	return filepath.Join(saveDir, filename), nil
}

func writeSave(saveName, gameID string, data []byte) (string, error) {
	savePath, err := getSaveFilePath(saveName)
	if err != nil {
		return "", fmt.Errorf("get save path: %w", err)
	}
	saveDir := filepath.Dir(savePath)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("create save directory: %w", err)
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("write save file: %w", err)
	}
	pruneOldSaves(saveDir, gameID, savePath)
	return savePath, nil
}

func readSave(savePath string) ([]byte, error) {
	return os.ReadFile(savePath)
}

// ListSaves returns the latest save for each game, sorted newest first.
func ListSaves(saveDir string) ([]SaveInfo, error) {
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var saves []SaveInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(saveDir, entry.Name())
		info, err := parseSaveInfo(path)
		if err == nil {
			saves = append(saves, info)
		}
	}
	return newestSaves(saves), nil
}

func parseSaveInfo(path string) (SaveInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SaveInfo{}, err
	}
	return parseSaveInfoData(path, data)
}

func pruneOldSaves(saveDir, gameID, keepPath string) {
	if gameID == "" {
		return
	}
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(saveDir, entry.Name())
		if path == keepPath {
			continue
		}
		info, err := parseSaveInfo(path)
		if err == nil && info.GameID == gameID {
			_ = os.Remove(path)
		}
	}
}
