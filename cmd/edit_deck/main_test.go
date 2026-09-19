package main

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewEditorCollectionDefault(t *testing.T) {
	collection, err := newEditorCollection("")
	if err != nil {
		t.Fatal(err)
	}
	if len(collection) != len(domain.CARDS) {
		t.Fatalf("collection size = %d, want %d", len(collection), len(domain.CARDS))
	}
	for _, card := range domain.CARDS {
		if got := collection.GetTotalCount(card); got != 4 {
			t.Fatalf("%s count = %d, want 4", card.Name(), got)
		}
	}
	if len(collection.GetDeck(0)) != 0 {
		t.Fatal("default deck is not empty")
	}
}

func TestNewEditorCollectionRogue(t *testing.T) {
	for name, rogue := range domain.Rogues {
		t.Run(name, func(t *testing.T) {
			collection, err := newEditorCollection(name)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(collection, rogue.CardCollection) {
				t.Fatal("collection does not match rogue deck and sideboard")
			}
			for card, item := range collection {
				original := *rogue.CardCollection[card]
				original.DeckCounts = slices.Clone(original.DeckCounts)
				item.Count++
				if len(item.DeckCounts) > 0 {
					item.DeckCounts[0]++
				}
				if !reflect.DeepEqual(&original, rogue.CardCollection[card]) {
					t.Fatal("editor changed the rogue collection")
				}
			}
			fresh, err := newEditorCollection(name)
			if err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(collection, fresh) {
				t.Fatal("editor collection is shared")
			}
		})
	}
}

func TestNewEditorCollectionRejectsUnknownRogue(t *testing.T) {
	if _, err := newEditorCollection("Definitely Not A Rogue"); err == nil {
		t.Fatal("expected unknown rogue error")
	}
}

type exitScreen struct {
	pointerUpdated *bool
}

func (s *exitScreen) Update(_, _ int, _ float64) (screenui.ScreenName, screenui.Screen, error) {
	if !*s.pointerUpdated {
		return screenui.NoScr, nil, errors.New("screen updated before pointer input")
	}
	return screenui.CityScr, nil, nil
}

func (s *exitScreen) Draw(*ebiten.Image, int, int, float64) {}
func (s *exitScreen) IsFramed() bool                        { return false }
func (s *exitScreen) IsOverlay() bool                       { return false }

func TestGameTerminatesWhenEditDeckReturnsToCity(t *testing.T) {
	pointerUpdated := false
	g := &editorGame{
		screen:        &exitScreen{pointerUpdated: &pointerUpdated},
		updatePointer: func() { pointerUpdated = true },
	}

	if err := g.Update(); err != ebiten.Termination {
		t.Fatalf("Update() error = %v, want ebiten.Termination", err)
	}
}
