package screens

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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
	enemy          *domain.Enemy
	enemyAnteCard  *ebiten.Image
	enemyVisage    *ebiten.Image
	enemyName      string
	lvl            *world.Level
	idx            int
	duelBtn        image.Rectangle
	bribeBtn       image.Rectangle
	visageBorder   []*ebiten.Image
	playerStatsUI  []*ebiten.Image
	player         *domain.Player
}

func NewDuelAnteScreen() *DuelAnteScreen {
	return &DuelAnteScreen{}
}

// horizontally center src on dest
func hCenter(dest, src *ebiten.Image) float64 {
	dw := dest.Bounds().Dx()
	sw := src.Bounds().Dx()
	return float64((dw / 2) - (sw / 2))
}

func NewDuelAnteScreenWithEnemy(l *world.Level, idx int) *DuelAnteScreen {
	enemy := l.GetEnemyAt(idx)
	s := &DuelAnteScreen{
		player:   l.Player,
		enemy:    enemy,
		lvl:      l,
		idx:      idx,
		duelBtn:  image.Rectangle{image.Point{400, 500}, image.Point{700, 540}},
		bribeBtn: image.Rectangle{image.Point{400, 550}, image.Point{700, 590}},
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

func (s *DuelAnteScreen) startDuel() (screenui.ScreenName, error) {
	s.lvl.RemoveEnemyAt(s.idx)
	return screenui.WorldScr, nil
}

func (s *DuelAnteScreen) bribe() (screenui.ScreenName, error) {
	s.lvl.RemoveEnemyAt(s.idx)
	s.player.Gold -= s.enemy.BribeAmount()
	return screenui.WorldScr, nil
}

func within(point image.Point, btn image.Rectangle) bool {
	click := image.Rectangle{point, point}
	return click.In(btn)
}

func (s *DuelAnteScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	mx, my := ebiten.CursorPosition()
	mousePoint := image.Point{mx, my}

	if ebiten.IsKeyPressed(ebiten.Key1) {
		return s.startDuel()
	}

	if ebiten.IsKeyPressed(ebiten.Key2) {
		return s.bribe()
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if within(mousePoint, s.duelBtn) {
			return s.startDuel()
		}
		if within(mousePoint, s.bribeBtn) {
			return s.bribe()
		}
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

	// Enemy Name
	nameImg := elements.ScaleImage(s.visageBorder[21], 2.0)
	nameTxt := elements.NewText(36, s.enemyName, 30, 20)
	nameTxt.Color = color.Black
	nameTxt.Draw(nameImg, &ebiten.DrawImageOptions{})
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(hCenter(screen, nameImg), 10)
	screen.DrawImage(nameImg, opts)

	// Draw enemy visage with border
	if len(s.visageBorder) > 0 && s.visageBorder[0] != nil {
		borderedVisageImg := ebiten.NewImageFromImage(s.visageBorder[20])
		opts := &ebiten.DrawImageOptions{}
		x := hCenter(borderedVisageImg, s.enemyVisage)
		opts.GeoM.Translate(x, 5)
		borderedVisageImg.DrawImage(s.enemyVisage, opts)
		borderedVisageImg = elements.ScaleImage(borderedVisageImg, 2.0)
		YPos := 80.0
		borderOpts := &ebiten.DrawImageOptions{}
		borderOpts.GeoM.Translate(hCenter(screen, borderedVisageImg), YPos)
		screen.DrawImage(borderedVisageImg, borderOpts)
	}

	// Main description text - centered, positioned better
	duelText := "Those who enter the stronghold of the Mighty Wizard\n will be met with the firmest resistance. You must..."
	textElement := elements.NewText(26, duelText, W/2-250, 450)
	textElement.Draw(screen, &ebiten.DrawImageOptions{})

	// Action buttons - centered, positioned better
	duelBtnText := elements.NewText(28, "1. Duel the Enemy", s.duelBtn.Min.X, s.duelBtn.Min.Y)
	duelBtnText.Draw(screen, &ebiten.DrawImageOptions{})

	bribeBtnText := elements.NewText(28, fmt.Sprintf("2. Bribe for %d gold", s.enemy.BribeAmount()), s.bribeBtn.Min.X, s.bribeBtn.Min.Y)
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
