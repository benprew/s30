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
	if !attacker.HasKeyword(effects.KeywordVigilance) {
		attacker.Tapped = true
	}
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
	if landType := attacker.LandwalkType(); landType != "" {
		defendingPlayer := g.Players[(g.ActivePlayer+1)%len(g.Players)]
		if defendingPlayer.ControlsLandType(landType) {
			return fmt.Errorf("creature with %s cannot be blocked", landType+"walk")
		}
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

func (g *GameState) combatHasFirstStrike() bool {
	for _, attacker := range g.Attackers {
		if attacker.HasKeyword(effects.KeywordFirstStrike) {
			return true
		}
		for _, blocker := range g.BlockerMap[attacker] {
			if blocker.HasKeyword(effects.KeywordFirstStrike) {
				return true
			}
		}
	}
	return false
}

func (g *GameState) ResolveFirstStrikeDamage() {
	defendingPlayer := g.Players[(g.ActivePlayer+1)%len(g.Players)]
	attackingPlayer := g.Players[g.ActivePlayer]

	for _, attacker := range g.Attackers {
		blockers := g.BlockerMap[attacker]
		attackerHasFS := attacker.HasKeyword(effects.KeywordFirstStrike)

		if len(blockers) == 0 {
			if attackerHasFS {
				damage := attacker.EffectivePower()
				defendingPlayer.ReceiveDamage(damage)
				if attacker.HasKeyword(effects.KeywordLifelink) {
					attackingPlayer.GainLife(damage)
				}
			}
		} else {
			if attackerHasFS {
				hasTrample := attacker.HasKeyword(effects.KeywordTrample)
				hasDeathtouch := attacker.HasKeyword(effects.KeywordDeathtouch)
				remainingDamage := attacker.EffectivePower()
				for i, blocker := range blockers {
					lethal := blocker.EffectiveToughness() - blocker.DamageTaken
					if lethal < 0 {
						lethal = 0
					}
					if hasDeathtouch && lethal > 1 {
						lethal = 1
					}
					assigned := lethal
					if !hasTrample && i == len(blockers)-1 {
						assigned = remainingDamage
					} else if assigned > remainingDamage {
						assigned = remainingDamage
					}
					blocker.ReceiveDamage(assigned)
					if hasDeathtouch {
						blocker.DeathtouchDamaged = true
					}
					remainingDamage -= assigned
				}
				if remainingDamage > 0 && hasTrample {
					defendingPlayer.ReceiveDamage(remainingDamage)
				}
				if attacker.HasKeyword(effects.KeywordLifelink) {
					attackingPlayer.GainLife(attacker.EffectivePower())
				}
			}
			for _, blocker := range blockers {
				if blocker.HasKeyword(effects.KeywordFirstStrike) {
					blockerDamage := blocker.EffectivePower()
					attacker.ReceiveDamage(blockerDamage)
					if blocker.HasKeyword(effects.KeywordDeathtouch) {
						attacker.DeathtouchDamaged = true
					}
					if blocker.HasKeyword(effects.KeywordLifelink) {
						defendingPlayer.GainLife(blockerDamage)
					}
				}
			}
		}
	}
}

// ResolveCombatDamage resolves normal combat damage. Creatures killed by
// first strike are already in the graveyard (via state-based actions) and
// won't deal damage since they're no longer on the battlefield.
func (g *GameState) ResolveCombatDamage() {
	defendingPlayer := g.Players[(g.ActivePlayer+1)%len(g.Players)]
	attackingPlayer := g.Players[g.ActivePlayer]

	for _, attacker := range g.Attackers {
		blockers := g.BlockerMap[attacker]
		attackerHasFS := attacker.HasKeyword(effects.KeywordFirstStrike)

		if len(blockers) == 0 {
			if !attackerHasFS && attacker.CurrentZone == ZoneBattlefield {
				damage := attacker.EffectivePower()
				defendingPlayer.ReceiveDamage(damage)
				if attacker.HasKeyword(effects.KeywordLifelink) {
					attackingPlayer.GainLife(damage)
				}
			}
		} else {
			hasTrample := attacker.HasKeyword(effects.KeywordTrample)
			hasDeathtouch := attacker.HasKeyword(effects.KeywordDeathtouch)
			if !attackerHasFS && attacker.CurrentZone == ZoneBattlefield {
				remainingDamage := attacker.EffectivePower()
				for i, blocker := range blockers {
					lethal := blocker.EffectiveToughness() - blocker.DamageTaken
					if lethal < 0 {
						lethal = 0
					}
					if hasDeathtouch && lethal > 1 {
						lethal = 1
					}
					assigned := lethal
					if !hasTrample && i == len(blockers)-1 {
						assigned = remainingDamage
					} else if assigned > remainingDamage {
						assigned = remainingDamage
					}
					blocker.ReceiveDamage(assigned)
					if hasDeathtouch {
						blocker.DeathtouchDamaged = true
					}
					remainingDamage -= assigned
				}
				if remainingDamage > 0 && hasTrample {
					defendingPlayer.ReceiveDamage(remainingDamage)
				}
				if attacker.HasKeyword(effects.KeywordLifelink) {
					attackingPlayer.GainLife(attacker.EffectivePower())
				}
			}
			for _, blocker := range blockers {
				if !blocker.HasKeyword(effects.KeywordFirstStrike) && blocker.CurrentZone == ZoneBattlefield {
					blockerDamage := blocker.EffectivePower()
					attacker.ReceiveDamage(blockerDamage)
					if blocker.HasKeyword(effects.KeywordDeathtouch) {
						attacker.DeathtouchDamaged = true
					}
					if blocker.HasKeyword(effects.KeywordLifelink) {
						defendingPlayer.GainLife(blockerDamage)
					}
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
