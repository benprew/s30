package duel

import "testing"

func TestCloseCancelsRunningGameLoop(t *testing.T) {
	canceled := false
	s := &DuelScreen{
		loopCancel: func() {
			canceled = true
		},
	}

	s.Close()

	if !canceled {
		t.Fatal("Close did not cancel the game loop")
	}
}
