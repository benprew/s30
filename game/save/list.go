package save

import (
	"encoding/json"
	"sort"
	"time"
)

type SaveInfo struct {
	Name    string
	GameID  string
	SavedAt time.Time
	Path    string
}

func newestSaves(saves []SaveInfo) []SaveInfo {
	sort.Slice(saves, func(i, j int) bool {
		return saves[i].SavedAt.After(saves[j].SavedAt)
	})
	return dedupByGame(saves)
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

func parseSaveInfoData(path string, data []byte) (SaveInfo, error) {
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
