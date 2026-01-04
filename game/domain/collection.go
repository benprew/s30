package domain

import (
	"fmt"
)

type CollectionItem struct {
	Card       *Card
	Count      int   // total number of cards in the collection
	DeckCounts []int // count of this card in each deck (index = deck number)
}

type CardCollection map[*Card]*CollectionItem

func NewCardCollection() CardCollection {
	return make(CardCollection)
}

func (cc CardCollection) GetDeck(deckIndex int) Deck {
	deck := make(Deck)
	for card, item := range cc {
		if deckIndex < len(item.DeckCounts) && item.DeckCounts[deckIndex] > 0 {
			deck[card] = item.DeckCounts[deckIndex]
		}
	}
	return deck
}

func (cc CardCollection) AddCardToDeck(card *Card, deckIndex int, count int) {
	item := cc[card]
	if item == nil {
		item = &CollectionItem{
			Card:       card,
			Count:      0,
			DeckCounts: make([]int, deckIndex+1),
		}
		cc[card] = item
	}

	// Extend DeckCounts if necessary
	if deckIndex >= len(item.DeckCounts) {
		newDeckCounts := make([]int, deckIndex+1)
		copy(newDeckCounts, item.DeckCounts)
		item.DeckCounts = newDeckCounts
	}

	item.DeckCounts[deckIndex] += count
	item.Count += count
}

func (cc CardCollection) MoveCardToDeck(card *Card, deckIndex int, count int) error {
	item := cc[card]
	if item == nil {
		return fmt.Errorf("card %s not found in collection", card.Name())
	}

	totalInDecks := 0
	for _, deckCount := range item.DeckCounts {
		totalInDecks += deckCount
	}

	availableCount := item.Count - totalInDecks
	if availableCount < count {
		return fmt.Errorf("insufficient available cards for %s (have %d available, need %d)",
			card.Name(), availableCount, count)
	}

	if deckIndex >= len(item.DeckCounts) {
		newDeckCounts := make([]int, deckIndex+1)
		copy(newDeckCounts, item.DeckCounts)
		item.DeckCounts = newDeckCounts
	}

	item.DeckCounts[deckIndex] += count

	return nil
}

func (cc CardCollection) MoveCardFromDeck(card *Card, deckIndex int, count int) error {
	item := cc[card]
	if item == nil {
		return fmt.Errorf("card %s not found in collection", card.Name())
	}

	if deckIndex >= len(item.DeckCounts) {
		return fmt.Errorf("deck %d does not exist for card %s", deckIndex, card.Name())
	}

	if item.DeckCounts[deckIndex] < count {
		return fmt.Errorf("insufficient cards in deck %d for card %s (have %d, need %d)",
			deckIndex, card.Name(), item.DeckCounts[deckIndex], count)
	}

	item.DeckCounts[deckIndex] -= count

	return nil
}

func (cc CardCollection) RemoveCardFromDeck(card *Card, deckIndex int, count int) error {
	item := cc[card]
	if item == nil {
		return fmt.Errorf("card %s not found in collection", card.Name())
	}

	if deckIndex >= len(item.DeckCounts) {
		return fmt.Errorf("deck %d does not exist for card %s", deckIndex, card.Name())
	}

	if item.DeckCounts[deckIndex] < count {
		return fmt.Errorf("insufficient cards in deck %d for card %s (have %d, need %d)",
			deckIndex, card.Name(), item.DeckCounts[deckIndex], count)
	}

	item.DeckCounts[deckIndex] -= count
	item.Count -= count

	// Remove item if no cards left
	if item.Count <= 0 {
		delete(cc, card)
	}

	return nil
}

func (cc CardCollection) GetTotalCount(card *Card) int {
	item := cc[card]
	if item == nil {
		return 0
	}
	return item.Count
}

func (cc CardCollection) GetDeckCount(card *Card, deckIndex int) int {
	item := cc[card]
	if item == nil || deckIndex >= len(item.DeckCounts) {
		return 0
	}
	return item.DeckCounts[deckIndex]
}

func (cc CardCollection) DecrementCardCount(card *Card) error {
	item := cc[card]
	if item == nil {
		return fmt.Errorf("card %s not found in collection", card.Name())
	}

	if item.Count <= 0 {
		return fmt.Errorf("no cards left to decrement for %s", card.Name())
	}

	// Decrement total count
	item.Count--

	// Update all deck counts to not exceed total count
	for i := range item.DeckCounts {
		if item.DeckCounts[i] > item.Count {
			item.DeckCounts[i] = item.Count
		}
	}

	// Remove item if no cards left
	if item.Count <= 0 {
		delete(cc, card)
	}

	return nil
}

func (cc CardCollection) AddCard(card *Card, count int) {
	item := cc[card]
	if item == nil {
		item = &CollectionItem{
			Card:       card,
			Count:      0,
			DeckCounts: make([]int, 0),
		}
		cc[card] = item
	}
	item.Count += count
}

func (cc CardCollection) RemoveCard(card *Card, count int) error {
	item := cc[card]
	if item == nil {
		return fmt.Errorf("card %s not found in collection", card.Name())
	}

	if item.Count < count {
		return fmt.Errorf("insufficient cards in collection for %s (have %d, need %d)",
			card.Name(), item.Count, count)
	}

	item.Count -= count

	// Update deck counts to not exceed total count
	for i := range item.DeckCounts {
		if item.DeckCounts[i] > item.Count {
			item.DeckCounts[i] = item.Count
		}
	}

	// Remove item if no cards left
	if item.Count <= 0 {
		delete(cc, card)
	}

	return nil
}

func (cc CardCollection) GetAllCards() []*Card {
	cards := make([]*Card, 0, len(cc))
	for card := range cc {
		cards = append(cards, card)
	}
	return cards
}

func (cc CardCollection) NumCards() int {
	total := 0
	for _, item := range cc {
		total += item.Count
	}
	return total
}