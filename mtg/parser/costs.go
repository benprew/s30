package parser

import (
	"regexp"
	"strconv"
	"strings"
)

type Cost struct {
	Mana        string
	Tap         bool
	Sacrifice   bool
	LifePayment int
}

var costRegex = regexp.MustCompile(`\{([^}]+)\}`)

func ParseCost(costStr string) *Cost {
	if costStr == "" {
		return nil
	}

	cost := &Cost{}
	matches := costRegex.FindAllStringSubmatch(costStr, -1)

	var manaComponents []string

	for _, match := range matches {
		symbol := match[1]
		switch strings.ToUpper(symbol) {
		case "T":
			cost.Tap = true
		case "W", "U", "B", "R", "G", "C":
			manaComponents = append(manaComponents, "{"+symbol+"}")
		default:
			if _, err := strconv.Atoi(symbol); err == nil {
				manaComponents = append(manaComponents, "{"+symbol+"}")
			}
		}
	}

	cost.Mana = strings.Join(manaComponents, "")
	return cost
}

func ParseManaSymbols(text string) []string {
	var symbols []string
	matches := costRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		symbols = append(symbols, match[1])
	}
	return symbols
}

func ExtractActivatedAbilityCost(text string) (cost *Cost, effect string, ok bool) {
	costPart, effectPart, found := strings.Cut(text, ":")
	if !found {
		return nil, "", false
	}

	costPart = strings.TrimSpace(costPart)
	effectPart = strings.TrimSpace(effectPart)

	cost = ParseCost(costPart)
	if cost == nil {
		cost = &Cost{}
	}

	if strings.Contains(strings.ToLower(costPart), "sacrifice") {
		cost.Sacrifice = true
	}

	return cost, effectPart, true
}
