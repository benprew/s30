package domain

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPlayerAmuletFunctionality(t *testing.T) {
	img := ebiten.NewImage(10, 10)
	player, err := NewPlayer("TestPlayer", img, false, DifficultyEasy, ColorColorless)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	totalStartingAmulets := 0
	for _, count := range player.Amulets {
		totalStartingAmulets += count
	}
	if totalStartingAmulets != 4 {
		t.Errorf("New easy player should have 4 starting amulets, got %d", totalStartingAmulets)
	}

	// Reset player amulets for clean unit testing of mutation methods
	player.Amulets = make(map[ColorMask]int)

	whiteAmulet := NewAmulet(ColorWhite)
	player.AddAmulet(whiteAmulet)

	if player.Amulets[ColorWhite] != 1 {
		t.Errorf("Player should have 1 white amulet after adding one, got %d", player.Amulets[ColorWhite])
	}

	if !player.HasAmulet(ColorWhite) {
		t.Error("Player should have white amulet")
	}

	if player.HasAmulet(ColorBlue) {
		t.Error("Player should not have blue amulet")
	}

	blueAmulet := NewAmulet(ColorBlue)
	player.AddAmulet(blueAmulet)
	player.AddAmulet(blueAmulet)

	counts := player.GetAmuletCount()
	if counts[ColorWhite] != 1 {
		t.Errorf("Expected 1 white amulet, got %d", counts[ColorWhite])
	}
	if counts[ColorBlue] != 2 {
		t.Errorf("Expected 2 blue amulets, got %d", counts[ColorBlue])
	}
	if counts[ColorRed] != 0 {
		t.Errorf("Expected 0 red amulets, got %d", counts[ColorRed])
	}

	allAmulets := player.GetAmulets()
	if len(allAmulets) != 3 {
		t.Errorf("Expected 3 total amulets, got %d", len(allAmulets))
	}

	err = player.RemoveAmulet(ColorBlue)
	if err != nil {
		t.Errorf("Failed to remove blue amulet: %v", err)
	}
	if player.Amulets[ColorBlue] != 1 {
		t.Errorf("Expected 1 blue amulet after removal, got %d", player.Amulets[ColorBlue])
	}

	err = player.RemoveAmulet(ColorBlue)
	if err != nil {
		t.Errorf("Failed to remove blue amulet: %v", err)
	}
	if player.Amulets[ColorBlue] != 0 {
		t.Errorf("Expected 0 blue amulets after second removal, got %d", player.Amulets[ColorBlue])
	}

	err = player.RemoveAmulet(ColorBlue)
	if err == nil {
		t.Error("Expected error when removing non-existent amulet")
	}

	err = player.RemoveAmulet(ColorRed)
	if err == nil {
		t.Error("Expected error when removing amulet player doesn't have")
	}
}
