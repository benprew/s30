package minimap

import "testing"

func TestBlinkTimingPreservesTenTPSCadence(t *testing.T) {
	if blinkPeriod != 60 {
		t.Fatalf("blinkPeriod = %d, want 60", blinkPeriod)
	}
	if blinkVisible != 42 {
		t.Fatalf("blinkVisible = %d, want 42", blinkVisible)
	}
}
