package duel

import (
	"image"
	"testing"
	"time"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func TestSpellCastAnimationEasesToMagnifierAndHolds(t *testing.T) {
	start := time.Now()
	animation := newSpellCastAnimation(uuid.New(), "Lightning Bolt", image.Rect(900, 600, 1000, 683), start)

	atStart := animation.frame(start, 1024, 768)
	if atStart.bounds != (spellAnimationBounds{900, 600, 100, 83}) {
		t.Fatalf("start bounds = %+v", atStart.bounds)
	}

	mid := animation.frame(start.Add(spellAnimationMoveDuration/2), 1024, 768)
	magnifier := animation.frame(start.Add(spellAnimationMoveDuration), 1024, 768)
	if magnifier.bounds != (spellAnimationBounds{0, 188, 245, 342}) {
		t.Fatalf("arrival bounds = %+v, want magnifier bounds", magnifier.bounds)
	}
	linearMidX := (atStart.bounds.x + magnifier.bounds.x) / 2
	if mid.bounds.x >= linearMidX {
		t.Fatalf("ease-out midpoint x = %f, want past linear midpoint %f", mid.bounds.x, linearMidX)
	}
	if mid.bounds.width <= (atStart.bounds.width+magnifier.bounds.width)/2 {
		t.Fatalf("ease-out midpoint width = %f, want past linear midpoint", mid.bounds.width)
	}

	held := animation.frame(start.Add(spellAnimationMoveDuration+spellAnimationHoldDuration), 1024, 768)
	if held.bounds != magnifier.bounds {
		t.Fatalf("hold bounds = %+v, want magnifier %+v", held.bounds, magnifier.bounds)
	}
}

func TestSpellCastAnimationEasesFromMagnifierToDestination(t *testing.T) {
	start := time.Now()
	animation := newSpellCastAnimation(uuid.New(), "Grizzly Bears", image.Rect(900, 600, 1000, 683), start)
	destination := image.Rect(400, 470, 500, 553)
	animation.resolve(destination, start.Add(spellAnimationMoveDuration+spellAnimationHoldDuration))

	magnifier := animation.frame(animation.resolvedAt, 1024, 768)
	mid := animation.frame(animation.resolvedAt.Add(spellAnimationMoveDuration/2), 1024, 768)
	linearMidX := (magnifier.bounds.x + float64(destination.Min.X)) / 2
	if mid.bounds.x <= linearMidX {
		t.Fatalf("ease-out exit midpoint x = %f, want past linear midpoint %f", mid.bounds.x, linearMidX)
	}

	end := animation.frame(animation.resolvedAt.Add(spellAnimationMoveDuration), 1024, 768)
	if end.bounds != (spellAnimationBounds{400, 470, 100, 83}) || !end.complete {
		t.Fatalf("end frame = %+v, want destination and complete", end)
	}
}

func TestSpellAnimationWaitsForMagnifierHoldBeforeDeparting(t *testing.T) {
	start := time.Now()
	animation := newSpellCastAnimation(uuid.New(), "Counterspell", image.Rect(900, 600, 1000, 683), start)
	animation.resolve(image.Rect(60, graveyardSelfY, 121, graveyardSelfY+graveyardH), start.Add(50*time.Millisecond))

	want := start.Add(spellAnimationMoveDuration + spellAnimationHoldDuration)
	if animation.resolvedAt != want {
		t.Fatalf("resolved at %v, want %v", animation.resolvedAt, want)
	}
}

func TestSyncSpellAnimationsTracksSpellUntilItsFinalZone(t *testing.T) {
	start := time.Now()
	id := uuid.New()
	s := &DuelScreen{
		self:            &duelPlayer{name: "You", handX: 860, handY: 430},
		opponent:        &duelPlayer{name: "Opponent", handX: 860, handY: 310},
		spellAnimations: make(map[uuid.UUID]*spellCastAnimation),
	}
	prev := &interactive.GameMsg{State: &interactive.GameState{
		You:      interactive.PlayerState{Name: "You", Hand: []interactive.CardState{{ID: id, Name: "Lightning Bolt"}}},
		Opponent: interactive.PlayerState{Name: "Opponent"},
	}}
	cast := &interactive.GameMsg{State: &interactive.GameState{
		You:        interactive.PlayerState{Name: "You"},
		Opponent:   interactive.PlayerState{Name: "Opponent"},
		StackItems: []interactive.StackItemState{{ID: id.String(), Name: "Lightning Bolt", Controller: "You"}},
	}}

	s.syncSpellAnimations(prev, cast, start)
	animation := s.spellAnimations[id]
	if animation == nil || animation.name != "Lightning Bolt" || animation.resolved {
		t.Fatalf("cast animation = %+v", animation)
	}

	resolved := &interactive.GameMsg{State: &interactive.GameState{
		You:      interactive.PlayerState{Name: "You", Graveyard: []interactive.CardState{{ID: id, Name: "Lightning Bolt"}}},
		Opponent: interactive.PlayerState{Name: "Opponent"},
	}}
	s.syncSpellAnimations(cast, resolved, start.Add(100*time.Millisecond))
	if !animation.resolved {
		t.Fatal("animation did not resolve when the card entered the graveyard")
	}
	if animation.destination.Min != image.Pt(graveyardX, graveyardSelfY) {
		t.Fatalf("destination = %v, want self graveyard", animation.destination)
	}
}

func TestSyncSpellAnimationsUsesBattlefieldDestination(t *testing.T) {
	start := time.Now()
	id := uuid.New()
	s := &DuelScreen{
		self:            &duelPlayer{name: "You"},
		opponent:        &duelPlayer{name: "Opponent"},
		cardPositions:   make(map[uuid.UUID]image.Point),
		spellAnimations: make(map[uuid.UUID]*spellCastAnimation),
	}
	cast := &interactive.GameMsg{State: &interactive.GameState{
		You:        interactive.PlayerState{Name: "You"},
		Opponent:   interactive.PlayerState{Name: "Opponent"},
		StackItems: []interactive.StackItemState{{ID: id.String(), Name: "Grizzly Bears", Controller: "You"}},
	}}
	s.syncSpellAnimations(nil, cast, start)

	resolved := &interactive.GameMsg{State: &interactive.GameState{
		You: interactive.PlayerState{Name: "You", Battlefield: []interactive.PermanentState{{
			ID: id, Name: "Grizzly Bears", IsCreature: true,
		}}},
		Opponent: interactive.PlayerState{Name: "Opponent"},
	}}
	s.syncSpellAnimations(cast, resolved, start.Add(time.Second))

	animation := s.spellAnimations[id]
	if animation == nil || !animation.resolved {
		t.Fatal("permanent spell animation did not resolve")
	}
	if animation.destination.Dx() != fieldCardW || animation.destination.Dy() != fieldCardH {
		t.Fatalf("battlefield destination = %v", animation.destination)
	}
}

func TestSyncSpellAnimationsIgnoresAbilities(t *testing.T) {
	id := uuid.New()
	s := &DuelScreen{spellAnimations: make(map[uuid.UUID]*spellCastAnimation)}
	cur := &interactive.GameMsg{State: &interactive.GameState{StackItems: []interactive.StackItemState{{
		ID: id.String(), Name: "Ability", IsAbility: true,
	}}}}
	s.syncSpellAnimations(nil, cur, time.Now())
	if len(s.spellAnimations) != 0 {
		t.Fatalf("created %d animations for an ability", len(s.spellAnimations))
	}
}
