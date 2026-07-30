package brawl

import (
	"testing"

	mage "github.com/benprew/mage-go/pkg/mage"
)

func TestNewBrawlUsesAnteGame(t *testing.T) {
	b := NewBrawl(mage.NewBasePlayer("A"), mage.NewBasePlayer("B"))

	if !b.game.AnteEnabled() {
		t.Fatal("brawl game does not have ante enabled")
	}
}
