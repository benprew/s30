package domain

import (
	"encoding/json"
	"testing"
)

func TestDeckQuestJSONRoundTrip(t *testing.T) {
	orig := &Quest{
		Type:          QuestTypeActionTracker,
		ID:            "cast_black_red",
		Title:         "Spells of Shadow & Flame",
		Description:   "Cast 30 black or red spells",
		DaysRemaining: 25,
		Metric:        MetricCastColor,
		Color:         ColorBlack | ColorRed,
		CardTypes:     []CardType{CardTypeInstant, CardTypeSorcery},
		Target:        30,
		Progress:      12,
		Constraint:    ConstraintNone,
		Reward:        QuestReward{Gold: 120, Cards: 1, CardColor: ColorRed},
	}

	data, err := json.Marshal([]*Quest{orig})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got []*Quest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d quests, want 1", len(got))
	}
	q := got[0]
	if q.ID != orig.ID || q.Metric != orig.Metric || q.Color != orig.Color ||
		q.Target != orig.Target || q.Progress != orig.Progress || q.Reward != orig.Reward {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", q, orig)
	}
	if len(q.CardTypes) != 2 || q.CardTypes[0] != CardTypeInstant {
		t.Errorf("CardTypes not preserved: %v", q.CardTypes)
	}
}

func TestCardManaValue(t *testing.T) {
	tests := []struct {
		cost string
		want int
	}{
		{"", 0},
		{"{0}", 0},
		{"{3}", 3},
		{"{G}", 1},
		{"{3}{G}{R}", 5},
		{"{2}{W}{W}", 4},
		{"{X}{R}", 1},
		{"{2/G}{2/G}", 2},
	}
	for _, tt := range tests {
		c := &Card{ManaCost: tt.cost}
		if got := c.ManaValue(); got != tt.want {
			t.Errorf("ManaValue(%q) = %d, want %d", tt.cost, got, tt.want)
		}
	}
}

func TestCardColorMask(t *testing.T) {
	c := &Card{Colors: []string{"B", "R"}}
	if got := c.ColorMask(); got != (ColorBlack | ColorRed) {
		t.Errorf("ColorMask = %b, want %b", got, ColorBlack|ColorRed)
	}
	colorless := &Card{}
	if got := colorless.ColorMask(); got != ColorColorless {
		t.Errorf("colorless ColorMask = %b, want 0", got)
	}
}

func TestPlayerQuestSlots(t *testing.T) {
	p := &Player{}
	if !p.CanAcceptQuest() {
		t.Fatal("empty player should accept a quest")
	}
	for i := range MaxActiveQuests {
		if !p.AddQuest(&Quest{ID: string(rune('a' + i))}) {
			t.Fatalf("AddQuest %d should succeed", i)
		}
	}
	if p.CanAcceptQuest() {
		t.Error("player at capacity should not accept more quests")
	}
	if p.AddQuest(&Quest{ID: "overflow"}) {
		t.Error("AddQuest past capacity should fail")
	}
	if !p.HasQuest("a") {
		t.Error("HasQuest should find an accepted quest")
	}
	if p.HasQuest("nope") {
		t.Error("HasQuest should not find a missing quest")
	}
}

func TestPlayerExpireQuests(t *testing.T) {
	p := &Player{}
	p.AddQuest(&Quest{ID: "expired", DaysRemaining: 0, Type: QuestTypeActionTracker, Target: 5})
	p.AddQuest(&Quest{ID: "alive", DaysRemaining: 3, Type: QuestTypeActionTracker, Target: 5})
	// A fulfilled-but-out-of-days quest must survive so it can still be redeemed.
	p.AddQuest(&Quest{ID: "done", DaysRemaining: 0, Type: QuestTypeActionTracker, Target: 5, Progress: 5})

	removed := p.ExpireQuests()
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if p.HasQuest("expired") {
		t.Error("expired quest should be removed")
	}
	if !p.HasQuest("alive") {
		t.Error("alive quest should remain")
	}
	if !p.HasQuest("done") {
		t.Error("fulfilled quest should remain redeemable even at 0 days")
	}
}

func TestExpireQuestsAlsoExpiresDeliveryAndDefeat(t *testing.T) {
	p := &Player{}
	// Delivery/defeat quests now auto-expire like any other (no city ban).
	p.AddQuest(&Quest{ID: "delivery", Type: QuestTypeDelivery, DaysRemaining: 0})
	p.AddQuest(&Quest{ID: "defeat", Type: QuestTypeDefeatEnemy, DaysRemaining: 0})
	// A defeated-but-unclaimed enemy quest survives so it can still be redeemed.
	p.AddQuest(&Quest{ID: "defeat-done", Type: QuestTypeDefeatEnemy, DaysRemaining: 0, IsCompleted: true})

	if removed := p.ExpireQuests(); removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	if p.HasQuest("delivery") || p.HasQuest("defeat") {
		t.Error("expired delivery/defeat quests should be removed")
	}
	if !p.HasQuest("defeat-done") {
		t.Error("completed-but-unclaimed defeat quest should remain redeemable")
	}
}
