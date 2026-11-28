package screens

import (
	"testing"

	"github.com/benprew/s30/game/ui/elements"
)

func TestCueTextFollowsTheToggleState(t *testing.T) {
	buttons, err := createFilterButtons()
	if err != nil {
		t.Fatal(err)
	}
	filter := newCollectionFilter()
	for _, button := range buttons {
		off := button.cueText(&filter)
		button.toggle(&filter)
		on := button.cueText(&filter)
		if off == "" || on == "" || off == on {
			t.Errorf("button at row %d, column %d: cues = %q and %q", button.row, button.col, off, on)
		}
		button.toggle(&filter)
	}

	white := buttons[0]
	if got := white.cueText(&filter); got != "Cards that produce or cost white mana" {
		t.Errorf("inactive white cue = %q", got)
	}
	white.toggle(&filter)
	if got := white.cueText(&filter); got != "Turn off the white mana filter" {
		t.Errorf("active white cue = %q", got)
	}

	creature := buttons[filterColorCount]
	creature.toggle(&filter)
	if got := creature.cueText(&filter); got != "Turn off the creature filter" {
		t.Errorf("active creature cue = %q", got)
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
