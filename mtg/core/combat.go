package core

import (
	"fmt"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
)

func (g *GameState) AvailableAttackers(player *Player) []*Card {
	attackers := []*Card{}
	for _, card := range player.Battlefield {
		if card.CardType == domain.CardTypeCreature && card.IsActive() {
			attackers = append(attackers, card)
		}
	}
	return attackers
}

func (g *GameState) DeclareAttacker(attacker *Card) error {
	if attacker.CardType != domain.CardTypeCreature {
		return fmt.Errorf("only creatures can attack")
	}
	if !attacker.IsActive() {
		return fmt.Errorf("creature cannot attack: tapped or inactive")
	}
	attacker.Tapped = true
	return nil
}

func (g *GameState) AvailableBlockers(player *Player) []*Card {
	blockers := []*Card{}
	for _, card := range player.Battlefield {
		if card.CardType == domain.CardTypeCreature && !card.Tapped && !g.isAlreadyBlocking(card) {
			blockers = append(blockers, card)
		}
	}
	return blockers
}

func (g *GameState) DeclareBlocker(blocker *Card, attacker *Card) error {
	if blocker.CardType != domain.CardTypeCreature {
		return fmt.Errorf("only creatures can block")
	}
	if blocker.Tapped {
		return fmt.Errorf("tapped creatures cannot block")
	}
	if g.isAlreadyBlocking(blocker) {
		return fmt.Errorf("creature is already blocking")
	}
	if attacker.HasKeyword(effects.KeywordFlying) && !blocker.HasKeyword(effects.KeywordFlying) && !blocker.HasKeyword(effects.KeywordReach) {
		return fmt.Errorf("creature with flying can only be blocked by creatures with flying or reach")
	}
	g.BlockerMap[attacker] = append(g.BlockerMap[attacker], blocker)
	return nil
}

func (g *GameState) isAlreadyBlocking(blocker *Card) bool {
	for _, blockers := range g.BlockerMap {
		for _, b := range blockers {
			if b == blocker {
				return true
			}
		}
	}
	return false
}

func (g *GameState) ResolveCombatDamage() {
	defendingPlayer := g.Players[(g.ActivePlayer+1)%len(g.Players)]
	attackingPlayer := g.Players[g.ActivePlayer]

	for _, attacker := range g.Attackers {
		blockers := g.BlockerMap[attacker]
		if len(blockers) == 0 {
			damage := attacker.EffectivePower()
			defendingPlayer.ReceiveDamage(damage)
			if attacker.HasKeyword(effects.KeywordLifelink) {
				attackingPlayer.GainLife(damage)
			}
		} else {
			for _, blocker := range blockers {
				attackerDamage := attacker.EffectivePower()
				blockerDamage := blocker.EffectivePower()
				blocker.ReceiveDamage(attackerDamage)
				attacker.ReceiveDamage(blockerDamage)
				if attacker.HasKeyword(effects.KeywordLifelink) {
					attackingPlayer.GainLife(attackerDamage)
				}
				if blocker.HasKeyword(effects.KeywordLifelink) {
					defendingPlayer.GainLife(blockerDamage)
				}
			}
		}
	}
}

func (g *GameState) CleanupDeadCreatures() {
	for _, player := range g.Players {
		deadCreatures := []*Card{}
		for _, card := range player.Battlefield {
			if card.CardType == domain.CardTypeCreature && card.IsDead() {
				deadCreatures = append(deadCreatures, card)
			}
		}
		for _, card := range deadCreatures {
			player.MoveTo(card, ZoneGraveyard)
		}
	}
}

func (g *GameState) ClearCombatState() {
	g.Attackers = nil
	g.BlockerMap = make(map[*Card][]*Card)
}
