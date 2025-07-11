package screens

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets/art"
	"github.com/benprew/s30/assets/fonts"
	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	SCALE = 1.6
)

type CityScreen struct {
	Frame   *ebiten.Image
	BgImage *ebiten.Image
	Buttons []*sprites.Button
}

func NewCityScreen(frame *ebiten.Image) CityScreen {
	return CityScreen{
		Frame:   frame,
		BgImage: nil,
		Buttons: mk_buttons(SCALE - 0.4),
	}
}

func (c *CityScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	cityOpts := &ebiten.DrawImageOptions{}
	cityOpts.GeoM.Scale(SCALE, SCALE)
	cityOpts.GeoM.Translate(100.0, 75.0) // Offset the image

	screen.DrawImage(c.BgImage, cityOpts)
	// Draw the worldFrame over everything
	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(c.Frame, frameOpts)

	for _, b := range c.Buttons {
		b.Draw(screen, frameOpts)
	}

}

func (c *CityScreen) Update() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		fmt.Println("Returned to world map from city")
		return true, nil
	}
	options := &ebiten.DrawImageOptions{}
	for _, b := range c.Buttons {
		b.Update(options)
	}

	return false, nil
}

// Make buttons for City screen
// Iconb and Icons "b" stands for border
func mk_buttons(scale float64) []*sprites.Button {
	Icons, err := sprites.LoadSpriteSheet(12, 2, art.Icons_png)
	if err != nil {
		panic(fmt.Errorf("failed to load icons sprite sheet: %w", err))
	}
	Iconb, err := sprites.LoadSpriteSheet(8, 1, art.Iconb_png)
	if err != nil {
		panic(fmt.Errorf("failed to load iconb sprite sheet: %w", err))
	}

	// Create a font face using ebiten's text v2
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.Magic_ttf))
	if err != nil {
		panic(fmt.Errorf("failed to create font source: %w", err))
	}

	fontFace := &text.GoTextFace{
		Source: fontSource,
		Size:   20,
	}

	buyCardsNorm := combineButton(Iconb[0][0], Icons[0][3], Iconb[0][1], scale)
	buyCardsHover := combineButton(Iconb[0][2], Icons[0][3], Iconb[0][3], scale)
	buyCardsPressed := combineButton(Iconb[0][0], Icons[0][3], Iconb[0][1], scale)

	buyCards := sprites.Button{
		Normal:     buyCardsNorm,
		Hover:      buyCardsHover,
		Pressed:    buyCardsPressed,
		Text:       "Buy Cards",
		Font:       fontFace,
		TextColor:  color.White,
		TextOffset: image.Point{X: int(10 * scale), Y: int(60 * scale)},
		State:      sprites.StateNormal,
		X:          150,
		Y:          100,
	}

	return []*sprites.Button{&buyCards}
}

func combineButton(btnFrame, btnIcon, txtBox *ebiten.Image, scale float64) *ebiten.Image {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	combinedImage := ebiten.NewImage(120, 100)
	combinedImage.DrawImage(btnFrame, op)
	op.GeoM.Translate(8.0*scale, 5.0*scale)
	combinedImage.DrawImage(btnIcon, op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(SCALE, SCALE)
	op.GeoM.Translate(1*scale, 55.0*scale)
	combinedImage.DrawImage(txtBox, op)
	return combinedImage
}
