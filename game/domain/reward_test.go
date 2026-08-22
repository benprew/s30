package domain

import "testing"

func TestDeckPrimaryColor(t *testing.T) {
	mountain := FindCardByName("Mountain")
	lightningBolt := FindCardByName("Lightning Bolt")
	forest := FindCardByName("Forest")

	deck := Deck{
		mountain:      4,
		lightningBolt: 4,
		forest:        1,
	}

	if got := DeckPrimaryColor(deck); got != ColorRed {
		t.Errorf("DeckPrimaryColor = %v, want ColorRed", got)
	}
}

func TestDeckPrimaryColor_Colorless(t *testing.T) {
	if got := DeckPrimaryColor(Deck{}); got != ColorColorless {
		t.Errorf("DeckPrimaryColor of empty deck = %v, want ColorColorless", got)
	}
}

func TestRandomBasicLand(t *testing.T) {
	land := RandomBasicLand()
	if land == nil {
		t.Fatal("RandomBasicLand returned nil")
		return
	}
	if _, ok := basicLands[land.CardName]; !ok {
		t.Errorf("RandomBasicLand returned %q, which is not a basic land", land.CardName)
	}
}

func TestRewardChoices(t *testing.T) {
	mountain := FindCardByName("Mountain")
	lightningBolt := FindCardByName("Lightning Bolt")
	giantGrowth := FindCardByName("Giant Growth")

	deck := Deck{mountain: 4, lightningBolt: 4}

	choices := RewardChoices(deck, giantGrowth)

	if len(choices) != 3 {
		t.Fatalf("expected 3 choices, got %d", len(choices))
	}

	if choices[0] != giantGrowth {
		t.Errorf("first choice = %q, want the opponent ante card %q", choices[0].Name(), giantGrowth.Name())
	}

	// Second choice should be a card in the player's primary color (red).
	if !cardMatchesColorOrColorless(choices[1], ColorRed) {
		t.Errorf("second choice %q does not match player's red color", choices[1].Name())
	}

	// Third choice should be a basic land.
	if _, ok := basicLands[choices[2].CardName]; !ok {
		t.Errorf("third choice %q is not a basic land", choices[2].Name())
	}
}

func TestRewardChoices_NoOpponentCard(t *testing.T) {
	mountain := FindCardByName("Mountain")
	deck := Deck{mountain: 4}

	choices := RewardChoices(deck, nil)
	for _, c := range choices {
		if c == nil {
			t.Fatal("RewardChoices returned a nil card")
		}
	}
}

func TestGenerateDuelReward_TierScaling(t *testing.T) {
	mountain := FindCardByName("Mountain")
	lightningBolt := FindCardByName("Lightning Bolt")
	giantGrowth := FindCardByName("Giant Growth")

	deck := Deck{mountain: 4, lightningBolt: 4}

	tests := []struct {
		tier         int
		hasAnte      bool
		wantCards    int
		wantMinLands int
	}{
		{tier: 1, hasAnte: true, wantCards: 3, wantMinLands: 1},
		{tier: 1, hasAnte: false, wantCards: 2, wantMinLands: 1},
		{tier: 3, hasAnte: true, wantCards: 4, wantMinLands: 1},
		{tier: 3, hasAnte: false, wantCards: 3, wantMinLands: 1},
		{tier: 4, hasAnte: true, wantCards: 4, wantMinLands: 1},
		{tier: 5, hasAnte: true, wantCards: 5, wantMinLands: 2},
		{tier: 5, hasAnte: false, wantCards: 4, wantMinLands: 2},
		{tier: 7, hasAnte: true, wantCards: 6, wantMinLands: 2},
		{tier: 8, hasAnte: true, wantCards: 6, wantMinLands: 2},
		{tier: 10, hasAnte: true, wantCards: 6, wantMinLands: 2},
		{tier: 11, hasAnte: true, wantCards: 6, wantMinLands: 2},
	}

	for _, tt := range tests {
		var ante *Card
		if tt.hasAnte {
			ante = giantGrowth
		}
		reward := GenerateDuelReward(deck, ante, tt.tier, ColorRed)
		if len(reward.Cards) != tt.wantCards {
			t.Errorf("tier %d (hasAnte=%v): got %d cards, want %d", tt.tier, tt.hasAnte, len(reward.Cards), tt.wantCards)
		}
		if tt.hasAnte && reward.Cards[0] != giantGrowth {
			t.Errorf("tier %d: first card = %q, want ante card %q", tt.tier, reward.Cards[0].Name(), giantGrowth.Name())
		}
		landCount := 0
		for _, c := range reward.Cards {
			if c.IsLand() {
				landCount++
			}
		}
		if landCount < tt.wantMinLands {
			t.Errorf("tier %d: got %d lands, want at least %d", tt.tier, landCount, tt.wantMinLands)
		}
	}
}

func TestGenerateDuelReward_LandDistribution(t *testing.T) {
	deck := Deck{FindCardByName("Mountain"): 4}

	// Tier 1 should always give basic lands
	for range 30 {
		reward := GenerateDuelReward(deck, nil, 1, ColorRed)
		for _, c := range reward.Cards {
			if c.IsLand() && !IsBasicLand(c) {
				t.Fatalf("tier 1 granted non-basic land %q", c.CardName)
			}
		}
	}

	// Tier 10/11 should yield predominantly non-basic and dual lands (90% chance)
	foundNonBasicOrDual := false
	for range 30 {
		reward := GenerateDuelReward(deck, nil, 11, ColorRed)
		for _, c := range reward.Cards {
			if c.IsLand() && !IsBasicLand(c) {
				foundNonBasicOrDual = true
			}
		}
	}
	if !foundNonBasicOrDual {
		t.Errorf("tier 11 never yielded a non-basic or dual land in 30 trials")
	}

	// Tier 5 should yield dual lands and non-basic lands over many runs
	foundDual := false
	foundNonBasic := false
	for range 100 {
		reward := GenerateDuelReward(deck, nil, 5, ColorRed)
		for _, c := range reward.Cards {
			if IsDualLand(c) {
				foundDual = true
			} else if IsNonBasicLand(c) {
				foundNonBasic = true
			}
		}
	}
	if !foundDual {
		t.Errorf("tier 5 never yielded a dual land across 100 trials")
	}
	if !foundNonBasic {
		t.Errorf("tier 5 never yielded a non-basic utility land across 100 trials")
	}
}

func TestGenerateDuelReward_GoldAndAmulets(t *testing.T) {
	deck := Deck{FindCardByName("Mountain"): 4}

	// High tier enemy should reliably reward gold and matching amulet
	hasGold := false
	hasAmulet := false
	for range 50 {
		reward := GenerateDuelReward(deck, nil, 11, ColorRed)
		if reward.Gold >= 150 {
			hasGold = true
		}
		if len(reward.Amulets) >= 1 {
			hasAmulet = true
			for _, a := range reward.Amulets {
				if a.Color != ColorRed {
					t.Errorf("expected Red amulet for Red enemy, got %v", a.Color)
				}
			}
		}
	}
	if !hasGold {
		t.Errorf("tier 11 never rewarded high gold (>= 150)")
	}
	if !hasAmulet {
		t.Errorf("tier 11 never rewarded amulets")
	}
}

func TestLandClassification(t *testing.T) {
	mountain := FindCardByName("Mountain")
	badlands := FindCardByName("Badlands")
	stripMine := FindCardByName("Strip Mine")
	bolt := FindCardByName("Lightning Bolt")

	if !IsBasicLand(mountain) || IsDualLand(mountain) || IsNonBasicLand(mountain) {
		t.Errorf("Mountain classification incorrect: basic=%v, dual=%v, nonbasic=%v",
			IsBasicLand(mountain), IsDualLand(mountain), IsNonBasicLand(mountain))
	}

	if IsBasicLand(badlands) || !IsDualLand(badlands) || !IsNonBasicLand(badlands) {
		t.Errorf("Badlands classification incorrect: basic=%v, dual=%v, nonbasic=%v",
			IsBasicLand(badlands), IsDualLand(badlands), IsNonBasicLand(badlands))
	}

	if IsBasicLand(stripMine) || IsDualLand(stripMine) || !IsNonBasicLand(stripMine) {
		t.Errorf("Strip Mine classification incorrect: basic=%v, dual=%v, nonbasic=%v",
			IsBasicLand(stripMine), IsDualLand(stripMine), IsNonBasicLand(stripMine))
	}

	if IsBasicLand(bolt) || IsDualLand(bolt) || IsNonBasicLand(bolt) {
		t.Errorf("Lightning Bolt should not be classified as any land")
	}
}

func TestGenerateDuelReward_RestrictedCardsAreRare(t *testing.T) {
	deck := Deck{FindCardByName("Mountain"): 4}

	// Over 100 high-tier duel rewards (300+ random spell/land cards),
	// restricted cards should appear only very rarely (< 15% of all cards drawn).
	restrictedCount := 0
	totalRandomCards := 0
	for range 100 {
		reward := GenerateDuelReward(deck, nil, 11, ColorRed)
		for _, c := range reward.Cards {
			totalRandomCards++
			if c.VintageRestricted {
				restrictedCount++
			}
		}
	}

	restrictedPct := float64(restrictedCount) / float64(totalRandomCards) * 100.0
	if restrictedPct > 15.0 {
		t.Errorf("restricted cards appeared too frequently: %d/%d (%.1f%%, want <= 15%%)",
			restrictedCount, totalRandomCards, restrictedPct)
	}
}

func TestRandomCardsForColor_ZeroRestrictedChance(t *testing.T) {
	// When restrictedChance is 0, no restricted cards should be returned
	for range 50 {
		cards := randomCardsForColorInTiersWithRestrictedChance(ColorBlue, 3, 0.0, TierMandatory, TierAlmostMandatory, TierStaple)
		for _, c := range cards {
			if c.VintageRestricted {
				t.Fatalf("drew restricted card %q with 0.0 restricted chance", c.CardName)
			}
		}
	}
}

func TestGenerateDuelReward_MulticolorEnemy_Whim(t *testing.T) {
	whimChar, ok := Rogues["Whim"]
	if !ok || whimChar == nil {
		t.Fatalf("rogue 'Whim' not found in loaded rogues")
	}

	whimEnemy := Enemy{Character: whimChar}
	mask := whimEnemy.ColorMask()
	expectedMask := ColorWhite | ColorBlue | ColorBlack
	if mask != expectedMask {
		t.Fatalf("Whim ColorMask = %v, want %v (White | Blue | Black)", mask, expectedMask)
	}

	deck := Deck{FindCardByName("Mountain"): 4}
	shivanDragon := FindCardByName("Shivan Dragon")

	awardedColors := make(map[ColorMask]int)
	amuletTrials := 0

	for range 100 {
		reward := GenerateDuelReward(deck, shivanDragon, whimChar.Level, mask)
		if len(reward.Cards) != 6 {
			t.Errorf("expected 6 reward cards for level %d enemy with ante, got %d", whimChar.Level, len(reward.Cards))
		}
		if reward.Cards[0] != shivanDragon {
			t.Errorf("first reward card = %q, want ante card %q", reward.Cards[0].Name(), shivanDragon.Name())
		}
		for _, a := range reward.Amulets {
			amuletTrials++
			if a.Color&expectedMask == 0 {
				t.Errorf("awarded amulet %v is not one of Whim's colors (White, Blue, Black)", a.Color)
			}
			if a.Color == ColorRed || a.Color == ColorGreen {
				t.Errorf("awarded off-color amulet %v for Whim", a.Color)
			}
			awardedColors[a.Color]++
		}
	}

	if amuletTrials == 0 {
		t.Errorf("no amulets were awarded across 100 trials for level %d enemy", whimChar.Level)
	}
	if len(awardedColors) < 2 {
		t.Errorf("expected at least 2 different amulet colors across trials, got %d: %v", len(awardedColors), awardedColors)
	}
}



