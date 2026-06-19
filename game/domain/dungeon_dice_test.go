package domain

import (
	"math/rand"
	"strings"
	"testing"
)

func TestApplyDiceLifeAdvantageAccumulatesAndClearsTile(t *testing.T) {
	st := &DungeonState{}
	p := newTestPlayer()
	tile := &DungeonTile{Type: DungeonTileDice, Dice: &DiceEffect{Type: DiceAdvantage, LifeMod: 3}}

	desc := st.ApplyDice(tile, p)

	if p.BonusDuelLife != 3 {
		t.Errorf("expected life bonus 3, got %d", p.BonusDuelLife)
	}
	if tile.Type != DungeonTileEmpty || tile.Dice != nil {
		t.Errorf("expected dice tile cleared, got type=%v dice=%v", tile.Type, tile.Dice)
	}
	if desc == "" {
		t.Error("expected a non-empty description")
	}
}

func TestApplyDiceLifeBonusIsAdditive(t *testing.T) {
	st := &DungeonState{}
	p := newTestPlayer()
	st.ApplyDice(&DungeonTile{Type: DungeonTileDice, Dice: &DiceEffect{Type: DiceAdvantage, LifeMod: 3}}, p)
	st.ApplyDice(&DungeonTile{Type: DungeonTileDice, Dice: &DiceEffect{Type: DiceDisadvantage, LifeMod: -2}}, p)

	if p.BonusDuelLife != 1 {
		t.Errorf("expected net life bonus 1 (3-2), got %d", p.BonusDuelLife)
	}
}

func TestApplyDiceCardAdvantageQueuesPendingCard(t *testing.T) {
	st := &DungeonState{}
	p := newTestPlayer()
	card := &Card{CardName: "Serra Angel"}
	tile := &DungeonTile{Type: DungeonTileDice, Dice: &DiceEffect{Type: DiceAdvantage, Card: card}}

	st.ApplyDice(tile, p)

	if len(p.BonusDuelCards) != 1 || p.BonusDuelCards[0] != card {
		t.Errorf("expected card queued in PendingCards, got %v", p.BonusDuelCards)
	}
	if p.BonusDuelLife != 0 {
		t.Errorf("card advantage should not change life bonus, got %d", p.BonusDuelLife)
	}
	if tile.Type != DungeonTileEmpty {
		t.Errorf("expected dice tile cleared, got %v", tile.Type)
	}
}

func TestApplyDiceNoOpOnNonDiceTile(t *testing.T) {
	st := &DungeonState{}
	p := newTestPlayer()
	if desc := st.ApplyDice(&DungeonTile{Type: DungeonTileEmpty}, p); desc != "" {
		t.Errorf("non-dice tile should return empty description, got %q", desc)
	}
	st.ApplyDice(nil, p) // must not panic
}

func TestRollDiceEffectApprenticeNeverDisadvantage(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for range 200 {
		e := rollDiceEffect(1, rng, nil)
		if e.Type == DiceDisadvantage {
			t.Fatalf("apprentice (level 1) dungeons must never roll a disadvantage")
		}
	}
}

func TestRollDiceEffectHigherLevelsCanGiveDisadvantage(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	sawDisadvantage := false
	for range 500 {
		if rollDiceEffect(5, rng, nil).Type == DiceDisadvantage {
			sawDisadvantage = true
			break
		}
	}
	if !sawDisadvantage {
		t.Error("expected higher-level dungeons to occasionally roll a disadvantage")
	}
}

func TestRollDiceEffectAdvantageIsLifeOrCard(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	pool := []*Card{{CardName: "Serra Angel", CardType: CardTypeCreature}}
	sawLife, sawCard := false, false
	for range 500 {
		e := rollDiceEffect(1, rng, pool)
		if e.Type != DiceAdvantage {
			continue
		}
		switch {
		case e.Card != nil:
			sawCard = true
		case e.LifeMod > 0:
			sawLife = true
		default:
			t.Fatalf("advantage with neither a card nor positive life: %+v", e)
		}
	}
	if !sawLife || !sawCard {
		t.Errorf("expected both life and card advantages over many rolls (life=%v card=%v)", sawLife, sawCard)
	}
}

func TestRollDiceEffectWithoutPoolNeverGrantsCard(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for range 500 {
		if e := rollDiceEffect(1, rng, nil); e.Card != nil {
			t.Fatalf("no card should be granted when the deck pool is empty, got %q", e.Card.CardName)
		}
	}
}

func TestRollDiceEffectCardGrantsComeFromDeckPool(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	pool := []*Card{
		{CardName: "Serra Angel", CardType: CardTypeCreature},
		{CardName: "Sol Ring", CardType: CardTypeArtifact},
	}
	inPool := map[string]bool{"Serra Angel": true, "Sol Ring": true}
	for range 500 {
		if e := rollDiceEffect(1, rng, pool); e.Card != nil && !inPool[e.Card.CardName] {
			t.Fatalf("granted card %q is not from the deck pool", e.Card.CardName)
		}
	}
}

func TestRandomDiceCardExcludesLands(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	pool := []*Card{{CardName: "Mountain", CardType: CardTypeLand}}
	for range 100 {
		if c := randomDiceCard(rng, pool); c != nil {
			t.Fatalf("lands must never be granted, got %q", c.CardName)
		}
	}
}

func TestRandomDiceCardOnlyGrantsCardsThatCanStartOnBattlefield(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	pool := []*Card{
		{CardName: "Eternal Warrior", CardType: CardTypeEnchantment, TypeLine: "Enchantment - Aura", Keywords: []string{"Enchant"}},
		{CardName: "Lightning Bolt", CardType: CardTypeInstant},
		{CardName: "Wrath of God", CardType: CardTypeSorcery},
		{CardName: "Forest", CardType: CardTypeLand},
		{CardName: "Serra Angel", CardType: CardTypeCreature},
		{CardName: "Sol Ring", CardType: CardTypeArtifact},
		{CardName: "Crusade", CardType: CardTypeEnchantment, TypeLine: "Enchantment"},
	}
	eligible := map[string]bool{
		"Serra Angel": true,
		"Sol Ring":    true,
		"Crusade":     true,
	}

	for range 500 {
		card := randomDiceCard(rng, pool)
		if card == nil {
			t.Fatal("expected an eligible card")
		}
		if !eligible[card.CardName] {
			t.Fatalf("card %q cannot start on an empty battlefield", card.CardName)
		}
	}
}

func TestRandomDiceCardFallsBackWhenOnlyAurasAreAvailable(t *testing.T) {
	rng := rand.New(rand.NewSource(8))
	pool := []*Card{
		{
			CardName: "Eternal Warrior",
			CardType: CardTypeEnchantment,
			TypeLine: "Enchantment - Aura",
			ParsedAbilities: []ParsedAbility{
				{TargetSpec: &ParsedTargetSpec{Type: "creature", Condition: "enchant"}},
			},
		},
	}

	if card := randomDiceCard(rng, pool); card != nil {
		t.Fatalf("auras must not be granted on an empty battlefield, got %q", card.CardName)
	}
}

func TestDescribeDiceEffectMentionsKind(t *testing.T) {
	if !strings.Contains(strings.ToLower(DescribeDiceEffect(&DiceEffect{Type: DiceAdvantage, LifeMod: 3})), "life") {
		t.Error("life advantage description should mention life")
	}
	card := &Card{CardName: "Serra Angel"}
	if !strings.Contains(DescribeDiceEffect(&DiceEffect{Type: DiceAdvantage, Card: card}), "Serra Angel") {
		t.Error("card advantage description should name the card")
	}
	if DescribeDiceEffect(nil) != "" {
		t.Error("nil effect should describe as empty")
	}
}
