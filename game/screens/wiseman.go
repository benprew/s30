package screens

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"strings"

	"github.com/benprew/s30/assets"
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

	s.determineState()
	return s
}

// determineState sets the wiseman screen state based on quest progress.
func (s *WisemanScreen) determineState() {
	if s.Player.ActiveQuest != nil {
		s.handleActiveQuest()
		return
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

	if rand.Float32() < .25 {
		s.generateQuest()
	} else {
		s.loadStory()
	}
}

func (s *WisemanScreen) handleActiveQuest() {
	q := s.Player.ActiveQuest

	if q.DaysRemaining <= 0 {
		s.handleExpiredQuest(q)
		return
	}

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
		s.Player.ActiveQuest = nil
		s.State = WisemanStateBanned
		s.TextLines = banText
		return
	}

	if q.OriginCity != nil {
		q.OriginCity.QuestBanDays = 20
	}
	s.Player.ActiveQuest = nil

	if s.City.QuestBanDays > 0 {
		s.State = WisemanStateBanned
		s.TextLines = banText
		return
	}

	if rand.Float32() < .25 {
		s.generateQuest()
	} else {
		s.loadStory()
	}
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

	if rand.Float32() < 0.5 {
		s.generateDeliveryQuest()
	} else {
		s.generateDefeatEnemyQuest()
	}

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

	switch q.RewardType {
	case domain.RewardManaLink:
		s.Player.Life++
		s.City.IsManaLinked = true
	case domain.RewardAmulet:
		count := 1 + rand.Intn(3)
		for i := 0; i < count; i++ {
			s.Player.AddAmulet(domain.NewAmulet(q.AmuletColor))
		}
	case domain.RewardCard:
		if len(domain.CARDS) > 0 {
			card := domain.CARDS[rand.Intn(len(domain.CARDS))]
			s.Player.CardCollection.AddCardToDeck(card, 0, 1)
		}
	}
}

// --- UI Setup ---

func (s *WisemanScreen) setupButtons() {
	Icons, err := imageutil.LoadSpriteSheet(12, 2, assets.Icons_png)
	if err != nil {
		panic(err)
	}
	Iconb, err := imageutil.LoadSpriteSheet(8, 1, assets.Iconb_png)
	if err != nil {
		panic(err)
	}

	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}

	yesBtn := mkWisemanButton("Accept", 0, 50, -50, fontFace, Icons, Iconb)
	yesBtn.ID = "yes"

	noBtn := mkWisemanButton("Refuse", 1, 150, -50, fontFace, Icons, Iconb)
	noBtn.ID = "no"

	s.Buttons = []*elements.Button{yesBtn, noBtn}
}

func mkWisemanButton(txt string, index int, x, y int, fontFace *text.GoTextFace, Icons, Iconb [][]*ebiten.Image) *elements.Button {
	scale := 1.2
	norm := elements.CombineButton(Iconb[0][0], Icons[0][index], Iconb[0][1], scale)
	hover := elements.CombineButton(Iconb[0][2], Icons[0][index], Iconb[0][3], scale)
	pressed := elements.CombineButton(Iconb[0][0], Icons[0][index], Iconb[0][1], scale)

	btn := elements.NewButton(norm, hover, pressed, x, y, 1.0)
	btn.ButtonText = elements.ButtonText{
		Text:       txt,
		Font:       fontFace,
		TextColor:  color.White,
		TextOffset: image.Point{X: 10, Y: 40},
		VAlign:     elements.AlignBottom,
	}
	return btn
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
		btnX := W/2 - 100
		if b.ID == "no" {
			btnX += 150
		}
		b.MoveTo(btnX, H-150)

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
