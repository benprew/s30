package core_engine

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func createTestPlayer(numPlayers int) []*Player {
	players := []*Player{}
	entityID := EntityID(1)

	for i := range numPlayers {
		player := &Player{
			ID:          EntityID(i),
			LifeTotal:   20,
			ManaPool:    ManaPool{},
			Hand:        []*Card{},
			Library:     []*Card{},
			Battlefield: []*Card{},
			Graveyard:   []*Card{},
			Exile:       []*Card{},
			Turn:        &Turn{},
			InputChan:   make(chan PlayerAction, 100),
			IsAI:        true,
		}

		addCard := func(name string) {
			domainCard := domain.FindCardByName(name)
			if domainCard != nil {
				coreCard := NewCardFromDomain(domainCard, entityID, player)
				player.Library = append(player.Library, coreCard)
				entityID++
			}
		}

		addCard("Forest")
		addCard("Forest")
		addCard("Llanowar Elves")
		addCard("Llanowar Elves")
		addCard("Lightning Bolt")
		addCard("Mountain")
		addCard("Sol Ring")

		players = append(players, player)
	}

	return players
}

func TestMoveTo(t *testing.T) {
	player := createTestPlayer(1)[0]

	for range 7 {
		player.DrawCard()
	}

	card := player.Hand[0]
	err := player.MoveTo(card, ZoneGraveyard)
	if err != nil {
		t.Errorf("MoveTo failed: %v", err)
	}

	if len(player.Hand) != 6 {
		t.Errorf("Card not removed from hand, expected 6 got %d", len(player.Hand))
	}

	if len(player.Graveyard) != 1 {
		t.Errorf("Card not added to graveyard, expected 1 got %d", len(player.Graveyard))
	}

	if card.CurrentZone != ZoneGraveyard {
		t.Errorf("Card zone not updated, expected %d got %d", ZoneGraveyard, card.CurrentZone)
	}
}

func TestMoveToWrongOwner(t *testing.T) {
	players := createTestPlayer(2)
	player1 := players[0]
	player2 := players[1]

	player1.DrawCard()
	card := player1.Hand[0]

	err := player2.MoveTo(card, ZoneGraveyard)
	if err == nil {
		t.Errorf("Expected error when moving card owned by different player")
	}
}

func TestReceiveDamage(t *testing.T) {
	player := createTestPlayer(1)[0]
	initialLife := player.LifeTotal

	player.ReceiveDamage(5)
	if player.LifeTotal != initialLife-5 {
		t.Errorf("Expected life total %d, got %d", initialLife-5, player.LifeTotal)
	}

	player.ReceiveDamage(20)
	if player.LifeTotal != initialLife-25 {
		t.Errorf("Expected life total %d, got %d", initialLife-25, player.LifeTotal)
	}
}

func TestPlayerIsDead(t *testing.T) {
	player := createTestPlayer(1)[0]

	if player.IsDead() {
		t.Errorf("Player should not be dead at 20 life")
	}

	player.LifeTotal = 1
	if player.IsDead() {
		t.Errorf("Player should not be dead at 1 life")
	}

	player.LifeTotal = 0
	if !player.IsDead() {
		t.Errorf("Player should be dead at 0 life")
	}

	player.LifeTotal = -5
	if !player.IsDead() {
		t.Errorf("Player should be dead at negative life")
	}
}

func TestPlayerTargetable(t *testing.T) {
	player := createTestPlayer(1)[0]
	player.ID = EntityID(1)

	if player.Name() != "Player 1" {
		t.Errorf("Expected 'Player 1', got '%s'", player.Name())
	}

	if player.TargetType() != TargetTypePlayer {
		t.Errorf("Expected TargetTypePlayer, got %d", player.TargetType())
	}
}

func TestMoveToInvalidSourceZone(t *testing.T) {
	player := createTestPlayer(1)[0]
	player.DrawCard()
	card := player.Hand[0]

	card.CurrentZone = Zone(99)
	err := player.MoveTo(card, ZoneGraveyard)
	if err == nil {
		t.Errorf("Expected error for invalid source zone")
	}
}

func TestMoveToInvalidDestZone(t *testing.T) {
	player := createTestPlayer(1)[0]
	player.DrawCard()
	card := player.Hand[0]

	err := player.MoveTo(card, Zone(99))
	if err == nil {
		t.Errorf("Expected error for invalid destination zone")
	}
}

func TestMoveToCardNotInZone(t *testing.T) {
	player := createTestPlayer(1)[0]
	player.DrawCard()
	card := player.Hand[0]

	player.Hand = []*Card{}
	err := player.MoveTo(card, ZoneGraveyard)
	if err == nil {
		t.Errorf("Expected error when card not found in source zone")
	}
}
