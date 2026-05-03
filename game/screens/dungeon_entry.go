package screens

import (
	"fmt"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type DungeonEntryScreen struct {
	Dungeon *domain.Dungeon
	Player  *domain.Player
	Level   *world.Level
	Buttons []*elements.Button
}

func NewDungeonEntryScreen(dungeon *domain.Dungeon, player *domain.Player, level *world.Level) *DungeonEntryScreen {
	s := &DungeonEntryScreen{
		Dungeon: dungeon,
		Player:  player,
		Level:   level,
	}
	s.setupButtons()
	return s
}

func (s *DungeonEntryScreen) IsFramed() bool { return false }

func (s *DungeonEntryScreen) setupButtons() {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}

	enterW, _ := elements.TextButtonSize("Enter", fontFace)
	leaveW, _ := elements.TextButtonSize("Leave", fontFace)
	totalW := enterW + 20 + leaveW
	startX := 512 - totalW/2

	enter := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: "Enter", Font: fontFace, ID: "enter",
		X: startX, Y: 600,
	})
	leave := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: "Leave", Font: fontFace, ID: "leave",
		X: startX + enterW + 20, Y: 600,
	})
	s.Buttons = []*elements.Button{enter, leave}
}

func (s *DungeonEntryScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	opts := &ebiten.DrawImageOptions{}
	for _, b := range s.Buttons {
		b.Update(opts, scale, W, H)
		if !b.IsClicked() {
			continue
		}
		switch b.ID {
		case "enter":
			s.Player.DungeonState = &domain.DungeonState{
				CurrentDungeon: s.Dungeon,
				Position:       s.Dungeon.Entrance,
				DungeonLife:    s.Player.Life,
			}
			s.Dungeon.RevealFrom(s.Dungeon.Entrance)
			return screenui.DungeonScr, NewDungeonScreen(s.Player, s.Level), nil
		case "leave":
			return screenui.WorldScr, nil, nil
		}
	}
	return screenui.DungeonEntryScr, nil, nil
}

func (s *DungeonEntryScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.Fill(color.RGBA{R: 12, G: 8, B: 20, A: 255})

	title := elements.NewText(36, fmt.Sprintf("You stand before %s", s.Dungeon.Name), 50, 80)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	subtitle := elements.NewText(20, fmt.Sprintf("A %s dungeon", domain.ColorMaskToString(s.Dungeon.Color)), 50, 130)
	subtitle.Color = color.RGBA{R: 200, G: 200, B: 220, A: 255}
	subtitle.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	clueY := 220
	heading := elements.NewText(24, "Known clues:", 50, clueY)
	heading.Color = color.White
	heading.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	clueY += 40

	any := false
	for _, c := range s.Dungeon.Clues {
		if !c.Revealed || c.Text == "" {
			continue
		}
		t := elements.NewText(20, "  - "+c.Text, 50, clueY)
		t.Color = color.RGBA{R: 220, G: 220, B: 240, A: 255}
		t.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		clueY += 32
		any = true
	}
	if !any {
		t := elements.NewText(20, "  (none — you have no clues about this place)", 50, clueY)
		t.Color = color.RGBA{R: 160, G: 160, B: 180, A: 255}
		t.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	for _, b := range s.Buttons {
		b.Draw(screen, opts, scale)
	}
}
