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

func NewDungeonDuelScreen(player *domain.Player, enemy *domain.Enemy, state *domain.DungeonState, tile *domain.DungeonTile) *DuelScreen {
	return duelscreen.NewDungeonDuelScreen(player, enemy, state, tile)
}

func NewDuelAnteScreen() *DuelAnteScreen {
	return duelscreen.NewDuelAnteScreen()
}

func NewDuelAnteScreenWithEnemy(l *world.Level, idx int) *DuelAnteScreen {
	return duelscreen.NewDuelAnteScreenWithEnemy(l, idx)
}

func NewWinDuelScreen(player *domain.Player, choices []*domain.Card, bonusCards []*domain.Card) *DuelWinScreen {
	return duelscreen.NewWinDuelScreen(player, choices, bonusCards)
}

func NewDuelLoseScreen(cards []*domain.Card) *DuelLoseScreen {
	return duelscreen.NewDuelLoseScreen(cards)
}
