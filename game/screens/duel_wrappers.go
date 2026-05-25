package screens

import (
	"github.com/benprew/s30/game/domain"
	duelscreen "github.com/benprew/s30/game/screens/duel"
	"github.com/benprew/s30/game/world"
)

type DuelScreen = duelscreen.DuelScreen
type DuelAnteScreen = duelscreen.DuelAnteScreen
type DuelWinScreen = duelscreen.DuelWinScreen
type DuelLoseScreen = duelscreen.DuelLoseScreen

func NewDuelScreen(player *domain.Player, enemy *domain.Enemy, lvl *world.Level, idx int, anteCard *domain.Card, enemyAnteCard *domain.Card) *DuelScreen {
	return duelscreen.NewDuelScreen(player, enemy, lvl, idx, anteCard, enemyAnteCard)
}

func NewDuelAnteScreen() *DuelAnteScreen {
	return duelscreen.NewDuelAnteScreen()
}

func NewDuelAnteScreenWithEnemy(l *world.Level, idx int) *DuelAnteScreen {
	return duelscreen.NewDuelAnteScreenWithEnemy(l, idx)
}

func NewWinDuelScreen(cards []*domain.Card) *DuelWinScreen {
	return duelscreen.NewWinDuelScreen(cards)
}

func NewDuelLoseScreen(cards []*domain.Card) *DuelLoseScreen {
	return duelscreen.NewDuelLoseScreen(cards)
}
