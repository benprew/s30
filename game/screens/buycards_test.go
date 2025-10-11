package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestBuyCard_PurchaseLogic(t *testing.T) {
	city := &domain.City{}
	city.CardsForSale = []int{0}
	player := &domain.Player{Gold: 10, CardCollection: make(map[int]int)}
	cardIdx := city.CardsForSale[0]
	card := domain.CARDS[cardIdx]
	card.Price = 5

	screen := &BuyCardsScreen{
		City:        city,
		Player:      player,
		PreviewIdx:  0,
		PreviewType: "card",
	}
	// Ensure CardImgs slice exists to avoid index out of range in buyCard
	screen.CardImgs = make([]*ebiten.Image, len(city.CardsForSale))

	screen.buyCard()

	if player.Gold != 5 {
		t.Errorf("Expected player gold to be 5, got %d", player.Gold)
	}
	if player.CardCollection[cardIdx] != 1 {
		t.Errorf("Expected player to have 1 of card %d, got %d", cardIdx, player.CardCollection[cardIdx])
	}
	if city.CardsForSale[0] != -1 {
		t.Errorf("Expected card to be marked as sold (-1), got %d", city.CardsForSale[0])
	}
	if screen.PreviewIdx != -1 {
		t.Errorf("Expected PreviewIdx to be reset to -1, got %d", screen.PreviewIdx)
	}
	if screen.ErrorMsg != "" {
		t.Errorf("Expected no error message, got %q", screen.ErrorMsg)
	}
}

func TestBuyCard_NotEnoughGold(t *testing.T) {
	city := &domain.City{}
	city.CardsForSale = []int{0}
	player := &domain.Player{Gold: 2, CardCollection: make(map[int]int)}
	cardIdx := city.CardsForSale[0]
	card := domain.CARDS[cardIdx]
	card.Price = 5

	screen := &BuyCardsScreen{
		City:        city,
		Player:      player,
		PreviewIdx:  0,
		PreviewType: "card",
	}
	// Ensure CardImgs slice exists to avoid index out of range in buyCard
	screen.CardImgs = make([]*ebiten.Image, len(city.CardsForSale))

	screen.buyCard()

	if player.Gold != 2 {
		t.Errorf("Expected player gold to remain 2, got %d", player.Gold)
	}
	if player.CardCollection[cardIdx] != 0 {
		t.Errorf("Expected player to have 0 of card %d, got %d", cardIdx, player.CardCollection[cardIdx])
	}
	if city.CardsForSale[0] != 0 {
		t.Errorf("Expected card to remain for sale, got %d", city.CardsForSale[0])
	}
	if screen.PreviewIdx != 0 {
		t.Errorf("Expected PreviewIdx to remain 0, got %d", screen.PreviewIdx)
	}
	if screen.ErrorMsg == "" {
		t.Errorf("Expected error message for not enough money, got empty string")
	}
}
