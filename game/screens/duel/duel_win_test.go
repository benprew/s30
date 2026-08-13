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
	if len(reward.choices) == 0 {
		t.Fatal("fifth wizard reward has no card choices")
	}
	if reward.ReturnScr != screenui.DuelScr {
		t.Fatalf("reward continues to %v, want DuelScr", reward.ReturnScr)
	}
	finalDuel, ok := reward.ReturnScreen.(*DuelScreen)
	if !ok || !finalDuel.finalBoss || finalDuel.enemy.Name() != FinalBossName {
		t.Fatalf("reward continuation = %#v, want Arzakon duel", reward.ReturnScreen)
	}
}

func TestDuelWin_RequiresConfirmation(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	bolt := domain.FindCardByName("Lightning Bolt")
	forest := domain.FindCardByName("Forest")

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	s := NewWinDuelScreen(player, []*domain.Card{mountain, bolt, forest}, nil)

	if s.selected != -1 {
		t.Fatalf("expected no card selected initially, got %d", s.selected)
	}

	// Selecting a card alone must not add it to the collection.
	s.selectCardAt(s.choices[1].rect.Min)
	if s.selected != 1 {
		t.Fatalf("expected selected index 1, got %d", s.selected)
	}
	if _, exists := player.CardCollection[bolt]; exists {
		t.Fatal("card added to collection before confirmation")
	}

	// Confirming with nothing selected is a no-op.
	noSel := NewWinDuelScreen(player, []*domain.Card{mountain}, nil)
	if noSel.confirmSelection() {
		t.Fatal("confirmSelection returned true with no card selected")
	}

	// Confirming adds the selected card.
	if !s.confirmSelection() {
		t.Fatal("confirmSelection returned false with a card selected")
	}
	if item := player.CardCollection[bolt]; item == nil || item.Count != 1 {
		t.Fatal("confirmed card was not added to collection")
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
		{Character: &domain.Character{Name: "Green Mage", Level: 1, CardCollection: enemy0Collection}},
		{Character: &domain.Character{Name: "Blue Mage", Level: 1, CardCollection: enemy1Collection}},
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

	if len(winScreen.choices) != 3 {
		t.Fatalf("expected 3 reward choices, got %d", len(winScreen.choices))
	}
	if winScreen.choices[0].card.Name() != giantGrowth.Name() {
		t.Errorf("expected first choice to be enemy ante card %q, got %q", giantGrowth.Name(), winScreen.choices[0].card.Name())
	}

	// The reward is a choice, so nothing is added to the collection until the
	// player picks one.
	if _, exists := player.CardCollection[giantGrowth]; exists {
		t.Error("enemy ante card was added to collection before the player chose it")
	}
}
