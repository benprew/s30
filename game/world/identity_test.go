package world

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestSaveName(t *testing.T) {
	l := &Level{}
	l.SetIdentity("foobar", domain.DifficultyEasy, domain.ColorBlack)

	want := "Apprentice-Black-foobar"
	if got := l.SaveName(); got != want {
		t.Errorf("SaveName() = %q, want %q", got, want)
	}
}

func TestNewGameIDUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := NewGameID()
		if id == "" {
			t.Fatal("NewGameID() returned empty string")
		}
		if seen[id] {
			t.Fatalf("NewGameID() returned duplicate id %q", id)
		}
		seen[id] = true
	}
}
