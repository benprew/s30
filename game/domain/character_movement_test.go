package domain

import (
	"math"
	"testing"

	"github.com/benprew/s30/game/timing"
)

func TestCharacterInstanceFractionalMovement(t *testing.T) {
	c := CharacterInstance{MoveSpeed: MovementSpeed(100)}

	for range 60 {
		c.Update(DirRight)
	}

	if c.X != 100 {
		t.Fatalf("X = %d after one second, want 100", c.X)
	}
}

func TestCharacterInstanceFractionalMovementChangesDirection(t *testing.T) {
	c := CharacterInstance{MoveSpeed: MovementSpeed(100)}

	for range 3 {
		c.Update(DirRight)
	}
	for range 3 {
		c.Update(DirLeft)
	}

	if c.X != 0 {
		t.Fatalf("X = %d after equal movement in opposite directions, want 0", c.X)
	}
}

func TestCharacterInstanceWalkingAnimationKeepsTenFramesPerSecond(t *testing.T) {
	c := CharacterInstance{MoveSpeed: MovementSpeed(100)}

	for tick := 1; tick <= 5; tick++ {
		c.Update(DirRight)
		if c.Frame != 0 {
			t.Fatalf("Frame = %d after %d updates, want 0", c.Frame, tick)
		}
	}

	c.Update(DirRight)
	if c.Frame != 1 {
		t.Fatalf("Frame = %d after six updates, want 1", c.Frame)
	}
}

func TestMovementSpeedUsesPixelsPerSecond(t *testing.T) {
	for pixelsPerSecond := 50; pixelsPerSecond <= 110; pixelsPerSecond += 10 {
		c := CharacterInstance{MoveSpeed: MovementSpeed(float64(pixelsPerSecond))}
		for range 60 {
			c.Update(DirRight)
		}

		want := pixelsPerSecond
		if c.X != want {
			t.Errorf("speed %d moved %d pixels in one second, want %d", pixelsPerSecond, c.X, want)
		}
	}
}

func TestCharacterInstanceDiagonalMovementMatchesCardinalSpeed(t *testing.T) {
	cardinal := CharacterInstance{MoveSpeed: MovementSpeed(100)}
	diagonal := CharacterInstance{MoveSpeed: MovementSpeed(100)}

	for range timing.UpdatesPerSecond {
		cardinal.Update(DirRight)
		diagonal.Update(DirRight | DirDown)
	}

	cardinalDistance := preciseDistance(cardinal)
	diagonalDistance := preciseDistance(diagonal)
	if math.Abs(cardinalDistance-diagonalDistance) > 1e-9 {
		t.Fatalf("cardinal distance = %v, diagonal distance = %v", cardinalDistance, diagonalDistance)
	}
}

func preciseDistance(c CharacterInstance) float64 {
	x := float64(c.X) + c.moveRemainderX
	y := float64(c.Y) + c.moveRemainderY
	return math.Hypot(x, y)
}
