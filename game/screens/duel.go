package screens

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	_ "git.sr.ht/~cdcarter/mage-go/cards"
	mage "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/s30/assets"
	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/logging"
	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	stepDeclareAttackers = "Declare Attackers"
	stepDeclareBlockers  = "Declare Blockers"
	stepCombatDamage     = "Combat Damage"
	playerNameYou        = "You"
)

type duelPlayer struct {
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

	game     *mage.Game
	human    *interactive.HumanPlayer
	aiPlayer *ai.AIPlayer
	lastMsg  *interactive.GameMsg

	self     *duelPlayer
	opponent *duelPlayer

	phaseDefaultBg  *ebiten.Image
	phaseActiveImgs []*ebiten.Image
	bigCardBg       *ebiten.Image
	spellChainBg    *ebiten.Image
	messageBg       *ebiten.Image
	manaPoolBg      *ebiten.Image

	doneBtn      [3]*ebiten.Image
	abilityIcons []*ebiten.Image

	selectedCardIdx        int
	cardPreviewImg         *ebiten.Image
	cardPreviewID          string
	cardPreviewPlaceholder bool
	cardPreviewName        string
	cardPreviewPerm        *interactive.PermanentState

	mouseState     mouseStateType
	mouseStartX    int
	mouseStartY    int
	dragTargetX    *int
	dragTargetY    *int
	dragOffsetX    int
	dragOffsetY    int
	draggingCardID uuid.UUID
	cardPositions  map[uuid.UUID]image.Point

	cardImgCache map[cardImgKey]cardImgEntry

	cardActions      map[uuid.UUID][]interactive.ActionOption
	pendingAttackers map[uuid.UUID]bool
	pendingBlockers  map[uuid.UUID]uuid.UUID
	selectedBlocker  uuid.UUID

	targetingCardID  uuid.UUID
	targetingActions map[uuid.UUID]interactive.ActionOption
	selectedTargetID uuid.UUID

	cardImageMap map[string]*domain.Card

	frameCount  int
	warningMsg  string
	lastMsgTime time.Time

	anteCard      *domain.Card
	enemyAnteCard *domain.Card

	choiceRequest  *interactive.ChoiceRequest
	choiceButtons  []*elements.Button
	choiceCardImg  *ebiten.Image
	choiceCardName string

	prevYouLife        int
	prevOppLife        int
	prevYouGraveLen    int
	prevOppGraveLen    int
	prevYouCreatureLen int
	prevOppCreatureLen int
}

type mouseStateType int

const (
	mouseIdle mouseStateType = iota
	mousePotentialDrag
	mouseDragging
)

type cardImgKey struct {
	name  string
	width int
}

type cardImgEntry struct {
	scaled      *ebiten.Image
	placeholder bool
}

func (s *DuelScreen) IsFramed() bool { return false }

func NewDuelScreen(player *domain.Player, enemy *domain.Enemy, lvl *world.Level, idx int, anteCard *domain.Card, enemyAnteCard *domain.Card) *DuelScreen {
	s := &DuelScreen{
		player:           player,
		enemy:            enemy,
		lvl:              lvl,
		idx:              idx,
		selectedCardIdx:  -1,
		anteCard:         anteCard,
		enemyAnteCard:    enemyAnteCard,
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
	}

	s.initGameState()
	s.loadImages()

	s.self.handX = 860
	s.self.handY = 430
	s.opponent.handX = 860
	s.opponent.handY = 310

	return s
}

func buildCardImageMap(decks ...domain.Deck) map[string]*domain.Card {
	m := make(map[string]*domain.Card)
	for _, deck := range decks {
		for card := range deck {
			m[card.CardName] = card
		}
	}
	return m
}

func (s *DuelScreen) initGameState() {
	s.human = interactive.NewHumanPlayer("You")
	s.human.SetLife(s.player.Life + s.player.BonusDuelLife)
	s.aiPlayer = ai.NewAIPlayer(s.enemy.Name())
	s.aiPlayer.SetLife(s.enemy.Character.Life)

	for card, count := range s.player.GetDuelDeck() {
		for range count {
			c, err := mage.CreateCard(card.CardName)
			if err != nil {
				logging.Printf(logging.Duel, "Failed to create card %s: %v\n", card.CardName, err)
				continue
			}
			s.human.AddToLibrary(c)
		}
	}
	for _, card := range s.player.BonusDuelCards {
		c, err := mage.CreateCard(card.CardName)
		if err != nil {
			logging.Printf(logging.Duel, "Failed to create bonus card %s: %v\n", card.CardName, err)
			continue
		}
		s.human.AddToLibrary(c)
	}
	s.player.BonusDuelLife = 0
	s.player.BonusDuelCards = nil
	for card, count := range s.enemy.Character.GetActiveDeck() {
		for range count {
			c, err := mage.CreateCard(card.CardName)
			if err != nil {
				logging.Printf(logging.Duel, "Failed to create card %s: %v\n", card.CardName, err)
				continue
			}
			s.aiPlayer.AddToLibrary(c)
		}
	}
	s.human.ShuffleLibrary()
	s.aiPlayer.ShuffleLibrary()

	s.game = mage.NewGame(s.human, s.aiPlayer)
	mage.DebugPriority = logging.Enabled(logging.Duel)

	logging.Printf(logging.Duel, "Drawing cards\n")

	for range 7 {
		s.human.DrawCard()
	}
	for range 7 {
		s.aiPlayer.DrawCard()
	}
	logging.Printf(logging.Duel, "Cards drawn\n")

	s.cardImageMap = buildCardImageMap(s.player.GetDuelDeck(), s.enemy.Character.GetActiveDeck())

	s.self = &duelPlayer{name: "You"}
	s.opponent = &duelPlayer{name: s.enemy.Name()}

	logging.Printf(logging.Duel, "Game init: human library=%d hand=%d, ai library=%d hand=%d, human life=%d, ai life=%d\n",
		len(s.human.Library()), len(s.human.Hand()),
		len(s.aiPlayer.Library()), len(s.aiPlayer.Hand()),
		s.human.Life(), s.aiPlayer.Life())
	logging.Printf(logging.Duel, "Game init: IsGameOver=%v Winner=%q\n", s.game.IsGameOver(), s.game.Winner())

	go interactive.RunGameLoop(s.game, 0, 300*time.Millisecond)
}

func (s *DuelScreen) drainMessages() {
	if time.Since(s.lastMsgTime) < 50*time.Millisecond {
		return
	}
	select {
	case msg, ok := <-s.human.ToTUI():
		if !ok {
			return
		}
		s.lastMsg = &msg
		s.lastMsgTime = time.Now()
		s.checkSoundTriggers(&msg)
		if logging.Enabled(logging.Duel) {
			optNames := make([]string, len(msg.Options))
			for i, o := range msg.Options {
				optNames[i] = fmt.Sprintf("%s(%s)", o.Type, o.CardName)
			}
			logging.Printf(logging.Duel, "MSG: turn=%d step=%q active=%q prompt=%v options=%v gameover=%v\n",
				msg.State.Turn, msg.State.Step, msg.State.ActivePlayer, msg.Prompt, optNames, msg.GameOver)
		}
	default:
		return
	}
}

func (s *DuelScreen) drainChoiceRequests() {
	for {
		select {
		case req, ok := <-s.human.ChoiceRequests():
			if !ok {
				return
			}
			s.choiceRequest = &req
			if logging.Enabled(logging.Duel) {
				optLabels := make([]string, len(req.Options))
				for i, o := range req.Options {
					optLabels[i] = o.Label
				}
				logging.Printf(logging.Duel, "CHOICE_REQ: type=%d reason=%q amount=%d options=%v\n",
					req.Type, req.Reason, req.Amount, optLabels)
			}
		default:
			return
		}
	}
}

func (s *DuelScreen) checkSoundTriggers(msg *interactive.GameMsg) {
	am := gameaudio.Get()
	if am == nil {
		return
	}

	youLife := msg.State.You.Life
	oppLife := msg.State.Opponent.Life
	youGraveLen := len(msg.State.You.Graveyard)
	oppGraveLen := len(msg.State.Opponent.Graveyard)
	youCreatureLen := len(msg.State.You.Battlefield)
	oppCreatureLen := len(msg.State.Opponent.Battlefield)

	if s.prevYouLife != 0 || s.prevOppLife != 0 {
		if youLife < s.prevYouLife || oppLife < s.prevOppLife {
			am.PlaySFX(gameaudio.SFXDamage)
		}

		if youGraveLen > s.prevYouGraveLen || oppGraveLen > s.prevOppGraveLen {
			am.PlaySFX(gameaudio.SFXCreatureDeath)
		}

		if youCreatureLen > s.prevYouCreatureLen || oppCreatureLen > s.prevOppCreatureLen {
			am.PlaySFX(gameaudio.SFXSummon)
		}
	}

	s.prevYouLife = youLife
	s.prevOppLife = oppLife
	s.prevYouGraveLen = youGraveLen
	s.prevOppGraveLen = oppGraveLen
	s.prevYouCreatureLen = youCreatureLen
	s.prevOppCreatureLen = oppCreatureLen
}

func (s *DuelScreen) getDomainCard(name string) *domain.Card {
	return s.cardImageMap[name]
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

	abilityData, err := assets.DuelFS.ReadFile("art/sprites/duel/Abilities.pic.png")
	if err == nil {
		sprites, err := imageutil.LoadSpriteSheet(1, 18, abilityData)
		if err == nil {
			s.abilityIcons = make([]*ebiten.Image, 18)
			for i := range 18 {
				s.abilityIcons[i] = sprites[i][0]
			}
		}
	}
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

func (s *DuelScreen) getKeywordIcons(perm interactive.PermanentState) []*ebiten.Image {
	if len(s.abilityIcons) == 0 {
		return nil
	}
	var icons []*ebiten.Image

	keywordIndex := map[string]int{
		"Flying":       11,
		"Trample":      12,
		"Banding":      13,
		"First Strike": 14,
		"Regeneration": 15,
		"Reach":        16,
		"Menace":       17,
	}
	protectionColors := map[string]int{
		"green": 5, "red": 6, "blue": 7, "black": 8, "white": 9, "artifacts": 10,
	}
	seen := map[int]bool{}
	for _, kw := range perm.Keywords {
		if idx, ok := keywordIndex[kw]; ok && !seen[idx] {
			seen[idx] = true
			icons = append(icons, s.abilityIcons[idx])
		}
	}

	if p := s.game.FindPermanent(perm.ID); p != nil {
		for _, ability := range p.Card.Abilities() {
			if pa, ok := ability.(*mage.ProtectionAbility); ok {
				for _, c := range pa.FromColors {
					if idx, ok := protectionColors[strings.ToLower(c.String())]; ok && !seen[idx] {
						seen[idx] = true
						icons = append(icons, s.abilityIcons[idx])
					}
				}
			}
		}
	}

	return icons
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
	s.drainMessages()
	s.drainChoiceRequests()

	if s.lastMsg == nil {
		return screenui.DuelScr, nil, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && s.targetingCardID == uuid.Nil {
		s.submitPendingAndPass()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if s.targetingCardID != uuid.Nil {
			if s.selectedTargetID != uuid.Nil {
				s.selectedTargetID = uuid.Nil
			} else {
				s.exitTargetingMode()
			}
		} else {
			return screenui.WorldScr, nil, nil
		}
	}

	s.frameCount++

	if s.choiceRequest != nil {
		s.handleChoiceRequest()
		return screenui.DuelScr, nil, nil
	}

	s.refreshCardActions()

	if s.targetingCardID != uuid.Nil {
		if _, ok := s.cardActions[s.targetingCardID]; !ok {
			s.exitTargetingMode()
		}
	}

	mx, my := ebiten.CursorPosition()

	if s.targetingCardID != uuid.Nil {
		s.updateTargetingMouse(mx, my)
	} else {
		s.updateMouse(mx, my)
		s.updateHoverPreview(mx, my)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		s.handleRightClick(mx, my)
	}

	if s.lastMsg.GameOver {
		logging.Printf(logging.Duel, "GameOver! Winner=%q Step=%q Turn=%d YouLife=%d OppLife=%d YouLib=%d OppLib=%d\n",
			s.lastMsg.Winner, s.lastMsg.State.Step, s.lastMsg.State.Turn,
			s.lastMsg.State.You.Life, s.lastMsg.State.Opponent.Life,
			s.lastMsg.State.You.LibraryCount, s.lastMsg.State.Opponent.LibraryCount)
		if s.lastMsg.Winner == playerNameYou {
			return s.handleWin()
		}
		return s.handleLoss()
	}

	return screenui.DuelScr, nil, nil
}

const (
	handCardOverlap = 20
	fieldCardW      = 100
	fieldCardH      = 83
)

func (s *DuelScreen) getFieldCardPos(perm interactive.PermanentState, dp *duelPlayer, idx int, total int, isLand bool) image.Point {
	if perm.ID == s.draggingCardID {
		if pos, ok := s.cardPositions[perm.ID]; ok {
			return pos
		}
	}
	var baseY int
	if dp == s.opponent {
		if isLand {
			baseY = duelOpponentBoardY + 70
		} else {
			baseY = duelOpponentBoardY + 70 + fieldCardH + 10
		}
	} else {
		if isLand {
			baseY = 600
		} else {
			baseY = duelPlayerBoardY + 20
		}
	}
	maxSpacing := 35
	if !isLand {
		maxSpacing = 120
	}
	availableW := duelBoardW - 30 - fieldCardW
	spacing := maxSpacing
	if total > 1 && (total-1)*spacing > availableW {
		spacing = availableW / (total - 1)
	}
	pos := image.Pt(duelBoardX+30+idx*spacing, baseY)
	s.cardPositions[perm.ID] = pos
	return pos
}

func (s *DuelScreen) fieldPerms(ps interactive.PlayerState, landsOnly bool) []interactive.PermanentState {
	var perms []interactive.PermanentState
	for _, perm := range ps.Battlefield {
		if perm.AttachedTo != uuid.Nil {
			continue
		}
		isLand := perm.IsLand
		if isLand == landsOnly {
			perms = append(perms, perm)
		}
	}
	return perms
}

func (s *DuelScreen) attachedPerms(hostID uuid.UUID) []interactive.PermanentState {
	if s.lastMsg == nil || s.lastMsg.State == nil {
		return nil
	}
	var perms []interactive.PermanentState
	for _, ps := range []interactive.PlayerState{s.lastMsg.State.You, s.lastMsg.State.Opponent} {
		for _, perm := range ps.Battlefield {
			if perm.AttachedTo == hostID {
				perms = append(perms, perm)
			}
		}
	}
	return perms
}

func (s *DuelScreen) fieldPermAtPoint(mx, my int, dp *duelPlayer) *interactive.PermanentState {
	ps := s.playerState(dp)
	if ps == nil {
		return nil
	}
	for _, landsOnly := range []bool{false, true} {
		perms := s.fieldPerms(*ps, landsOnly)
		for i := len(perms) - 1; i >= 0; i-- {
			perm := perms[i]
			pos := s.getFieldCardPos(perm, dp, i, len(perms), landsOnly)
			if mx >= pos.X && mx < pos.X+fieldCardW {
				auras := s.attachedPerms(perm.ID)
				for j := len(auras) - 1; j >= 0; j-- {
					auraY := pos.Y - (j+1)*14
					if my >= auraY && my < auraY+14 {
						auraCopy := auras[j]
						return &auraCopy
					}
				}
				if my >= pos.Y && my < pos.Y+fieldCardH {
					return &perms[i]
				}
			}
		}
	}
	return nil
}

func (s *DuelScreen) playerState(dp *duelPlayer) *interactive.PlayerState {
	if s.lastMsg == nil || s.lastMsg.State == nil {
		return nil
	}
	if dp == s.self {
		return &s.lastMsg.State.You
	}
	return &s.lastMsg.State.Opponent
}

func (s *DuelScreen) refreshCardActions() {
	if s.lastMsg == nil {
		return
	}

	step := s.lastMsg.State.Step
	inDeclareAttackers := s.lastMsg.State.ActivePlayer == playerNameYou &&
		step == stepDeclareAttackers
	if !inDeclareAttackers && len(s.pendingAttackers) > 0 {
		s.pendingAttackers = make(map[uuid.UUID]bool)
	}
	if !s.isInDeclareBlockers() && (len(s.pendingBlockers) > 0 || s.selectedBlocker != uuid.Nil) {
		s.pendingBlockers = make(map[uuid.UUID]uuid.UUID)
		s.selectedBlocker = uuid.Nil
	}
	s.cardActions = make(map[uuid.UUID][]interactive.ActionOption)
	for _, opt := range s.lastMsg.Options {
		id := opt.CardID
		if id == uuid.Nil {
			id = opt.PermanentID
		}
		if id == uuid.Nil {
			continue
		}
		s.cardActions[id] = append(s.cardActions[id], opt)
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

func (s *DuelScreen) handCardIdxAtPoint(mx, my, panelX, panelY int, count int, dp *duelPlayer) int {
	headerH := s.panelCardH(dp)
	w := s.panelCardW(dp)
	cardListY := panelY + headerH
	if my < cardListY || mx < panelX || mx >= panelX+w {
		return -1
	}
	idx := (my - cardListY) / handCardOverlap
	if idx >= count {
		idx = count - 1
	}
	if idx < 0 {
		return -1
	}
	return idx
}

func (s *DuelScreen) isInDeclareBlockers() bool {
	if s.lastMsg == nil || s.lastMsg.State == nil {
		return false
	}
	return s.lastMsg.Prompt == interactive.PromptDeclareBlockers
}

func (s *DuelScreen) isValidBlock(blockerID, attackerID uuid.UUID) bool {
	if s.lastMsg == nil {
		return false
	}
	for _, opt := range s.lastMsg.Options {
		if opt.Type != interactive.ActionSelectBlockers || opt.PermanentID != blockerID {
			continue
		}
		for _, target := range opt.ValidTargets {
			if target == attackerID {
				return true
			}
		}
	}
	return false
}

func (s *DuelScreen) canBlockAnything(blockerID uuid.UUID) bool {
	if s.lastMsg == nil {
		return false
	}
	for _, opt := range s.lastMsg.Options {
		if opt.Type == interactive.ActionSelectBlockers && opt.PermanentID == blockerID && len(opt.ValidTargets) > 0 {
			return true
		}
	}
	return false
}

// canBeBlocked returns true if any eligible blocker can legally block this attacker.
func (s *DuelScreen) canBeBlocked(attackerID uuid.UUID) bool {
	if s.lastMsg == nil {
		return false
	}
	for _, opt := range s.lastMsg.Options {
		if opt.Type != interactive.ActionSelectBlockers {
			continue
		}
		for _, target := range opt.ValidTargets {
			if target == attackerID {
				return true
			}
		}
	}
	return false
}

func (s *DuelScreen) handleBlockerClick(mx, my int) {
	s.warningMsg = ""
	if perm := s.fieldPermAtPoint(mx, my, s.self); perm != nil {
		s.loadCardPreview(perm.Name, perm)
		if _, assigned := s.pendingBlockers[perm.ID]; assigned {
			delete(s.pendingBlockers, perm.ID)
			return
		}
		if s.selectedBlocker == perm.ID {
			s.selectedBlocker = uuid.Nil
			return
		}
		if s.canBlockAnything(perm.ID) {
			s.selectedBlocker = perm.ID
		}
		return
	}

	if perm := s.fieldPermAtPoint(mx, my, s.opponent); perm != nil {
		s.loadCardPreview(perm.Name, perm)
		if s.selectedBlocker != uuid.Nil && perm.Attacking && s.isValidBlock(s.selectedBlocker, perm.ID) {
			s.pendingBlockers[s.selectedBlocker] = perm.ID
			s.selectedBlocker = uuid.Nil
		}
		return
	}
}

func (s *DuelScreen) handleCardClick(mx, my int) {
	if s.lastMsg == nil {
		return
	}
	hand := s.lastMsg.State.You.Hand
	if idx := s.handCardIdxAtPoint(mx, my, s.self.handX, s.self.handY, len(hand), s.self); idx >= 0 {
		logging.Printf(logging.Duel, "HAND CLICK: idx: %d\n", idx)
		s.selectedCardIdx = idx
		card := hand[idx]
		s.performCardAction(card.ID, card.Name)
		return
	}

	if s.isInDeclareBlockers() {
		s.handleBlockerClick(mx, my)
		return
	}

	if perm := s.fieldPermAtPoint(mx, my, s.self); perm != nil {
		s.loadCardPreview(perm.Name, perm)
		if actions, ok := s.cardActions[perm.ID]; ok && hasActionType(actions, interactive.ActionSelectAttackers) {
			if s.pendingAttackers[perm.ID] {
				delete(s.pendingAttackers, perm.ID)
			} else {
				s.pendingAttackers[perm.ID] = true
			}
			return
		}
		s.performCardAction(perm.ID, perm.Name)
		return
	}
}

func (s *DuelScreen) handleRightClick(mx, my int) {
	if s.lastMsg == nil {
		return
	}
	hand := s.lastMsg.State.You.Hand
	if idx := s.handCardIdxAtPoint(mx, my, s.self.handX, s.self.handY, len(hand), s.self); idx >= 0 {
		s.loadCardPreviewByName(hand[idx].Name)
		return
	}

	if perm := s.fieldPermAtPoint(mx, my, s.self); perm != nil {
		s.loadCardPreview(perm.Name, perm)
		return
	}

	if perm := s.fieldPermAtPoint(mx, my, s.opponent); perm != nil {
		s.loadCardPreview(perm.Name, perm)
		return
	}
}

func (s *DuelScreen) updateHoverPreview(mx, my int) {
	if s.lastMsg == nil {
		return
	}
	hand := s.lastMsg.State.You.Hand
	if idx := s.handCardIdxAtPoint(mx, my, s.self.handX, s.self.handY, len(hand), s.self); idx >= 0 {
		s.loadCardPreviewByName(hand[idx].Name)
		return
	}

	if perm := s.fieldPermAtPoint(mx, my, s.self); perm != nil {
		s.loadCardPreview(perm.Name, perm)
		return
	}

	if perm := s.fieldPermAtPoint(mx, my, s.opponent); perm != nil {
		s.loadCardPreview(perm.Name, perm)
		return
	}

	if len(s.lastMsg.State.StackItems) > 0 {
		item := s.lastMsg.State.StackItems[len(s.lastMsg.State.StackItems)-1]
		s.loadCardPreviewByName(item.Name)
	}
}

func (s *DuelScreen) performCardAction(id uuid.UUID, name string) {
	actions, ok := s.cardActions[id]
	if !ok || len(actions) == 0 {
		logging.Printf(logging.Duel, "CLICK: %s (no action available)\n", name)
		return
	}

	if len(actions) > 1 || actions[0].NeedsTarget {
		s.enterTargetingMode(id, name, actions)
		return
	}

	action := actions[0]
	logging.Printf(logging.Duel, "CLICK: %s -> action=%v\n", name, action.Type)
	pa := actionOptionToPriorityAction(action)
	select {
	case s.human.FromTUI() <- pa:
		if am := gameaudio.Get(); am != nil {
			am.PlaySFX(gameaudio.SFXCast)
		}
	default:
	}
}

func actionOptionToPriorityAction(opt interactive.ActionOption) interactive.PriorityAction {
	return interactive.PriorityAction{
		Type:         opt.Type,
		CardID:       opt.CardID,
		CardName:     opt.CardName,
		PermanentID:  opt.PermanentID,
		AbilityIndex: opt.AbilityIndex,
	}
}

func (s *DuelScreen) enterTargetingMode(id uuid.UUID, name string, actions []interactive.ActionOption) {
	s.targetingCardID = id
	s.targetingActions = make(map[uuid.UUID]interactive.ActionOption)

	for _, a := range actions {
		if !a.NeedsTarget {
			continue
		}
		for _, tid := range a.ValidTargets {
			s.targetingActions[tid] = a
		}
	}
	s.selectedTargetID = uuid.Nil
	s.loadCardPreviewByName(name)
}

func (s *DuelScreen) exitTargetingMode() {
	s.targetingCardID = uuid.Nil
	s.targetingActions = nil
	s.selectedTargetID = uuid.Nil
}

func (s *DuelScreen) updateTargetingMouse(mx, my int) {
	if s.mouseState != mouseIdle {
		s.dragTargetX = nil
		s.dragTargetY = nil
		s.draggingCardID = uuid.Nil
		s.mouseState = mouseIdle
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	doneBounds := s.doneBtn[0].Bounds()
	doneX := duelBoardX + 2
	doneY := duelMsgY
	if mx >= doneX && mx < doneX+doneBounds.Dx() && my >= doneY && my < doneY+doneBounds.Dy() {
		if s.selectedTargetID != uuid.Nil {
			if action, ok := s.targetingActions[s.selectedTargetID]; ok {
				pa := actionOptionToPriorityAction(action)
				pa.Targets = []uuid.UUID{s.selectedTargetID}
				select {
				case s.human.FromTUI() <- pa:
					if am := gameaudio.Get(); am != nil {
						am.PlaySFX(gameaudio.SFXCast)
					}
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
	if s.selectedTargetID != uuid.Nil && mx >= msgBarLeft && my >= msgBarTop && my < msgBarTop+20 {
		s.selectedTargetID = uuid.Nil
		return
	}

	s.handleTargetClick(mx, my)
}

func (s *DuelScreen) handleTargetClick(mx, my int) {
	for _, dp := range []*duelPlayer{s.opponent, s.self} {
		if perm := s.fieldPermAtPoint(mx, my, dp); perm != nil {
			if _, ok := s.targetingActions[perm.ID]; ok {
				s.selectedTargetID = perm.ID
				s.loadCardPreview(perm.Name, perm)
				return
			}
		}
	}

	for _, dp := range []*duelPlayer{s.opponent, s.self} {
		ps := s.playerState(dp)
		if ps != nil && s.isPlayerBoardClick(mx, my, dp) {
			if _, ok := s.targetingActions[ps.ID]; ok {
				s.selectedTargetID = ps.ID
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

func (s *DuelScreen) getCardArtImg(name string, targetW int) *ebiten.Image {
	key := cardImgKey{name: name, width: targetW}
	if entry, ok := s.cardImgCache[key]; ok {
		if !entry.placeholder {
			return entry.scaled
		}
	}

	domainCard := s.getDomainCard(name)
	if domainCard == nil {
		return nil
	}

	artImg, err := domainCard.CardImage(domain.CardViewArtOnly)
	if err != nil || artImg == nil {
		return nil
	}

	loaded := domainCard.ImageLoaded()
	scale := float64(targetW) / float64(artImg.Bounds().Dx())
	scaled := imageutil.ScaleImage(artImg, scale)
	s.cardImgCache[key] = cardImgEntry{scaled: scaled, placeholder: !loaded}
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

		for _, dp := range []*duelPlayer{s.self, s.opponent} {
			if perm := s.fieldPermAtPoint(mx, my, dp); perm != nil {
				s.draggingCardID = perm.ID
				s.mouseState = mousePotentialDrag
				return
			}
		}

		s.handleClick(mx, my)

	case mousePotentialDrag:
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			s.handleClick(s.mouseStartX, s.mouseStartY)
			s.draggingCardID = uuid.Nil
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
			s.draggingCardID = uuid.Nil
			s.mouseState = mouseIdle
			return
		}
		if s.draggingCardID != uuid.Nil {
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

func (s *DuelScreen) hasPendingMenaceViolation() bool {
	if s.lastMsg == nil {
		return false
	}
	blockerCount := map[uuid.UUID]int{}
	for _, attackerID := range s.pendingBlockers {
		blockerCount[attackerID]++
	}
	for attackerID, count := range blockerCount {
		if count >= 2 {
			continue
		}
		for _, perm := range s.lastMsg.State.Opponent.Battlefield {
			if perm.ID == attackerID && perm.Attacking && hasKeyword(perm.Keywords, "Menace") {
				return true
			}
		}
	}
	return false
}

func (s *DuelScreen) removePendingMenaceViolations() {
	if s.lastMsg == nil {
		return
	}
	blockerCount := map[uuid.UUID]int{}
	for _, attackerID := range s.pendingBlockers {
		blockerCount[attackerID]++
	}
	menaceAttackers := map[uuid.UUID]bool{}
	for attackerID, count := range blockerCount {
		if count >= 2 {
			continue
		}
		for _, perm := range s.lastMsg.State.Opponent.Battlefield {
			if perm.ID == attackerID && perm.Attacking && hasKeyword(perm.Keywords, "Menace") {
				menaceAttackers[attackerID] = true
			}
		}
	}
	for blockerID, attackerID := range s.pendingBlockers {
		if menaceAttackers[attackerID] {
			delete(s.pendingBlockers, blockerID)
		}
	}
}

func hasKeyword(keywords []string, kw string) bool {
	for _, k := range keywords {
		if k == kw {
			return true
		}
	}
	return false
}

func (s *DuelScreen) submitPendingAndPass() {
	if s.lastMsg != nil && s.lastMsg.Prompt == interactive.PromptDeclareAttackers {
		var attackerIDs []uuid.UUID
		for id := range s.pendingAttackers {
			attackerIDs = append(attackerIDs, id)
		}
		s.pendingAttackers = make(map[uuid.UUID]bool)
		select {
		case s.human.FromTUI() <- interactive.PriorityAction{
			Type:      interactive.ActionSelectAttackers,
			Attackers: attackerIDs,
		}:
		default:
		}
		return
	}

	if s.lastMsg != nil && s.lastMsg.Prompt == interactive.PromptDeclareBlockers {
		var blockers []mage.BlockAssignment
		for blockerID, attackerID := range s.pendingBlockers {
			blockers = append(blockers, mage.BlockAssignment{
				BlockerID:  blockerID,
				AttackerID: attackerID,
			})
		}
		s.pendingBlockers = make(map[uuid.UUID]uuid.UUID)
		s.selectedBlocker = uuid.Nil
		select {
		case s.human.FromTUI() <- interactive.PriorityAction{
			Type:     interactive.ActionSelectBlockers,
			Blockers: blockers,
		}:
		default:
		}
		return
	}

	select {
	case s.human.FromTUI() <- interactive.PriorityAction{Type: interactive.ActionPass}:
	default:
	}
}

func (s *DuelScreen) handleClick(mx, my int) {
	doneBounds := s.doneBtn[0].Bounds()
	doneX := duelBoardX + 2
	doneY := duelMsgY
	if mx >= doneX && mx < doneX+doneBounds.Dx() && my >= doneY && my < doneY+doneBounds.Dy() {
		if s.isInDeclareBlockers() && s.hasPendingMenaceViolation() {
			s.warningMsg = "Menace: must be blocked by 2 or more creatures!"
			s.removePendingMenaceViolations()
			return
		}
		s.warningMsg = ""
		s.submitPendingAndPass()
		return
	}

	s.handleCardClick(mx, my)
}

func (s *DuelScreen) handleChoiceRequest() {
	if s.choiceRequest == nil {
		return
	}

	if s.choiceButtons == nil {
		s.initChoiceUI()
	}

	s.updateChoiceUI()
}

func (s *DuelScreen) initChoiceUI() {
	req := s.choiceRequest

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		logging.Printf(logging.Duel, "Error loading choice button sprites: %v\n", err)
		return
	}

	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 16}
	s.choiceButtons = make([]*elements.Button, len(req.Options))
	for i, opt := range req.Options {
		label := fmt.Sprintf("%d. %s", i+1, opt.Label)
		btn := elements.NewButton(btnSprites[0][0], btnSprites[0][1], btnSprites[0][2], 0, 0, 1.0)
		btn.ButtonText = elements.ButtonText{
			Text:      label,
			Font:      fontFace,
			TextColor: color.White,
			HAlign:    elements.AlignCenter,
			VAlign:    elements.AlignMiddle,
		}
		s.choiceButtons[i] = btn
	}

	s.choiceCardName = req.Reason
	domainCard := s.getDomainCard(req.Reason)
	if domainCard != nil {
		s.choiceCardImg, _ = domainCard.CardImage(domain.CardViewFull)
	} else {
		s.choiceCardImg = nil
	}
}

func (s *DuelScreen) updateChoiceUI() {
	req := s.choiceRequest

	for i := range req.Options {
		key := ebiten.Key1 + ebiten.Key(i)
		if i < 9 && inpututil.IsKeyJustPressed(key) {
			s.respondToChoice(i)
			return
		}
	}

	btnW := 0
	if len(s.choiceButtons) > 0 {
		btnW = s.choiceButtons[0].Normal.Bounds().Dx()
	}
	cardH := 0
	if s.choiceCardImg != nil {
		cardH = s.choiceCardImg.Bounds().Dy()
	}

	centerX := 512
	titleH := 30
	cardTopY := 768/2 - (cardH+titleH+len(s.choiceButtons)*40)/2
	btnStartY := cardTopY + titleH + cardH + 10

	for i, btn := range s.choiceButtons {
		btnX := centerX - btnW/2
		btnY := btnStartY + i*40
		btn.MoveTo(btnX, btnY)
		opts := &ebiten.DrawImageOptions{}
		btn.Update(opts, 1.0, 1024, 768)
		if btn.IsClicked() {
			s.respondToChoice(i)
			return
		}
	}
}

func (s *DuelScreen) respondToChoice(index int) {
	req := s.choiceRequest

	switch req.Type {
	case interactive.ChoiceMay:
		accepted := index == 0
		logging.Printf(logging.Duel, "CHOICE_RESP: type=May accepted=%v reason=%q\n", accepted, req.Reason)
		s.human.ChoiceResponses() <- interactive.ChoiceResponse{Accepted: accepted}
	case interactive.ChoiceManaColor:
		if index < len(req.Options) {
			logging.Printf(logging.Duel, "CHOICE_RESP: type=ManaColor color=%v reason=%q\n", req.Options[index].Color, req.Reason)
			s.human.ChoiceResponses() <- interactive.ChoiceResponse{SelectedColor: req.Options[index].Color}
		}
	case interactive.ChoiceMode:
		logging.Printf(logging.Duel, "CHOICE_RESP: type=Mode index=%d reason=%q\n", index, req.Reason)
		s.human.ChoiceResponses() <- interactive.ChoiceResponse{SelectedIndex: index}
	case interactive.ChoicePermanent:
		if index < len(req.Options) {
			logging.Printf(logging.Duel, "CHOICE_RESP: type=Permanent selected=%q reason=%q\n", req.Options[index].Label, req.Reason)
			s.human.ChoiceResponses() <- interactive.ChoiceResponse{
				SelectedIDs: []uuid.UUID{req.Options[index].ID},
			}
		}
	case interactive.ChoiceCardsFromHand:
		if index < len(req.Options) {
			logging.Printf(logging.Duel, "CHOICE_RESP: type=CardsFromHand selected=%q reason=%q\n", req.Options[index].Label, req.Reason)
			s.human.ChoiceResponses() <- interactive.ChoiceResponse{
				SelectedIDs: []uuid.UUID{req.Options[index].ID},
			}
		}
	default:
		if index < len(req.Options) {
			logging.Printf(logging.Duel, "CHOICE_RESP: type=%d selected=%q reason=%q\n", req.Type, req.Options[index].Label, req.Reason)
			s.human.ChoiceResponses() <- interactive.ChoiceResponse{
				SelectedIDs: []uuid.UUID{req.Options[index].ID},
			}
		}
	}

	s.choiceRequest = nil
	s.choiceButtons = nil
	s.choiceCardImg = nil
	s.choiceCardName = ""
}

func (s *DuelScreen) drawChoiceUI(screen *ebiten.Image, W, H int) {
	if s.choiceRequest == nil || s.choiceButtons == nil {
		return
	}

	vector.DrawFilledRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 160}, false)

	centerX := float64(W) / 2

	cardH := 0
	if s.choiceCardImg != nil {
		cardH = s.choiceCardImg.Bounds().Dy()
	}
	titleH := 30
	totalH := titleH + cardH + 10 + len(s.choiceButtons)*40
	startY := float64(H)/2 - float64(totalH)/2

	reasonText := s.choiceCardName
	if reasonText == "" {
		reasonText = "Choose"
	}
	title := elements.NewText(24, reasonText, 0, int(startY))
	title.HAlign = elements.AlignCenter
	title.BoundsW = float64(W)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if s.choiceCardImg != nil {
		cardW := float64(s.choiceCardImg.Bounds().Dx())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(centerX-cardW/2, startY+float64(titleH))
		screen.DrawImage(s.choiceCardImg, op)
	}

	btnOpts := &ebiten.DrawImageOptions{}
	for _, btn := range s.choiceButtons {
		btn.Draw(screen, btnOpts, 1.0)
	}
}

func (s *DuelScreen) loadCardPreviewByName(name string) {
	s.loadCardPreview(name, nil)
}

func (s *DuelScreen) loadCardPreview(name string, perm *interactive.PermanentState) {
	if s.cardPreviewID == name {
		s.cardPreviewPerm = perm
		return
	}
	domainCard := s.getDomainCard(name)
	if domainCard == nil {
		s.cardPreviewImg = nil
		s.cardPreviewID = ""
		s.cardPreviewName = ""
		s.cardPreviewPerm = nil
		return
	}
	img, err := domainCard.CardImage(domain.CardViewFull)
	if err != nil || img == nil {
		s.cardPreviewImg = nil
		s.cardPreviewID = ""
		s.cardPreviewName = ""
		s.cardPreviewPerm = nil
		return
	}
	s.cardPreviewImg = img
	s.cardPreviewID = name
	s.cardPreviewPlaceholder = !domainCard.ImageLoaded()
	s.cardPreviewName = name
	s.cardPreviewPerm = perm
}

func (s *DuelScreen) handleWin() (screenui.ScreenName, screenui.Screen, error) {
	if s.player.ActiveQuest != nil &&
		s.player.ActiveQuest.Type == domain.QuestTypeDefeatEnemy &&
		s.player.ActiveQuest.EnemyName == s.enemy.Character.Name {
		s.player.ActiveQuest.IsCompleted = true
	}

	logging.Printf(logging.Duel, "you just beat: %s\n", s.enemy.Name())

	wonCards := []*domain.Card{}
	if s.enemyAnteCard != nil {
		s.player.CardCollection.AddCard(s.enemyAnteCard, 1)
		wonCards = append(wonCards, s.enemyAnteCard)
	}

	s.lvl.RemoveEnemyAt(s.idx)

	return screenui.DuelWinScr, NewWinDuelScreen(wonCards), nil
}

func (s *DuelScreen) handleLoss() (screenui.ScreenName, screenui.Screen, error) {
	if s.anteCard != nil {
		_ = s.player.RemoveCard(s.anteCard)
	}
	lostCards := []*domain.Card{}
	if s.anteCard != nil {
		lostCards = append(lostCards, s.anteCard)
	}

	s.lvl.RemoveEnemyAt(s.idx)

	return screenui.DuelLoseScr, NewDuelLoseScreen(lostCards), nil
}

func (s *DuelScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.Fill(color.RGBA{30, 30, 30, 255})

	if s.lastMsg == nil {
		return
	}

	s.drawPhasePanel(screen)
	s.drawBoard(screen, s.opponent, &s.lastMsg.State.Opponent, duelOpponentBoardY, duelMsgY)
	s.drawBoard(screen, s.self, &s.lastMsg.State.You, duelPlayerBoardY, H-duelPlayerBoardY)
	s.drawMessageBar(screen)
	s.drawSidebar(screen, W, H)
	s.drawBattlefield(screen, s.opponent, s.lastMsg.State.Opponent)
	s.drawBattlefield(screen, s.self, s.lastMsg.State.You)
	s.drawBlockerArrows(screen)
	s.drawHandPanel(screen, s.opponent, s.lastMsg.State.Opponent)
	s.drawHandPanel(screen, s.self, s.lastMsg.State.You)
	s.drawCardPreview(screen, H)
	s.drawChoiceUI(screen, W, H)
}

func (s *DuelScreen) drawPhasePanel(screen *ebiten.Image) {
	if s.phaseDefaultBg == nil {
		return
	}

	const phaseX = 250

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(phaseX, 4)
	screen.DrawImage(s.phaseDefaultBg, opts)

	step := s.lastMsg.State.Step
	idx := phaseIndex(step)
	var row int
	if s.lastMsg.State.ActivePlayer == playerNameYou {
		row = 10 + idx
	} else {
		row = idx
	}
	if row < len(s.phaseActiveImgs) && s.phaseActiveImgs[row] != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(phaseX, float64(4+row*42))
		screen.DrawImage(s.phaseActiveImgs[row], opts)
	}
}

func phaseIndex(step string) int {
	phaseMap := map[string]int{
		"Untap":               0,
		"Upkeep":              1,
		"Draw":                2,
		"Precombat Main":      3,
		"Begin Combat":        4,
		stepDeclareAttackers:  4,
		stepDeclareBlockers:   4,
		"First Strike Damage": 4,
		stepCombatDamage:      4,
		"End of Combat":       4,
		"Postcombat Main":     5,
		"End Step":            6,
		"Cleanup":             6,
	}
	if idx, ok := phaseMap[step]; ok {
		return idx
	}
	return 0
}

func (s *DuelScreen) drawBoard(screen *ebiten.Image, dp *duelPlayer, ps *interactive.PlayerState, startY, boardH int) {
	if dp.boardBg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(duelBoardX), float64(startY))
		screen.DrawImage(dp.boardBg, opts)
	}

	lifeText := fmt.Sprintf("Life: %d", ps.Life)
	txt := elements.NewText(16, dp.name, duelBoardX+10, startY+10)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	txt = elements.NewText(14, lifeText, duelBoardX+10, startY+32)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	infoText := fmt.Sprintf("Hand: %d  Library: %d  Graveyard: %d",
		ps.HandCount, ps.LibraryCount, ps.GraveyardCount)
	txt = elements.NewText(12, infoText, duelBoardX+10, startY+52)
	txt.Color = color.RGBA{200, 200, 200, 255}
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if dp == s.self && s.lastMsg != nil {
		y := startY + 68
		for _, opt := range s.lastMsg.Options {
			line := fmt.Sprintf("%v %s", opt.Type, opt.Label)
			at := elements.NewText(14, line, duelBoardX+10, y)
			at.Color = color.RGBA{180, 180, 180, 255}
			at.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
			y += 12
		}
	}

	if s.targetingCardID != uuid.Nil {
		if _, isTarget := s.targetingActions[ps.ID]; isTarget {
			borderColor := color.RGBA{255, 255, 0, 255}
			strokeW := float32(2)
			if s.selectedTargetID == ps.ID {
				borderColor = color.RGBA{0, 255, 0, 255}
				strokeW = 3
			}
			vector.StrokeRect(screen, float32(duelBoardX), float32(startY),
				float32(duelBoardW), float32(boardH), strokeW, borderColor, false)
		}
	}
}

func (s *DuelScreen) humanHasPriority() bool {
	return s.lastMsg != nil && len(s.lastMsg.Options) > 0
}

func (s *DuelScreen) drawMessageBar(screen *ebiten.Image) {
	if s.doneBtn[0] != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(duelBoardX+2), float64(duelMsgY))
		screen.DrawImage(s.doneBtn[0], opts)

		if s.humanHasPriority() {
			bounds := s.doneBtn[0].Bounds()
			vector.StrokeRect(screen,
				float32(duelBoardX+2), float32(duelMsgY),
				float32(bounds.Dx()), float32(bounds.Dy()),
				2, color.RGBA{255, 255, 0, 255}, false)
		}
	}

	if s.messageBg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(duelBoardX+52), float64(duelMsgY+6))
		screen.DrawImage(s.messageBg, opts)
	}

	var msg string
	var msgColor color.RGBA
	if s.warningMsg != "" {
		msg = s.warningMsg
		msgColor = color.RGBA{255, 100, 100, 255}
	} else if s.targetingCardID != uuid.Nil && s.selectedTargetID != uuid.Nil {
		targetName := s.targetNameByID(s.selectedTargetID)
		msg = fmt.Sprintf("targeting %s (Cancel)", targetName)
		msgColor = color.RGBA{255, 255, 255, 255}
	} else if s.targetingCardID != uuid.Nil {
		cardName := s.cardNameByID(s.targetingCardID)
		msg = fmt.Sprintf("Choose a target for %s", cardName)
		msgColor = color.RGBA{255, 255, 255, 255}
	} else {
		msg = s.statusMessage()
		msgColor = color.RGBA{255, 255, 255, 255}
	}
	txt := elements.NewText(14, msg, duelBoardX+60, duelMsgY+12)
	txt.Color = msgColor
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func (s *DuelScreen) cardNameByID(id uuid.UUID) string {
	if s.lastMsg == nil {
		return ""
	}
	for _, c := range s.lastMsg.State.You.Hand {
		if c.ID == id {
			return c.Name
		}
	}
	for _, p := range s.lastMsg.State.You.Battlefield {
		if p.ID == id {
			return p.Name
		}
	}
	for _, p := range s.lastMsg.State.Opponent.Battlefield {
		if p.ID == id {
			return p.Name
		}
	}
	return ""
}

func (s *DuelScreen) targetNameByID(id uuid.UUID) string {
	name := s.cardNameByID(id)
	if name != "" {
		return name
	}
	if s.lastMsg != nil {
		if s.lastMsg.State.You.ID == id {
			return "You"
		}
		if s.lastMsg.State.Opponent.ID == id {
			return s.opponent.name
		}
	}
	return ""
}

func (s *DuelScreen) drawBattlefield(screen *ebiten.Image, dp *duelPlayer, ps interactive.PlayerState) {
	if s.frameCount%120 == 0 {
		logging.Printf(logging.Duel, "drawBattlefield %s: %d permanents on battlefield\n", dp.name, len(ps.Battlefield))
		for _, p := range ps.Battlefield {
			logging.Printf(logging.Duel, "  perm: %s land=%v creature=%v tapped=%v\n", p.Name, p.IsLand, p.IsCreature, p.Tapped)
		}
	}
	for _, landsOnly := range []bool{true, false} {
		perms := s.fieldPerms(ps, landsOnly)
		for i, perm := range perms {
			pos := s.getFieldCardPos(perm, dp, i, len(perms), landsOnly)

			auras := s.attachedPerms(perm.ID)
			// reverse order so it draws correctly on the screen
			for j := len(auras) - 1; j >= 0; j-- {
				aura := auras[j]
				auraY := pos.Y - (j+1)*14
				auraImg := s.getCardArtImg(aura.Name, fieldCardW)
				if auraImg != nil {
					auraOpts := &ebiten.DrawImageOptions{}
					auraOpts.GeoM.Translate(float64(pos.X), float64(auraY))
					screen.DrawImage(auraImg, auraOpts)
				} else {
					vector.DrawFilledRect(screen, float32(pos.X), float32(auraY),
						float32(fieldCardW), 14, color.RGBA{80, 60, 80, 255}, false)
					auraTxt := elements.NewText(10, aura.Name, pos.X+4, auraY+2)
					auraTxt.Color = color.RGBA{220, 200, 255, 255}
					auraTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
				}
			}

			cardImg := s.getCardArtImg(perm.Name, fieldCardW)
			if cardImg != nil {
				cardOpts := &ebiten.DrawImageOptions{}
				if perm.Tapped {
					imgH := float64(cardImg.Bounds().Dy())
					cardOpts.GeoM.Rotate(math.Pi / 2)
					cardOpts.GeoM.Translate(imgH, 0)
				}
				cardOpts.GeoM.Translate(float64(pos.X), float64(pos.Y))
				screen.DrawImage(cardImg, cardOpts)
			} else {
				vector.DrawFilledRect(screen, float32(pos.X), float32(pos.Y),
					float32(fieldCardW), float32(fieldCardH), color.RGBA{60, 60, 80, 255}, false)
				vector.StrokeRect(screen, float32(pos.X), float32(pos.Y),
					float32(fieldCardW), float32(fieldCardH), 1, color.RGBA{120, 120, 140, 255}, false)
				nameTxt := elements.NewText(10, perm.Name, pos.X+4, pos.Y+4)
				nameTxt.Color = color.RGBA{220, 220, 220, 255}
				nameTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
			}

			if perm.IsCreature {
				s.drawCreatureStats(screen, perm, pos)
			}

			if !perm.Tapped {
				icons := s.getKeywordIcons(perm)
				for idx, icon := range icons {
					iconX := pos.X + idx*22
					if iconX+22 > pos.X+fieldCardW {
						break
					}
					iconOpts := &ebiten.DrawImageOptions{}
					iconOpts.GeoM.Translate(float64(iconX), float64(pos.Y+fieldCardH-22))
					screen.DrawImage(icon, iconOpts)
				}
			}

			if actions, hasAction := s.cardActions[perm.ID]; hasAction && dp == s.self && s.targetingCardID == uuid.Nil {
				if hasActionType(actions, interactive.ActionSelectAttackers) {
					borderColor := color.RGBA{255, 255, 0, 255}
					if s.pendingAttackers[perm.ID] {
						borderColor = color.RGBA{0, 255, 0, 255}
					}
					vector.StrokeRect(screen,
						float32(pos.X), float32(pos.Y),
						float32(fieldCardW), float32(fieldCardH),
						2, borderColor, false)
				} else {
					star := elements.NewText(14, "*", pos.X+2, pos.Y+2)
					star.Color = color.RGBA{255, 255, 0, 255}
					star.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
				}
			}

			if dp == s.self && s.isInDeclareBlockers() && s.targetingCardID == uuid.Nil {
				if s.selectedBlocker == perm.ID {
					vector.StrokeRect(screen,
						float32(pos.X), float32(pos.Y),
						float32(fieldCardW), float32(fieldCardH),
						2, color.RGBA{0, 255, 0, 255}, false)
				} else if s.pendingBlockers[perm.ID] != uuid.Nil {
					vector.StrokeRect(screen,
						float32(pos.X), float32(pos.Y),
						float32(fieldCardW), float32(fieldCardH),
						2, color.RGBA{0, 255, 0, 255}, false)
				} else if s.canBlockAnything(perm.ID) {
					vector.StrokeRect(screen,
						float32(pos.X), float32(pos.Y),
						float32(fieldCardW), float32(fieldCardH),
						2, color.RGBA{255, 255, 0, 255}, false)
				}
			}

			if dp == s.opponent && s.isInDeclareBlockers() && perm.Attacking && s.canBeBlocked(perm.ID) {
				borderColor := color.RGBA{255, 255, 0, 255}
				if hasKeyword(perm.Keywords, "Menace") {
					borderColor = color.RGBA{255, 140, 0, 255}
				}
				if s.selectedBlocker != uuid.Nil && s.isValidBlock(s.selectedBlocker, perm.ID) {
					borderColor = color.RGBA{0, 255, 0, 255}
				}
				vector.StrokeRect(screen,
					float32(pos.X), float32(pos.Y),
					float32(fieldCardW), float32(fieldCardH),
					2, borderColor, false)
			}

			if s.targetingCardID != uuid.Nil {
				if _, isTarget := s.targetingActions[perm.ID]; isTarget {
					borderColor := color.RGBA{255, 255, 0, 255}
					strokeW := float32(2)
					if s.selectedTargetID == perm.ID {
						borderColor = color.RGBA{0, 255, 0, 255}
						strokeW = 3
					}
					vector.StrokeRect(screen, float32(pos.X), float32(pos.Y),
						float32(fieldCardW), float32(fieldCardH), strokeW, borderColor, false)
				}
			}
		}
	}
}

func (s *DuelScreen) drawCreatureStats(screen *ebiten.Image, perm interactive.PermanentState, pos image.Point) {
	statText := fmt.Sprintf("%d/%d", perm.Power, perm.Toughness)
	textX := pos.X + fieldCardW - len(statText)*8 - 2
	textY := pos.Y + fieldCardH - 18
	bg := elements.NewText(16, statText, textX-1, textY-1)
	bg.Color = color.RGBA{0, 0, 0, 200}
	bg.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	stat := elements.NewText(16, statText, textX, textY)

	domainCard := s.getDomainCard(perm.Name)
	if domainCard != nil && (perm.Power > domainCard.Power || perm.Toughness > domainCard.Toughness) {
		stat.Color = color.RGBA{100, 255, 100, 255}
	} else if domainCard != nil && (perm.Power < domainCard.Power || perm.Toughness < domainCard.Toughness) {
		stat.Color = color.RGBA{255, 100, 100, 255}
	} else {
		stat.Color = color.RGBA{255, 255, 255, 255}
	}
	stat.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

// getAIBlockerArrows returns a map of AI blocker ID -> player attacker ID
// from the game state's Blocking field on opponent creatures.
func (s *DuelScreen) getAIBlockerArrows() map[uuid.UUID]uuid.UUID {
	arrows := make(map[uuid.UUID]uuid.UUID)
	if s.lastMsg == nil {
		return arrows
	}
	for _, perm := range s.lastMsg.State.Opponent.Battlefield {
		if perm.Blocking != uuid.Nil {
			arrows[perm.ID] = perm.Blocking
		}
	}
	return arrows
}

func (s *DuelScreen) drawBlockerArrows(screen *ebiten.Image) {
	step := ""
	if s.lastMsg != nil && s.lastMsg.State != nil {
		step = s.lastMsg.State.Step
	}
	showAIArrows := step == stepDeclareBlockers || step == "First Strike Damage"
	if showAIArrows {
		for blockerID, attackerID := range s.getAIBlockerArrows() {
			s.drawArrow(screen, blockerID, attackerID)
		}
	}
	for blockerID, attackerID := range s.pendingBlockers {
		s.drawArrow(screen, blockerID, attackerID)
	}
}

func (s *DuelScreen) drawArrow(screen *ebiten.Image, blockerID, attackerID uuid.UUID) {
	blockerPos, bOK := s.cardPositions[blockerID]
	attackerPos, aOK := s.cardPositions[attackerID]
	if !bOK || !aOK {
		return
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
		return
	}
	dx /= length
	dy /= length

	arrowLen := float32(10)
	px := -dy
	py := dx
	vector.StrokeLine(screen, ax, ay, ax+dx*arrowLen+px*arrowLen*0.5, ay+dy*arrowLen+py*arrowLen*0.5, 2, lineColor, false)
	vector.StrokeLine(screen, ax, ay, ax+dx*arrowLen-px*arrowLen*0.5, ay+dy*arrowLen-py*arrowLen*0.5, 2, lineColor, false)
}

func (s *DuelScreen) drawHandPanel(screen *ebiten.Image, dp *duelPlayer, ps interactive.PlayerState) {
	if dp.handBg == nil {
		return
	}

	handBgW := dp.handBg.Bounds().Dx()
	handBgH := dp.handBg.Bounds().Dy()

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(dp.handX), float64(dp.handY))
	screen.DrawImage(dp.handBg, opts)

	label := fmt.Sprintf("Your Hand (%d)", ps.HandCount)
	txt := elements.NewText(16, label, dp.handX+15, dp.handY+13)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if dp != s.self {
		return
	}

	for i, card := range ps.Hand {
		y := dp.handY + handBgH + i*handCardOverlap
		cardImg := s.getCardArtImg(card.Name, handBgW)
		if cardImg != nil {
			cardOpts := &ebiten.DrawImageOptions{}
			cardOpts.GeoM.Translate(float64(dp.handX), float64(y))
			screen.DrawImage(cardImg, cardOpts)
		} else {
			vector.DrawFilledRect(screen, float32(dp.handX), float32(y),
				float32(handBgW), float32(handCardOverlap+10), color.RGBA{60, 60, 80, 255}, false)
			nameTxt := elements.NewText(10, card.Name, dp.handX+4, y+2)
			nameTxt.Color = color.RGBA{220, 220, 220, 255}
			nameTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}

		if _, hasAction := s.cardActions[card.ID]; hasAction {
			star := elements.NewText(14, "*", dp.handX+2, y+2)
			star.Color = color.RGBA{255, 255, 0, 255}
			star.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}
	}
}

func drawManaPool(screen, manaPoolBg *ebiten.Image, ps interactive.PlayerState, manaPoolY int) {
	const manaPoolX = 120

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(manaPoolX, float64(manaPoolY))
	screen.DrawImage(manaPoolBg, opts)

	manaCounts := []int{
		ps.ManaPool.Black,
		ps.ManaPool.Blue,
		ps.ManaPool.Green,
		ps.ManaPool.Red,
		ps.ManaPool.White,
		ps.ManaPool.Colorless,
	}
	for i, count := range manaCounts {
		x := manaPoolX + 50
		y := (i * 30) + 10
		countTxt := elements.NewText(24, fmt.Sprintf("%d", count), x, manaPoolY+y)
		countTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}
}

func drawLife(screen *ebiten.Image, dp *duelPlayer, life int, Y int) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(0, float64(Y))
	screen.DrawImage(dp.lifeBg, opts)
	countTxt := elements.NewText(64, fmt.Sprintf("%d", life), 15, Y)
	countTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func drawGraveyard(screen *ebiten.Image, player *duelPlayer, Y float64) {
	if player.graveyardImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(60, Y)
		screen.DrawImage(player.graveyardImg, opts)
	}
}

func (s *DuelScreen) drawSidebar(screen *ebiten.Image, W, H int) {
	if s.lastMsg == nil {
		return
	}
	drawManaPool(screen, s.manaPoolBg, s.lastMsg.State.Opponent, 0)
	drawManaPool(screen, s.manaPoolBg, s.lastMsg.State.You, 580)

	drawGraveyard(screen, s.opponent, 94)
	drawGraveyard(screen, s.self, 580)

	drawLife(screen, s.opponent, s.lastMsg.State.Opponent.Life, 0)
	drawLife(screen, s.self, s.lastMsg.State.You.Life, 671)
}

func (s *DuelScreen) drawCardPreview(screen *ebiten.Image, H int) {
	if s.cardPreviewImg == nil {
		return
	}
	previewX := 0
	previewY := 188
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(previewX), float64(previewY))
	screen.DrawImage(s.cardPreviewImg, opts)

	if s.cardPreviewName != "" {
		domainCard := s.getDomainCard(s.cardPreviewName)
		if domainCard != nil && domainCard.CardType == domain.CardTypeCreature {
			power, toughness := domainCard.Power, domainCard.Toughness
			if s.cardPreviewPerm != nil {
				power, toughness = s.cardPreviewPerm.Power, s.cardPreviewPerm.Toughness
			}
			imgW := s.cardPreviewImg.Bounds().Dx()
			imgH := s.cardPreviewImg.Bounds().Dy()
			statText := fmt.Sprintf("%d/%d", power, toughness)
			textX := previewX + imgW - len(statText)*9 - 4
			textY := previewY + imgH - 22
			bg := elements.NewText(16, statText, textX-1, textY-1)
			bg.Color = color.RGBA{0, 0, 0, 200}
			bg.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
			stat := elements.NewText(16, statText, textX, textY)
			if s.cardPreviewPerm != nil && (power > domainCard.Power || toughness > domainCard.Toughness) {
				stat.Color = color.RGBA{100, 255, 100, 255}
			} else if s.cardPreviewPerm != nil && (power < domainCard.Power || toughness < domainCard.Toughness) {
				stat.Color = color.RGBA{255, 100, 100, 255}
			} else {
				stat.Color = color.RGBA{255, 255, 255, 255}
			}
			stat.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}
	}
}

func (s *DuelScreen) stackDescription() string {
	if s.lastMsg == nil || len(s.lastMsg.State.StackItems) == 0 {
		return ""
	}
	var names []string
	for _, item := range s.lastMsg.State.StackItems {
		names = append(names, item.Name)
	}
	if len(names) == 0 {
		return ""
	}
	return fmt.Sprintf("Responding to %s. ", strings.Join(names, ", "))
}

func (s *DuelScreen) statusMessage() string {
	if s.lastMsg == nil {
		return ""
	}
	state := s.lastMsg.State
	step := state.Step
	isMyTurn := state.ActivePlayer == playerNameYou

	stackMsg := s.stackDescription()

	if isCombatStep(step) {
		return stackMsg + s.combatStatusMessage(step, isMyTurn)
	}

	prefix := "Your"
	if !isMyTurn {
		prefix = s.opponent.name + "'s"
	}

	switch step {
	case "Untap":
		return stackMsg + fmt.Sprintf("%s turn - Untap", prefix)
	case "Upkeep":
		return stackMsg + fmt.Sprintf("%s turn - Upkeep", prefix)
	case "Draw":
		return stackMsg + fmt.Sprintf("%s turn - Draw", prefix)
	case "Precombat Main":
		if isMyTurn {
			return stackMsg + "Main phase: play a land or cast spells. Done to go to combat."
		}
		return stackMsg + fmt.Sprintf("%s main phase. Cast instants or Done to pass.", prefix)
	case "Postcombat Main":
		if isMyTurn {
			return stackMsg + "Main phase 2: play a land or cast spells. Done to end turn."
		}
		return stackMsg + fmt.Sprintf("%s main phase 2. Cast instants or Done to pass.", prefix)
	case "End Step":
		return stackMsg + fmt.Sprintf("%s turn - End step", prefix)
	case "Cleanup":
		return stackMsg + fmt.Sprintf("%s turn - Cleanup", prefix)
	default:
		return stackMsg
	}
}

func isCombatStep(step string) bool {
	switch step {
	case "Begin Combat", stepDeclareAttackers, stepDeclareBlockers,
		"First Strike Damage", stepCombatDamage, "End of Combat":
		return true
	}
	return false
}

func (s *DuelScreen) combatStatusMessage(step string, isMyTurn bool) string {
	switch step {
	case "Begin Combat":
		if isMyTurn {
			return "Beginning of combat"
		}
		return fmt.Sprintf("%s declares combat", s.opponent.name)
	case stepDeclareAttackers:
		if isMyTurn {
			return "Choose creatures to attack with. Done when finished."
		}
		return fmt.Sprintf("%s is choosing attackers...", s.opponent.name)
	case stepDeclareBlockers:
		if !isMyTurn {
			return "Choose creatures to block with. Done when finished."
		}
		if s.lastMsg.Prompt == interactive.PromptPriority {
			return "Blockers declared. Cast instants or Done to continue."
		}
		return fmt.Sprintf("%s is choosing blockers...", s.opponent.name)
	case "First Strike Damage":
		return "First strike damage"
	case stepCombatDamage:
		return "Combat damage resolves"
	case "End of Combat":
		return "End of combat"
	default:
		if isMyTurn {
			return "Your combat phase"
		}
		return fmt.Sprintf("%s's combat phase", s.opponent.name)
	}
}

func hasActionType(actions []interactive.ActionOption, actionType interactive.ActionType) bool {
	for _, a := range actions {
		if a.Type == actionType {
			return true
		}
	}
	return false
}
