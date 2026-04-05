package screens

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/benprew/s30/assets"
	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type DuelAnteScreen struct {
	background        *ebiten.Image
	playerAnteCardImg *ebiten.Image
	playerAnteCard    *domain.Card
	enemy             *domain.Enemy
	enemyAnteCard     *domain.Card
	enemyAnteCardImg  *ebiten.Image
	enemyVisage       *ebiten.Image
	enemyName         string
	lvl               *world.Level
	idx               int
	duelBtn           *elements.Button
	bribeBtn          *elements.Button
	visageBorder      []*ebiten.Image
	playerStatsUI     []*ebiten.Image
	player            *domain.Player
	wonCards          []*domain.Card
}

func (s *DuelAnteScreen) IsFramed() bool { return false }

func NewDuelAnteScreen() *DuelAnteScreen {
	return &DuelAnteScreen{}
}

func NewDuelAnteScreenWithEnemy(l *world.Level, idx int) *DuelAnteScreen {
	enemy := l.GetEnemyAt(idx)
	s := &DuelAnteScreen{
		player: l.Player,
		enemy:  enemy,
		lvl:    l,
		idx:    idx,
	}

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(fmt.Sprintf("Error loading button sprites: %v", err))
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}

	duelText := "1. Duel the Enemy"
	bribeText := fmt.Sprintf("2. Bribe for %d gold", enemy.BribeAmount())
	duelW, duelH := elements.TextButtonSize(duelText, fontFace)
	bribeW, _ := elements.TextButtonSize(bribeText, fontFace)

	btnY := 500
	s.duelBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    duelText,
		Font:    fontFace,
		ID:      "duel",
		X:       512 - duelW/2,
		Y:       btnY,
	})

	s.bribeBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    bribeText,
		Font:    fontFace,
		ID:      "bribe",
		X:       512 - bribeW/2,
		Y:       btnY + duelH + 10,
	})

	s.background = loadBackgroundForEnemy(enemy)

	s.playerAnteCard = selectPlayerAnteCard(l.Player.GetActiveDeck())
	card, err := s.playerAnteCard.CardImage(domain.CardViewFull)
	if err != nil || card == nil {
		panic(fmt.Sprintf("No card image for %s\n", s.playerAnteCard.Name()))
	}
	s.playerAnteCardImg = imageutil.ScaleImage(card, 0.75)

	s.enemyAnteCard = selectEnemyAnteCard(enemy.Character.GetActiveDeck())
	card, err = s.enemyAnteCard.CardImage(domain.CardViewFull)
	if err != nil {
		panic(fmt.Sprintf("No card image for %s\n", s.enemyAnteCard.Name()))
	}
	s.enemyAnteCardImg = imageutil.ScaleImage(card, 0.75)
	s.visageBorder = loadVisageBorder()
	s.playerStatsUI = loadPlayerStatsUI()

	s.enemyVisage = borderedVisage(enemy.Character.Visage, s.visageBorder[20])
	s.enemyName = enemy.Character.Name

	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.EnemySFXForName(enemy.Character.Name))
	}

	return s
}

func borderedVisage(visage, border *ebiten.Image) *ebiten.Image {
	borderedVisageImg := ebiten.NewImageFromImage(border)
	opts := &ebiten.DrawImageOptions{}
	x := hCenter(borderedVisageImg, visage)
	opts.GeoM.Translate(x, 5)
	borderedVisageImg.DrawImage(visage, opts)
	return imageutil.ScaleImage(borderedVisageImg, 1.5)
}

func (s *DuelAnteScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if ebiten.IsKeyPressed(ebiten.Key1) {
		return s.startDuel()
	}

	if ebiten.IsKeyPressed(ebiten.Key2) {
		return s.bribe()
	}

	opts := &ebiten.DrawImageOptions{}
	s.duelBtn.Update(opts, scale, W, H)
	s.bribeBtn.Update(opts, scale, W, H)

	if s.duelBtn.IsClicked() {
		return s.startDuel()
	}
	if s.bribeBtn.IsClicked() {
		return s.bribe()
	}

	return screenui.DuelAnteScr, nil, nil
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

	// Player ante card - left side
	if s.playerAnteCardImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(50, 50)
		screen.DrawImage(s.playerAnteCardImg, opts)
	}

	// Enemy ante card - right side
	if s.enemyAnteCardImg != nil {
		opts := &ebiten.DrawImageOptions{}
		cardBounds := s.enemyAnteCardImg.Bounds()
		xPos := W - cardBounds.Dx() - 50
		opts.GeoM.Translate(float64(xPos), 50)
		screen.DrawImage(s.enemyAnteCardImg, opts)
	}

	// Enemy Name
	nameImg := imageutil.ScaleImage(s.visageBorder[21], 1.5)
	nameTxt := elements.NewText(30, s.enemyName, 30, 15)
	nameTxt.Color = color.Black
	nameTxt.Draw(nameImg, &ebiten.DrawImageOptions{}, 1.0)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(hCenter(screen, nameImg), 10)
	screen.DrawImage(nameImg, opts)

	YPos := 80.0
	borderOpts := &ebiten.DrawImageOptions{}
	borderOpts.GeoM.Translate(hCenter(screen, s.enemyVisage), YPos)
	screen.DrawImage(s.enemyVisage, borderOpts)

	// Main description text - centered, positioned better
	duelText := "Those who enter the stronghold of the Mighty Wizard\n will be met with the firmest resistance. You must..."
	textElement := elements.NewText(24, duelText, W/2-250, 450)
	textElement.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	btnOpts := &ebiten.DrawImageOptions{}
	s.duelBtn.Draw(screen, btnOpts, scale)
	s.bribeBtn.Draw(screen, btnOpts, scale)

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
		cardsText := fmt.Sprintf("%d", s.lvl.Player.NumCards())

		// Position text within the scaled stats UI
		statsY := float64(H) - scaledStatsH
		elements.NewText(12, lifeText, 60, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		elements.NewText(12, goldText, 110, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		elements.NewText(12, foodText, 160, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		elements.NewText(12, cardsText, 210, int(statsY-5)).Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}
}

// horizontally center src on dest
func hCenter(dest, src *ebiten.Image) float64 {
	dw := dest.Bounds().Dx()
	sw := src.Bounds().Dx()
	return float64((dw / 2) - (sw / 2))
}

func (s *DuelAnteScreen) startDuel() (screenui.ScreenName, screenui.Screen, error) {
	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.SFXDice)
	}
	return screenui.DuelScr, NewDuelScreen(s.player, s.enemy, s.lvl, s.idx, s.playerAnteCard, s.enemyAnteCard), nil
}

func (s *DuelAnteScreen) bribe() (screenui.ScreenName, screenui.Screen, error) {
	s.lvl.RemoveEnemyAt(s.idx)
	s.player.Gold -= s.enemy.BribeAmount()
	return screenui.WorldScr, nil, nil
}

func loadBackgroundForEnemy(enemy *domain.Enemy) *ebiten.Image {
	var backgroundFile string
	backgroundFile = "art/screens/duel_ante/Prdwht.pic.png"

	switch enemy.Character.PrimaryColor {
	case "White":
		backgroundFile = "art/screens/duel_ante/Prdwht.pic.png"
	case "Blue":
		backgroundFile = "art/screens/duel_ante/Prdblu.pic.png"
	case "Black":
		backgroundFile = "art/screens/duel_ante/Prdblk.pic.png"
	case "Red":
		backgroundFile = "art/screens/duel_ante/Prdred.pic.png"
	case "Green":
		backgroundFile = "art/screens/duel_ante/Prdgrn.pic.png"
	}

	data, err := assets.DuelAnteFS.ReadFile(backgroundFile)
	if err != nil {
		fmt.Printf("Error loading background %s: %v\n", backgroundFile, err)
	}

	img, err := imageutil.LoadImage(data)
	if err != nil {
		fmt.Printf("Error decoding background %s: %v\n", backgroundFile, err)
	}
	return img
}

func selectPlayerAnteCard(deck domain.Deck) *domain.Card {
	validCards := deck.ValidAnteCards(domain.ExcludeBasicLand)

	if len(validCards) == 0 {
		panic("No valid ante cards!!")
	}

	return validCards[rand.Intn(len(validCards))]
}

func selectEnemyAnteCard(deck domain.Deck) *domain.Card {
	// 5% chance to allow VintageRestricted cards as ante
	if rand.Intn(100) >= 5 {
		validCards := deck.ValidAnteCards(domain.ExcludeVintageRestricted)
		if len(validCards) > 0 {
			return validCards[rand.Intn(len(validCards))]
		}
	}

	validCards := deck.ValidAnteCards()

	if len(validCards) == 0 {
		panic("No valid ante cards!!")
	}

	return validCards[rand.Intn(len(validCards))]
}

func loadVisageBorder() []*ebiten.Image {
	return loadButtonMap(assets.DuelAnteBorder_png, assets.DuelAnteBorderMap_json)
}

func loadPlayerStatsUI() []*ebiten.Image {
	return loadButtonMap(assets.DuelAnteStats_png, assets.DuelAnteStatsMap_json)
}

func (s *DuelAnteScreen) WonCards() []*domain.Card {
	return s.wonCards
}

func (s *DuelAnteScreen) LostCards() []*domain.Card {
	return []*domain.Card{s.playerAnteCard}
}
