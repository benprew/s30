// Package timing centralizes fixed-update timing for the game loop.
package timing

import (
	"math"
	"time"
)

const UpdatesPerSecond = 60

// Ticks converts a wall-clock duration to fixed updates.
func Ticks(duration time.Duration) int {
	return int(math.Round(duration.Seconds() * UpdatesPerSecond))
}

// ProbabilityPerUpdate converts a probability over a duration to an equivalent per-update probability.
func ProbabilityPerUpdate(probability float64, duration time.Duration) float64 {
	if probability <= 0 {
		return 0
	}
	if probability >= 1 {
		return 1
	}
	return 1 - math.Pow(1-probability, 1.0/float64(Ticks(duration)))
}
