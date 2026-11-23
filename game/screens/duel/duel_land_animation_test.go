package duel

import (
	"image"
	"testing"
	"time"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestBattlefieldAnimatesLandTapChanges(t *testing.T) {
	s := &DuelScreen{
		self:          &duelPlayer{name: "You"},
		opponent:      &duelPlayer{name: "Opponent"},
		cardPositions: make(map[uuid.UUID]image.Point),
	}
	forestID, islandID := uuid.New(), uuid.New()
	state := func(tapped bool) *interactive.GameState {
		return &interactive.GameState{
			You: interactive.PlayerState{Battlefield: []interactive.PermanentState{
				{ID: forestID, Name: "Forest", IsLand: true, Tapped: tapped},
			}},
			Opponent: interactive.PlayerState{Battlefield: []interactive.PermanentState{
				{ID: islandID, Name: "Island", IsLand: true, Tapped: !tapped},
			}},
		}
	}
	screen := ebiten.NewImage(1024, 768)
	defer screen.Deallocate()
	positionsAt := func(now time.Time) (forest, island image.Point) {
		s.drawBattlefield(screen, s.self, s.playerState(s.self), now)
		s.drawBattlefield(screen, s.opponent, s.playerState(s.opponent), now)
		return s.cardPositions[forestID], s.cardPositions[islandID]
	}
	s.applyGameMsg(interactive.GameMsg{State: state(false)})
	forestStart, islandStart := positionsAt(s.lastMsgTime)

	s.applyGameMsg(interactive.GameMsg{State: state(true)})
	forest, island := positionsAt(s.lastMsgTime)
	if forest != forestStart || island != islandStart {
		t.Fatalf("positions jumped to %v and %v, want %v and %v", forest, island, forestStart, islandStart)
	}
	forestEnd, islandEnd := positionsAt(s.lastMsgTime.Add(landAnimationDuration))
	if forestEnd.X != islandStart.X || islandEnd.X != forestStart.X {
		t.Errorf("lands settled at %v and %v, want them in the opposite groups", forestEnd, islandEnd)
	}
	midTime := s.lastMsgTime.Add(landAnimationDuration / 3)
	forestMid, islandMid := positionsAt(midTime)
	if forestMid.X <= forestStart.X || forestMid.X >= forestEnd.X || forestMid.Y != forestStart.Y {
		t.Errorf("Forest midpoint %v is outside the path from %v to %v", forestMid, forestStart, forestEnd)
	}
	if islandMid.X >= islandStart.X || islandMid.X <= islandEnd.X || islandMid.Y != islandStart.Y {
		t.Errorf("Island midpoint %v is outside the path from %v to %v", islandMid, islandStart, islandEnd)
	}

	forestState := s.lastMsg.State.You.Battlefield[0]
	frame := s.fieldCardFrame(forestID, forestEnd, midTime)
	geo := frame.artTransform(true, fieldCardH)
	leftX, leftY := geo.Apply(0, 0)
	rightX, rightY := geo.Apply(fieldCardW, 0)
	if rightX-leftX < 1 || rightY-leftY < 1 {
		t.Error("Forest is not tilted clockwise during the slide")
	}
	x, y := geo.Apply(fieldCardW/2.0, fieldCardH/2.0)
	if !frame.contains(image.Pt(int(x), int(y)), forestState) || frame.contains(forestStart, forestState) {
		t.Error("Forest hit area does not follow the moving card")
	}
}

func TestLandAnimationRetargetsFromCurrentPose(t *testing.T) {
	s := &DuelScreen{self: &duelPlayer{}, opponent: &duelPlayer{}}
	id := uuid.New()
	msg := func(tapped bool) *interactive.GameMsg {
		return &interactive.GameMsg{State: &interactive.GameState{
			You: interactive.PlayerState{Battlefield: []interactive.PermanentState{
				{ID: id, Name: "Forest", IsLand: true, Tapped: tapped},
			}},
		}}
	}
	untapped, tapped := msg(false), msg(true)
	start := time.Now()
	mid := start.Add(landAnimationDuration / 2)
	s.syncLandAnimations(untapped, tapped, start)
	pose := s.landAnimations[id].poseAt(mid)

	s.syncLandAnimations(tapped, msg(true), mid)
	if got := s.landAnimations[id].startedAt; !got.Equal(start) {
		t.Fatalf("unchanged land restarted its animation at %v", got)
	}
	s.syncLandAnimations(tapped, untapped, mid)
	if got := s.landAnimations[id].poseAt(mid); got != pose {
		t.Errorf("untap resumed at %+v, want the current pose %+v", got, pose)
	}

	s.pruneLandAnimations(mid.Add(landAnimationDuration))
	if len(s.landAnimations) != 0 {
		t.Error("completed land animation was retained")
	}
}

func TestEnteringTappedLandAppearsAtItsSlot(t *testing.T) {
	s := &DuelScreen{self: &duelPlayer{}, opponent: &duelPlayer{}}
	id := uuid.New()
	empty := &interactive.GameMsg{State: &interactive.GameState{}}
	entered := &interactive.GameMsg{State: &interactive.GameState{
		You: interactive.PlayerState{Battlefield: []interactive.PermanentState{
			{ID: id, Name: "Forest", IsLand: true, Tapped: true},
		}},
	}}
	s.syncLandAnimations(empty, entered, time.Now())
	if _, moving := s.landAnimations[id]; moving {
		t.Error("land that entered tapped was animated from an untapped position")
	}
}

func TestTurningLandStaysOnScreen(t *testing.T) {
	s := &DuelScreen{self: &duelPlayer{}, opponent: &duelPlayer{}}
	for _, dp := range []*duelPlayer{s.self, s.opponent} {
		y := s.fieldRowLayout(dp, permRowLand, nil).y
		pose := landPose{x: float64(fieldRowLeft), angle: tappedCardAngle / 2}
		geo := pose.transform(y)
		for _, corner := range [...]image.Point{{}, {X: fieldCardW}, {X: fieldCardW, Y: fieldCardH}, {Y: fieldCardH}} {
			_, screenY := geo.Apply(float64(corner.X), float64(corner.Y))
			if screenY < 2 || screenY > duelScreenH-2 {
				t.Errorf("rotating card at row %d has a corner at y=%.2f without room for its border", y, screenY)
			}
		}
	}
}
