package duel

import (
	"testing"

	mage "github.com/benprew/mage-go/pkg/mage"
)

func newTestAnteGame(t testing.TB, playerA, playerB mage.Player) *mage.Game {
	t.Helper()
	g, err := mage.NewGameWithAnte(playerA, playerB, nil, nil)
	if err != nil {
		t.Fatalf("NewGameWithAnte: %v", err)
	}
	return g
}
