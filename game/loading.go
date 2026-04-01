package game

import (
	"fmt"
	"image/color"
	"sync/atomic"

	"github.com/benprew/s30/game/ui/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// LoadingGame is a lightweight game wrapper that shows a loading screen
// while the real game initializes in the background. This prevents the
// heavy NewGame() initialization from blocking the Android main thread
// during Go runtime init(), which would cause the splash screen to hang.
type LoadingGame struct {
	game    *Game
	ready   atomic.Bool
	initErr error
	dots    int
	ticks   int
}

func NewLoadingGame() *LoadingGame {
	lg := &LoadingGame{}
	go func() {
		g, err := NewGame()
		if err != nil {
			lg.initErr = err
			lg.ready.Store(true)
			return
		}
		lg.game = g
		lg.ready.Store(true)
		fmt.Println("Game initialization complete")
	}()
	return lg
}

func (lg *LoadingGame) Update() error {
	if lg.ready.Load() {
		if lg.initErr != nil {
			return lg.initErr
		}
		return lg.game.Update()
	}
	lg.ticks++
	if lg.ticks%10 == 0 {
		lg.dots = (lg.dots + 1) % 4
	}
	return nil
}

func (lg *LoadingGame) Draw(screen *ebiten.Image) {
	if lg.ready.Load() && lg.game != nil {
		lg.game.Draw(screen)
		return
	}

	screen.Fill(color.Black)

	msg := "Loading"
	for range lg.dots {
		msg += "."
	}

	face := &text.GoTextFace{Source: fonts.MtgFont, Size: 32}
	w, h := text.Measure(msg, face, 0)

	opts := text.DrawOptions{}
	opts.GeoM.Translate((1024-w)/2, (768-h)/2)
	opts.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, face, &opts)
}

func (lg *LoadingGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	if lg.ready.Load() && lg.game != nil {
		return lg.game.Layout(outsideWidth, outsideHeight)
	}
	return 1024, 768
}
