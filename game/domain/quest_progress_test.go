package domain

import "testing"

func TestApplyDuelResultActionTracker(t *testing.T) {
	tally := DuelTally{
		SpellsByColor: map[ColorMask]int{ColorBlack: 2, ColorRed: 3, ColorGreen: 1},
		SpellsByType:  map[CardType]int{CardTypeInstant: 4, CardTypeSorcery: 2, CardTypeCreature: 5},
		LandsPlayed:   6,
		AttackersDeclared:       7,
		EnemyCreaturesDestroyed: 3,
		DirectDamage:            8,
	}

	tests := []struct {
		name string
		q    *Quest
		want int
	}{
		{"cast B or R", &Quest{Type: QuestTypeActionTracker, Metric: MetricCastColor, Color: ColorBlack | ColorRed, Target: 100}, 5},
		{"cast U only", &Quest{Type: QuestTypeActionTracker, Metric: MetricCastColor, Color: ColorBlue, Target: 100}, 0},
		{"play lands", &Quest{Type: QuestTypeActionTracker, Metric: MetricPlayLands, Target: 100}, 6},
		{"attack", &Quest{Type: QuestTypeActionTracker, Metric: MetricAttackCreatures, Target: 100}, 7},
		{"destroy", &Quest{Type: QuestTypeActionTracker, Metric: MetricDestroyEnemyCreatures, Target: 100}, 3},
		{"instants+sorceries", &Quest{Type: QuestTypeActionTracker, Metric: MetricCastType, CardTypes: []CardType{CardTypeInstant, CardTypeSorcery}, Target: 100}, 6},
		{"direct dmg", &Quest{Type: QuestTypeActionTracker, Metric: MetricDirectDamage, Target: 100}, 8},
	}
	for _, tt := range tests {
		tt.q.ApplyDuelResult(tally, false, false) // win/loss irrelevant for trackers
		if tt.q.Progress != tt.want {
			t.Errorf("%s: progress = %d, want %d", tt.name, tt.q.Progress, tt.want)
		}
	}
}

func TestApplyDuelResultConstraintDeckCheck(t *testing.T) {
	q := &Quest{Type: QuestTypeDeckConstraint, Constraint: ConstraintMonoColor}

	// Loss does not complete even if the deck held.
	q.ApplyDuelResult(DuelTally{}, false, true)
	if q.IsCompleted {
		t.Error("loss should not complete a constraint quest")
	}
	// Win but deck didn't hold.
	q.ApplyDuelResult(DuelTally{}, true, false)
	if q.IsCompleted {
		t.Error("win with failed deck check should not complete")
	}
	// Win with deck held.
	q.ApplyDuelResult(DuelTally{}, true, true)
	if !q.IsCompleted {
		t.Error("win with deck check held should complete")
	}
}

func TestApplyDuelResultNoAttacking(t *testing.T) {
	// Win without attacking → complete (deckConstraintMet irrelevant).
	q := &Quest{Type: QuestTypeDeckConstraint, Constraint: ConstraintNoAttacking}
	q.ApplyDuelResult(DuelTally{AttackersDeclared: 0}, true, false)
	if !q.IsCompleted {
		t.Error("win with zero attackers should complete no-attacking quest")
	}

	// Win but attacked → not complete.
	q2 := &Quest{Type: QuestTypeDeckConstraint, Constraint: ConstraintNoAttacking}
	q2.ApplyDuelResult(DuelTally{AttackersDeclared: 2}, true, true)
	if q2.IsCompleted {
		t.Error("win after attacking should not complete no-attacking quest")
	}
}

func TestApplyDuelResultToQuestsBatch(t *testing.T) {
	tracker := &Quest{Type: QuestTypeActionTracker, Metric: MetricPlayLands, Target: 10}
	constraint := &Quest{Type: QuestTypeDeckConstraint, Constraint: ConstraintFatDeck}
	quests := []*Quest{tracker, constraint}

	ApplyDuelResultToQuests(quests, DuelTally{LandsPlayed: 4}, true, map[*Quest]bool{constraint: true})

	if tracker.Progress != 4 {
		t.Errorf("tracker progress = %d, want 4", tracker.Progress)
	}
	if !constraint.IsCompleted {
		t.Error("constraint quest should complete on qualifying win")
	}
}
