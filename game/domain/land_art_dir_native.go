//go:build !js

package domain

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Linux players get some slack on capitalization; on Windows and macOS the
// filesystem already matches these case-insensitively.
var landArtDirNames = []string{"landArt", "landart", "LandArt"}

// landArtFS returns the player's land art folder. It looks beside the
// executable first, so dropping landArt next to the game works, then in the
// working directory, which is where "go run ." leaves a developer.
func landArtFS() (fs.FS, bool) {
	return landArtFSIn(landArtSearchDirs())
}

func landArtSearchDirs() []string {
	dirs := make([]string, 0, 2)
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	return dirs
}

func landArtFSIn(dirs []string) (fs.FS, bool) {
	for _, dir := range dirs {
		for _, name := range landArtDirNames {
			candidate := filepath.Join(dir, name)
			info, err := os.Stat(candidate)
			if err != nil || !info.IsDir() {
				continue
			}
			return os.DirFS(candidate), true
		}
	}
	return nil, false
}
