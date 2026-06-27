package domain

import "testing"

func TestGenerateQuestRewardTiers(t *testing.T) {
	standard := &Quest{ID: "s", Type: QuestTypeActionTracker, Metric: MetricPlayLands, Target: 10, DeadlineDays: 20, RewardTier: TierStandard}
	q := standard.GenerateQuest(0)
	if q.Reward.Gold <= 0 {
		t.Errorf("standard reward should give gold, got %d", q.Reward.Gold)
	}
	if q.Reward.Cards != 0 {
		t.Errorf("standard reward should give no cards, got %d", q.Reward.Cards)
	}

	themed := &Quest{ID: "t", Type: QuestTypeActionTracker, Metric: MetricCastColor, Color: ColorRed, Target: 10, DeadlineDays: 20, RewardTier: TierThemed}
	qt := themed.GenerateQuest(0)
	if qt.Reward.Cards != 2 {
		t.Errorf("themed reward should give 2 cards, got %d", qt.Reward.Cards)
	}
	if qt.Reward.CardColor != ColorRed {
		t.Errorf("themed cast-red reward color = %b, want red", qt.Reward.CardColor)
	}

	challenge := &Quest{ID: "c", Type: QuestTypeDeckConstraint, Constraint: ConstraintMonoColor, Color: ColorGreen, DeadlineDays: 20, RewardTier: TierChallenge}
	qc := challenge.GenerateQuest(0)
	if qc.Reward.Cards != 1 {
		t.Errorf("challenge reward should give 1 card, got %d", qc.Reward.Cards)
	}
	if qc.Reward.CardColor != ColorGreen {
		t.Errorf("challenge mono-green reward color = %b, want green", qc.Reward.CardColor)
	}

	// Themed quests grind across multiple duels, so they must out-pay the
	// single-duel challenge quests.
	if qt.Reward.Gold <= qc.Reward.Gold {
		t.Errorf("themed gold %d should exceed challenge gold %d", qt.Reward.Gold, qc.Reward.Gold)
	}
	if qt.Reward.Cards <= qc.Reward.Cards {
		t.Errorf("themed cards %d should exceed challenge cards %d", qt.Reward.Cards, qc.Reward.Cards)
	}
}

func TestGenerateQuestScalesWithProgression(t *testing.T) {
	def := &Quest{ID: "x", Type: QuestTypeActionTracker, Metric: MetricPlayLands, Target: 10, DeadlineDays: 20, RewardTier: TierStandard}
	low := def.GenerateQuest(0).Reward.Gold
	high := def.GenerateQuest(5).Reward.Gold
	if high <= low {
		t.Errorf("reward should scale up with progression: low=%d high=%d", low, high)
	}
}

func TestGenerateQuestCopiesObjective(t *testing.T) {
	def := &Quest{
		ID: "cast", Type: QuestTypeActionTracker, Metric: MetricCastType,
		CardTypes: []CardType{CardTypeInstant, CardTypeSorcery},
		Target:    15, DeadlineDays: 25, RewardTier: TierThemed,
		Title: "Slinger", Description: "Cast 15 instants or sorceries",
	}
	q := def.GenerateQuest(0)
	if q.ID != "cast" || q.Target != 15 || q.DaysRemaining != 25 {
		t.Errorf("quest fields not copied: %+v", q)
	}
	if len(q.CardTypes) != 2 {
		t.Errorf("card types not copied: %v", q.CardTypes)
	}
	if q.Title != "Slinger" {
		t.Errorf("title not copied: %q", q.Title)
	}
}

func TestGrantQuestReward(t *testing.T) {
	p := &Player{Character: Character{CardCollection: NewCardCollection(), Life: 10}, Gold: 100, Amulets: make(map[ColorMask]int)}
	r := QuestReward{Gold: 250, Cards: 2, CardColor: ColorRed, Amulets: 1, AmuletColor: ColorBlue, ManaLinks: 1}

	before := p.NumCards()
	cards := GrantQuestReward(p, r)

	if p.Gold != 350 {
		t.Errorf("gold = %d, want 350", p.Gold)
	}
	if p.Life != 11 {
		t.Errorf("life = %d, want 11 (one mana link)", p.Life)
	}
	if p.Amulets[ColorBlue] != 1 {
		t.Errorf("blue amulets = %d, want 1", p.Amulets[ColorBlue])
	}
	if got := p.NumCards() - before; got != 2 {
		t.Errorf("added %d cards, want 2", got)
	}
	if len(cards) != 2 {
		t.Errorf("returned %d cards, want 2", len(cards))
	}
}

func TestRandomSingleRewardIsSingleKind(t *testing.T) {
	// A legacy reward should always grant exactly one kind of thing.
	for range 50 {
		r := RandomSingleReward(2, ColorWhite)
		kinds := 0
		if r.Gold > 0 {
			kinds++
		}
		if r.Cards > 0 {
			kinds++
		}
		if r.Amulets > 0 {
			kinds++
		}
		if r.ManaLinks > 0 {
			kinds++
		}
		if kinds != 1 {
			t.Fatalf("RandomSingleReward granted %d kinds, want exactly 1: %+v", kinds, r)
		}
	}
}

func TestRedeemFulfilledQuests(t *testing.T) {
	p := &Player{Character: Character{CardCollection: NewCardCollection()}, Gold: 0}
	done := &Quest{Type: QuestTypeActionTracker, Target: 5, Progress: 5, Reward: QuestReward{Gold: 100, Cards: 1, CardColor: ColorGreen}}
	pending := &Quest{Type: QuestTypeActionTracker, Target: 5, Progress: 2, Reward: QuestReward{Gold: 100}}
	p.ActiveQuests = []*Quest{done, pending}

	rewards := p.RedeemFulfilledQuests(&City{Name: "Anytown"})
	if len(rewards) != 1 {
		t.Fatalf("redeemed %d quests, want 1", len(rewards))
	}
	if rewards[0].Reward.Gold != 100 || len(rewards[0].Cards) != 1 {
		t.Errorf("reward = %+v, want gold 100 and 1 card", rewards[0])
	}
	if p.Gold != 100 {
		t.Errorf("player gold = %d, want 100", p.Gold)
	}
	if len(p.ActiveQuests) != 1 || p.ActiveQuests[0] != pending {
		t.Errorf("only the unfulfilled quest should remain, got %d", len(p.ActiveQuests))
	}
}

func TestRedeemFulfilledQuestsCompletesDeliveryOnArrival(t *testing.T) {
	p := &Player{Character: Character{CardCollection: NewCardCollection()}}
	target := &City{Name: "Destination"}
	delivery := &Quest{Type: QuestTypeDelivery, TargetCity: target, Reward: QuestReward{Gold: 50}}
	p.ActiveQuests = []*Quest{delivery}

	// Entering an unrelated city must not complete or redeem the delivery.
	if rewards := p.RedeemFulfilledQuests(&City{Name: "Elsewhere"}); len(rewards) != 0 {
		t.Fatalf("delivery should not pay out at the wrong city, got %d", len(rewards))
	}

	// Arriving at the target city fulfills and redeems it in the same visit.
	rewards := p.RedeemFulfilledQuests(target)
	if len(rewards) != 1 || rewards[0].Reward.Gold != 50 {
		t.Fatalf("delivery should pay out on arrival, got %+v", rewards)
	}
	if len(p.ActiveQuests) != 0 {
		t.Errorf("redeemed delivery should be removed, %d remain", len(p.ActiveQuests))
	}
}

func TestQuestRewardDescription(t *testing.T) {
	tests := []struct {
		r    QuestReward
		want string
	}{
		{QuestReward{}, "a reward"},
		{QuestReward{ManaLinks: 1}, "a mana link"},
		{QuestReward{Amulets: 1, AmuletColor: ColorWhite}, "a White amulet"},
		{QuestReward{Gold: 120, Cards: 1}, "120 gold and a card"},
		{QuestReward{Gold: 150, Cards: 2}, "150 gold and 2 cards"},
	}
	for _, tt := range tests {
		if got := tt.r.Description(); got != tt.want {
			t.Errorf("Description(%+v) = %q, want %q", tt.r, got, tt.want)
		}
	}
}
