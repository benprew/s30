package screens

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"strings"

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
)

type WisemanState int

const (
	WisemanStateStory WisemanState = iota
	WisemanStateOffer
)

type WisemanScreen struct {
	BgImage       *ebiten.Image
	City          *domain.City
	Player        *domain.Player
	Level         *world.Level
	State         WisemanState
	TextLines     []string
	Page          int
	ProposedQuest *domain.Quest
	Buttons       []*elements.Button
}

func (s *WisemanScreen) IsFramed() bool {
	return false
}

// ==============================================================================
// 1.13 What do these Wise Men do?
// ==============================================================================
//
//   If you are currently on a quest, you can talk to the wise men in
//   the villages/towns. On the rare occasion you will get a nice gift. The
//   following is a listing of what you can get from wise men.
//
//   1. A story (taken from the introduction) There are about 5 of these..
//   2. +2 lives in the next duel
//   3. A dungeon clue
//   4. A deck of a creature from a particular color
//   5. Tell you where a world magic is.
//   6. Give you a card for the next duel.

func NewWisemanScreen(city *domain.City, player *domain.Player, level *world.Level) *WisemanScreen {
	bgImg, err := imageutil.LoadImage(assets.Wiseman_png)
	if err != nil {
		panic(fmt.Sprintf("Unable to load Wiseman.png: %s", err))
	}

	s := &WisemanScreen{
		BgImage: bgImg,
		City:    city,
		Player:  player,
		Level:   level,
	}

	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.SFXWizardMale)
	}

	s.determineState()
	return s
}

// determineState sets the wiseman screen state: it offers a quest or grants a
// boon. Quest completion, reward, and expiry are handled outside the Wiseman
// now (rewards are claimed by walking into any town).
func (s *WisemanScreen) determineState() {
	if s.maybeOfferDeckQuest() {
		return
	}

	if s.City.BoonGranted {
		s.loadStory()
		return
	}

	if s.City.WisemanBoon == domain.BoonQuest && !s.Player.CanAcceptQuest() {
		s.loadStory()
		return
	}

	switch s.City.WisemanBoon {
	case domain.BoonQuest:
		s.generateQuest()
	default:
		if rand.Float32() < 0.5 {
			s.grantBoon()
		} else {
			s.loadStory()
		}
	}
}

// deckQuestOfferChance is the probability the Wiseman offers a deck-changing
// quest (when the player has a free slot) instead of a boon / delivery quest.
// It is a var so tests can make the offer deterministic.
var deckQuestOfferChance float32 = 0.5

// maybeOfferDeckQuest offers a deck-changing quest when the player has a free
// quest slot. Returns true if an offer was set up.
func (s *WisemanScreen) maybeOfferDeckQuest() bool {
	if !s.Player.CanAcceptQuest() {
		return false
	}
	if rand.Float32() >= deckQuestOfferChance {
		return false
	}
	def := s.pickDeckQuestDef()
	if def == nil {
		return false
	}

	s.ProposedQuest = def.GenerateQuest(s.questProgressionLevel())
	s.State = WisemanStateOffer
	s.prepareQuestOfferText(s.ProposedQuest)
	s.setupButtons()
	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.SFXNewsflash)
	}
	return true
}

// pickDeckQuestDef chooses a random deck-quest template the player isn't
// already running, or nil if none are available.
func (s *WisemanScreen) pickDeckQuestDef() *domain.Quest {
	var eligible []*domain.Quest
	for _, def := range domain.QuestDefList() {
		if !s.Player.HasQuest(def.ID) {
			eligible = append(eligible, def)
		}
	}
	if len(eligible) == 0 {
		return nil
	}
	return eligible[rand.Intn(len(eligible))]
}

// questProgressionLevel maps the local enemy difficulty to a 0-based reward
// scaling level.
func (s *WisemanScreen) questProgressionLevel() int {
	lvl := max(s.questEnemyMaxLevel()-wisemanFallbackEnemyLevel, 0)
	return lvl
}

func (s *WisemanScreen) generateQuest() {
	s.State = WisemanStateOffer

	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.SFXNewsflash)
	}

	if s.City.ProposedQuest != nil {
		s.ProposedQuest = s.City.ProposedQuest
		s.prepareQuestOfferText(s.ProposedQuest)
		s.setupButtons()
		return
	}

	if rand.Float32() < 0.5 {
		s.generateDeliveryQuest()
	} else {
		s.generateDefeatEnemyQuest()
	}

	s.City.ProposedQuest = s.ProposedQuest
	s.setupButtons()
}

func (s *WisemanScreen) generateDeliveryQuest() {
	targetCity := s.findRandomCity()
	if targetCity == nil {
		s.State = WisemanStateStory
		s.loadStory()
		return
	}

	reward := domain.RandomSingleReward(s.questProgressionLevel(), targetCity.AmuletColor)
	s.ProposedQuest = &domain.Quest{
		Type:          domain.QuestTypeDelivery,
		TargetCity:    targetCity,
		DaysRemaining: 20 + rand.Intn(20),
		Reward:        reward,
	}

	rewardText := reward.Description()
	hooks := [][]string{
		{
			fmt.Sprintf("Dark forces threaten the road to %s.", targetCity.Name),
			"An urgent message must reach their keeper",
			"before the wards fail.",
		},
		{
			fmt.Sprintf("The people of %s are cut off", targetCity.Name),
			"and desperately need word from our village.",
			"Only a planeswalker can brave the journey.",
		},
		{
			"An ancient pact binds our village to",
			fmt.Sprintf("%s. Their keeper awaits a message", targetCity.Name),
			"that may turn the tide against Arzakon.",
		},
		{
			fmt.Sprintf("Our scouts report %s", targetCity.Name),
			"is under siege by Arzakon's minions.",
			"Deliver this scroll before it is too late.",
		},
	}
	hook := hooks[rand.Intn(len(hooks))]
	s.TextLines = append(hook,
		fmt.Sprintf("You will be rewarded with %s.", rewardText),
		"",
		"Accept the Quest?",
	)
}

func (s *WisemanScreen) generateDefeatEnemyQuest() {
	enemyName := randomRogueName(s.questEnemyMaxLevel())

	reward := domain.RandomSingleReward(s.questProgressionLevel(), s.City.AmuletColor)
	s.ProposedQuest = &domain.Quest{
		Type:          domain.QuestTypeDefeatEnemy,
		EnemyName:     enemyName,
		DaysRemaining: 15 + rand.Intn(15),
		Reward:        reward,
	}

	rewardText := reward.Description()
	hooks := [][]string{
		{
			fmt.Sprintf("A %s has been terrorizing", enemyName),
			"the roads near our village. Travelers",
			"are afraid to leave their homes.",
		},
		{
			fmt.Sprintf("A savage %s lurks nearby,", enemyName),
			"growing bolder each day. Our militia",
			"is no match for such a creature.",
		},
		{
			fmt.Sprintf("The dreaded %s has slain", enemyName),
			"two of our bravest warriors already.",
			"We need a planeswalker's strength.",
		},
		{
			"Our children cannot play outside",
			fmt.Sprintf("while the %s roams free.", enemyName),
			"Please, rid us of this menace.",
		},
	}
	hook := hooks[rand.Intn(len(hooks))]
	s.TextLines = append(hook,
		fmt.Sprintf("Slay it and earn %s.", rewardText),
		"",
		"Accept the Quest?",
	)
}

func (s *WisemanScreen) spawnQuestEnemy(enemyName string) {
	cityTile := image.Point{X: s.City.X, Y: s.City.Y}
	if err := s.Level.SpawnEnemyNear(enemyName, cityTile); err != nil {
		fmt.Println("Failed to spawn quest enemy:", err)
	}
}

// questEnemyMaxLevel returns the highest enemy level appropriate for a quest
// from this city, scaled to player progression so quests stay winnable.
func (s *WisemanScreen) questEnemyMaxLevel() int {
	if s.Level == nil {
		return wisemanFallbackEnemyLevel
	}
	return s.Level.EnemySpawnMaxLevelAt(image.Point{X: s.City.X, Y: s.City.Y})
}

const wisemanFallbackEnemyLevel = 2

// randomRogueName picks a random rogue whose level is in (0, maxLevel]. Quest
// levels are always at least baseEnemySpawnLevel and the weakest rogue is level
// 1, so a qualifying rogue normally always exists; the fallback only guards
// against a caller passing a cap below every rogue's level (avoids a panic on
// rand.Intn(0)).
func randomRogueName(maxLevel int) string {
	var names []string
	for name, char := range domain.Rogues {
		if char.Level > 0 && char.Level <= maxLevel {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return lowestLevelRogueName()
	}
	return names[rand.Intn(len(names))]
}

// lowestLevelRogueName returns the name of the weakest rogue (lowest positive
// level), used as a guaranteed-valid fallback enemy.
func lowestLevelRogueName() string {
	name := ""
	level := 0
	for n, char := range domain.Rogues {
		if char.Level > 0 && (name == "" || char.Level < level) {
			name = n
			level = char.Level
		}
	}
	return name
}

func (s *WisemanScreen) prepareQuestOfferText(q *domain.Quest) {
	rewardText := q.Reward.Description()
	switch q.Type {
	case domain.QuestTypeDelivery:
		s.TextLines = []string{
			fmt.Sprintf("We need a message delivered to %s.", q.TargetCity.Name),
			fmt.Sprintf("You will be rewarded with %s.", rewardText),
			"",
			"Accept the Quest?",
		}
	case domain.QuestTypeDefeatEnemy:
		s.TextLines = []string{
			fmt.Sprintf("A %s threatens our village.", q.EnemyName),
			fmt.Sprintf("Slay it and earn %s.", rewardText),
			"",
			"Accept the Quest?",
		}
	case domain.QuestTypeActionTracker:
		s.TextLines = append(questFlavorLines(q),
			q.Description+".",
			fmt.Sprintf("You have %d days.", q.DaysRemaining),
			fmt.Sprintf("Reward: %s.", rewardText),
			"",
			"Accept the Quest?",
		)
	case domain.QuestTypeDeckConstraint:
		s.TextLines = append(questFlavorLines(q),
			q.Description+".",
			"Edit your deck before you duel.",
			fmt.Sprintf("You have %d days.", q.DaysRemaining),
			fmt.Sprintf("Reward: %s.", rewardText),
			"",
			"Accept the Quest?",
		)
	}
}

// questFlavor maps a deck-changing quest template ID to the Wiseman's spoken
// intro, so each offer reads as a plea from the village rather than a bare
// objective line (mirroring the delivery/defeat-enemy hooks).
var questFlavor = map[string][]string{
	"cast_black_red": {
		"The old powers of shadow and flame stir again.",
		"Wield them often, planeswalker, and the dark roads",
		"will bend to your will.",
	},
	"cast_green_white": {
		"Field and forest whisper the same ancient song.",
		"Call upon green and white in your battles,",
		"and the land itself will fight beside you.",
	},
	"cast_blue": {
		"There is wisdom in the deep waters.",
		"Loose the magic of the tides upon your foes",
		"and they will drown in your cunning.",
	},
	"play_lands": {
		"A planeswalker is only as strong as the land",
		"that answers their call. Lay claim to it,",
		"again and again, until the soil knows your name.",
	},
	"attack_creatures": {
		"The time for caution has passed.",
		"Send your creatures forth without fear --",
		"Arzakon respects only those who strike.",
	},
	"destroy_enemy_creatures": {
		"Our enemies grow bold behind their monstrous host.",
		"Thin their ranks, planeswalker. Leave them nothing",
		"to hide behind.",
	},
	"cast_instants_sorceries": {
		"The cleverest mages win before the first blow lands.",
		"Show me you can sling spell after spell",
		"and turn any moment to your favor.",
	},
	"direct_damage": {
		"Steel and fang are not the only weapons.",
		"Burn your foes with raw magic alone,",
		"and let them learn to fear your hand.",
	},
	"mono_color_win": {
		"Scattered loyalties make a weak mage.",
		"Bind yourself to a single color and triumph --",
		"true power flows from purity of purpose.",
	},
	"fat_deck_win": {
		"Some say a lean deck wins the day.",
		"I say a planeswalker of true might can carry",
		"a great tome of spells to victory all the same.",
	},
	"low_curve_win": {
		"Patience is a luxury on the battlefield.",
		"Win with only the swiftest, smallest creatures",
		"and prove that speed conquers all.",
	},
	"no_blue_win": {
		"The tides of the deep are not to be trusted.",
		"Spurn the blue arts entirely and claim your win --",
		"let no cunning current carry you.",
	},
	"no_attacking_win": {
		"True mastery needs no bloodshed.",
		"Win your duel without sending a single attacker,",
		"and the village will sing of your restraint.",
	},
}

// questFlavorLines returns a fresh copy of the Wiseman's flavor intro for a
// deck-changing quest, falling back to the quest title when the template has no
// bespoke flavor. A copy is returned so callers may append without mutating the
// shared questFlavor map.
func questFlavorLines(q *domain.Quest) []string {
	if lines, ok := questFlavor[q.ID]; ok {
		return append([]string(nil), lines...)
	}
	if q.Title != "" {
		return []string{q.Title + "."}
	}
	return []string{"I have a task for you, planeswalker."}
}

func (s *WisemanScreen) findRandomCity() *domain.City {
	var cities []*domain.City
	w, h := s.Level.Size()
	for y := range h {
		for x := range w {
			tile := s.Level.Tile(image.Point{X: x, Y: y})
			if tile != nil && tile.IsCity && tile.City.Name != s.City.Name {
				cities = append(cities, &tile.City)
			}
		}
	}
	if len(cities) == 0 {
		return nil
	}
	return cities[rand.Intn(len(cities))]
}

// --- UI Setup ---

func (s *WisemanScreen) setupButtons() {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}

	acceptW, _ := elements.TextButtonSize("Accept", fontFace)
	refuseW, _ := elements.TextButtonSize("Refuse", fontFace)
	totalW := acceptW + 10 + refuseW
	startX := 512 - totalW/2

	yesBtn := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "Accept",
		Font:    fontFace,
		ID:      "yes",
		X:       startX,
		Y:       618,
	})

	noBtn := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "Refuse",
		Font:    fontFace,
		ID:      "no",
		X:       startX + acceptW + 10,
		Y:       618,
	})

	s.Buttons = []*elements.Button{yesBtn, noBtn}
}

// --- Update / Draw ---

func (s *WisemanScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	switch s.State {
	case WisemanStateOffer:
		return s.updateOffer(W, H, scale)
	default:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return screenui.CityScr, nil, nil
		}
	}

	return screenui.WisemanScr, nil, nil
}

func (s *WisemanScreen) updateOffer(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	opts := &ebiten.DrawImageOptions{}
	for _, b := range s.Buttons {
		b.Update(opts, scale, W, H)
		if b.IsClicked() {
			if b.ID == "yes" {
				s.acceptQuest()
			}
			return screenui.CityScr, nil, nil
		}
	}
	return screenui.WisemanScr, nil, nil
}

func (s *WisemanScreen) acceptQuest() {
	if !s.Player.AddQuest(s.ProposedQuest) {
		return
	}
	switch s.ProposedQuest.Type {
	case domain.QuestTypeDefeatEnemy:
		s.spawnQuestEnemy(s.ProposedQuest.EnemyName)
		s.consumeCityQuestBoon()
	case domain.QuestTypeDelivery:
		s.consumeCityQuestBoon()
	}
}

// consumeCityQuestBoon marks this city's quest boon as spent once its
// delivery/defeat quest is taken, so the Wiseman won't re-offer it.
func (s *WisemanScreen) consumeCityQuestBoon() {
	s.City.BoonGranted = true
	s.City.ProposedQuest = nil
}

func (s *WisemanScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	bgOpts := &ebiten.DrawImageOptions{}
	bgW, bgH := s.BgImage.Bounds().Dx(), s.BgImage.Bounds().Dy()
	scaleX := float64(1024) / float64(bgW)
	scaleY := float64(768) / float64(bgH)
	bgOpts.GeoM.Scale(scaleX, scaleY)
	bgOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(s.BgImage, bgOpts)

	y := 50
	for _, line := range s.TextLines {
		txt := elements.NewText(24, line, 50, y)
		txt.Color = color.White
		txt.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		y += 35
	}

	if s.State == WisemanStateOffer {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(scale, scale)
		for _, b := range s.Buttons {
			b.Draw(screen, opts, scale)
		}
	}
}

// --- Boons ---

func (s *WisemanScreen) grantBoon() {
	s.State = WisemanStateStory

	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.SFXReward)
	}

	switch s.City.WisemanBoon {
	case domain.BoonBonusLife:
		s.giveBonusLife()
	case domain.BoonEnemyDeckInfo:
		s.giveEnemyDeckInfo()
	case domain.BoonWorldMagicLocation:
		s.giveWorldMagicLocation()
	case domain.BoonBonusCard:
		s.giveBonusCard()
	default:
		s.loadStory()
		return
	}

	s.City.BoonGranted = true
}

func (s *WisemanScreen) giveBonusLife() {
	s.Player.BonusDuelLife += 2
	s.TextLines = []string{
		"I sense great danger ahead.",
		"Take this blessing with you.",
		"",
		"You will start your next duel",
		"with +2 life.",
	}
}

func (s *WisemanScreen) giveEnemyDeckInfo() {
	var names []string
	for name, char := range domain.Rogues {
		if char.Level <= 10 {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		s.loadStory()
		return
	}

	name := names[rand.Intn(len(names))]
	rogue := domain.Rogues[name]

	s.TextLines = []string{
		fmt.Sprintf("I have studied the %s.", name),
		fmt.Sprintf("It fights with a %s deck.", rogue.PrimaryColor),
		"",
		"Prepare your cards accordingly.",
	}
}

func (s *WisemanScreen) findWorldMagicCity() *domain.City {
	var cities []*domain.City
	w, h := s.Level.Size()
	for y := range h {
		for x := range w {
			tile := s.Level.Tile(image.Point{X: x, Y: y})
			if tile != nil && tile.IsCity && tile.City.HasWorldMagic() {
				cities = append(cities, &tile.City)
			}
		}
	}
	if len(cities) == 0 {
		return nil
	}
	return cities[rand.Intn(len(cities))]
}

func (s *WisemanScreen) giveWorldMagicLocation() {
	city := s.findWorldMagicCity()
	if city == nil {
		s.loadStory()
		return
	}

	magic := city.GetWorldMagic()
	s.TextLines = []string{
		"I have heard rumors of a",
		"powerful artifact...",
		"",
		fmt.Sprintf("The %s can be found", magic.Name),
		fmt.Sprintf("in %s.", city.Name),
	}
}

func (s *WisemanScreen) giveBonusCard() {
	if len(domain.CARDS) == 0 {
		s.loadStory()
		return
	}

	card := domain.CARDS[rand.Intn(len(domain.CARDS))]
	s.Player.BonusDuelCards = append(s.Player.BonusDuelCards, card)
	s.TextLines = []string{
		"Take this card, planeswalker.",
		"It may aid you in your",
		"next battle.",
		"",
		fmt.Sprintf("Received %s for your", card.CardName),
		"next duel.",
	}
}

// --- Story text ---

func loadStories() []string {
	content := string(assets.Advblocks_txt)
	var stories []string

	parts := strings.SplitSeq(content, "STARTBLOCK")
	for part := range parts {
		before, _, ok := strings.Cut(part, "ENDBLOCK")
		if ok {
			story := strings.TrimSpace(before)
			if story != "" {
				stories = append(stories, story)
			}
		}
	}
	return stories
}

func (s *WisemanScreen) loadStory() {
	s.State = WisemanStateStory
	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.SFXScroll)
	}
	stories := loadStories()
	story := "No stories found."
	if len(stories) > 0 {
		story = stories[rand.Intn(len(stories))]
	}

	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   24,
	}
	pages := paginateText(story, fontFace, 290, 768)
	if len(pages) > 0 {
		s.TextLines = strings.Split(pages[0], "\n")
	} else {
		s.TextLines = []string{"I have no quests for you today.", "Travel safely."}
	}
}

func paginateText(textStr string, face *text.GoTextFace, maxWidth, maxHeight float64) []string {
	var pages []string

	words := strings.Fields(textStr)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		newLine := currentLine + " " + word
		w, _ := text.Measure(newLine, face, 30.0)
		if w > maxWidth-40 {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = newLine
		}
	}
	lines = append(lines, currentLine)

	var currentPageLines []string
	currentHeight := 0.0
	lineHeight := 30.0

	for _, line := range lines {
		if currentHeight+lineHeight > maxHeight-40 {
			if len(currentPageLines) > 0 {
				currentPageLines[len(currentPageLines)-1] += "..."
			}

			pages = append(pages, strings.Join(currentPageLines, "\n"))
			currentPageLines = []string{line}
			currentHeight = lineHeight
		} else {
			currentPageLines = append(currentPageLines, line)
			currentHeight += lineHeight
		}
	}
	if len(currentPageLines) > 0 {
		pages = append(pages, strings.Join(currentPageLines, "\n"))
	}

	return pages
}
