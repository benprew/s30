package domain

import (
	"testing"
)

func TestNewAmulet(t *testing.T) {
	tests := []struct {
		color    ColorMask
		expected string
	}{
		{ColorWhite, "Amulet of Order"},
		{ColorBlue, "Amulet of Knowledge"},
		{ColorBlack, "Amulet of Power"},
		{ColorRed, "Amulet of Passion"},
		{ColorGreen, "Amulet of Life"},
	}

	for _, test := range tests {
		amulet := NewAmulet(test.color)
		if amulet.Name != test.expected {
			t.Errorf("NewAmulet(%v).Name = %s, expected %s", test.color, amulet.Name, test.expected)
		}
		if amulet.Color != test.color {
			t.Errorf("NewAmulet(%v).Color = %v, expected %v", test.color, amulet.Color, test.color)
		}
		if amulet.Description == "" {
			t.Errorf("NewAmulet(%v).Description is empty", test.color)
		}
	}
}

func TestGetAllAmuletColors(t *testing.T) {
	colors := GetAllAmuletColors()
	expected := 5
	if len(colors) != expected {
		t.Errorf("GetAllAmuletColors() returned %d colors, expected %d", len(colors), expected)
	}

	expectedColors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}
	for i, color := range expectedColors {
		if colors[i] != color {
			t.Errorf("GetAllAmuletColors()[%d] = %v, expected %v", i, colors[i], color)
		}
	}
}

func TestColorMaskToString(t *testing.T) {
	tests := []struct {
		color    ColorMask
		expected string
	}{
		{ColorWhite, "White"},
		{ColorBlue, "Blue"},
		{ColorBlack, "Black"},
		{ColorRed, "Red"},
		{ColorGreen, "Green"},
		{ColorColorless, "Unknown"},
	}

	for _, test := range tests {
		result := ColorMaskToString(test.color)
		if result != test.expected {
			t.Errorf("ColorMaskToString(%v) = %s, expected %s", test.color, result, test.expected)
		}
	}
}
