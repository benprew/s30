package duel

import (
	"image"
	"image/color"
	"math"
	"slices"
	"time"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tanema/gween/ease"
)

const (
	landAnimationDuration = 250 * time.Millisecond
	tappedCardAngle       = math.Pi / 2
)

type landPose struct {
	x     float64
	angle float64
}

func landSlot(x int, tapped bool) landPose {
	pose := landPose{x: float64(x)}
	if tapped {
		pose.angle = tappedCardAngle
	}
	return pose
}

func (pose landPose) transform(y int) ebiten.GeoM {
	const halfW, halfH = fieldCardW / 2.0, fieldCardH / 2.0
	centerShift := (halfW - halfH) * (pose.angle / tappedCardAngle)
	centerY := float64(y) + halfH + centerShift
	sin, cos := math.Sincos(pose.angle)
	// Leave space for the border when the card turns near the edge of the screen.
	margin := sin*halfW + cos*halfH + 3
	centerY = min(max(centerY, margin), duelScreenH-margin)
	var geo ebiten.GeoM
	geo.Translate(-halfW, -halfH)
	geo.Rotate(pose.angle)
	geo.Translate(pose.x+halfW-centerShift, centerY)
	return geo
}

type landAnimation struct {
	startedAt time.Time
	from, to  landPose
}

func (a landAnimation) complete(now time.Time) bool {
	return now.Sub(a.startedAt) >= landAnimationDuration
}

func (a landAnimation) poseAt(now time.Time) landPose {
	if a.complete(now) {
		return a.to
	}
	elapsed := max(now.Sub(a.startedAt), 0)
	progress := float64(ease.OutCubic(float32(elapsed), 0, 1, float32(landAnimationDuration)))
	return landPose{
		x:     a.from.x + (a.to.x-a.from.x)*progress,
		angle: a.from.angle + (a.to.angle-a.from.angle)*progress,
	}
}

func (s *DuelScreen) syncLandAnimations(prev, cur *interactive.GameMsg, now time.Time) {
	if prev == nil || prev.State == nil || cur == nil || cur.State == nil {
		clear(s.landAnimations)
		return
	}
	s.syncLandRowAnimations(s.self, &prev.State.You, &cur.State.You, now)
	s.syncLandRowAnimations(s.opponent, &prev.State.Opponent, &cur.State.Opponent, now)
}

func (s *DuelScreen) syncLandRowAnimations(dp *duelPlayer, prev, cur *interactive.PlayerState, now time.Time) {
	if slices.EqualFunc(prev.Battlefield, cur.Battlefield, sameLandRowEntry) {
		return
	}
	previousLands := s.fieldPerms(prev, permRowLand)
	previousLayout := s.fieldRowLayout(dp, permRowLand, previousLands)
	previousSlots := make(map[uuid.UUID]landPose, len(previousLands))
	for i, land := range previousLands {
		previousSlots[land.ID] = landSlot(previousLayout.position(i).X, land.Tapped)
	}

	lands := s.fieldPerms(cur, permRowLand)
	layout := s.fieldRowLayout(dp, permRowLand, lands)
	for i, land := range lands {
		from, existed := previousSlots[land.ID]
		if !existed {
			continue
		}
		delete(previousSlots, land.ID)
		to := landSlot(layout.position(i).X, land.Tapped)
		if from == to {
			continue
		}
		if active, ok := s.landAnimations[land.ID]; ok {
			from = active.poseAt(now)
		}
		if from == to {
			delete(s.landAnimations, land.ID)
			continue
		}
		if s.landAnimations == nil {
			s.landAnimations = make(map[uuid.UUID]landAnimation)
		}
		s.landAnimations[land.ID] = landAnimation{startedAt: now, from: from, to: to}
	}
	for id := range previousSlots {
		delete(s.landAnimations, id)
	}
}

func sameLandRowEntry(a, b interactive.PermanentState) bool {
	aInRow := a.AttachedTo == uuid.Nil() && permRowFor(a) == permRowLand
	bInRow := b.AttachedTo == uuid.Nil() && permRowFor(b) == permRowLand
	return aInRow == bInRow && (!aInRow || (a.ID == b.ID && a.Name == b.Name && a.Tapped == b.Tapped))
}

func (s *DuelScreen) pruneLandAnimations(now time.Time) {
	for id, animation := range s.landAnimations {
		if animation.complete(now) {
			delete(s.landAnimations, id)
		}
	}
}

type fieldCardFrame struct {
	pos       image.Point
	transform ebiten.GeoM
	moving    bool
}

func (s *DuelScreen) fieldCardFrame(id uuid.UUID, pos image.Point, now time.Time) fieldCardFrame {
	pos.Y += int(math.Round(s.attackerLiftY(id, now)))
	if animation, ok := s.landAnimations[id]; ok && !animation.complete(now) {
		pose := animation.poseAt(now)
		pos.X = int(math.Round(pose.x))
		return fieldCardFrame{pos: pos, transform: pose.transform(pos.Y), moving: true}
	}
	return fieldCardFrame{pos: pos}
}

func (frame fieldCardFrame) artTransform(tapped bool, artHeight int) ebiten.GeoM {
	if frame.moving {
		return frame.transform
	}
	var geo ebiten.GeoM
	if tapped {
		geo.Rotate(tappedCardAngle)
		geo.Translate(float64(artHeight), 0)
	}
	geo.Translate(float64(frame.pos.X), float64(frame.pos.Y))
	return geo
}

func (frame fieldCardFrame) contains(point image.Point, perm interactive.PermanentState) bool {
	if !frame.moving {
		return point.In(fieldPermanentBounds(perm, frame.pos))
	}
	inverse := frame.transform
	inverse.Invert()
	x, y := inverse.Apply(float64(point.X)+0.5, float64(point.Y)+0.5)
	return x >= 0 && x < fieldCardW && y >= 0 && y < fieldCardH
}

func (frame fieldCardFrame) path() vector.Path {
	var path vector.Path
	x, y := frame.transform.Apply(0, 0)
	path.MoveTo(float32(x), float32(y))
	for _, corner := range [...]image.Point{{X: fieldCardW}, {X: fieldCardW, Y: fieldCardH}, {Y: fieldCardH}} {
		x, y := frame.transform.Apply(float64(corner.X), float64(corner.Y))
		path.LineTo(float32(x), float32(y))
	}
	path.Close()
	return path
}

func (frame fieldCardFrame) stroke(screen *ebiten.Image, bounds image.Rectangle, width float32, clr color.RGBA) {
	if !frame.moving {
		vector.StrokeRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()), width, clr, false)
		return
	}
	path := frame.path()
	drawOpts := &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: width, MiterLimit: 10}, drawOpts)
}
