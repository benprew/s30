package world

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestAssignAmuletColor(t *testing.T) {
	tests := []struct {
		cityIndex int
		expected  domain.ColorMask
	}{
		{0, domain.ColorWhite},
		{1, domain.ColorBlue},
		{2, domain.ColorBlack},
		{3, domain.ColorRed},
		{4, domain.ColorGreen},
		{5, domain.ColorWhite}, // Should wrap around
		{6, domain.ColorBlue},
		{10, domain.ColorWhite}, // 10 % 5 = 0
	}

	for _, test := range tests {
		result := assignAmuletColor(test.cityIndex)
		if result != test.expected {
			t.Errorf("assignAmuletColor(%d) = %v, expected %v", test.cityIndex, result, test.expected)
		}
	}
}

func TestColorDistribution(t *testing.T) {
	colorCounts := make(map[domain.ColorMask]int)
	numCities := 20

	for i := 0; i < numCities; i++ {
		color := assignAmuletColor(i)
		colorCounts[color]++
	}

	expectedColors := domain.GetAllAmuletColors()
	for _, color := range expectedColors {
		if colorCounts[color] == 0 {
			t.Errorf("Color %v was not assigned to any cities", color)
		}
		if colorCounts[color] != numCities/len(expectedColors) {
			t.Errorf("Color %v was assigned %d times, expected %d times for balanced distribution",
				color, colorCounts[color], numCities/len(expectedColors))
		}
	}
}
