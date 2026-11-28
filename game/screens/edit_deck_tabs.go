package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const deckTabWidth = 136

func loadDeckTabImages() ([domain.MaxPlayerDecks]*ebiten.Image, error) {
	var images [domain.MaxPlayerDecks]*ebiten.Image
	art := [domain.MaxPlayerDecks][]byte{assets.EditDeckBtn1_png, assets.EditDeckBtn2_png, assets.EditDeckBtn3_png}
	for slot, data := range art {
		img, err := imageutil.LoadImage(data)
		if err != nil {
			return images, fmt.Errorf("load deck button %d: %w", slot+1, err)
		}
		size := img.Bounds().Size()
		images[slot] = imageutil.ScaleImageInd(img, float64(deckTabWidth)/float64(size.X), float64(filterBtnSize)/float64(size.Y))
	}
	return images, nil
}

func deckTabBounds(slot, W, H int) image.Rectangle {
	const gap = 8
	const totalWidth = domain.MaxPlayerDecks*deckTabWidth + (domain.MaxPlayerDecks-1)*gap
	startX := editDeckBackBounds(W, H).Min.X - 10 - totalWidth
	x := startX + slot*(deckTabWidth+gap)
	y := filterRowTop(H)
	return image.Rect(x, y, x+deckTabWidth, y+filterBtnSize)
}

func (s *EditDeckScreen) drawDeckTabs(screen *ebiten.Image, W, H int, scale float64) {
	gold := color.RGBA{R: 245, G: 215, B: 110, A: 255}
	for slot, count := range s.deckCardCounts() {
		bounds := deckTabBounds(slot, W, H)
		active := slot == s.Player.ActiveDeck
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(float64(bounds.Min.X)*scale, float64(bounds.Min.Y)*scale)
		if !active {
			opts.ColorScale.ScaleAlpha(0.60)
		}
		screen.DrawImage(s.deckTabImages[slot], opts)

		const countInset = 103
		label := elements.NewText(12, fmt.Sprintf("(%d)", count), bounds.Min.X+countInset, bounds.Min.Y)
		label.BoundsW = float64(bounds.Dx() - countInset - 4)
		label.BoundsH = float64(bounds.Dy())
		label.HAlign, label.VAlign = elements.AlignCenter, elements.AlignMiddle
		if active {
			strokeDeckRectangle(screen, bounds, 2, gold, scale)
			label.Color = gold
		}
		labelOpts := &ebiten.DrawImageOptions{}
		labelOpts.GeoM.Scale(scale, scale)
		label.Draw(screen, labelOpts, scale)
	}
}

func (s *EditDeckScreen) deckCardCounts() [domain.MaxPlayerDecks]int {
	var totals [domain.MaxPlayerDecks]int
	for _, item := range s.Player.CardCollection {
		for slot := range totals {
			totals[slot] += item.DeckCount(slot)
		}
	}
	return totals
}

func (s *EditDeckScreen) updateDeckTabs(W, H int) {
	keys := [domain.MaxPlayerDecks][2]ebiten.Key{
		{ebiten.Key1, ebiten.KeyNumpad1},
		{ebiten.Key2, ebiten.KeyNumpad2},
		{ebiten.Key3, ebiten.KeyNumpad3},
	}
	for slot, pair := range keys {
		if ui.Click(deckTabBounds(slot, W, H)) || inpututil.IsKeyJustPressed(pair[0]) || inpututil.IsKeyJustPressed(pair[1]) {
			s.SwitchDeck(slot)
			return
		}
	}
}

// SwitchDeck selects a deck and updates the editor.
func (s *EditDeckScreen) SwitchDeck(deckIndex int) {
	current := s.Player.ActiveDeck
	s.Player.SetActiveDeck(deckIndex)
	if current == s.Player.ActiveDeck {
		return
	}
	s.dragManager.Cancel()
	s.ignoreDrag = ui.Pressed()
	s.deckPage = 0
	s.updateState()
}
