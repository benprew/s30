package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
)

const targetTypeCreature = "creature"

func makeCreature(id EntityID, name string, power, toughness int, owner *Player) *Card {
	return &Card{
		Card: domain.Card{
			CardName:  name,
			CardType:  domain.CardTypeCreature,
			Power:     power,
			Toughness: toughness,
		},
		ID:          id,
		Owner:       owner,
		CurrentZone: ZoneBattlefield,
		Active:      true,
	}
}

func makeAura(id EntityID, name string, owner *Player, abilities []domain.ParsedAbility) *Card {
	return &Card{
		Card: domain.Card{
			CardName:        name,
			CardType:        domain.CardTypeEnchantment,
			ManaCost:        "{R}",
			ParsedAbilities: abilities,
		},
		ID:          id,
		Owner:       owner,
		CurrentZone: ZoneHand,
	}
}

func TestAuraStatBoost(t *testing.T) {
	player := &Player{
		ID:          0,
		LifeTotal:   20,
		ManaPool:    ManaPool{},
		Hand:        []*Card{},
		Battlefield: []*Card{},
		Graveyard:   []*Card{},
		Turn:        &Turn{},
		InputChan:   make(chan PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
	}

	creature := makeCreature(1, "Grizzly Bears", 2, 2, player)
	player.Battlefield = append(player.Battlefield, creature)

	aura := makeAura(2, "Holy Strength", player, []domain.ParsedAbility{
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     1,
				ToughnessBoost: 2,
			},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})

	creature.Attachments = append(creature.Attachments, aura)
	aura.AttachedTo = creature

	if creature.EffectivePower() != 3 {
		t.Errorf("expected effective power 3, got %d", creature.EffectivePower())
	}
	if creature.EffectiveToughness() != 4 {
		t.Errorf("expected effective toughness 4, got %d", creature.EffectiveToughness())
	}
}

func TestAuraKeywordGrant(t *testing.T) {
	player := &Player{
		ID:          0,
		LifeTotal:   20,
		Hand:        []*Card{},
		Battlefield: []*Card{},
		Graveyard:   []*Card{},
		Turn:        &Turn{},
		InputChan:   make(chan PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
	}

	creature := makeCreature(1, "Grizzly Bears", 2, 2, player)
	player.Battlefield = append(player.Battlefield, creature)

	aura := makeAura(2, "Flight", player, []domain.ParsedAbility{
		{
			Type:     "Static",
			Keywords: []string{"Flying"},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})

	creature.Attachments = append(creature.Attachments, aura)
	aura.AttachedTo = creature

	if !creature.HasKeyword(effects.KeywordFlying) {
		t.Error("expected creature to have Flying from aura")
	}
}

func TestAuraLandwalkGrant(t *testing.T) {
	player := &Player{
		ID:          0,
		LifeTotal:   20,
		Hand:        []*Card{},
		Battlefield: []*Card{},
		Graveyard:   []*Card{},
		Turn:        &Turn{},
		InputChan:   make(chan PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
	}

	creature := makeCreature(1, "Grizzly Bears", 2, 2, player)
	player.Battlefield = append(player.Battlefield, creature)

	aura := makeAura(2, "Burrowing", player, []domain.ParsedAbility{
		{
			Type:     "Static",
			Keywords: []string{"Landwalk"},
			Effect: &domain.ParsedEffect{
				Modifier: "mountainwalk",
			},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})

	creature.Attachments = append(creature.Attachments, aura)
	aura.AttachedTo = creature

	if creature.LandwalkType() != "Mountain" {
		t.Errorf("expected landwalk type Mountain, got %q", creature.LandwalkType())
	}
}

func TestAuraDetachOnCreatureDeath(t *testing.T) {
	player := &Player{
		ID:          0,
		LifeTotal:   20,
		Hand:        []*Card{},
		Battlefield: []*Card{},
		Graveyard:   []*Card{},
		Turn:        &Turn{},
		InputChan:   make(chan PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
	}

	creature := makeCreature(1, "Grizzly Bears", 2, 2, player)
	player.Battlefield = append(player.Battlefield, creature)

	aura := makeAura(2, "Holy Strength", player, []domain.ParsedAbility{
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     1,
				ToughnessBoost: 2,
			},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})
	aura.CurrentZone = ZoneBattlefield
	player.Battlefield = append(player.Battlefield, aura)

	creature.Attachments = append(creature.Attachments, aura)
	aura.AttachedTo = creature

	creature.DamageTaken = 4
	game := NewGame([]*Player{player})
	game.CleanupDeadCreatures()

	if len(player.Graveyard) != 2 {
		t.Errorf("expected 2 cards in graveyard (creature + aura), got %d", len(player.Graveyard))
	}
	if aura.AttachedTo != nil {
		t.Error("expected aura.AttachedTo to be nil after creature death")
	}
	if len(creature.Attachments) != 0 {
		t.Error("expected creature.Attachments to be empty after death")
	}
}

func TestAuraDetachOnCreatureZoneChange(t *testing.T) {
	player := &Player{
		ID:          0,
		LifeTotal:   20,
		Hand:        []*Card{},
		Battlefield: []*Card{},
		Graveyard:   []*Card{},
		Exile:       []*Card{},
		Turn:        &Turn{},
		InputChan:   make(chan PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
	}

	creature := makeCreature(1, "Grizzly Bears", 2, 2, player)
	player.Battlefield = append(player.Battlefield, creature)

	aura := makeAura(2, "Holy Strength", player, []domain.ParsedAbility{
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     1,
				ToughnessBoost: 2,
			},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})
	aura.CurrentZone = ZoneBattlefield
	player.Battlefield = append(player.Battlefield, aura)

	creature.Attachments = append(creature.Attachments, aura)
	aura.AttachedTo = creature

	player.MoveTo(creature, ZoneExile)

	if aura.CurrentZone != ZoneGraveyard {
		t.Errorf("expected aura in graveyard, got zone %d", aura.CurrentZone)
	}
	if aura.AttachedTo != nil {
		t.Error("expected aura.AttachedTo to be nil")
	}
}

func TestAuraResolveAttaches(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	player := players[0]

	creature := makeCreature(100, "Grizzly Bears", 2, 2, player)
	creature.CurrentZone = ZoneHand
	player.Hand = append(player.Hand, creature)
	player.MoveTo(creature, ZoneBattlefield)

	aura := makeAura(101, "Holy Strength", player, []domain.ParsedAbility{
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     1,
				ToughnessBoost: 2,
			},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})
	player.Hand = append(player.Hand, aura)

	item := &StackItem{
		Events: nil,
		Player: player,
		Card:   aura,
		Target: creature,
	}

	if err := game.Resolve(item); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if aura.AttachedTo != creature {
		t.Error("expected aura attached to creature")
	}
	if len(creature.Attachments) != 1 || creature.Attachments[0] != aura {
		t.Error("expected creature to have aura in attachments")
	}
	if aura.CurrentZone != ZoneBattlefield {
		t.Errorf("expected aura on battlefield, got zone %d", aura.CurrentZone)
	}
	if creature.EffectivePower() != 3 {
		t.Errorf("expected effective power 3 after aura, got %d", creature.EffectivePower())
	}
}

func TestAuraResolveTargetGone(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	player := players[0]

	creature := makeCreature(100, "Grizzly Bears", 2, 2, player)
	creature.CurrentZone = ZoneGraveyard
	player.Graveyard = append(player.Graveyard, creature)

	aura := makeAura(101, "Holy Strength", player, []domain.ParsedAbility{
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     1,
				ToughnessBoost: 2,
			},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})
	player.Hand = append(player.Hand, aura)

	item := &StackItem{
		Events: nil,
		Player: player,
		Card:   aura,
		Target: creature,
	}

	if err := game.Resolve(item); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if aura.CurrentZone != ZoneGraveyard {
		t.Errorf("expected aura in graveyard when target is gone, got zone %d", aura.CurrentZone)
	}
}

func TestIsAura(t *testing.T) {
	aura := &Card{
		Card: domain.Card{
			CardType: domain.CardTypeEnchantment,
			ParsedAbilities: []domain.ParsedAbility{
				{
					Type: "Static",
					TargetSpec: &domain.ParsedTargetSpec{
						Type:      targetTypeCreature,
						Condition: "enchanted",
					},
				},
			},
		},
	}
	if !aura.IsAura() {
		t.Error("expected IsAura() to be true")
	}

	nonAura := &Card{
		Card: domain.Card{
			CardType: domain.CardTypeEnchantment,
		},
	}
	if nonAura.IsAura() {
		t.Error("expected IsAura() to be false for enchantment without aura abilities")
	}

	creature := &Card{
		Card: domain.Card{
			CardType: domain.CardTypeCreature,
		},
	}
	if creature.IsAura() {
		t.Error("expected IsAura() to be false for creature")
	}
}

func TestAuraGetTargetSpec(t *testing.T) {
	aura := &Card{
		Card: domain.Card{
			CardType: domain.CardTypeEnchantment,
			ParsedAbilities: []domain.ParsedAbility{
				{
					Type:       "Static",
					TargetSpec: &domain.ParsedTargetSpec{Type: targetTypeCreature, Count: 1, Condition: "enchant"},
				},
				{
					Type: "Static",
					Effect: &domain.ParsedEffect{
						PowerBoost:     1,
						ToughnessBoost: 2,
					},
					TargetSpec: &domain.ParsedTargetSpec{Type: targetTypeCreature, Count: 1, Condition: "enchanted"},
				},
			},
		},
	}

	spec := aura.GetTargetSpec()
	if spec == nil {
		t.Fatal("expected target spec for aura")
	}
	if spec.Type != targetTypeCreature {
		t.Errorf("expected creature target type, got %s", spec.Type)
	}
}

func makeImmolation(id EntityID, owner *Player) *Card {
	return makeAura(id, "Immolation", owner, []domain.ParsedAbility{
		{
			Type:       "Static",
			TargetSpec: &domain.ParsedTargetSpec{Type: targetTypeCreature, Count: 1, Condition: "enchant"},
			RawText:    "Enchant creature",
		},
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     2,
				ToughnessBoost: -2,
			},
			TargetSpec: &domain.ParsedTargetSpec{Type: targetTypeCreature, Count: 1, Condition: "enchanted"},
			RawText:    "Enchanted creature gets +2/-2",
		},
	})
}

func TestImmolationKillsGrizzlyBears(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	player := players[0]

	bears := makeCreature(100, "Grizzly Bears", 2, 2, player)
	bears.CurrentZone = ZoneHand
	player.Hand = append(player.Hand, bears)
	player.MoveTo(bears, ZoneBattlefield)

	immolation := makeImmolation(101, player)
	player.Hand = append(player.Hand, immolation)

	item := &StackItem{
		Player: player,
		Card:   immolation,
		Target: bears,
	}

	if err := game.Resolve(item); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Immolation gives +2/-2, making Grizzly Bears 4/0
	if bears.EffectiveToughness() != 0 {
		t.Errorf("expected bears effective toughness 0, got %d", bears.EffectiveToughness())
	}

	// State-based actions should kill the 0-toughness creature
	game.CleanupDeadCreatures()

	if bears.CurrentZone != ZoneGraveyard {
		t.Errorf("expected bears in graveyard, got zone %d", bears.CurrentZone)
	}
	if immolation.CurrentZone != ZoneGraveyard {
		t.Errorf("expected immolation in graveyard, got zone %d", immolation.CurrentZone)
	}
	if immolation.AttachedTo != nil {
		t.Error("expected immolation.AttachedTo to be nil after creature death")
	}
}

func TestImmolationOnWarMammoth(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	player := players[0]

	mammoth := makeCreature(100, "War Mammoth", 3, 3, player)
	mammoth.CurrentZone = ZoneHand
	player.Hand = append(player.Hand, mammoth)
	player.MoveTo(mammoth, ZoneBattlefield)

	immolation := makeImmolation(101, player)
	player.Hand = append(player.Hand, immolation)

	item := &StackItem{
		Player: player,
		Card:   immolation,
		Target: mammoth,
	}

	if err := game.Resolve(item); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Immolation gives +2/-2, making War Mammoth 5/1
	if mammoth.EffectivePower() != 5 {
		t.Errorf("expected mammoth effective power 5, got %d", mammoth.EffectivePower())
	}
	if mammoth.EffectiveToughness() != 1 {
		t.Errorf("expected mammoth effective toughness 1, got %d", mammoth.EffectiveToughness())
	}

	// War Mammoth should survive state-based actions
	game.CleanupDeadCreatures()

	if mammoth.CurrentZone != ZoneBattlefield {
		t.Errorf("expected mammoth on battlefield, got zone %d", mammoth.CurrentZone)
	}
	if immolation.CurrentZone != ZoneBattlefield {
		t.Errorf("expected immolation on battlefield, got zone %d", immolation.CurrentZone)
	}
	if immolation.AttachedTo != mammoth {
		t.Error("expected immolation still attached to mammoth")
	}
}

func TestCantCastAuraWithoutCreatures(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	player := players[0]
	player.Turn.Phase = PhaseMain1

	aura := makeAura(101, "Holy Strength", player, []domain.ParsedAbility{
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     1,
				ToughnessBoost: 2,
			},
			TargetSpec: &domain.ParsedTargetSpec{
				Type:      targetTypeCreature,
				Count:     1,
				Condition: "enchanted",
			},
		},
	})
	aura.ManaCost = "{W}"
	player.Hand = append(player.Hand, aura)
	player.ManaPool = ManaPool{{'W'}}

	if game.CanCast(player, aura) {
		t.Error("should not be able to cast aura when no creatures on battlefield")
	}
}
