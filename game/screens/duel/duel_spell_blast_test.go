package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestSpellBlastChoosesStackTargetBeforeX(t *testing.T) {
	for _, tc := range []struct {
		name string
		x    int
		idx  int
	}{
		{name: "zero", x: 0, idx: 0},
		{name: "fixed", x: 3, idx: 1},
		{name: "X spell", x: 5, idx: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("You"), mage.NewBasePlayer("Enemy")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			b.ManaPool().Add(core.Red, 30)
			var targets []*mage.Card
			for _, cost := range []string{"{0}", "{3}", "{X}{R}", "{20}"} {
				card := mage.NewInstant("Target", cost)
				b.AddToHand(card)
				if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 4); err != nil {
					t.Fatal(err)
				}
				targets = append(targets, card)
			}
			spell, err := mage.CreateCard("Spell Blast")
			if err != nil {
				t.Fatal(err)
			}
			a.AddToHand(spell)
			a.ManaPool().Add(core.Blue, 6)
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
			if s.isChoosingX() || !s.isChoosingStackTarget() {
				t.Fatal("Spell Blast must open the stack target chooser")
			}
			cards, _ := s.stackTargetLayout(1024, 768)
			if len(cards) != 3 {
				t.Fatalf("stack chooser has %d targets, want three affordable spells", len(cards))
			}
			for _, card := range cards {
				if card.ID == targets[tc.idx].ID() {
					s.updateStackTargetMouse(card.Rect.Min.X+1, card.Rect.Min.Y+1, true, 1024, 768)
					break
				}
			}
			select {
			case action := <-responses:
				if action.XValue != tc.x || len(action.Targets) != 1 || action.Targets[0] != targets[tc.idx].ID() {
					t.Fatalf("wrong Spell Blast action: %+v", action)
				}
				if err := g.CastSpellByID(a.PlayerID(), action.CardID, action.Targets, action.XValue); err != nil {
					t.Fatal(err)
				}
				g.ResolveStackObject(g.Stack().Pop())
				if g.FindStackObject(action.Targets[0]) != nil || len(g.StackObjects()) != 3 {
					t.Fatal("Spell Blast did not counter the selected spell")
				}
			default:
				t.Fatal("stack target selection did not submit Spell Blast")
			}
		})
	}
}
