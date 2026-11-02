//go:build !js

package domain

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func writeLandArtDir(t *testing.T, parent, dirName, fileName string) string {
	t.Helper()
	dir := filepath.Join(parent, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("not a real image"), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", fileName, err)
	}
	return dir
}

func TestLandArtFSInFindsFolder(t *testing.T) {
	root := t.TempDir()
	writeLandArtDir(t, root, "landArt", "Forest.full.png")

	fsys, found := landArtFSIn([]string{root})
	if !found {
		t.Fatal("landArtFSIn did not find the landArt folder")
	}
	if got := ScanLandArt(fsys).Paths("forest"); len(got) != 1 {
		t.Errorf("scanned art = %v, want 1 forest", got)
	}
}

func TestLandArtFSInSearchesEveryDirInOrder(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	writeLandArtDir(t, second, "landArt", "Island.full.png")

	fsys, found := landArtFSIn([]string{first, second})
	if !found {
		t.Fatal("landArtFSIn did not fall through to the second directory")
	}
	if got := ScanLandArt(fsys).Paths("island"); len(got) != 1 {
		t.Errorf("scanned art = %v, want 1 island", got)
	}
}

// This one's a little confusing but it's testing that multiple different landArt
// folders aren't merged, only the first one is used.
func TestLandArtFSInPrefersEarlierDir(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	writeLandArtDir(t, first, "landArt", "Forest.full.png")
	writeLandArtDir(t, second, "landArt", "Island.full.png")

	fsys, found := landArtFSIn([]string{first, second})
	if !found {
		t.Fatal("landArtFSIn found nothing")
	}
	lib := ScanLandArt(fsys)
	if len(lib.Paths("forest")) != 1 || len(lib.Paths("island")) != 0 {
		t.Errorf("expected only the first directory's art, got forest=%v island=%v",
			lib.Paths("forest"), lib.Paths("island"))
	}
}

func TestLandArtFSInLowercaseFolderName(t *testing.T) {
	root := t.TempDir()
	writeLandArtDir(t, root, "landart", "swamp.png")

	if _, found := landArtFSIn([]string{root}); !found {
		t.Error("a folder named landart was not found")
	}
}

func TestLandArtFSInMissingFolder(t *testing.T) {
	fsys, found := landArtFSIn([]string{t.TempDir()})
	if found {
		t.Error("landArtFSIn reported a folder that does not exist")
	}
	if fsys != nil {
		t.Error("landArtFSIn returned a filesystem alongside found=false")
	}
}

func TestLandArtFSInIgnoresPlainFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "landArt"), []byte("not a folder"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if _, found := landArtFSIn([]string{root}); found {
		t.Error("a file named landArt was treated as a folder")
	}
}

func TestLandArtFSInNoDirs(t *testing.T) {
	if _, found := landArtFSIn(nil); found {
		t.Error("landArtFSIn(nil) reported a folder")
	}
}

func TestInitLandArtFromPublishesLibrary(t *testing.T) {
	previous := landArtLibrary
	t.Cleanup(func() { landArtLibrary = previous })

	lib := initLandArtFrom(fstest.MapFS{"beta/Mountain.full.png": &fstest.MapFile{}})
	if lib == nil {
		t.Fatal("initLandArtFrom returned nil")
	}
	if got := lib.Total(); got != 1 {
		t.Errorf("Total() = %d, want 1", got)
	}
	if LandArt() != lib {
		t.Error("LandArt() does not return the library that was just initialized")
	}
	if got := LandArt().Paths("Mountain"); len(got) != 1 {
		t.Errorf("LandArt().Paths(\"Mountain\") = %v, want 1 entry", got)
	}
}

func TestLandArtBeforeInitIsSafe(t *testing.T) {
	previous := landArtLibrary
	t.Cleanup(func() { landArtLibrary = previous })

	landArtLibrary = nil
	if got := LandArt().Total(); got != 0 {
		t.Errorf("LandArt().Total() = %d before init, want 0", got)
	}
	if got := LandArt().Paths("forest"); got != nil {
		t.Errorf("LandArt().Paths() = %v before init, want nil", got)
	}
}
