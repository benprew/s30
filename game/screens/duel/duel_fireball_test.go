package duel

import (
	"slices"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestFireballChoosesTargetsBeforeAffordableX(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mountains  int
		maxX       int
		maxTargets int
		cancel     bool
	}{
		{name: "two targets leave four for X", mountains: 6, maxX: 4, maxTargets: 3},
		{name: "two targets use all available mana", mountains: 2, maxX: 0, maxTargets: 2},
		{name: "cancel after selecting targets", mountains: 6, maxX: 4, maxTargets: 3, cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("You"), mage.NewBasePlayer("Enemy")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			create := func(name string) *mage.Card {
				t.Helper()
				card, err := mage.CreateCard(name)
				if err != nil {
					t.Fatal(err)
				}
				return card
			}
			var lands []*mage.Permanent
			for range tc.mountains {
				lands = append(lands, g.PutOnBattlefield(create("Mountain"), a.PlayerID()))
			}
			bear := g.PutOnBattlefield(create("Grizzly Bears"), b.PlayerID())
			spell := create("Fireball")
			a.AddToHand(spell)

			human, responses := newTestHuman()
			s := &DuelScreen{
				human:       human,
				lastMsg:     &interactive.GameMsg{State: interactive.SnapshotGameState(g, 0)},
				cardActions: make(map[uuid.UUID][]interactive.ActionOption),
				doneBtn:     [3]*ebiten.Image{ebiten.NewImage(10, 10)},
			}
			for _, action := range interactive.GetAvailableActions(g, a.PlayerID()) {
				if action.CardID == spell.ID() {
					s.cardActions[spell.ID()] = append(s.cardActions[spell.ID()], action)
				}
			}
			s.performCardAction(spell.ID(), spell.Name())
			if s.isChoosingX() || s.targetingCardID != spell.ID() {
				t.Fatal("Fireball must select targets first")
			}
			if minTargets, maxTargets := s.targetBounds(); minTargets != 1 || maxTargets != tc.maxTargets {
				t.Fatalf("target bounds = %d–%d, want 1–%d", minTargets, maxTargets, tc.maxTargets)
			}
			targets := []uuid.UUID{bear.ID(), b.PlayerID()}
			for _, target := range targets {
				s.selectTarget(target)
			}
			if tc.maxTargets == 2 {
				s.selectTarget(a.PlayerID())
				if !slices.Equal(s.selectedTargetIDs, targets) {
					t.Fatal("the third target costs more than the available mana")
				}
			}
			s.updateTargetingMouse(duelBoardX+2, duelMsgY, true, 1024)
			if !s.isChoosingX() || s.targetingCardID != uuid.Nil() || s.xMaxValue != tc.maxX {
				t.Fatalf("Done must open the X chooser with max X=%d, got %d", tc.maxX, s.xMaxValue)
			}
			select {
			case <-responses:
				t.Fatal("cast action sent before X confirmation")
			default:
			}
			for _, land := range lands {
				if land.Tapped {
					t.Fatal("a land tapped before X confirmation")
				}
			}
			if tc.cancel {
				s.handleEscape()
				if s.isChoosingX() {
					t.Fatal("Escape must close the X chooser")
				}
				select {
				case <-responses:
					t.Fatal("cancel sent a cast action")
				default:
				}
				return
			}
			s.selectXValue(tc.maxX)
			select {
			case action := <-responses:
				if action.CardID != spell.ID() || action.XValue != tc.maxX || !slices.Equal(action.Targets, targets) {
					t.Fatalf("unexpected cast action: %+v", action)
				}
				if err := g.CastSpellByID(a.PlayerID(), action.CardID, action.Targets, action.XValue); err != nil {
					t.Fatal(err)
				}
			default:
				t.Fatal("missing cast action")
			}
			for _, land := range lands {
				if !land.Tapped {
					t.Fatal("the cast must use all available mana")
				}
			}
			g.ResolveStack()
			if damage := tc.maxX / len(targets); b.Life() != 20-damage || (g.FindPermanent(bear.ID()) == nil) != (damage >= 2) {
				t.Fatalf("want %d damage per target, opponent life is %d", damage, b.Life())
			}
		})
	}
}
