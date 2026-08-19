package timing

import (
	"math"
	"testing"
	"time"
)

func TestTicksConvertsDurationToUpdates(t *testing.T) {
	if got := Ticks(10 * time.Second); got != 600 {
		t.Fatalf("Ticks(10 seconds) = %d, want 600", got)
	}
}

func TestProbabilityPerUpdatePreservesProbabilityOverTime(t *testing.T) {
	perUpdate := ProbabilityPerUpdate(0.02, 100*time.Millisecond)
	probabilityAfterSixUpdates := 1 - math.Pow(1-perUpdate, 6)

	if math.Abs(probabilityAfterSixUpdates-0.02) > 1e-12 {
		t.Fatalf("probability after six updates = %v, want 0.02", probabilityAfterSixUpdates)
	}
}
