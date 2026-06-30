package screens

import "github.com/benprew/s30/game/domain"

// collectionFilter holds the active filter toggles for the deck-editor
// collection list. Colors and types accumulate independently: a card must
// match at least one active color (if any) AND one active type (if any).
type collectionFilter struct {
	colors map[string]bool
	types  map[domain.CardType]bool
}

func newCollectionFilter() collectionFilter {
	return collectionFilter{
		colors: make(map[string]bool),
		types:  make(map[domain.CardType]bool),
	}
}

func (f *collectionFilter) active() bool {
	return len(f.colors) > 0 || len(f.types) > 0
}

func (f *collectionFilter) toggleColor(c string) {
	if f.colors[c] {
		delete(f.colors, c)
	} else {
		f.colors[c] = true
	}
}

func (f *collectionFilter) toggleType(t domain.CardType) {
	if f.types[t] {
		delete(f.types, t)
	} else {
		f.types[t] = true
	}
}

// matches reports whether a card passes the active filters. With no toggles
// active every card matches.
func (f *collectionFilter) matches(c *domain.Card) bool {
	if len(f.colors) > 0 && !f.matchesColor(c) {
		return false
	}

	if len(f.types) > 0 && !f.types[c.CardType] {
		return false
	}

	return true
}

// matchesColor reports whether a card satisfies the active color filter. Cards
// match by their color, and lands additionally match by the mana they can
// produce, so a red filter surfaces Mountains and any-color lands like City of
// Brass while remaining colorless for other purposes.
func (f *collectionFilter) matchesColor(c *domain.Card) bool {
	for _, col := range c.Colors {
		if f.colors[col] {
			return true
		}
	}

	if c.IsLand() {
		for _, col := range c.ManaProduction {
			if f.colors[col] {
				return true
			}
		}
	}

	return false
}
