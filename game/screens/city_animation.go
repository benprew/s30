package screens

import (
	"image/color"
	"math"

	"github.com/benprew/s30/game/timing"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

const (
	cityEntranceAnimationFrames = 1 * timing.UpdatesPerSecond
	cityEntranceCellSize        = 8
	cityEntranceGlowWidth       = 0.12
)

type cityEntranceAnimation struct {
	tween    *gween.Tween
	progress float32
	complete bool
}

func newCityEntranceAnimation() cityEntranceAnimation {
	return cityEntranceAnimation{
		tween: gween.New(0, 1, cityEntranceAnimationFrames, ease.OutCubic),
	}
}

func (a *cityEntranceAnimation) update() {
	if a.complete {
		return
	}
	a.progress, a.complete = a.tween.Update(1)
}

func (a *cityEntranceAnimation) draw(screen *ebiten.Image) {
	if a.complete {
		return
	}
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	cols := (w + cityEntranceCellSize - 1) / cityEntranceCellSize
	rows := (h + cityEntranceCellSize - 1) / cityEntranceCellSize
	for y := range rows {
		for x := range cols {
			threshold := cityEntranceCellThreshold(x, y)
			if a.progress < threshold {
				vector.FillRect(screen,
					float32(x*cityEntranceCellSize), float32(y*cityEntranceCellSize),
					cityEntranceCellSize, cityEntranceCellSize,
					color.RGBA{8, 5, 14, 255}, false)
				continue
			}
			if a.progress < threshold+cityEntranceGlowWidth {
				alpha := uint8(190 * (1 - (a.progress-threshold)/cityEntranceGlowWidth))
				vector.FillRect(screen,
					float32(x*cityEntranceCellSize), float32(y*cityEntranceCellSize),
					cityEntranceCellSize, cityEntranceCellSize,
					color.RGBA{225, 174, 74, alpha}, false)
			}
		}
	}
}

func cityEntranceCellThreshold(x, y int) float32 {
	n := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca77 ^ 0xc2b2ae3d
	n ^= n >> 16
	n *= 0x7feb352d
	n ^= n >> 15
	random := float32(n&0xffff) / 65535

	dx := float64(x) - 15.5
	dy := (float64(y) - 11.5) * 1.3
	distance := float32(math.Min(math.Hypot(dx, dy)/22, 1))
	return 0.65*random + 0.35*distance
}
