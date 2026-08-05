package duel

import (
	"strings"
	"testing"

	_ "github.com/benprew/mage-go/cards"
	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestPutBonusPermanentsInPlayAddsToBattlefield(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	opp := ai.NewAIPlayer("Opp", heuristic.New(ai.MidrangeWeighted))
	g := newTestAnteGame(t, human, opp)

	s := &DuelScreen{human: human, game: g}
	s.putBonusPermanentsInPlay([]*domain.Card{{CardName: "Orcish Oriflamme"}})

	bf := g.AllBattlefield()
	if len(bf) != 1 {
		t.Fatalf("expected 1 permanent on the battlefield, got %d", len(bf))
	}
	if bf[0].Name() != "Orcish Oriflamme" {
		t.Errorf("expected Orcish Oriflamme on the battlefield, got %q", bf[0].Name())
	}
	if bf[0].ControllerID() != human.PlayerID() {
		t.Errorf("expected bonus permanent controlled by the human player, got %s want %s", bf[0].ControllerID(), human.PlayerID())
	}
}

func TestPutBonusPermanentsInPlayProcessesStaticAbilities(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	opp := ai.NewAIPlayer("Opp", heuristic.New(ai.MidrangeWeighted))
	g := newTestAnteGame(t, human, opp)
	bear, err := mage.CreateCard("Mesa Pegasus")
	if err != nil {
		t.Fatalf("CreateCard Mesa Pegasus: %v", err)
	}
	bearPerm := g.PutOnBattlefield(bear, human.PlayerID())

	s := &DuelScreen{human: human, game: g}
	s.putBonusPermanentsInPlay([]*domain.Card{{CardName: "Crusade"}})

	if got := bearPerm.CurrentPower(g); got != 2 {
		t.Errorf("expected Crusade to boost Mesa Pegasus power to 2, got %d", got)
	}
	if got := bearPerm.CurrentToughness(g); got != 2 {
		t.Errorf("expected Crusade to boost Mesa Pegasus toughness to 2, got %d", got)
	}
}

func TestDiceNotice(t *testing.T) {
	if got := diceNotice(0, nil); got != "" {
		t.Errorf("expected empty notice for no effects, got %q", got)
	}

	got := diceNotice(3, []*domain.Card{{CardName: "Serra Angel"}})
	if got == "" {
		t.Fatal("expected a non-empty notice")
	}
	for _, want := range []string{"+3", "Serra Angel"} {
		if !strings.Contains(got, want) {
			t.Errorf("notice %q missing %q", got, want)
		}
	}

	if got := diceNotice(-2, nil); !strings.Contains(got, "-2") {
		t.Errorf("disadvantage notice %q should mention -2", got)
	}
}

func TestInitGameStateUsesAnteGameWithoutInitialAnteCards(t *testing.T) {
	player, enemy := duelTestPlayers(t, nil, nil)
	s := &DuelScreen{player: player, enemy: enemy}

	s.initGameState()

	if !s.game.AnteEnabled() {
		t.Fatal("duel game does not have ante enabled")
	}
	cards, err := s.game.AnteCards()
	if err != nil {
		t.Fatalf("AnteCards: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("initial ante contains %d cards, want 0", len(cards))
	}
}

func TestInitGameStateUsesLevelEnemyStartingLife(t *testing.T) {
	player, enemy := duelTestPlayers(t, nil, nil)
	s := &DuelScreen{
		player: player,
		enemy:  enemy,
		lvl:    &world.Level{EnemyStartingLife: 1},
	}

	s.initGameState()

	if got := s.aiPlayer.Life(); got != 1 {
		t.Fatalf("enemy life = %d, want 1", got)
	}
}

func TestInitGameStateMovesSelectedCardsToAnte(t *testing.T) {
	playerAnte := domain.FindCardByName("Lightning Bolt")
	enemyAnte := domain.FindCardByName("Giant Growth")
	player, enemy := duelTestPlayers(t, playerAnte, enemyAnte)
	s := &DuelScreen{
		player:        player,
		enemy:         enemy,
		anteCard:      playerAnte,
		enemyAnteCard: enemyAnte,
	}

	s.initGameState()

	if !s.game.AnteEnabled() {
		t.Fatal("duel game does not have ante enabled")
	}
	cards, err := s.game.AnteCards()
	if err != nil {
		t.Fatalf("AnteCards: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("initial ante contains %d cards, want 2", len(cards))
	}
	got := map[string]bool{}
	for _, card := range cards {
		got[card.Name()] = true
	}
	if !got[playerAnte.Name()] || !got[enemyAnte.Name()] {
		t.Fatalf("initial ante cards = %v, want %q and %q", got, playerAnte.Name(), enemyAnte.Name())
	}
}

func duelTestPlayers(t *testing.T, playerAnte, enemyAnte *domain.Card) (*domain.Player, *domain.Enemy) {
	t.Helper()
	playerCards := domain.NewCardCollection()
	enemyCards := domain.NewCardCollection()
	playerCards.AddCardToDeck(domain.FindCardByName("Mountain"), 0, 8)
	enemyCards.AddCardToDeck(domain.FindCardByName("Forest"), 0, 8)
	if playerAnte != nil {
		playerCards.AddCardToDeck(playerAnte, 0, 1)
	}
	if enemyAnte != nil {
		enemyCards.AddCardToDeck(enemyAnte, 0, 1)
	}
	player := &domain.Player{Character: domain.Character{Life: 10, CardCollection: playerCards}}
	enemy := &domain.Enemy{Character: &domain.Character{Name: "Opponent", Life: 10, CardCollection: enemyCards}}
	return player, enemy
}
