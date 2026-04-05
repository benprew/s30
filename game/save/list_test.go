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

	if err := os.WriteFile(file1, []byte(save1), 0644); err != nil {
		t.Fatal(err)
	}
	// Ensure file2 is newer
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(file2, []byte(save2), 0644); err != nil {
		t.Fatal(err)
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

func TestListSavesNonExistentDir(t *testing.T) {
	saves, err := ListSaves("/nonexistent/path/that/doesnt/exist")
	if err != nil {
		t.Fatalf("ListSaves on nonexistent dir should not error: %v", err)
	}
	if len(saves) != 0 {
		t.Fatalf("expected 0 saves, got %d", len(saves))
	}
}
