package screens

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

func TestIsXSpell(t *testing.T) {
	tests := []struct {
		manaCost string
		want     bool
	}{
		{"{X}{R}", true},
		{"{X}{X}{G}", true},
		{"{3}{R}", false},
		{"{0}", false},
		{"{X}", true},
	}
	for _, tt := range tests {
		got := isXSpell(tt.manaCost)
		if got != tt.want {
			t.Errorf("isXSpell(%q) = %v, want %v", tt.manaCost, got, tt.want)
		}
	}
}

func TestMaxXValue(t *testing.T) {
	tests := []struct {
		name     string
		pool     interactive.ManaPoolState
		perms    []interactive.PermanentState
		manaCost string
		want     int
	}{
		{
			name:     "3 untapped lands, X+R spell",
			pool:     interactive.ManaPoolState{},
			perms:    makeLands(3, false),
			manaCost: "{X}{R}",
			want:     2,
		},
		{
			name:     "5 untapped lands, X spell",
			pool:     interactive.ManaPoolState{},
			perms:    makeLands(5, false),
			manaCost: "{X}",
			want:     5,
		},
		{
			name:     "2 untapped lands + 1 mana in pool, X+G spell",
			pool:     interactive.ManaPoolState{Green: 1},
			perms:    makeLands(2, false),
			manaCost: "{X}{G}",
			want:     2,
		},
		{
			name:     "all tapped, 2 in pool, X spell",
			pool:     interactive.ManaPoolState{Red: 2},
			perms:    makeLands(3, true),
			manaCost: "{X}",
			want:     2,
		},
		{
			name:     "no mana available beyond base cost",
			pool:     interactive.ManaPoolState{Red: 1},
			perms:    nil,
			manaCost: "{X}{R}",
			want:     0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := interactive.PlayerState{
				ManaPool:    tt.pool,
				Battlefield: tt.perms,
			}
			got := maxXValue(ps, tt.manaCost)
			if got != tt.want {
				t.Errorf("maxXValue() = %d, want %d", got, tt.want)
			}
		})
	}
}

func makeLands(n int, tapped bool) []interactive.PermanentState {
	perms := make([]interactive.PermanentState, n)
	for i := range n {
		perms[i] = interactive.PermanentState{
			IsLand: true,
			Tapped: tapped,
		}
	}
	return perms
}
