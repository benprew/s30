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

func newArenaMessageScreen(champion bool) *ArenaEntryScreen {
	background, _ := imageutil.LoadImage(assets.DuelWinBg_png)
	panel := ebiten.NewImage(924, 230)
	panel.Fill(color.RGBA{R: 12, G: 8, B: 20, A: 220})
	s := &ArenaEntryScreen{champion: champion, background: background, panel: panel}
	labels := []string{"Enter Arena (300 gold)", "Leave Quietly"}
	if champion {
		labels = []string{"Return to the Overworld"}
	}
	for i, label := range labels {
		s.buttons = append(s.buttons, arenaButton(label, i, 500+70*i))
	}
	return s
}

func arenaButton(label string, index, y int) *elements.Button {
	sprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	face := &text.GoTextFace{Source: fonts.MtgFont, Size: 24}
	width, _ := elements.TextButtonSize(label, face)
	return elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: sprites[0][0], Hover: sprites[0][1], Pressed: sprites[0][2],
		Text: label, Font: face, ID: fmt.Sprintf("arena-%d", index), X: 512 - width/2, Y: y,
	})
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
	if !s.champion && inpututil.IsKeyJustPressed(ebiten.Key1) {
		return s.enter(W, H)
	}
	for i, button := range s.buttons {
		if !s.champion && i == 0 && s.level.Player.Gold < domain.ArenaEntryCost {
			button.State = elements.StateDisabled
			button.ButtonText.TextColor = color.RGBA{R: 150, G: 150, B: 150, A: 255}
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
	panelOpts := &ebiten.DrawImageOptions{}
	panelOpts.GeoM.Translate(50, 100)
	screen.DrawImage(s.panel, panelOpts)
	title := "Welcome to the Arena"
	body := "Challenge all comers and hear the roar of the crowd!\nBuild a deck from a booster and basic lands. Each win brings\nnew cards and prizes. Win seven rounds to become champion.\nYour arena deck stays here; your earned prizes go with you."
	if s.champion {
		title = "Champion of the Arena"
		body = "Seven victories! The crowd cheers your triumph.\nYou leave with the prizes from every round."
	}
	titleText := elements.NewText(44, title, 0, 110)
	titleText.HAlign, titleText.BoundsW = elements.AlignCenter, float64(W)
	titleText.Draw(screen, &ebiten.DrawImageOptions{}, 1)
	elements.NewText(24, body, 80, 190).Draw(screen, &ebiten.DrawImageOptions{}, 1)
	if !s.champion {
		info := fmt.Sprintf("Entry: %d gold. You have %d gold.", domain.ArenaEntryCost, s.level.Player.Gold)
		if s.level.Player.Gold < domain.ArenaEntryCost {
			info += " You need more gold to enter."
		}
		elements.NewText(22, info, 80, 400).Draw(screen, &ebiten.DrawImageOptions{}, 1)
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
