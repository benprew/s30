package main

import (
	"errors"
	"testing"

	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

type transitionScreen struct {
	next   screenui.ScreenName
	screen screenui.Screen
	err    error
	closed int
}

func (s *transitionScreen) Update(int, int, float64) (screenui.ScreenName, screenui.Screen, error) {
	return s.next, s.screen, s.err
}

func (s *transitionScreen) Draw(*ebiten.Image, int, int, float64) {}
func (s *transitionScreen) IsFramed() bool                        { return false }
func (s *transitionScreen) IsOverlay() bool                       { return false }
func (s *transitionScreen) Close()                                { s.closed++ }

func TestArenaGameFollowsEditorDuelRewardAndChampion(t *testing.T) {
	champion := &transitionScreen{next: screenui.WorldScr}
	win := &transitionScreen{next: screenui.RandomEncounterScr, screen: champion}
	duel := &transitionScreen{next: screenui.DuelWinScr, screen: win}
	ante := &transitionScreen{next: screenui.DuelScr, screen: duel}
	editor := &transitionScreen{next: screenui.DuelAnteScr, screen: ante}
	entry := &transitionScreen{next: screenui.EditDeckScr, screen: editor}
	pointers := 0
	g := &arenaGame{screens: map[screenui.ScreenName]screenui.Screen{screenui.RandomEncounterScr: entry},
		current: screenui.RandomEncounterScr, updatePointer: func() { pointers++ }}
	for _, want := range []screenui.ScreenName{screenui.EditDeckScr, screenui.DuelAnteScr, screenui.DuelScr, screenui.DuelWinScr, screenui.RandomEncounterScr} {
		if err := g.Update(); err != nil || g.current != want {
			t.Fatalf("transition to %v: current=%v, error=%v", want, g.current, err)
		}
	}
	if g.screens[screenui.RandomEncounterScr] != champion || duel.closed != 1 {
		t.Fatal("champion did not replace entry or duel was not closed")
	}
	if err := g.Update(); err != ebiten.Termination || pointers != 6 || champion.closed != 1 {
		t.Fatal("returning to the world must close the launcher")
	}
}

func TestArenaGameRetainsExistingAnteAfterEditing(t *testing.T) {
	ante := &transitionScreen{next: screenui.DuelAnteScr}
	editor := &transitionScreen{next: screenui.DuelAnteScr}
	g := &arenaGame{screens: map[screenui.ScreenName]screenui.Screen{screenui.EditDeckScr: editor, screenui.DuelAnteScr: ante},
		current: screenui.EditDeckScr, updatePointer: func() {}}
	if err := g.Update(); err != nil || g.screens[g.current] != ante {
		t.Fatal("editor did not return to the existing opponent")
	}
}

func TestArenaGamePropagatesScreenError(t *testing.T) {
	want := errors.New("arena error")
	screen := &transitionScreen{err: want}
	g := &arenaGame{screens: map[screenui.ScreenName]screenui.Screen{screenui.RandomEncounterScr: screen},
		current: screenui.RandomEncounterScr, updatePointer: func() {}}
	if err := g.Update(); !errors.Is(err, want) {
		t.Fatalf("error=%v, want %v", err, want)
	}
}
