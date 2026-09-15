package domain

import (
	"encoding/json"
	"log"

	"github.com/benprew/s30/assets"
)

// TOKENS holds printings for the tokens the rules engine puts onto the
// battlefield, so the duel screen can draw their art. They are kept out of
// CARDS because a token is never owned, traded or sold. Tokens that were never
// printed leave BorderCropURL empty and are drawn as a blank card labeled with
// their name.
var TOKENS = loadTokenCards()

func loadTokenCards() []*Card {
	var tokenJSON []*CardJSON
	if err := json.Unmarshal(assets.TokenCards_json, &tokenJSON); err != nil {
		log.Printf("Error unmarshalling token card data: %v", err)
		return nil
	}

	tokens := make([]*Card, 0, len(tokenJSON))
	for _, cardJSON := range tokenJSON {
		tokens = append(tokens, cardJSON.ToCard())
	}
	return tokens
}

// FindTokenByName returns the token printing with the given name, or nil when
// no printing is known for it.
func FindTokenByName(name string) *Card {
	for _, token := range TOKENS {
		if token.CardName == name {
			return token
		}
	}
	return nil
}
