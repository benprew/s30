package duel

import (
	"testing"
	"time"

	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestLossAnimationLifeInterpolatesToFinalLife(t *testing.T) {
	start := time.Unix(100, 0)
	s := &DuelScreen{}
	s.selfLifeAnimation.start(7, -3, start)

	if got := s.displayedSelfLife(start); got != 7 {
		t.Errorf("start life = %d, want 7", got)
	}

	mid := start.Add(lossLifeAnimationDuration / 2)
	if got := s.displayedSelfLife(mid); got >= 7 || got <= -3 {
		t.Errorf("mid life = %d, want between 7 and -3", got)
	}

	done := start.Add(lossLifeAnimationDuration)
	if got := s.displayedSelfLife(done); got != -3 {
		t.Errorf("done life = %d, want -3", got)
	}
}

func TestLossAnimationHoldsBeforeTransition(t *testing.T) {
	start := time.Unix(100, 0)
	s := &DuelScreen{}
	s.selfLifeAnimation.start(2, 0, start)

	if s.lossAnimationComplete(start.Add(lossLifeAnimationDuration)) {
		t.Fatal("animation should hold final life before transition")
	}

	if !s.lossAnimationComplete(start.Add(lossLifeAnimationDuration + lossLifeHoldDuration)) {
		t.Fatal("animation should complete after animation plus hold")
	}
}

func TestLossAnimationDoesNotStartTwice(t *testing.T) {
	start := time.Unix(100, 0)
	s := &DuelScreen{}
	s.selfLifeAnimation.start(5, 0, start)
	s.selfLifeAnimation.start(20, -9, start.Add(time.Second))

	if got := s.displayedSelfLife(start); got != 5 {
		t.Errorf("life after second start = %d, want original start 5", got)
	}
}

func TestLossAnimationOpponentLifeInterpolatesToFinalLife(t *testing.T) {
	start := time.Unix(100, 0)
	s := &DuelScreen{}
	s.opponentLifeAnimation.start(4, -2, start)

	if got := s.displayedOpponentLife(start); got != 4 {
		t.Errorf("start life = %d, want 4", got)
	}

	done := start.Add(lossLifeAnimationDuration)
	if got := s.displayedOpponentLife(done); got != -2 {
		t.Errorf("done life = %d, want -2", got)
	}
}

func TestLossAnimationWaitsForOpponentBeforeWinTransition(t *testing.T) {
	start := time.Unix(100, 0)
	s := &DuelScreen{}
	s.opponentLifeAnimation.start(3, 0, start)

	if s.lossAnimationComplete(start.Add(lossLifeAnimationDuration)) {
		t.Fatal("opponent animation should hold final life before transition")
	}

	if !s.lossAnimationComplete(start.Add(lossLifeAnimationDuration + lossLifeHoldDuration)) {
		t.Fatal("opponent animation should complete after animation plus hold")
	}
}

func TestLossAnimationStartsForOpponentGameOver(t *testing.T) {
	start := time.Unix(100, 0)
	s := &DuelScreen{}
	prev := &interactive.GameMsg{
		State: &interactive.GameState{
			You:      interactive.PlayerState{Life: 8},
			Opponent: interactive.PlayerState{Life: 2},
		},
	}
	cur := &interactive.GameMsg{
		GameOver: true,
		Winner:   "You",
		State: &interactive.GameState{
			You:      interactive.PlayerState{Life: 8},
			Opponent: interactive.PlayerState{Life: -1},
		},
	}

	s.startLossAnimationFromMessage(prev, cur, start)

	if !s.opponentLifeAnimation.started {
		t.Fatal("opponent life animation should start")
	}
	if s.selfLifeAnimation.started {
		t.Fatal("self life animation should not start")
	}
	if got := s.displayedOpponentLife(start.Add(lossLifeAnimationDuration)); got != -1 {
		t.Errorf("opponent final life = %d, want -1", got)
	}
}
