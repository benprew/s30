package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
)

// Every hint has to carry both halves: what the toggle does while it is off, and
// what it reads as while it is on.
func TestEveryCueHasBothStates(t *testing.T) {
	if len(cueTable) == 0 {
		t.Fatal("the cue table is empty")
	}
	for key, pair := range cueTable {
		if pair[0] == "" || pair[1] == "" {
			t.Errorf("%s: want both an on and an off text, got %q", key, pair)
		}
		if pair[0] == pair[1] {
			t.Errorf("%s: on and off texts are identical (%q)", key, pair[0])
		}
	}
}

func TestCueTextFollowsTheToggleState(t *testing.T) {
	cases := []struct {
		key    string
		active bool
		want   string
	}{
		{"GREEN", true, "Click to remove green"},
		{"GREEN", false, "Cards that produce or cost green mana"},
		{"CREATURE", true, "Click to remove creatures"},
		{"CREATURE", false, "Cards that are creatures"},
	}
	for _, c := range cases {
		if got := cueText(cueTable, c.key, c.active); got != c.want {
			t.Errorf("cueText(%s, %v) = %q, want %q", c.key, c.active, got, c.want)
		}
	}
}

func TestCueTextIsEmptyForAnUnknownKey(t *testing.T) {
	if got := cueText(cueTable, "NOT_A_FILTER", true); got != "" {
		t.Errorf("cueText for an unknown key = %q, want empty", got)
	}
	if got := cueText(cueTable, "", true); got != "" {
		t.Errorf("cueText for an empty key = %q, want empty", got)
	}
}

// The filter names colors in single letters while the hints spell them out, so
// the translation is part of the contract.
func TestCueKeyNamesMatchTheTable(t *testing.T) {
	colors := map[string]string{"W": "WHITE", "U": "BLUE", "B": "BLACK", "R": "RED", "G": "GREEN"}
	for letter, want := range colors {
		if got := cueKeyForColor(letter); got != want {
			t.Errorf("cueKeyForColor(%s) = %q, want %q", letter, got, want)
		}
	}
	types := map[domain.CardType]string{
		domain.CardTypeCreature:    "CREATURE",
		domain.CardTypeSorcery:     "SORCERY",
		domain.CardTypeInstant:     "INSTANT",
		domain.CardTypeArtifact:    "ARTIFACT",
		domain.CardTypeLand:        "LAND",
		domain.CardTypeEnchantment: "ENCHANTMENT",
	}
	for cardType, want := range types {
		if got := cueKeyForType(cardType); got != want {
			t.Errorf("cueKeyForType(%s) = %q, want %q", cardType, got, want)
		}
	}
}

// Every toggle on the screen must resolve to a hint, in both states. This is
// what catches a button added without wording behind it.
func TestEveryFilterButtonHasACue(t *testing.T) {
	buttons, err := createFilterButtons()
	if err != nil {
		t.Fatalf("createFilterButtons: %v", err)
	}
	if len(buttons) == 0 {
		t.Fatal("no filter buttons were built")
	}
	for i, fb := range buttons {
		if fb.cue == "" {
			t.Errorf("button %d has no cue key", i)
			continue
		}
		if _, ok := cueTable[fb.cue]; !ok {
			t.Errorf("button %d: cue key %q has no text", i, fb.cue)
		}
	}
}

// An active filter has to read as pressed whatever the cursor is doing. The bug
// this pins: hovering an active toggle used to lift it, so a filter that was on
// looked identical to one that was off as soon as the mouse touched it.
func TestActiveFilterStaysPressedUnderTheCursor(t *testing.T) {
	cases := []struct {
		name    string
		active  bool
		current elements.ButtonState
		want    elements.ButtonState
	}{
		{"active and idle", true, elements.StateNormal, elements.StatePressed},
		{"active under the cursor", true, elements.StateHover, elements.StatePressed},
		{"active mid-click", true, elements.StatePressed, elements.StatePressed},
		{"inactive under the cursor", false, elements.StateHover, elements.StateHover},
		{"inactive and idle", false, elements.StateNormal, elements.StateNormal},
	}
	for _, c := range cases {
		if got := filterDrawState(c.active, c.current); got != c.want {
			t.Errorf("%s: filterDrawState(%v, %v) = %v, want %v",
				c.name, c.active, c.current, got, c.want)
		}
	}
}
