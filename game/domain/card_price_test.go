package domain

import (
	"testing"
)

func TestBasicLandsPrice(t *testing.T) {
	basicLands := []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}
	for _, landName := range basicLands {
		c := FindCardByName(landName)
		if c == nil {
			t.Fatalf("card %s not found in CARDS", landName)
		}
		if c.Price != 30 {
			t.Errorf("Basic land %s price = %d, want 30", landName, c.Price)
		}
	}
}

func TestMandatoryCardsPriceCap(t *testing.T) {
	for _, c := range CardsByTier[TierS] {
		if c.Price > 7000 {
			t.Errorf("Tier S card %s price = %d, exceeds cap of 7000", c.CardName, c.Price)
		}
		if c.Price < 3500 {
			t.Errorf("Tier S card %s price = %d, unexpectedly low for Tier S", c.CardName, c.Price)
		}
	}

	lotus := FindCardByName("Black Lotus")
	if lotus == nil {
		t.Fatal("Black Lotus not found")
	}
	if lotus.Price != 7000 {
		t.Errorf("Black Lotus price = %d, want 7000 (cap)", lotus.Price)
	}
}

func TestTierABasePrice(t *testing.T) {
	if base := BasePriceForTier(TierA); base != 350 {
		t.Errorf("TierA base price = %d, want 350", base)
	}

	tierACards := CardsByTier[TierA]
	if len(tierACards) == 0 {
		t.Fatal("No Tier A cards found")
	}

	for _, c := range tierACards {
		if c.Price < 150 || c.Price > 500 {
			t.Errorf("Tier A card %s price = %d out of expected range [150, 500]", c.CardName, c.Price)
		}
	}
}

func TestTierFPrice(t *testing.T) {
	for _, c := range CardsByTier[TierF] {
		if c.Price > 35 {
			t.Errorf("Tier F card %s price = %d, should be <= 35", c.CardName, c.Price)
		}
		if c.Price < 7 {
			t.Errorf("Tier F card %s price = %d, below minimum of 7", c.CardName, c.Price)
		}
	}
}

func TestGlobalMinimumPrice(t *testing.T) {
	for _, c := range CARDS {
		if c.Price < 7 {
			t.Errorf("Card %s has price %d, which is below global minimum 7", c.CardName, c.Price)
		}
	}
}

func TestTierHierarchy(t *testing.T) {
	tiers := []CardTier{
		TierS,
		TierA,
		TierB,
		TierC,
		TierD,
		TierF,
	}

	for i := 0; i < len(tiers)-1; i++ {
		baseA := BasePriceForTier(tiers[i])
		baseB := BasePriceForTier(tiers[i+1])
		if baseA <= baseB {
			t.Errorf("Tier %d base price (%d) should be strictly greater than tier %d base price (%d)",
				tiers[i], baseA, tiers[i+1], baseB)
		}
	}
}

func TestCalculateCardPrice_ValueAdjustment(t *testing.T) {
	// For the same card tier, higher USD price should yield higher or equal in-game price.
	cheapStaplePrice := CalculateCardPrice("Lightning Bolt", "Instant", 1.50)
	expensiveStaplePrice := CalculateCardPrice("Underground Sea", "Land - Island Swamp", 800.00)

	if expensiveStaplePrice <= cheapStaplePrice {
		t.Errorf("Expected expensive staple ($800) price %d to exceed cheap staple ($1.50) price %d",
			expensiveStaplePrice, cheapStaplePrice)
	}

	// Basic lands should always return 30 regardless of USD value
	if p := CalculateCardPrice("Mountain", "Basic Land — Mountain", 0.10); p != 30 {
		t.Errorf("Mountain price = %d, want 30", p)
	}
	if p := CalculateCardPrice("Island", "Basic Land — Island", 500.00); p != 30 {
		t.Errorf("Island price with high USD = %d, want 30", p)
	}
}
