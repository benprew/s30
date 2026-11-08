package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestEditDeckKeepsArtworkAcrossSessions(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	forest := domain.FindCardByName("Forest")
	for _, artwork := range forest.Artworks {
		domain.CacheCardImage(artwork.ID, image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	}
	owned := domain.NewOwnedCard(forest)
	cc := domain.NewCardCollection()
	cc.AddOwnedCard(owned)
	player := &domain.Player{Character: domain.Character{CardCollection: cc}}
	first, err := NewEditDeckScreen(player, nil, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewEditDeckScreen(player, nil, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}
	a, b := first.collectionGroups["Forest"].owned, second.collectionGroups["Forest"].owned
	if a.ID != owned.ID || b.ArtworkID != a.ArtworkID {
		t.Fatal("editor changed the owned artwork")
	}
	tile, _ := a.CardImage(domain.CardViewFull)
	magnifier, _ := b.CardImage(domain.CardViewFull)
	if tile == nil || tile != magnifier {
		t.Fatal("magnifier and stack use different artwork")
	}
}
