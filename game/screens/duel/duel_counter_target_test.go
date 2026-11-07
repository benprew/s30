package duel

import (
	"testing"
	"uuid"

	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func newTestHuman() (*interactive.HumanPlayer, chan interactive.PriorityAction) {
	fromTUI := make(chan interactive.PriorityAction, 1)
	h := interactive.NewHumanPlayerWithChannels(
		"You",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)
	return h, fromTUI
}

func TestCounterAutoTargetsSingleOpponentSpell(t *testing.T) {
	human, fromTUI := newTestHuman()
	spellID := uuid.New()
	counterID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: spellID.String(), Name: "Bolt", Controller: "Enemy"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			counterID: {{
				Type:         interactive.ActionCastSpell,
				CardID:       counterID,
				CardName:     "Counterspell",
				NeedsTarget:  true,
				TargetType:   mage.TargetSpellOnStack(),
				ValidTargets: []uuid.UUID{spellID},
			}},
		},
	}

	s.performCardAction(counterID, "Counterspell")

	if s.targetingCardID != uuid.Nil() {
		t.Fatal("should not enter targeting mode when only one opponent spell is on the stack")
	}

	select {
	case pa := <-fromTUI:
		if len(pa.Targets) != 1 || pa.Targets[0] != spellID {
			t.Fatalf("expected auto-target %v, got %v", spellID, pa.Targets)
		}
	default:
		t.Fatal("expected a cast action to be sent")
	}
}

func TestCounterPromptsWithMultipleOpponentSpells(t *testing.T) {
	human, _ := newTestHuman()
	spell1 := uuid.New()
	spell2 := uuid.New()
	counterID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: spell1.String(), Name: "Bolt", Controller: "Enemy"},
					{ID: spell2.String(), Name: "Growth", Controller: "Enemy"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			counterID: {{
				Type:         interactive.ActionCastSpell,
				CardID:       counterID,
				CardName:     "Counterspell",
				NeedsTarget:  true,
				TargetType:   mage.TargetSpellOnStack(),
				ValidTargets: []uuid.UUID{spell1, spell2},
			}},
		},
	}

	s.performCardAction(counterID, "Counterspell")

	if s.targetingCardID != counterID {
		t.Fatal("should enter targeting mode when multiple opponent spells are on the stack")
	}
}

func TestXCounterAutoTargetsSingleOpponentSpell(t *testing.T) {
	human, fromTUI := newTestHuman()
	spellID := uuid.New()
	sinkID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: spellID.String(), Name: "Bolt", Controller: "Enemy"},
				},
			},
		},
		xChoosingActions: []interactive.ActionOption{{
			Type:         interactive.ActionCastSpell,
			CardID:       sinkID,
			CardName:     "Power Sink",
			NeedsTarget:  true,
			NeedsX:       true,
			TargetType:   mage.TargetSpellOnStack(),
			ValidTargets: []uuid.UUID{spellID},
		}},
	}

	s.selectXValue(2)

	if s.targetingCardID != uuid.Nil() {
		t.Fatal("should not enter targeting mode when only one opponent spell is on the stack")
	}

	select {
	case pa := <-fromTUI:
		if len(pa.Targets) != 1 || pa.Targets[0] != spellID {
			t.Fatalf("expected auto-target %v, got %v", spellID, pa.Targets)
		}
		if pa.XValue != 2 {
			t.Fatalf("expected X=2, got %d", pa.XValue)
		}
	default:
		t.Fatal("expected a cast action to be sent")
	}
}

func TestForkAutoTargetsSingleOwnSpell(t *testing.T) {
	human, fromTUI := newTestHuman()
	boltID := uuid.New()
	forkID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				You:      interactive.PlayerState{Name: "You"},
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: boltID.String(), Name: "Lightning Bolt", Controller: "You"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			forkID: {{
				Type:         interactive.ActionCastSpell,
				CardID:       forkID,
				CardName:     "Fork",
				NeedsTarget:  true,
				TargetType:   mage.TargetSpellOnStack(),
				ValidTargets: []uuid.UUID{boltID},
			}},
		},
	}

	s.performCardAction(forkID, "Fork")

	if s.targetingCardID != uuid.Nil() {
		t.Fatal("should not enter targeting mode when only one own spell is on the stack for Fork")
	}

	select {
	case pa := <-fromTUI:
		if len(pa.Targets) != 1 || pa.Targets[0] != boltID {
			t.Fatalf("expected auto-target %v, got %v", boltID, pa.Targets)
		}
	default:
		t.Fatal("expected a cast action to be sent")
	}
}

func TestForkAutoTargetsSingleOpponentSpell(t *testing.T) {
	human, fromTUI := newTestHuman()
	boltID := uuid.New()
	forkID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				You:      interactive.PlayerState{Name: "You"},
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: boltID.String(), Name: "Lightning Bolt", Controller: "Enemy"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			forkID: {{
				Type:         interactive.ActionCastSpell,
				CardID:       forkID,
				CardName:     "Fork",
				NeedsTarget:  true,
				TargetType:   mage.TargetSpellOnStack(),
				ValidTargets: []uuid.UUID{boltID},
			}},
		},
	}

	s.performCardAction(forkID, "Fork")

	if s.targetingCardID != uuid.Nil() {
		t.Fatal("should not enter targeting mode when only one opponent spell is on the stack for Fork")
	}

	select {
	case pa := <-fromTUI:
		if len(pa.Targets) != 1 || pa.Targets[0] != boltID {
			t.Fatalf("expected auto-target %v, got %v", boltID, pa.Targets)
		}
	default:
		t.Fatal("expected a cast action to be sent")
	}
}

func TestGoblinArtisansAutoTargetsSingleOwnArtifact(t *testing.T) {
	human, fromTUI := newTestHuman()
	ornithopterID := uuid.New()
	artisansID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				You:      interactive.PlayerState{Name: "You"},
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: ornithopterID.String(), Name: "Ornithopter", Controller: "You"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			artisansID: {{
				Type:         interactive.ActionActivateAbility,
				CardID:       artisansID,
				CardName:     "Goblin Artisans",
				NeedsTarget:  true,
				TargetType:   mage.TargetOwnSpellOnStack(mage.IsArtifactCard),
				ValidTargets: []uuid.UUID{ornithopterID},
			}},
		},
	}

	s.performCardAction(artisansID, "Goblin Artisans")

	if s.targetingCardID != uuid.Nil() {
		t.Fatal("should not enter targeting mode when only one own artifact is on the stack for Goblin Artisans")
	}

	select {
	case pa := <-fromTUI:
		if len(pa.Targets) != 1 || pa.Targets[0] != ornithopterID {
			t.Fatalf("expected auto-target %v, got %v", ornithopterID, pa.Targets)
		}
	default:
		t.Fatal("expected an activate ability action to be sent")
	}
}

func TestCounterspellDoesNotAutoCounterOwnSpell(t *testing.T) {
	human, _ := newTestHuman()
	boltID := uuid.New()
	counterID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				You:      interactive.PlayerState{Name: "You"},
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: boltID.String(), Name: "Lightning Bolt", Controller: "You"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			counterID: {{
				Type:         interactive.ActionCastSpell,
				CardID:       counterID,
				CardName:     "Counterspell",
				NeedsTarget:  true,
				TargetType:   mage.TargetSpellOnStack(),
				ValidTargets: []uuid.UUID{boltID},
			}},
		},
	}

	s.performCardAction(counterID, "Counterspell")

	if s.targetingCardID != counterID {
		t.Fatal("Counterspell should not auto-counter own lone spell without explicit choice")
	}
}

func TestAbilityChooserAutoTargetsGoblinArtisans(t *testing.T) {
	human, fromTUI := newTestHuman()
	ornithopterID := uuid.New()
	artisansID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				You:      interactive.PlayerState{Name: "You"},
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: ornithopterID.String(), Name: "Ornithopter", Controller: "You"},
				},
			},
		},
		abilityChoosingActions: []interactive.ActionOption{{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  artisansID,
			CardID:       artisansID,
			CardName:     "Goblin Artisans",
			NeedsTarget:  true,
			TargetType:   mage.TargetOwnSpellOnStack(mage.IsArtifactCard),
			ValidTargets: []uuid.UUID{ornithopterID},
		}},
	}

	s.selectAbility(0)

	if s.targetingCardID != uuid.Nil() {
		t.Fatal("should not enter targeting mode when selecting Goblin Artisans in ability chooser")
	}

	select {
	case pa := <-fromTUI:
		if len(pa.Targets) != 1 || pa.Targets[0] != ornithopterID {
			t.Fatalf("expected auto-target %v, got %v", ornithopterID, pa.Targets)
		}
	default:
		t.Fatal("expected an activate ability action to be sent")
	}
}
