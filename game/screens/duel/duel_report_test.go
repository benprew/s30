package duel

import (
	"testing"

	"github.com/benprew/s30/game/bugreport"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestDuelScreenImplementsDuelReporter(t *testing.T) {
	player, err := domain.NewPlayer("Hero", nil, false, domain.DifficultyEasy, domain.ColorGreen)
	if err != nil {
		t.Fatalf("NewPlayer failed: %v", err)
	}
	level, err := world.NewLevel(player)
	if err != nil {
		t.Fatalf("NewLevel failed: %v", err)
	}

	enemyRogue := domain.Rogues["Sea Troll"]
	if enemyRogue == nil {
		t.Skip("Sea Troll rogue not found")
	}
	enemy := domain.NewEnemyFromCharacter(enemyRogue)

	duelScr := NewDuelScreen(player, &enemy, level, 0, nil, nil)
	defer duelScr.Close()

	var reporter bugreport.DuelReporter = duelScr
	state := reporter.DuelReportState()

	if state == nil {
		t.Fatal("DuelReportState returned nil")
	}
	if state.OpponentName != "Sea Troll" {
		t.Errorf("OpponentName = %q, want Sea Troll", state.OpponentName)
	}
	if state.GameState == nil {
		t.Error("GameState snapshot was not generated")
	}
}
