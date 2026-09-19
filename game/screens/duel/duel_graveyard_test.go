package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func setupGraveyardTest() *DuelScreen {
	state := &interactive.GameState{
		Step:         stepPrecombatMain,
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "You",
			Life: 20,
			Graveyard: []interactive.CardState{
				{ID: uuid.New(), Name: "Lightning Bolt"},
				{ID: uuid.New(), Name: "Counterspell"},
			},
		},
		Opponent: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "Opponent",
			Life: 20,
			Graveyard: []interactive.CardState{
				{ID: uuid.New(), Name: "Giant Growth"},
			},
		},
	}

	msg := &interactive.GameMsg{State: state}

	s := &DuelScreen{
		lastMsg:          msg,
		self:             &duelPlayer{name: "You"},
		opponent:         &duelPlayer{name: "Opponent"},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}
	return s
}

func TestGraveyardBounds_Self(t *testing.T) {
	s := setupGraveyardTest()
	r := s.graveyardBounds(s.self)
	if !image.Pt(80, 600).In(r) {
		t.Errorf("expected (80,600) inside self graveyard bounds %v", r)
	}
	if image.Pt(0, 0).In(r) {
		t.Errorf("did not expect (0,0) inside self graveyard bounds %v", r)
	}
}

func TestGraveyardBounds_Opponent(t *testing.T) {
	s := setupGraveyardTest()
	r := s.graveyardBounds(s.opponent)
	if !image.Pt(80, 120).In(r) {
		t.Errorf("expected (80,120) inside opponent graveyard bounds %v", r)
	}
}

func TestGraveyardClick_OpensView(t *testing.T) {
	s := setupGraveyardTest()
	r := s.graveyardBounds(s.self)
	if !s.handleGraveyardClick(r.Min.X+5, r.Min.Y+5) {
		t.Fatal("expected click on self graveyard to be handled")
	}
	if s.viewingGraveyard != s.self {
		t.Errorf("expected viewingGraveyard to be self, got %v", s.viewingGraveyard)
	}
}

func TestGraveyardClick_OpensOpponentView(t *testing.T) {
	s := setupGraveyardTest()
	r := s.graveyardBounds(s.opponent)
	if !s.handleGraveyardClick(r.Min.X+5, r.Min.Y+5) {
		t.Fatal("expected click on opponent graveyard to be handled")
	}
	if s.viewingGraveyard != s.opponent {
		t.Errorf("expected viewingGraveyard to be opponent, got %v", s.viewingGraveyard)
	}
}

func TestGraveyardClick_EmptyGraveyardIgnored(t *testing.T) {
	s := setupGraveyardTest()
	s.lastMsg.State.You.Graveyard = nil
	r := s.graveyardBounds(s.self)
	if s.handleGraveyardClick(r.Min.X+5, r.Min.Y+5) {
		t.Fatal("expected click on empty graveyard to be ignored")
	}
	if s.viewingGraveyard != nil {
		t.Errorf("expected viewingGraveyard to remain nil")
	}
}

func TestGraveyardClick_OutsideBounds(t *testing.T) {
	s := setupGraveyardTest()
	if s.handleGraveyardClick(500, 500) {
		t.Fatal("expected click outside graveyard bounds to be ignored")
	}
}

func TestGraveyardClick_ToggleClose(t *testing.T) {
	s := setupGraveyardTest()
	r := s.graveyardBounds(s.self)
	s.handleGraveyardClick(r.Min.X+5, r.Min.Y+5)
	if s.viewingGraveyard != s.self {
		t.Fatal("expected view to be open")
	}
	if !s.handleGraveyardClick(r.Min.X+5, r.Min.Y+5) {
		t.Fatal("expected second click on same graveyard to be handled")
	}
	if s.viewingGraveyard != nil {
		t.Errorf("expected second click on same graveyard to close view")
	}
}

func TestCompactCardHitAreasMatchSharedSize(t *testing.T) {
	s := setupGraveyardTest()
	for _, dp := range []*duelPlayer{s.self, s.opponent} {
		if got := s.graveyardBounds(dp).Size(); got != image.Pt(110, 96) {
			t.Fatalf("graveyard size = %v", got)
		}
	}
	w, h := fieldCardW, fieldCardH
	if w != 110 || h != 96 {
		t.Fatalf("battlefield size = %dx%d", w, h)
	}
}

func TestGraveyardBrowserUsesArtCropBounds(t *testing.T) {
	s := setupGraveyardTest()
	s.viewingGraveyard = s.self
	sections := s.graveyardViewLayout(1024)
	if len(sections) != 1 || len(sections[0].Cards) != 2 {
		t.Fatal("missing graveyard cards")
	}
	for _, card := range sections[0].Cards {
		if card.Rect.Size() != image.Pt(147, 129) {
			t.Fatalf("graveyard browser card bounds = %v", card.Rect)
		}
	}
}
