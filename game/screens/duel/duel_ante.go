package duel

import (
	"fmt"
	"image"
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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)


type DuelAnteScreen struct {
	background     *ebiten.Image
	playerAnteCard *domain.Card
	enemy          *domain.Enemy
	enemyAnteCard  *domain.Card
	enemyVisage    *ebiten.Image
	enemyName      string
	lvl            *world.Level
	idx            int
	duelBtn        elements.Button
	bribeBtn       elements.Button
	visageBorder   []*ebiten.Image
	playerStatsUI  []*ebiten.Image
	player         *domain.Player
	wonCards       []*domain.Card
	nameImage      *ebiten.Image
	arenaRound     int
	arenaOutcome   func(bool) (screenui.ScreenName, screenui.Screen, error)
	arenaLeave     func() (screenui.ScreenName, screenui.Screen, error)
	arenaEdit      func(int, int) (screenui.ScreenName, screenui.Screen, error)
	editBtn        *elements.Button
}

func (s *DuelAnteScreen) IsFramed() bool { return false }

func (s *DuelAnteScreen) IsOverlay() bool { return false }

func NewDuelAnteScreen() *DuelAnteScreen {
	return &DuelAnteScreen{}
}

func NewDuelAnteScreenWithEnemy(l *world.Level, idx int) *DuelAnteScreen {
	enemy := l.GetEnemyAt(idx)
	return newDuelAnteScreen(l.Player, enemy, l, idx, 0)
}

// NewArenaDuelAnteScreen offers a duel, withdrawal, and editing without an ante.
func NewArenaDuelAnteScreen(player *domain.Player, enemy *domain.Enemy, round int,
	outcome func(bool) (screenui.ScreenName, screenui.Screen, error),
	leave func() (screenui.ScreenName, screenui.Screen, error),
	edit func(int, int) (screenui.ScreenName, screenui.Screen, error),
) *DuelAnteScreen {
	s := newDuelAnteScreen(player, enemy, nil, -1, round)
	s.arenaOutcome, s.arenaLeave, s.arenaEdit = outcome, leave, edit
	return s
}

func newDuelAnteScreen(player *domain.Player, enemy *domain.Enemy, level *world.Level, idx, arenaRound int) *DuelAnteScreen {
	s := &DuelAnteScreen{
		player:     player,
		enemy:      enemy,
		lvl:        level,
		idx:        idx,
		arenaRound: arenaRound,
	}

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(fmt.Sprintf("Error loading button sprites: %v", err))
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}

	duelText := "1. Duel the Enemy"
	bribeText := fmt.Sprintf("2. Bribe for %d gold", enemy.BribeAmount())
	if arenaRound > 0 {
		duelText = "1. Duel the Creature"
		bribeText = "2. Leave the Arena"
	}
	duelW, duelH := elements.TextButtonSize(duelText, fontFace)
	bribeW, _ := elements.TextButtonSize(bribeText, fontFace)

	btnY := 500
	buttonW := duelW
	if arenaRound > 0 {
		editText := "3. Edit Deck"
		editW, _ := elements.TextButtonSize(editText, fontFace)
		buttonW = max(duelW, bribeW, editW, 260)
	}

	btnX := 512 - buttonW/2
	s.duelBtn = *elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    duelText,
		Font:    fontFace,
		ID:      "duel",
		X:       btnX,
		Y:       btnY,
	})
	if arenaRound > 0 && s.duelBtn.Bounds.Dx() != buttonW {
		scaleX := float64(buttonW) / float64(s.duelBtn.Bounds.Dx())
		s.duelBtn.Normal = imageutil.ScaleImageInd(s.duelBtn.Normal, scaleX, 1.0)
		s.duelBtn.Hover = imageutil.ScaleImageInd(s.duelBtn.Hover, scaleX, 1.0)
		s.duelBtn.Pressed = imageutil.ScaleImageInd(s.duelBtn.Pressed, scaleX, 1.0)
		s.duelBtn.Bounds = image.Rect(btnX, btnY, btnX+buttonW, btnY+duelH)
	}

	if arenaRound > 0 || canBribe(s) {
		bribeX := 512 - bribeW/2
		if arenaRound > 0 {
			bribeX = btnX
		}
		s.bribeBtn = *elements.NewButtonFromConfig(elements.ButtonConfig{
			Normal:  btnSprites[0][0],
			Hover:   btnSprites[0][1],
			Pressed: btnSprites[0][2],
			Text:    bribeText,
			Font:    fontFace,
			ID:      "bribe",
			X:       bribeX,
			Y:       btnY + duelH + 10,
		})
		if arenaRound > 0 && s.bribeBtn.Bounds.Dx() != buttonW {
			scaleX := float64(buttonW) / float64(s.bribeBtn.Bounds.Dx())
			s.bribeBtn.Normal = imageutil.ScaleImageInd(s.bribeBtn.Normal, scaleX, 1.0)
			s.bribeBtn.Hover = imageutil.ScaleImageInd(s.bribeBtn.Hover, scaleX, 1.0)
			s.bribeBtn.Pressed = imageutil.ScaleImageInd(s.bribeBtn.Pressed, scaleX, 1.0)
			s.bribeBtn.Bounds = image.Rect(btnX, btnY+duelH+10, btnX+buttonW, btnY+2*duelH+10)
		}
	}
	if arenaRound > 0 {
		editText := "3. Edit Deck"
		s.editBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
			Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
			Text: editText, Font: fontFace, ID: "edit",
			X: btnX, Y: btnY + 2*(duelH+10),
		})
		if s.editBtn.Bounds.Dx() != buttonW {
			scaleX := float64(buttonW) / float64(s.editBtn.Bounds.Dx())
			s.editBtn.Normal = imageutil.ScaleImageInd(s.editBtn.Normal, scaleX, 1.0)
			s.editBtn.Hover = imageutil.ScaleImageInd(s.editBtn.Hover, scaleX, 1.0)
			s.editBtn.Pressed = imageutil.ScaleImageInd(s.editBtn.Pressed, scaleX, 1.0)
			s.editBtn.Bounds = image.Rect(btnX, btnY+2*(duelH+10), btnX+buttonW, btnY+3*duelH+20)
		}
	}


	s.background = loadBackgroundForEnemy(enemy)

	if arenaRound == 0 {
		s.playerAnteCard = selectPlayerAnteCard(player.GetActiveDeck())
		card, imageErr := s.playerAnteCard.CardImage(domain.CardViewFullMini)
		if imageErr != nil || card == nil {
			panic(fmt.Sprintf("No card image for %s\n", s.playerAnteCard.Name()))
		}
		s.enemyAnteCard = selectEnemyAnteCard(enemy.Character.GetActiveDeck())
	}

	s.visageBorder = loadVisageBorder()
	s.playerStatsUI = loadPlayerStatsUI()

	s.enemyVisage = borderedVisage(enemy.Character.Visage, s.visageBorder[20])
	s.enemyName = enemy.Character.Name
	s.nameImage = imageutil.ScaleImage(s.visageBorder[21], 1.5)
	nameTxt := elements.NewText(30, s.enemyName, 30, 15)
	nameTxt.Color = color.Black
	nameTxt.Draw(s.nameImage, &ebiten.DrawImageOptions{}, 1)

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
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		return s.startDuel()
	}

	if s.arenaRound > 0 {
		if inpututil.IsKeyJustPressed(ebiten.Key2) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			return s.arenaLeave()
		}
		if inpututil.IsKeyJustPressed(ebiten.Key3) {
			return s.arenaEdit(W, H)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) && canBribe(s) {
		return s.bribe()
	}

	opts := &ebiten.DrawImageOptions{}
	s.duelBtn.Update(opts, scale, W, H)
	s.bribeBtn.Update(opts, scale, W, H)
	if s.editBtn != nil {
		s.editBtn.Update(opts, scale, W, H)
		if s.editBtn.IsClicked() {
			return s.arenaEdit(W, H)
		}
	}

	if s.duelBtn.IsClicked() {
		return s.startDuel()
	}
	if s.bribeBtn.IsClicked() {
		if s.arenaRound > 0 {
			return s.arenaLeave()
		}
		return s.bribe()
	}

	return screenui.DuelAnteScr, nil, nil
}

func (s *DuelAnteScreen) cardImages() (player, enemy *ebiten.Image) {
	if s.playerAnteCard != nil {
		player, _ = s.playerAnteCard.CardImage(domain.CardViewFullMini)
	}
	if s.enemyAnteCard != nil {
		enemy, _ = s.enemyAnteCard.CardImage(domain.CardViewFullMini)
	}
	return player, enemy
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

	playerImg, enemyImg := s.cardImages()

	// Player ante card - left side
	if playerImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(50, 50)
		screen.DrawImage(playerImg, opts)
	}

	// Enemy ante card - right side
	if enemyImg != nil {
		opts := &ebiten.DrawImageOptions{}
		cardBounds := enemyImg.Bounds()
		xPos := W - cardBounds.Dx() - 50
		opts.GeoM.Translate(float64(xPos), 50)
		screen.DrawImage(enemyImg, opts)
	}

	// Enemy Name
	nameImg := s.nameImage
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(hCenter(screen, nameImg), 10)
	screen.DrawImage(nameImg, opts)

	YPos := 80.0
	borderOpts := &ebiten.DrawImageOptions{}
	borderOpts.GeoM.Translate(hCenter(screen, s.enemyVisage), YPos)
	screen.DrawImage(s.enemyVisage, borderOpts)

	if s.arenaRound > 0 {
		panelW, panelH := 580, 115
		panelX := (W - panelW) / 2
		panelY := 365
		vector.FillRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), color.RGBA{14, 10, 22, 225}, false)
		vector.StrokeRect(screen, float32(panelX), float32(panelY), float32(panelW), float32(panelH), 1, color.RGBA{135, 115, 80, 200}, false)

		roundText := fmt.Sprintf("Arena Round %d of %d — Win to advance!", s.arenaRound, domain.ArenaRounds)
		rt := elements.NewText(22, roundText, 0, panelY+12)
		rt.Color = color.RGBA{R: 255, G: 230, B: 150, A: 255}
		rt.HAlign, rt.BoundsW = elements.AlignCenter, float64(W)
		rt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

		life := fmt.Sprintf("Your Life: %d          Opponent Life: %d", s.player.Life, s.enemy.Character.Life)
		lt := elements.NewText(20, life, 0, panelY+44)
		lt.Color = color.RGBA{R: 220, G: 240, B: 220, A: 255}
		lt.HAlign, lt.BoundsW = elements.AlignCenter, float64(W)
		lt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)


		deckCount := 0
		for _, count := range s.player.GetActiveDeck() {
			deckCount += count
		}

		var dt *elements.Text
		if deckCount >= s.player.MinDeckSize {
			deckMsg := fmt.Sprintf("Deck: %d / %d cards (Ready)  •  No ante", deckCount, s.player.MinDeckSize)
			dt = elements.NewText(18, deckMsg, 0, panelY+76)
			dt.Color = color.RGBA{R: 200, G: 220, B: 255, A: 255}
		} else {
			deckMsg := fmt.Sprintf("Deck: %d / %d cards (Shortfall filled with lands)  •  No ante", deckCount, s.player.MinDeckSize)
			dt = elements.NewText(18, deckMsg, 0, panelY+76)
			dt.Color = color.RGBA{R: 255, G: 200, B: 120, A: 255}
		}
		dt.HAlign, dt.BoundsW = elements.AlignCenter, float64(W)
		dt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	} else {
		// Main description text - centered, positioned better
		duelText := "Those who enter the stronghold of the Mighty Wizard\n will be met with the firmest resistance. You must..."
		textElement := elements.NewText(24, duelText, W/2-250, 450)
		textElement.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}

	btnOpts := &ebiten.DrawImageOptions{}
	s.duelBtn.Draw(screen, btnOpts, scale)
	s.bribeBtn.Draw(screen, btnOpts, scale)
	if s.editBtn != nil {
		s.editBtn.Draw(screen, btnOpts, scale)
	}


	// Player stats UI background in lower-left
	if s.arenaRound == 0 && len(s.playerStatsUI) > 0 && s.playerStatsUI[0] != nil {
		statsOpts := &ebiten.DrawImageOptions{}
		statsUIBounds := s.playerStatsUI[0].Bounds()
		statsScale := 0.4 // Scale down the stats UI
		statsOpts.GeoM.Scale(statsScale, statsScale)
		scaledStatsH := float64(statsUIBounds.Dy()) * statsScale
		statsOpts.GeoM.Translate(20, float64(H)-scaledStatsH-20)
		screen.DrawImage(s.playerStatsUI[0], statsOpts)

		// Player stats text overlay - positioned within the stats UI
		lifeText := fmt.Sprintf("%d", s.player.Life)
		goldText := fmt.Sprintf("%d", s.player.Gold)
		foodText := fmt.Sprintf("%d", s.player.Food)
		cardsText := fmt.Sprintf("%d", s.player.NumCards())

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
	duel := NewDuelScreen(s.player, s.enemy, s.lvl, s.idx, s.playerAnteCard, s.enemyAnteCard)
	duel.arenaOutcome = s.arenaOutcome
	return screenui.DuelScr, duel, nil
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
	return imageutil.LoadButtonMap(assets.DuelAnteBorder_png, assets.DuelAnteBorderMap_json)
}

func loadPlayerStatsUI() []*ebiten.Image {
	return imageutil.LoadButtonMap(assets.DuelAnteStats_png, assets.DuelAnteStatsMap_json)
}

func (s *DuelAnteScreen) WonCards() []*domain.Card {
	return s.wonCards
}

func (s *DuelAnteScreen) LostCards() []*domain.Card {
	return []*domain.Card{s.playerAnteCard}
}

func canBribe(s *DuelAnteScreen) bool {
	if s.arenaRound > 0 {
		return false
	}
	bribeAmount := s.enemy.BribeAmount()
	return bribeAmount != 0 && s.player.Gold >= bribeAmount // 0 means cannot be bribed
}
