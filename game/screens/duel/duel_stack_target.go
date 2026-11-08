package duel

import (
	"fmt"
	"image"
	"image/color"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type stackTargetCard struct {
	Item interactive.StackItemState
	ID   uuid.UUID
	Rect image.Rectangle
}

func (s *DuelScreen) isChoosingStackTarget() bool {
	_, ok := s.targetingAction.TargetType.(*mage.SpellOnStackTarget)
	return s.targetingCardID != uuid.Nil() && ok
}

func (s *DuelScreen) stackTargetLayout(W, H int) ([]stackTargetCard, int) {
	if !s.isChoosingStackTarget() || s.lastMsg == nil || s.lastMsg.State == nil {
		return nil, 0
	}
	var cards []stackTargetCard
	for i := len(s.lastMsg.State.StackItems) - 1; i >= 0; i-- {
		item := s.lastMsg.State.StackItems[i]
		id, err := uuid.Parse(item.ID)
		if err != nil || item.IsAbility {
			continue
		}
		if _, ok := s.targetingActions[id]; ok {
			cards = append(cards, stackTargetCard{Item: item, ID: id})
		}
	}

	const gap = 16
	const cardW = 200
	const cardH = domain.CardArtHeight + 56
	cols := max((W-gap)/(cardW+gap), 1)
	rows := max((H-140)/(cardH+gap), 1)
	perPage := cols * rows
	pages := max((len(cards)+perPage-1)/perPage, 1)
	page := min(s.stackTargetPage, pages-1)
	start := page * perPage
	cards = cards[start:min(start+perPage, len(cards))]
	startX := (W - cols*cardW - (cols-1)*gap) / 2
	for i := range cards {
		x := startX + (i%cols)*(cardW+gap)
		y := 76 + (i/cols)*(cardH+gap)
		cards[i].Rect = image.Rect(x, y, x+cardW, y+cardH)
	}
	return cards, pages
}

func stackTargetPageBounds(W, H int) (image.Rectangle, image.Rectangle) {
	return image.Rect(W/2-200, H-52, W/2-90, H-16), image.Rect(W/2+90, H-52, W/2+200, H-16)
}

func (s *DuelScreen) updateStackTargetMouse(mx, my int, clicked bool, W, H int) {
	if !clicked {
		return
	}
	cards, pages := s.stackTargetLayout(W, H)
	point := image.Pt(mx, my)
	previous, next := stackTargetPageBounds(W, H)
	switch {
	case point.In(previous) && s.stackTargetPage > 0:
		s.stackTargetPage--
	case point.In(next) && s.stackTargetPage+1 < pages:
		s.stackTargetPage++
	default:
		for _, card := range cards {
			if point.In(card.Rect) {
				s.selectTarget(card.ID)
				if s.submitTargetingAction() {
					s.exitTargetingMode()
				}
				return
			}
		}
	}
}

func (s *DuelScreen) drawStackTargetView(screen *ebiten.Image, W, H int) {
	if !s.isChoosingStackTarget() {
		return
	}
	cards, pages := s.stackTargetLayout(W, H)
	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 225}, false)
	title := elements.NewText(22, "Choose a spell on the stack", 0, 16)
	title.HAlign, title.BoundsW = elements.AlignCenter, float64(W)
	title.Draw(screen, nil, 1)
	help := elements.NewText(16, "Click a spell to select it. Cancel or Esc to return.", 0, 46)
	help.HAlign, help.BoundsW = elements.AlignCenter, float64(W)
	help.Draw(screen, nil, 1)
	for _, card := range cards {
		r := card.Rect
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.RGBA{40, 40, 40, 255}, false)
		if img := s.getCardArtImg(card.Item.Name, domain.CardViewArtOnly); img != nil {
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(r.Min.X+(r.Dx()-domain.CardArtWidth)/2), float64(r.Min.Y))
			screen.DrawImage(img, opts)
		}
		drawStackTargetLabel(screen, card.Item.Name, image.Rect(r.Min.X+4, r.Min.Y+domain.CardArtHeight+6, r.Max.X-4, r.Max.Y-26))
		drawStackTargetLabel(screen, card.Item.Controller, image.Rect(r.Min.X+4, r.Max.Y-24, r.Max.X-4, r.Max.Y-2))
		vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 2, color.RGBA{255, 255, 0, 255}, false)
	}
	if pages > 1 {
		previous, next := stackTargetPageBounds(W, H)
		drawStackTargetLabel(screen, "Previous", previous)
		drawStackTargetLabel(screen, "Next", next)
		drawStackTargetLabel(screen, fmt.Sprintf("%d / %d", min(s.stackTargetPage+1, pages), pages), image.Rect(W/2-85, H-52, W/2+85, H-16))
	}
}

func drawStackTargetLabel(screen *ebiten.Image, label string, bounds image.Rectangle) {
	txt := elements.NewText(16, label, bounds.Min.X, bounds.Min.Y)
	if width, _ := txt.Measure(); width > float64(bounds.Dx()) {
		txt = elements.NewText(16*float64(bounds.Dx())/width, label, bounds.Min.X, bounds.Min.Y)
	}
	txt.HAlign, txt.VAlign = elements.AlignCenter, elements.AlignMiddle
	txt.BoundsW, txt.BoundsH = float64(bounds.Dx()), float64(bounds.Dy())
	txt.Draw(screen, nil, 1)
}
