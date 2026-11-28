package screens

import (
	"image/color"

	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
)

func (c filterCell) cueText(filter *collectionFilter) string {
	if c.isActive(filter) {
		return c.activeCue
	}
	return c.inactiveCue
}

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
		text := fb.cueText(&s.filter)
		y := filterRowTop(H) - deckHintGap
		hint := elements.NewText(14, text, 0, y)
		hint.HAlign = elements.AlignCenter
		hint.BoundsW = float64(W)
		hint.Color = color.RGBA{R: 226, G: 226, B: 236, A: 255}
		hint.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		return
	}
}
