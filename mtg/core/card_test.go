package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestGiantGrowthTargetsCreaturesOnly(t *testing.T) {
	domainCard := domain.FindCardByName("Giant Growth")
	if domainCard == nil {
		t.Fatal("Giant Growth card not found")
	}

	card := NewCardFromDomain(domainCard, 1, nil)
	if !card.TargetsCreaturesOnly() {
		t.Errorf("Giant Growth should return true for TargetsCreaturesOnly()")
	}
}
