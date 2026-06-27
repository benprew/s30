package screens

import (
	"fmt"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// QuestRewardScreen is an overlay shown when the player walks into a town with
// one or more completed deck-changing quests. It displays the gold and cards
// won over the Winbk_Questn quest-rewards background, then enters the town.

const (
	qrLogicalW   = 1024
	qrLogicalH   = 768
	qrPanelW     = 760
	qrCardScale  = 0.30
	qrCardGap    = 12
	qrLineHeight = 30
)

type qrReward struct {
	title    string
	gold     int
	cardImgs []*ebiten.Image
}

type QuestRewardScreen struct {
	rewards []qrReward
	bg      *ebiten.Image
	dim     *ebiten.Image
	panelX  int
	panelY  int
	panelH  int

	city   *domain.City
	player *domain.Player
	level  *world.Level
}

func (s *QuestRewardScreen) IsFramed() bool { return false }

func NewQuestRewardScreen(rewards []domain.DeckQuestReward, city *domain.City, player *domain.Player, level *world.Level) *QuestRewardScreen {
	bg := loadQuestRewardBg()
	panelW, panelH := qrPanelW, qrPanelW/2
	if bg != nil {
		b := bg.Bounds()
		scale := float64(qrPanelW) / float64(b.Dx())
		bg = imageutil.ScaleImage(bg, scale)
		panelW = bg.Bounds().Dx()
		panelH = bg.Bounds().Dy()
	}

	dim := ebiten.NewImage(qrLogicalW, qrLogicalH)
	dim.Fill(color.RGBA{0, 0, 0, 160})

	qr := make([]qrReward, 0, len(rewards))
	for _, r := range rewards {
		imgs := make([]*ebiten.Image, 0, len(r.Cards))
		for _, c := range r.Cards {
			img, err := c.CardImage(domain.CardViewFull)
			if err != nil {
				continue
			}
			imgs = append(imgs, imageutil.ScaleImage(img, qrCardScale))
		}
		qr = append(qr, qrReward{title: r.Quest.Title, gold: r.Reward.Gold, cardImgs: imgs})
	}

	return &QuestRewardScreen{
		rewards: qr,
		bg:      bg,
		dim:     dim,
		panelX:  (qrLogicalW - panelW) / 2,
		panelY:  (qrLogicalH - panelH) / 2,
		panelH:  panelH,
		city:    city,
		player:  player,
		level:   level,
	}
}

func loadQuestRewardBg() *ebiten.Image {
	data, err := assets.DuelFS.ReadFile("art/screens/duel/Winbk_Questn.pic.png")
	if err != nil {
		return nil
	}
	img, err := imageutil.LoadImage(data)
	if err != nil {
		return nil
	}
	return img
}

func (s *QuestRewardScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	dimOpts := &ebiten.DrawImageOptions{}
	dimOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(s.dim, dimOpts)

	if s.bg != nil {
		bgOpts := &ebiten.DrawImageOptions{}
		bgOpts.GeoM.Scale(scale, scale)
		bgOpts.GeoM.Translate(float64(s.panelX)*scale, float64(s.panelY)*scale)
		screen.DrawImage(s.bg, bgOpts)
	}

	title := elements.NewText(36, "Quest Complete!", s.panelX, s.panelY+24)
	title.Color = color.RGBA{255, 230, 150, 255}
	title.HAlign = elements.AlignCenter
	title.BoundsW = qrPanelW
	title.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	y := s.panelY + 90
	for _, r := range s.rewards {
		header := r.title
		if r.gold > 0 {
			header = fmt.Sprintf("%s   +%d gold", r.title, r.gold)
		}
		line := elements.NewText(24, header, s.panelX+60, y)
		line.Color = color.White
		line.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		y += qrLineHeight

		if len(r.cardImgs) > 0 {
			cardX := s.panelX + 60
			for _, img := range r.cardImgs {
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Scale(scale, scale)
				opts.GeoM.Translate(float64(cardX)*scale, float64(y)*scale)
				screen.DrawImage(img, opts)
				cardX += img.Bounds().Dx() + qrCardGap
			}
			y += r.cardImgs[0].Bounds().Dy() + 12
		}
		y += 8
	}

	prompt := elements.NewText(20, "Click to continue", s.panelX, s.panelY+s.panelH-40)
	prompt.Color = color.RGBA{220, 220, 220, 255}
	prompt.HAlign = elements.AlignCenter
	prompt.BoundsW = qrPanelW
	prompt.Draw(screen, &ebiten.DrawImageOptions{}, scale)
}

func (s *QuestRewardScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return screenui.CityScr, NewCityScreen(s.city, s.player, s.level), nil
	}
	return screenui.QuestRewardScr, nil, nil
}
