package rules

import (
	"fmt"
	"slices"
)

type ManaPool [][]rune

func (m *ManaPool) AddMana(manaType []rune) {
	*m = append(*m, manaType)
}

func (m *ManaPool) RemoveMana(manaType rune) {
	for i, mt := range *m {
		if len(mt) == 1 && mt[0] == manaType {
			*m = slices.Delete((*m), i, i+1)
			break
		}
	}
}

func (m ManaPool) CanPay(cost string) bool {
	requiredMana := make(map[rune]int)
	for _, mana := range cost {
		requiredMana[mana]++
	}

	availableMana := make(map[rune]int)
	for _, manaType := range m {
		if len(manaType) == 1 {
			availableMana[manaType[0]]++
		}
	}

	for manaType, required := range requiredMana {
		if availableMana[manaType] < required {
			return false
		}
	}

	return true
}

func (m *ManaPool) Pay(cost string) error {
	if !m.CanPay(cost) {
		return fmt.Errorf("not enough mana to pay the cost")
	}
	for _, mana := range cost {
		m.RemoveMana(mana)
	}
	return nil
}
