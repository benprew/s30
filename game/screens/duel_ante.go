package screens

// This is the screen the player sees when they run into an enemy.
// It shows the ante for the duel and asks the player if they want to duel or
// bribe the enemy.

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type DuelAnteScreen struct {
	backgrounds map[string]*ebiten.Image // backgrounds for the different colors
	// there aren't any buttons on this screen, but there is text you can click, which is like
	// a button

}

func (s *DuelAnteScreen) Update() error {
	// Update logic for the duel ante screen
	return nil
}

func (s *DuelAnteScreen) Draw(screen *ebiten.Image) {
	// Draw the duel ante screen
}
