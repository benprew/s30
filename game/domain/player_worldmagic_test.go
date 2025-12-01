package domain

import (
	"testing"
)

func TestPlayerWorldMagicTracking(t *testing.T) {
	magic1 := &WorldMagic{Name: "Magic1", Cost: 100, Description: "First magic"}
	magic2 := &WorldMagic{Name: "Magic2", Cost: 200, Description: "Second magic"}

	player := Player{
		WorldMagics: make([]*WorldMagic, 0),
	}

	if player.HasWorldMagic(magic1) {
		t.Error("Player should not have magic1 initially")
	}

	if len(player.GetWorldMagics()) != 0 {
		t.Error("Player should have no world magics initially")
	}

	player.AddWorldMagic(magic1)

	if !player.HasWorldMagic(magic1) {
		t.Error("Player should have magic1 after adding")
	}

	if len(player.GetWorldMagics()) != 1 {
		t.Error("Player should have 1 world magic after adding")
	}

	player.AddWorldMagic(magic1)
	if len(player.GetWorldMagics()) != 1 {
		t.Error("Adding duplicate magic should not increase count")
	}

	player.AddWorldMagic(magic2)
	if len(player.GetWorldMagics()) != 2 {
		t.Error("Player should have 2 world magics after adding second")
	}

	if !player.HasWorldMagic(magic2) {
		t.Error("Player should have magic2 after adding")
	}
}

func TestWorldMagicDefinitions(t *testing.T) {
	expectedMagics := []struct {
		name string
		cost int
	}{
		{"Sword of Resistance", 400},
		{"Quickening", 300},
		{"Leap of Fate", 300},
		{"Ring of the Guardian", 500},
		{"Haggler's Coin", 250},
		{"Tome of Enlightenment", 300},
		{"Sleight of Hand", 300},
		{"Staff of Thunder", 100},
		{"Conjurer's Will", 300},
		{"Dwarven Pick", 125},
		{"Amulet of Swampwalk", 125},
		{"Fruit of Sustenance", 50},
	}

	if len(AllWorldMagics) != len(expectedMagics) {
		t.Errorf("Expected %d world magics, got %d", len(expectedMagics), len(AllWorldMagics))
	}

	for i, expected := range expectedMagics {
		if i >= len(AllWorldMagics) {
			break
		}
		magic := AllWorldMagics[i]
		if magic.Name != expected.name {
			t.Errorf("Expected magic name %s at index %d, got %s", expected.name, i, magic.Name)
		}
		if magic.Cost != expected.cost {
			t.Errorf("Expected cost %d for %s, got %d", expected.cost, expected.name, magic.Cost)
		}
		if magic.Description == "" {
			t.Errorf("Magic %s should have a description", magic.Name)
		}
	}
}
