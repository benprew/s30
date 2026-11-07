package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func stackTargetScreen(count int) (*DuelScreen, chan interactive.PriorityAction) {
	human, responses := newTestHuman()
	action := interactive.ActionOption{
		Type: interactive.ActionCastSpell, CardID: uuid.New(),
		CardName: "Test spell", NeedsTarget: true, TargetType: mage.TargetSpellOnStack(),
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
			ID: id.String(), Name: "Lightning Bolt", Controller: controller,
		})
	}
	s := &DuelScreen{human: human, lastMsg: &interactive.GameMsg{State: state}}
	s.enterTargetingMode(action.CardID, action.CardName, []interactive.ActionOption{action})
	return s, responses
}

func TestStackTargetChooserSelectsLegalSpell(t *testing.T) {
	s, responses := stackTargetScreen(2)
	s.lastMsg.State.StackItems = append(s.lastMsg.State.StackItems,
		interactive.StackItemState{ID: uuid.New().String(), Name: "Invalid target"})
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
	s, responses := stackTargetScreen(50)
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

func TestStackTargetChooserActionPaths(t *testing.T) {
	for _, path := range []string{"cast", "ability", "X spell"} {
		t.Run(path, func(t *testing.T) {
			s, responses := stackTargetScreen(2)
			action := s.targetingAction
			s.exitTargetingMode()
			switch path {
			case "cast":
				s.cardActions = map[uuid.UUID][]interactive.ActionOption{action.CardID: {action}}
				s.performCardAction(action.CardID, action.CardName)
			case "ability":
				action.Type = interactive.ActionActivateAbility
				action.PermanentID = uuid.New()
				action.AbilityIndex = 2
				s.abilityChoosingActions = []interactive.ActionOption{action}
				s.selectAbility(0)
			case "X spell":
				action.NeedsX = true
				s.xChoosingActions = []interactive.ActionOption{action}
				s.selectXValue(3)
			}
			if !s.isChoosingStackTarget() {
				t.Fatal("multiple legal targets should open the chooser")
			}
			cards, _ := s.stackTargetLayout(1024, 768)
			s.updateStackTargetMouse(cards[0].Rect.Min.X+1, cards[0].Rect.Min.Y+1, true, 1024, 768)
			select {
			case got := <-responses:
				if got.Type != action.Type || got.CardID != action.CardID ||
					got.PermanentID != action.PermanentID || got.AbilityIndex != action.AbilityIndex ||
					len(got.Targets) != 1 || got.Targets[0] != cards[0].ID {
					t.Fatalf("unexpected action: %+v", got)
				}
				if path == "X spell" && got.XValue != 3 {
					t.Fatalf("X = %d, want 3", got.XValue)
				}
			default:
				t.Fatal("expected an action after choosing a stack target")
			}
		})
	}
}
