package screens

import (
	"fmt"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
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

const (
	FrameOffsetX = 100
	FrameOffsetY = 75
	FrameWidth   = 820
	FrameHeight  = 425
)

type WorldFrame struct {
	Buttons       []*elements.Button
	Text          []*elements.Text
	img           *ebiten.Image
	player        *domain.Player // handle to player so we can get player stats
	amuletSprites []*ebiten.Image
}

func NewWorldFrame(p *domain.Player) (*WorldFrame, error) {
	img, err := imageutil.LoadImage(assets.WorldFrame_png)
	if err != nil {
		return nil, err
	}
	worldSprs, err := imageutil.LoadSpriteSheet(12, 5, assets.WorldSpr_png)
	if err != nil {
		return nil, err
	}

	amuletSprs, err := imageutil.LoadSpriteSheet(5, 1, assets.Amsprite_png)
	if err != nil {
		return nil, err
	}

	return &WorldFrame{
		img:           img,
		Buttons:       mkWfButtons(worldSprs),
		player:        p,
		amuletSprites: amuletSprs[0],
	}, nil
}

func (f *WorldFrame) Draw(screen *ebiten.Image, scale float64) {
	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(f.img, frameOpts)

	for _, b := range f.Buttons {
		b.Draw(screen, frameOpts, scale)
	}

	for _, t := range f.Text {
		t.Draw(screen, frameOpts, scale)
	}

	amuletPositions := []int{125, 250, 375, 500, 625}
	amuletY := 628
	for i, sprite := range f.amuletSprites {
		if i < len(amuletPositions) {
			amuletOpts := &ebiten.DrawImageOptions{}
			amuletOpts.GeoM.Scale(scale, scale)
			amuletOpts.GeoM.Translate(float64(amuletPositions[i])*scale, float64(amuletY)*scale)
			screen.DrawImage(sprite, amuletOpts)
		}
	}

}

func (f *WorldFrame) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	options := &ebiten.DrawImageOptions{}
	for i := range f.Buttons {
		b := f.Buttons[i]
		b.Update(options, scale, W, H)
		if b.ID == "minimap" && b.IsClicked() {
			return screenui.MiniMapScr, nil
		}
	}

	f.Text = mkWfText(f.player)

	return -1, nil
}

func mkWfButtons(worldSprs [][]*ebiten.Image) []*elements.Button {
	sidebar := []string{"book", "minimap", "dungeon", "character"}
	buttons := []*elements.Button{}
	for i, n := range sidebar {
		offset := i * 90
		y := 110 + offset
		normalImg := worldSprs[4][i*2]
		btn := elements.NewButton(normalImg, worldSprs[4][i*2+1], normalImg, 8, y, 1.7)
		btn.ID = n
		buttons = append(buttons, btn)
	}
	return buttons
}

func mkWfText(p *domain.Player) []*elements.Text {
	amuletCounts := p.GetAmuletCount()

	texts := []*elements.Text{
		elements.NewText(30, fmt.Sprintf("%d", p.Gold), 140, 560),
		elements.NewText(30, fmt.Sprintf("%d", p.Food), 270, 560),
		elements.NewText(30, fmt.Sprintf("%d", p.Life), 400, 560),
		elements.NewText(30, fmt.Sprintf("%d", p.NumCards()), 530, 560),
	}

	amuletColors := []domain.ColorMask{
		domain.ColorWhite,
		domain.ColorBlue,
		domain.ColorBlack,
		domain.ColorRed,
		domain.ColorGreen,
	}
	amuletPositions := []int{125, 250, 375, 500, 625}
	amuletY := 648

	for i, color := range amuletColors {
		if i < len(amuletPositions) {
			count := amuletCounts[color]
			x := amuletPositions[i] + 20
			texts = append(texts, elements.NewText(18, fmt.Sprintf("%d", count), x, amuletY))
		}
	}

	return texts
}
