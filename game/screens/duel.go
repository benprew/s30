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

type duelPlayer struct {
	core         *core.Player
	name         string
	boardBg      *ebiten.Image
	handBg       *ebiten.Image
	lifeBg       *ebiten.Image
	graveyardImg *ebiten.Image
	handX, handY int
}

type DuelScreen struct {
	player *domain.Player
	enemy  *domain.Enemy
	lvl    *world.Level
	idx    int

	gameState *core.GameState
	self      *duelPlayer
	opponent  *duelPlayer
	gameDone  chan struct{}

	phaseDefaultBg  *ebiten.Image
	phaseActiveImgs []*ebiten.Image
	bigCardBg       *ebiten.Image
	spellChainBg    *ebiten.Image
	messageBg       *ebiten.Image
	manaPoolBg      *ebiten.Image

	doneBtn [3]*ebiten.Image

	selectedCardIdx int
	cardPreviewImg  *ebiten.Image

	draggingHand *duelPlayer
	dragOffsetX  int
	dragOffsetY  int

	handCardCache map[core.EntityID]handCardEntry

	anteCard *domain.Card
}

type handCardEntry struct {
	source *ebiten.Image
	scaled *ebiten.Image
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
		handCardCache:   make(map[core.EntityID]handCardEntry),
	}

	s.initGameState()
	s.loadImages()

	s.self.handX = 860
	s.self.handY = 430
	s.opponent.handX = 860
	s.opponent.handY = 310

	return s
}

func (s *DuelScreen) initGameState() {
	entityID := core.EntityID(1)

	corePlayer := &core.Player{
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
			coreCard := core.NewCardFromDomain(card, entityID, corePlayer)
			corePlayer.Library = append(corePlayer.Library, coreCard)
			entityID++
		}
	}
	rand.Shuffle(len(corePlayer.Library), func(i, j int) {
		corePlayer.Library[i], corePlayer.Library[j] = corePlayer.Library[j], corePlayer.Library[i]
	})

	coreOpponent := &core.Player{
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
			coreCard := core.NewCardFromDomain(card, entityID, coreOpponent)
			coreOpponent.Library = append(coreOpponent.Library, coreCard)
			entityID++
		}
	}
	rand.Shuffle(len(coreOpponent.Library), func(i, j int) {
		coreOpponent.Library[i], coreOpponent.Library[j] = coreOpponent.Library[j], coreOpponent.Library[i]
	})

	s.self = &duelPlayer{core: corePlayer, name: "You"}
	s.opponent = &duelPlayer{core: coreOpponent, name: s.enemy.Character.Name}

	s.gameState = core.NewGame([]*core.Player{corePlayer, coreOpponent})
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
	s.self.boardBg = imageutil.ScaleImageInd(loadDuelImage(fmt.Sprintf("Terr_%smana.pic.png", playerColor)), 1.02, 1.01)
	s.self.handBg = loadDuelImage(fmt.Sprintf("Hand_%s.pic.png", playerColor))
	s.self.graveyardImg = loadDuelImage(fmt.Sprintf("Grave_%s.pic.png", playerColor))
	s.self.lifeBg = imageutil.ScaleImageInd(loadDuelImage(fmt.Sprintf("Life_%spict.pic.png", playerColor)), 1, 1.09)

	s.opponent.boardBg = imageutil.ScaleImageInd(loadDuelImage(fmt.Sprintf("Terr_%smana.pic.png", enemyColor)), 1.02, 1.01)
	s.opponent.handBg = loadDuelImage(fmt.Sprintf("Hand_%s.pic.png", enemyColor))
	s.opponent.graveyardImg = loadDuelImage(fmt.Sprintf("Grave_%s.pic.png", enemyColor))
	s.opponent.lifeBg = imageutil.ScaleImageInd(loadDuelImage(fmt.Sprintf("Life_%spict.pic.png", enemyColor)), 1, 1.09)

	s.manaPoolBg = loadDuelImage("Winbk_Manapool.pic.png")
	s.bigCardBg = loadDuelImage("Winbk_Bigcard.pic.png")
	s.spellChainBg = loadDuelImage("Winbk_Spellchain.pic.png")
	s.messageBg = loadDuelImage("Winbk_Telluser.pic.png")

	statbutt, err := imageutil.LoadSpriteSheet(16, 1, assets.Statbutt_png)
	if err != nil {
		fmt.Printf("Error loading Statbutt sprite sheet: %v\n", err)
		return
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
	duelPhaseX         = 0
	duelBoardX         = 293
	duelBoardW         = 721
	duelMsgY           = 370
	duelPlayerBoardY   = 384
	duelOpponentBoardY = 0
)

func (s *DuelScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil, nil
	}

	mx, my := ebiten.CursorPosition()

	if s.updateHandDrag(mx, my) {
		return screenui.DuelScr, nil, nil
	}

	// Done button click (positioned at right of message bar)
	doneBounds := s.doneBtn[0].Bounds()
	doneX := duelBoardX + 2
	doneY := duelMsgY
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if mx >= doneX && mx < doneX+doneBounds.Dx() && my >= doneY && my < doneY+doneBounds.Dy() {
			select {
			case s.self.core.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
			default:
			}
		}
	}

	// Hand card selection (click on card images below hand header)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		handBgW, handBgH := 145, 51
		if s.self.handBg != nil {
			b := s.self.handBg.Bounds()
			handBgW, handBgH = b.Dx(), b.Dy()
		}
		cardListY := s.self.handY + handBgH
		if my >= cardListY && mx >= s.self.handX && mx < s.self.handX+handBgW {
			cardIdx := (my - cardListY) / handCardOverlap
			if cardIdx >= len(s.self.core.Hand) {
				cardIdx = len(s.self.core.Hand) - 1
			}
			if cardIdx >= 0 && cardIdx < len(s.self.core.Hand) {
				s.selectedCardIdx = cardIdx
				s.loadCardPreview(s.self.core.Hand[cardIdx])
			}
		}
	}

	// Check win/loss
	s.gameState.CheckWinConditions()
	if s.opponent.core.HasLost {
		return s.handleWin()
	}
	if s.self.core.HasLost {
		return s.handleLoss()
	}

	return screenui.DuelScr, nil, nil
}

const handCardOverlap = 20

func (s *DuelScreen) getHandCardImg(card *core.Card, targetW int) *ebiten.Image {
	artImg, err := card.CardImage(domain.CardViewArtOnly)
	if err != nil || artImg == nil {
		return nil
	}

	entry, ok := s.handCardCache[card.ID]
	if ok && entry.source == artImg {
		return entry.scaled
	}

	scale := float64(targetW) / float64(artImg.Bounds().Dx())
	scaled := imageutil.ScaleImage(artImg, scale)
	s.handCardCache[card.ID] = handCardEntry{source: artImg, scaled: scaled}
	return scaled
}

func (s *DuelScreen) handHeaderBounds(dp *duelPlayer) image.Rectangle {
	w, h := 145, 51
	if dp.handBg != nil {
		b := dp.handBg.Bounds()
		w, h = b.Dx(), b.Dy()
	}
	return image.Rect(dp.handX, dp.handY, dp.handX+w, dp.handY+h)
}

func (s *DuelScreen) updateHandDrag(mx, my int) bool {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && s.draggingHand != nil {
		s.draggingHand = nil
		return false
	}

	if s.draggingHand != nil {
		s.draggingHand.handX = mx - s.dragOffsetX
		s.draggingHand.handY = my - s.dragOffsetY
		return true
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		pt := image.Pt(mx, my)
		if pt.In(s.handHeaderBounds(s.self)) {
			s.draggingHand = s.self
			s.dragOffsetX = mx - s.self.handX
			s.dragOffsetY = my - s.self.handY
			return true
		}
		if pt.In(s.handHeaderBounds(s.opponent)) {
			s.draggingHand = s.opponent
			s.dragOffsetX = mx - s.opponent.handX
			s.dragOffsetY = my - s.opponent.handY
			return true
		}
	}

	return false
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
		case <-s.opponent.core.WaitingChan:
		}

		if s.opponent.core.HasLost || s.self.core.HasLost {
			return
		}

		actions := s.gameState.AvailableActions(s.opponent.core)
		action := chooseAIAction(actions)

		select {
		case s.opponent.core.InputChan <- action:
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
	s.drawBoard(screen, s.opponent, duelOpponentBoardY, duelMsgY)
	s.drawBoard(screen, s.self, duelPlayerBoardY, H-duelPlayerBoardY)
	s.drawMessageBar(screen)
	s.drawSidebar(screen, W, H)
	s.drawHandPanel(screen, s.opponent)
	s.drawHandPanel(screen, s.self)
	s.drawCardPreview(screen, H)
}

func (s *DuelScreen) drawPhasePanel(screen *ebiten.Image) {
	if s.phaseDefaultBg == nil {
		return
	}

	const phaseX = 250

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(phaseX, 4)
	screen.DrawImage(s.phaseDefaultBg, opts)

	active := s.gameState.Players[s.gameState.ActivePlayer]
	idx := phaseIndex(active.Turn.Phase)
	var row int
	if active == s.self.core {
		row = 10 + idx
	} else {
		row = idx
	}
	if s.phaseActiveImgs[row] != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(phaseX, float64(4+row*42))
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

func (s *DuelScreen) drawBoard(screen *ebiten.Image, dp *duelPlayer, startY, boardH int) {
	if dp.boardBg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(duelBoardX), float64(startY))
		screen.DrawImage(dp.boardBg, opts)
	}

	lifeText := fmt.Sprintf("Life: %d", dp.core.LifeTotal)
	txt := elements.NewText(16, dp.name, duelBoardX+10, startY+10)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	txt = elements.NewText(14, lifeText, duelBoardX+10, startY+32)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	infoText := fmt.Sprintf("Hand: %d  Library: %d  Graveyard: %d",
		len(dp.core.Hand), len(dp.core.Library), len(dp.core.Graveyard))
	txt = elements.NewText(12, infoText, duelBoardX+10, startY+52)
	txt.Color = color.RGBA{200, 200, 200, 255}
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	s.drawBattlefield(screen, dp.core, duelBoardX+20, startY+80)
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
	phaseDesc := phaseDescription(s.self.core.Turn.Phase)
	txt := elements.NewText(14, phaseDesc, duelBoardX+60, duelMsgY+12)
	txt.Color = color.RGBA{255, 255, 255, 255}
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

func (s *DuelScreen) drawHandPanel(screen *ebiten.Image, dp *duelPlayer) {
	if dp.handBg == nil {
		return
	}

	handBgW := dp.handBg.Bounds().Dx()
	handBgH := dp.handBg.Bounds().Dy()

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(dp.handX), float64(dp.handY))
	screen.DrawImage(dp.handBg, opts)

	label := fmt.Sprintf("Your Hand (%d)", len(dp.core.Hand))
	txt := elements.NewText(16, label, dp.handX+15, dp.handY+13)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if dp != s.self {
		return
	}

	for i, card := range dp.core.Hand {
		y := dp.handY + handBgH + i*handCardOverlap
		cardImg := s.getHandCardImg(card, handBgW)
		if cardImg == nil {
			continue
		}
		cardOpts := &ebiten.DrawImageOptions{}
		cardOpts.GeoM.Translate(float64(dp.handX), float64(y))
		screen.DrawImage(cardImg, cardOpts)
	}
}

func drawManaPool(screen, manaPoolBg *ebiten.Image, player *duelPlayer, manaPoolY int) {
	const manaPoolX = 120

	// Mana pool display
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(manaPoolX, float64(manaPoolY))
	screen.DrawImage(manaPoolBg, opts)

	manaColors := []rune{'B', 'U', 'G', 'R', 'W', 'C'}
	for i := range manaColors {
		x := manaPoolX + 50
		y := (i * 30) + 10
		count := countManaOfColor(player.core.ManaPool, manaColors[i])
		countTxt := elements.NewText(24, fmt.Sprintf("%d", count), x, manaPoolY+y)
		countTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}
}

func drawLife(screen *ebiten.Image, player *duelPlayer, Y int) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(0, float64(Y))
	screen.DrawImage(player.lifeBg, opts)
	countTxt := elements.NewText(64, fmt.Sprintf("%d", player.core.LifeTotal), 15, Y)
	countTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func drawGraveyard(screen *ebiten.Image, player *duelPlayer, Y float64) {
	// Graveyard (in left sidebar, below phases)
	if player.graveyardImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(60, Y)
		screen.DrawImage(player.graveyardImg, opts)
	}
}

func (s *DuelScreen) drawSidebar(screen *ebiten.Image, W, H int) {
	drawManaPool(screen, s.manaPoolBg, s.opponent, 0)
	drawManaPool(screen, s.manaPoolBg, s.self, 580)

	drawGraveyard(screen, s.opponent, 94)
	drawGraveyard(screen, s.self, 580)

	drawLife(screen, s.opponent, 0)
	drawLife(screen, s.self, 671)
}

func (s *DuelScreen) drawCardPreview(screen *ebiten.Image, H int) {
	if s.cardPreviewImg == nil {
		return
	}
	opts := &ebiten.DrawImageOptions{}
	previewX := float64(0)
	previewY := float64(188)
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
