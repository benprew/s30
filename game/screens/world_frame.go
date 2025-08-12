package screens

import (
	"bytes"
	"image"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
)

// This is the frame that you see when you're walking around the world and in cities
//
// World frame shows character stats, current quest, available money, etc
// And has buttons to go to the minimap
// Not technically a screen, but it has draw and update functions, so I'm including it here

// Buybuttons.spr.png - buy buttons
//
// Worlds.spr.png - sprite for world frame
// Compnew.spr.new - compass sprite in world frame
// Questnew.spr.png - quest sprite in world frame
// Clocknew.spr.png - clock sprite
// Daysnew.spr.png - 0-5 for clock
// Days.spr.png - 0-5 for clock
// Prdfrma.pic.png - food/gold/life/cards quest icons - old?
// Statbut1.pic.png - stat buttons - old?

type WorldFrame struct {
	button []*elements.Button
	img    *ebiten.Image
	player *domain.Player // handle to player so we can get player stats
}

func NewWorldFrame(p *domain.Player) (*WorldFrame, error) {
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

func (f *WorldFrame) Update(W, H int, scale float64) error {
	return nil
}
