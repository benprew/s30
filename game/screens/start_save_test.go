package screens

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benprew/s30/game/save"
)

func TestStartScreenChecksForSavesAfterConstruction(t *testing.T) {
	saveDir := filepath.Join(t.TempDir(), "saves")
	save.SetSaveDir(saveDir)
	t.Cleanup(func() { save.SetSaveDir("") })

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		t.Fatal(err)
	}
	savePath := filepath.Join(saveDir, "android-save.json")
	contents := `{"name":"Android","game_id":"android","version":1,"saved_at":"2026-08-19T12:00:00Z","world":null}`
	if err := os.WriteFile(savePath, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	screen := NewStartScreen()
	if screen.hasSaves {
		t.Fatal("save discovery ran during construction")
	}
	screen.refreshHasSaves()
	if !screen.hasSaves {
		t.Fatal("save discovery did not find the configured Android save")
	}
}
