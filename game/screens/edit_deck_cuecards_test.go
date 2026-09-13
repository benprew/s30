package screens

import (
	"testing"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
)

// The cue texts come from the game's own table, so the parser has to read every
// entry and each entry has to carry both halves: the text for a toggle that is on
// and the text for one that is off.
func TestParseCuecardsReadsEveryEntry(t *testing.T) {
	cues := parseCuecards(assets.Cuecards_txt)
	if len(cues) < 30 {
		t.Fatalf("expected at least 30 cue entries, got %d", len(cues))
	}
	for key, pair := range cues {
		if pair[0] == "" || pair[1] == "" {
			t.Errorf("%s: want both an on and an off text, got %q", key, pair)
		}
		if pair[0] == pair[1] {
			t.Errorf("%s: on and off texts are identical (%q)", key, pair[0])
		}
	}
}

func TestCueTextFollowsTheToggleState(t *testing.T) {
	cues := parseCuecards(assets.Cuecards_txt)
	cases := []struct {
		key    string
		active bool
		want   string
	}{
		{"WHITE", true, "White cards are in the list"},
		{"WHITE", false, "White cards are filtered out"},
		{"CREATURE", true, "Creature cards are in the list"},
		{"CREATURE", false, "Creature cards are filtered out"},
		{"CASTCOST", true, "Cards are filtered by cast cost"},
		{"CASTCOST", false, "Cards are not filtered by cast cost"},
	}
	for _, c := range cases {
		if got := cueText(cues, c.key, c.active); got != c.want {
			t.Errorf("cueText(%s, %v) = %q, want %q", c.key, c.active, got, c.want)
		}
	}
}

func TestCueTextIsEmptyForAnUnknownKey(t *testing.T) {
	cues := parseCuecards(assets.Cuecards_txt)
	if got := cueText(cues, "NOT_A_FILTER", true); got != "" {
		t.Errorf("cueText for an unknown key = %q, want empty", got)
	}
	if got := cueText(cues, "", true); got != "" {
		t.Errorf("cueText for an empty key = %q, want empty", got)
	}
}

// The filter names it in single letters while the cue table spells them out, so
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

// Every button on the screen must resolve to a cue in the table. This is what
// catches a key that is spelled differently from the game's file.
func TestEveryFilterButtonHasACueInTheTable(t *testing.T) {
	cues := parseCuecards(assets.Cuecards_txt)
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
		if _, ok := cues[fb.cue]; !ok {
			t.Errorf("button %d: cue key %q is not in Cuecards.txt", i, fb.cue)
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
