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
	WisemanStateActive
	WisemanStateReward
	WisemanStateBanned
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

// determineState sets the wiseman screen state based on quest progress.
func (s *WisemanScreen) determineState() {
	if s.Player.ActiveQuest != nil {
		q := s.Player.ActiveQuest
		if q.DaysRemaining <= 0 {
			s.handleExpiredQuest(q)
			return
		}
		if s.isQuestRelevantCity(q) {
			s.handleActiveQuest()
			return
		}
	}

	if s.City.QuestBanDays > 0 {
		s.State = WisemanStateBanned
		s.TextLines = []string{
			"You failed to complete your",
			"last quest for this village.",
			"",
			"The people have no new quest for you.",
		}
		return
	}

	if s.City.BoonGranted {
		s.loadStory()
		return
	}

	if s.City.WisemanBoon == domain.BoonQuest && s.Player.ActiveQuest != nil {
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

func (s *WisemanScreen) isQuestRelevantCity(q *domain.Quest) bool {
	switch q.Type {
	case domain.QuestTypeDelivery:
		return q.OriginCity.Name == s.City.Name || q.TargetCity.Name == s.City.Name
	case domain.QuestTypeDefeatEnemy:
		return q.OriginCity.Name == s.City.Name
	}
	return false
}

func (s *WisemanScreen) handleActiveQuest() {
	q := s.Player.ActiveQuest

	if s.isQuestComplete(q) {
		s.State = WisemanStateReward
		s.TextLines = []string{"You have completed the quest!", "Here is your reward."}
		return
	}

	s.State = WisemanStateActive
	s.prepareActiveText()
}

func (s *WisemanScreen) handleExpiredQuest(q *domain.Quest) {
	banText := []string{
		"You failed to complete your",
		"last quest for this village.",
		"",
		"The people have no new quest for you.",
	}
	if q.OriginCity.Name == s.City.Name {
		s.City.QuestBanDays = 20
		s.City.WisemanBoon = domain.BoonNone
		s.City.BoonGranted = false
		s.City.ProposedQuest = nil
		s.Player.ActiveQuest = nil
		s.State = WisemanStateBanned
		s.TextLines = banText
		return
	}

	if q.OriginCity != nil {
		q.OriginCity.QuestBanDays = 20
		q.OriginCity.WisemanBoon = domain.BoonNone
		q.OriginCity.BoonGranted = false
		q.OriginCity.ProposedQuest = nil
	}
	s.Player.ActiveQuest = nil

	if s.City.QuestBanDays > 0 {
		s.State = WisemanStateBanned
		s.TextLines = banText
		return
	}

	s.loadStory()
}

func (s *WisemanScreen) isQuestComplete(q *domain.Quest) bool {
	switch q.Type {
	case domain.QuestTypeDelivery:
		return q.TargetCity.Name == s.City.Name
	case domain.QuestTypeDefeatEnemy:
		return q.OriginCity.Name == s.City.Name && q.IsCompleted
	}
	return false
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

	rewardType := s.randomRewardType()
	s.ProposedQuest = &domain.Quest{
		Type:          domain.QuestTypeDelivery,
		TargetCity:    targetCity,
		OriginCity:    s.City,
		DaysRemaining: 20 + rand.Intn(20),
		RewardType:    rewardType,
		AmuletColor:   targetCity.AmuletColor,
	}

	rewardText := rewardDescription(rewardType, targetCity.AmuletColor)
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
	enemyName := s.randomRogueName()

	rewardType := domain.RewardAmulet
	if rand.Float32() < 0.3 {
		rewardType = domain.RewardManaLink
	}

	s.ProposedQuest = &domain.Quest{
		Type:          domain.QuestTypeDefeatEnemy,
		EnemyName:     enemyName,
		OriginCity:    s.City,
		DaysRemaining: 15 + rand.Intn(15),
		RewardType:    rewardType,
		AmuletColor:   s.City.AmuletColor,
	}

	rewardText := rewardDescription(rewardType, s.City.AmuletColor)
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

func (s *WisemanScreen) randomRogueName() string {
	var names []string
	for name, char := range domain.Rogues {
		if char.Level <= 10 {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "Goblin Lord"
	}
	return names[rand.Intn(len(names))]
}

func (s *WisemanScreen) randomRewardType() domain.RewardType {
	r := rand.Float32()
	if r < 0.3 {
		return domain.RewardManaLink
	}
	if r < 0.6 {
		return domain.RewardCard
	}
	return domain.RewardAmulet
}

func rewardDescription(rt domain.RewardType, amuletColor domain.ColorMask) string {
	switch rt {
	case domain.RewardManaLink:
		return "a mana link"
	case domain.RewardCard:
		return "a card"
	case domain.RewardAmulet:
		return fmt.Sprintf("a %s amulet", domain.ColorMaskToString(amuletColor))
	}
	return "a reward"
}

func (s *WisemanScreen) prepareQuestOfferText(q *domain.Quest) {
	switch q.Type {
	case domain.QuestTypeDelivery:
		rewardText := rewardDescription(q.RewardType, q.AmuletColor)
		s.TextLines = []string{
			fmt.Sprintf("We need a message delivered to %s.", q.TargetCity.Name),
			fmt.Sprintf("You will be rewarded with %s.", rewardText),
			"",
			"Accept the Quest?",
		}
	case domain.QuestTypeDefeatEnemy:
		rewardText := rewardDescription(q.RewardType, q.AmuletColor)
		s.TextLines = []string{
			fmt.Sprintf("A %s threatens our village.", q.EnemyName),
			fmt.Sprintf("Slay it and earn %s.", rewardText),
			"",
			"Accept the Quest?",
		}
	}
}

func (s *WisemanScreen) prepareActiveText() {
	q := s.Player.ActiveQuest
	switch q.Type {
	case domain.QuestTypeDelivery:
		s.TextLines = []string{
			fmt.Sprintf("Please deliver the message to %s.", q.TargetCity.Name),
			fmt.Sprintf("You have %d days remaining.", q.DaysRemaining),
		}
	case domain.QuestTypeDefeatEnemy:
		s.TextLines = []string{
			fmt.Sprintf("Defeat the %s!", q.EnemyName),
			fmt.Sprintf("You have %d days remaining.", q.DaysRemaining),
		}
	}
}

func (s *WisemanScreen) findRandomCity() *domain.City {
	var cities []*domain.City
	w, h := s.Level.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
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

// --- Reward ---

func (s *WisemanScreen) giveReward() {
	q := s.Player.ActiveQuest
	if q == nil {
		return
	}

	am := gameaudio.Get()
	switch q.RewardType {
	case domain.RewardManaLink:
		s.Player.Life++
		s.City.IsManaLinked = true
		if am != nil {
			am.PlaySFX(gameaudio.SFXManalink)
		}
	case domain.RewardAmulet:
		count := 1 + rand.Intn(3)
		for i := 0; i < count; i++ {
			s.Player.AddAmulet(domain.NewAmulet(q.AmuletColor))
		}
		if am != nil {
			am.PlaySFX(gameaudio.SFXManaball)
		}
	case domain.RewardCard:
		if len(domain.CARDS) > 0 {
			card := domain.CARDS[rand.Intn(len(domain.CARDS))]
			s.Player.CardCollection.AddCardToDeck(card, 0, 1)
		}
		if am != nil {
			am.PlaySFX(gameaudio.SFXFindCard)
		}
	}

	q.OriginCity.BoonGranted = true
	q.OriginCity.ProposedQuest = nil
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
	case WisemanStateReward:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			s.giveReward()
			s.Player.ActiveQuest = nil
			return screenui.CityScr, nil, nil
		}
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
	s.Player.ActiveQuest = s.ProposedQuest
	if s.ProposedQuest.Type == domain.QuestTypeDefeatEnemy {
		s.spawnQuestEnemy(s.ProposedQuest.EnemyName)
	}
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
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
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

	parts := strings.Split(content, "STARTBLOCK")
	for _, part := range parts {
		end := strings.Index(part, "ENDBLOCK")
		if end != -1 {
			story := strings.TrimSpace(part[:end])
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
