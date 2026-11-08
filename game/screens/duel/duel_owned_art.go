package duel

import (
	"sort"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/logging"
)

func (s *DuelScreen) addOwnedDeckToLibrary(player mage.Player, deck domain.Deck, collection domain.CardCollection, deckIndex int, anteCard *domain.Card) ([]*mage.Card, *domain.OwnedCard) {
	if s.cardInstances == nil {
		s.cardInstances = make(map[string]*domain.OwnedCard)
	}
	copies := make(map[*domain.Card][]*domain.OwnedCard)
	for _, owned := range collection.CardsInDeck(deckIndex) {
		copies[owned.Card] = append(copies[owned.Card], owned)
	}
	cards := make([]*domain.Card, 0, len(deck))
	for card := range deck {
		cards = append(cards, card)
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].CardID() < cards[j].CardID() })
	var ante []*mage.Card
	var ownedAnte *domain.OwnedCard
	for _, card := range cards {
		for i := range deck[card] {
			var owned *domain.OwnedCard
			if i < len(copies[card]) {
				owned = copies[card][i]
			} else {
				owned = domain.NewOwnedCard(card)
			}
			c, err := mage.CreateCard(card.Name())
			if err != nil {
				logging.Printf(logging.Duel, "Failed to create card %s: %v\n", card.Name(), err)
				continue
			}
			s.cardInstances[c.ID().String()] = owned
			player.AddToLibrary(c)
			if anteCard != nil && len(ante) == 0 && card.CardID() == anteCard.CardID() {
				ante, ownedAnte = []*mage.Card{c}, owned
			}
		}
	}
	return ante, ownedAnte
}

func anteOwnedCard(collection domain.CardCollection, deckIndex int, card *domain.Card) *domain.OwnedCard {
	if card == nil {
		return nil
	}
	for _, owned := range collection.CardsInDeck(deckIndex) {
		if owned.CardID() == card.CardID() {
			return owned
		}
	}
	return domain.NewOwnedCard(card)
}

func (s *DuelScreen) loadCardPreviewInstance(name, key string) {
	s.loadCardPreviewByName(name)
	if s.preview != nil {
		s.preview.Key = key
	}
}

func (s *DuelScreen) loadActionPreview(action interactive.ActionOption) {
	id := action.CardID
	if id == uuid.Nil() {
		id = action.PermanentID
	}
	s.loadCardPreviewInstance(action.CardName, id.String())
}

func (s *DuelScreen) newLossScreen(cards []*domain.Card) *DuelLoseScreen {
	screen := NewDuelLoseScreen(cards)
	if s.anteOwned != nil {
		screen.ownedCards = []*domain.OwnedCard{s.anteOwned}
	}
	return screen
}
