package domain

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
)

type CollectionItem struct {
	Card       *Card
	Count      int
	DeckCounts []int
	Instances  []*OwnedCard
}

type CardCollection map[*Card]*CollectionItem

func NewCardCollection() CardCollection { return make(CardCollection) }

// Clone copies the collection without sharing owned copies or deck counts.
func (cc CardCollection) Clone() CardCollection {
	cloned := NewCardCollection()
	for _, item := range cc {
		for _, owned := range item.Instances {
			copy := *owned
			cloned.AddOwnedCard(&copy)
		}
		if copy := cloned[item.Card]; copy != nil {
			copy.ensureDeckSlots(len(item.DeckCounts))
		}
	}
	return cloned
}

func (item *CollectionItem) ensureDeckSlots(count int) {
	if len(item.DeckCounts) < count {
		item.DeckCounts = append(item.DeckCounts, make([]int, count-len(item.DeckCounts))...)
	}
}

func (item *CollectionItem) updateCounts() {
	item.Count = len(item.Instances)
	clear(item.DeckCounts)
	for _, owned := range item.Instances {
		if owned.DeckIndex < 0 {
			continue
		}
		for len(item.DeckCounts) <= owned.DeckIndex {
			item.DeckCounts = append(item.DeckCounts, 0)
		}
		item.DeckCounts[owned.DeckIndex]++
	}
}

// AddOwnedCard transfers a specific copy into the collection.
func (cc CardCollection) AddOwnedCard(owned *OwnedCard) {
	if owned == nil || owned.Card == nil {
		return
	}
	if _, _, existing := cc.findOwned(owned.ID); existing != nil {
		return
	}
	card := owned.Card
	item := cc[card]
	if item == nil {
		item = &CollectionItem{Card: card}
		cc[card] = item
	}
	item.Instances = append(item.Instances, owned)
	item.updateCounts()
}

func (cc CardCollection) findOwned(id string) (*CollectionItem, int, *OwnedCard) {
	for _, item := range cc {
		for i, owned := range item.Instances {
			if owned.ID == id {
				return item, i, owned
			}
		}
	}
	return nil, 0, nil
}

// MoveOwnedCard changes the deck of one copy. Use -1 to return it to the collection.
func (cc CardCollection) MoveOwnedCard(id string, deckIndex int) error {
	item, _, owned := cc.findOwned(id)
	if owned == nil {
		return fmt.Errorf("owned card %s not found", id)
	}
	if deckIndex < -1 {
		return fmt.Errorf("invalid deck index %d", deckIndex)
	}
	owned.DeckIndex = deckIndex
	item.updateCounts()
	return nil
}

// RemoveOwnedCard removes the specified copy and its deck membership.
func (cc CardCollection) RemoveOwnedCard(id string) error {
	item, i, owned := cc.findOwned(id)
	if owned == nil {
		return fmt.Errorf("owned card %s not found", id)
	}
	item.Instances = slices.Delete(item.Instances, i, i+1)
	item.updateCounts()
	if item.Count == 0 {
		delete(cc, item.Card)
	}
	return nil
}

// CardsInDeck returns specific copies in a stable order.
func (cc CardCollection) CardsInDeck(deckIndex int) []*OwnedCard {
	var cards []*OwnedCard
	for _, item := range cc {
		for _, owned := range item.Instances {
			if owned.DeckIndex == deckIndex {
				cards = append(cards, owned)
			}
		}
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].CardID() != cards[j].CardID() {
			return cards[i].CardID() < cards[j].CardID()
		}
		return cards[i].ID < cards[j].ID
	})
	return cards
}

// AvailableCards returns copies that are not assigned to a deck.
func (cc CardCollection) AvailableCards(card *Card) []*OwnedCard {
	var cards []*OwnedCard
	if item := cc[card]; item != nil {
		for _, owned := range item.Instances {
			if owned.DeckIndex == -1 {
				cards = append(cards, owned)
			}
		}
	}
	return cards
}

func (cc CardCollection) GetDeck(deckIndex int) Deck {
	deck := make(Deck)
	for card, item := range cc {
		if deckIndex >= 0 && deckIndex < len(item.DeckCounts) && item.DeckCounts[deckIndex] > 0 {
			deck[card] = item.DeckCounts[deckIndex]
		}
	}
	return deck
}

func (cc CardCollection) AddCardToDeck(card *Card, deckIndex int, count int) {
	if deckIndex < 0 {
		return
	}
	for range count {
		owned := NewOwnedCard(card)
		owned.DeckIndex = deckIndex
		cc.AddOwnedCard(owned)
	}
}

func (cc CardCollection) MoveCardToDeck(card *Card, deckIndex int, count int) error {
	cards := cc.AvailableCards(card)
	if count < 0 || deckIndex < 0 || len(cards) < count {
		return fmt.Errorf("insufficient available cards for %s", card.Name())
	}
	for _, owned := range cards[:count] {
		owned.DeckIndex = deckIndex
	}
	if item := cc[card]; item != nil {
		item.updateCounts()
	}
	return nil
}

func (cc CardCollection) MoveCardFromDeck(card *Card, deckIndex int, count int) error {
	var cards []*OwnedCard
	if item := cc[card]; item != nil {
		for _, owned := range item.Instances {
			if owned.DeckIndex == deckIndex {
				cards = append(cards, owned)
			}
		}
	}
	if count < 0 || deckIndex < 0 || len(cards) < count {
		return fmt.Errorf("insufficient cards in deck %d for %s", deckIndex, card.Name())
	}
	for _, owned := range cards[:count] {
		owned.DeckIndex = -1
	}
	if item := cc[card]; item != nil {
		item.updateCounts()
	}
	return nil
}

func (cc CardCollection) RemoveCardFromDeck(card *Card, deckIndex int, count int) error {
	var cards []*OwnedCard
	if item := cc[card]; item != nil {
		for _, owned := range item.Instances {
			if owned.DeckIndex == deckIndex {
				cards = append(cards, owned)
			}
		}
	}
	if count < 0 || deckIndex < 0 || len(cards) < count {
		return fmt.Errorf("insufficient cards in deck %d for %s", deckIndex, card.Name())
	}
	for _, owned := range cards[:count] {
		_ = cc.RemoveOwnedCard(owned.ID)
	}
	return nil
}

func (cc CardCollection) GetTotalCount(card *Card) int {
	if item := cc[card]; item != nil {
		return item.Count
	}
	return 0
}

func (cc CardCollection) GetDeckCount(card *Card, deckIndex int) int {
	if item := cc[card]; item != nil && deckIndex >= 0 && deckIndex < len(item.DeckCounts) {
		return item.DeckCounts[deckIndex]
	}
	return 0
}

func (cc CardCollection) DecrementCardCount(card *Card) error { return cc.RemoveCard(card, 1) }

func (cc CardCollection) AddCard(card *Card, count int) {
	for range count {
		cc.AddOwnedCard(NewOwnedCard(card))
	}
}

func (cc CardCollection) RemoveCard(card *Card, count int) error {
	item := cc[card]
	if count < 0 || item == nil || item.Count < count {
		return fmt.Errorf("insufficient cards in collection for %s", card.Name())
	}
	cards := slices.Clone(item.Instances)
	slices.SortStableFunc(cards, func(a, b *OwnedCard) int {
		if a.DeckIndex == -1 && b.DeckIndex != -1 {
			return -1
		}
		if b.DeckIndex == -1 && a.DeckIndex != -1 {
			return 1
		}
		return 0
	})
	for _, owned := range cards[:count] {
		_ = cc.RemoveOwnedCard(owned.ID)
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

type collectionItemJSON struct {
	CardID     string       `json:"card_id"`
	CardName   string       `json:"card_name"`
	Count      int          `json:"count"`
	DeckCounts []int        `json:"deck_counts"`
	Instances  []*OwnedCard `json:"instances,omitempty"`
}

func (cc CardCollection) MarshalJSON() ([]byte, error) {
	items := make([]collectionItemJSON, 0, len(cc))
	for card, item := range cc {
		items = append(items, collectionItemJSON{CardID: card.CardID(), CardName: card.CardName, Count: item.Count, DeckCounts: item.DeckCounts, Instances: item.Instances})
	}
	slices.SortFunc(items, func(a, b collectionItemJSON) int {
		if a.CardID < b.CardID {
			return -1
		}
		if a.CardID > b.CardID {
			return 1
		}
		return 0
	})
	return json.Marshal(items)
}

func (cc *CardCollection) UnmarshalJSON(data []byte) error {
	var items []collectionItemJSON
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	restored := NewCardCollection()
	for _, item := range items {
		card := FindCard(item.CardID, item.CardName)
		if card == nil {
			continue
		}
		if item.Instances != nil {
			for _, owned := range item.Instances {
				if owned == nil {
					continue
				}
				owned.Card = card
				if owned.ID == "" {
					owned.ID = newDefaultOwnedCard(card).ID
				}
				if owned.ArtworkID == "" {
					owned.ArtworkID = card.DefaultArtwork().ID
				}
				restored.AddOwnedCard(owned)
			}
			if loaded := restored[card]; loaded != nil {
				loaded.ensureDeckSlots(len(item.DeckCounts))
			}
			continue
		}
		remaining := max(item.Count, 0)
		for deckIndex, count := range item.DeckCounts {
			for range min(max(count, 0), remaining) {
				owned := newDefaultOwnedCard(card)
				if item.CardID != "" && item.CardID != card.Name() {
					owned.ArtworkID = item.CardID
				}
				owned.DeckIndex = deckIndex
				restored.AddOwnedCard(owned)
				remaining--
			}
		}
		for range remaining {
			owned := newDefaultOwnedCard(card)
			if item.CardID != "" && item.CardID != card.Name() {
				owned.ArtworkID = item.CardID
			}
			restored.AddOwnedCard(owned)
		}
		if loaded := restored[card]; loaded != nil {
			loaded.ensureDeckSlots(len(item.DeckCounts))
		}
	}
	*cc = restored
	return nil
}
