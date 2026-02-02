package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/parser"
)

type ParsedCard struct {
	CardName  string                   `json:"card_name"`
	Text      string                   `json:"text"`
	Abilities []*parser.ParsedAbility  `json:"abilities,omitempty"`
	Unparsed  []string                 `json:"unparsed,omitempty"`
}

type UnparsedEntry struct {
	CardName string `json:"card_name"`
	Text     string `json:"text"`
}

type Output struct {
	Parsed   []ParsedCard    `json:"parsed"`
	Unparsed []UnparsedEntry `json:"unparsed"`
	Stats    Stats           `json:"stats"`
}

type Stats struct {
	TotalCards      int `json:"total_cards"`
	CardsWithText   int `json:"cards_with_text"`
	FullyParsed     int `json:"fully_parsed"`
	PartiallyParsed int `json:"partially_parsed"`
	NoParse         int `json:"no_parse"`
}

func main() {
	showUnparsed := flag.Bool("unparsed", false, "only show unparsed text")
	cardName := flag.String("card", "", "parse a specific card by name")
	outFile := flag.String("o", "", "output file (default: stdout)")
	verbose := flag.Bool("verbose", false, "show full output including stats and unparsed")
	flag.Parse()

	cards := domain.CARDS

	if *cardName != "" {
		card := domain.FindCardByName(*cardName)
		if card == nil {
			fmt.Fprintf(os.Stderr, "Card not found: %s\n", *cardName)
			os.Exit(1)
		}
		cards = []*domain.Card{card}
	}

	p := parser.DefaultParser
	output := Output{
		Parsed:   make([]ParsedCard, 0),
		Unparsed: make([]UnparsedEntry, 0),
	}

	seen := make(map[string]bool)

	for _, card := range cards {
		if card.Text == "" {
			output.Stats.TotalCards++
			continue
		}

		if seen[card.CardName] {
			continue
		}
		seen[card.CardName] = true

		output.Stats.TotalCards++
		output.Stats.CardsWithText++

		result := p.Parse(card.CardName, card.Text)

		parsed := ParsedCard{
			CardName:  card.CardName,
			Text:      card.Text,
			Abilities: result.Abilities,
			Unparsed:  result.Unparsed,
		}

		if len(result.Unparsed) == 0 && len(result.Abilities) > 0 {
			output.Stats.FullyParsed++
		} else if len(result.Abilities) > 0 {
			output.Stats.PartiallyParsed++
		} else {
			output.Stats.NoParse++
		}

		if !*showUnparsed || len(result.Unparsed) > 0 {
			output.Parsed = append(output.Parsed, parsed)
		}

		for _, text := range result.Unparsed {
			output.Unparsed = append(output.Unparsed, UnparsedEntry{
				CardName: card.CardName,
				Text:     text,
			})
		}
	}

	sort.Slice(output.Unparsed, func(i, j int) bool {
		return output.Unparsed[i].Text < output.Unparsed[j].Text
	})

	var out *os.File
	if *outFile != "" {
		var err error
		out, err = os.Create(*outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	if *verbose {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding output: %v\n", err)
			os.Exit(1)
		}
	} else {
		for _, card := range output.Parsed {
			if len(card.Abilities) == 0 {
				continue
			}
			fmt.Fprintf(out, "%s:\n", card.CardName)
			for _, ability := range card.Abilities {
				fmt.Fprintf(out, "  [%s] %s\n", ability.Type, ability.RawText)
			}
		}
	}

	if *outFile != "" {
		fmt.Fprintf(os.Stderr, "Wrote output to %s\n", *outFile)
	}
}
