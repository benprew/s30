//go:build js

package save

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/benprew/s30/game/save/internal/browserstore"
)

const (
	webSaveDir       = "localStorage://s30/saves"
	webSaveKeyPrefix = "s30.save."
)

// SaveDir returns the browser-local virtual save directory.
func SaveDir() (string, error) {
	_, err := browserstore.Open()
	if err != nil {
		return "", err
	}
	return webSaveDir, nil
}

func getSaveFilePath(saveName string) (string, error) {
	filename := fmt.Sprintf("%s_%s.json", saveName, time.Now().Format("2006-01-02_15-04-05"))
	return webSaveDir + "/" + filename, nil
}

func writeSave(saveName, gameID string, data []byte) (string, error) {
	storage, err := browserstore.Open()
	if err != nil {
		return "", err
	}
	savePath, err := getSaveFilePath(saveName)
	if err != nil {
		return "", err
	}
	key, err := webKeyForPath(savePath)
	if err != nil {
		return "", err
	}
	encoded, err := browserstore.Encode(data)
	if err != nil {
		return "", err
	}
	if err := storage.Set(key, encoded); err != nil {
		return "", fmt.Errorf("write browser save: %w", err)
	}
	pruneOldBrowserSaves(storage, gameID, key)
	return savePath, nil
}

func readSave(savePath string) ([]byte, error) {
	storage, err := browserstore.Open()
	if err != nil {
		return nil, err
	}
	key, err := webKeyForPath(savePath)
	if err != nil {
		return nil, err
	}
	value, found, err := storage.Get(key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("save does not exist")
	}
	return browserstore.Decode(value)
}

// ListSaves returns browser saves for the supplied virtual directory.
func ListSaves(saveDir string) ([]SaveInfo, error) {
	if saveDir != webSaveDir {
		return nil, fmt.Errorf("unsupported browser save directory %q", saveDir)
	}
	storage, err := browserstore.Open()
	if err != nil {
		return nil, err
	}
	entries, err := browserSaveEntries(storage)
	if err != nil {
		return nil, err
	}

	saves := make([]SaveInfo, 0, len(entries))
	for _, entry := range entries {
		data, err := browserstore.Decode(entry.Value)
		if err != nil {
			continue
		}
		info, err := parseSaveInfoData(webPathForKey(entry.Key), data)
		if err == nil {
			saves = append(saves, info)
		}
	}
	return newestSaves(saves), nil
}

func browserSaveEntries(storage browserstore.Store) ([]browserstore.Entry, error) {
	return storage.Entries(webSaveKeyPrefix)
}

func pruneOldBrowserSaves(storage browserstore.Store, gameID, keepKey string) {
	if gameID == "" {
		return
	}
	entries, err := browserSaveEntries(storage)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.Key == keepKey {
			continue
		}
		data, err := browserstore.Decode(entry.Value)
		if err != nil {
			continue
		}
		info, err := parseSaveInfoData(webPathForKey(entry.Key), data)
		if err == nil && info.GameID == gameID {
			_ = storage.Remove(entry.Key)
		}
	}
}

func webKeyForPath(savePath string) (string, error) {
	prefix := webSaveDir + "/"
	if !strings.HasPrefix(savePath, prefix) {
		return "", fmt.Errorf("invalid browser save path %q", savePath)
	}
	filename := strings.TrimPrefix(savePath, prefix)
	if filename == "" || path.Base(filename) != filename {
		return "", fmt.Errorf("invalid browser save path %q", savePath)
	}
	return webSaveKeyPrefix + filename, nil
}

func webPathForKey(key string) string {
	return webSaveDir + "/" + strings.TrimPrefix(key, webSaveKeyPrefix)
}
