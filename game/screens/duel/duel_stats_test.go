package duel

import (
	"image"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

func TestCreatureStatsTextSubtractsMarkedDamage(t *testing.T) {
	perm := interactive.PermanentState{Power: 3, Toughness: 4, Damage: 2}

	got := creatureStatsText(perm)
	if got != "3/2" {
		t.Errorf("want 3/2, got %q", got)
	}
}

func TestCreatureStatsTextDoesNotGoBelowZero(t *testing.T) {
	perm := interactive.PermanentState{Power: 1, Toughness: 1, Damage: 3}

	got := creatureStatsText(perm)
	if got != "1/0" {
		t.Errorf("want 1/0, got %q", got)
	}
}

func TestCreatureStatsTextPositionRightAlignsMeasuredText(t *testing.T) {
	pos := image.Point{X: 25, Y: 40}

	got := creatureStatsTextPosition(pos, 41.2)
	want := image.Point{
		X: pos.X + fieldCardW - 42 - creatureStatsRightPadding,
		Y: pos.Y + fieldCardH - creatureStatsBottomInset,
	}
	if got != want {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestCardPreviewStatsTextPositionRightAlignsMeasuredText(t *testing.T) {
	pos := image.Point{X: 7, Y: 11}
	size := image.Point{X: 225, Y: 315}

	got := cardPreviewStatsTextPosition(pos, size, 58.7)
	want := image.Point{
		X: pos.X + size.X - 59 - cardPreviewStatsRightPadding,
		Y: pos.Y + size.Y - cardPreviewStatsBottomInset,
	}
	if got != want {
		t.Errorf("want %v, got %v", want, got)
	}
}
