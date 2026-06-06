package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func setupHandLayoutTest() *DuelScreen {
	cardID := uuid.New()
	hand := []interactive.CardState{{ID: cardID, Name: "Lightning Bolt"}}

	state := &interactive.GameState{
		Step:         stepDeclareAttackers,
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:        uuid.New(),
			Name:      "You",
			Life:      20,
			Hand:      hand,
			HandCount: len(hand),
		},
		Opponent: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "Opponent",
			Life: 20,
		},
	}

	msg := &interactive.GameMsg{State: state}

	s := &DuelScreen{
		lastMsg:          msg,
		self:             &duelPlayer{name: "You", handX: 860, handY: 430},
		opponent:         &duelPlayer{name: "Opponent", handX: 860, handY: 310},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}
	return s
}

// firstHandCardPoint returns coordinates that land on the first hand card when
// the panel is expanded.
func (s *DuelScreen) firstHandCardPoint() (int, int) {
	headerH := s.panelCardH(s.self)
	x := s.self.handX + 5
	y := s.self.handY + headerH + 2
	return x, y
}

func TestHandToggle_Collapse(t *testing.T) {
	s := setupHandLayoutTest()

	if s.handCollapsed {
		t.Fatal("hand should start expanded")
	}

	s.toggleHand()
	if !s.handCollapsed {
		t.Fatal("toggleHand should collapse the hand")
	}

	s.toggleHand()
	if s.handCollapsed {
		t.Fatal("toggleHand should expand the hand again")
	}
}

func TestHandCardIdx_CollapsedNotClickable(t *testing.T) {
	s := setupHandLayoutTest()
	hand := s.lastMsg.State.You.Hand

	x, y := s.firstHandCardPoint()

	if idx := s.handCardIdxAtPoint(x, y, s.self.handX, s.self.handY, len(hand), s.self); idx < 0 {
		t.Fatal("expanded hand card should be clickable")
	}

	s.toggleHand()

	if idx := s.handCardIdxAtPoint(x, y, s.self.handX, s.self.handY, len(hand), s.self); idx >= 0 {
		t.Fatalf("collapsed hand cards should not be clickable, got idx %d", idx)
	}
}

func TestHandHeader_AlwaysClickableForToggle(t *testing.T) {
	s := setupHandLayoutTest()

	headerMidX := s.self.handX + s.panelCardW(s.self)/2
	headerMidY := s.self.handY + s.panelCardH(s.self)/2

	if !s.pointInHandHeader(headerMidX, headerMidY, s.self) {
		t.Fatal("header point should be inside hand header when expanded")
	}

	s.toggleHand()

	if !s.pointInHandHeader(headerMidX, headerMidY, s.self) {
		t.Fatal("header point should remain inside hand header when collapsed")
	}
}

func TestHandHeader_OutsidePointsRejected(t *testing.T) {
	s := setupHandLayoutTest()

	if s.pointInHandHeader(s.self.handX-1, s.self.handY+1, s.self) {
		t.Fatal("point left of header should be rejected")
	}
	if s.pointInHandHeader(s.self.handX+1, s.self.handY-1, s.self) {
		t.Fatal("point above header should be rejected")
	}
}
