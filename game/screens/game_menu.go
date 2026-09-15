package screens

import (
	"image"
	"image/color"

	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game menu geometry, in the screen's 1024x768 design coordinates. The panel
// hangs from the world frame's menu button, in the upper right.
const (
	gameMenuPanelW    = 300
	gameMenuRowH      = 44
	gameMenuTitleH    = 38
	gameMenuPadBottom = 10
	gameMenuMargin    = 12
)

// gameMenuRow is one entry of the world menu: the label it shows and the screen
// it opens.
type gameMenuRow struct {
	label  string
	target screenui.ScreenName
}

// The original's Escape menu (Advstrings.txt line 4) lists Save, Load, Quit and
// five "See ..." entries. Only the entries that lead somewhere in the port are
// rows here: one that opens nothing would be worse than no row at all. Save and
// Load are not rows yet for the same reason — the port saves with F5 and loads
// from the opening menu, so neither has a screen to open from inside a game.
var gameMenuRows = []gameMenuRow{
	{"Quit", screenui.StartScr},
	{"See/Edit Deck", screenui.EditDeckScr},
	{"See Map", screenui.MiniMapScr},
}

// gameMenuPanelHeight grows with the rows, so adding one cannot push the last
// row past the panel's edge.
func gameMenuPanelHeight() int {
	return gameMenuTitleH + len(gameMenuRows)*gameMenuRowH + gameMenuPadBottom
}

// gameMenuBounds is the panel's rectangle, hanging below the menu button.
func gameMenuBounds(W int) image.Rectangle {
	x := W - gameMenuMargin - gameMenuPanelW
	y := worldMenuButtonSize + 2*gameMenuMargin
	return image.Rect(x, y, x+gameMenuPanelW, y+gameMenuPanelHeight())
}

// gameMenuRowBounds is the rectangle of one row, inside the panel.
func gameMenuRowBounds(i, W int) image.Rectangle {
	panel := gameMenuBounds(W)
	top := panel.Min.Y + gameMenuTitleH + i*gameMenuRowH
	return image.Rect(panel.Min.X, top, panel.Max.X, top+gameMenuRowH)
}

// gameMenuChoice decides what the menu does with a click: open the row that was
// hit, stay open on the panel's dead space, or pop back to the world. It is kept
// out of Update so all three outcomes are testable without Ebiten.
func gameMenuChoice(pos image.Point, clicked bool, W int) screenui.ScreenName {
	if !clicked {
		return screenui.GameMenuScr
	}
	for i, row := range gameMenuRows {
		if pos.In(gameMenuRowBounds(i, W)) {
			return row.target
		}
	}
	if pos.In(gameMenuBounds(W)) {
		return screenui.GameMenuScr
	}
	return screenui.PopScr
}

// GameMenuScreen is the transparent overlay the world frame's menu button opens,
// carrying the entries of the menu the original shows on Escape. The screen
// underneath (the world) is drawn beneath it.
type GameMenuScreen struct {
	panelBg *ebiten.Image
	hoverBg *ebiten.Image
}

func NewGameMenuScreen() *GameMenuScreen {
	panelBg := ebiten.NewImage(gameMenuPanelW, gameMenuPanelHeight())
	panelBg.Fill(color.RGBA{20, 12, 4, 220})

	hoverBg := ebiten.NewImage(1, 1)
	hoverBg.Fill(color.White)

	return &GameMenuScreen{panelBg: panelBg, hoverBg: hoverBg}
}

func (s *GameMenuScreen) IsFramed() bool { return true }

func (s *GameMenuScreen) IsOverlay() bool { return true }

// Update closes the menu on Escape, and otherwise lets a click choose a row.
func (s *GameMenuScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.PopScr, nil, nil
	}
	return gameMenuChoice(ui.Position(), ui.Click(image.Rect(0, 0, W, H)), W), nil, nil
}

func (s *GameMenuScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	panel := gameMenuBounds(W)
	panelOpts := &ebiten.DrawImageOptions{}
	panelOpts.GeoM.Scale(scale, scale)
	panelOpts.GeoM.Translate(float64(panel.Min.X)*scale, float64(panel.Min.Y)*scale)
	screen.DrawImage(s.panelBg, panelOpts)

	title := elements.NewText(20, "Will you ...", panel.Min.X, panel.Min.Y)
	title.Color = color.White
	title.HAlign = elements.AlignCenter
	title.VAlign = elements.AlignMiddle
	title.BoundsW = float64(panel.Dx())
	title.BoundsH = float64(gameMenuTitleH)
	title.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	pos := ui.Position()
	for i, row := range gameMenuRows {
		bounds := gameMenuRowBounds(i, W)
		if pos.In(bounds) {
			hoverOpts := &ebiten.DrawImageOptions{}
			hoverOpts.GeoM.Scale(float64(bounds.Dx())*scale, float64(bounds.Dy())*scale)
			hoverOpts.GeoM.Translate(float64(bounds.Min.X)*scale, float64(bounds.Min.Y)*scale)
			hoverOpts.ColorScale.ScaleAlpha(0.15)
			screen.DrawImage(s.hoverBg, hoverOpts)
		}
		label := elements.NewText(20, row.label, bounds.Min.X, bounds.Min.Y)
		label.Color = color.White
		label.HAlign = elements.AlignCenter
		label.VAlign = elements.AlignMiddle
		label.BoundsW = float64(bounds.Dx())
		label.BoundsH = float64(bounds.Dy())
		label.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}
}
