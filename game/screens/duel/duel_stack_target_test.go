package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
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

func newStackTargetScreen(count int) (*DuelScreen, interactive.ActionOption, chan interactive.PriorityAction) {
	human, responses := newTestHuman()
	action := interactive.ActionOption{
		Type: interactive.ActionCastSpell, CardID: uuid.New(),
		CardName: "Test action", NeedsTarget: true, TargetType: mage.TargetSpellOnStack(),
	}
	state := &interactive.GameState{}
	for i := range count {
		id := uuid.New()
		action.ValidTargets = append(action.ValidTargets, id)
		controller := "You"
		if i%2 == 1 {
			controller = "Enemy"
		}
		state.StackItems = append(state.StackItems, interactive.StackItemState{
			ID: id.String(), Name: "Stack spell", Controller: controller,
		})
	}
	s := &DuelScreen{human: human, lastMsg: &interactive.GameMsg{State: state}}
	return s, action, responses
}

func TestStackTargetChooserSelectsLegalSpell(t *testing.T) {
	s, action, responses := newStackTargetScreen(2)
	s.lastMsg.State.StackItems = append(s.lastMsg.State.StackItems,
		interactive.StackItemState{ID: uuid.New().String(), Name: "Invalid target"})
	s.enterTargetingMode(action.CardID, action.CardName, []interactive.ActionOption{action})
	if !s.isChoosingStackTarget() {
		t.Fatal("expected the stack target chooser")
	}
	cards, pages := s.stackTargetLayout(1024, 768)
	if len(cards) != 2 || pages != 1 {
		t.Fatalf("layout has %d cards and %d pages", len(cards), pages)
	}
	want, err := uuid.Parse(cards[1].Item.ID)
	if err != nil {
		t.Fatal(err)
	}
	point := cards[1].Rect.Min.Add(cards[1].Rect.Size().Div(2))
	s.updateStackTargetMouse(point.X, point.Y, true, 1024, 768)
	select {
	case got := <-responses:
		if len(got.Targets) != 1 || got.Targets[0] != want {
			t.Fatalf("targets = %v, want [%v]", got.Targets, want)
		}
	default:
		t.Fatal("expected the selected action")
	}
	if s.isChoosingStackTarget() || s.targetingCardID != uuid.Nil() {
		t.Fatal("chooser should close after selection")
	}
}

func TestStackTargetChooserPagesAndCancels(t *testing.T) {
	s, action, responses := newStackTargetScreen(50)
	s.enterTargetingMode(action.CardID, action.CardName, []interactive.ActionOption{action})
	s.xChosenValue = 3
	first, pages := s.stackTargetLayout(1024, 768)
	if pages < 2 {
		t.Fatal("expected multiple pages")
	}
	_, next := stackTargetPageBounds(1024, 768)
	s.updateStackTargetMouse(next.Min.X+1, next.Min.Y+1, true, 1024, 768)
	second, _ := s.stackTargetLayout(1024, 768)
	if len(second) == 0 || second[0].Item.ID == first[0].Item.ID {
		t.Fatal("next page should show different targets")
	}
	s.handleEscape()
	if s.isChoosingStackTarget() || s.targetingCardID != uuid.Nil() {
		t.Fatal("Escape should cancel stack targeting")
	}
	if s.xChosenValue != 0 {
		t.Fatal("cancel should clear the chosen X value")
	}
	select {
	case got := <-responses:
		t.Fatalf("cancel sent an action: %v", got)
	default:
	}
}

func TestAutoTargetStackSpellUsesLegalTargetsWithoutCardNames(t *testing.T) {
	s := &DuelScreen{}
	id := uuid.New()
	opt := interactive.ActionOption{TargetType: mage.TargetSpellOnStack(), ValidTargets: []uuid.UUID{id}}
	if got, ok := s.autoTargetStackSpell(opt); !ok || got != id {
		t.Fatalf("auto target = %v, %v", got, ok)
	}
	opt.ValidTargets = append(opt.ValidTargets, uuid.New())
	if _, ok := s.autoTargetStackSpell(opt); ok {
		t.Fatal("multiple legal targets require a choice")
	}
	opt.ValidTargets = nil
	if _, ok := s.autoTargetStackSpell(opt); ok {
		t.Fatal("no legal targets must not produce an automatic target")
	}
	opt.TargetType = mage.TargetCreature()
	opt.ValidTargets = []uuid.UUID{id}
	if _, ok := s.autoTargetStackSpell(opt); ok {
		t.Fatal("non-stack targets must use their existing selection flow")
	}
}

func TestStackTargetActionPaths(t *testing.T) {
	for _, targets := range []struct {
		name       string
		start, end int
	}{
		{"one legal spell you control", 0, 1},
		{"one legal opponent spell", 1, 2},
		{"two legal spells", 0, 2},
	} {
		for _, path := range []string{"cast", "activate", "choose ability", "choose X"} {
			t.Run(path+"/"+targets.name, func(t *testing.T) {
				s, action, responses := newStackTargetScreen(2)
				action.ValidTargets = action.ValidTargets[targets.start:targets.end]
				xValue := 0
				switch path {
				case "cast":
					s.cardActions = map[uuid.UUID][]interactive.ActionOption{action.CardID: {action}}
					s.performCardAction(action.CardID, action.CardName)
				case "activate", "choose ability":
					action.Type = interactive.ActionActivateAbility
					action.PermanentID = uuid.New()
					action.CardID = uuid.Nil()
					action.AbilityIndex = 2
					if path == "activate" {
						s.cardActions = map[uuid.UUID][]interactive.ActionOption{action.PermanentID: {action}}
						s.performCardAction(action.PermanentID, action.CardName)
					} else {
						s.abilityChoosingActions = []interactive.ActionOption{action}
						s.selectAbility(0)
					}
				case "choose X":
					action.NeedsX = true
					s.xChoosingActions = []interactive.ActionOption{action}
					xValue = 3
					s.selectXValue(xValue)
				}

				want := action.ValidTargets[0]
				if len(action.ValidTargets) > 1 {
					if !s.isChoosingStackTarget() {
						t.Fatal("multiple legal targets should open the chooser")
					}
					cards, _ := s.stackTargetLayout(1024, 768)
					if len(cards) != len(action.ValidTargets) {
						t.Fatalf("chooser has %d cards, want %d", len(cards), len(action.ValidTargets))
					}
					want = cards[0].ID
					s.updateStackTargetMouse(cards[0].Rect.Min.X+1, cards[0].Rect.Min.Y+1, true, 1024, 768)
				}
				if s.targetingCardID != uuid.Nil() {
					t.Fatal("target selection should be complete")
				}
				select {
				case got := <-responses:
					if got.Type != action.Type || got.CardID != action.CardID ||
						got.PermanentID != action.PermanentID || got.AbilityIndex != action.AbilityIndex {
						t.Fatalf("unexpected action: %+v", got)
					}
					if len(got.Targets) != 1 || got.Targets[0] != want {
						t.Fatalf("targets = %v, want [%v]", got.Targets, want)
					}
					if got.XValue != xValue {
						t.Fatalf("X = %d, want %d", got.XValue, xValue)
					}
				default:
					t.Fatal("expected an action with a stack target")
				}
			})
		}
	}
}
