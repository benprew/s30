package screens

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/mtg/core"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var phaseNames = []core.Phase{
	core.PhaseUntap,
	core.PhaseUpkeep,
	core.PhaseDraw,
	core.PhaseMain1,
	core.PhaseCombat,
	core.PhaseMain2,
	core.PhaseEnd,
}

type DuelScreen struct {
	player *domain.Player
	enemy  *domain.Enemy
	lvl    *world.Level
	idx    int

	gameState    *core.GameState
	corePlayer   *core.Player
	coreOpponent *core.Player
	gameDone     chan struct{}

	phaseDefaultBg  *ebiten.Image
	phaseActiveImgs []*ebiten.Image
	playerBoardBg *ebiten.Image
	opponentBg    *ebiten.Image
	manaPoolBg    *ebiten.Image
	handBg        *ebiten.Image
	graveyardImg  *ebiten.Image
	bigCardBg     *ebiten.Image
	spellChainBg  *ebiten.Image
	messageBg     *ebiten.Image

	manaSymbols []*ebiten.Image // 5 mana symbols (B, U, R, G, W)
	doneBtn     [3]*ebiten.Image

	selectedCardIdx int
	cardPreviewImg  *ebiten.Image

	anteCard *domain.Card
}

func (s *DuelScreen) IsFramed() bool { return false }

func NewDuelScreen(player *domain.Player, enemy *domain.Enemy, lvl *world.Level, idx int, anteCard *domain.Card) *DuelScreen {
	s := &DuelScreen{
		player:          player,
		enemy:           enemy,
		lvl:             lvl,
		idx:             idx,
		selectedCardIdx: -1,
		anteCard:        anteCard,
	}

	s.initGameState()
	s.loadImages()
	return s
}

func (s *DuelScreen) initGameState() {
	entityID := core.EntityID(1)

	s.corePlayer = &core.Player{
		ID:          1,
		LifeTotal:   20,
		ManaPool:    core.ManaPool{},
		Hand:        []*core.Card{},
		Library:     []*core.Card{},
		Battlefield: []*core.Card{},
		Graveyard:   []*core.Card{},
		Exile:       []*core.Card{},
		Turn:        &core.Turn{Phase: core.PhaseMain1},
		InputChan:   make(chan core.PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
		IsAI:        false,
	}

	playerDeck := s.player.GetActiveDeck()
	for card, count := range playerDeck {
		for range count {
			coreCard := core.NewCardFromDomain(card, entityID, s.corePlayer)
			s.corePlayer.Library = append(s.corePlayer.Library, coreCard)
			entityID++
		}
	}
	rand.Shuffle(len(s.corePlayer.Library), func(i, j int) {
		s.corePlayer.Library[i], s.corePlayer.Library[j] = s.corePlayer.Library[j], s.corePlayer.Library[i]
	})

	s.coreOpponent = &core.Player{
		ID:          2,
		LifeTotal:   s.enemy.Character.CalculateLifeFromLevel(),
		ManaPool:    core.ManaPool{},
		Hand:        []*core.Card{},
		Library:     []*core.Card{},
		Battlefield: []*core.Card{},
		Graveyard:   []*core.Card{},
		Exile:       []*core.Card{},
		Turn:        &core.Turn{},
		InputChan:   make(chan core.PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
		IsAI:        true,
	}

	enemyDeck := s.enemy.Character.GetActiveDeck()
	for card, count := range enemyDeck {
		for range count {
			coreCard := core.NewCardFromDomain(card, entityID, s.coreOpponent)
			s.coreOpponent.Library = append(s.coreOpponent.Library, coreCard)
			entityID++
		}
	}
	rand.Shuffle(len(s.coreOpponent.Library), func(i, j int) {
		s.coreOpponent.Library[i], s.coreOpponent.Library[j] = s.coreOpponent.Library[j], s.coreOpponent.Library[i]
	})

	s.gameState = core.NewGame([]*core.Player{s.corePlayer, s.coreOpponent})
	s.gameState.StartGame()

	s.gameDone = make(chan struct{})
	go s.runOpponentAI()
	go s.runGameLoop()
}

func (s *DuelScreen) loadImages() {
	playerColor := colorNameForDeck(s.player.PrimaryColor)
	enemyColor := colorNameForDeck(s.enemy.Character.PrimaryColor)

	phaseImg := loadDuelImage("Winbk_Phase.pic.png")
	if phaseImg != nil {
		s.phaseDefaultBg = phaseImg.SubImage(image.Rect(0, 0, 41, 760)).(*ebiten.Image)
		s.phaseActiveImgs = make([]*ebiten.Image, 18)
		for r := range 18 {
			s.phaseActiveImgs[r] = phaseImg.SubImage(image.Rect(41, r*42, 82, (r+1)*42)).(*ebiten.Image)
		}
	}
	s.playerBoardBg = loadDuelImage(fmt.Sprintf("Terr_%smana.pic.png", playerColor))
	s.opponentBg = loadDuelImage(fmt.Sprintf("Terr_%smana.pic.png", enemyColor))
	s.manaPoolBg = loadDuelImage("Winbk_Manapool.pic.png")
	s.handBg = loadDuelImage(fmt.Sprintf("Hand_%s.pic.png", playerColor))
	s.graveyardImg = loadDuelImage(fmt.Sprintf("Grave_%s.pic.png", playerColor))
	s.bigCardBg = loadDuelImage("Winbk_Bigcard.pic.png")
	s.spellChainBg = loadDuelImage("Winbk_Spellchain.pic.png")
	s.messageBg = loadDuelImage("Winbk_Telluser.pic.png")

	statbutt, err := imageutil.LoadSpriteSheet(16, 1, assets.Statbutt_png)
	if err != nil {
		fmt.Printf("Error loading Statbutt sprite sheet: %v\n", err)
		return
	}
	s.manaSymbols = make([]*ebiten.Image, 5)
	for i := range 5 {
		s.manaSymbols[i] = imageutil.ScaleImage(statbutt[0][i], 0.35)
	}
	s.doneBtn = [3]*ebiten.Image{statbutt[0][11], statbutt[0][12], statbutt[0][13]}
}

func loadDuelImage(name string) *ebiten.Image {
	data, err := assets.DuelFS.ReadFile("art/sprites/duel/" + name)
	if err != nil {
		fmt.Printf("Error loading duel image %s: %v\n", name, err)
		return nil
	}
	img, err := imageutil.LoadImage(data)
	if err != nil {
		fmt.Printf("Error decoding duel image %s: %v\n", name, err)
		return nil
	}
	return img
}

func colorNameForDeck(primaryColor string) string {
	if primaryColor == "" {
		return "Red"
	}
	return primaryColor
}

const (
	duelPhaseX      = 0
	duelBoardX      = 82
	duelBoardW      = 942
	duelMsgY        = 370
	duelPlayerBoardY = 405
	duelOpponentBoardY = 0
)

func (s *DuelScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil, nil
	}

	mx, my := ebiten.CursorPosition()

	// Done button click (positioned at right of message bar)
	doneBounds := s.doneBtn[0].Bounds()
	doneX := duelBoardX + 2
	doneY := duelMsgY
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if mx >= doneX && mx < doneX+doneBounds.Dx() && my >= doneY && my < doneY+doneBounds.Dy() {
			select {
			case s.corePlayer.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
			default:
			}
		}
	}

	// Hand card selection
	handStartY := duelPlayerBoardY + 300
	if my >= handStartY && mx >= duelBoardX+20 && mx < duelBoardX+400 {
		cardIdx := (my - handStartY) / 20
		if cardIdx >= 0 && cardIdx < len(s.corePlayer.Hand) {
			s.selectedCardIdx = cardIdx
			s.loadCardPreview(s.corePlayer.Hand[cardIdx])
		}
	}

	// Check win/loss
	s.gameState.CheckWinConditions()
	if s.coreOpponent.HasLost {
		return s.handleWin()
	}
	if s.corePlayer.HasLost {
		return s.handleLoss()
	}

	return screenui.DuelScr, nil, nil
}

func (s *DuelScreen) runGameLoop() {
	core.PlayGame(s.gameState, 100)
	close(s.gameDone)
}

func (s *DuelScreen) runOpponentAI() {
	for {
		select {
		case <-s.gameDone:
			return
		case <-s.coreOpponent.WaitingChan:
		}

		if s.coreOpponent.HasLost || s.corePlayer.HasLost {
			return
		}

		actions := s.gameState.AvailableActions(s.coreOpponent)
		action := chooseAIAction(actions)

		select {
		case s.coreOpponent.InputChan <- action:
		case <-s.gameDone:
			return
		}
	}
}

func chooseAIAction(actions []core.PlayerAction) core.PlayerAction {
	castActions := []core.PlayerAction{}
	landActions := []core.PlayerAction{}
	attackActions := []core.PlayerAction{}
	blockActions := []core.PlayerAction{}

	for _, a := range actions {
		switch a.Type {
		case core.ActionCastSpell:
			if a.Card.CardType != domain.CardTypeLand {
				castActions = append(castActions, a)
			}
		case core.ActionPlayLand:
			landActions = append(landActions, a)
		case core.ActionDeclareAttacker:
			attackActions = append(attackActions, a)
		case core.ActionDeclareBlocker:
			blockActions = append(blockActions, a)
		}
	}

	if len(castActions) > 0 {
		return castActions[rand.Intn(len(castActions))]
	}
	if len(landActions) > 0 {
		return landActions[rand.Intn(len(landActions))]
	}
	if len(attackActions) > 0 {
		return attackActions[rand.Intn(len(attackActions))]
	}
	if len(blockActions) > 0 {
		return blockActions[rand.Intn(len(blockActions))]
	}

	return core.PlayerAction{Type: core.ActionPassPriority}
}

func (s *DuelScreen) loadCardPreview(card *core.Card) {
	img, err := card.CardImage(domain.CardViewFull)
	if err != nil || img == nil {
		s.cardPreviewImg = nil
		return
	}
	s.cardPreviewImg = imageutil.ScaleImage(img, 0.65)
}

func (s *DuelScreen) handleWin() (screenui.ScreenName, screenui.Screen, error) {
	s.lvl.RemoveEnemyAt(s.idx)

	if s.player.ActiveQuest != nil &&
		s.player.ActiveQuest.Type == domain.QuestTypeDefeatEnemy &&
		s.player.ActiveQuest.EnemyName == s.enemy.Character.Name {
		s.player.ActiveQuest.IsCompleted = true
	}

	enemyLevel := s.enemy.Character.Level
	cardCount := getRewardCardCount(enemyLevel)
	enemyDeck := s.enemy.Character.GetActiveDeck()
	wonCards := selectRewardCards(enemyDeck, cardCount)

	for _, card := range wonCards {
		if card != nil {
			s.player.CardCollection.AddCard(card, 1)
		}
	}

	return screenui.DuelWinScr, NewWinDuelScreen(wonCards), nil
}

func (s *DuelScreen) handleLoss() (screenui.ScreenName, screenui.Screen, error) {
	s.lvl.RemoveEnemyAt(s.idx)

	if s.anteCard != nil {
		_ = s.player.RemoveCard(s.anteCard)
	}
	lostCards := []*domain.Card{}
	if s.anteCard != nil {
		lostCards = append(lostCards, s.anteCard)
	}

	return screenui.DuelLoseScr, NewDuelLoseScreen(lostCards), nil
}

func (s *DuelScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.Fill(color.RGBA{30, 30, 30, 255})

	s.drawPhasePanel(screen)
	s.drawOpponentBoard(screen)
	s.drawMessageBar(screen)
	s.drawPlayerBoard(screen, H)
	s.drawSidebar(screen, W, H)
	s.drawCardPreview(screen, H)
}

func (s *DuelScreen) drawPhasePanel(screen *ebiten.Image) {
	if s.phaseDefaultBg == nil {
		return
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(0, 4)
	screen.DrawImage(s.phaseDefaultBg, opts)

	active := s.gameState.Players[s.gameState.ActivePlayer]
	idx := phaseIndex(active.Turn.Phase)
	var row int
	if active == s.corePlayer {
		row = 10 + idx
	} else {
		row = idx
	}
	if s.phaseActiveImgs[row] != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(0, float64(4+row*42))
		screen.DrawImage(s.phaseActiveImgs[row], opts)
	}
}

func phaseIndex(phase core.Phase) int {
	for i, p := range phaseNames {
		if p == phase {
			return i
		}
	}
	return 0
}

func (s *DuelScreen) drawOpponentBoard(screen *ebiten.Image) {
	if s.opponentBg != nil {
		opts := &ebiten.DrawImageOptions{}
		bgW := float64(s.opponentBg.Bounds().Dx())
		bgH := float64(s.opponentBg.Bounds().Dy())
		scaleX := float64(duelBoardW) / bgW
		scaleY := float64(duelMsgY) / bgH
		opts.GeoM.Scale(scaleX, scaleY)
		opts.GeoM.Translate(float64(duelBoardX), float64(duelOpponentBoardY))
		screen.DrawImage(s.opponentBg, opts)
	}

	// Opponent life and info
	lifeText := fmt.Sprintf("Life: %d", s.coreOpponent.LifeTotal)
	txt := elements.NewText(16, s.enemy.Character.Name, duelBoardX+10, 10)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	txt = elements.NewText(14, lifeText, duelBoardX+10, 32)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	handText := fmt.Sprintf("Hand: %d  Library: %d  Graveyard: %d",
		len(s.coreOpponent.Hand), len(s.coreOpponent.Library), len(s.coreOpponent.Graveyard))
	txt = elements.NewText(12, handText, duelBoardX+10, 52)
	txt.Color = color.RGBA{200, 200, 200, 255}
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	// Opponent battlefield
	s.drawBattlefield(screen, s.coreOpponent, duelBoardX+20, 80)
}

func (s *DuelScreen) drawMessageBar(screen *ebiten.Image) {
	// Done button
	if s.doneBtn[0] != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(duelBoardX+2), float64(duelMsgY))
		screen.DrawImage(s.doneBtn[0], opts)
	}

	// Message bar background
	if s.messageBg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(duelBoardX+52), float64(duelMsgY+6))
		screen.DrawImage(s.messageBg, opts)
	}

	// Phase description text
	phaseDesc := phaseDescription(s.corePlayer.Turn.Phase)
	txt := elements.NewText(14, phaseDesc, duelBoardX+60, duelMsgY+12)
	txt.Color = color.RGBA{255, 255, 255, 255}
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func (s *DuelScreen) drawPlayerBoard(screen *ebiten.Image, H int) {
	if s.playerBoardBg != nil {
		opts := &ebiten.DrawImageOptions{}
		bgW := float64(s.playerBoardBg.Bounds().Dx())
		bgH := float64(s.playerBoardBg.Bounds().Dy())
		boardH := float64(H - duelPlayerBoardY)
		scaleX := float64(duelBoardW) / bgW
		scaleY := boardH / bgH
		opts.GeoM.Scale(scaleX, scaleY)
		opts.GeoM.Translate(float64(duelBoardX), float64(duelPlayerBoardY))
		screen.DrawImage(s.playerBoardBg, opts)
	}

	// Player life
	lifeText := fmt.Sprintf("Life: %d", s.corePlayer.LifeTotal)
	txt := elements.NewText(16, "You", duelBoardX+10, duelPlayerBoardY+10)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	txt = elements.NewText(14, lifeText, duelBoardX+10, duelPlayerBoardY+32)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	// Player battlefield
	s.drawBattlefield(screen, s.corePlayer, duelBoardX+20, duelPlayerBoardY+60)

	// Player hand
	s.drawHand(screen, duelBoardX+20, duelPlayerBoardY+200)

	// Library / Graveyard counts
	infoText := fmt.Sprintf("Library: %d  Graveyard: %d", len(s.corePlayer.Library), len(s.corePlayer.Graveyard))
	txt = elements.NewText(12, infoText, duelBoardX+10, duelPlayerBoardY+52)
	txt.Color = color.RGBA{200, 200, 200, 255}
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func (s *DuelScreen) drawBattlefield(screen *ebiten.Image, player *core.Player, startX, startY int) {
	for i, card := range player.Battlefield {
		x := startX + (i%8)*100
		y := startY + (i/8)*25

		displayName := card.Name()
		if card.Tapped {
			displayName += " (T)"
		}
		if card.CardType == domain.CardTypeCreature {
			displayName += fmt.Sprintf(" %d/%d", card.EffectivePower(), card.EffectiveToughness())
		}

		txt := elements.NewText(12, displayName, x, y)
		if !card.Active && card.CardType == domain.CardTypeCreature {
			txt.Color = color.RGBA{150, 150, 150, 255}
		} else if card.Tapped {
			txt.Color = color.RGBA{180, 180, 120, 255}
		}
		txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}
}

func (s *DuelScreen) drawHand(screen *ebiten.Image, startX, startY int) {
	for i, card := range s.corePlayer.Hand {
		y := startY + i*20
		displayName := card.Name()
		if card.ManaCost != "" {
			displayName += " " + card.ManaCost
		}

		txt := elements.NewText(12, displayName, startX, y)
		if i == s.selectedCardIdx {
			txt.Color = color.RGBA{255, 255, 100, 255}
		}
		txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}
}

func (s *DuelScreen) drawSidebar(screen *ebiten.Image, W, H int) {
	// Mana pool display
	if s.manaPoolBg != nil {
		opts := &ebiten.DrawImageOptions{}
		poolScale := 0.6
		opts.GeoM.Scale(poolScale, poolScale)
		opts.GeoM.Translate(float64(duelBoardX+duelBoardW-90), float64(duelPlayerBoardY+10))
		screen.DrawImage(s.manaPoolBg, opts)
	}

	// Mana symbols with counts
	manaColors := []rune{'B', 'U', 'R', 'G', 'W'}
	for i, sym := range s.manaSymbols {
		if sym == nil {
			continue
		}
		x := duelBoardX + duelBoardW - 85
		y := duelPlayerBoardY + 30 + i*22

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(sym, opts)

		count := countManaOfColor(s.corePlayer.ManaPool, manaColors[i])
		countTxt := elements.NewText(10, fmt.Sprintf("%d", count), x+20, y+2)
		countTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}

	// Graveyard (in left sidebar, below phases)
	if s.graveyardImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(0.6, 0.6)
		opts.GeoM.Translate(40, float64(H-70))
		screen.DrawImage(s.graveyardImg, opts)
	}
}

func (s *DuelScreen) drawCardPreview(screen *ebiten.Image, H int) {
	if s.cardPreviewImg == nil {
		return
	}
	opts := &ebiten.DrawImageOptions{}
	previewX := float64(duelBoardX + duelBoardW - 260)
	previewY := float64(duelPlayerBoardY + 120)
	opts.GeoM.Translate(previewX, previewY)
	screen.DrawImage(s.cardPreviewImg, opts)
}

func phaseDescription(phase core.Phase) string {
	switch phase {
	case core.PhaseUntap:
		return "Untap phase: untap all permanents"
	case core.PhaseUpkeep:
		return "Upkeep phase: upkeep triggers"
	case core.PhaseDraw:
		return "Draw phase: draw a card"
	case core.PhaseMain1:
		return "Main phase (before combat): cast spells, play land"
	case core.PhaseCombat:
		return "Combat phase: declare attackers and blockers"
	case core.PhaseMain2:
		return "Main phase (after combat): cast spells, play land"
	case core.PhaseEnd:
		return "End phase: end of turn triggers"
	default:
		return ""
	}
}

func countManaOfColor(pool core.ManaPool, c rune) int {
	count := 0
	for _, mana := range pool {
		if len(mana) == 1 && mana[0] == c {
			count++
		}
	}
	return count
}
