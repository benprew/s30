package screens

import (
	"image"
	"image/color"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
)

func TestPurchaseButtonsProvideTouchConfirmation(t *testing.T) {
	buttons := mkPurchaseButtons()
	if len(buttons) != 2 {
		t.Fatalf("Expected two purchase buttons, got %d", len(buttons))
	}
	if buttons[0].ID != "buy_yes" || buttons[0].ButtonText.Text != "Yes" {
		t.Errorf("Expected first button to confirm purchase, got ID %q and text %q", buttons[0].ID, buttons[0].ButtonText.Text)
	}
	if buttons[1].ID != "buy_no" || buttons[1].ButtonText.Text != "No" {
		t.Errorf("Expected second button to cancel purchase, got ID %q and text %q", buttons[1].ID, buttons[1].ButtonText.Text)
	}
}

func TestBuyCard_PurchaseLogic(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	card.Price = 5

	city := &domain.City{}
	city.CardsForSale = []*domain.Card{card}
	player := &domain.Player{
		Gold: 10,
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	screen := &BuyCardsScreen{
		City:       city,
		Player:     player,
		PreviewIdx: 0,
	}

	screen.buyCard()

	if player.Gold != 5 {
		t.Errorf("Expected player gold to be 5, got %d", player.Gold)
	}
	if player.CardCollection.GetTotalCount(card) != 1 {
		t.Errorf("Expected player to have 1 of card %s, got %d", card.Name(), player.CardCollection.GetTotalCount(card))
	}
	if len(city.CardsForSale) != 0 {
		t.Errorf("Expected card to be removed from sale, got %d cards remaining", len(city.CardsForSale))
	}
	if screen.PreviewIdx != -1 {
		t.Errorf("Expected PreviewIdx to be reset to -1, got %d", screen.PreviewIdx)
	}
	if screen.ErrorMsg != "" {
		t.Errorf("Expected no error message, got %q", screen.ErrorMsg)
	}
}

func TestBuyCard_NotEnoughGold(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	card.Price = 5

	city := &domain.City{}
	city.CardsForSale = []*domain.Card{card}
	player := &domain.Player{
		Gold: 2,
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	screen := &BuyCardsScreen{
		City:       city,
		Player:     player,
		PreviewIdx: 0,
	}

	screen.buyCard()

	if player.Gold != 2 {
		t.Errorf("Expected player gold to remain 2, got %d", player.Gold)
	}
	if player.CardCollection.GetTotalCount(card) != 0 {
		t.Errorf("Expected player to have 0 of card %s, got %d", card.Name(), player.CardCollection.GetTotalCount(card))
	}
	if len(city.CardsForSale) != 1 || city.CardsForSale[0] != card {
		t.Errorf("Expected card to remain for sale")
	}
	if screen.PreviewIdx != 0 {
		t.Errorf("Expected PreviewIdx to remain 0, got %d", screen.PreviewIdx)
	}
	if screen.ErrorMsg == "" {
		t.Errorf("Expected error message for not enough money, got empty string")
	}
}

func TestBuyCardsScreen_ReplacesPlaceholderWhenImageLoads(t *testing.T) {
	domain.ClearCardImageCache()
	t.Cleanup(domain.ClearCardImageCache)

	card := domain.FindCardByName("Mountain")
	city := &domain.City{
		CardsForSale: []*domain.Card{card},
	}
	player := &domain.Player{
		Gold: 10,
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	screen := NewBuyCardsScreen(city, player, 1024, 768)

	var cardBtn *elements.Button
	for _, b := range screen.Buttons {
		if b.ID == "card_0" {
			cardBtn = b
			break
		}
	}
	if cardBtn == nil {
		t.Fatal("Expected card_0 button to exist")
	}
	initialNormalImg := cardBtn.Normal

	// Now cache the real card art
	img := image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342))
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			img.Set(x, y, color.RGBA{R: 0xff, A: 0xff})
		}
	}
	domain.CacheCardImage(card.CardID(), img)

	if !card.ImageLoaded() {
		t.Fatal("Expected card image to be loaded")
	}

	// Update the screen, which should swap the button image
	_, _, err := screen.Update(1024, 768, 1.0)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if cardBtn.Normal == initialNormalImg {
		t.Errorf("Expected card_0 button image to be replaced after card art loaded")
	}
}

func TestResetCards_Success(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	city := &domain.City{
		CardsForSale: []*domain.Card{card},
	}
	player := &domain.Player{
		Gold: 50,
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	screen := &BuyCardsScreen{
		City:       city,
		Player:     player,
		PreviewIdx: 0,
		W:          1024,
		H:          768,
	}

	screen.resetCards()

	if player.Gold != 25 {
		t.Errorf("Expected player gold to be 25, got %d", player.Gold)
	}
	if len(city.CardsForSale) != 5 {
		t.Errorf("Expected 5 new cards for sale, got %d", len(city.CardsForSale))
	}
	if screen.PreviewIdx != -1 {
		t.Errorf("Expected PreviewIdx to be -1, got %d", screen.PreviewIdx)
	}
	if screen.ErrorMsg != "" {
		t.Errorf("Expected no error message, got %q", screen.ErrorMsg)
	}
}

func TestResetCards_NotEnoughGold(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	city := &domain.City{
		CardsForSale: []*domain.Card{card},
	}
	player := &domain.Player{
		Gold: 10,
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	screen := &BuyCardsScreen{
		City:       city,
		Player:     player,
		PreviewIdx: 0,
		W:          1024,
		H:          768,
	}

	screen.resetCards()

	if player.Gold != 10 {
		t.Errorf("Expected player gold to remain 10, got %d", player.Gold)
	}
	if len(city.CardsForSale) != 1 || city.CardsForSale[0] != card {
		t.Errorf("Expected cards for sale to remain unchanged")
	}
	if screen.ErrorMsg != "Not enough money!" {
		t.Errorf("Expected 'Not enough money!' error message, got %q", screen.ErrorMsg)
	}
}

func TestBuyCardsScreen_NewCardsButtonExists(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	city := &domain.City{
		CardsForSale: []*domain.Card{card},
	}
	player := &domain.Player{
		Gold: 50,
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	screen := NewBuyCardsScreen(city, player, 1024, 768)

	var newCardsBtn *elements.Button
	var doneBtn *elements.Button
	for _, b := range screen.Buttons {
		if b.ID == "new_cards" {
			newCardsBtn = b
		}
		if b.ID == "done" {
			doneBtn = b
		}
	}

	if newCardsBtn == nil {
		t.Fatal("Expected new_cards button to exist")
	}
	if newCardsBtn.ButtonText.Text != "New Cards" {
		t.Errorf("Expected button text 'New Cards', got %q", newCardsBtn.ButtonText.Text)
	}
	if doneBtn == nil {
		t.Fatal("Expected done button to exist")
	}
	if doneBtn.ButtonText.Text != "Done" {
		t.Errorf("Expected button text 'Done', got %q", doneBtn.ButtonText.Text)
	}
}

