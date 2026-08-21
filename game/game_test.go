package game

import (
	"testing"

	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

type stubScreen struct {
	overlay bool
}

type closableStubScreen struct {
	stubScreen
	closed bool
}

func (s *closableStubScreen) Close() {
	s.closed = true
}

func TestEndScreenBGM(t *testing.T) {
	tests := []struct {
		screen screenui.ScreenName
		want   gameaudio.BGM
	}{
		{screenui.DuelWinScr, gameaudio.BGMVictory},
		{screenui.DuelLoseScr, gameaudio.BGMDefeat},
		{screenui.GameWinScr, gameaudio.BGMWinGame},
		{screenui.GameLoseScr, gameaudio.BGMDeath},
	}
	for _, test := range tests {
		got, ok := endScreenBGM(test.screen)
		if !ok || got != test.want {
			t.Errorf("endScreenBGM(%v) = %v, %t; want %v, true", test.screen, got, ok, test.want)
		}
	}
}

func TestDuelScreenStopsBGM(t *testing.T) {
	am := gameaudio.NewAudioManager()
	am.Mute()
	am.PlayBGM(gameaudio.BGMBattle)
	g := &Game{audio: am}

	g.updateBGM(screenui.DuelScr)

	if got := am.CurrentBGM(); got != gameaudio.BGMNone {
		t.Fatalf("duel BGM = %v, want BGMNone", got)
	}
}

func TestCityTransitionKeepsCastleBGM(t *testing.T) {
	am := gameaudio.NewAudioManager()
	am.Mute()
	am.PlayBGM(gameaudio.BGMCastleRed)
	g := &Game{audio: am}

	g.updateBGM(screenui.CityScr)

	if got := am.CurrentBGM(); got != gameaudio.BGMCastleRed {
		t.Fatalf("city BGM = %v, want castle entry theme", got)
	}
}

func TestFixedScreenBGM(t *testing.T) {
	tests := []struct {
		screen screenui.ScreenName
		want   gameaudio.BGM
	}{
		{screenui.StartScr, gameaudio.BGMTitle},
		{screenui.WorldScr, gameaudio.BGMCity},
		{screenui.MiniMapScr, gameaudio.BGMCity},
		{screenui.DungeonEntryScr, gameaudio.BGMDungeon},
		{screenui.DungeonScr, gameaudio.BGMDungeon},
	}
	for _, test := range tests {
		got, ok := fixedScreenBGM(test.screen)
		if !ok || got != test.want {
			t.Errorf("fixedScreenBGM(%v) = %v, %t; want %v, true", test.screen, got, ok, test.want)
		}
	}
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

func TestNavigateClosesPreviousLifecycleScreen(t *testing.T) {
	previous := &closableStubScreen{}
	g := newTestGame()
	g.screenMap[screenui.WorldScr] = previous

	g.navigate(screenui.MiniMapScr)

	if !previous.closed {
		t.Fatal("navigate did not close the previous lifecycle screen")
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
