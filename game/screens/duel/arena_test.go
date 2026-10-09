package duel

import (
	"fmt"
	"math/rand"
	"testing"

	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"

)

func TestArenaRewardContinueLabel(t *testing.T) {
	for round := 2; round <= 7; round++ {
		win := &DuelWinScreen{doneBtn: &elements.Button{}, textbox: &elements.Button{}, cards: make([]winCard, 4)}
		label := fmt.Sprintf("Go to Round %d", round)
		win.SetContinueText(label)
		if win.doneBtn.ButtonText.Text != "Next" {
			t.Fatal("intermediate reward page must keep Next")
		}
		win.nextPage()
		if win.doneBtn.ButtonText.Text != label {
			t.Fatalf("final reward page label = %q, want %q", win.doneBtn.ButtonText.Text, label)
		}
	}
}

func TestArenaOutcomesBypassCampaign(t *testing.T) {
	quest := &domain.Quest{Type: domain.QuestTypeDefeatEnemy, EnemyName: "Arena Enemy"}
	player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}, ActiveQuests: []*domain.Quest{quest}}
	enemy := &domain.Enemy{Character: &domain.Character{Name: "Arena Enemy"}}
	level := &world.Level{Player: player, Enemies: []domain.Enemy{*enemy}}
	for _, won := range []bool{true, false} {
		calls := 0
		s := &DuelScreen{player: player, enemy: enemy, lvl: level, arenaOutcome: func(result bool) (screenui.ScreenName, screenui.Screen, error) {
			calls++
			if result != won {
				t.Fatal("incorrect arena outcome")
			}
			return screenui.WorldScr, nil, nil
		}}
		var err error
		if won {
			_, _, err = s.handleWin()
		} else {
			_, _, err = s.handleLoss()
		}
		if err != nil || calls != 1 || quest.IsCompleted || level.CombatsWon != 0 || len(level.Enemies) != 1 || player.CardCollection.NumCards() != 0 {
			t.Fatal("arena outcome changed campaign state")
		}
		s.applyQuestProgress(won)
	}
}

func TestArenaPackCardsHaveEngineImplementations(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	seen := map[string]bool{}
	for range 500 {
		for _, card := range domain.ArenaBooster(rng) {
			if seen[card.CardName] {
				continue
			}
			seen[card.CardName] = true
			if _, err := mage.CreateCard(card.CardName); err != nil {
				t.Errorf("arena card %s is not supported: %v", card.CardName, err)
			}
		}
	}
}

func TestArenaDuelUsesStartingLifeAndNoAnte(t *testing.T) {
	bonus := domain.FindCardByName("Serra Angel")
	campaign := &domain.Player{Character: domain.Character{Life: 13, CardCollection: domain.NewCardCollection()},
		Gold: 300, BonusDuelLife: 5, BonusDuelCards: []*domain.Card{bonus}}
	run, err := domain.NewArenaRun(campaign, rand.New(rand.NewSource(21)))
	if err != nil {
		t.Fatal(err)
	}
	enemy, err := run.NewOpponent()
	if err != nil {
		t.Fatal(err)
	}
	ante := &DuelAnteScreen{player: run.Player, enemy: enemy, arenaRound: 1,
		arenaOutcome: func(bool) (screenui.ScreenName, screenui.Screen, error) { return screenui.WorldScr, nil, nil }}
	name, screen, err := ante.startDuel()
	if err != nil || name != screenui.DuelScr {
		t.Fatal("arena did not start a duel")
	}
	duel := screen.(*DuelScreen)
	defer duel.Close()
	if duel.human.Life() != 13 || duel.aiPlayer.Life() != enemy.Character.Life || duel.arenaOutcome == nil {
		t.Fatal("arena duel has incorrect life or outcome handler")
	}
	if len(duel.human.Library())+len(duel.human.Hand()) != 15 {
		t.Fatal("arena deck was not padded to the round minimum")
	}
	anteCards, err := duel.game.AnteCards()
	if err != nil || len(anteCards) != 0 || len(duel.game.AllBattlefield()) != 0 {
		t.Fatal("arena duel has ante cards or campaign bonus permanents")
	}
	if campaign.BonusDuelLife != 5 || len(campaign.BonusDuelCards) != 1 {
		t.Fatal("arena consumed pending campaign bonuses")
	}
}

func TestArenaDuelAnteUniformButtonWidthsAndAlignment(t *testing.T) {
	player := &domain.Player{Character: domain.Character{Life: 10, CardCollection: domain.NewCardCollection()}}
	enemy := &domain.Enemy{Character: &domain.Character{Name: "Test Enemy", Life: 10, Visage: ebiten.NewImage(10, 10)}}
	ante := NewArenaDuelAnteScreen(player, enemy, 1, nil, nil, nil)

	if ante.editBtn == nil {
		t.Fatal("expected edit button in arena ante")
	}
	w1 := ante.duelBtn.Bounds.Dx()
	w2 := ante.bribeBtn.Bounds.Dx()
	w3 := ante.editBtn.Bounds.Dx()
	if w1 != w2 || w2 != w3 {
		t.Errorf("buttons should have uniform width: duel=%d, bribe=%d, edit=%d", w1, w2, w3)
	}
	x1 := ante.duelBtn.Bounds.Min.X
	x2 := ante.bribeBtn.Bounds.Min.X
	x3 := ante.editBtn.Bounds.Min.X
	if x1 != x2 || x2 != x3 {
		t.Errorf("buttons should have same X alignment: duel=%d, bribe=%d, edit=%d", x1, x2, x3)
	}
}

