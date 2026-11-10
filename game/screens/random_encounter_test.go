package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestRandomEncounterRefreshesSelectedArt(t *testing.T) {
	card := domain.FindCardByName("Forest")
	for _, art := range card.Artworks {
		domain.CacheCardImage(art.ID, image.NewRGBA(image.Rect(0, 0, 245, 342)))
	}
	t.Cleanup(domain.ClearCardImageCache)
	player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}}
	s := NewRandomEncounterScreen(player, "Forest", world.TerrainForest)
	owned := player.CardCollection.AvailableCards(card)[0]
	old := s.cardImage()
	domain.CacheCardImage(owned.ArtworkID, image.NewRGBA(image.Rect(0, 0, 245, 342)))
	want, _ := owned.CardImage(domain.CardViewFullMini)
	if got := s.cardImage(); got == old || got != want {
		t.Fatal("encounter retained old art or lost the selected artwork")
	}
}
