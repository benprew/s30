package screens

import "testing"

func TestTickBasedScreenTimingPreservesTenTPSDurations(t *testing.T) {
	if dungeonAmbientTicks != 1800 {
		t.Errorf("dungeonAmbientTicks = %d, want 1800", dungeonAmbientTicks)
	}
}
