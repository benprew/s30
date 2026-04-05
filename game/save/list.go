package save

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type SaveInfo struct {
	Name    string
	SavedAt time.Time
	Path    string
}

// ListSaves returns metadata for all save files in the given directory,
// sorted by modification time (newest first). If the directory doesn't exist,
// it returns an empty slice with no error.
func ListSaves(saveDir string) ([]SaveInfo, error) {
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var saves []SaveInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(saveDir, e.Name())
		info, err := parseSaveInfo(path)
		if err != nil {
			continue
		}
		saves = append(saves, info)
	}

	sort.Slice(saves, func(i, j int) bool {
		return saves[i].SavedAt.After(saves[j].SavedAt)
	})

	return saves, nil
}

// SaveDir returns the default save directory path.
func SaveDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".s30", "saves"), nil
}

func parseSaveInfo(path string) (SaveInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SaveInfo{}, err
	}

	var header struct {
		Name    string    `json:"name"`
		SavedAt time.Time `json:"saved_at"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return SaveInfo{}, err
	}

	return SaveInfo{
		Name:    header.Name,
		SavedAt: header.SavedAt,
		Path:    path,
	}, nil
}
