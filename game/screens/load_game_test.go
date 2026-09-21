package screens

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/benprew/s30/game/save"
	"github.com/benprew/s30/game/ui/screenui"
)

func TestLoadGameScreenIsOverlay(t *testing.T) {
	scr := NewLoadGameScreen()
	if !scr.IsOverlay() {
		t.Error("LoadGameScreen must be an overlay")
	}
	if scr.IsFramed() {
		t.Error("LoadGameScreen must not be framed")
	}
}

func TestLoadGameScreenBackBtnReturnsPopScr(t *testing.T) {
	scr := NewLoadGameScreen()
	if scr.backBtn == nil {
		t.Fatal("expected backBtn to be initialized")
	}
	clickPt := scr.backBtn.Bounds.Min.Add(image.Pt(5, 5))
	name := scr.handleSelection(clickPt, true)
	if name != screenui.PopScr {
		t.Errorf("name = %v, want PopScr", name)
	}
}

func TestLoadGameScreenDiscoversSavesAndLoads(t *testing.T) {
	saveDir := filepath.Join(t.TempDir(), "saves")
	save.SetSaveDir(saveDir)
	t.Cleanup(func() { save.SetSaveDir("") })

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		t.Fatal(err)
	}
	savePath := filepath.Join(saveDir, "test-save.json")
	contents := `{"name":"Hero","game_id":"hero","version":1,"saved_at":"2026-09-19T12:00:00Z","world":null}`
	if err := os.WriteFile(savePath, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	scr := NewLoadGameScreen()
	if len(scr.saveButtons) != 1 {
		t.Fatalf("expected 1 save button, got %d", len(scr.saveButtons))
	}

	clickPt := scr.saveButtons[0].Bounds.Min.Add(image.Pt(5, 5))
	name := scr.handleSelection(clickPt, true)
	if name != screenui.WorldScr {
		t.Errorf("name = %v, want WorldScr", name)
	}
	if scr.SelectedSave != savePath {
		t.Errorf("SelectedSave = %q, want %q", scr.SelectedSave, savePath)
	}
}
