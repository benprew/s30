package domain

import "testing"

func newTestPlayer() *Player {
	return &Player{
		Character: Character{CardCollection: NewCardCollection()},
		Amulets:   make(map[ColorMask]int),
	}
}

func TestCollectRewardRestrictedCardAddsToCollectionAndStateAndClearsTile(t *testing.T) {
	card := &Card{CardName: "Black Lotus"}
	tile := &DungeonTile{
		Type:   DungeonTileTreasure,
		Reward: &DungeonReward{Type: DungeonRewardRestrictedCard, Card: card},
	}
	st := &DungeonState{}
	p := newTestPlayer()

	st.CollectReward(tile, p)

	if len(st.CollectedCards) != 1 || st.CollectedCards[0] != card {
		t.Errorf("expected card recorded in CollectedCards, got %v", st.CollectedCards)
	}
	deck := p.CardCollection.GetDeck(0)
	if deck[card] != 1 {
		t.Errorf("expected card added to player deck 0, got count %d", deck[card])
	}
	if tile.Type != DungeonTileEmpty || tile.Reward != nil {
		t.Errorf("expected tile cleared to empty, got type=%v reward=%v", tile.Type, tile.Reward)
	}
}

func TestCollectRewardGoldAndAmulets(t *testing.T) {
	tile := &DungeonTile{
		Type: DungeonTileTreasure,
		Reward: &DungeonReward{
			Type:    DungeonRewardGoldAmulets,
			Gold:    75,
			Amulets: []Amulet{NewAmulet(ColorWhite), NewAmulet(ColorWhite)},
		},
	}
	st := &DungeonState{}
	p := newTestPlayer()
	startGold := p.Gold

	st.CollectReward(tile, p)

	if p.Gold != startGold+75 {
		t.Errorf("expected gold +75 (= %d), got %d", startGold+75, p.Gold)
	}
	if p.Amulets[ColorWhite] != 2 {
		t.Errorf("expected 2 white amulets, got %d", p.Amulets[ColorWhite])
	}
	if tile.Type != DungeonTileEmpty {
		t.Errorf("expected tile cleared, got %v", tile.Type)
	}
}

func TestCollectRewardNoOpOnNonTreasureTile(t *testing.T) {
	tile := &DungeonTile{Type: DungeonTileEmpty}
	st := &DungeonState{}
	p := newTestPlayer()

	st.CollectReward(tile, p)

	if p.Gold != 0 || len(st.CollectedCards) != 0 {
		t.Errorf("non-treasure tile should not award anything")
	}
}

func TestCollectRewardNoOpOnNilTile(t *testing.T) {
	st := &DungeonState{}
	p := newTestPlayer()

	st.CollectReward(nil, p) // must not panic
}
