package screens

// This is the screen the player sees when they run into an enemy.
// It shows the ante for the duel and asks the player if they want to duel or
// bribe the enemy.

import (
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type DuelAnteScreen struct {
	backgrounds map[string]*ebiten.Image // backgrounds for the different colors
	// there aren't any buttons on this screen, but there is text you can click, which is like
	// a button
	lvl *world.Level
	idx int
}

func NewDuelAnteScreen() *DuelAnteScreen {
	return &DuelAnteScreen{backgrounds: make(map[string]*ebiten.Image)}
}

// NewDuelAnteScreenWithEnemy creates a DuelAnteScreen pre-populated with the encountering enemy.
// It receives the level pointer and enemy index so the screen can accept/bribe and modify the level.
func NewDuelAnteScreenWithEnemy(l *world.Level, idx int) *DuelAnteScreen {
	s := NewDuelAnteScreen()
	// TODO: populate screen with enemy data (visage, name, ante)
	_ = l
	_ = idx
	s.lvl = l
	s.idx = idx
	return s
}

// Ensure DuelAnteScreen implements screenui.Screen
func (s *DuelAnteScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	// Key handling: A = Accept duel (mark engaged), B = Bribe (remove enemy), Esc = cancel
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		if s.lvl != nil {
			s.lvl.SetEnemyEngaged(s.idx, true)
		}
		return screenui.WorldScr, nil
	}
	if ebiten.IsKeyPressed(ebiten.KeyB) {
		if s.lvl != nil {
			s.lvl.RemoveEnemyAt(s.idx)
		}
		return screenui.WorldScr, nil
	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		// cancel: clear engaged flag if set
		if s.lvl != nil {
			s.lvl.SetEnemyEngaged(s.idx, false)
		}
		return screenui.WorldScr, nil
	}
	return screenui.DuelAnteScr, nil
}

func (s *DuelAnteScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	// Minimal UI: instructions for Accept/Bribe/Cancel
	ebitenutil.DebugPrint(screen, "Encounter! Press A to Accept duel, B to Bribe (remove), Esc to Cancel")
}

func (s *DuelAnteScreen) IsFramed() bool { return false }
