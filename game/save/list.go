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
	GameID  string
	SavedAt time.Time
	Path    string
}

// ListSaves returns metadata for the save files in the given directory, keeping
// only the latest save of each game (grouped by GameID) and sorted newest
// first. If the directory doesn't exist, it returns an empty slice with no
// error.
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

	return dedupByGame(saves), nil
}

// dedupByGame keeps only the first (newest) save of each game. Saves predating
// game ids (empty GameID) are grouped by path so each remains visible.
func dedupByGame(saves []SaveInfo) []SaveInfo {
	seen := make(map[string]bool, len(saves))
	deduped := make([]SaveInfo, 0, len(saves))
	for _, s := range saves {
		key := s.GameID
		if key == "" {
			key = s.Path
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, s)
	}
	return deduped
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
		GameID  string    `json:"game_id"`
		SavedAt time.Time `json:"saved_at"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return SaveInfo{}, err
	}

	return SaveInfo{
		Name:    header.Name,
		GameID:  header.GameID,
		SavedAt: header.SavedAt,
		Path:    path,
	}, nil
}

// pruneOldSaves removes every save of game gameID except keepPath, enforcing
// the "only keep the latest version" rule after a fresh save is written.
func pruneOldSaves(saveDir, gameID, keepPath string) {
	if gameID == "" {
		return
	}
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(saveDir, e.Name())
		if path == keepPath {
			continue
		}
		info, err := parseSaveInfo(path)
		if err != nil || info.GameID != gameID {
			continue
		}
		os.Remove(path)
	}
}
