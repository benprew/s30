package game

import (
	"testing"

	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

type stubScreen struct {
	overlay bool
}

func (s *stubScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	return screenui.NoScr, nil, nil
}
func (s *stubScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {}
func (s *stubScreen) IsFramed() bool                                     { return false }
func (s *stubScreen) IsOverlay() bool                                    { return s.overlay }

func newTestGame() *Game {
	return &Game{
		currentScreen: screenui.WorldScr,
		prevScreen:    screenui.WorldScr,
		screenMap: map[screenui.ScreenName]screenui.Screen{
			screenui.WorldScr:   &stubScreen{},
			screenui.MiniMapScr: &stubScreen{overlay: true},
		},
	}
}

func TestNavigateToNewScreen(t *testing.T) {
	g := newTestGame()
	g.navigate(screenui.MiniMapScr)
	if g.currentScreen != screenui.MiniMapScr {
		t.Errorf("currentScreen = %v, want MiniMapScr", g.currentScreen)
	}
	if g.prevScreen != screenui.WorldScr {
		t.Errorf("prevScreen = %v, want WorldScr", g.prevScreen)
	}
}

func TestNavigatePopReturnsToPrevious(t *testing.T) {
	g := newTestGame()
	g.navigate(screenui.MiniMapScr)
	g.navigate(screenui.PopScr)
	if g.currentScreen != screenui.WorldScr {
		t.Errorf("after Pop currentScreen = %v, want WorldScr", g.currentScreen)
	}
}

func TestNavigateNoScrIsNoOp(t *testing.T) {
	g := newTestGame()
	g.navigate(screenui.NoScr)
	if g.currentScreen != screenui.WorldScr || g.prevScreen != screenui.WorldScr {
		t.Errorf("NoScr changed screens: cur=%v prev=%v", g.currentScreen, g.prevScreen)
	}
}

func TestNavigateSameScreenDoesNotClobberPrev(t *testing.T) {
	g := newTestGame()
	g.navigate(screenui.CityScr)
	g.navigate(screenui.CityScr) // staying put must not lose the back target
	if g.prevScreen != screenui.WorldScr {
		t.Errorf("prevScreen = %v, want WorldScr (unchanged when staying)", g.prevScreen)
	}
}
