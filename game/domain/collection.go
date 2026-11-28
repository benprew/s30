package domain

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

type CollectionItem struct {
	Card       *OwnedCard
	Count      int
	DeckCounts []int
}

type CardCollection map[*OwnedCard]*CollectionItem

func NewCardCollection() CardCollection { return make(CardCollection) }

// Clone gives the new collection its own counts.
func (cc CardCollection) Clone() CardCollection {
	cloned := NewCardCollection()
	for printing, item := range cc {
		cloned[printing] = &CollectionItem{Card: printing, Count: item.Count, DeckCounts: slices.Clone(item.DeckCounts)}
	}
	return cloned
}

// Items returns the inventory in card and artwork ID order.
func (cc CardCollection) Items() []*CollectionItem {
	return slices.SortedFunc(maps.Values(cc), func(a, b *CollectionItem) int {
		if order := cmp.Compare(a.Card.Card().CardID(), b.Card.Card().CardID()); order != 0 {
			return order
		}
		return cmp.Compare(a.Card.ArtworkID(), b.Card.ArtworkID())
	})
}

// DeckCount returns the quantity of this printing in one deck.
func (item *CollectionItem) DeckCount(index int) int {
	if index < 0 || index >= len(item.DeckCounts) {
		return 0
	}
	return item.DeckCounts[index]
}

func (item *CollectionItem) ensureDeckSlots(count int) {
	if len(item.DeckCounts) < count {
		item.DeckCounts = append(item.DeckCounts, make([]int, count-len(item.DeckCounts))...)
	}
}

func (item *CollectionItem) addToDeck(index, count int) {
	item.ensureDeckSlots(index + 1)
	item.DeckCounts[index] += count
}

// AddPrinting increases ownership of the specified printing.
func (cc CardCollection) AddPrinting(printing *OwnedCard, count int) *CollectionItem {
	if printing.Card() == nil || count <= 0 {
		return nil
	}
	item := cc[printing]
	if item == nil {
		item = &CollectionItem{Card: printing}
		cc[printing] = item
	}
	item.Count += count
	return item
}

// MovePrintingToDeck assigns available cards of one printing to one deck.
func (cc CardCollection) MovePrintingToDeck(printing *OwnedCard, index, count int) error {
	item := cc[printing]
	if index < 0 || count < 0 || item == nil || count > item.Count-item.DeckCount(index) {
		return fmt.Errorf("insufficient available cards for deck %d", index)
	}
	if count > 0 {
		item.addToDeck(index, count)
	}
	return nil
}

// MovePrintingFromDeck returns cards of one printing to the collection.
func (cc CardCollection) MovePrintingFromDeck(printing *OwnedCard, index, count int) error {
	item := cc[printing]
	if index < 0 || count < 0 || item == nil || count > item.DeckCount(index) {
		return fmt.Errorf("insufficient cards in deck %d", index)
	}
	if count > 0 {
		item.DeckCounts[index] -= count
	}
	return nil
}

// RemovePrinting reduces ownership. Use -1 to leave deck counts at the owned limit.
func (cc CardCollection) RemovePrinting(printing *OwnedCard, count, deckIndex int) error {
	item := cc[printing]
	if deckIndex < -1 || count < 0 || item == nil || item.Count < count {
		return fmt.Errorf("insufficient cards in the collection")
	}
	if deckIndex >= 0 && item.DeckCount(deckIndex) < count {
		return fmt.Errorf("insufficient cards in deck %d", deckIndex)
	}
	cc.reduceOwned(item, count, deckIndex)
	return nil
}

func (cc CardCollection) reduceOwned(item *CollectionItem, count, deckIndex int) {
	if deckIndex >= 0 && count > 0 {
		item.DeckCounts[deckIndex] -= count
	}
	item.Count -= count
	for slot, deckCount := range item.DeckCounts {
		item.DeckCounts[slot] = min(deckCount, item.Count)
	}
	if item.Count == 0 {
		delete(cc, item.Card)
	}
}

// CardsInDeck lists the printing for each card in a deck.
func (cc CardCollection) CardsInDeck(index int) []*OwnedCard {
	var cards []*OwnedCard
	for _, item := range cc.Items() {
		for range item.DeckCount(index) {
			cards = append(cards, item.Card)
		}
	}
	return cards
}

func (cc CardCollection) GetDeck(index int) Deck {
	deck := make(Deck)
	for printing, item := range cc {
		if count := item.DeckCount(index); count > 0 {
			deck[printing.Card()] += count
		}
	}
	return deck
}

func (cc CardCollection) GetTotalCount(card *Card) int {
	total := 0
	for printing, item := range cc {
		if printing.Card() == card {
			total += item.Count
		}
	}
	return total
}

func (cc CardCollection) GetDeckCount(card *Card, index int) int {
	total := 0
	for printing, item := range cc {
		if printing.Card() == card {
			total += item.DeckCount(index)
		}
	}
	return total
}

// AddCard increases ownership. Use -1 to leave the cards outside the decks.
func (cc CardCollection) AddCard(card *Card, count, deckIndex int) {
	for range count {
		item := cc.AddPrinting(card.RandomPrinting(), 1)
		if deckIndex >= 0 {
			item.addToDeck(deckIndex, 1)
		}
	}
}

// RemoveCard reduces ownership. Use -1 to use spare stock first.
func (cc CardCollection) RemoveCard(card *Card, count, deckIndex int) error {
	available := cc.GetTotalCount(card)
	if deckIndex >= 0 {
		available = cc.GetDeckCount(card, deckIndex)
	}
	if deckIndex < -1 || count < 0 || available < count {
		return fmt.Errorf("insufficient cards for removal from deck %d for %s", deckIndex, card.Name())
	}
	if count == 0 {
		return nil
	}
	items := cc.Items()
	// Use the stock that each deck can spare before reducing deck counts.
	for _, spareOnly := range []bool{true, false} {
		for _, item := range items {
			if item.Card.Card() != card {
				continue
			}
			quantity := min(count, item.Count)
			if deckIndex >= 0 {
				quantity = min(count, item.DeckCount(deckIndex))
			} else if spareOnly && len(item.DeckCounts) > 0 {
				quantity = min(quantity, item.Count-slices.Max(item.DeckCounts))
			}
			if quantity > 0 {
				cc.reduceOwned(item, quantity, deckIndex)
				count -= quantity
			}
			if count == 0 {
				return nil
			}
		}
	}
	return nil
}

func (cc CardCollection) GetAllCards() []*Card {
	cards := make(map[*Card]struct{}, len(cc))
	for printing := range cc {
		cards[printing.Card()] = struct{}{}
	}
	return slices.Collect(maps.Keys(cards))
}

func (cc CardCollection) NumCards() int {
	total := 0
	for _, item := range cc {
		total += item.Count
	}
	return total
}

type collectionItemJSON struct {
	printingJSON
	Count      int   `json:"count"`
	DeckCounts []int `json:"deck_counts"`
}

func (cc CardCollection) MarshalJSON() ([]byte, error) {
	items := make([]collectionItemJSON, 0, len(cc))
	for _, item := range cc.Items() {
		items = append(items, collectionItemJSON{
			printingJSON: item.Card.reference(),
			Count:        item.Count,
			DeckCounts:   item.DeckCounts,
		})
	}
	return json.Marshal(items)
}

type legacyInstance struct {
	savedPrinting
	ID        string `json:"id"`
	DeckIndex int    `json:"deck_index"`
}

type savedCollectionItem struct {
	collectionItemJSON
	CardName    string            `json:"card_name"`
	Instances   []*legacyInstance `json:"instances"`
	DeckCopyIDs [][]string        `json:"deck_copy_ids"`
}

func (cc *CardCollection) UnmarshalJSON(data []byte) error {
	var items []savedCollectionItem
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	restored := NewCardCollection()
	seenIDs := make(map[string]bool)
	for _, saved := range items {
		card := FindCard(saved.CardID, saved.CardName)
		if card == nil {
			continue
		}
		if saved.Instances != nil {
			saved.restoreInstances(restored, card, seenIDs)
		} else {
			saved.restoreCounts(restored, card)
		}
	}
	*cc = restored
	return nil
}

func (saved savedCollectionItem) restoreCounts(cc CardCollection, card *Card) {
	artworkID := saved.ArtworkID
	if artworkID == "" && saved.CardID != card.CardID() {
		artworkID = saved.CardID
	}
	item := cc.AddPrinting(card.Printing(artworkID), saved.Count)
	if item != nil {
		item.ensureDeckSlots(len(saved.DeckCounts))
		for slot, count := range saved.DeckCounts {
			item.DeckCounts[slot] += min(max(count, 0), saved.Count)
		}
	}
}

func (saved savedCollectionItem) restoreInstances(cc CardCollection, card *Card, seenIDs map[string]bool) {
	byID := make(map[string]*CollectionItem)
	for _, instance := range saved.Instances {
		if instance == nil || seenIDs[instance.ID] {
			continue
		}
		item := cc.AddPrinting(instance.resolve(card), 1)
		item.ensureDeckSlots(max(len(saved.DeckCopyIDs), len(saved.DeckCounts)))
		if instance.ID != "" {
			seenIDs[instance.ID] = true
			byID[instance.ID] = item
		}
		if saved.DeckCopyIDs == nil && instance.DeckIndex >= 0 && !instance.bareReference() {
			item.addToDeck(instance.DeckIndex, 1)
		}
	}

	inDeck := make(map[string]bool)
	for slot, ids := range saved.DeckCopyIDs {
		clear(inDeck)
		for _, id := range ids {
			if item := byID[id]; item != nil && !inDeck[id] {
				item.addToDeck(slot, 1)
				inDeck[id] = true
			}
		}
	}
}
