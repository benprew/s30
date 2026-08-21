package mobile

import (
	"fmt"
	"path/filepath"

	"github.com/benprew/s30/game"
	"github.com/benprew/s30/game/save"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

var currentGame = game.NewLoadingGame()

func init() {
	mobile.SetGame(currentGame)
}

// Dummy is required by gomobile.
func Dummy() {}

// SetSaveDir configures saves beneath Android's app-private files directory.
func SetSaveDir(appFilesDir string) {
	save.SetSaveDir(filepath.Join(appFilesDir, "saves"))
}

// SaveGame persists the active Android game when the activity is paused.
func SaveGame() {
	if err := currentGame.SaveGame(); err != nil {
		fmt.Printf("Error auto-saving Android game: %v\n", err)
	}
}
