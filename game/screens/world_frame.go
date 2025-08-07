package screens

import (
	"bytes"
	"image"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
)

// This is the frame that you see when you're walking around the world and in cities
//
// World frame shows character stats, current quest, available money, etc
// And has buttons to go to the minimap
// Not technically a screen, but it has draw and update functions, so I'm including it here

type WorldFrame struct {
	button []*elements.Button
	img    *ebiten.Image
}

func NewWorldFrame() (*WorldFrame, error) {
	img, _, err := image.Decode(bytes.NewReader(assets.WorldFrame_png))
	if err != nil {
		return nil, err
	}

	return &WorldFrame{
		img: ebiten.NewImageFromImage(img),
	}, nil

}

func (f *WorldFrame) Draw(screen *ebiten.Image, scale float64) {
	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(f.img, frameOpts)
}

func (f *WorldFrame) Update(W, H int) error {
	return nil
}
