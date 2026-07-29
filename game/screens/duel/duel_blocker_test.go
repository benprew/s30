package duel

import (
	"image"
	"testing"
	"time"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func setupBlockerTest() (*DuelScreen, interactive.PermanentState, interactive.PermanentState) {
	blockerID := uuid.New()
	attackerID := uuid.New()

	blocker := interactive.PermanentState{
		ID:         blockerID,
		Name:       "Llanowar Elves",
		Power:      1,
		Toughness:  1,
		IsCreature: true,
	}

	attacker := interactive.PermanentState{
		ID:         attackerID,
		Name:       "Llanowar Elves",
		Power:      1,
		Toughness:  1,
		IsCreature: true,
		Attacking:  true,
	}

	state := &interactive.GameState{
		Step:         stepDeclareBlockers,
		ActivePlayer: "Opponent",
		You: interactive.PlayerState{
			ID:          uuid.New(),
			Name:        "You",
			Life:        20,
			Battlefield: []interactive.PermanentState{blocker},
		},
		Opponent: interactive.PlayerState{
			ID:          uuid.New(),
			Name:        "Opponent",
			Life:        20,
			Battlefield: []interactive.PermanentState{attacker},
		},
	}

	msg := &interactive.GameMsg{
		State:  state,
		Prompt: interactive.PromptDeclareBlockers,
		Options: []interactive.ActionOption{
			{
				Type:         interactive.ActionSelectBlockers,
				Label:        "Llanowar Elves",
				CardName:     "Llanowar Elves",
				PermanentID:  blockerID,
				ValidTargets: []uuid.UUID{attackerID},
			},
		},
	}

	s := &DuelScreen{
		lastMsg:          msg,
		self:             &duelPlayer{name: "You"},
		opponent:         &duelPlayer{name: "Opponent"},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.refreshCardActions()

	return s, blocker, attacker
}

func TestIsInDeclareBlockers(t *testing.T) {
	s, _, _ := setupBlockerTest()

	if !s.isInDeclareBlockers() {
		t.Error("expected isInDeclareBlockers to return true during declare blockers")
	}

	s.lastMsg.Prompt = interactive.PromptNone
	if s.isInDeclareBlockers() {
		t.Error("expected isInDeclareBlockers to return false outside declare blockers")
	}
}

func TestPendingBlockers_SelectBlocker(t *testing.T) {
	s, blocker, _ := setupBlockerTest()

	pos := s.getFieldCardPos(blocker, s.self, 0, 1, permRowCreature)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	if s.selectedBlocker != blocker.ID {
		t.Errorf("expected selectedBlocker to be %v, got %v", blocker.ID, s.selectedBlocker)
	}
}

func TestSelectingBlockerLiftsItForward(t *testing.T) {
	s, blocker, _ := setupBlockerTest()
	pos := s.getFieldCardPos(blocker, s.self, 0, 1, permRowCreature)

	s.handleBlockerClick(pos.X+fieldCardW/2, pos.Y+fieldCardH/2)

	if got := s.attackerLifts[blocker.ID].to; got != -attackerLiftOffset {
		t.Fatalf("expected selected blocker lift target -%f, got %f", attackerLiftOffset, got)
	}
}

func TestClickingLiftedSelectedBlockerLowersIt(t *testing.T) {
	s, blocker, _ := setupBlockerTest()
	base := time.Now()
	pos := s.getFieldCardPos(blocker, s.self, 0, 1, permRowCreature)
	liftedPos := image.Pt(pos.X, pos.Y-int(attackerLiftOffset))
	s.selectedBlocker = blocker.ID
	s.startAttackerLift(blocker.ID, -attackerLiftOffset, base.Add(-time.Hour))

	s.handleBlockerClick(liftedPos.X+fieldCardW/2, liftedPos.Y+fieldCardH-2)

	if s.selectedBlocker != uuid.Nil {
		t.Fatal("expected lifted selected blocker to be deselected")
	}
	if got := s.attackerLifts[blocker.ID].to; got != 0 {
		t.Fatalf("expected deselected blocker to lower, got lift target %f", got)
	}
}

func TestPendingBlockers_AssignBlocker(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	s.selectedBlocker = blocker.ID

	pos := s.getFieldCardPos(attacker, s.opponent, 0, 1, permRowCreature)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	assignedTo, ok := s.pendingBlockers[blocker.ID]
	if !ok {
		t.Fatal("expected blocker to be in pendingBlockers")
	}
	if assignedTo != attacker.ID {
		t.Errorf("expected blocker assigned to attacker %v, got %v", attacker.ID, assignedTo)
	}
	if s.selectedBlocker != uuid.Nil {
		t.Error("expected selectedBlocker to be cleared after assignment")
	}
	if got := s.attackerLifts[blocker.ID].to; got != -attackerLiftOffset {
		t.Fatalf("expected assigned blocker to stay lifted at -%f, got %f", attackerLiftOffset, got)
	}
}

func TestPendingBlockers_RemoveBlocker(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	s.pendingBlockers[blocker.ID] = attacker.ID

	pos := s.getFieldCardPos(blocker, s.self, 0, 1, permRowCreature)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	if _, ok := s.pendingBlockers[blocker.ID]; ok {
		t.Error("expected blocker to be removed from pendingBlockers after clicking it again")
	}
	if got := s.attackerLifts[blocker.ID].to; got != 0 {
		t.Fatalf("expected unassigned blocker to lower, got lift target %f", got)
	}
}

func TestAssignedBlockerStaysLiftedUntilCombatEnds(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()
	base := time.Now()
	s.lastMsg.Prompt = interactive.PromptNone
	s.lastMsg.State.Step = stepCombatDamage
	s.lastMsg.State.You.Battlefield[0].Blocking = attacker.ID
	s.startAttackerLift(blocker.ID, -attackerLiftOffset, base.Add(-time.Hour))

	s.refreshCardActions()

	if got := s.attackerLifts[blocker.ID].to; got != -attackerLiftOffset {
		t.Fatalf("expected blocker to stay lifted during combat at -%f, got %f", attackerLiftOffset, got)
	}

	s.lastMsg.State.Step = "Postcombat Main"
	s.lastMsg.State.You.Battlefield[0].Blocking = uuid.Nil
	s.refreshCardActions()

	if got := s.attackerLifts[blocker.ID].to; got != 0 {
		t.Fatalf("expected blocker to lower after combat, got %f", got)
	}
}

func TestCanBlockAnything_OnlyValidBlockers(t *testing.T) {
	s, blocker, _ := setupBlockerTest()

	if !s.canBlockAnything(blocker.ID) {
		t.Error("expected untapped blocker listed in options to be a valid blocker")
	}

	tappedCreatureID := uuid.New()
	if s.canBlockAnything(tappedCreatureID) {
		t.Error("expected creature not in options (e.g. tapped) to not be a valid blocker")
	}
}

func TestIsValidBlock_OnlyValidBlockers(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	if !s.isValidBlock(blocker.ID, attacker.ID) {
		t.Error("expected valid blocker to be allowed to block attacker")
	}

	tappedCreatureID := uuid.New()
	if s.isValidBlock(tappedCreatureID, attacker.ID) {
		t.Error("expected creature not in options (e.g. tapped) to not be a valid blocker")
	}
}

func TestPendingBlockers_TappedCreatureCannotBeSelected(t *testing.T) {
	s, blocker, _ := setupBlockerTest()

	tappedCreature := interactive.PermanentState{
		ID:         uuid.New(),
		Name:       "Birds of Paradise",
		Power:      0,
		Toughness:  1,
		IsCreature: true,
	}

	s.lastMsg.State.You.Battlefield = append(
		s.lastMsg.State.You.Battlefield, tappedCreature,
	)

	tappedPos := s.getFieldCardPos(tappedCreature, s.self, 1, 2, permRowCreature)
	s.handleBlockerClick(tappedPos.X+fieldCardW/2, tappedPos.Y+fieldCardH/2)

	if s.selectedBlocker == tappedCreature.ID {
		t.Error("tapped creature (not in options) should not be selectable as blocker")
	}

	validPos := s.getFieldCardPos(blocker, s.self, 0, 2, permRowCreature)
	s.handleBlockerClick(validPos.X+fieldCardW/2, validPos.Y+fieldCardH/2)

	if s.selectedBlocker != blocker.ID {
		t.Errorf("valid blocker should be selectable, got %v", s.selectedBlocker)
	}
}

func TestAIBlockerArrows(t *testing.T) {
	playerAttackerID := uuid.New()
	aiBlockerID := uuid.New()

	playerAttacker := interactive.PermanentState{
		ID:         playerAttackerID,
		Name:       "Grizzly Bears",
		Power:      2,
		Toughness:  2,
		IsCreature: true,
		Attacking:  true,
	}

	aiBlocker := interactive.PermanentState{
		ID:         aiBlockerID,
		Name:       "Llanowar Elves",
		Power:      1,
		Toughness:  1,
		IsCreature: true,
		Blocking:   playerAttackerID,
	}

	state := &interactive.GameState{
		Step:         stepDeclareBlockers,
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:          uuid.New(),
			Name:        "You",
			Life:        20,
			Battlefield: []interactive.PermanentState{playerAttacker},
		},
		Opponent: interactive.PlayerState{
			ID:          uuid.New(),
			Name:        "Opponent",
			Life:        20,
			Battlefield: []interactive.PermanentState{aiBlocker},
		},
	}

	msg := &interactive.GameMsg{
		State:  state,
		Prompt: interactive.PromptNone,
	}

	s := &DuelScreen{
		lastMsg:          msg,
		self:             &duelPlayer{name: "You"},
		opponent:         &duelPlayer{name: "Opponent"},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	arrows := s.getAIBlockerArrows()
	if len(arrows) != 1 {
		t.Fatalf("expected 1 AI blocker arrow, got %d", len(arrows))
	}
	attackerID, ok := arrows[aiBlockerID]
	if !ok {
		t.Fatal("expected AI blocker to be in arrows map")
	}
	if attackerID != playerAttackerID {
		t.Errorf("expected AI blocker to point to player attacker %v, got %v", playerAttackerID, attackerID)
	}
}

func TestAIBlockerArrows_Empty(t *testing.T) {
	s, _, _ := setupBlockerTest()
	arrows := s.getAIBlockerArrows()
	if len(arrows) != 0 {
		t.Errorf("expected no AI blocker arrows when no AI blockers, got %d", len(arrows))
	}
}

func TestIsValidBlock_AttackerNotInValidTargets(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	s.lastMsg.Options[0].ValidTargets = nil

	if s.isValidBlock(blocker.ID, attacker.ID) {
		t.Error("expected isValidBlock to return false when attacker is not in ValidTargets")
	}
}

func TestIsValidBlock_AttackerInValidTargets(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	if !s.isValidBlock(blocker.ID, attacker.ID) {
		t.Error("expected isValidBlock to return true when attacker is in ValidTargets")
	}
}
func TestPendingBlockers_ClearedWhenLeavingPhase(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	s.pendingBlockers[blocker.ID] = attacker.ID
	s.selectedBlocker = blocker.ID

	s.lastMsg.Prompt = interactive.PromptNone
	s.lastMsg.State.Step = stepCombatDamage
	s.refreshCardActions()

	if len(s.pendingBlockers) != 0 {
		t.Error("pendingBlockers should be cleared when leaving declare blockers step")
	}
	if s.selectedBlocker != uuid.Nil {
		t.Error("selectedBlocker should be cleared when leaving declare blockers step")
	}
}
