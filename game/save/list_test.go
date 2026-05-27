package save

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListSaves(t *testing.T) {
	tmpDir := t.TempDir()

	saves, err := ListSaves(tmpDir)
	if err != nil {
		t.Fatalf("ListSaves on empty dir: %v", err)
	}
	if len(saves) != 0 {
		t.Fatalf("expected 0 saves, got %d", len(saves))
	}

	// Create some fake save files
	save1 := `{"name":"quicksave","version":1,"saved_at":"2026-01-15T10:30:00Z","world":null}`
	save2 := `{"name":"autosave","version":1,"saved_at":"2026-02-20T14:00:00Z","world":null}`

	file1 := filepath.Join(tmpDir, "quicksave_2026-01-15_10-30-00.json")
	file2 := filepath.Join(tmpDir, "autosave_2026-02-20_14-00-00.json")

	if writeErr := os.WriteFile(file1, []byte(save1), 0644); writeErr != nil {
		t.Fatal(writeErr)
	}
	// Ensure file2 is newer
	time.Sleep(10 * time.Millisecond)
	if writeErr := os.WriteFile(file2, []byte(save2), 0644); writeErr != nil {
		t.Fatal(writeErr)
	}

	saves, err = ListSaves(tmpDir)
	if err != nil {
		t.Fatalf("ListSaves: %v", err)
	}
	if len(saves) != 2 {
		t.Fatalf("expected 2 saves, got %d", len(saves))
	}

	// Should be sorted newest first
	if saves[0].Name != "autosave" {
		t.Errorf("expected newest save first, got %s", saves[0].Name)
	}
	if saves[1].Name != "quicksave" {
		t.Errorf("expected oldest save second, got %s", saves[1].Name)
	}
	if saves[0].Path != file2 {
		t.Errorf("expected path %s, got %s", file2, saves[0].Path)
	}
}

func TestListSavesDedupByGameID(t *testing.T) {
	tmpDir := t.TempDir()

	// Two saves of the same game (same game_id), plus one of a different game.
	older := `{"name":"Apprentice-Black-abc","game_id":"abc","version":1,"saved_at":"2026-01-15T10:30:00Z","world":null}`
	newer := `{"name":"Apprentice-Black-abc","game_id":"abc","version":1,"saved_at":"2026-03-20T10:30:00Z","world":null}`
	other := `{"name":"Wizard-Red-xyz","game_id":"xyz","version":1,"saved_at":"2026-02-20T14:00:00Z","world":null}`

	writeFile(t, filepath.Join(tmpDir, "Apprentice-Black-abc_2026-01-15_10-30-00.json"), older)
	writeFile(t, filepath.Join(tmpDir, "Apprentice-Black-abc_2026-03-20_10-30-00.json"), newer)
	writeFile(t, filepath.Join(tmpDir, "Wizard-Red-xyz_2026-02-20_14-00-00.json"), other)

	saves, err := ListSaves(tmpDir)
	if err != nil {
		t.Fatalf("ListSaves: %v", err)
	}

	if len(saves) != 2 {
		t.Fatalf("expected 2 saves after dedup, got %d", len(saves))
	}

	// Newest game first, and the "abc" entry must be the newer of its two saves.
	if saves[0].GameID != "abc" {
		t.Errorf("expected newest game 'abc' first, got %q", saves[0].GameID)
	}
	if !saves[0].SavedAt.Equal(mustTime(t, "2026-03-20T10:30:00Z")) {
		t.Errorf("expected latest save of game 'abc', got %s", saves[0].SavedAt)
	}
}

func TestListSavesNonExistentDir(t *testing.T) {
	saves, err := ListSaves("/nonexistent/path/that/doesnt/exist")
	if err != nil {
		t.Fatalf("ListSaves on nonexistent dir should not error: %v", err)
	}
	if len(saves) != 0 {
		t.Fatalf("expected 0 saves, got %d", len(saves))
	}
}

func TestPruneOldSaves(t *testing.T) {
	tmpDir := t.TempDir()

	keep := filepath.Join(tmpDir, "Apprentice-Black-abc_2026-03-20_10-30-00.json")
	old := filepath.Join(tmpDir, "Apprentice-Black-abc_2026-01-15_10-30-00.json")
	other := filepath.Join(tmpDir, "Wizard-Red-xyz_2026-02-20_14-00-00.json")

	writeFile(t, keep, `{"name":"Apprentice-Black-abc","game_id":"abc","version":1,"saved_at":"2026-03-20T10:30:00Z","world":null}`)
	writeFile(t, old, `{"name":"Apprentice-Black-abc","game_id":"abc","version":1,"saved_at":"2026-01-15T10:30:00Z","world":null}`)
	writeFile(t, other, `{"name":"Wizard-Red-xyz","game_id":"xyz","version":1,"saved_at":"2026-02-20T14:00:00Z","world":null}`)

	pruneOldSaves(tmpDir, "abc", keep)

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("expected old save of game 'abc' to be removed")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("expected kept save to remain: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("expected save of other game to remain: %v", err)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
