package screens

import (
	"image/color"

	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type RandomEncounterScreen struct {
	Sprite *ebiten.Image
}

func NewRandomEncounterScreen(sprite *ebiten.Image) *RandomEncounterScreen {
	return &RandomEncounterScreen{
		Sprite: sprite,
	}
}

func (s *RandomEncounterScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil, nil
	}
	return screenui.RandomEncounterScr, nil, nil
}

func (s *RandomEncounterScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	centerX := float64(FrameOffsetX)*scale + (float64(FrameWidth)*scale)/2
	centerY := float64(FrameOffsetY)*scale + (float64(FrameHeight)*scale)/2

	// Draw the encounter sprite if available
	if s.Sprite != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale*2, scale*2) // Make it a bit bigger for the detail screen
		
		// Center the sprite
		w := float64(s.Sprite.Bounds().Dx())
		h := float64(s.Sprite.Bounds().Dy())
		op.GeoM.Translate(-w*scale, -h*scale) // *2/2 = *1
		
		op.GeoM.Translate(centerX, centerY - 50*scale) // Shift up slightly to make room for text
		screen.DrawImage(s.Sprite, op)
	}

	// Draw "Random Encounter" text
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   48,
	}

	textStr := "Random Encounter"
	width, _ := text.Measure(textStr, fontFace, 0)

	x := centerX - width/2
	y := centerY + 100*scale // Below the sprite

	opts := &text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(color.White)
	
	text.Draw(screen, textStr, fontFace, opts)
}

func (s *RandomEncounterScreen) IsFramed() bool {
	return true 
}
