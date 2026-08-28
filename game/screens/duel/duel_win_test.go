package duel

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
)

func TestHandleWinRegularEnemyDoesNotResolvePendingCastle(t *testing.T) {
	player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}}
	castle := &domain.Castle{Name: "Castle", Color: domain.ColorRed, MapTile: image.Point{}}
	lvl := &world.Level{
		W: 1, H: 1, Player: player,
		Tiles: [][]*world.Tile{{{IsCastle: true, Castle: castle}}},
		Enemies: []domain.Enemy{{Character: &domain.Character{
			Name: "Regular Enemy", CardCollection: domain.NewCardCollection(),
		}}},
	}
	lvl.SetPendingCastle(castle, castle.MapTile)
	s := &DuelScreen{player: player, enemy: lvl.GetEnemyAt(0), lvl: lvl, idx: 0}

	if _, _, err := s.handleWin(); err != nil {
		t.Fatal(err)
	}
	if castle.Defeated {
		t.Fatal("regular overworld win defeated the pending castle")
	}
	if lvl.CombatsWon != 1 {
		t.Fatalf("overworld win recorded %d combat wins, want 1", lvl.CombatsWon)
	}
}

func TestHandleLossDoesNotRecordCombatWin(t *testing.T) {
	player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}}
	lvl := &world.Level{Enemies: []domain.Enemy{{Character: &domain.Character{
		Name: "Regular Enemy", CardCollection: domain.NewCardCollection(),
	}}}}
	s := &DuelScreen{player: player, enemy: lvl.GetEnemyAt(0), lvl: lvl, idx: 0}

	if _, _, err := s.handleLoss(); err != nil {
		t.Fatal(err)
	}
	if lvl.CombatsWon != 0 {
		t.Fatalf("loss recorded %d combat wins", lvl.CombatsWon)
	}
}

func TestHandleDungeonWinRecordsCombatWin(t *testing.T) {
	for _, test := range []struct {
		name string
		boss bool
		want int
	}{
		{name: "random monster", want: 1},
		{name: "boss", boss: true, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}}
			enemy := &domain.Enemy{Character: &domain.Character{
				Name: "Dungeon Enemy", CardCollection: domain.NewCardCollection(),
			}}
			lvl := &world.Level{}
			tile := &domain.DungeonTile{Type: domain.DungeonTileEnemy, Enemy: enemy.Character, Boss: test.boss}
			state := &domain.DungeonState{}
			s := &DuelScreen{
				player: player,
				enemy:  enemy,
				lvl:    lvl,
				dungeon: &dungeonDuelContext{
					state: state,
					tile:  tile,
				},
			}

			if _, _, err := s.handleWin(); err != nil {
				t.Fatal(err)
			}
			if lvl.CombatsWon != test.want {
				t.Fatalf("dungeon win recorded %d combat wins, want %d", lvl.CombatsWon, test.want)
			}
		})
	}
}

func TestFinalBossOutcomeUsesGameResultScreens(t *testing.T) {
	player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}}
	enemy := &domain.Enemy{Character: &domain.Character{Name: "Arzakon", CardCollection: domain.NewCardCollection()}}

	winDuel := &DuelScreen{player: player, enemy: enemy, finalBoss: true}
	name, screen, err := winDuel.handleWin()
	if err != nil {
		t.Fatal(err)
	}
	if name != screenui.GameWinScr {
		t.Fatalf("final boss win returned %v, want GameWinScr", name)
	}
	result, ok := screen.(*GameResultScreen)
	if !ok || !result.Won {
		t.Fatalf("final boss win returned %#v, want winning GameResultScreen", screen)
	}

	loseDuel := &DuelScreen{player: player, enemy: enemy, finalBoss: true}
	name, screen, err = loseDuel.handleLoss()
	if err != nil {
		t.Fatal(err)
	}
	if name != screenui.GameLoseScr {
		t.Fatalf("final boss loss returned %v, want GameLoseScr", name)
	}
	result, ok = screen.(*GameResultScreen)
	if !ok || result.Won {
		t.Fatalf("final boss loss returned %#v, want losing GameResultScreen", screen)
	}
}

func TestArzakonIsLevelTwelveWithThreeHundredLife(t *testing.T) {
	arzakon := domain.Rogues[FinalBossName]
	if arzakon == nil {
		t.Fatalf("rogue %q is not configured", FinalBossName)
		return
	}
	if arzakon.Level != 12 || arzakon.Life != 300 {
		t.Fatalf("Arzakon level/life = %d/%d, want 12/300", arzakon.Level, arzakon.Life)
	}
}

func TestFifthWizardRewardContinuesToArzakon(t *testing.T) {
	player := &domain.Player{
		Character:    domain.Character{CardCollection: domain.NewCardCollection()},
		DungeonState: &domain.DungeonState{DungeonLife: 20},
	}
	finalCastle := &domain.Castle{Name: "Final Castle", Color: domain.ColorRed, MapTile: image.Point{}}
	lvl := &world.Level{
		W:      1,
		H:      1,
		Player: player,
		Tiles:  [][]*world.Tile{{{IsCastle: true, Castle: finalCastle}}},
		Castles: []*domain.Castle{
			{Defeated: true},
			{Defeated: true},
			{Defeated: true},
			{Defeated: true},
			finalCastle,
		},
	}
	lvl.SetPendingCastle(finalCastle, finalCastle.MapTile)
	wizard := &domain.Enemy{Character: &domain.Character{
		Name:           "Mighty Wizard",
		CardCollection: domain.NewCardCollection(),
	}}
	tile := &domain.DungeonTile{Type: domain.DungeonTileEnemy, Enemy: wizard.Character, Boss: true}
	duel := &DuelScreen{
		player:  player,
		enemy:   wizard,
		lvl:     lvl,
		dungeon: &dungeonDuelContext{state: player.DungeonState, tile: tile},
	}

	name, screen, err := duel.handleWin()
	if err != nil {
		t.Fatal(err)
	}
	if name != screenui.DuelWinScr {
		t.Fatalf("fifth wizard win returned %v, want reward screen", name)
	}
	reward, ok := screen.(*DuelWinScreen)
	if !ok {
		t.Fatalf("fifth wizard win returned %T, want DuelWinScreen", screen)
	}
	if len(reward.cards) == 0 {
		t.Fatal("fifth wizard reward has no cards")
	}
	if reward.ReturnScr != screenui.DuelScr {
		t.Fatalf("reward continues to %v, want DuelScr", reward.ReturnScr)
	}
	finalDuel, ok := reward.ReturnScreen.(*DuelScreen)
	if !ok || !finalDuel.finalBoss || finalDuel.enemy.Name() != FinalBossName {
		t.Fatalf("reward continuation = %#v, want Arzakon duel", reward.ReturnScreen)
	}
}

func TestDuelWin_AllCardsAwarded(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	bolt := domain.FindCardByName("Lightning Bolt")
	forest := domain.FindCardByName("Forest")

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	reward := domain.DuelReward{
		Cards: []*domain.Card{mountain, bolt, forest},
		Gold:  50,
	}

	s := NewWinDuelScreen(player, reward, nil)

	if len(s.cards) != 3 {
		t.Fatalf("expected 3 cards on win screen, got %d", len(s.cards))
	}
	if s.reward.Gold != 50 {
		t.Fatalf("expected 50 gold in reward, got %d", s.reward.Gold)
	}

	// Done button or click dismisses screen to ReturnScr
	nextScr, _, err := s.Update(1024, 768, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if nextScr != screenui.DuelWinScr {
		t.Fatalf("expected DuelWinScr without click, got %v", nextScr)
	}
}

func TestDuelWinScreen_DynamicLayoutScaling(t *testing.T) {
	c1 := domain.FindCardByName("Mountain")
	c2 := domain.FindCardByName("Island")
	c3 := domain.FindCardByName("Plains")
	c4 := domain.FindCardByName("Swamp")
	c5 := domain.FindCardByName("Forest")
	c6 := domain.FindCardByName("Badlands")

	cards := []*domain.Card{c1, c2, c3, c4, c5, c6}
	layout := layoutCards(cards)
	if len(layout) != 6 {
		t.Fatalf("expected 6 cards in layout, got %d", len(layout))
	}

	// Ensure all cards fit within 0..1024 bounds
	for i, wc := range layout {
		if wc.scale >= 1.0 {
			t.Errorf("card %d scale = %f, expected scale < 1.0 for 6 cards", i, wc.scale)
		}
		if wc.rect.Min.X < 0 || wc.rect.Max.X > winLogicalW {
			t.Errorf("card %d rect %v out of screen bounds [0, %d]", i, wc.rect, winLogicalW)
		}
	}
}

func TestHandleWin_HigherTierEnemyRewards(t *testing.T) {
	player := &domain.Player{
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
		Amulets: make(map[domain.ColorMask]int),
	}

	bolt := domain.FindCardByName("Lightning Bolt")
	player.CardCollection.AddCardToDeck(bolt, 0, 4)

	enemyCollection := domain.NewCardCollection()
	shivanDragon := domain.FindCardByName("Shivan Dragon")
	enemyCollection.AddCardToDeck(shivanDragon, 0, 4)

	lvl := &world.Level{Player: player}
	lvl.Enemies = []domain.Enemy{
		{Character: &domain.Character{Name: "Dragon Lord", Level: 11, PrimaryColor: "Red", ColorIdentity: []string{"Red"}, CardCollection: enemyCollection}},
	}
	enemy := lvl.GetEnemyAt(0)

	s := &DuelScreen{
		player:        player,
		enemy:         enemy,
		lvl:           lvl,
		idx:           0,
		enemyAnteCard: shivanDragon,
	}

	startGold := player.Gold
	startAmuletRed := player.Amulets[domain.ColorRed]

	_, screen, err := s.handleWin()
	if err != nil {
		t.Fatalf("handleWin error: %v", err)
	}

	winScreen, ok := screen.(*DuelWinScreen)
	if !ok {
		t.Fatalf("expected DuelWinScreen, got %T", screen)
	}

	// Tier 11 enemy: ante card + 3 high cards + 2 lands = 6 cards total
	if len(winScreen.cards) != 6 {
		t.Errorf("expected 6 reward cards for tier 11 enemy, got %d", len(winScreen.cards))
	}

	// Player collection should have all awarded cards
	if player.CardCollection[shivanDragon] == nil || player.CardCollection[shivanDragon].Count != 1 {
		t.Errorf("ante card was not added to player collection")
	}

	// Gold should be increased (tier 10/11 grants 150+ gold)
	if player.Gold < startGold+150 {
		t.Errorf("player gold = %d, expected at least %d", player.Gold, startGold+150)
	}

	// Amulets should be increased if awarded in reward bundle
	if len(winScreen.reward.Amulets) > 0 && player.Amulets[domain.ColorRed] < startAmuletRed+1 {
		t.Errorf("player red amulets = %d, expected at least %d", player.Amulets[domain.ColorRed], startAmuletRed+1)
	}
}

func TestHandleWin_RewardsFromCorrectEnemy(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	lightningBolt := domain.FindCardByName("Lightning Bolt")

	playerDeck := make(domain.Deck)
	playerDeck[mountain] = 4
	playerDeck[lightningBolt] = 4

	playerCollection := domain.NewCardCollection()
	for card, count := range playerDeck {
		playerCollection.AddCardToDeck(card, 0, count)
	}

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: playerCollection,
		},
		Amulets: make(map[domain.ColorMask]int),
	}

	forest := domain.FindCardByName("Forest")
	giantGrowth := domain.FindCardByName("Giant Growth")

	enemy0Deck := make(domain.Deck)
	enemy0Deck[forest] = 4
	enemy0Deck[giantGrowth] = 4

	enemy0Collection := domain.NewCardCollection()
	for card, count := range enemy0Deck {
		enemy0Collection.AddCardToDeck(card, 0, count)
	}

	island := domain.FindCardByName("Island")
	counterspell := domain.FindCardByName("Counterspell")

	enemy1Deck := make(domain.Deck)
	enemy1Deck[island] = 4
	enemy1Deck[counterspell] = 4

	enemy1Collection := domain.NewCardCollection()
	for card, count := range enemy1Deck {
		enemy1Collection.AddCardToDeck(card, 0, count)
	}

	lvl := &world.Level{
		Player: player,
	}

	lvl.Enemies = []domain.Enemy{
		{Character: &domain.Character{Name: "Green Mage", Level: 1, PrimaryColor: "Green", CardCollection: enemy0Collection}},
		{Character: &domain.Character{Name: "Blue Mage", Level: 1, PrimaryColor: "Blue", CardCollection: enemy1Collection}},
	}

	enemy := lvl.GetEnemyAt(0)

	s := &DuelScreen{
		player:        player,
		enemy:         enemy,
		lvl:           lvl,
		idx:           0,
		anteCard:      mountain,
		enemyAnteCard: giantGrowth,
	}

	_, screen, err := s.handleWin()
	if err != nil {
		t.Fatalf("handleWin returned error: %v", err)
	}

	winScreen, ok := screen.(*DuelWinScreen)
	if !ok {
		t.Fatalf("expected DuelWinScreen, got %T", screen)
	}

	if len(winScreen.cards) != 3 {
		t.Fatalf("expected 3 reward cards, got %d", len(winScreen.cards))
	}
	if winScreen.cards[0].card.Name() != giantGrowth.Name() {
		t.Errorf("expected first card to be enemy ante card %q, got %q", giantGrowth.Name(), winScreen.cards[0].card.Name())
	}

	// All reward cards should be added to player's collection
	if _, exists := player.CardCollection[giantGrowth]; !exists {
		t.Error("enemy ante card was not added to player collection upon winning")
	}
}
