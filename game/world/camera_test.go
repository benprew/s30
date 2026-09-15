package world

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/timing"
)

func TestCamera_Loc(t *testing.T) {
	cam := NewCamera()
	cam.SetPosition(100.4, 200.6)

	loc := cam.Loc()
	if loc.X != 100 || loc.Y != 201 {
		t.Errorf("expected Loc (100, 201), got (%d, %d)", loc.X, loc.Y)
	}

	char := domain.NewCharacterInstance()
	char.X = 500
	char.Y = 600
	cam.Follow(&char)
	cam.SetDynamicOffset(15.6, -10.4)

	loc = cam.Loc()
	if loc.X != 516 || loc.Y != 590 {
		t.Errorf("expected Loc (516, 590), got (%d, %d)", loc.X, loc.Y)
	}
}

func TestCamera_DynamicOffset_SmoothTransitions(t *testing.T) {
	cam := NewCamera()
	char := domain.NewCharacterInstance()
	cam.Follow(&char)

	dt := timing.DeltaTime

	// Moving right should gradually move dynamicOffsetX towards target
	cam.Update(0, domain.DirRight, dt)

	firstFrameOffset := cam.dynamicOffsetX
	if firstFrameOffset <= 0 {
		t.Fatalf("expected dynamicOffsetX > 0, got %f", firstFrameOffset)
	}

	// Smooth acceleration should ensure first-frame movement is gentle
	// rather than an immediate full-speed leap (which would be cam.dynamicPanSpeed * dt)
	fullSpeedStep := cam.dynamicPanSpeed * dt
	if firstFrameOffset >= fullSpeedStep {
		t.Errorf("expected gentle ramp-up (< %f), got %f", fullSpeedStep, firstFrameOffset)
	}

	// Simulate half a second of movement
	for range 30 {
		cam.Update(0, domain.DirRight, dt)
	}

	if cam.dynamicOffsetX <= firstFrameOffset {
		t.Errorf("expected offset to increase over time, got %f", cam.dynamicOffsetX)
	}
	if cam.dynamicOffsetX > cam.targetDynamicOffsetX {
		t.Errorf("expected offset not to overshoot target %f, got %f", cam.targetDynamicOffsetX, cam.dynamicOffsetX)
	}
}

func TestCamera_QuickDirectionSwitch_DoesNotSpikeVelocity(t *testing.T) {
	cam := NewCamera()
	char := domain.NewCharacterInstance()
	cam.Follow(&char)

	dt := timing.DeltaTime

	// Run right for 1 second so offset reaches steady state
	for range 60 {
		cam.Update(0, domain.DirRight, dt)
	}

	rightOffset := cam.dynamicOffsetX
	if rightOffset <= 0 {
		t.Fatalf("expected positive offset after moving right, got %f", rightOffset)
	}

	// Now suddenly switch direction to Left for 1 frame
	cam.Update(0, domain.DirLeft, dt)

	// In the next frame, the camera should decelerate smoothly rather than instantly
	// leaping in the negative direction at full speed.
	delta := cam.dynamicOffsetX - rightOffset
	maxInstantJump := -(cam.dynamicPanSpeed * dt)
	if delta <= maxInstantJump {
		t.Errorf("camera jerked too fast on reversal: delta %f <= %f", delta, maxInstantJump)
	}

	// For a quick direction switch (only 3 frames left, then release),
	// the camera should barely budge compared to the total displacement.
	for range 2 {
		cam.Update(0, domain.DirLeft, dt)
	}
	changeIn3Frames := rightOffset - cam.dynamicOffsetX
	if changeIn3Frames > DynamicOffsetDisplacement*0.3 {
		t.Errorf("camera moved too much on a quick tap: %f", changeIn3Frames)
	}
}

func TestCamera_ResetDynamicOffset(t *testing.T) {
	cam := NewCamera()
	char := domain.NewCharacterInstance()
	cam.Follow(&char)

	cam.SetDynamicOffset(25.0, 25.0)
	cam.ResetDynamicOffset()

	if cam.dynamicOffsetX != 0 || cam.dynamicOffsetY != 0 {
		t.Errorf("expected 0 offsets, got (%f, %f)", cam.dynamicOffsetX, cam.dynamicOffsetY)
	}
	if cam.dynamicVelocityX != 0 || cam.dynamicVelocityY != 0 {
		t.Errorf("expected 0 velocities, got (%f, %f)", cam.dynamicVelocityX, cam.dynamicVelocityY)
	}
}

func TestCamera_NilSafety(t *testing.T) {
	var cam *Camera
	loc := cam.Loc()
	if loc.X != 0 || loc.Y != 0 {
		t.Errorf("expected (0, 0) for nil camera, got (%d, %d)", loc.X, loc.Y)
	}

	// Ensure calling other methods on nil camera doesn't panic
	cam.Update(0, domain.DirRight, 0.016)
	cam.CancelFollow()
	cam.ResetManualOffset()
	cam.ResetDynamicOffset()
}

func TestLevel_RebuildSprites_InitializesCamera(t *testing.T) {
	player := &domain.Player{}
	player.X = 350
	player.Y = 450

	l := &Level{
		Player: player,
		Camera: nil,
	}

	if err := l.RebuildSprites(); err != nil {
		t.Fatalf("RebuildSprites failed: %v", err)
	}

	if l.Camera == nil {
		t.Fatalf("expected l.Camera to be initialized after RebuildSprites")
	}

	loc := l.Camera.Loc()
	if loc.X != 350 || loc.Y != 450 {
		t.Errorf("expected camera loc (350, 450), got (%d, %d)", loc.X, loc.Y)
	}
}
