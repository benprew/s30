package screens

// This is the screen the player sees when they run into an enemy.
// It shows the ante for the duel and asks the player if they want to duel or
// bribe the enemy.

import (
	"github.com/benprew/s30/game/entities"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

type DuelAnteScreen struct {
	backgrounds map[string]*ebiten.Image // backgrounds for the different colors
	// there aren't any buttons on this screen, but there is text you can click, which is like
	// a button
}

func NewDuelAnteScreen() *DuelAnteScreen {
	return &DuelAnteScreen{backgrounds: make(map[string]*ebiten.Image)}
}

// NewDuelAnteScreenWithEnemy creates a DuelAnteScreen pre-populated with the encountering enemy.
func NewDuelAnteScreenWithEnemy(e entities.Enemy) *DuelAnteScreen {
	s := NewDuelAnteScreen()
	// TODO: populate screen with enemy data (visage, name, ante)
	_ = e
	return s
}

// Ensure DuelAnteScreen implements screenui.Screen
func (s *DuelAnteScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	// For now, pressing Escape returns to World screen. Actual duel flow will be added later.
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil
	}
	return screenui.DuelAnteScr, nil
}

func (s *DuelAnteScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	// Minimal placeholder: clear screen to black (do nothing else for now)
}

func (s *DuelAnteScreen) IsFramed() bool { return false }
