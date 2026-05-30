package duel

import (
	"image"
	"testing"
	"time"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func newLiftScreen() *DuelScreen {
	return &DuelScreen{attackerLifts: map[uuid.UUID]*attackerLiftAnimation{}}
}

func TestAttackerLiftSelectAnimatesUp(t *testing.T) {
	base := time.Now()
	s := newLiftScreen()
	id := uuid.New()

	s.startAttackerLift(id, -attackerLiftOffset, base)

	if got := s.attackerLiftY(id, base); got != 0 {
		t.Fatalf("expected 0 at start, got %f", got)
	}

	mid := s.attackerLiftY(id, base.Add(attackerLiftDuration/2))
	if mid >= 0 || mid <= -attackerLiftOffset {
		t.Fatalf("expected mid between -%f and 0, got %f", attackerLiftOffset, mid)
	}

	end := s.attackerLiftY(id, base.Add(attackerLiftDuration))
	if end != -attackerLiftOffset {
		t.Fatalf("expected -%f at end, got %f", attackerLiftOffset, end)
	}
}

func TestAttackerLiftDeselectAnimatesDown(t *testing.T) {
	base := time.Now()
	s := newLiftScreen()
	id := uuid.New()

	s.startAttackerLift(id, -attackerLiftOffset, base.Add(-time.Hour))
	if settled := s.attackerLiftY(id, base); settled != -attackerLiftOffset {
		t.Fatalf("expected settled at -%f, got %f", attackerLiftOffset, settled)
	}

	s.startAttackerLift(id, 0, base)

	if got := s.attackerLiftY(id, base); got != -attackerLiftOffset {
		t.Fatalf("expected -%f at start of deselect, got %f", attackerLiftOffset, got)
	}

	mid := s.attackerLiftY(id, base.Add(attackerLiftDuration/2))
	if mid <= -attackerLiftOffset || mid >= 0 {
		t.Fatalf("expected mid between -%f and 0, got %f", attackerLiftOffset, mid)
	}

	end := s.attackerLiftY(id, base.Add(attackerLiftDuration))
	if end != 0 {
		t.Fatalf("expected 0 at end, got %f", end)
	}
}

func TestAttackerLiftMidAnimationToggleIsSmooth(t *testing.T) {
	base := time.Now()
	s := newLiftScreen()
	id := uuid.New()

	s.startAttackerLift(id, -attackerLiftOffset, base)

	midTime := base.Add(attackerLiftDuration / 2)
	offsetAtMid := s.attackerLiftY(id, midTime)

	s.startAttackerLift(id, 0, midTime)

	if got := s.attackerLifts[id].from; got != offsetAtMid {
		t.Fatalf("expected new from to equal offset at mid %f, got %f", offsetAtMid, got)
	}

	if got := s.attackerLiftY(id, midTime); got != offsetAtMid {
		t.Fatalf("expected no jump at toggle, got %f vs %f", got, offsetAtMid)
	}
}

func TestAttackerLiftUnknownIDIsZero(t *testing.T) {
	s := newLiftScreen()
	if got := s.attackerLiftY(uuid.New(), time.Now()); got != 0 {
		t.Fatalf("expected 0 for unknown id, got %f", got)
	}
}

func TestAttackerLiftStaysUpAfterAttackersAreDeclared(t *testing.T) {
	base := time.Now()
	s, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true
	s.startAttackerLift(creature.ID, -attackerLiftOffset, base.Add(-time.Hour))
	s.lastMsg.State.Step = stepDeclareBlockers
	s.lastMsg.State.You.Battlefield[0].Attacking = true

	s.refreshCardActions()

	if got := s.attackerLifts[creature.ID].to; got != -attackerLiftOffset {
		t.Fatalf("expected declared attacker to stay lifted at -%f, got %f", attackerLiftOffset, got)
	}
	if len(s.pendingAttackers) != 0 {
		t.Fatal("expected pending attackers to clear after declaration")
	}
}

func TestSubmittingAttackersDoesNotLowerLift(t *testing.T) {
	base := time.Now()
	s, creature := setupAttackerTest()
	fromTUI := make(chan interactive.PriorityAction, 1)
	s.human = interactive.NewHumanPlayerWithChannels(
		"You",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)

	s.pendingAttackers[creature.ID] = true
	s.startAttackerLift(creature.ID, -attackerLiftOffset, base.Add(-time.Hour))

	s.submitPendingAndPass()

	if got := s.attackerLifts[creature.ID].to; got != -attackerLiftOffset {
		t.Fatalf("expected submitted attacker to stay lifted at -%f, got %f", attackerLiftOffset, got)
	}

	select {
	case action := <-fromTUI:
		if action.Type != interactive.ActionSelectAttackers {
			t.Fatalf("expected select attackers action, got %s", action.Type)
		}
	default:
		t.Fatal("expected submitted attacker action")
	}
}

func TestAttackerLiftLowersAfterCombat(t *testing.T) {
	base := time.Now()
	s, creature := setupAttackerTest()

	s.startAttackerLift(creature.ID, -attackerLiftOffset, base.Add(-time.Hour))
	s.lastMsg.State.Step = "Postcombat Main"
	s.lastMsg.State.You.Battlefield[0].Attacking = false

	s.refreshCardActions()

	if got := s.attackerLifts[creature.ID].to; got != 0 {
		t.Fatalf("expected attacker lift to lower after combat, got %f", got)
	}
}

func TestFieldCardPositionIgnoresStaleStoredPosition(t *testing.T) {
	s, creature := setupAttackerTest()
	basePos := s.getFieldCardPos(creature, s.self, 0, 1, permRowCreature)
	liftedPos := image.Pt(basePos.X, basePos.Y-int(attackerLiftOffset))
	s.cardPositions[creature.ID] = liftedPos

	got := s.getFieldCardPos(creature, s.self, 0, 1, permRowCreature)

	if got != basePos {
		t.Fatalf("expected fixed layout to use base position %v, got %v", basePos, got)
	}
}

func TestClickingLiftedSelectedAttackerDeselects(t *testing.T) {
	base := time.Now()
	s, creature := setupAttackerTest()
	basePos := s.getFieldCardPos(creature, s.self, 0, 1, permRowCreature)
	liftedPos := image.Pt(basePos.X, basePos.Y-int(attackerLiftOffset))
	s.cardPositions[creature.ID] = liftedPos
	s.pendingAttackers[creature.ID] = true
	s.startAttackerLift(creature.ID, -attackerLiftOffset, base.Add(-time.Hour))

	s.handleCardClick(liftedPos.X+fieldCardW/2, liftedPos.Y+fieldCardH-2)

	if s.pendingAttackers[creature.ID] {
		t.Fatal("expected click on lifted selected attacker to deselect it")
	}
	if got := s.attackerLifts[creature.ID].to; got != 0 {
		t.Fatalf("expected deselected attacker to lower, got lift target %f", got)
	}
}
