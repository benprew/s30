package core_engine

import "fmt"

// The stack in MTG is complex enough that it deserves it's own file.
// It's central to the gamea and most effects use it
type Stack struct {
	Items             []*StackItem
	CurrentState      StackState // Current state of the stack state machine
	ConsecutivePasses int        // Tracks consecutive passes to determine stack resolution
}

type StackItem struct {
	Events []Event // a spell can have multiple events associated with it
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
			if s.ConsecutivePasses == 2 { // Both players have passed in succession
				s.CurrentState = StateResolve
				s.ConsecutivePasses = 0
				if s.IsEmpty() {
					return Resolve, nil
				}
				return Resolve, s.Pop()
			} else {
				return NonActPlayerPriority, nil
			}
		}

	case StateResolve:
		if s.IsEmpty() {
			s.CurrentState = StateEmpty
		} else {
			s.CurrentState = StateWaitPlayer
		}

	case StateEmpty:
		if !s.IsEmpty() {
			s.CurrentState = StateStartStack
		}
	default:
		panic(fmt.Sprintf("Unhandled State: %d", s.CurrentState))
	}

	return -1, nil
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
