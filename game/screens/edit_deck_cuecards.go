package screens

import (
	"image/color"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
)

// cueTable is the game's hint table, parsed once. It is small (154 lines) and
// every filter button's hover text comes from it.
var cueTable = parseCuecards(assets.Cuecards_txt)

// The game ships its hover hints as a table of @KEY blocks, each with a count
// line followed by the text for the toggle switched on and the text for it
// switched off. Reading the table instead of hardcoding the sentences keeps the
// port speaking the original's words: "Black cards are in the list" and "Black
// cards are filtered out".
//
// Source: assets/text/Cuecards.txt, byte-identical to the shipped 1.3.2 file.
type cueCards map[string][2]string

func parseCuecards(data []byte) cueCards {
	cues := make(cueCards)
	key := ""
	var lines []string
	flush := func() {
		if key != "" && len(lines) >= 2 {
			cues[key] = [2]string{lines[0], lines[1]}
		}
		key, lines = "", nil
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "@"):
			flush()
			key = strings.TrimPrefix(line, "@")
		case isAllDigits(line):
			// count line; the texts follow
		default:
			if key != "" {
				lines = append(lines, line)
			}
		}
	}
	flush()
	return cues
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// cueText returns the hint for a filter button, empty when the table has no
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

// The filter buttons name colors by their single letter while the cue table
// spells them out; card types match the table's spelling directly.
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

// drawFilterCue names the filter the cursor rests on, using the game's own
// sentence for that toggle's current state. The icons alone leave the player
// guessing what each button does; the original answers that with these texts.
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
