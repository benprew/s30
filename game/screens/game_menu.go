package screens

import (
	"image"
	"image/color"

	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game menu geometry, in the screen's 1024x768 design coordinates. The panel
// hangs from the world frame's menu button, in the frame's upper right corner.
const (
	gameMenuPanelW    = 300
	gameMenuRowH      = 44
	gameMenuTitleH    = 38
	gameMenuPadBottom = 10
	gameMenuMargin    = 12
	gameMenuTitlePadX = 16

	gameMenuTitle        = "Will you ..."
	gameMenuConfirmTitle = "Ready to Quit?"
)

// menuStep is which face of the menu is showing: the list of entries, or the
// question the original puts behind Quit.
type menuStep int

const (
	menuStepRows menuStep = iota
	menuStepConfirmQuit
)

// gameMenuRow is one entry of the menu: the label it shows, the screen it opens,
// and whether it asks before going there.
type gameMenuRow struct {
	label    string
	target   screenui.ScreenName
	confirms bool
}

// The original's Escape menu (Advstrings.txt line 4) lists Save, Load, Quit and
// five "See ..." entries. Only the entries that lead somewhere from the map are
// rows here: one that opens nothing would be worse than no row at all.
//
// See/Edit Deck is not a row even though the screen exists. It is built with the
// city the player is standing in (city.go) and that is the only way to reach it,
// so from the map there is no screen to open at all. Whether the original let you
// open it on the road, and at what price, is a separate question.
var gameMenuRows = []gameMenuRow{
	{"Save", screenui.PopScr, false},
	{"Load", screenui.LoadGameScr, false},
	{"Quit", screenui.QuitScr, true},
	{"See Map", screenui.MiniMapScr, false},
}

// The answers behind Quit, in the original's wording and order (Advstrings.txt
// line 9: "Ready to Quit?\n No.\n Yes."). No keeps the game, Yes leaves it, and
// a click past the question counts as No.
var gameMenuConfirmRows = []gameMenuRow{
	{"No.", screenui.GameMenuScr, false},
	{"Yes.", screenui.QuitScr, false},
}

// gameMenuPanelHeight fits the taller of the two faces, so neither can push a row
// past the panel's edge.
func gameMenuPanelHeight() int {
	rows := max(len(gameMenuConfirmRows), len(gameMenuRows))
	return gameMenuTitleH + rows*gameMenuRowH + gameMenuPadBottom
}

// gameMenuBounds is the panel's rectangle, hanging below the menu button so the
// button stays clickable while the menu is open.
func gameMenuBounds(W int) image.Rectangle {
	btn := worldMenuButtonBounds(W)
	x := W - gameMenuMargin - gameMenuPanelW
	y := btn.Max.Y + gameMenuMargin
	return image.Rect(x, y, x+gameMenuPanelW, y+gameMenuPanelHeight())
}

// gameMenuRowBounds is the rectangle of one row, inside the panel. Both faces use
// the same rows, so a face decides what a row says, not where it is.
func gameMenuRowBounds(i, W int) image.Rectangle {
	panel := gameMenuBounds(W)
	top := panel.Min.Y + gameMenuTitleH + i*gameMenuRowH
	return image.Rect(panel.Min.X, top, panel.Max.X, top+gameMenuRowH)
}

// gameMenuChoice decides what a click does. It returns the face to show and the
// screen to open, so every path - a row, the question, the panel's dead space and
// the world behind it - is testable without Ebiten.
func gameMenuChoice(pos image.Point, clicked bool, W int, step menuStep) (menuStep, screenui.ScreenName) {
	if !clicked {
		return step, screenui.GameMenuScr
	}
	if step == menuStepConfirmQuit {
		for i, row := range gameMenuConfirmRows {
			if pos.In(gameMenuRowBounds(i, W)) {
				return menuStepRows, row.target
			}
		}
		return menuStepRows, screenui.GameMenuScr
	}
	for i, row := range gameMenuRows {
		if !pos.In(gameMenuRowBounds(i, W)) {
			continue
		}
		if row.confirms {
			return menuStepConfirmQuit, screenui.GameMenuScr
		}
		return menuStepRows, row.target
	}
	if pos.In(gameMenuBounds(W)) {
		return menuStepRows, screenui.GameMenuScr
	}
	return menuStepRows, screenui.PopScr
}

// GameMenuScreen is the transparent overlay the world frame's menu button opens,
// carrying the entries of the menu the original shows on Escape. The screen
// underneath (the world) is drawn beneath it.
type GameMenuScreen struct {
	panelBg *ebiten.Image
	hoverBg *ebiten.Image
	step    menuStep
	onSave  func() error
	open    bool
}

func NewGameMenuScreen(onSave ...func() error) *GameMenuScreen {
	panelBg := ebiten.NewImage(gameMenuPanelW, gameMenuPanelHeight())
	panelBg.Fill(color.RGBA{20, 12, 4, 220})

	hoverBg := ebiten.NewImage(1, 1)
	hoverBg.Fill(color.White)

	var saveFn func() error
	if len(onSave) > 0 {
		saveFn = onSave[0]
	}

	return &GameMenuScreen{panelBg: panelBg, hoverBg: hoverBg, onSave: saveFn}
}

func (s *GameMenuScreen) IsOpen() bool { return s.open }

func (s *GameMenuScreen) Open() {
	s.open = true
	s.step = menuStepRows
}

func (s *GameMenuScreen) Close() {
	s.open = false
	s.step = menuStepRows
}

func (s *GameMenuScreen) IsFramed() bool { return false }

func (s *GameMenuScreen) IsOverlay() bool { return true }

func (s *GameMenuScreen) handleChoice(pos image.Point, clicked bool, W int) (screenui.ScreenName, screenui.Screen) {
	if clicked && s.step == menuStepRows {
		for i, row := range gameMenuRows {
			if pos.In(gameMenuRowBounds(i, W)) {
				if row.label == "Save" {
					if s.onSave != nil {
						_ = s.onSave()
					}
					if am := gameaudio.Get(); am != nil {
						am.PlaySFX(gameaudio.SFXClick2)
					}
					return screenui.PopScr, nil
				}
				if row.label == "Load" {
					if am := gameaudio.Get(); am != nil {
						am.PlaySFX(gameaudio.SFXClick2)
					}
					return screenui.LoadGameScr, NewLoadGameScreen()
				}
			}
		}
	}
	step, name := gameMenuChoice(pos, clicked, W, s.step)
	s.step = step
	return name, nil
}

// UpdateMenu runs the menu. When closed, clicking the button or pressing Escape
// (if allowed) opens the menu. When open, Escape leaves the question or closes
// the menu.
func (s *GameMenuScreen) UpdateMenu(W, H int, scale float64, allowEscapeOpen bool) (screenui.ScreenName, screenui.Screen, error) {
	if !s.open {
		btn := gameMenuButtonBounds(W)
		if ui.Click(btn) || (allowEscapeOpen && inpututil.IsKeyJustPressed(ebiten.KeyEscape)) {
			if am := gameaudio.Get(); am != nil {
				am.PlaySFX(gameaudio.SFXClick2)
			}
			s.Open()
			return screenui.GameMenuScr, nil, nil
		}
		return screenui.NoScr, nil, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if s.step == menuStepConfirmQuit {
			s.step = menuStepRows
			return screenui.GameMenuScr, nil, nil
		}
		s.Close()
		return screenui.PopScr, nil, nil
	}

	name, scr := s.handleChoice(ui.Position(), ui.Click(image.Rect(0, 0, W, H)), W)
	if name == screenui.QuitScr {
		s.Close()
		return screenui.QuitScr, nil, ebiten.Termination
	}
	if name == screenui.PopScr || name == screenui.LoadGameScr || name == screenui.MiniMapScr {
		s.Close()
	}
	return name, scr, nil
}

// Update implements screenui.Screen.
func (s *GameMenuScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	return s.UpdateMenu(W, H, scale, true)
}

// titleAndRows is the face showing now: the list of entries, or the question
// behind Quit.
func (s *GameMenuScreen) titleAndRows() (string, []string) {
	if s.step == menuStepConfirmQuit {
		labels := make([]string, 0, len(gameMenuConfirmRows))
		for _, row := range gameMenuConfirmRows {
			labels = append(labels, row.label)
		}
		return gameMenuConfirmTitle, labels
	}
	labels := make([]string, 0, len(gameMenuRows))
	for _, row := range gameMenuRows {
		labels = append(labels, row.label)
	}
	return gameMenuTitle, labels
}

func gameMenuTitleElement(panel image.Rectangle, titleText string) *elements.Text {
	title := elements.NewText(20, titleText, panel.Min.X+gameMenuTitlePadX, panel.Min.Y)
	title.Color = color.White
	title.HAlign = elements.AlignLeft
	title.VAlign = elements.AlignMiddle
	title.BoundsW = float64(panel.Dx() - gameMenuTitlePadX)
	title.BoundsH = float64(gameMenuTitleH)
	return title
}

func (s *GameMenuScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	btn := gameMenuButtonBounds(W)
	drawGameMenuButton(screen, btn, scale)

	if !s.open {
		return
	}

	panel := gameMenuBounds(W)
	panelOpts := &ebiten.DrawImageOptions{}
	panelOpts.GeoM.Scale(scale, scale)
	panelOpts.GeoM.Translate(float64(panel.Min.X)*scale, float64(panel.Min.Y)*scale)
	screen.DrawImage(s.panelBg, panelOpts)

	titleText, labels := s.titleAndRows()
	title := gameMenuTitleElement(panel, titleText)
	title.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	pos := ui.Position()
	for i, label := range labels {
		bounds := gameMenuRowBounds(i, W)
		if pos.In(bounds) {
			hoverOpts := &ebiten.DrawImageOptions{}
			hoverOpts.GeoM.Scale(float64(bounds.Dx())*scale, float64(bounds.Dy())*scale)
			hoverOpts.GeoM.Translate(float64(bounds.Min.X)*scale, float64(bounds.Min.Y)*scale)
			hoverOpts.ColorScale.ScaleAlpha(0.15)
			screen.DrawImage(s.hoverBg, hoverOpts)
		}
		row := elements.NewText(20, label, bounds.Min.X, bounds.Min.Y)
		row.Color = color.White
		row.HAlign = elements.AlignCenter
		row.VAlign = elements.AlignMiddle
		row.BoundsW = float64(bounds.Dx())
		row.BoundsH = float64(bounds.Dy())
		row.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}
}
