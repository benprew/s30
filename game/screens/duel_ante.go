package screens

import (
	"bytes"
	"fmt"
	"image"
	"math/rand"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
)

type DuelAnteScreen struct {
	background     *ebiten.Image
	playerAnteCard *ebiten.Image
	enemyAnteCard  *ebiten.Image
	enemyVisage    *ebiten.Image
	enemyName      string
	lvl            *world.Level
	idx            int
	duelBtnX       int
	duelBtnY       int
	duelBtnW       int
	duelBtnH       int
	bribeBtnX      int
	bribeBtnY      int
	bribeBtnW      int
	bribeBtnH      int
	clicked        bool
	visageBorder   []*ebiten.Image
	playerStatsUI  []*ebiten.Image
}

func NewDuelAnteScreen() *DuelAnteScreen {
	return &DuelAnteScreen{}
}

func NewDuelAnteScreenWithEnemy(l *world.Level, idx int, enemy domain.Enemy) *DuelAnteScreen {
	s := &DuelAnteScreen{
		lvl:       l,
		idx:       idx,
		duelBtnX:  400,
		duelBtnY:  500,
		duelBtnW:  300,
		duelBtnH:  40,
		bribeBtnX: 400,
		bribeBtnY: 550,
		bribeBtnW: 300,
		bribeBtnH: 40,
	}

	s.background = loadRandomBackground()
	s.playerAnteCard = selectAnteCard(l.Player.Character.Deck, true)
	s.enemyAnteCard = selectAnteCard(enemy.Character.Deck, false)
	s.enemyVisage = enemy.Character.Visage
	s.enemyName = enemy.Character.Name
	s.visageBorder = loadVisageBorder()
	s.playerStatsUI = loadPlayerStatsUI()

	return s
}

func (s *DuelAnteScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	mx, my := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !s.clicked {
			if mx >= s.duelBtnX && mx <= s.duelBtnX+s.duelBtnW &&
				my >= s.duelBtnY && my <= s.duelBtnY+s.duelBtnH {
				s.clicked = true
				if s.lvl != nil {
					s.lvl.RemoveEnemyAt(s.idx)
				}
				return screenui.WorldScr, nil
			}
			if mx >= s.bribeBtnX && mx <= s.bribeBtnX+s.bribeBtnW &&
				my >= s.bribeBtnY && my <= s.bribeBtnY+s.bribeBtnH {
				s.clicked = true
				if s.lvl != nil {
					s.lvl.RemoveEnemyAt(s.idx)
				}
				return screenui.WorldScr, nil
			}
		}
	} else {
		s.clicked = false
	}

	return screenui.DuelAnteScr, nil
}

func (s *DuelAnteScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	// Scale background to fill screen (1024x768)
	if s.background != nil {
		opts := &ebiten.DrawImageOptions{}
		bgBounds := s.background.Bounds()
		scaleX := float64(W) / float64(bgBounds.Dx())
		scaleY := float64(H) / float64(bgBounds.Dy())
		opts.GeoM.Scale(scaleX, scaleY)
		screen.DrawImage(s.background, opts)
	}

	// Player ante card - left side, scaled down
	if s.playerAnteCard != nil {
		opts := &ebiten.DrawImageOptions{}
		cardScale := 0.35 // Scale cards down to 35%
		opts.GeoM.Scale(cardScale, cardScale)
		opts.GeoM.Translate(50, 50)
		screen.DrawImage(s.playerAnteCard, opts)
	}

	// Enemy ante card - right side, scaled down
	if s.enemyAnteCard != nil {
		opts := &ebiten.DrawImageOptions{}
		cardScale := 0.35 // Scale cards down to 35%
		opts.GeoM.Scale(cardScale, cardScale)
		cardBounds := s.enemyAnteCard.Bounds()
		scaledWidth := float64(cardBounds.Dx()) * cardScale
		xPos := float64(W) - scaledWidth - 50
		opts.GeoM.Translate(xPos, 50)
		screen.DrawImage(s.enemyAnteCard, opts)
	}

	// Draw border frame first (behind visage)
	if len(s.visageBorder) > 0 && s.visageBorder[0] != nil {
		borderOpts := &ebiten.DrawImageOptions{}
		borderBounds := s.visageBorder[0].Bounds()
		borderScale := 0.8 // Scale border appropriately
		borderOpts.GeoM.Scale(borderScale, borderScale)
		scaledBorderW := float64(borderBounds.Dx()) * borderScale
		borderXPos := (float64(W) - scaledBorderW) / 2
		borderYPos := 80.0
		borderOpts.GeoM.Translate(borderXPos, borderYPos)
		screen.DrawImage(s.visageBorder[0], borderOpts)
	}

	// Enemy visage - center, scaled smaller
	if s.enemyVisage != nil {
		opts := &ebiten.DrawImageOptions{}
		visageBounds := s.enemyVisage.Bounds()
		visageScale := 2.0 // Reduced scale for visage
		opts.GeoM.Scale(visageScale, visageScale)
		// Center the scaled visage
		scaledW := float64(visageBounds.Dx()) * visageScale
		xPos := (float64(W) - scaledW) / 2
		yPos := 140.0 // Position visage within the border frame
		opts.GeoM.Translate(xPos, yPos)
		screen.DrawImage(s.enemyVisage, opts)
	}

	// Main description text - centered, positioned better
	duelText := "Those who enter the stronghold of the Mighty Wizard will be met with the firmest resistance. You must..."
	textElement := elements.NewText(16, duelText, W/2-250, 450)
	textElement.Draw(screen, &ebiten.DrawImageOptions{})

	// Action buttons - centered, positioned better
	duelBtnText := elements.NewText(18, "1. Duel the Enemy", s.duelBtnX, s.duelBtnY)
	duelBtnText.Draw(screen, &ebiten.DrawImageOptions{})

	bribeBtnText := elements.NewText(18, "2. Bribe the Enemy", s.bribeBtnX, s.bribeBtnY)
	bribeBtnText.Draw(screen, &ebiten.DrawImageOptions{})

	// Player stats UI background in lower-left
	if len(s.playerStatsUI) > 0 && s.playerStatsUI[0] != nil {
		statsOpts := &ebiten.DrawImageOptions{}
		statsUIBounds := s.playerStatsUI[0].Bounds()
		statsScale := 0.4 // Scale down the stats UI
		statsOpts.GeoM.Scale(statsScale, statsScale)
		scaledStatsH := float64(statsUIBounds.Dy()) * statsScale
		statsOpts.GeoM.Translate(20, float64(H)-scaledStatsH-20)
		screen.DrawImage(s.playerStatsUI[0], statsOpts)

		// Player stats text overlay - positioned within the stats UI
		lifeText := fmt.Sprintf("%d", s.lvl.Player.Life)
		goldText := fmt.Sprintf("%d", s.lvl.Player.Gold)
		foodText := fmt.Sprintf("%d", s.lvl.Player.Food)
		cardsText := fmt.Sprintf("%d", len(s.lvl.Player.Character.Deck))

		// Position text within the scaled stats UI
		statsY := float64(H) - scaledStatsH
		elements.NewText(12, lifeText, 60, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{})
		elements.NewText(12, goldText, 110, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{})
		elements.NewText(12, foodText, 160, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{})
		elements.NewText(12, cardsText, 210, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{})
	}
}

func (s *DuelAnteScreen) IsFramed() bool { return false }

func loadRandomBackground() *ebiten.Image {
	backgrounds := []string{
		"art/sprites/duel_ante/Prdblk.pic.png",
		"art/sprites/duel_ante/Prdblu.pic.png",
		"art/sprites/duel_ante/Prdgrn.pic.png",
		"art/sprites/duel_ante/Prdrd.pic.png",
		"art/sprites/duel_ante/Prdred.pic.png",
		"art/sprites/duel_ante/Prdwht.pic.png",
		"art/sprites/duel_ante/Prdwt.pic.png",
	}

	chosen := backgrounds[rand.Intn(len(backgrounds))]
	data, err := assets.DuelAnteFS.ReadFile(chosen)
	if err != nil {
		fmt.Printf("Error loading background %s: %v\n", chosen, err)
		return nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Printf("Error decoding background %s: %v\n", chosen, err)
		return nil
	}

	return ebiten.NewImageFromImage(img)
}

func selectAnteCard(deck []domain.DeckEntry, excludeBasicLand bool) *ebiten.Image {
	var validCards []domain.DeckEntry

	for _, entry := range deck {
		card := domain.FindCardByName(entry.Name)
		if card == nil {
			continue
		}

		if excludeBasicLand && card.CardType == domain.CardTypeLand {
			continue
		}

		validCards = append(validCards, entry)
	}

	if len(validCards) == 0 {
		return nil
	}

	chosen := validCards[rand.Intn(len(validCards))]
	card := domain.FindCardByName(chosen.Name)
	if card == nil {
		return nil
	}

	filename := card.Filename()
	data, err := readFromEmbeddedZip(assets.CardImages_zip, "carddata/"+filename)
	if err != nil {
		fmt.Printf("Error loading card %s: %v\n", filename, err)
		return nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Printf("Error decoding card %s: %v\n", filename, err)
		return nil
	}

	return ebiten.NewImageFromImage(img)
}

func loadVisageBorder() []*ebiten.Image {
	return loadButtonMap(assets.DuelAnteBorder_png, assets.DuelAnteBorderMap_json)
}


func loadPlayerStatsUI() []*ebiten.Image {
	return loadButtonMap(assets.DuelAnteStats_png, assets.DuelAnteStatsMap_json)
}

