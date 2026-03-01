package core

import (
	"testing"
)

func TestStackEventPlayerAddsAction(t *testing.T) {
	stack := NewStack()

	if stack.CurrentState != StateStartStack {
		t.Errorf("Expected initial state StateStartStack, got %d", stack.CurrentState)
	}

	result, _ := stack.Next(-1, nil)
	if stack.CurrentState != StateWaitPlayer {
		t.Errorf("Expected StateWaitPlayer after start, got %d", stack.CurrentState)
	}
	if result != ActPlayerPriority {
		t.Errorf("Expected ActPlayerPriority, got %d", result)
	}

	item := &StackItem{Events: nil, Player: nil, Card: nil}
	result, _ = stack.Next(EventPlayerAddsAction, item)

	if stack.ConsecutivePasses != 0 {
		t.Errorf("ConsecutivePasses should reset to 0 after action, got %d", stack.ConsecutivePasses)
	}
	if len(stack.Items) != 1 {
		t.Errorf("Stack should have 1 item, got %d", len(stack.Items))
	}
	if result != ActPlayerPriority {
		t.Errorf("Expected ActPlayerPriority after adding action, got %d", result)
	}
}

func TestStackLIFOResolution(t *testing.T) {
	stack := NewStack()

	item1 := &StackItem{Card: &Card{}}
	item1.Card.CardName = "First"
	item2 := &StackItem{Card: &Card{}}
	item2.Card.CardName = "Second"
	item3 := &StackItem{Card: &Card{}}
	item3.Card.CardName = "Third"

	stack.Next(-1, nil)
	stack.Next(EventPlayerAddsAction, item1)
	stack.Next(EventPlayerAddsAction, item2)
	stack.Next(EventPlayerAddsAction, item3)

	stack.Next(EventPlayerPassesPriority, nil)
	_, resolved := stack.Next(EventPlayerPassesPriority, nil)

	if resolved.Card.CardName != "Third" {
		t.Errorf("Expected Third to resolve first (LIFO), got %s", resolved.Card.CardName)
	}

	stack.Next(-1, nil)
	stack.Next(EventPlayerPassesPriority, nil)
	_, resolved = stack.Next(EventPlayerPassesPriority, nil)

	if resolved.Card.CardName != "Second" {
		t.Errorf("Expected Second to resolve second, got %s", resolved.Card.CardName)
	}

	stack.Next(-1, nil)
	stack.Next(EventPlayerPassesPriority, nil)
	_, resolved = stack.Next(EventPlayerPassesPriority, nil)

	if resolved.Card.CardName != "First" {
		t.Errorf("Expected First to resolve last, got %s", resolved.Card.CardName)
	}
}

func TestStackResolveEmpty(t *testing.T) {
	stack := NewStack()

	stack.Next(-1, nil)
	stack.Next(EventPlayerPassesPriority, nil)
	result, item := stack.Next(EventPlayerPassesPriority, nil)

	if result != -1 {
		t.Errorf("Expected -1 result when both pass with empty stack, got %d", result)
	}
	if item != nil {
		t.Errorf("Expected nil item when both pass with empty stack, got %v", item)
	}
	if stack.CurrentState != StateEmpty {
		t.Errorf("Expected StateEmpty when both pass with empty stack, got %d", stack.CurrentState)
	}
}

func TestStackConsecutivePassesTracking(t *testing.T) {
	stack := NewStack()

	stack.Next(-1, nil)

	result, _ := stack.Next(EventPlayerPassesPriority, nil)
	if stack.ConsecutivePasses != 1 {
		t.Errorf("Expected ConsecutivePasses=1, got %d", stack.ConsecutivePasses)
	}
	if result != NonActPlayerPriority {
		t.Errorf("Expected NonActPlayerPriority after first pass, got %d", result)
	}

	stack.Next(EventPlayerAddsAction, &StackItem{})
	if stack.ConsecutivePasses != 0 {
		t.Errorf("Expected ConsecutivePasses=0 after action, got %d", stack.ConsecutivePasses)
	}
}

func TestRunStack(t *testing.T) {
	players := createTestPlayer(2)
	player := players[0]
	player2 := players[1]
	game := NewGame(players)

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-player.WaitingChan
		player.InputChan <- PlayerAction{Type: ActionPassPriority}
		<-player2.WaitingChan
		player2.InputChan <- PlayerAction{Type: ActionPassPriority}
	}()

	player.Turn.Phase = PhaseUntap
	game.RunStack()
	<-done
}
