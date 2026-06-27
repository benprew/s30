package domain

// DuelTally is the s30-side, per-player summary of objective-relevant actions
// from a finished duel. The duel screen builds it by translating the engine's
// per-duel objective accessor into domain types.
type DuelTally struct {
	SpellsByColor           map[ColorMask]int // single-color mask -> spells cast
	SpellsByType            map[CardType]int  // type -> spells cast
	LandsPlayed             int
	AttackersDeclared       int
	EnemyCreaturesDestroyed int
	DirectDamage            int // non-combat damage dealt to the enemy
}

var allColors = []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}

// metricIncrement returns how much an action-tracker quest's progress should
// advance given a duel's tally.
//
// Note: a multicolor spell is recorded once per color in SpellsByColor, so a
// gold card matching two of a quest's colors counts twice toward a 2-color
// "cast color" quest. Gold cards are rare and the over-count only helps the
// player, so this approximation is accepted for v1.
func metricIncrement(q *Quest, t DuelTally) int {
	switch q.Metric {
	case MetricCastColor:
		n := 0
		for _, c := range allColors {
			if q.Color&c != 0 {
				n += t.SpellsByColor[c]
			}
		}
		return n
	case MetricPlayLands:
		return t.LandsPlayed
	case MetricAttackCreatures:
		return t.AttackersDeclared
	case MetricDestroyEnemyCreatures:
		return t.EnemyCreaturesDestroyed
	case MetricCastType:
		n := 0
		for _, ct := range q.CardTypes {
			n += t.SpellsByType[ct]
		}
		return n
	case MetricDirectDamage:
		return t.DirectDamage
	default:
		return 0
	}
}

// ApplyDuelResult updates a single quest from a finished duel.
//
//   - Action-tracker quests accumulate metric progress regardless of win/loss.
//   - Deck-constraint quests complete only on a win while the constraint held:
//     ConstraintNoAttacking checks that the player declared zero attackers this
//     duel; every other constraint uses deckConstraintMet (the deck check
//     performed at duel start).
func (q *Quest) ApplyDuelResult(t DuelTally, won bool, deckConstraintMet bool) {
	switch q.Type {
	case QuestTypeActionTracker:
		q.AddProgress(metricIncrement(q, t))
	case QuestTypeDeckConstraint:
		if !won || q.IsCompleted {
			return
		}
		if q.Constraint == ConstraintNoAttacking {
			if t.AttackersDeclared == 0 {
				q.IsCompleted = true
			}
			return
		}
		if deckConstraintMet {
			q.IsCompleted = true
		}
	}
}

// ApplyDuelResultToQuests applies a finished duel's outcome to every quest in
// the slice. deckConstraintMet maps a quest to whether its deck check held at
// duel start (absent/false for action and no-attacking quests).
func ApplyDuelResultToQuests(quests []*Quest, t DuelTally, won bool, deckConstraintMet map[*Quest]bool) {
	for _, q := range quests {
		q.ApplyDuelResult(t, won, deckConstraintMet[q])
	}
}
