package duel

import (
	"image/color"

	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

// GameResultScreen is a placeholder ending screen for the final boss result.
type GameResultScreen struct {
	Won bool
}

func NewGameResultScreen(won bool) *GameResultScreen {
	return &GameResultScreen{Won: won}
}

func (s *GameResultScreen) IsFramed() bool { return false }

func (s *GameResultScreen) IsOverlay() bool { return false }

func (s *GameResultScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if s.Won {
		return screenui.GameWinScr, nil, nil
	}
	return screenui.GameLoseScr, nil, nil
}

func (s *GameResultScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.Fill(color.RGBA{R: 15, G: 12, B: 20, A: 255})
	message := "You lost the game"
	textColor := color.RGBA{R: 210, G: 90, B: 90, A: 255}
	if s.Won {
		message = "You won the game"
		textColor = color.RGBA{R: 235, G: 205, B: 90, A: 255}
	}
	result := elements.NewText(54, message, 0, 0)
	result.Color = textColor
	w, h := result.Measure()
	result.X = (W - int(w)) / 2
	result.Y = (H - int(h)) / 2
	result.Draw(screen, &ebiten.DrawImageOptions{}, scale)
}
