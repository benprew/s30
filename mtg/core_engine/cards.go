package core_engine

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/benprew/s30/mtg/cards"
)

var CardDatabase = map[string]*Card{}

func LoadCardDatabase() {
	cardFile := "testset/cards.json"

	fmt.Printf("Loading cards from %s\n", cardFile)

	byteValue, err := cards.CardData.ReadFile(cardFile)
	if err != nil {
		log.Fatalf("Error reading embedded card file %s: %v", cardFile, err)
		return
	}

	var cards map[string]*Card
	err = json.Unmarshal(byteValue, &cards)
	if err != nil {
		log.Fatalf("Error unmarshalling card file %s: %v", cardFile, err)
		return
	}

	for name, card := range cards {
		CardDatabase[name] = card
	}
}

func init() {
	LoadCardDatabase()
}
