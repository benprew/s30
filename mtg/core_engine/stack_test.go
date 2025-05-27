package core_engine

import (
    "fmt"
    "sync"
    "testing"
)

func TestRunStack(t *testing.T) {
    // Test the next turn functionality with 1 player, make sure the player
    // has the opportunity to respond in each phase
    players := createTestPlayer(2)
    player := players[0]
    player2 := players[1]
    game := NewGame(players)

    // Check that the player had an opportunity to respond in each phase
    expectedPhases := []Phase{
        PhaseUpkeep,
        PhaseDraw,
        PhaseMain1,
        PhaseCombat,
        PhaseMain2,
        PhaseEnd,
    }
    var wg sync.WaitGroup
    wg.Add(1)

    // Start a goroutine to simulate player responses
    go func() {
        defer wg.Done()
        // For each expected phase, send a PassPriority action
        for range expectedPhases {
            fmt.Println("player2 Passing Priority")
            player.InputChan <- PlayerAction{Type: "PassPriority"}
            player2.InputChan <- PlayerAction{Type: "PassPriority"}
        }
    }()

    // Start a turn
    player.Turn.Phase = PhaseUntap
    game.RunStack()
    wg.Wait()
    fmt.Println("waitgroup finished")
}
