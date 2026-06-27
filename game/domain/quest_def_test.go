package domain

import "testing"

func TestQuestDefsLoad(t *testing.T) {
	if len(QuestDefs) == 0 {
		t.Fatal("expected quest defs to load from TOML, got none")
	}

	// Spot-check a representative action quest.
	d, ok := QuestDefs["cast_black_red"]
	if !ok {
		t.Fatal("expected cast_black_red quest def")
	}
	if d.Type != QuestTypeActionTracker {
		t.Errorf("cast_black_red: type = %v, want ActionTracker", d.Type)
	}
	if d.Metric != MetricCastColor {
		t.Errorf("cast_black_red: metric = %v, want MetricCastColor", d.Metric)
	}
	if d.Color != (ColorBlack | ColorRed) {
		t.Errorf("cast_black_red: color mask = %b, want %b", d.Color, ColorBlack|ColorRed)
	}
	if d.Target != 30 {
		t.Errorf("cast_black_red: target = %d, want 30", d.Target)
	}
	if d.RewardTier != TierThemed {
		t.Errorf("cast_black_red: tier = %v, want Themed", d.RewardTier)
	}

	// Spot-check a representative constraint quest.
	c, ok := QuestDefs["fat_deck_win"]
	if !ok {
		t.Fatal("expected fat_deck_win quest def")
	}
	if c.Type != QuestTypeDeckConstraint {
		t.Errorf("fat_deck_win: type = %v, want DeckConstraint", c.Type)
	}
	if c.Constraint != ConstraintFatDeck {
		t.Errorf("fat_deck_win: constraint = %v, want ConstraintFatDeck", c.Constraint)
	}
	if c.ConstraintN != 75 {
		t.Errorf("fat_deck_win: n = %d, want 75", c.ConstraintN)
	}
	if c.RewardTier != TierChallenge {
		t.Errorf("fat_deck_win: tier = %v, want Challenge", c.RewardTier)
	}
}

func TestQuestDefsCoverAllKinds(t *testing.T) {
	metrics := map[QuestMetric]bool{}
	constraints := map[QuestConstraint]bool{}
	for _, d := range QuestDefs {
		switch d.Type {
		case QuestTypeActionTracker:
			metrics[d.Metric] = true
		case QuestTypeDeckConstraint:
			constraints[d.Constraint] = true
		}
	}
	wantMetrics := []QuestMetric{
		MetricCastColor, MetricPlayLands, MetricAttackCreatures,
		MetricDestroyEnemyCreatures, MetricCastType, MetricDirectDamage,
	}
	for _, m := range wantMetrics {
		if !metrics[m] {
			t.Errorf("no quest def covers metric %v", m)
		}
	}
	wantConstraints := []QuestConstraint{
		ConstraintMonoColor, ConstraintFatDeck, ConstraintLowCurve,
		ConstraintColorLight, ConstraintNoAttacking,
	}
	for _, c := range wantConstraints {
		if !constraints[c] {
			t.Errorf("no quest def covers constraint %v", c)
		}
	}
}

func TestParseColorMask(t *testing.T) {
	tests := []struct {
		in   string
		want ColorMask
	}{
		{"", ColorColorless},
		{"R", ColorRed},
		{"r", ColorRed},
		{"BR", ColorBlack | ColorRed},
		{"red", ColorRed},
		{"green", ColorGreen},
		{"wubrg", ColorAny},
		{"any", ColorAny},
		{"xyz", ColorColorless},
	}
	for _, tt := range tests {
		if got := ParseColorMask(tt.in); got != tt.want {
			t.Errorf("ParseColorMask(%q) = %b, want %b", tt.in, got, tt.want)
		}
	}
}

func TestQuestProgressLifecycle(t *testing.T) {
	q := &Quest{Type: QuestTypeActionTracker, Target: 10}
	if q.IsFulfilled() {
		t.Error("new tracker should not be fulfilled")
	}
	q.AddProgress(4)
	q.AddProgress(4)
	if q.Progress != 8 {
		t.Errorf("progress = %d, want 8", q.Progress)
	}
	if q.IsFulfilled() {
		t.Error("tracker at 8/10 should not be fulfilled")
	}
	q.AddProgress(5) // clamps to target
	if q.Progress != 10 {
		t.Errorf("progress = %d, want clamped 10", q.Progress)
	}
	if !q.IsFulfilled() {
		t.Error("tracker at 10/10 should be fulfilled")
	}

	c := &Quest{Type: QuestTypeDeckConstraint}
	if c.IsFulfilled() {
		t.Error("constraint quest should not be fulfilled before a win")
	}
	c.IsCompleted = true
	if !c.IsFulfilled() {
		t.Error("constraint quest should be fulfilled after IsCompleted")
	}
}
