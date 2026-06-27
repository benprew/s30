package duel

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/s30/game/domain"
)

// evaluateQuestConstraints validates the player's submitted (constructed) deck
// against each active deck-constraint quest at duel start, recording which held
// so a qualifying win can complete them. It also collects warnings for quests
// the deck doesn't satisfy — the player may still duel, but it won't count.
//
// The constructed deck (un-padded) is used deliberately: GetDuelDeck pads to
// MinDeckSize with random basics, which would inflate fat-deck/color-light
// checks. ConstraintNoAttacking is not a deck property and is settled at duel
// end.
func (s *DuelScreen) evaluateQuestConstraints() {
	s.deckConstraintMet = make(map[*domain.Quest]bool)
	s.questWarnings = nil

	deck := s.player.CardCollection.GetDeck(s.player.ActiveDeck)
	for _, q := range s.player.ActiveQuests {
		if q.Type != domain.QuestTypeDeckConstraint || q.Constraint == domain.ConstraintNoAttacking {
			continue
		}
		ok, reason := domain.ValidateDeckConstraint(q, deck)
		s.deckConstraintMet[q] = ok
		if !ok {
			s.questWarnings = append(s.questWarnings,
				fmt.Sprintf("%s: %s — this duel won't count", q.Title, reason))
		}
	}

	for _, w := range s.questWarnings {
		if s.diceNotice != "" {
			s.diceNotice += "\n"
		}
		s.diceNotice += w
	}
}

// applyQuestProgress reads the engine's per-duel objective tally and applies it
// to the player's active deck-changing quests.
func (s *DuelScreen) applyQuestProgress(won bool) {
	if len(s.player.ActiveQuests) == 0 {
		return
	}
	tally := s.buildDuelTally()
	domain.ApplyDuelResultToQuests(s.player.ActiveQuests, tally, won, s.deckConstraintMet)
}

func (s *DuelScreen) buildDuelTally() domain.DuelTally {
	obj := s.game.DuelObjectivesFor(s.human.PlayerID())
	tally := domain.DuelTally{
		SpellsByColor:           make(map[domain.ColorMask]int),
		SpellsByType:            make(map[domain.CardType]int),
		LandsPlayed:             obj.LandsPlayed,
		AttackersDeclared:       obj.AttackersDeclared,
		EnemyCreaturesDestroyed: obj.OpponentCreaturesDestroyed,
		DirectDamage:            obj.NonCombatDamageDealt,
	}
	for c, n := range obj.SpellsByColor {
		if mask := coreColorToMask(c); mask != domain.ColorColorless {
			tally.SpellsByColor[mask] += n
		}
	}
	for ct, n := range obj.SpellsByType {
		if dct := coreTypeToCardType(ct); dct != "" {
			tally.SpellsByType[dct] += n
		}
	}
	return tally
}

func coreColorToMask(c core.Color) domain.ColorMask {
	switch c {
	case core.White:
		return domain.ColorWhite
	case core.Blue:
		return domain.ColorBlue
	case core.Black:
		return domain.ColorBlack
	case core.Red:
		return domain.ColorRed
	case core.Green:
		return domain.ColorGreen
	default:
		return domain.ColorColorless
	}
}

func coreTypeToCardType(t core.CardType) domain.CardType {
	switch t {
	case core.TypeCreature:
		return domain.CardTypeCreature
	case core.TypeInstant:
		return domain.CardTypeInstant
	case core.TypeSorcery:
		return domain.CardTypeSorcery
	case core.TypeArtifact:
		return domain.CardTypeArtifact
	case core.TypeEnchantment:
		return domain.CardTypeEnchantment
	case core.TypeLand:
		return domain.CardTypeLand
	default:
		return ""
	}
}
