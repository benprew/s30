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
		if size < run.Player.MinDeckSize {
			t.Fatalf("opponent deck too small: %d", size)
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
