package screens

import (
	"image/color"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
)

// cueCards holds the two hints a filter toggle shows: the text for switching it
// on, then the text for switching it off.
type cueCards map[string][2]string

// These sentences belong to the port, not to the original. Cuecards.txt says
// "Green cards are in the list" and "Green cards are filtered out", which
// described a row where each toggle hid its own color. This screen does not
// filter that way: colors add to each other inside their group and types add to
// each other inside theirs, while a color and a type must both match (see
// collectionFilter.matches). Reusing the original's wording would describe a
// filter this screen does not have, in the one place a player is certain to
// read.
var cueTable = cueCards{
	"WHITE":    {"Click to remove white", "Cards that produce or cost white mana"},
	"BLUE":     {"Click to remove blue", "Cards that produce or cost blue mana"},
	"BLACK":    {"Click to remove black", "Cards that produce or cost black mana"},
	"RED":      {"Click to remove red", "Cards that produce or cost red mana"},
	"GREEN":    {"Click to remove green", "Cards that produce or cost green mana"},
	"CREATURE": {"Click to remove creatures", "Cards that are creatures"},
	"SORCERY":  {"Click to remove sorceries", "Cards that are sorceries"},
	"INSTANT":  {"Click to remove instants", "Cards that are instants"},
	"ARTIFACT": {"Click to remove artifacts", "Cards that are artifacts"},
	"LAND":     {"Click to remove lands", "Cards that are lands"},
}

// cueText returns the hint for a filter toggle, empty when the table has no
// entry for it, so a missing key shows nothing rather than a wrong sentence.
func cueText(cues cueCards, key string, active bool) string {
	pair, ok := cues[key]
	if !ok {
		return ""
	}
	if active {
		return pair[0]
	}
	return pair[1]
}

// The filter buttons name colors by their single letter while the hints spell
// them out; card types match the spelling directly.
func cueKeyForColor(letter string) string {
	switch letter {
	case "W":
		return "WHITE"
	case "U":
		return "BLUE"
	case "B":
		return "BLACK"
	case "R":
		return "RED"
	case "G":
		return "GREEN"
	}
	return ""
}

func cueKeyForType(t domain.CardType) string {
	switch t {
	case domain.CardTypeCreature:
		return "CREATURE"
	case domain.CardTypeSorcery:
		return "SORCERY"
	case domain.CardTypeInstant:
		return "INSTANT"
	case domain.CardTypeArtifact:
		return "ARTIFACT"
	case domain.CardTypeLand:
		return "LAND"
	case domain.CardTypeEnchantment:
		return "ENCHANTMENT"
	}
	return ""
}

// drawFilterCue names the filter the cursor rests on. The icons alone leave the
// player guessing what each button does, and the original answered that with a
// hint line in this spot.
func (s *EditDeckScreen) drawFilterCue(screen *ebiten.Image, W, H int, scale float64) {
	if len(s.filterButtons) == 0 {
		return
	}
	s.layoutFilterButtons(H)
	pos := ui.Position()
	for _, fb := range s.filterButtons {
		if !pos.In(fb.btn.Bounds) {
			continue
		}
		text := cueText(cueTable, fb.cue, fb.isActive(&s.filter))
		if text == "" {
			return
		}
		y := H - COLLECTION_HEIGHT - filterBtnSize - filterGapAboveCarousel - 22
		hint := elements.NewText(14, text, 0, y)
		hint.HAlign = elements.AlignCenter
		hint.BoundsW = float64(W)
		hint.Color = color.RGBA{R: 226, G: 226, B: 236, A: 255}
		hint.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		return
	}
}
