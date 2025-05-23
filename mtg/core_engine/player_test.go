package core_engine

import (
	"testing"
)

func TestRemoveFrom(t *testing.T) {
	player := createTestPlayer(1)[0]

	for range 7 {
		player.DrawCard()
	}

	card := player.Hand[0]
	player.RemoveFrom(card, player.Hand, "Hand")

	if len(player.Hand) != 6 {
		t.Errorf("Card %v not removed from hand: %v", card, player.Hand)
	}
}

func TestAddTo(t *testing.T) {
	player := createTestPlayer(1)[0]

	card := player.Library[0]
	player.AddTo(card, "Hand")

	if len(player.Hand) != 1 {
		t.Errorf("Card %v not added to hand: %v", card, player.Hand)
	}
}
