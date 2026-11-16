package updates

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AndroidInstaller interface {
	InstallUpdate(apkPath string) error
	OpenDownload(url string) error
}

var androidUpdate struct {
	sync.RWMutex
	dir         string
	versionCode int64
	installer   AndroidInstaller
}

// ConfigureAndroid supplies the private cache directory and the system installer.
func ConfigureAndroid(cacheDir string, versionCode int64, installer AndroidInstaller) {
	androidUpdate.Lock()
	androidUpdate.dir = cacheDir
	androidUpdate.versionCode = versionCode
	androidUpdate.installer = installer
	androidUpdate.Unlock()
}

func cleanupAndroidUpdates(cacheDir string, installedCode int64) {
	if cacheDir == "" {
		return
	}
	root := filepath.Join(cacheDir, "updates")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".s30-update-") {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		data, readErr := os.ReadFile(filepath.Join(dir, "version-code"))
		code, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		info, statErr := entry.Info()
		installed := readErr == nil && parseErr == nil && code <= installedCode
		stale := statErr == nil && time.Since(info.ModTime()) > 48*time.Hour
		if installed || stale {
			os.RemoveAll(dir)
		}
	}
}

func androidVersionCode() int64 {
	androidUpdate.RLock()
	defer androidUpdate.RUnlock()
	return androidUpdate.versionCode
}
