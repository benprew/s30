package domain

import (
	"testing"
)

func TestEnemyAITimingPreservesTenTPSDurations(t *testing.T) {
	for range 100 {
		wait := randomEnemyWaitTicks()
		if wait < 120 || wait > 474 || wait%6 != 0 {
			t.Fatalf("randomEnemyWaitTicks() = %d, want a 100ms step in [120, 474]", wait)
		}

		direction := randomEnemyDirectionTicks()
		if direction < 180 || direction > 534 || direction%6 != 0 {
			t.Fatalf("randomEnemyDirectionTicks() = %d, want a 100ms step in [180, 534]", direction)
		}
	}
}
