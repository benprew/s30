package duel

import (
	"image"
	"math"
	"sort"
	"time"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

const (
	spellAnimationMoveDuration = 300 * time.Millisecond
	spellAnimationHoldDuration = 200 * time.Millisecond
	spellAnimationCardWidth    = 245.0
	spellAnimationCardHeight   = 342.0
)

type spellAnimationBounds struct {
	x      float64
	y      float64
	width  float64
	height float64
}

type spellAnimationFrame struct {
	bounds   spellAnimationBounds
	complete bool
}

type spellCastAnimation struct {
	id          uuid.UUID
	name        string
	controller  string
	startedAt   time.Time
	source      image.Rectangle
	destination image.Rectangle
	resolvedAt  time.Time
	resolved    bool
	enterTween  *gween.Tween
	exitTween   *gween.Tween
}

func newSpellCastAnimation(id uuid.UUID, name string, source image.Rectangle, now time.Time) *spellCastAnimation {
	return &spellCastAnimation{
		id:         id,
		name:       name,
		startedAt:  now,
		source:     source,
		enterTween: gween.New(0, 1, float32(spellAnimationMoveDuration), ease.OutCubic),
		exitTween:  gween.New(0, 1, float32(spellAnimationMoveDuration), ease.OutCubic),
	}
}

func (a *spellCastAnimation) resolve(destination image.Rectangle, now time.Time) {
	a.destination = destination
	a.resolvedAt = maxTime(now, a.startedAt.Add(spellAnimationMoveDuration+spellAnimationHoldDuration))
	a.resolved = true
}

func (a *spellCastAnimation) frame(now time.Time, _, _ int) spellAnimationFrame {
	magnifier := spellAnimationBounds{
		x:      cardPreviewX,
		y:      cardPreviewY,
		width:  spellAnimationCardWidth,
		height: spellAnimationCardHeight,
	}
	source := rectangleAnimationBounds(a.source)

	enterElapsed := max(now.Sub(a.startedAt), 0)
	enterProgress, _ := a.enterTween.Set(float32(enterElapsed))
	if enterElapsed < spellAnimationMoveDuration {
		return spellAnimationFrame{bounds: interpolateSpellBounds(source, magnifier, float64(enterProgress))}
	}
	if !a.resolved || now.Before(a.resolvedAt) {
		return spellAnimationFrame{bounds: magnifier}
	}

	exitElapsed := max(now.Sub(a.resolvedAt), 0)
	exitProgress, complete := a.exitTween.Set(float32(exitElapsed))
	destination := rectangleAnimationBounds(a.destination)
	return spellAnimationFrame{
		bounds:   interpolateSpellBounds(magnifier, destination, float64(exitProgress)),
		complete: complete,
	}
}

func rectangleAnimationBounds(bounds image.Rectangle) spellAnimationBounds {
	return spellAnimationBounds{
		x:      float64(bounds.Min.X),
		y:      float64(bounds.Min.Y),
		width:  float64(bounds.Dx()),
		height: float64(bounds.Dy()),
	}
}

func interpolateSpellBounds(from, to spellAnimationBounds, progress float64) spellAnimationBounds {
	return spellAnimationBounds{
		x:      from.x + (to.x-from.x)*progress,
		y:      from.y + (to.y-from.y)*progress,
		width:  from.width + (to.width-from.width)*progress,
		height: from.height + (to.height-from.height)*progress,
	}
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func (s *DuelScreen) syncSpellAnimations(prev, cur *interactive.GameMsg, now time.Time) {
	if cur == nil || cur.State == nil {
		return
	}
	if s.spellAnimations == nil {
		s.spellAnimations = make(map[uuid.UUID]*spellCastAnimation)
	}

	previousStack := stackItemIDs(prev)
	for _, item := range cur.State.StackItems {
		if item.IsAbility {
			continue
		}
		id, err := uuid.Parse(item.ID)
		if err != nil || previousStack[id] || s.spellAnimations[id] != nil {
			continue
		}
		animation := newSpellCastAnimation(id, item.Name, s.spellAnimationSource(prev, item, id), now)
		animation.controller = item.Controller
		s.spellAnimations[id] = animation
	}

	currentStack := stackItemIDs(cur)
	for id, animation := range s.spellAnimations {
		if animation.resolved || currentStack[id] {
			continue
		}
		animation.resolve(s.spellAnimationDestination(cur.State, animation), now)
	}
}

func stackItemIDs(msg *interactive.GameMsg) map[uuid.UUID]bool {
	ids := make(map[uuid.UUID]bool)
	if msg == nil || msg.State == nil {
		return ids
	}
	for _, item := range msg.State.StackItems {
		if id, err := uuid.Parse(item.ID); err == nil {
			ids[id] = true
		}
	}
	return ids
}

func (s *DuelScreen) spellAnimationSource(prev *interactive.GameMsg, item interactive.StackItemState, id uuid.UUID) image.Rectangle {
	dp := s.opponent
	var hand []interactive.CardState
	if item.Controller == "You" {
		dp = s.self
	}
	if prev != nil && prev.State != nil {
		if item.Controller == "You" {
			hand = handDisplayOrder(prev.State.You.Hand)
		} else {
			hand = handDisplayOrder(prev.State.Opponent.Hand)
		}
	}

	width, height, panelHeight := fieldCardW, fieldCardH, 0
	panelX, panelY := 860, 384
	if dp != nil {
		panelX, panelY = dp.handX, dp.handY
		if dp.handBg != nil {
			width = dp.handBg.Bounds().Dx()
			panelHeight = dp.handBg.Bounds().Dy()
		}
	}
	y := panelY + panelHeight
	for index, card := range hand {
		if card.ID == id {
			y += index * handCardOverlap
			break
		}
	}
	return image.Rect(panelX, y, panelX+width, y+height)
}

func (s *DuelScreen) spellAnimationDestination(state *interactive.GameState, animation *spellCastAnimation) image.Rectangle {
	if bounds, ok := s.spellBattlefieldDestination(state, animation.id); ok {
		return bounds
	}
	for _, entry := range []struct {
		dp    *duelPlayer
		cards []interactive.CardState
	}{
		{s.self, state.You.Graveyard},
		{s.opponent, state.Opponent.Graveyard},
	} {
		for _, card := range entry.cards {
			if card.ID == animation.id {
				return s.graveyardBounds(entry.dp)
			}
		}
	}
	if animation.controller == "You" {
		return s.graveyardBounds(s.self)
	}
	return s.graveyardBounds(s.opponent)
}

func (s *DuelScreen) spellBattlefieldDestination(state *interactive.GameState, id uuid.UUID) (image.Rectangle, bool) {
	if s.cardPositions == nil {
		s.cardPositions = make(map[uuid.UUID]image.Point)
	}
	for _, entry := range []struct {
		dp    *duelPlayer
		state interactive.PlayerState
	}{
		{s.self, state.You},
		{s.opponent, state.Opponent},
	} {
		for _, row := range allPermRows {
			perms := s.fieldPerms(entry.state, row)
			for index, perm := range perms {
				pos := s.getFieldCardPos(perm, entry.dp, index, len(perms), row)
				if perm.ID == id {
					return image.Rect(pos.X, pos.Y, pos.X+fieldCardW, pos.Y+fieldCardH), true
				}
			}
		}
		for _, perm := range entry.state.Battlefield {
			if perm.ID != id || perm.AttachedTo == uuid.Nil {
				continue
			}
			if host, ok := s.cardPositions[perm.AttachedTo]; ok {
				return image.Rect(host.X, host.Y-14, host.X+fieldCardW, host.Y+fieldCardH-14), true
			}
		}
	}
	return image.Rectangle{}, false
}

func (s *DuelScreen) spellIsAnimating(id uuid.UUID, now time.Time) bool {
	animation := s.spellAnimations[id]
	return animation != nil && !animation.frame(now, 1024, 768).complete
}

func (s *DuelScreen) pruneSpellAnimations(now time.Time) {
	for id, animation := range s.spellAnimations {
		if animation.frame(now, 1024, 768).complete {
			delete(s.spellAnimations, id)
		}
	}
}

func (s *DuelScreen) drawSpellAnimations(screen *ebiten.Image, W, H int) {
	now := time.Now()
	animations := make([]*spellCastAnimation, 0, len(s.spellAnimations))
	for _, animation := range s.spellAnimations {
		animations = append(animations, animation)
	}
	sort.Slice(animations, func(i, j int) bool {
		return animations[i].startedAt.Before(animations[j].startedAt)
	})
	for _, animation := range animations {
		frame := animation.frame(now, W, H)
		if frame.complete {
			continue
		}
		card := s.getDomainCard(animation.name)
		if card == nil {
			continue
		}
		img, err := card.CardImage(domain.CardViewFull)
		if err != nil || img == nil {
			continue
		}
		bounds := frame.bounds
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(bounds.width/float64(img.Bounds().Dx()), bounds.height/float64(img.Bounds().Dy()))
		opts.GeoM.Translate(math.Round(bounds.x), math.Round(bounds.y))
		screen.DrawImage(img, opts)
	}
}
