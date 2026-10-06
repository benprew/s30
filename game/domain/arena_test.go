package domain

import (
	"math/rand"
	"testing"
)

func TestArenaEntryAndPoolIsolation(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	campaign := &Player{Character: Character{Life: 13, CardCollection: NewCardCollection()}, Gold: 299, MinDeckSize: 36, ActiveDeck: 2, BonusDuelLife: 5}
	campaign.CardCollection.AddCardToDeck(bolt, 2, 1)
	if _, err := NewArenaRun(campaign, rand.New(rand.NewSource(1))); err == nil || campaign.Gold != 299 {
		t.Fatal("entry must require 300 gold")
	}
	campaign.Gold = 300
	run, err := NewArenaRun(campaign, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if campaign.Gold != 0 || run.Round != 1 || run.Player.MinDeckSize != 15 || run.Player.Life != 13 {
		t.Fatal("incorrect entry state")
	}
	if run.Player.CardCollection.NumCards() != 114 || run.Player.BonusDuelLife != 0 || len(run.Player.ActiveQuests) != 0 {
		t.Fatal("arena must contain one pack and 100 lands without campaign bonuses")
	}
	for name := range basicLands {
		if run.Player.CardCollection.GetTotalCount(FindCardByName(name)) != 20 {
			t.Fatalf("expected 20 %s", name)
		}
	}
	if campaign.CardCollection.NumCards() != 1 || campaign.MinDeckSize != 36 || campaign.ActiveDeck != 2 || campaign.BonusDuelLife != 5 {
		t.Fatal("arena entry changed campaign deck or bonuses")
	}
}

func TestArenaBoosterDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	for range 100 {
		pack := ArenaBooster(rng)
		counts := map[string]int{}
		for _, card := range pack {
			counts[card.Rarity]++
			if IsBasicLand(card) || arenaAnteCard(card) {
				t.Fatalf("ineligible pack card %s", card.CardName)
			}
		}
		if len(pack) != 14 || counts["common"] != 10 || counts["uncommon"] != 3 || counts["rare"] != 1 {
			t.Fatalf("pack distribution = %v", counts)
		}
	}
}

func TestArenaRoundsRewardsAndChampion(t *testing.T) {
	campaign := &Player{Character: Character{Life: 10, CardCollection: NewCardCollection()}, Gold: 300}
	run, err := NewArenaRun(campaign, rand.New(rand.NewSource(9)))
	if err != nil {
		t.Fatal(err)
	}
	gold := []int{50, 50, 25, 25, 50, 100, 100}
	cardCounts := []int{1, 1, 2, 2, 2, 2, 3}
	amulets := []int{0, 1, 1, 1, 1, 1, 2}
	tiers := [][]CardTier{{TierD, TierF}, {TierD, TierF}, {TierD, TierF}, {TierC, TierD}, {TierB, TierC}, {TierA, TierB}, {TierS, TierA, TierB}}
	for round := 1; round <= 7; round++ {
		if run.Round != round || run.Player.MinDeckSize != 10+5*round {
			t.Fatalf("incorrect round %d state", round)
		}
		enemy, opponentErr := run.NewOpponent()
		if opponentErr != nil {
			t.Fatal(opponentErr)
		}
		if enemy.Character.Level != round {
			t.Fatalf("opponent level %d, want %d", enemy.Character.Level, round)
		}
		size := 0
		for _, n := range enemy.Character.GetActiveDeck() {
			size += n
		}
		if size != run.Player.MinDeckSize {
			t.Fatalf("opponent deck size = %d, want %d", size, run.Player.MinDeckSize)
		}
		reward := run.Win()
		if reward.Gold != gold[round-1] || len(reward.Cards) != cardCounts[round-1] || len(reward.Amulets) != amulets[round-1] {
			t.Fatalf("round %d reward = %+v", round, reward)
		}
		for _, card := range reward.Cards {
			tier, ok := CardTierForName(card.CardName)
			allowed := false
			for _, want := range tiers[round-1] {
				allowed = allowed || tier == want
			}
			if !ok || !allowed {
				t.Fatalf("round %d rewarded %s in tier %d", round, card.CardName, tier)
			}
		}
	}
	if !run.Champion || !run.Finished || campaign.Gold != 400 || campaign.CardCollection.NumCards() != 13 {
		t.Fatal("champion must retain only the seven round rewards")
	}
	if run.Player.CardCollection.NumCards() != 100+7*14 {
		t.Fatal("champion must not open an eighth pack")
	}
	if reward := run.Win(); reward.Gold != 0 || len(reward.Cards) != 0 {
		t.Fatal("finished arena granted another reward")
	}
}

func TestArenaFinishRetainsOnlyEarnedRewards(t *testing.T) {
	campaign := &Player{Character: Character{CardCollection: NewCardCollection()}, Gold: 400}
	run, err := NewArenaRun(campaign, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	run.Win()
	run.Finish()
	if !run.Finished || run.Champion || campaign.Gold != 150 || campaign.CardCollection.NumCards() != 1 {
		t.Fatal("leaving must preserve earned rewards without arena cards")
	}
	if reward := run.Win(); reward.Gold != 0 {
		t.Fatal("left arena granted a reward")
	}
}

func TestArenaChampionBonusDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	sTier := 0
	for range 2000 {
		reward := arenaReward(7, rng)
		if len(reward.Cards) != 3 {
			t.Fatal("champion reward must always contain three cards")
		}
		tier, _ := CardTierForName(reward.Cards[2].CardName)
		if tier == TierS {
			sTier++
		} else if tier != TierA && tier != TierB {
			t.Fatal("champion bonus must be S, A, or B")
		}
	}
	if sTier < 40 || sTier > 85 {
		t.Fatalf("S-tier rewards = %d of 2000, want approximately 3%%", sTier)
	}
}

func TestArenaOpponentUsesOnlyItsPoolAndBasicLandSupply(t *testing.T) {
	bears := FindCardByName("Grizzly Bears")
	bolt := FindCardByName("Lightning Bolt")
	pool := []*Card{bears, bears, bolt}
	collection := arenaOpponentCollection(pool, 45)
	if collection.GetTotalCount(bears) != 2 || collection.GetTotalCount(bolt) != 1 || collection.NumCards() != 45 {
		t.Fatal("opponent changed the opened cards or failed to meet the minimum")
	}
	for card, item := range collection {
		if IsBasicLand(card) {
			if item.Count > 20 {
				t.Fatalf("opponent used too many %s", card.CardName)
			}
		} else if card != bears && card != bolt {
			t.Fatalf("opponent used unopened card %s", card.CardName)
		}
	}
}

func TestArenaOpponentColorSelectionUsesCardTier(t *testing.T) {
	squire := FindCardByName("Squire")
	bolt := FindCardByName("Lightning Bolt")
	counterspell := FindCardByName("Counterspell")
	collection := arenaOpponentCollection([]*Card{squire, bolt, counterspell}, 15)
	if collection.GetDeckCount(bolt, 0) != 1 || collection.GetDeckCount(counterspell, 0) != 1 {
		t.Fatal("opponent must choose the colors with stronger cards")
	}
	if collection.GetDeckCount(squire, 0) != 0 {
		t.Fatal("a weaker creature must not displace the stronger spell colors")
	}
}

func TestArenaOpponentUnrankedCardsUseLowestTier(t *testing.T) {
	unranked := &Card{CardName: "Unranked arena spell", CardType: CardTypeInstant, ColorIdentity: []string{"W"}}
	bears := FindCardByName("Grizzly Bears")
	bolt := FindCardByName("Lightning Bolt")
	collection := arenaOpponentCollection([]*Card{unranked, bears, bolt}, 15)
	if collection.GetDeckCount(bears, 0) != 1 || collection.GetDeckCount(bolt, 0) != 1 {
		t.Fatal("opponent must prefer the creature and ranked spell to the unranked spell")
	}
	if collection.GetDeckCount(unranked, 0) != 0 {
		t.Fatal("an unranked card must not receive a top-tier score")
	}
}

func TestArenaOpponentCutsLowestScoringCards(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	goblin := FindCardByName("Mons's Goblin Raiders")
	chaoslace := FindCardByName("Chaoslace")
	gem := FindCardByName("Gem Bazaar")
	pool := []*Card{gem, chaoslace, chaoslace, chaoslace}
	for range 6 {
		pool = append(pool, goblin, bolt)
	}
	collection := arenaOpponentCollection(pool, 15)
	if collection.NumCards() != 15 {
		t.Fatalf("deck size = %d, want 15", collection.NumCards())
	}
	if collection.GetDeckCount(bolt, 0) != 6 || collection.GetDeckCount(goblin, 0) != 4 || collection.GetDeckCount(chaoslace, 0) != 0 {
		t.Fatal("deck must retain the strongest ten nonland cards")
	}
	if collection.GetDeckCount(gem, 0) != 1 || collection.GetDeckCount(FindCardByName("Mountain"), 0) != 4 {
		t.Fatal("opened lands must count toward the five land slots")
	}
}

func TestArenaOpponentCapsOpenedLands(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	gem := FindCardByName("Gem Bazaar")
	var pool []*Card
	for range 20 {
		pool = append(pool, gem, bolt)
	}
	collection := arenaOpponentCollection(pool, 15)
	if collection.NumCards() != 15 || collection.GetDeckCount(bolt, 0) != 10 || collection.GetDeckCount(gem, 0) != 5 {
		t.Fatal("excess opened lands and spells must not exceed the round deck size")
	}
}

func TestArenaOpponentLandCountUsesSelectedCurve(t *testing.T) {
	for _, tt := range []struct {
		name  string
		size  int
		cost  string
		lands int
	}{
		{"small aggressive", 15, "{R}", 5},
		{"small midrange", 15, "{2}{R}", 6},
		{"small expensive", 15, "{5}{R}", 8},
		{"large aggressive", 45, "{R}", 15},
		{"large midrange", 45, "{2}{R}", 19},
		{"large expensive", 45, "{5}{R}", 23},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := &Card{CardName: "Curve spell", CardType: CardTypeSorcery, ManaCost: tt.cost, ColorIdentity: []string{"R"}}
			var pool []*Card
			for range 60 {
				pool = append(pool, card)
			}
			collection := arenaOpponentCollection(pool, tt.size)
			if collection.NumCards() != tt.size || collection.GetDeckCount(card, 0) != tt.size-tt.lands {
				t.Fatalf("deck has %d cards and %d spells, want %d cards and %d spells", collection.NumCards(), collection.GetDeckCount(card, 0), tt.size, tt.size-tt.lands)
			}
		})
	}
}

func TestArenaOpponentCheapRampReducesLands(t *testing.T) {
	elves := FindCardByName("Llanowar Elves")
	bears := FindCardByName("Grizzly Bears")
	pool := []*Card{elves, elves, elves}
	for range 20 {
		pool = append(pool, bears)
	}
	collection := arenaOpponentCollection(pool, 15)
	if collection.NumCards() != 15 || collection.GetDeckCount(elves, 0) != 3 || collection.GetDeckCount(bears, 0) != 7 {
		t.Fatal("three cheap mana creatures must allow ten spells and five lands")
	}
}

func TestArenaOpponentCheapDrawReducesLands(t *testing.T) {
	recall := FindCardByName("Ancestral Recall")
	counterspell := FindCardByName("Counterspell")
	pool := []*Card{recall, recall, recall}
	for range 20 {
		pool = append(pool, counterspell)
	}
	collection := arenaOpponentCollection(pool, 15)
	if collection.NumCards() != 15 || collection.GetDeckCount(recall, 0) != 3 || collection.GetDeckCount(counterspell, 0) != 7 {
		t.Fatal("three cheap draw spells must allow ten spells and five lands")
	}
}

func TestArenaOpponentAggressiveLandFloor(t *testing.T) {
	elves := FindCardByName("Llanowar Elves")
	var pool []*Card
	for range 30 {
		pool = append(pool, elves)
	}
	collection := arenaOpponentCollection(pool, 30)
	if collection.NumCards() != 30 || collection.GetDeckCount(elves, 0) != 21 || collection.GetDeckCount(FindCardByName("Forest"), 0) != 9 {
		t.Fatal("cheap ramp must not reduce an aggressive deck below its land floor")
	}
}

func TestArenaOpponentExcludedCardsDoNotChangeLandCount(t *testing.T) {
	counterspell := FindCardByName("Counterspell")
	cantrip := &Card{CardName: "Weak cantrip", CardType: CardTypeSorcery, ManaCost: "{U}", ColorIdentity: []string{"U"}, Text: "Draw a card."}
	expensive := &Card{CardName: "Weak expensive spell", CardType: CardTypeSorcery, ManaCost: "{9}{U}", ColorIdentity: []string{"U"}}
	var pool []*Card
	for range 30 {
		pool = append(pool, counterspell, cantrip, expensive)
	}
	collection := arenaOpponentCollection(pool, 40)
	if collection.NumCards() != 40 || collection.GetDeckCount(counterspell, 0) != 24 || collection.GetDeckCount(cantrip, 0) != 0 || collection.GetDeckCount(expensive, 0) != 0 {
		t.Fatal("excluded cantrips and expensive spells must not affect the selected curve")
	}
}

func TestArenaCheapManaSupport(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"Ancestral Recall", true},
		{"Llanowar Elves", true},
		{"Wild Growth", true},
		{"Sol Ring", true},
		{"Fellwar Stone", true},
		{"Lightning Bolt", false},
		{"Jayemdae Tome", false},
		{"Howling Mine", false},
		{"Island", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := arenaCheapManaSupport(FindCardByName(tt.name)); got != tt.want {
				t.Fatalf("cheap mana support = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArenaDeckPaddingDoesNotChangePool(t *testing.T) {
	campaign := &Player{Character: Character{CardCollection: NewCardCollection()}, Gold: 300}
	run, err := NewArenaRun(campaign, rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatal(err)
	}
	for round := 1; round <= 7; round++ {
		before := run.Player.CardCollection.NumCards()
		deck := run.Player.GetDuelDeck()
		size := 0
		for card, n := range deck {
			if !IsBasicLand(card) {
				t.Fatal("empty deck was padded with a non-basic land")
			}
			size += n
		}
		if size != run.Player.MinDeckSize || run.Player.CardCollection.NumCards() != before || len(run.Player.GetActiveDeck()) != 0 {
			t.Fatal("padding changed the arena pool or submitted deck")
		}
		run.Win()
	}
}
