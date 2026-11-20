package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestDetonateChoosesTargetBeforeX(t *testing.T) {
	for _, tc := range []struct {
		name string
		x    int
	}{
		{name: "Ornithopter", x: 0},
		{name: "Jayemdae Tome", x: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("You"), mage.NewBasePlayer("Enemy")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			for range 6 {
				mountain, err := mage.CreateCard("Mountain")
				if err != nil {
					t.Fatal(err)
				}
				g.PutOnBattlefield(mountain, a.PlayerID())
			}
			targets := make(map[string]uuid.UUID)
			for _, name := range []string{"Ornithopter", "Jayemdae Tome", "Colossus of Sardia"} {
				card, err := mage.CreateCard(name)
				if err != nil {
					t.Fatal(err)
				}
				targets[name] = g.PutOnBattlefield(card, b.PlayerID()).ID()
			}
			spell, err := mage.CreateCard("Detonate")
			if err != nil {
				t.Fatal(err)
			}
			a.AddToHand(spell)
			var actions []interactive.ActionOption
			for _, action := range interactive.GetAvailableActions(g, a.PlayerID()) {
				if action.CardID == spell.ID() {
					actions = append(actions, action)
				}
			}
			human, responses := newTestHuman()
			s := &DuelScreen{
				human:       human,
				lastMsg:     &interactive.GameMsg{State: interactive.SnapshotGameState(g, 0)},
				cardActions: map[uuid.UUID][]interactive.ActionOption{spell.ID(): actions},
			}

			s.performCardAction(spell.ID(), spell.Name())
			if s.isChoosingX() {
				t.Fatal("Detonate must select a target before X")
			}
			if s.targetingCardID != spell.ID() {
				t.Fatal("Detonate must enter targeting mode")
			}
			for _, name := range []string{"Ornithopter", "Jayemdae Tome"} {
				if _, ok := s.targetingActions[targets[name]]; !ok {
					t.Fatalf("affordable target %s is missing", name)
				}
			}
			if _, ok := s.targetingActions[targets["Colossus of Sardia"]]; ok {
				t.Fatal("unaffordable artifact must not be selectable")
			}
			select {
			case got := <-responses:
				t.Fatalf("sent action before target selection: %+v", got)
			default:
			}

			s.selectTarget(targets[tc.name])
			if !s.submitTargetingAction() {
				t.Fatal("target selection must submit Detonate")
			}
			select {
			case got := <-responses:
				if got.CardID != spell.ID() || got.XValue != tc.x || len(got.Targets) != 1 || got.Targets[0] != targets[tc.name] {
					t.Fatalf("unexpected cast action: %+v", got)
				}
				if err := g.CastSpellByID(a.PlayerID(), got.CardID, got.Targets, got.XValue); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if g.FindPermanent(targets[tc.name]) != nil || b.Life() != 20-tc.x {
					t.Fatalf("Detonate did not destroy its target and deal %d damage", tc.x)
				}
			default:
				t.Fatal("missing cast action")
			}
		})
	}
}

func TestDetonateTargetingFiltersArtifactsByChosenX(t *testing.T) {
	ornithopterID := uuid.New()
	tomeID := uuid.New()

	state := &interactive.GameState{
		Opponent: interactive.PlayerState{
			Name: "Enemy",
			Battlefield: []interactive.PermanentState{
				{ID: ornithopterID, Name: "Ornithopter"},
				{ID: tomeID, Name: "Jayemdae Tome"},
			},
		},
	}

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       uuid.New(),
		CardName:     "Detonate",
		NeedsX:       true,
		MaxXValue:    6,
		NeedsTarget:  true,
		TargetType:   mage.TargetArtifactWithManaValueX(),
		ValidTargets: []uuid.UUID{ornithopterID, tomeID},
		XTargetOptions: []mage.SpellTargetOption{
			{XValue: 0, Candidates: [][]uuid.UUID{{ornithopterID}}},
			{XValue: 4, Candidates: [][]uuid.UUID{{tomeID}}},
		},
	}

	human, fromTUI := newTestHuman()
	s := &DuelScreen{human: human, lastMsg: &interactive.GameMsg{State: state}}

	s.enterXChoosingMode([]interactive.ActionOption{action})
	s.selectXValue(4)
	if _, ok := s.targetingActions[tomeID]; !ok {
		t.Fatal("expected Jayemdae Tome (CMC 4) to be targetable when X = 4")
	}
	if _, ok := s.targetingActions[ornithopterID]; ok {
		t.Fatal("did not expect Ornithopter (CMC 0) to be targetable when X = 4")
	}

	s.selectTarget(tomeID)
	if !s.submitTargetingAction() {
		t.Fatal("expected submitTargetingAction to succeed")
	}
	select {
	case pa := <-fromTUI:
		if pa.XValue != 4 {
			t.Fatalf("expected XValue 4, got %d", pa.XValue)
		}
		if len(pa.Targets) != 1 || pa.Targets[0] != tomeID {
			t.Fatalf("expected target %v, got %v", tomeID, pa.Targets)
		}
	default:
		t.Fatal("expected priority action to be sent")
	}

	s.exitTargetingMode()
	s.enterXChoosingMode([]interactive.ActionOption{action})
	s.selectXValue(0)
	if _, ok := s.targetingActions[ornithopterID]; !ok {
		t.Fatal("expected Ornithopter (CMC 0) to be targetable when X = 0")
	}
	if _, ok := s.targetingActions[tomeID]; ok {
		t.Fatal("did not expect Jayemdae Tome (CMC 4) to be targetable when X = 0")
	}

	s.exitTargetingMode()
	s.enterXChoosingMode([]interactive.ActionOption{action})
	s.selectXValue(2)
	if len(s.targetingActions) != 0 {
		t.Fatalf("expected no valid targets for X = 2, got %d", len(s.targetingActions))
	}
}

func TestStandardXSpellRetainsValidTargets(t *testing.T) {
	creatureID := uuid.New()
	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       uuid.New(),
		CardName:     "Fireball",
		NeedsX:       true,
		MaxXValue:    5,
		NeedsTarget:  true,
		TargetType:   mage.TargetCreature(),
		ValidTargets: []uuid.UUID{creatureID},
	}

	s := &DuelScreen{
		cardActions: map[uuid.UUID][]interactive.ActionOption{action.CardID: {action}},
	}
	s.performCardAction(action.CardID, action.CardName)
	if !s.isChoosingX() {
		t.Fatal("a standard X spell must ask for X")
	}
	s.selectXValue(5)
	if _, ok := s.targetingActions[creatureID]; !ok {
		t.Fatal("expected creature to be targetable for standard X spell")
	}
}

func TestXTargetSelectionUsesEngineValues(t *testing.T) {
	human, responses := newTestHuman()
	targetID := uuid.New()
	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       uuid.New(),
		CardName:     "Test removal",
		NeedsX:       true,
		MaxXValue:    6,
		NeedsTarget:  true,
		TargetType:   mage.TargetCreature(),
		ValidTargets: []uuid.UUID{targetID},
		XTargetOptions: []mage.SpellTargetOption{
			{XValue: 2, Candidates: [][]uuid.UUID{{targetID}}},
		},
	}
	s := &DuelScreen{
		human:       human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{action.CardID: {action}},
	}
	s.performCardAction(action.CardID, action.CardName)
	if s.isChoosingX() || s.targetingCardID != action.CardID {
		t.Fatal("engine target options must allow target selection without card data")
	}
	s.selectTarget(targetID)
	if !s.submitTargetingAction() {
		t.Fatal("target selection must submit the spell")
	}
	select {
	case got := <-responses:
		if got.XValue != 2 {
			t.Fatalf("X = %d, want the engine value 2", got.XValue)
		}
	default:
		t.Fatal("missing cast action")
	}
}

func TestXTargetSelectionPromptsForAmbiguousX(t *testing.T) {
	human, responses := newTestHuman()
	firstID, secondID := uuid.New(), uuid.New()
	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       uuid.New(),
		CardName:     "Test removal",
		NeedsX:       true,
		MaxXValue:    4,
		NeedsTarget:  true,
		TargetType:   mage.TargetCreature(),
		ValidTargets: []uuid.UUID{firstID, secondID},
		XTargetOptions: []mage.SpellTargetOption{
			{XValue: 2, Candidates: [][]uuid.UUID{{firstID}}},
			{XValue: 3, Candidates: [][]uuid.UUID{{firstID}}},
			{XValue: 4, Candidates: [][]uuid.UUID{{secondID}}},
		},
	}
	s := &DuelScreen{
		human:       human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{action.CardID: {action}},
	}
	s.performCardAction(action.CardID, action.CardName)
	if !s.isChoosingX() {
		t.Fatal("multiple X values for one target require an X choice")
	}
	s.selectXValue(3)
	if len(s.targetingActions) != 1 {
		t.Fatalf("targets = %v, want only the target legal for X=3", s.targetingActions)
	}
	if _, ok := s.targetingActions[firstID]; !ok {
		t.Fatal("the engine target for X=3 must be selectable")
	}
	if len(action.ValidTargets) != 2 || len(s.cardActions[action.CardID][0].ValidTargets) != 2 {
		t.Fatal("X selection changed the available action")
	}
	s.selectTarget(firstID)
	if !s.submitTargetingAction() {
		t.Fatal("target selection must submit the spell")
	}
	select {
	case got := <-responses:
		if got.XValue != 3 {
			t.Fatalf("X = %d, want the chosen value 3", got.XValue)
		}
	default:
		t.Fatal("missing cast action")
	}
}
