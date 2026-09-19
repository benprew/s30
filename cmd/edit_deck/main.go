package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	editorScreenWidth  = 1024
	editorScreenHeight = 768
)

func newEditorCollection(rogueName string) (domain.CardCollection, error) {
	collection := domain.NewCardCollection()
	if rogueName == "" {
		for _, card := range domain.CARDS {
			collection.AddCard(card, 4)
		}
		return collection, nil
	}
	rogue, ok := domain.Rogues[rogueName]
	if !ok {
		names := make([]string, 0, len(domain.Rogues))
		for name := range domain.Rogues {
			names = append(names, name)
		}
		slices.Sort(names)
		return nil, fmt.Errorf("unknown rogue %q; available rogues: %s", rogueName, strings.Join(names, ", "))
	}
	for card, item := range rogue.CardCollection {
		cloned := *item
		cloned.DeckCounts = slices.Clone(item.DeckCounts)
		collection[card] = &cloned
	}
	return collection, nil
}

type editorGame struct {
	screen        screenui.Screen
	updatePointer func()
}

func (g *editorGame) Update() error {
	g.updatePointer()
	next, _, err := g.screen.Update(editorScreenWidth, editorScreenHeight, 1)
	if err != nil {
		return err
	}
	if next == screenui.CityScr {
		return ebiten.Termination
	}
	return nil
}

func (g *editorGame) Draw(screen *ebiten.Image) {
	g.screen.Draw(screen, editorScreenWidth, editorScreenHeight, 1)
}

func (g *editorGame) Layout(_, _ int) (int, int) {
	return editorScreenWidth, editorScreenHeight
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: edit_deck [\"Rogue Name\"]")
		fmt.Fprintln(os.Stderr, "With no rogue name, open a blank deck with four copies of every card.")
	}
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	collection, err := newEditorCollection(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	if _, err = domain.LoadEmbeddedCardImages(); err != nil {
		log.Fatalf("Failed to load embedded card images: %v", err)
	}

	player, err := domain.NewPlayer("Deck Editor", nil, false, domain.DifficultyEasy, domain.ColorGreen)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	player.CardCollection = collection
	player.Gold = 1000

	city := &domain.City{Name: "Test City", Tier: domain.TierTown}
	editDeckScreen, err := screens.NewEditDeckScreen(player, city, editorScreenWidth, editorScreenHeight)
	if err != nil {
		log.Fatalf("Failed to create edit deck screen: %v", err)
	}

	g := &editorGame{screen: editDeckScreen, updatePointer: ui.UpdatePointer}
	ebiten.SetWindowSize(editorScreenWidth, editorScreenHeight)
	ebiten.SetWindowTitle("Edit Deck")
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
