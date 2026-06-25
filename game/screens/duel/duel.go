package duel

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	_ "github.com/benprew/mage-go/cards"
	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/search"
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
	stepPrecombatMain     = "Precombat Main"
	stepBeginCombat       = "Begin Combat"
	stepDeclareAttackers  = "Declare Attackers"
	stepDeclareBlockers   = "Declare Blockers"
	stepEndOfCombat       = "End of Combat"
	stepCombatDamage      = "Combat Damage"
	stepFirstStrikeDamage = "First Strike Damage"
)

// Minimum time each game message stays on screen before the next one is shown,
// so phases don't flash past faster than the player can follow. Enemy actions
// and life-total changes linger longer because those are the moments the player
// most needs to register (e.g. a Ball Lightning being cast and connecting).
const (
	phaseDisplayDelay = 100 * time.Millisecond
	enemyPhaseDelay   = 300 * time.Millisecond
	lifeChangeDelay   = 600 * time.Millisecond
)

const (
	lossLifeAnimationDuration = 900 * time.Millisecond
	lossLifeHoldDuration      = 500 * time.Millisecond
)

const (
	attackerLiftOffset   = 20.0
	attackerLiftDuration = 150 * time.Millisecond
)

type duelPlayer struct {
	name         string
	boardBg      *ebiten.Image
	handBg       *ebiten.Image
	lifeBg       *ebiten.Image
	graveyardImg *ebiten.Image
	handX, handY int
}

type lifeCounterAnimation struct {
	started   bool
	startedAt time.Time
	from      int
	to        int
}

type attackerLiftAnimation struct {
	startedAt time.Time
	from      float64
	to        float64
}

// dungeonDuelContext marks a duel as taking place inside a dungeon. When set,
// the duel skips the overworld level bookkeeping (which is keyed by enemy index)
// and instead resolves against the dungeon tile, returning to the dungeon
// screen afterwards.
type dungeonDuelContext struct {
	state *domain.DungeonState
	tile  *domain.DungeonTile
}

type DuelScreen struct {
	player  *domain.Player
	enemy   *domain.Enemy
	lvl     *world.Level
	idx     int
	dungeon *dungeonDuelContext

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

	cardPositions map[uuid.UUID]image.Point

	cardImgCache map[cardImgKey]cardImgEntry

	cardActions      map[uuid.UUID][]interactive.ActionOption
	pendingAttackers map[uuid.UUID]bool
	attackerLifts    map[uuid.UUID]*attackerLiftAnimation
	pendingBlockers  map[uuid.UUID]uuid.UUID
	selectedBlocker  uuid.UUID

	damageAssignment map[uuid.UUID]int
	damageAttackerID uuid.UUID
	damageTotal      int

	targetingCardID  uuid.UUID
	targetingActions map[uuid.UUID]interactive.ActionOption
	selectedTargetID uuid.UUID

	xChoosingActions []interactive.ActionOption
	xButtons         []*elements.Button
	xMaxValue        int
	xChosenValue     int

	abilityChoosingActions []interactive.ActionOption
	abilityButtons         []*elements.Button

	cardImageMap map[string]*domain.Card

	frameCount   int
	warningMsg   string
	lastMsgTime  time.Time
	nextMsgDelay time.Duration

	anteCard      *domain.Card
	enemyAnteCard *domain.Card

	// diceNotice describes the dungeon dice effects active for this duel, shown
	// as a banner at the top of the screen. Empty for ordinary duels.
	diceNotice string

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

	selfLifeAnimation     lifeCounterAnimation
	opponentLifeAnimation lifeCounterAnimation

	viewingGraveyard *duelPlayer

	handCollapsed bool

	inMulligan          bool
	mulliganCount       int
	mulliganBottoming   bool
	mulliganSelected    map[uuid.UUID]bool
	mulliganKeepBtn     *elements.Button
	mulliganMullBtn     *elements.Button
	mulliganConfirmBtn  *elements.Button
	mulliganPreviewImg  *ebiten.Image
	mulliganPreviewIdx  int
	mulliganPreviewName string
}

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
		attackerLifts:    make(map[uuid.UUID]*attackerLiftAnimation),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
	}

	s.initGameState()
	s.loadImages()

	s.self.handX = 860
	s.self.handY = 430
	s.opponent.handX = 860
	s.opponent.handY = 310

	s.initMulligan()

	return s
}

// NewDungeonDuelScreen starts a duel against a dungeon enemy. There is no ante
// screen or bribe option inside a dungeon: the duel begins immediately when the
// player lands on the enemy's tile. Ante cards are still wagered, mirroring an
// overworld duel.
func NewDungeonDuelScreen(player *domain.Player, enemy *domain.Enemy, state *domain.DungeonState, tile *domain.DungeonTile) *DuelScreen {
	var anteCard *domain.Card
	var enemyAnteCard *domain.Card
	s := NewDuelScreen(player, enemy, nil, -1, anteCard, enemyAnteCard)
	s.dungeon = &dungeonDuelContext{state: state, tile: tile}
	s.diceNotice = diceNotice(player.BonusDuelLife, player.BonusDuelCards)
	return s
}

// diceNotice builds the banner text summarizing the dice effects in force for a
// dungeon duel. Returns "" when there are no effects.
func diceNotice(lifeBonus int, cards []*domain.Card) string {
	var parts []string
	if lifeBonus > 0 {
		parts = append(parts, fmt.Sprintf("+%d starting life", lifeBonus))
	} else if lifeBonus < 0 {
		parts = append(parts, fmt.Sprintf("%d starting life", lifeBonus))
	}
	for _, c := range cards {
		parts = append(parts, fmt.Sprintf("%s in play", c.CardName))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Dice: " + strings.Join(parts, ", ")
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
	s.aiPlayer = ai.NewAIPlayer(s.enemy.Name(), heuristic.NewAdaptive())
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
	bonusPermanents := s.player.BonusDuelCards
	s.player.BonusDuelLife = 0
	s.player.BonusDuelCards = nil
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
	search.DebugStats = logging.Enabled(logging.Duel)

	logging.Printf(logging.Duel, "Drawing cards\n")

	for range 7 {
		s.human.DrawCard()
	}
	for range 7 {
		s.aiPlayer.DrawCard()
	}
	logging.Printf(logging.Duel, "Cards drawn\n")

	s.putBonusPermanentsInPlay(bonusPermanents)

	s.cardImageMap = buildCardImageMap(s.player.GetDuelDeck(), s.enemy.Character.GetActiveDeck())

	s.self = &duelPlayer{name: "You"}
	s.opponent = &duelPlayer{name: s.enemy.Name()}

	logging.Printf(logging.Duel, "Game init: human library=%d hand=%d, ai library=%d hand=%d, human life=%d, ai life=%d\n",
		len(s.human.Library()), len(s.human.Hand()),
		len(s.aiPlayer.Library()), len(s.aiPlayer.Hand()),
		s.human.Life(), s.aiPlayer.Life())
	logging.Printf(logging.Duel, "Game init: IsGameOver=%v Winner=%q\n", s.game.IsGameOver(), s.game.Winner())
}

// putBonusPermanentsInPlay drops each bonus card directly onto the human
// player's battlefield before the game loop starts. Used by dungeon dice that
// grant a card "in play" for the duel.
func (s *DuelScreen) putBonusPermanentsInPlay(cards []*domain.Card) {
	for _, card := range cards {
		c, err := mage.CreateCard(card.CardName)
		if err != nil {
			logging.Printf(logging.Duel, "Failed to create bonus permanent %s: %v\n", card.CardName, err)
			continue
		}
		c.SetOwner(s.human.PlayerID())
		s.game.PutOnBattlefield(c, s.human.PlayerID())
	}
}

func (s *DuelScreen) startGameLoop() {
	go interactive.RunGameLoop(s.game, 0, 300*time.Millisecond)
}

func (s *DuelScreen) drainMessages() {
	delay := s.nextMsgDelay
	if delay <= 0 {
		delay = phaseDisplayDelay
	}
	if time.Since(s.lastMsgTime) < delay {
		return
	}
	select {
	case msg, ok := <-s.human.ToTUI():
		if !ok {
			return
		}
		s.applyGameMsg(msg)
	default:
		return
	}
}

func (s *DuelScreen) applyGameMsg(msg interactive.GameMsg) {
	prev := s.lastMsg
	s.lastMsg = &msg
	s.refreshDamageAssignmentPrompt(&msg)
	s.lastMsgTime = time.Now()
	s.startLossAnimationFromMessage(prev, &msg, s.lastMsgTime)
	s.nextMsgDelay = phaseDelay(prev, &msg)
	s.checkSoundTriggers(&msg)
	if logging.Enabled(logging.Duel) {
		optNames := make([]string, len(msg.Options))
		for i, o := range msg.Options {
			optNames[i] = fmt.Sprintf("%s(%s)", o.Type, o.CardName)
		}
		logging.Printf(logging.Duel, "MSG: turn=%d step=%q active=%q prompt=%v options=%v gameover=%v\n",
			msg.State.Turn, msg.State.Step, msg.State.ActivePlayer, msg.Prompt, optNames, msg.GameOver)
	}
}

func (s *DuelScreen) drainQueuedMessages() {
	if s.human == nil {
		return
	}
	for {
		select {
		case msg, ok := <-s.human.ToTUI():
			if !ok {
				return
			}
			s.applyGameMsg(msg)
		default:
			return
		}
	}
}

// phaseDelay returns how long cur should stay on screen before the next message
// is shown. Enemy turns and life-total changes get a longer dwell so the player
// can follow what happened instead of phases racing past.
func phaseDelay(prev, cur *interactive.GameMsg) time.Duration {
	if cur == nil || cur.State == nil {
		return phaseDisplayDelay
	}
	delay := phaseDisplayDelay
	if cur.State.ActivePlayer != cur.State.You.Name {
		delay = max(delay, enemyPhaseDelay)
	}
	if prev != nil && prev.State != nil &&
		(cur.State.You.Life != prev.State.You.Life ||
			cur.State.Opponent.Life != prev.State.Opponent.Life ||
			permanentDamageChanged(prev.State, cur.State) ||
			newDamageLog(prev.Log, cur.Log)) {
		delay = max(delay, lifeChangeDelay)
	}
	return delay
}

func permanentDamageChanged(prev, cur *interactive.GameState) bool {
	prevDamage := permanentDamageByID(prev.You.Battlefield, prev.Opponent.Battlefield)
	for _, perm := range cur.You.Battlefield {
		if damage, ok := prevDamage[perm.ID]; ok && damage != perm.Damage {
			return true
		}
	}
	for _, perm := range cur.Opponent.Battlefield {
		if damage, ok := prevDamage[perm.ID]; ok && damage != perm.Damage {
			return true
		}
	}
	return false
}

func permanentDamageByID(groups ...[]interactive.PermanentState) map[uuid.UUID]int {
	damage := make(map[uuid.UUID]int)
	for _, group := range groups {
		for _, perm := range group {
			damage[perm.ID] = perm.Damage
		}
	}
	return damage
}

func newDamageLog(prev, cur []string) bool {
	if len(cur) <= len(prev) {
		return false
	}
	for _, line := range cur[len(prev):] {
		if strings.Contains(line, " deals ") && strings.Contains(line, " damage to ") {
			return true
		}
	}
	return false
}

func (s *DuelScreen) startLossAnimationFromMessage(prev, cur *interactive.GameMsg, now time.Time) {
	if cur == nil || cur.State == nil || !cur.GameOver {
		return
	}
	fromYou := cur.State.You.Life
	fromOpponent := cur.State.Opponent.Life
	if prev != nil && prev.State != nil {
		fromYou = prev.State.You.Life
		fromOpponent = prev.State.Opponent.Life
	}
	if cur.State.You.Life <= 0 {
		s.selfLifeAnimation.start(fromYou, cur.State.You.Life, now)
	}
	if cur.State.Opponent.Life <= 0 {
		s.opponentLifeAnimation.start(fromOpponent, cur.State.Opponent.Life, now)
	}
}

func (a *lifeCounterAnimation) start(from, to int, now time.Time) {
	if a.started {
		return
	}
	a.started = true
	a.startedAt = now
	a.from = from
	a.to = to
}

func (s *DuelScreen) displayedSelfLife(now time.Time) int {
	fallback := 0
	if s.lastMsg != nil && s.lastMsg.State != nil {
		fallback = s.lastMsg.State.You.Life
	}
	return s.displayedLife(s.selfLifeAnimation, fallback, now)
}

func (s *DuelScreen) displayedOpponentLife(now time.Time) int {
	fallback := 0
	if s.lastMsg != nil && s.lastMsg.State != nil {
		fallback = s.lastMsg.State.Opponent.Life
	}
	return s.displayedLife(s.opponentLifeAnimation, fallback, now)
}

func (s *DuelScreen) displayedLife(animation lifeCounterAnimation, fallback int, now time.Time) int {
	if !animation.started {
		return fallback
	}
	elapsed := now.Sub(animation.startedAt)
	if elapsed <= 0 {
		return animation.from
	}
	if elapsed >= lossLifeAnimationDuration {
		return animation.to
	}
	progress := float64(elapsed) / float64(lossLifeAnimationDuration)
	life := float64(animation.from) + float64(animation.to-animation.from)*progress
	return int(math.Round(life))
}

func (s *DuelScreen) lossAnimationComplete(now time.Time) bool {
	return s.selfLifeAnimation.complete(now) && s.opponentLifeAnimation.complete(now)
}

func (a lifeCounterAnimation) complete(now time.Time) bool {
	if !a.started {
		return true
	}
	return now.Sub(a.startedAt) >= lossLifeAnimationDuration+lossLifeHoldDuration
}

func (s *DuelScreen) attackerLiftY(id uuid.UUID, now time.Time) float64 {
	anim, ok := s.attackerLifts[id]
	if !ok {
		return 0
	}
	elapsed := now.Sub(anim.startedAt)
	if elapsed <= 0 {
		return anim.from
	}
	if elapsed >= attackerLiftDuration {
		return anim.to
	}
	progress := float64(elapsed) / float64(attackerLiftDuration)
	return anim.from + (anim.to-anim.from)*progress
}

func (s *DuelScreen) startAttackerLift(id uuid.UUID, to float64, now time.Time) {
	if s.attackerLifts == nil {
		s.attackerLifts = make(map[uuid.UUID]*attackerLiftAnimation)
	}
	s.attackerLifts[id] = &attackerLiftAnimation{
		startedAt: now,
		from:      s.attackerLiftY(id, now),
		to:        to,
	}
}

func (s *DuelScreen) syncAttackerLifts(now time.Time) {
	if s.lastMsg == nil || s.lastMsg.State == nil {
		return
	}
	targets := s.attackerLiftTargets()
	for id, target := range targets {
		if anim, ok := s.attackerLifts[id]; ok && anim.to == target {
			continue
		}
		s.startAttackerLift(id, target, now)
	}
	for id, anim := range s.attackerLifts {
		if _, ok := targets[id]; ok || anim.to == 0 {
			continue
		}
		s.startAttackerLift(id, 0, now)
	}
}

func (s *DuelScreen) attackerLiftTargets() map[uuid.UUID]float64 {
	targets := make(map[uuid.UUID]float64)
	state := s.lastMsg.State
	if state.ActivePlayer == "You" && state.Step == stepDeclareAttackers {
		for id := range s.pendingAttackers {
			targets[id] = -attackerLiftOffset
		}
		for id, anim := range s.attackerLifts {
			if anim.to != 0 {
				targets[id] = anim.to
			}
		}
		return targets
	}
	if !isCombatStep(state.Step) {
		return targets
	}
	for _, perm := range state.You.Battlefield {
		if perm.Attacking {
			targets[perm.ID] = -attackerLiftOffset
		}
	}
	for _, perm := range state.Opponent.Battlefield {
		if perm.Attacking {
			targets[perm.ID] = attackerLiftOffset
		}
	}
	if state.Step == stepEndOfCombat {
		for id, anim := range s.attackerLifts {
			if anim.to != 0 {
				targets[id] = anim.to
			}
		}
	}
	return targets
}

func (s *DuelScreen) drainChoiceRequests() {
	for {
		select {
		case req, ok := <-s.human.ChoiceRequests():
			if !ok {
				return
			}
			s.syncStateForChoice()
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

func (s *DuelScreen) syncStateForChoice() {
	s.drainQueuedMessages()
	if s.game == nil || s.human == nil {
		return
	}
	state := interactive.SnapshotGameState(s.game, s.humanPlayerIndex())
	if state == nil {
		return
	}
	msg := interactive.GameMsg{
		State:   state,
		Prompt:  interactive.PromptNone,
		Log:     append([]string(nil), s.lastLog()...),
		CanUndo: s.lastMsg != nil && s.lastMsg.CanUndo,
	}
	s.applyGameMsg(msg)
}

func (s *DuelScreen) humanPlayerIndex() int {
	if s.game == nil || s.human == nil {
		return 0
	}
	for i, p := range s.game.AllPlayers() {
		if p.PlayerID() == s.human.PlayerID() {
			return i
		}
	}
	return 0
}

func (s *DuelScreen) lastLog() []string {
	if s.lastMsg == nil {
		return nil
	}
	return s.lastMsg.Log
}

func (s *DuelScreen) refreshDamageAssignmentPrompt(msg *interactive.GameMsg) {
	if msg == nil || msg.Prompt != interactive.PromptAssignCombatDamage || len(msg.Options) == 0 {
		s.damageAssignment = nil
		s.damageAttackerID = uuid.Nil
		s.damageTotal = 0
		return
	}
	opt := msg.Options[0]
	s.damageAttackerID = opt.PermanentID
	s.damageTotal = opt.MaxXValue
	s.damageAssignment = suggestedDamageAssignment(s.blockersForDamageOption(opt), s.damageTotal)
}

func (s *DuelScreen) blockersForDamageOption(opt interactive.ActionOption) []interactive.PermanentState {
	blockers := make([]interactive.PermanentState, 0, len(opt.ValidTargets))
	if s.lastMsg == nil || s.lastMsg.State == nil {
		return blockers
	}
	byID := make(map[uuid.UUID]interactive.PermanentState)
	for _, perm := range s.lastMsg.State.Opponent.Battlefield {
		byID[perm.ID] = perm
	}
	for _, id := range opt.ValidTargets {
		if perm, ok := byID[id]; ok {
			blockers = append(blockers, perm)
		}
	}
	return blockers
}

func suggestedDamageAssignment(blockers []interactive.PermanentState, total int) map[uuid.UUID]int {
	assignment := make(map[uuid.UUID]int, len(blockers))
	remaining := total
	for i, blocker := range blockers {
		if remaining <= 0 {
			break
		}
		lethal := max(blocker.Toughness-blocker.Damage, 0)
		amount := min(lethal, remaining)
		if i == len(blockers)-1 {
			amount = remaining
		}
		assignment[blocker.ID] = amount
		remaining -= amount
	}
	return assignment
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

	abilityData, err := assets.DuelFS.ReadFile("art/screens/duel/Abilities.pic.png")
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
	data, err := assets.DuelFS.ReadFile("art/screens/duel/" + name)
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
	if s.inMulligan {
		s.updateMulliganUI(W, H)
		return screenui.DuelScr, nil, nil
	}

	s.drainMessages()
	s.drainChoiceRequests()

	if s.lastMsg == nil {
		return screenui.DuelScr, nil, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && s.targetingCardID == uuid.Nil && !s.isChoosingAbility() {
		s.submitPendingAndPass()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.handleEscape()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		s.toggleHand()
	}

	s.frameCount++

	if s.choiceRequest != nil {
		s.handleChoiceRequest()
		return screenui.DuelScr, nil, nil
	}

	if s.isChoosingAbility() {
		s.updateAbilityChoosingUI()
		return screenui.DuelScr, nil, nil
	}

	if s.isChoosingX() {
		s.updateXChoosingUI()
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
		if !s.lossAnimationComplete(time.Now()) {
			return screenui.DuelScr, nil, nil
		}
		if s.lastMsg.Winner == "You" {
			return s.handleWin()
		}
		return s.handleLoss()
	}

	return screenui.DuelScr, nil, nil
}

func (s *DuelScreen) handleEscape() {
	if s.viewingGraveyard != nil {
		s.viewingGraveyard = nil
		return
	}
	if s.isChoosingAbility() {
		s.exitAbilityChoosingMode()
		return
	}
	if s.isChoosingX() {
		s.exitXChoosingMode()
		return
	}
	if s.targetingCardID == uuid.Nil {
		return
	}
	if s.selectedTargetID != uuid.Nil {
		s.selectedTargetID = uuid.Nil
		return
	}
	s.exitTargetingMode()
}

const (
	handCardOverlap              = 20
	fieldCardW                   = 100
	fieldCardH                   = 83
	battlefieldCreatureStatsSize = 20
	creatureStatsRightPadding    = 3
	creatureStatsBottomInset     = 22
	cardPreviewCreatureStatsSize = 24
	cardPreviewStatsRightPadding = 6
	cardPreviewStatsBottomInset  = 31
)

type permRow int

const (
	permRowCreature permRow = iota // front row (closest to opponent)
	permRowOther                   // middle row (non-creature, non-land permanents)
	permRowLand                    // back row (lands)
)

var allPermRows = []permRow{permRowLand, permRowOther, permRowCreature}

func permRowFor(perm interactive.PermanentState) permRow {
	if perm.IsCreature {
		return permRowCreature
	}
	if perm.IsLand {
		return permRowLand
	}
	return permRowOther
}

func (s *DuelScreen) getFieldCardPos(perm interactive.PermanentState, dp *duelPlayer, idx int, total int, row permRow) image.Point {
	rowGap := 20
	rowH := fieldCardH + rowGap
	totalH := 3*fieldCardH + 2*rowGap
	var baseY int
	if dp == s.opponent {
		// Opponent half: 0 to duelMsgY (370px)
		// Lands at top (back), other middle, creatures at bottom (front, near center)
		topPad := (duelMsgY - totalH) / 2
		baseY = duelOpponentBoardY + topPad + int(row)*rowH
	} else {
		// Player half: duelPlayerBoardY to 768 (384px)
		// Creatures at top (front, near center), other middle, lands at bottom (back)
		boardH := 768 - duelPlayerBoardY
		topPad := (boardH - totalH) / 2
		switch row {
		case permRowCreature:
			baseY = duelPlayerBoardY + topPad
		case permRowOther:
			baseY = duelPlayerBoardY + topPad + rowH
		case permRowLand:
			baseY = duelPlayerBoardY + topPad + 2*rowH
		}
	}

	maxSpacing := 35
	if row == permRowCreature {
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

func (s *DuelScreen) fieldPerms(ps interactive.PlayerState, row permRow) []interactive.PermanentState {
	var perms []interactive.PermanentState
	for _, perm := range ps.Battlefield {
		if perm.AttachedTo != uuid.Nil {
			continue
		}
		if permRowFor(perm) == row {
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
	now := time.Now()
	for _, row := range allPermRows {
		perms := s.fieldPerms(*ps, row)
		for i := len(perms) - 1; i >= 0; i-- {
			perm := perms[i]
			pos := s.getFieldCardPos(perm, dp, i, len(perms), row)
			pos.Y += int(math.Round(s.attackerLiftY(perm.ID, now)))
			if mx >= pos.X && mx < pos.X+fieldCardW {
				auras := s.attachedPerms(perm.ID)
				for j, auraCopy := range slices.Backward(auras) {
					auraY := pos.Y - (j+1)*14
					if my >= auraY && my < auraY+14 {

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
	inDeclareAttackers := s.lastMsg.State.ActivePlayer == "You" &&
		step == stepDeclareAttackers
	s.syncAttackerLifts(time.Now())
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

// toggleHand collapses or expands the player's hand. Collapsing hides the hand
// card list so it no longer covers the player's attackers during combat, while
// the header bar stays visible so the hand can be expanded again.
func (s *DuelScreen) toggleHand() {
	s.handCollapsed = !s.handCollapsed
}

// pointInHandHeader reports whether the point is over the hand header bar, which
// stays clickable in both collapsed and expanded states to toggle the hand.
func (s *DuelScreen) pointInHandHeader(mx, my int, dp *duelPlayer) bool {
	w := s.panelCardW(dp)
	headerH := s.panelCardH(dp)
	return mx >= dp.handX && mx < dp.handX+w &&
		my >= dp.handY && my < dp.handY+headerH
}

func (s *DuelScreen) handCardIdxAtPoint(mx, my, panelX, panelY int, count int, dp *duelPlayer) int {
	if dp == s.self && s.handCollapsed {
		return -1
	}
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
		if slices.Contains(opt.ValidTargets, attackerID) {
			return true
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
		if slices.Contains(opt.ValidTargets, attackerID) {
			return true
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
	if s.lastMsg.Prompt == interactive.PromptAssignCombatDamage {
		s.handleDamageAssignmentClick(mx, my)
		return
	}
	if s.pointInHandHeader(mx, my, s.self) {
		s.toggleHand()
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
				s.startAttackerLift(perm.ID, 0, time.Now())
			} else {
				s.pendingAttackers[perm.ID] = true
				s.startAttackerLift(perm.ID, -attackerLiftOffset, time.Now())
			}
			return
		}
		s.performCardAction(perm.ID, perm.Name)
		return
	}
}

func (s *DuelScreen) handleDamageAssignmentClick(mx, my int) {
	blockerID, delta := s.damageControlAt(mx, my)
	if blockerID == uuid.Nil || delta == 0 {
		return
	}
	if delta > 0 {
		s.increaseAssignedDamage(blockerID)
		return
	}
	s.decreaseAssignedDamage(blockerID)
}

func (s *DuelScreen) damageControlAt(mx, my int) (uuid.UUID, int) {
	if s.damageAssignment == nil || s.lastMsg == nil || s.lastMsg.State == nil {
		return uuid.Nil, 0
	}
	for _, perm := range s.lastMsg.State.Opponent.Battlefield {
		if _, ok := s.damageAssignment[perm.ID]; !ok {
			continue
		}
		pos, ok := s.cardPositions[perm.ID]
		if !ok {
			continue
		}
		minus, plus := damageControlBounds(pos)
		if image.Pt(mx, my).In(minus) {
			return perm.ID, -1
		}
		if image.Pt(mx, my).In(plus) {
			return perm.ID, 1
		}
	}
	return uuid.Nil, 0
}

func damageControlBounds(pos image.Point) (image.Rectangle, image.Rectangle) {
	y := pos.Y + 4
	return image.Rect(pos.X+4, y, pos.X+22, y+18),
		image.Rect(pos.X+fieldCardW-22, y, pos.X+fieldCardW-4, y+18)
}

func (s *DuelScreen) increaseAssignedDamage(blockerID uuid.UUID) {
	for id, amount := range s.damageAssignment {
		if id != blockerID && amount > 0 {
			s.damageAssignment[id]--
			s.damageAssignment[blockerID]++
			return
		}
	}
}

func (s *DuelScreen) decreaseAssignedDamage(blockerID uuid.UUID) {
	if s.damageAssignment[blockerID] <= 0 {
		return
	}
	for id := range s.damageAssignment {
		if id != blockerID {
			s.damageAssignment[blockerID]--
			s.damageAssignment[id]++
			return
		}
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

	if actions[0].NeedsX {
		s.enterXChoosingMode(actions)
		return
	}

	if len(actions) > 1 {
		s.enterAbilityChoosingMode(actions)
		return
	}

	if actions[0].NeedsTarget {
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
				pa.XValue = s.xValueForAction()
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

func (s *DuelScreen) updateMouse(mx, my int) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	s.handleClick(mx, my)
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
	return slices.Contains(keywords, kw)
}

func (s *DuelScreen) submitPendingAndPass() {
	if s.lastMsg != nil && s.lastMsg.Prompt == interactive.PromptAssignCombatDamage {
		damage := make(map[uuid.UUID]int, len(s.damageAssignment))
		for id, amount := range s.damageAssignment {
			if amount > 0 {
				damage[id] = amount
			}
		}
		select {
		case s.human.FromTUI() <- interactive.PriorityAction{
			Type:        interactive.ActionAssignCombatDamage,
			Damage:      damage,
			DamageOrder: s.damageAssignmentOrder(),
		}:
		default:
		}
		return
	}

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

func (s *DuelScreen) damageAssignmentOrder() []uuid.UUID {
	if s.lastMsg == nil || len(s.lastMsg.Options) == 0 {
		return nil
	}
	blockers := s.blockersForDamageOption(s.lastMsg.Options[0])
	sort.SliceStable(blockers, func(i, j int) bool {
		iAssigned := s.damageAssignment[blockers[i].ID]
		jAssigned := s.damageAssignment[blockers[j].ID]
		iLethal := iAssigned >= max(blockers[i].Toughness-blockers[i].Damage, 0)
		jLethal := jAssigned >= max(blockers[j].Toughness-blockers[j].Damage, 0)
		if iLethal != jLethal {
			return iLethal
		}
		return iAssigned > jAssigned
	})
	order := make([]uuid.UUID, 0, len(blockers))
	for _, blocker := range blockers {
		order = append(order, blocker.ID)
	}
	return order
}

func (s *DuelScreen) handleClick(mx, my int) {
	if s.viewingGraveyard != nil {
		if !s.handleGraveyardClick(mx, my) {
			s.viewingGraveyard = nil
		}
		return
	}

	if s.handleGraveyardClick(mx, my) {
		return
	}

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

	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 160}, false)

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

	if s.dungeon != nil {
		s.dungeon.state.DefeatEnemy(s.dungeon.tile)
		return screenui.DungeonScr, nil, nil
	}

	choices := domain.RewardChoices(s.player.GetActiveDeck(), s.enemyAnteCard)

	s.lvl.RecordCombatWin()
	s.lvl.RemoveEnemyAt(s.idx)

	bonusCards := []*domain.Card{}
	if defeatedCastle := s.lvl.HandleCastleDuelOutcome(true); defeatedCastle != nil {
		bonus := domain.RandomPowerfulCardsForColor(defeatedCastle.Color, 5)
		for _, c := range bonus {
			s.player.CardCollection.AddCard(c, 1)
			bonusCards = append(bonusCards, c)
		}
	}

	return screenui.DuelWinScr, NewWinDuelScreen(s.player, choices, bonusCards), nil
}

func (s *DuelScreen) handleLoss() (screenui.ScreenName, screenui.Screen, error) {
	if s.anteCard != nil {
		_ = s.player.RemoveCard(s.anteCard)
	}
	lostCards := []*domain.Card{}
	if s.anteCard != nil {
		lostCards = append(lostCards, s.anteCard)
	}

	if s.dungeon != nil {
		// Losing a duel inside a dungeon expels the player back to the overworld.
		s.player.ExitDungeon()
		return screenui.DuelLoseScr, NewDuelLoseScreen(lostCards), nil
	}

	s.lvl.RecordCombatWin()
	s.lvl.RemoveEnemyAt(s.idx)
	s.lvl.HandleCastleDuelOutcome(false)

	return screenui.DuelLoseScr, NewDuelLoseScreen(lostCards), nil
}

func (s *DuelScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.Fill(color.RGBA{30, 30, 30, 255})

	if s.inMulligan {
		s.drawMulliganUI(screen, W, H)
		return
	}

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
	s.drawStackArrows(screen)
	s.drawHandPanel(screen, s.opponent, s.lastMsg.State.Opponent)
	s.drawHandPanel(screen, s.self, s.lastMsg.State.You)
	s.drawCardPreview(screen, H)
	s.drawDiceNotice(screen, W)
	s.drawGraveyardView(screen, W, H)
	s.drawChoiceUI(screen, W, H)
	s.drawXChoosingUI(screen, W, H)
	s.drawAbilityChoosingUI(screen, W, H)
}

// drawDiceNotice renders the dungeon dice banner across the top of the screen.
func (s *DuelScreen) drawDiceNotice(screen *ebiten.Image, W int) {
	if s.diceNotice == "" {
		return
	}
	face := &text.GoTextFace{Source: fonts.MtgFont, Size: 16}
	tw, th := text.Measure(s.diceNotice, face, 0)
	const padX, padY = 16.0, 6.0
	bw := tw + padX*2
	bh := th + padY*2
	bx := (float64(W) - bw) / 2
	const by = 2.0
	vector.FillRect(screen, float32(bx), float32(by), float32(bw), float32(bh), color.RGBA{20, 16, 40, 220}, false)
	vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 1, color.RGBA{200, 180, 90, 255}, false)
	txt := elements.NewText(16, s.diceNotice, int(bx+padX), int(by+padY))
	txt.Color = color.RGBA{240, 220, 130, 255}
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func (s *DuelScreen) drawGraveyardView(screen *ebiten.Image, W, H int) {
	if s.viewingGraveyard == nil {
		return
	}
	ps := s.playerState(s.viewingGraveyard)
	if ps == nil {
		return
	}

	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 200}, false)

	title := fmt.Sprintf("%s's Graveyard (%d)", s.viewingGraveyard.name, len(ps.Graveyard))
	titleTxt := elements.NewText(24, title, 0, 20)
	titleTxt.HAlign = elements.AlignCenter
	titleTxt.BoundsW = float64(W)
	titleTxt.Color = color.White
	titleTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if len(ps.Graveyard) == 0 {
		return
	}

	cardW := 150
	gap := 12
	startY := 60
	cols := max((W-gap)/(cardW+gap), 1)
	totalW := cols*cardW + (cols-1)*gap
	startX := (W - totalW) / 2

	for i, c := range ps.Graveyard {
		col := i % cols
		row := i / cols
		domainCard := s.getDomainCard(c.Name)
		if domainCard == nil {
			continue
		}
		img, err := domainCard.CardImage(domain.CardViewFull)
		if err != nil || img == nil {
			continue
		}
		scale := float64(cardW) / float64(img.Bounds().Dx())
		cardH := int(float64(img.Bounds().Dy()) * scale)
		x := startX + col*(cardW+gap)
		y := startY + row*(cardH+gap)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, opts)
	}
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
	if s.lastMsg.State.ActivePlayer == "You" {
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
		stepPrecombatMain:     3,
		stepBeginCombat:       4,
		stepDeclareAttackers:  4,
		stepDeclareBlockers:   4,
		stepFirstStrikeDamage: 4,
		stepCombatDamage:      4,
		stepEndOfCombat:       4,
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
	now := time.Now()
	for _, row := range allPermRows {
		perms := s.fieldPerms(ps, row)
		for i, perm := range perms {
			pos := s.getFieldCardPos(perm, dp, i, len(perms), row)
			pos.Y += int(math.Round(s.attackerLiftY(perm.ID, now)))
			s.cardPositions[perm.ID] = pos

			auras := s.attachedPerms(perm.ID)
			// reverse order so it draws correctly on the screen
			for j, aura := range slices.Backward(auras) {

				auraY := pos.Y - (j+1)*14
				auraImg := s.getCardArtImg(aura.Name, fieldCardW)
				if auraImg != nil {
					auraOpts := &ebiten.DrawImageOptions{}
					auraOpts.GeoM.Translate(float64(pos.X), float64(auraY))
					screen.DrawImage(auraImg, auraOpts)
				} else {
					vector.FillRect(screen, float32(pos.X), float32(auraY),
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
				vector.FillRect(screen, float32(pos.X), float32(pos.Y),
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

			s.drawPermanentBorders(screen, dp, perm, pos)
		}
	}
}

func (s *DuelScreen) drawPermanentBorders(screen *ebiten.Image, dp *duelPlayer, perm interactive.PermanentState, pos image.Point) {
	if dp == s.opponent && s.lastMsg != nil && s.lastMsg.Prompt == interactive.PromptAssignCombatDamage {
		if amount, ok := s.damageAssignment[perm.ID]; ok {
			vector.StrokeRect(screen,
				float32(pos.X), float32(pos.Y),
				float32(fieldCardW), float32(fieldCardH),
				2, color.RGBA{0, 255, 0, 255}, false)
			s.drawDamageControls(screen, pos, amount)
			return
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
			vector.StrokeRect(screen,
				float32(pos.X), float32(pos.Y),
				float32(fieldCardW), float32(fieldCardH),
				2, color.RGBA{255, 140, 0, 255}, false)
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

func (s *DuelScreen) drawDamageControls(screen *ebiten.Image, pos image.Point, amount int) {
	minus, plus := damageControlBounds(pos)
	for _, rect := range []image.Rectangle{minus, plus} {
		vector.FillRect(screen, float32(rect.Min.X), float32(rect.Min.Y),
			float32(rect.Dx()), float32(rect.Dy()), color.RGBA{0, 0, 0, 210}, false)
		vector.StrokeRect(screen, float32(rect.Min.X), float32(rect.Min.Y),
			float32(rect.Dx()), float32(rect.Dy()), 1, color.RGBA{255, 255, 255, 255}, false)
	}
	minusTxt := elements.NewText(16, "-", minus.Min.X+6, minus.Min.Y-1)
	minusTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	plusTxt := elements.NewText(16, "+", plus.Min.X+4, plus.Min.Y-1)
	plusTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	label := fmt.Sprintf("%d", amount)
	txt := elements.NewText(18, label, pos.X+fieldCardW/2-len(label)*5, pos.Y+3)
	txt.Color = color.RGBA{255, 255, 255, 255}
	bg := elements.NewText(18, label, txt.X-1, txt.Y-1)
	bg.Color = color.RGBA{0, 0, 0, 220}
	bg.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func (s *DuelScreen) drawCreatureStats(screen *ebiten.Image, perm interactive.PermanentState, pos image.Point) {
	power, toughness := displayedCreatureStats(perm)
	statText := creatureStatsText(perm)
	stat := elements.NewText(battlefieldCreatureStatsSize, statText, 0, 0)
	textW, _ := stat.Measure()
	textPos := creatureStatsTextPosition(pos, textW)
	bg := elements.NewText(battlefieldCreatureStatsSize, statText, textPos.X-1, textPos.Y-1)
	bg.Color = color.RGBA{0, 0, 0, 200}
	bg.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	stat.X = textPos.X
	stat.Y = textPos.Y

	domainCard := s.getDomainCard(perm.Name)
	if domainCard != nil && (power > domainCard.Power || toughness > domainCard.Toughness) {
		stat.Color = color.RGBA{100, 255, 100, 255}
	} else if domainCard != nil && (power < domainCard.Power || toughness < domainCard.Toughness) {
		stat.Color = color.RGBA{255, 100, 100, 255}
	} else {
		stat.Color = color.RGBA{255, 255, 255, 255}
	}
	stat.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func creatureStatsTextPosition(pos image.Point, textWidth float64) image.Point {
	return image.Point{
		X: pos.X + fieldCardW - int(math.Ceil(textWidth)) - creatureStatsRightPadding,
		Y: pos.Y + fieldCardH - creatureStatsBottomInset,
	}
}

func creatureStatsText(perm interactive.PermanentState) string {
	power, toughness := displayedCreatureStats(perm)
	return fmt.Sprintf("%d/%d", power, toughness)
}

func displayedCreatureStats(perm interactive.PermanentState) (int, int) {
	return perm.Power, max(perm.Toughness-perm.Damage, 0)
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
	showAIArrows := step == stepDeclareBlockers || step == stepFirstStrikeDamage
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

	drawArrowLine(screen, bx, by, ax, ay, color.RGBA{255, 0, 0, 255})
}

func drawArrowLine(screen *ebiten.Image, fromX, fromY, toX, toY float32, lineColor color.RGBA) {
	vector.StrokeLine(screen, fromX, fromY, toX, toY, 2, lineColor, false)

	dx := fromX - toX
	dy := fromY - toY
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length == 0 {
		return
	}
	dx /= length
	dy /= length

	arrowLen := float32(10)
	px := -dy
	py := dx
	vector.StrokeLine(screen, toX, toY, toX+dx*arrowLen+px*arrowLen*0.5, toY+dy*arrowLen+py*arrowLen*0.5, 2, lineColor, false)
	vector.StrokeLine(screen, toX, toY, toX+dx*arrowLen-px*arrowLen*0.5, toY+dy*arrowLen-py*arrowLen*0.5, 2, lineColor, false)
}

// drawStackArrows draws an arrow from the casting player's hand panel to each
// target of every spell currently on the stack, so the user can see what is
// being cast (especially by the opponent) and what it targets.
func (s *DuelScreen) drawStackArrows(screen *ebiten.Image) {
	if s.lastMsg == nil || len(s.lastMsg.State.StackItems) == 0 {
		return
	}
	for _, item := range s.lastMsg.State.StackItems {
		if len(item.TargetIDs) == 0 {
			continue
		}
		dp := s.self
		if item.Controller != "You" {
			dp = s.opponent
		}
		fromX, fromY := s.stackArrowOrigin(dp)
		for _, tid := range item.TargetIDs {
			tx, ty, ok := s.targetPosition(tid)
			if !ok {
				continue
			}
			drawArrowLine(screen, fromX, fromY, tx, ty, color.RGBA{255, 200, 0, 255})
		}
	}
}

func (s *DuelScreen) stackArrowOrigin(dp *duelPlayer) (float32, float32) {
	w, h := 0, 0
	if dp.handBg != nil {
		w = dp.handBg.Bounds().Dx()
		h = dp.handBg.Bounds().Dy()
	}
	return float32(dp.handX + w/2), float32(dp.handY + h/2)
}

func (s *DuelScreen) targetPosition(id uuid.UUID) (float32, float32, bool) {
	if pos, ok := s.cardPositions[id]; ok {
		return float32(pos.X) + float32(fieldCardW)/2, float32(pos.Y) + float32(fieldCardH)/2, true
	}
	if s.lastMsg == nil || s.lastMsg.State == nil {
		return 0, 0, false
	}
	if id == s.lastMsg.State.You.ID {
		return 30, float32(671 + 32), true
	}
	if id == s.lastMsg.State.Opponent.ID {
		return 30, 32, true
	}
	return 0, 0, false
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
	if dp != s.self {
		label = fmt.Sprintf("%s's Hand (%d)", dp.name, ps.HandCount)
	} else if s.handCollapsed {
		label = fmt.Sprintf("Your Hand (%d) [+]", ps.HandCount)
	}
	txt := elements.NewText(16, label, dp.handX+15, dp.handY+13)
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if dp == s.self && s.handCollapsed {
		return
	}

	// The opponent's hand is only populated when the engine reveals it
	// (interactive.RevealOpponentHand, a debug aid); otherwise it is empty and
	// nothing below draws.
	if dp != s.self && len(ps.Hand) == 0 {
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
			vector.FillRect(screen, float32(dp.handX), float32(y),
				float32(handBgW), float32(handCardOverlap+10), color.RGBA{60, 60, 80, 255}, false)
			nameTxt := elements.NewText(10, card.Name, dp.handX+4, y+2)
			nameTxt.Color = color.RGBA{220, 220, 220, 255}
			nameTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}
	}

	if dp != s.self {
		return
	}

	for i, card := range ps.Hand {
		if _, hasAction := s.cardActions[card.ID]; !hasAction {
			continue
		}
		y := dp.handY + handBgH + i*handCardOverlap
		cardH := handCardOverlap
		if i == len(ps.Hand)-1 {
			if cardImg := s.getCardArtImg(card.Name, handBgW); cardImg != nil {
				cardH = cardImg.Bounds().Dy()
			}
		}
		vector.StrokeRect(screen, float32(dp.handX), float32(y),
			float32(handBgW), float32(cardH), 2, color.RGBA{255, 140, 0, 255}, false)
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

const (
	graveyardX         = 60
	graveyardW         = 61
	graveyardH         = 91
	graveyardSelfY     = 580
	graveyardOpponentY = 94
)

func (s *DuelScreen) graveyardBounds(dp *duelPlayer) image.Rectangle {
	y := graveyardOpponentY
	if dp == s.self {
		y = graveyardSelfY
	}
	return image.Rect(graveyardX, y, graveyardX+graveyardW, y+graveyardH)
}

func (s *DuelScreen) drawGraveyard(screen *ebiten.Image, dp *duelPlayer) {
	bounds := s.graveyardBounds(dp)
	if dp.graveyardImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
		screen.DrawImage(dp.graveyardImg, opts)
	}

	ps := s.playerState(dp)
	if ps == nil || len(ps.Graveyard) == 0 {
		return
	}
	top := ps.Graveyard[len(ps.Graveyard)-1]
	art := s.getCardArtImg(top.Name, graveyardW)
	if art == nil {
		return
	}
	artH := art.Bounds().Dy()
	offsetY := max(bounds.Min.Y+(graveyardH-artH)/2, bounds.Min.Y)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(bounds.Min.X), float64(offsetY))
	screen.DrawImage(art, opts)
}

func (s *DuelScreen) handleGraveyardClick(mx, my int) bool {
	for _, dp := range []*duelPlayer{s.self, s.opponent} {
		if !image.Pt(mx, my).In(s.graveyardBounds(dp)) {
			continue
		}
		ps := s.playerState(dp)
		if ps == nil || len(ps.Graveyard) == 0 {
			return false
		}
		if s.viewingGraveyard == dp {
			s.viewingGraveyard = nil
		} else {
			s.viewingGraveyard = dp
		}
		return true
	}
	return false
}

func (s *DuelScreen) drawSidebar(screen *ebiten.Image, W, H int) {
	if s.lastMsg == nil {
		return
	}
	drawManaPool(screen, s.manaPoolBg, s.lastMsg.State.Opponent, 0)
	drawManaPool(screen, s.manaPoolBg, s.lastMsg.State.You, 580)

	s.drawGraveyard(screen, s.opponent)
	s.drawGraveyard(screen, s.self)

	now := time.Now()
	drawLife(screen, s.opponent, s.displayedOpponentLife(now), 0)
	drawLife(screen, s.self, s.displayedSelfLife(now), 671)
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
				power, toughness = displayedCreatureStats(*s.cardPreviewPerm)
			}
			imgW := s.cardPreviewImg.Bounds().Dx()
			imgH := s.cardPreviewImg.Bounds().Dy()
			statText := fmt.Sprintf("%d/%d", power, toughness)
			stat := elements.NewText(cardPreviewCreatureStatsSize, statText, 0, 0)
			textW, _ := stat.Measure()
			textPos := cardPreviewStatsTextPosition(
				image.Point{X: previewX, Y: previewY},
				image.Point{X: imgW, Y: imgH},
				textW,
			)
			bg := elements.NewText(cardPreviewCreatureStatsSize, statText, textPos.X-1, textPos.Y-1)
			bg.Color = color.RGBA{0, 0, 0, 200}
			bg.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
			stat.X = textPos.X
			stat.Y = textPos.Y
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

func cardPreviewStatsTextPosition(pos image.Point, size image.Point, textWidth float64) image.Point {
	return image.Point{
		X: pos.X + size.X - int(math.Ceil(textWidth)) - cardPreviewStatsRightPadding,
		Y: pos.Y + size.Y - cardPreviewStatsBottomInset,
	}
}

func (s *DuelScreen) stackDescription() string {
	if s.lastMsg == nil || len(s.lastMsg.State.StackItems) == 0 {
		return ""
	}
	var parts []string
	for _, item := range s.lastMsg.State.StackItems {
		parts = append(parts, formatStackItem(item))
	}
	return strings.Join(parts, ". ") + ". "
}

func formatStackItem(item interactive.StackItemState) string {
	verb := "casts"
	if item.IsAbility {
		verb = "activates"
	}
	desc := fmt.Sprintf("%s %s %s", item.Controller, verb, item.Name)
	if item.XValue > 0 {
		desc += fmt.Sprintf(" for %d", item.XValue)
	} else if item.EventAmount > 0 {
		desc += fmt.Sprintf(" for %d", item.EventAmount)
	}
	if len(item.Targets) > 0 {
		desc += " targeting " + strings.Join(item.Targets, ", ")
	}
	return desc
}

func (s *DuelScreen) statusMessage() string {
	if s.lastMsg == nil {
		return ""
	}
	state := s.lastMsg.State
	step := state.Step
	isMyTurn := state.ActivePlayer == "You"

	stackMsg := s.stackDescription()
	if s.lastMsg.Prompt == interactive.PromptAssignCombatDamage {
		return stackMsg + "Assign combat damage with +/- on blockers. Done when finished."
	}

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
	case stepPrecombatMain:
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
	case stepBeginCombat, stepDeclareAttackers, stepDeclareBlockers,
		stepFirstStrikeDamage, stepCombatDamage, stepEndOfCombat:
		return true
	}
	return false
}

func (s *DuelScreen) combatStatusMessage(step string, isMyTurn bool) string {
	switch step {
	case stepBeginCombat:
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
	case stepFirstStrikeDamage:
		return "First strike damage"
	case stepCombatDamage:
		return "Combat damage resolves"
	case stepEndOfCombat:
		return "End of combat"
	default:
		if isMyTurn {
			return "Your combat phase"
		}
		return fmt.Sprintf("%s's combat phase", s.opponent.name)
	}
}

func (s *DuelScreen) initMulligan() {
	s.inMulligan = true
	s.mulliganCount = 0
	s.mulliganBottoming = false
	s.mulliganSelected = make(map[uuid.UUID]bool)
	s.mulliganPreviewIdx = -1

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		logging.Printf(logging.Duel, "Error loading mulligan button sprites: %v\n", err)
		return
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 16}
	mkBtn := func(label string) *elements.Button {
		btn := elements.NewButton(btnSprites[0][0], btnSprites[0][1], btnSprites[0][2], 0, 0, 1.0)
		btn.ButtonText = elements.ButtonText{
			Text:      label,
			Font:      fontFace,
			TextColor: color.White,
			HAlign:    elements.AlignCenter,
			VAlign:    elements.AlignMiddle,
		}
		return btn
	}
	s.mulliganKeepBtn = mkBtn("Keep")
	s.mulliganMullBtn = mkBtn("Mulligan")
	s.mulliganConfirmBtn = mkBtn("Confirm")
}

// aiMulliganDecision runs a simple heuristic mulligan for the AI (London
// rules): keep hands with 2-5 lands, otherwise mulligan up to 2 times. After
// mulligans, place N cards from hand on the bottom of the library where N is
// the number of mulligans taken.
func (s *DuelScreen) aiMulliganDecision() {
	mulls := 0
	for mulls < 2 {
		lands := 0
		for _, c := range s.aiPlayer.Hand() {
			if dc := s.getDomainCard(c.Name()); dc != nil && dc.CardType == domain.CardTypeLand {
				lands++
			}
		}
		if lands >= 2 && lands <= 5 {
			break
		}
		for _, c := range s.aiPlayer.Hand() {
			if _, ok := s.aiPlayer.RemoveFromHand(c.ID()); ok {
				s.aiPlayer.AddToLibrary(c)
			}
		}
		s.aiPlayer.ShuffleLibrary()
		for range 7 {
			s.aiPlayer.DrawCard()
		}
		mulls++
	}
	if mulls == 0 {
		return
	}
	lib := s.aiPlayer.Library()
	hand := s.aiPlayer.Hand()
	for i := 0; i < mulls && len(hand) > 0; i++ {
		worst := hand[len(hand)-1]
		for _, c := range hand {
			dc := s.getDomainCard(c.Name())
			if dc != nil && dc.CardType == domain.CardTypeLand {
				worst = c
				break
			}
		}
		if c, ok := s.aiPlayer.RemoveFromHand(worst.ID()); ok {
			lib = append(lib, c)
		}
		hand = s.aiPlayer.Hand()
	}
	s.aiPlayer.SetLibrary(lib)
}

func (s *DuelScreen) doHumanMulligan() {
	hand := s.human.Hand()
	for _, c := range hand {
		if _, ok := s.human.RemoveFromHand(c.ID()); ok {
			s.human.AddToLibrary(c)
		}
	}
	s.human.ShuffleLibrary()
	for range 7 {
		s.human.DrawCard()
	}
	s.mulliganCount++
	s.mulliganSelected = make(map[uuid.UUID]bool)
	if s.mulliganCount >= 7 {
		s.finishMulligan()
	}
}

func (s *DuelScreen) finishMulligan() {
	if s.mulliganCount > 0 {
		lib := s.human.Library()
		for id := range s.mulliganSelected {
			if c, ok := s.human.RemoveFromHand(id); ok {
				lib = append(lib, c)
			}
		}
		s.human.SetLibrary(lib)
	}
	s.aiMulliganDecision()
	s.inMulligan = false
	s.mulliganSelected = nil
	s.startGameLoop()
}

const (
	mulliganCardW         = 120
	mulliganCardGap       = 12
	mulliganPreviewMargin = 20
)

func (s *DuelScreen) mulliganCardRects(W, H int) []image.Rectangle {
	hand := s.human.Hand()
	n := len(hand)
	if n == 0 {
		return nil
	}
	totalW := n*mulliganCardW + (n-1)*mulliganCardGap
	startX := (W - totalW) / 2
	cardH := int(float64(mulliganCardW) * 1.4)
	y := (H - cardH) / 2
	rects := make([]image.Rectangle, n)
	for i := range hand {
		x := startX + i*(mulliganCardW+mulliganCardGap)
		rects[i] = image.Rect(x, y, x+mulliganCardW, y+cardH)
	}
	return rects
}

func (s *DuelScreen) updateMulliganUI(W, H int) {
	cardRects := s.mulliganCardRects(W, H)
	hand := s.human.Hand()
	mx, my := ebiten.CursorPosition()
	s.updateMulliganPreview(cardRects, mx, my)

	if s.mulliganBottoming && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if i := rectIndexAtPoint(cardRects, mx, my); i >= 0 {
			id := hand[i].ID()
			if s.mulliganSelected[id] {
				delete(s.mulliganSelected, id)
			} else if len(s.mulliganSelected) < s.mulliganCount {
				s.mulliganSelected[id] = true
			}
		}
	}

	btnW, btnH := 160, 40
	btnY := H - 80
	if s.mulliganBottoming {
		s.mulliganConfirmBtn.MoveTo((W-btnW)/2, btnY)
		s.mulliganConfirmBtn.Update(&ebiten.DrawImageOptions{}, 1.0, W, H)
		if s.mulliganConfirmBtn.IsClicked() && len(s.mulliganSelected) == s.mulliganCount {
			s.finishMulligan()
		}
	} else {
		s.mulliganKeepBtn.MoveTo(W/2-btnW-10, btnY)
		s.mulliganKeepBtn.Update(&ebiten.DrawImageOptions{}, 1.0, W, H)
		if s.mulliganCount < 7 {
			s.mulliganMullBtn.MoveTo(W/2+10, btnY)
			s.mulliganMullBtn.Update(&ebiten.DrawImageOptions{}, 1.0, W, H)
		}
		if s.mulliganKeepBtn.IsClicked() {
			if s.mulliganCount == 0 {
				s.finishMulligan()
			} else {
				s.mulliganBottoming = true
			}
		} else if s.mulliganCount < 7 && s.mulliganMullBtn.IsClicked() {
			s.doHumanMulligan()
		}
	}
	_ = btnH
}

func rectIndexAtPoint(rects []image.Rectangle, x, y int) int {
	point := image.Pt(x, y)
	for i, rect := range rects {
		if point.In(rect) {
			return i
		}
	}
	return -1
}

func (s *DuelScreen) updateMulliganPreview(rects []image.Rectangle, mx, my int) {
	idx := rectIndexAtPoint(rects, mx, my)
	if idx < 0 {
		s.mulliganPreviewImg = nil
		s.mulliganPreviewIdx = -1
		s.mulliganPreviewName = ""
		return
	}

	card := s.human.Hand()[idx]
	if s.mulliganPreviewImg != nil && s.mulliganPreviewName == card.Name() {
		s.mulliganPreviewIdx = idx
		return
	}

	domainCard := s.getDomainCard(card.Name())
	if domainCard == nil {
		s.mulliganPreviewImg = nil
		s.mulliganPreviewIdx = -1
		s.mulliganPreviewName = ""
		return
	}
	img, err := domainCard.CardImage(domain.CardViewFull)
	if err != nil || img == nil {
		s.mulliganPreviewImg = nil
		s.mulliganPreviewIdx = -1
		s.mulliganPreviewName = ""
		return
	}
	s.mulliganPreviewImg = img
	s.mulliganPreviewIdx = idx
	s.mulliganPreviewName = card.Name()
}

func mulliganPreviewPosition(W, H int, cardRect, previewBounds image.Rectangle) image.Point {
	x := mulliganPreviewMargin
	if cardRect.Min.X+cardRect.Dx()/2 < W/2 {
		x = W - previewBounds.Dx() - mulliganPreviewMargin
	}
	y := (H - previewBounds.Dy()) / 2
	return image.Pt(max(0, min(x, W-previewBounds.Dx())), max(0, min(y, H-previewBounds.Dy())))
}

func (s *DuelScreen) drawMulliganUI(screen *ebiten.Image, W, H int) {
	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{20, 20, 30, 255}, false)

	var title string
	if s.mulliganBottoming {
		title = fmt.Sprintf("Select %d card(s) to put on the bottom of your library (%d selected)",
			s.mulliganCount, len(s.mulliganSelected))
	} else if s.mulliganCount == 0 {
		title = "Opening hand — Keep or Mulligan?"
	} else {
		title = fmt.Sprintf("Mulligan #%d — Keep this hand or take another mulligan?", s.mulliganCount)
	}
	t := elements.NewText(20, title, 0, 30)
	t.HAlign = elements.AlignCenter
	t.BoundsW = float64(W)
	t.Color = color.White
	t.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	hint := elements.NewText(14, "Hover a card to magnify it", 0, 64)
	hint.HAlign = elements.AlignCenter
	hint.BoundsW = float64(W)
	hint.Color = color.RGBA{190, 190, 205, 255}
	hint.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	hand := s.human.Hand()
	rects := s.mulliganCardRects(W, H)
	for i, c := range hand {
		r := rects[i]
		domainCard := s.getDomainCard(c.Name())
		drawn := false
		if domainCard != nil {
			if img, err := domainCard.CardImage(domain.CardViewFull); err == nil && img != nil {
				scale := float64(r.Dx()) / float64(img.Bounds().Dx())
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Scale(scale, scale)
				opts.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
				screen.DrawImage(img, opts)
				drawn = true
			}
		}
		if !drawn {
			vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y),
				float32(r.Dx()), float32(r.Dy()), color.RGBA{60, 60, 80, 255}, false)
			nameTxt := elements.NewText(12, c.Name(), r.Min.X+4, r.Min.Y+4)
			nameTxt.Color = color.White
			nameTxt.BoundsW = float64(r.Dx() - 8)
			nameTxt.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}
		if s.mulliganBottoming && s.mulliganSelected[c.ID()] {
			vector.StrokeRect(screen, float32(r.Min.X)-2, float32(r.Min.Y)-2,
				float32(r.Dx())+4, float32(r.Dy())+4, 3, color.RGBA{255, 80, 80, 255}, false)
		}
	}
	s.drawMulliganPreview(screen, W, H, rects)

	if s.mulliganBottoming {
		s.mulliganConfirmBtn.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	} else {
		s.mulliganKeepBtn.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		if s.mulliganCount < 7 {
			s.mulliganMullBtn.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
		}
	}
}

func (s *DuelScreen) drawMulliganPreview(screen *ebiten.Image, W, H int, cardRects []image.Rectangle) {
	if s.mulliganPreviewImg == nil || s.mulliganPreviewIdx < 0 || s.mulliganPreviewIdx >= len(cardRects) {
		return
	}

	pos := mulliganPreviewPosition(W, H, cardRects[s.mulliganPreviewIdx], s.mulliganPreviewImg.Bounds())
	bounds := s.mulliganPreviewImg.Bounds()
	vector.FillRect(screen, float32(pos.X-6), float32(pos.Y-6), float32(bounds.Dx()+12),
		float32(bounds.Dy()+12), color.RGBA{0, 0, 0, 210}, false)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(pos.X), float64(pos.Y))
	screen.DrawImage(s.mulliganPreviewImg, opts)
	vector.StrokeRect(screen, float32(pos.X-2), float32(pos.Y-2), float32(bounds.Dx()+4),
		float32(bounds.Dy()+4), 2, color.RGBA{230, 210, 120, 255}, false)
}

func hasActionType(actions []interactive.ActionOption, actionType interactive.ActionType) bool {
	for _, a := range actions {
		if a.Type == actionType {
			return true
		}
	}
	return false
}
