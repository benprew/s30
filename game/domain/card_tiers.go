package domain

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/BurntSushi/toml"
	"github.com/benprew/s30/assets"
)

// CardTier ranks a card's competitive strength in Old School 93/94,
// from mandatory staples (0) down to meme-tier unplayables (9).
type CardTier int

const (
	TierMandatory CardTier = iota
	TierAlmostMandatory
	TierStaple
	TierPlayedInMostDecks
	TierPlayedQuiteOften
	TierPlayedFromTimeToTime
	TierPlayedInSpecificArchetypes
	TierRarelyPlayed
	TierAlmostNeverPlayed
	TierMeme
)

// cardTiersRaw mirrors the TOML layout in assets/configs/card_tiers.toml.
type cardTiersRaw struct {
	MandatoryCards               []string `toml:"mandatory_cards"`
	AlmostMandatory              []string `toml:"almost_mandatory"`
	Staples                      []string `toml:"staples"`
	PlayedInMostDecks            []string `toml:"played_in_most_decks"`
	PlayedQuiteOften             []string `toml:"played_quite_often"`
	PlayedFromTimeToTime         []string `toml:"played_from_time_to_time"`
	PlayedInSpecificArchetypes   []string `toml:"played_in_specific_archetypes"`
	RarelyPlayed                 []string `toml:"rarely_played"`
	AlmostNeverPlayed            []string `toml:"almost_never_played"`
	MemeCard                     []string `toml:"meme_card"`
}

// CardsByTier holds the cards for each power tier, resolved to *Card pointers.
var CardsByTier = loadCardTiers()

func loadCardTiers() map[CardTier][]*Card {
	var raw cardTiersRaw
	if _, err := toml.Decode(string(assets.CardTiers_toml), &raw); err != nil {
		panic(fmt.Errorf("error decoding card_tiers.toml: %w", err))
	}

	tiers := map[CardTier][]string{
		TierMandatory:                  raw.MandatoryCards,
		TierAlmostMandatory:            raw.AlmostMandatory,
		TierStaple:                     raw.Staples,
		TierPlayedInMostDecks:          raw.PlayedInMostDecks,
		TierPlayedQuiteOften:           raw.PlayedQuiteOften,
		TierPlayedFromTimeToTime:       raw.PlayedFromTimeToTime,
		TierPlayedInSpecificArchetypes: raw.PlayedInSpecificArchetypes,
		TierRarelyPlayed:               raw.RarelyPlayed,
		TierAlmostNeverPlayed:          raw.AlmostNeverPlayed,
		TierMeme:                       raw.MemeCard,
	}

	out := make(map[CardTier][]*Card, len(tiers))
	tieredNames := make(map[string]bool)
	for tier, names := range tiers {
		for _, name := range names {
			// card_tiers.toml holds the full Old School 93/94 list, including
			// cards from sets the game hasn't imported yet. Silently skip
			// unresolved names so new sets can be added without touching the
			// tier file.
			card := FindCardByName(name)
			if card == nil {
				continue
			}
			out[tier] = append(out[tier], card)
			tieredNames[card.CardName] = true
		}
	}

	// Surface cards that the game loads but the tier file doesn't rank.
	// These cards won't show up in tier-filtered pools (e.g. starting decks),
	// so someone needs to slot them into the appropriate tier. Dedupe by
	// card name since CARDS holds multiple printings per card.
	seen := make(map[string]bool)
	for _, card := range CARDS {
		if card.CardType == CardTypeLand {
			continue
		}
		if seen[card.CardName] || tieredNames[card.CardName] {
			continue
		}
		seen[card.CardName] = true
		log.Printf("card_tiers: %q is in the card database but not in any tier", card.CardName)
	}
	return out
}

// CardsInTiers returns every card in the listed tiers.
func CardsInTiers(tiers ...CardTier) []*Card {
	var cards []*Card
	for _, t := range tiers {
		cards = append(cards, CardsByTier[t]...)
	}
	return cards
}

// RandomPowerfulCardsForColor picks up to count unique high-tier cards whose
// color identity matches the requested color or are colorless (artifacts).
// Vintage-restricted cards are eligible. Lands are excluded so the reward is
// always a playable spell. Returns fewer than count if the eligible pool is
// smaller.
func RandomPowerfulCardsForColor(color ColorMask, count int) []*Card {
	pool := CardsInTiers(TierMandatory, TierAlmostMandatory, TierStaple)
	seen := make(map[string]bool)
	eligible := make([]*Card, 0, len(pool))
	for _, c := range pool {
		if c.CardType == CardTypeLand {
			continue
		}
		if seen[c.CardName] {
			continue
		}
		if !cardMatchesColorOrColorless(c, color) {
			continue
		}
		seen[c.CardName] = true
		eligible = append(eligible, c)
	}
	rand.Shuffle(len(eligible), func(i, j int) {
		eligible[i], eligible[j] = eligible[j], eligible[i]
	})
	if len(eligible) > count {
		eligible = eligible[:count]
	}
	return eligible
}

func cardMatchesColorOrColorless(c *Card, color ColorMask) bool {
	if len(c.ColorIdentity) == 0 {
		return true
	}
	for _, s := range c.ColorIdentity {
		if m, ok := colorStringToMask[s]; ok && color&m != 0 {
			return true
		}
	}
	return false
}
