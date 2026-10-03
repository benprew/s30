package domain

import (
	"io/fs"
	"path"
	"slices"
	"strings"
)

var landArtExtensions = []string{".jpg", ".jpeg", ".png"}

var landArtNames = lowercaseBasicLandNames()

func lowercaseBasicLandNames() []string {
	names := make([]string, 0, len(basicLands))
	for name := range basicLands {
		names = append(names, strings.ToLower(name))
	}
	slices.Sort(names)
	return names
}

// classifyLandArt reports which basic land an art file depicts. Any filename
// beginning with a basic land name counts, so the per-set names that art packs
// already use ("Island.full.jpg", "mountain-beta-3.png") need no renaming. The
// returned name is lowercase, matching the keys LandArtLibrary is indexed by.
func classifyLandArt(filename string) (string, bool) {
	base := strings.ToLower(path.Base(filename))
	ext := path.Ext(base)
	if !slices.Contains(landArtExtensions, ext) {
		return "", false
	}

	stem := strings.TrimSuffix(base, ext)
	for _, name := range landArtNames {
		if strings.HasPrefix(stem, name) {
			return name, true
		}
	}
	return "", false
}

// LandArtLibrary holds the alternate basic land art a player has supplied,
// grouped by basic land name. It stores paths rather than decoded images so
// that a large art pack costs nothing until the art is actually drawn.
type LandArtLibrary struct {
	fsys fs.FS
	arts map[string][]string
}

// ScanLandArt collects every image in fsys whose filename identifies a basic
// land. It walks the whole tree instead of a single level so that a player can
// either drop every swamp into one folder or split them up by set.
func ScanLandArt(fsys fs.FS) *LandArtLibrary {
	lib := &LandArtLibrary{fsys: fsys, arts: make(map[string][]string)}
	if fsys == nil {
		return lib
	}

	// An unreadable entry skips that entry rather than discarding the art
	// already found, so one bad folder can't cost the player their whole pack.
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if land, ok := classifyLandArt(d.Name()); ok {
			lib.arts[land] = append(lib.arts[land], p)
		}
		return nil
	})

	for _, paths := range lib.arts {
		slices.Sort(paths)
	}
	return lib
}

// Paths returns the art available for a basic land, which may be named either
// as a card ("Forest") or as a library key ("forest"). The result is nil when
// the player supplied no art for that land.
func (l *LandArtLibrary) Paths(landName string) []string {
	if l == nil {
		return nil
	}
	return l.arts[strings.ToLower(landName)]
}

// Total returns how many art files the library holds across every land.
func (l *LandArtLibrary) Total() int {
	if l == nil {
		return 0
	}
	total := 0
	for _, paths := range l.arts {
		total += len(paths)
	}
	return total
}
