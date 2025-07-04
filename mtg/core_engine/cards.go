package core_engine

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/benprew/s30/mtg/cards"
)

var CardDatabase = LoadCardDatabase()

func LoadCardDatabase() map[string]*Card {
	cardFile := "testset/cards.json"

	fmt.Printf("Loading cards from %s\n", cardFile)

	byteValue, err := cards.CardData.ReadFile(cardFile)
	if err != nil {
		log.Fatalf("Error reading embedded card file %s: %v", cardFile, err)
		return nil
	}

	var cards map[string]*Card
	err = json.Unmarshal(byteValue, &cards)
	if err != nil {
		log.Fatalf("Error unmarshalling card file %s: %v", cardFile, err)
		return nil
	}

	for _, c := range cards {
		fmt.Println("working on ", c.Name())
		c.UnMarshalActions()
		if len(c.Actions) > 0 {
			fmt.Printf("%v\n", c.Actions[0])
		}
	}
	return cards
}
