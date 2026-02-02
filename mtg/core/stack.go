package core

import (
	"fmt"

	"github.com/benprew/s30/mtg/effects"
)

type Stack struct {
	Items             []*StackItem
	CurrentState      StackState
	ConsecutivePasses int
}

type StackItem struct {
	Events []effects.Event
	Player *Player
	Card   *Card
}

// NewStack creates a new Stack instance
func NewStack() Stack {
	return Stack{
		Items:             []*StackItem{},
		CurrentState:      StateStartStack, // Start in the initial state
		ConsecutivePasses: 0,
	}
}

// Define states for the stack state machine
type StackState int

const (
	StateStartStack StackState = iota
	StateWaitPlayer
	StateResolve
	StateEmpty
)

// Define event types that trigger state transitions
type StackEvent int

const (
	EventPlayerAddsAction StackEvent = iota
	EventPlayerPassesPriority
)

// Types of return values
type StackResult int

const (
	ActPlayerPriority StackResult = iota
	NonActPlayerPriority
	PlayerWait
	Resolve
)

// Transitions the stack to the next state
func (s *Stack) Next(event StackEvent, item *StackItem) (StackResult, *StackItem) {
	switch s.CurrentState {

	case StateStartStack:
		s.CurrentState = StateWaitPlayer
		return ActPlayerPriority, nil

	case StateWaitPlayer:
		switch event {
		case EventPlayerAddsAction:
			s.ConsecutivePasses = 0
			s.Push(item)
			return ActPlayerPriority, nil

		case EventPlayerPassesPriority:
			s.ConsecutivePasses++
			if s.ConsecutivePasses == 2 {
				s.CurrentState = StateResolve
				s.ConsecutivePasses = 0
				if s.IsEmpty() {
					return Resolve, nil
				}
				return Resolve, s.Pop()
			}
			return NonActPlayerPriority, nil

		default:
			panic(fmt.Sprintf("Unhandled event in StateWaitPlayer: %d", event))
		}

	case StateResolve:
		if s.IsEmpty() {
			s.CurrentState = StateEmpty
			return -1, nil
		}
		s.CurrentState = StateWaitPlayer
		return ActPlayerPriority, nil

	case StateEmpty:
		return -1, nil

	default:
		panic(fmt.Sprintf("Unhandled State: %d", s.CurrentState))
	}
}

// Push adds an action to the top of the stack.
func (s *Stack) Push(item *StackItem) {
	s.Items = append(s.Items, item)
}

func (s *Stack) Pop() *StackItem {
	if s.IsEmpty() {
		panic("Nothing to pop!")
	}
	topAction := s.Items[len(s.Items)-1]
	s.Items = s.Items[:len(s.Items)-1]
	return topAction
}

func (s *Stack) IsEmpty() bool {
	return len(s.Items) == 0
}
