package screens

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/mtg/ai"
	"github.com/benprew/s30/mtg/core"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

	mouseState    mouseStateType
	mouseStartX   int
	mouseStartY   int
	dragTargetX   *int
	dragTargetY   *int
	dragOffsetX   int
	dragOffsetY   int
	draggingCardID core.EntityID
	cardPositions  map[core.EntityID]image.Point

	cardImgCache map[cardImgKey]cardImgEntry

	cardActions      map[core.EntityID]core.PlayerAction
	pendingAttackers map[core.EntityID]bool
	pendingBlockers  map[core.EntityID]core.EntityID
	selectedBlocker  core.EntityID

	targetingCard    *core.Card
	targetingActions map[int]core.PlayerAction
	selectedTarget   core.Targetable

	frameCount int

	anteCard *domain.Card
}

type mouseStateType int

const (
	mouseIdle mouseStateType = iota
	mousePotentialDrag
	mouseDragging
)

type cardImgKey struct {
	id    core.EntityID
	width int
}

type cardImgEntry struct {
	source *ebiten.Image
	scaled *ebiten.Image
}

func (s *DuelScreen) IsFramed() bool { return false }

func NewDuelScreen(player *domain.Player, enemy *domain.Enemy, lvl *world.Level, idx int, anteCard *domain.Card) *DuelScreen {
	s := &DuelScreen{
		player:           player,
		enemy:            enemy,
		lvl:              lvl,
		idx:              idx,
		selectedCardIdx:  -1,
		anteCard:         anteCard,
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[core.EntityID]image.Point),
		pendingAttackers: make(map[core.EntityID]bool),
		pendingBlockers:  make(map[core.EntityID]core.EntityID),
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

	s.gameState = core.NewGame([]*core.Player{corePlayer, coreOpponent}, false)
	s.gameState.StartGame()

	s.gameDone = make(chan struct{})
	go s.runOpponentAI()
	go s.runAutoPass()
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
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && s.targetingCard == nil {
		s.submitPendingAndPass()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if s.targetingCard != nil {
			if s.selectedTarget != nil {
				s.selectedTarget = nil
			} else {
				s.exitTargetingMode()
			}
		} else {
			return screenui.WorldScr, nil, nil
		}
	}

	s.frameCount++
	s.refreshCardActions()

	if s.targetingCard != nil {
		if _, ok := s.cardActions[s.targetingCard.ID]; !ok {
			s.exitTargetingMode()
		}
	}

	mx, my := ebiten.CursorPosition()

	if s.targetingCard != nil {
		s.updateTargetingMouse(mx, my)
	} else {
		s.updateMouse(mx, my)
		s.updateHoverPreview(mx, my)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		s.handleRightClick(mx, my)
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

const (
	handCardOverlap = 20
	fieldCardW      = 100
	fieldCardH      = 83
)

func (s *DuelScreen) getFieldCardPos(card *core.Card, dp *duelPlayer, idx int) image.Point {
	if pos, ok := s.cardPositions[card.ID]; ok {
		return pos
	}
	baseY := duelPlayerBoardY + 20
	if dp == s.opponent {
		baseY = duelOpponentBoardY + 70
	}
	pos := image.Pt(duelBoardX+30+idx*35, baseY)
	s.cardPositions[card.ID] = pos
	return pos
}

func (s *DuelScreen) fieldCardAtPoint(mx, my int, dp *duelPlayer) *core.Card {
	for i := len(dp.core.Battlefield) - 1; i >= 0; i-- {
		card := dp.core.Battlefield[i]
		pos := s.getFieldCardPos(card, dp, i)
		if mx >= pos.X && mx < pos.X+fieldCardW && my >= pos.Y && my < pos.Y+fieldCardH {
			return card
		}
	}
	return nil
}

func (s *DuelScreen) refreshCardActions() {
	activePlayer := s.gameState.Players[s.gameState.ActivePlayer]
	inDeclareAttackers := activePlayer == s.self.core &&
		activePlayer.Turn.Phase == core.PhaseCombat &&
		activePlayer.Turn.CombatStep == core.CombatStepDeclareAttackers
	if !inDeclareAttackers && len(s.pendingAttackers) > 0 {
		s.pendingAttackers = make(map[core.EntityID]bool)
	}
	if !s.isInDeclareBlockers() && (len(s.pendingBlockers) > 0 || s.selectedBlocker != 0) {
		s.pendingBlockers = make(map[core.EntityID]core.EntityID)
		s.selectedBlocker = 0
	}
	s.cardActions = make(map[core.EntityID]core.PlayerAction)
	for _, action := range s.gameState.AvailableActions(s.self.core) {
		if action.Card == nil {
			continue
		}
		s.cardActions[action.Card.ID] = action
	}
	for _, card := range s.self.core.Battlefield {
		if _, exists := s.cardActions[card.ID]; exists {
			continue
		}
		if !card.Tapped && card.GetManaAbility() != nil {
			s.cardActions[card.ID] = core.PlayerAction{Type: "ActivateMana", Card: card}
		}
	}
}

func (s *DuelScreen) panelCardW(dp *duelPlayer) int {
	if dp.handBg != nil {
		return dp.handBg.Bounds().Dx()
	}
	return 145
}

func (s *DuelScreen) panelCardH(dp *duelPlayer) int {
	if dp.handBg != nil {
		return dp.handBg.Bounds().Dy()
	}
	return 51
}

func (s *DuelScreen) cardIdxAtPoint(mx, my, panelX, panelY int, cards []*core.Card, dp *duelPlayer) int {
	headerH := s.panelCardH(dp)
	w := s.panelCardW(dp)
	cardListY := panelY + headerH
	if my < cardListY || mx < panelX || mx >= panelX+w {
		return -1
	}
	idx := (my - cardListY) / handCardOverlap
	if idx >= len(cards) {
		idx = len(cards) - 1
	}
	if idx < 0 {
		return -1
	}
	return idx
}

func (s *DuelScreen) isInDeclareBlockers() bool {
	activePlayer := s.gameState.Players[s.gameState.ActivePlayer]
	return activePlayer == s.opponent.core &&
		activePlayer.Turn.Phase == core.PhaseCombat &&
		activePlayer.Turn.CombatStep == core.CombatStepDeclareBlockers
}

func (s *DuelScreen) findBattlefieldCard(dp *duelPlayer, id core.EntityID) *core.Card {
	for _, card := range dp.core.Battlefield {
		if card.ID == id {
			return card
		}
	}
	return nil
}

func (s *DuelScreen) isValidBlock(blockerID, attackerID core.EntityID) bool {
	for _, action := range s.gameState.AvailableActions(s.self.core) {
		if action.Type != core.ActionDeclareBlocker || action.Card == nil {
			continue
		}
		targetCard, ok := action.Target.(*core.Card)
		if !ok {
			continue
		}
		if action.Card.ID == blockerID && targetCard.ID == attackerID {
			return true
		}
	}
	return false
}

func (s *DuelScreen) canBlockAnything(blockerID core.EntityID) bool {
	for _, action := range s.gameState.AvailableActions(s.self.core) {
		if action.Type == core.ActionDeclareBlocker && action.Card != nil && action.Card.ID == blockerID {
			return true
		}
	}
	return false
}

func (s *DuelScreen) handleBlockerClick(mx, my int) {
	if card := s.fieldCardAtPoint(mx, my, s.self); card != nil {
		s.loadCardPreview(card)
		if _, assigned := s.pendingBlockers[card.ID]; assigned {
			delete(s.pendingBlockers, card.ID)
			return
		}
		if s.selectedBlocker == card.ID {
			s.selectedBlocker = 0
			return
		}
		if s.canBlockAnything(card.ID) {
			s.selectedBlocker = card.ID
		}
		return
	}

	if card := s.fieldCardAtPoint(mx, my, s.opponent); card != nil {
		s.loadCardPreview(card)
		if s.selectedBlocker != 0 && s.isValidBlock(s.selectedBlocker, card.ID) {
			s.pendingBlockers[s.selectedBlocker] = card.ID
			s.selectedBlocker = 0
		}
		return
	}
}

func (s *DuelScreen) handleCardClick(mx, my int) {
	// hand cards
	if idx := s.cardIdxAtPoint(mx, my, s.self.handX, s.self.handY, s.self.core.Hand, s.self); idx >= 0 {
		fmt.Printf("HAND CLICK: idx: %d\n", idx)
		s.selectedCardIdx = idx
		s.performCardAction(s.self.core.Hand[idx])
		return
	}

	if s.isInDeclareBlockers() {
		s.handleBlockerClick(mx, my)
		return
	}

	// battlefield cards
	if card := s.fieldCardAtPoint(mx, my, s.self); card != nil {
		s.loadCardPreview(card)
		fmt.Printf("CLICK: card: %s, actions: %v\n", card.Name(), s.cardActions[card.ID])
		if action, ok := s.cardActions[card.ID]; ok && action.Type == core.ActionDeclareAttacker {
			if s.pendingAttackers[card.ID] {
				delete(s.pendingAttackers, card.ID)
			} else {
				s.pendingAttackers[card.ID] = true
			}
			return
		}
		s.performCardAction(card)
		return
	} else {
		fmt.Printf("No card at %d, %d\n", mx, my)
	}
}

func (s *DuelScreen) handleRightClick(mx, my int) {
	if idx := s.cardIdxAtPoint(mx, my, s.self.handX, s.self.handY, s.self.core.Hand, s.self); idx >= 0 {
		s.loadCardPreview(s.self.core.Hand[idx])
		return
	}

	if card := s.fieldCardAtPoint(mx, my, s.self); card != nil {
		s.loadCardPreview(card)
		return
	}

	if card := s.fieldCardAtPoint(mx, my, s.opponent); card != nil {
		s.loadCardPreview(card)
		return
	}
}

func (s *DuelScreen) updateHoverPreview(mx, my int) {
	if idx := s.cardIdxAtPoint(mx, my, s.self.handX, s.self.handY, s.self.core.Hand, s.self); idx >= 0 {
		s.loadCardPreview(s.self.core.Hand[idx])
		return
	}

	if card := s.fieldCardAtPoint(mx, my, s.self); card != nil {
		s.loadCardPreview(card)
		return
	}

	if card := s.fieldCardAtPoint(mx, my, s.opponent); card != nil {
		s.loadCardPreview(card)
		return
	}
}

func (s *DuelScreen) performCardAction(card *core.Card) {
	action, ok := s.cardActions[card.ID]
	if !ok {
		return
	}

	if action.Type == "ActivateMana" {
		s.gameState.ActivateManaAbility(s.self.core, card)
		s.refreshCardActions()
		return
	}

	if card.GetTargetSpec() != nil {
		s.enterTargetingMode(card)
		return
	}

	select {
	case s.self.core.InputChan <- action:
	default:
	}
	s.refreshCardActions()
}

func (s *DuelScreen) enterTargetingMode(card *core.Card) {
	targets := s.gameState.AvailableTargets(card)
	s.targetingCard = card
	s.targetingActions = make(map[int]core.PlayerAction)
	for _, t := range targets {
		s.targetingActions[t.EntityID()] = core.PlayerAction{
			Type:   core.ActionCastSpell,
			Card:   card,
			Target: t,
		}
	}
	s.selectedTarget = nil
	s.loadCardPreview(card)
}

func (s *DuelScreen) exitTargetingMode() {
	s.targetingCard = nil
	s.targetingActions = nil
	s.selectedTarget = nil
}

func (s *DuelScreen) updateTargetingMouse(mx, my int) {
	if s.mouseState != mouseIdle {
		s.dragTargetX = nil
		s.dragTargetY = nil
		s.draggingCardID = 0
		s.mouseState = mouseIdle
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	doneBounds := s.doneBtn[0].Bounds()
	doneX := duelBoardX + 2
	doneY := duelMsgY
	if mx >= doneX && mx < doneX+doneBounds.Dx() && my >= doneY && my < doneY+doneBounds.Dy() {
		if s.selectedTarget != nil {
			if action, ok := s.targetingActions[s.selectedTarget.EntityID()]; ok {
				select {
				case s.self.core.InputChan <- action:
				default:
				}
			}
			s.exitTargetingMode()
		} else {
			s.exitTargetingMode()
		}
		return
	}

	msgBarTop := duelMsgY + 6
	msgBarLeft := duelBoardX + 52
	if s.selectedTarget != nil && mx >= msgBarLeft && my >= msgBarTop && my < msgBarTop+20 {
		s.selectedTarget = nil
		return
	}

	s.handleTargetClick(mx, my)
}

func (s *DuelScreen) handleTargetClick(mx, my int) {
	for _, dp := range []*duelPlayer{s.opponent, s.self} {
		if card := s.fieldCardAtPoint(mx, my, dp); card != nil {
			if _, ok := s.targetingActions[card.EntityID()]; ok {
				s.selectedTarget = card
				s.loadCardPreview(card)
				return
			}
		}
	}

	for _, dp := range []*duelPlayer{s.opponent, s.self} {
		if s.isPlayerBoardClick(mx, my, dp) {
			if _, ok := s.targetingActions[dp.core.EntityID()]; ok {
				s.selectedTarget = dp.core
				return
			}
		}
	}
}

func (s *DuelScreen) isPlayerBoardClick(mx, my int, dp *duelPlayer) bool {
	if mx < duelBoardX || mx >= duelBoardX+duelBoardW {
		return false
	}
	if dp == s.self {
		return my >= duelPlayerBoardY
	}
	return my >= duelOpponentBoardY && my < duelMsgY
}

func (s *DuelScreen) getCardArtImg(card *core.Card, targetW int) *ebiten.Image {
	artImg, err := card.CardImage(domain.CardViewArtOnly)
	if err != nil || artImg == nil {
		return nil
	}

	key := cardImgKey{id: card.ID, width: targetW}
	entry, ok := s.cardImgCache[key]
	if ok && entry.source == artImg {
		return entry.scaled
	}

	scale := float64(targetW) / float64(artImg.Bounds().Dx())
	scaled := imageutil.ScaleImage(artImg, scale)
	s.cardImgCache[key] = cardImgEntry{source: artImg, scaled: scaled}
	return scaled
}

func (s *DuelScreen) panelHeaderBounds(dp *duelPlayer, px, py int) image.Rectangle {
	w, h := 145, 51
	if dp.handBg != nil {
		b := dp.handBg.Bounds()
		w, h = b.Dx(), b.Dy()
	}
	return image.Rect(px, py, px+w, py+h)
}

const dragThreshold = 4

func (s *DuelScreen) updateMouse(mx, my int) {
	switch s.mouseState {
	case mouseIdle:
		if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return
		}
		s.mouseStartX = mx
		s.mouseStartY = my

		// Hand panel headers start dragging immediately
		pt := image.Pt(mx, my)
		type dragTarget struct {
			bounds image.Rectangle
			x, y   *int
		}
		targets := []dragTarget{
			{s.panelHeaderBounds(s.self, s.self.handX, s.self.handY), &s.self.handX, &s.self.handY},
			{s.panelHeaderBounds(s.opponent, s.opponent.handX, s.opponent.handY), &s.opponent.handX, &s.opponent.handY},
		}
		for _, t := range targets {
			if pt.In(t.bounds) {
				s.dragTargetX = t.x
				s.dragTargetY = t.y
				s.dragOffsetX = mx - *t.x
				s.dragOffsetY = my - *t.y
				s.mouseState = mouseDragging
				return
			}
		}

		// Battlefield cards enter potential drag state
		for _, dp := range []*duelPlayer{s.self, s.opponent} {
			if card := s.fieldCardAtPoint(mx, my, dp); card != nil {
				s.draggingCardID = card.ID
				s.mouseState = mousePotentialDrag
				return
			}
		}

		// Everything else is an immediate click (hand cards, done button, etc.)
		s.handleClick(mx, my)

	case mousePotentialDrag:
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			s.handleClick(s.mouseStartX, s.mouseStartY)
			s.draggingCardID = 0
			s.mouseState = mouseIdle
			return
		}
		dx := mx - s.mouseStartX
		dy := my - s.mouseStartY
		if dx*dx+dy*dy >= dragThreshold*dragThreshold {
			pos := s.cardPositions[s.draggingCardID]
			s.dragOffsetX = s.mouseStartX - pos.X
			s.dragOffsetY = s.mouseStartY - pos.Y
			s.mouseState = mouseDragging
		}

	case mouseDragging:
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			s.dragTargetX = nil
			s.dragTargetY = nil
			s.draggingCardID = 0
			s.mouseState = mouseIdle
			return
		}
		if s.draggingCardID != 0 {
			pos := s.cardPositions[s.draggingCardID]
			pos.X = mx - s.dragOffsetX
			pos.Y = my - s.dragOffsetY
			s.cardPositions[s.draggingCardID] = pos
		}
		if s.dragTargetX != nil {
			*s.dragTargetX = mx - s.dragOffsetX
			*s.dragTargetY = my - s.dragOffsetY
		}
	}
}

func (s *DuelScreen) submitPendingAndPass() {
	for id, action := range s.cardActions {
		if action.Type == core.ActionDeclareAttacker && s.pendingAttackers[id] {
			select {
			case s.self.core.InputChan <- action:
			default:
			}
		}
	}
	s.pendingAttackers = make(map[core.EntityID]bool)

	for blockerID, attackerID := range s.pendingBlockers {
		blocker := s.findBattlefieldCard(s.self, blockerID)
		attacker := s.findBattlefieldCard(s.opponent, attackerID)
		if blocker != nil && attacker != nil {
			select {
			case s.self.core.InputChan <- core.PlayerAction{
				Type:   core.ActionDeclareBlocker,
				Card:   blocker,
				Target: attacker,
			}:
			default:
			}
		}
	}
	s.pendingBlockers = make(map[core.EntityID]core.EntityID)
	s.selectedBlocker = 0

	select {
	case s.self.core.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
	default:
	}
}

func (s *DuelScreen) handleClick(mx, my int) {
	// Done button
	doneBounds := s.doneBtn[0].Bounds()
	doneX := duelBoardX + 2
	doneY := duelMsgY
	if mx >= doneX && mx < doneX+doneBounds.Dx() && my >= doneY && my < doneY+doneBounds.Dy() {
		s.submitPendingAndPass()
		return
	}

	s.handleCardClick(mx, my)
}

func (s *DuelScreen) runGameLoop() {
	core.PlayGame(s.gameState, 100)
	close(s.gameDone)
}

func (s *DuelScreen) runAutoPass() {
	for {
		select {
		case <-s.gameDone:
			return
		case <-s.self.core.WaitingChan:
		}

		actions := s.gameState.AvailableActions(s.self.core)
		onlyPass := true
		for _, a := range actions {
			if a.Type != core.ActionPassPriority {
				onlyPass = false
				break
			}
		}

		if onlyPass {
			select {
			case <-time.After(50 * time.Millisecond):
			case <-s.gameDone:
				return
			}
			select {
			case s.self.core.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
			case <-s.gameDone:
				return
			}
		}
	}
}

func (s *DuelScreen) runOpponentAI() {
	ai.RunAI(s.gameState, s.opponent.core, s.gameDone)
}

func (s *DuelScreen) loadCardPreview(card *core.Card) {
	img, err := card.CardImage(domain.CardViewFull)
	if err != nil || img == nil {
		s.cardPreviewImg = nil
		return
	}
	s.cardPreviewImg = imageutil.ScaleImage(img, 0.95)
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
	s.drawBattlefield(screen, s.opponent)
	s.drawBattlefield(screen, s.self)
	s.drawBlockerArrows(screen)
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

	if dp == s.self {
		actions := s.gameState.AvailableActions(dp.core)
		y := startY + 68
		for _, a := range actions {
			name := ""
			if a.Card != nil {
				name = a.Card.Name()
			}
			line := fmt.Sprintf("%s %s", a.Type, name)
			at := elements.NewText(10, line, duelBoardX+10, y)
			at.Color = color.RGBA{180, 180, 180, 255}
			at.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
			y += 12
		}
	}

	if s.targetingCard != nil {
		if _, isTarget := s.targetingActions[dp.core.EntityID()]; isTarget {
			borderColor := color.RGBA{255, 255, 0, 255}
			strokeW := float32(2)
			if s.selectedTarget != nil && s.selectedTarget.EntityID() == dp.core.EntityID() {
				borderColor = color.RGBA{0, 255, 0, 255}
				strokeW = 3
			}
			vector.StrokeRect(screen, float32(duelBoardX), float32(startY),
				float32(duelBoardW), float32(boardH), strokeW, borderColor, false)
		}
	}
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

	var msg string
	if s.targetingCard != nil && s.selectedTarget != nil {
		msg = fmt.Sprintf("targeting %s (Cancel)", s.selectedTarget.Name())
	} else if s.targetingCard != nil {
		msg = fmt.Sprintf("Choose a target for %s", s.targetingCard.Name())
	} else {
		msg = s.statusMessage()
	}
	txt := elements.NewText(14, msg, duelBoardX+60, duelMsgY+12)
	txt.Color = color.RGBA{255, 255, 255, 255}
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func (s *DuelScreen) drawBattlefield(screen *ebiten.Image, dp *duelPlayer) {
	for i, card := range dp.core.Battlefield {
		if card == nil {
			continue
		}
		pos := s.getFieldCardPos(card, dp, i)
		cardImg := s.getCardArtImg(card, fieldCardW)
		if cardImg == nil {
			continue
		}

		cardOpts := &ebiten.DrawImageOptions{}
		if card.Tapped {
			imgH := float64(cardImg.Bounds().Dy())
			cardOpts.GeoM.Rotate(math.Pi / 2)
			cardOpts.GeoM.Translate(imgH, 0)
		}
		cardOpts.GeoM.Translate(float64(pos.X), float64(pos.Y))
		screen.DrawImage(cardImg, cardOpts)

		if _, hasAction := s.cardActions[card.ID]; hasAction && dp == s.self && s.targetingCard == nil {
			star := elements.NewText(14, "*", pos.X+2, pos.Y+2)
			star.Color = color.RGBA{255, 255, 0, 255}
			star.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}

		if s.pendingAttackers[card.ID] && dp == s.self {
			vector.StrokeRect(screen,
				float32(pos.X), float32(pos.Y),
				float32(fieldCardW), float32(fieldCardH),
				2, color.RGBA{255, 255, 0, 255}, false)
		}

		if dp == s.self && (s.selectedBlocker == card.ID || s.pendingBlockers[card.ID] != 0) {
			vector.StrokeRect(screen,
				float32(pos.X), float32(pos.Y),
				float32(fieldCardW), float32(fieldCardH),
				2, color.RGBA{255, 0, 0, 255}, false)
		}

		if dp == s.opponent && s.isInDeclareBlockers() {
			isAttacker := false
			for _, atk := range s.gameState.Attackers {
				if atk.ID == card.ID {
					isAttacker = true
					break
				}
			}
			if isAttacker {
				borderColor := color.RGBA{255, 255, 0, 255}
				if s.selectedBlocker != 0 && s.isValidBlock(s.selectedBlocker, card.ID) {
					borderColor = color.RGBA{0, 255, 0, 255}
				}
				vector.StrokeRect(screen,
					float32(pos.X), float32(pos.Y),
					float32(fieldCardW), float32(fieldCardH),
					2, borderColor, false)
			}
		}

		if s.targetingCard != nil {
			if _, isTarget := s.targetingActions[card.EntityID()]; isTarget {
				borderColor := color.RGBA{255, 255, 0, 255}
				strokeW := float32(2)
				if s.selectedTarget != nil && s.selectedTarget.EntityID() == card.EntityID() {
					borderColor = color.RGBA{0, 255, 0, 255}
					strokeW = 3
				}
				vector.StrokeRect(screen, float32(pos.X), float32(pos.Y),
					float32(fieldCardW), float32(fieldCardH), strokeW, borderColor, false)
			}
		}
	}
}

func (s *DuelScreen) drawBlockerArrows(screen *ebiten.Image) {
	for blockerID, attackerID := range s.pendingBlockers {
		blockerPos, bOK := s.cardPositions[blockerID]
		attackerPos, aOK := s.cardPositions[attackerID]
		if !bOK || !aOK {
			continue
		}

		bx := float32(blockerPos.X) + float32(fieldCardW)/2
		by := float32(blockerPos.Y)
		ax := float32(attackerPos.X) + float32(fieldCardW)/2
		ay := float32(attackerPos.Y) + float32(fieldCardH)

		lineColor := color.RGBA{255, 0, 0, 255}
		vector.StrokeLine(screen, bx, by, ax, ay, 2, lineColor, false)

		dx := bx - ax
		dy := by - ay
		length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if length == 0 {
			continue
		}
		dx /= length
		dy /= length

		arrowLen := float32(10)
		px := -dy
		py := dx
		vector.StrokeLine(screen, ax, ay, ax+dx*arrowLen+px*arrowLen*0.5, ay+dy*arrowLen+py*arrowLen*0.5, 2, lineColor, false)
		vector.StrokeLine(screen, ax, ay, ax+dx*arrowLen-px*arrowLen*0.5, ay+dy*arrowLen-py*arrowLen*0.5, 2, lineColor, false)
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
		if card == nil {
			continue
		}
		y := dp.handY + handBgH + i*handCardOverlap
		cardImg := s.getCardArtImg(card, handBgW)
		if cardImg == nil {
			continue
		}
		cardOpts := &ebiten.DrawImageOptions{}
		cardOpts.GeoM.Translate(float64(dp.handX), float64(y))
		screen.DrawImage(cardImg, cardOpts)

		if _, hasAction := s.cardActions[card.ID]; hasAction {
			star := elements.NewText(14, "*", dp.handX+2, y+2)
			star.Color = color.RGBA{255, 255, 0, 255}
			star.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}
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

func (s *DuelScreen) stackDescription() string {
	stack := s.gameState.Stack
	if stack.IsEmpty() {
		return ""
	}
	var names []string
	for _, item := range stack.Items {
		if item.Card != nil {
			names = append(names, item.Card.Name())
		}
	}
	if len(names) == 0 {
		return ""
	}
	return fmt.Sprintf("Respond to %s. ", strings.Join(names, ", "))
}

func (s *DuelScreen) statusMessage() string {
	active := s.gameState.Players[s.gameState.ActivePlayer]
	isMyTurn := active == s.self.core
	phase := active.Turn.Phase

	if s.self.core.Turn.Discarding {
		return fmt.Sprintf("Hand size > max hand size. Choose a card to discard. (%d cards)", len(s.self.core.Hand))
	}

	stackMsg := s.stackDescription()

	if phase == core.PhaseCombat {
		return stackMsg + s.combatStatusMessage(active, isMyTurn)
	}

	prefix := "Your"
	if !isMyTurn {
		prefix = s.opponent.name + "'s"
	}

	switch phase {
	case core.PhaseUntap:
		return stackMsg + fmt.Sprintf("%s turn - Untap", prefix)
	case core.PhaseUpkeep:
		return stackMsg + fmt.Sprintf("%s turn - Upkeep", prefix)
	case core.PhaseDraw:
		return stackMsg + fmt.Sprintf("%s turn - Draw", prefix)
	case core.PhaseMain1:
		if isMyTurn {
			return stackMsg + "Main phase: play a land or cast spells. Done to go to combat."
		}
		return stackMsg + fmt.Sprintf("%s main phase. Cast instants or Done to pass.", prefix)
	case core.PhaseMain2:
		if isMyTurn {
			return stackMsg + "Main phase 2: play a land or cast spells. Done to end turn."
		}
		return stackMsg + fmt.Sprintf("%s main phase 2. Cast instants or Done to pass.", prefix)
	case core.PhaseEnd:
		return stackMsg + fmt.Sprintf("%s turn - End step", prefix)
	case core.PhaseCleanup:
		return stackMsg + fmt.Sprintf("%s turn - Cleanup", prefix)
	default:
		return stackMsg
	}
}

func (s *DuelScreen) combatStatusMessage(active *core.Player, isMyTurn bool) string {
	step := active.Turn.CombatStep
	switch step {
	case core.CombatStepBeginning:
		if isMyTurn {
			return "Beginning of combat"
		}
		return fmt.Sprintf("%s declares combat", s.opponent.name)
	case core.CombatStepDeclareAttackers:
		if isMyTurn {
			return "Choose creatures to attack with. Done when finished."
		}
		return fmt.Sprintf("%s is choosing attackers...", s.opponent.name)
	case core.CombatStepDeclareBlockers:
		if !isMyTurn {
			return "Choose creatures to block with. Done when finished."
		}
		return fmt.Sprintf("%s is choosing blockers...", s.opponent.name)
	case core.CombatStepFirstStrikeDamage:
		return "First strike damage"
	case core.CombatStepCombatDamage:
		return "Combat damage resolves"
	case core.CombatStepEndOfCombat:
		return "End of combat"
	default:
		if isMyTurn {
			return "Your combat phase"
		}
		return fmt.Sprintf("%s's combat phase", s.opponent.name)
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
