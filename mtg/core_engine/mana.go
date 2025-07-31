package core_engine

import (
	"fmt"
	"slices"
	"strconv"
	"unicode"
)

// A player's mana pool is represented as a list of list of runes.
// These runes represent alternatives, so a board with a Forest and BW dual land
// would be represented as
// [['G'],['B', 'W']]

type ManaPool [][]rune

func (m *ManaPool) AddMana(manaType []rune) {
	*m = append(*m, manaType)
}

func (m *ManaPool) RemoveMana(manaType rune) bool {
	// If the requested manaType is a digit or 'C', treat it as a request for colorless mana.
	// Note: The Pay function currently iterates over the cost string runes.
	// If the cost is "1G", it calls RemoveMana('1') and RemoveMana('G').
	// This logic handles digit runes ('0'-'9') as requests for colorless.
	isColorlessRequest := unicode.IsDigit(manaType) || manaType == 'C'

	if isColorlessRequest {
		// Try to remove an explicit colorless source first ('C')
		for i, mt := range *m {
			if len(mt) == 1 && mt[0] == 'C' {
				*m = slices.Delete((*m), i, i+1)
				return true
			}
		}
		// If no explicit colorless source, try to remove any single-color source
		colors := []rune{'W', 'U', 'B', 'R', 'G'}
		for i, mt := range *m {
			if len(mt) == 1 {
				for _, color := range colors {
					if mt[0] == color {
						*m = slices.Delete((*m), i, i+1)
						return true
					}
				}
			}
		}
		// Could not find any source for colorless
		return false
	} else {
		// If the requested manaType is a specific color (W, U, B, R, G)
		for i, mt := range *m {
			if len(mt) == 1 && mt[0] == manaType {
				*m = slices.Delete((*m), i, i+1)
				return true
			}
		}
		// Could not find the specific colored source
		return false
	}
}

func (m ManaPool) ParseCost(cost string) map[rune]int {
	costMap := make(map[rune]int, 0)

	// Parse the new format with curly braces: {3}{G}{R}
	i := 0
	for i < len(cost) {
		if cost[i] == '{' {
			// Find the closing brace
			j := i + 1
			for j < len(cost) && cost[j] != '}' {
				j++
			}
			if j < len(cost) {
				// Extract content between braces
				content := cost[i+1 : j]

				// Check if it's a number (colorless mana)
				if intValue, err := strconv.Atoi(content); err == nil {
					costMap['C'] += intValue
				} else if len(content) == 1 {
					// Single color like {G}, {R}, etc.
					costMap[rune(content[0])]++
				} else if len(content) == 3 && content[1] == '/' {
					// Hybrid mana like {R/W} - for now, treat as requiring either color
					// This is a simplified implementation - hybrid mana is more complex
					color1 := rune(content[0])
					// For simplicity, we'll require one of the colors
					// A more complete implementation would need special handling
					costMap[color1]++ // Just use the first color for now
				}
				i = j + 1
			} else {
				i++
			}
		} else {
			i++
		}
	}
	return costMap
}

func (m ManaPool) CanPay(cost string) bool {
	requiredMana := m.ParseCost(cost)
	fmt.Println("CanPay: requiredMana", requiredMana)

	availableColored := make(map[rune]int)
	availableColorlessFromSources := 0 // Mana from sources like Sol Ring ("2")
	requiredColorless := requiredMana['C']

	// Count available mana from sources in the pool
	for _, manaType := range m {
		fmt.Println("manaType", string(manaType), len(manaType))
		if len(manaType) == 1 {
			r := manaType[0]
			switch r {
			case 'W', 'U', 'B', 'R', 'G':
				availableColored[r]++
			case 'C': // Source produces explicit colorless mana
				availableColorlessFromSources++
			default:
				if r >= '0' && r <= '9' {
					digitValue := int(r - '0')
					availableColorlessFromSources += digitValue
				}

				// Ignore other single-rune types?
			}
		} else {
			// This is a multi-rune source (like ['B', 'W']) or a digit string like []rune{'2'}.
			// Let's handle digit strings here.
			digitValue := 0
			isDigitString := true
			for _, r := range manaType {
				if r < '0' || r > '9' {
					isDigitString = false
					break
				}
				digitValue = digitValue*10 + int(r-'0')
			}
			if isDigitString && len(manaType) > 0 {
				availableColorlessFromSources += digitValue
			} else {
				// Multi-rune sources like ['B', 'W'] are not counted for specific colors
				// in this simplified model. They could potentially contribute to generic,
				// but the current structure makes this hard to model correctly without
				// a combinatorial check. Let's ignore them for now in the counts.
			}
		}
	}

	fmt.Println("CanPay: availableColored", availableColored)
	fmt.Println("CanPay: availableColorlessFromSources", availableColorlessFromSources)

	// Check if available colored mana is sufficient for required colored mana
	for manaType, required := range requiredMana {
		if manaType == 'C' {
			continue
		}
		if availableColored[manaType] < required {
			fmt.Printf("CanPay: Not enough %c mana. Required: %d, Available: %d\n", manaType, required, availableColored[manaType])
			return false
		}
	}

	// Calculate mana available to pay for colorless
	// This includes dedicated colorless sources and any excess colored mana
	availableForColorless := availableColorlessFromSources
	for manaType, available := range availableColored {
		required := requiredMana[manaType] // requiredColored[manaType] will be 0 if not required
		excess := available - required
		if excess > 0 {
			availableForColorless += excess // Excess colored mana can pay for colorless
		}
	}

	// Check if available for colorless is sufficient for required colorless
	if availableForColorless < requiredColorless {
		fmt.Printf("CanPay: Not enough generic/colorless mana. Required: %d, Available for colorless: %d\n", requiredColorless, availableForColorless)
		return false
	}

	// If we passed all checks, we can pay
	return true
}

func (g *GameState) AvailableMana(player *Player, pPool ManaPool) (pool ManaPool) {
	for _, card := range player.Battlefield {
		fmt.Println("checking:", card)
		if !card.IsActive() || card.ManaProduction == nil || len(card.ManaProduction) == 0 {
			continue
		}

		for _, manaStr := range card.ManaProduction {
			manaRunes := []rune(manaStr)
			pool.AddMana(manaRunes)
		}
	}
	pool = append(pool, pPool...)
	return pool
}

func (m *ManaPool) Pay(cost string) error {
	if !m.CanPay(cost) {
		return fmt.Errorf("not enough mana to pay the cost")
	}

	requiredMana := m.ParseCost(cost)
	fmt.Printf("Pay: paying: %v\n", requiredMana)

	// First, pay for specific colored mana requirements
	for manaType, count := range requiredMana {
		if manaType == 'C' {
			continue // Handle colorless separately
		}
		for i := 0; i < count; i++ {
			if !m.RemoveMana(manaType) {
				panic(fmt.Sprintf("Should have been able to remove mana (%c), but couldn't (%v)", manaType, *m))
			}
		}
	}

	// Then pay for colorless mana requirements
	colorlessRequired := requiredMana['C']
	for i := 0; i < colorlessRequired; i++ {
		if !m.RemoveMana('C') {
			panic(fmt.Sprintf("Should have been able to remove colorless mana, but couldn't (%v)", *m))
		}
	}

	return nil
}
