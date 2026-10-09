package screens

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"time"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	duelscreen "github.com/benprew/s30/game/screens/duel"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)


type ArenaEntryScreen struct {
	level      *world.Level
	tile       image.Point
	background *ebiten.Image
	panel      *ebiten.Image
	buttons    []*elements.Button
	champion   bool
}

// NewArenaEntryScreen offers entry or a basic land at a generated arena.
func NewArenaEntryScreen(level *world.Level, encounter world.RandomEncounter) *ArenaEntryScreen {
	s := newArenaMessageScreen(false)
	s.level, s.tile = level, encounter.Tile
	return s
}

// NewArenaChampionScreen shows the victory celebration after winning 7 rounds.
func NewArenaChampionScreen() *ArenaEntryScreen {
	return newArenaMessageScreen(true)
}


func newArenaMessageScreen(champion bool) *ArenaEntryScreen {
	background, _ := imageutil.LoadImage(assets.DuelWinBg_png)
	panelW, panelH := 820, 310
	if champion {
		panelW, panelH = 740, 230
	}
	panel := ebiten.NewImage(panelW, panelH)
	panel.Fill(color.RGBA{R: 14, G: 10, B: 24, A: 235})
	s := &ArenaEntryScreen{champion: champion, background: background, panel: panel}
	labels := []string{fmt.Sprintf("1. Enter Arena (%d gold)", domain.ArenaEntryCost), "2. Leave Quietly"}
	if champion {
		labels = []string{"Return to the Overworld"}
	}

	face := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}
	maxWidth := 0
	for _, label := range labels {
		w, _ := elements.TextButtonSize(label, face)
		if w > maxWidth {
			maxWidth = w
		}
	}
	buttonW := max(maxWidth, 280)
	btnX := (1024 - buttonW) / 2
	btnY := 440

	for i, label := range labels {
		s.buttons = append(s.buttons, arenaButton(label, i, btnX, btnY+58*i, buttonW, face))
	}
	return s
}

func arenaButton(label string, index, x, y, width int, face *text.GoTextFace) *elements.Button {
	sprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	btn := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: sprites[0][0], Hover: sprites[0][1], Pressed: sprites[0][2],
		Text: label, Font: face, ID: fmt.Sprintf("arena-%d", index), X: x, Y: y,
	})
	if btn.Bounds.Dx() != width {
		scaleX := float64(width) / float64(btn.Bounds.Dx())
		btn.Normal = imageutil.ScaleImageInd(btn.Normal, scaleX, 1.0)
		btn.Hover = imageutil.ScaleImageInd(btn.Hover, scaleX, 1.0)
		btn.Pressed = imageutil.ScaleImageInd(btn.Pressed, scaleX, 1.0)
		btn.Bounds = image.Rect(x, y, x+width, btn.Bounds.Max.Y)
	}
	return btn
}

func (s *ArenaEntryScreen) IsFramed() bool  { return false }
func (s *ArenaEntryScreen) IsOverlay() bool { return false }

func (s *ArenaEntryScreen) enter(W, H int) (screenui.ScreenName, screenui.Screen, error) {
	if s.level.Player.Gold < domain.ArenaEntryCost {
		return screenui.RandomEncounterScr, nil, nil
	}
	run, err := domain.NewArenaRun(s.level.Player, rand.New(rand.NewSource(time.Now().UnixNano())))
	if err != nil {
		return screenui.RandomEncounterScr, nil, err
	}
	session := &arenaSession{run: run, level: s.level, tile: s.tile}
	return session.nextEditor(W, H)
}

func (s *ArenaEntryScreen) leave() (screenui.ScreenName, screenui.Screen, error) {
	if !s.champion {
		if land := domain.RandomBasicLand(); land != nil {
			s.level.Player.CardCollection.AddCard(land, 1)
		}
		s.level.CompleteRandomEncounter(s.tile)
	}
	return screenui.WorldScr, nil, nil
}

func (s *ArenaEntryScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.Key2) ||
		(s.champion && inpututil.IsKeyJustPressed(ebiten.KeySpace)) {
		return s.leave()
	}
	if !s.champion && inpututil.IsKeyJustPressed(ebiten.Key1) && s.level.Player.Gold >= domain.ArenaEntryCost {
		return s.enter(W, H)
	}
	for i, button := range s.buttons {
		if !s.champion && i == 0 && s.level.Player.Gold < domain.ArenaEntryCost {
			button.State = elements.StateDisabled
			button.ButtonText.TextColor = color.RGBA{R: 140, G: 140, B: 140, A: 255}
			continue
		}
		button.ButtonText.TextColor = color.White
		button.Update(&ebiten.DrawImageOptions{}, scale, W, H)
		if button.IsClicked() {
			if s.champion || i == 1 {
				return s.leave()
			}
			return s.enter(W, H)
		}
	}
	return screenui.RandomEncounterScr, nil, nil
}

func (s *ArenaEntryScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(float64(W)/float64(s.background.Bounds().Dx()), float64(H)/float64(s.background.Bounds().Dy()))
	screen.DrawImage(s.background, opts)

	panelW, panelH := s.panel.Bounds().Dx(), s.panel.Bounds().Dy()
	panelX := (W - panelW) / 2
	panelY := 85
	if s.champion {
		panelY = 145
	}

	panelOpts := &ebiten.DrawImageOptions{}
	panelOpts.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(s.panel, panelOpts)
	vector.StrokeRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), 2, color.RGBA{135, 115, 80, 220}, false)

	title := "Welcome to the Arena"
	body := "Challenge all comers and hear the roar of the crowd!\nBuild a deck from a booster and basic lands. Each win brings\nnew cards and prizes. Win seven rounds to become champion.\nYour arena deck stays here; your earned prizes go with you."
	if s.champion {
		title = "Champion of the Arena"
		body = "Seven victories! The crowd cheers your triumph.\nYou leave with the prizes from every round."
	}

	titleText := elements.NewText(38, title, 0, panelY+28)
	titleText.Color = color.RGBA{R: 255, G: 230, B: 150, A: 255}
	titleText.HAlign, titleText.BoundsW = elements.AlignCenter, float64(W)
	titleText.Draw(screen, &ebiten.DrawImageOptions{}, 1)

	if s.champion {
		bodyText := elements.NewText(24, body, 0, panelY+105)
		bodyText.Color = color.RGBA{R: 230, G: 230, B: 240, A: 255}
		bodyText.HAlign, bodyText.BoundsW = elements.AlignCenter, float64(W)
		bodyText.Draw(screen, &ebiten.DrawImageOptions{}, 1)
	} else {
		bodyText := elements.NewText(22, body, panelX+45, panelY+95)
		bodyText.Color = color.RGBA{R: 230, G: 230, B: 240, A: 255}
		bodyText.Draw(screen, &ebiten.DrawImageOptions{}, 1)

		var infoText *elements.Text
		if s.level.Player.Gold >= domain.ArenaEntryCost {
			msg := fmt.Sprintf("Entry Fee: %d gold      Your Gold: %d", domain.ArenaEntryCost, s.level.Player.Gold)
			infoText = elements.NewText(20, msg, 0, panelY+245)
			infoText.Color = color.RGBA{R: 255, G: 230, B: 150, A: 255}
		} else {
			msg := fmt.Sprintf("Entry Fee: %d gold      Your Gold: %d (Need %d more gold)", domain.ArenaEntryCost, s.level.Player.Gold, domain.ArenaEntryCost-s.level.Player.Gold)
			infoText = elements.NewText(20, msg, 0, panelY+245)
			infoText.Color = color.RGBA{R: 255, G: 120, B: 120, A: 255}
		}
		infoText.HAlign, infoText.BoundsW = elements.AlignCenter, float64(W)
		infoText.Draw(screen, &ebiten.DrawImageOptions{}, 1)
	}

	for _, button := range s.buttons {
		button.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}
}


type arenaSession struct {
	run   *domain.ArenaRun
	level *world.Level
	tile  image.Point
}

func (s *arenaSession) nextEditor(W, H int) (screenui.ScreenName, screenui.Screen, error) {
	enemy, err := s.run.NewOpponent()
	if err != nil {
		return screenui.EditDeckScr, nil, err
	}
	if err := enemy.Character.LoadImages(); err != nil {
		return screenui.EditDeckScr, nil, err
	}
	var ante *DuelAnteScreen
	edit := func(w, h int) (screenui.ScreenName, screenui.Screen, error) {
		return s.editor(ante, w, h)
	}
	ante = duelscreen.NewArenaDuelAnteScreen(s.run.Player, enemy, s.run.Round, s.outcome, s.leave, edit)
	return s.editor(ante, W, H)
}

func (s *arenaSession) editor(ante *DuelAnteScreen, W, H int) (screenui.ScreenName, screenui.Screen, error) {
	editor, err := NewEditDeckScreen(s.run.Player, nil, W, H)
	if err != nil {
		return screenui.EditDeckScr, nil, err
	}
	editor.DisableSelling = true
	editor.ReturnScr, editor.ReturnScreen = screenui.DuelAnteScr, ante
	return screenui.EditDeckScr, editor, nil
}

func (s *arenaSession) leave() (screenui.ScreenName, screenui.Screen, error) {
	s.run.Finish()
	s.level.CompleteRandomEncounter(s.tile)
	return screenui.WorldScr, nil, nil
}

func (s *arenaSession) outcome(won bool) (screenui.ScreenName, screenui.Screen, error) {
	if !won {
		s.leave()
		return screenui.DuelLoseScr, NewDuelLoseScreen(nil), nil
	}
	reward := s.run.Win()
	win := NewWinDuelScreen(s.run.Campaign, reward, nil)
	if s.run.Champion {
		s.level.CompleteRandomEncounter(s.tile)
		win.ReturnScr, win.ReturnScreen = screenui.RandomEncounterScr, newArenaMessageScreen(true)
	} else {
		win.SetContinueText(fmt.Sprintf("Go to Round %d", s.run.Round))
		name, editor, err := s.nextEditor(1024, 768)
		if err != nil {
			return screenui.DuelWinScr, nil, err
		}
		win.ReturnScr, win.ReturnScreen = name, editor
	}
	return screenui.DuelWinScr, win, nil
}
