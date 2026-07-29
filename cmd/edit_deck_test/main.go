package main

import (
	"fmt"
	"log"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	testScreenWidth  = 1024
	testScreenHeight = 768
)

type cardSpec struct {
	name   string
	total  int
	inDeck int
}

var testCardSpecs = []cardSpec{
	{name: "Plains", total: 12, inDeck: 8},
	{name: "Island", total: 12, inDeck: 8},
	{name: "Swamp", total: 12, inDeck: 8},
	{name: "Mountain", total: 12, inDeck: 8},
	{name: "Forest", total: 12, inDeck: 8},
	{name: "Serra Angel", total: 5, inDeck: 2},
	{name: "Counterspell", total: 7, inDeck: 4},
	{name: "Dark Ritual", total: 7, inDeck: 4},
	{name: "Lightning Bolt", total: 7, inDeck: 4},
	{name: "Llanowar Elves", total: 7, inDeck: 4},
	{name: "Sol Ring", total: 3, inDeck: 1},
	{name: "Air Elemental", total: 3},
	{name: "Disenchant", total: 3},
	{name: "Drain Life", total: 3},
	{name: "Fireball", total: 3},
	{name: "Giant Growth", total: 3},
}

func newTestCollection(specs []cardSpec) (domain.CardCollection, error) {
	collection := domain.NewCardCollection()
	for _, spec := range specs {
		card := domain.FindCardByName(spec.name)
		if card == nil {
			return nil, fmt.Errorf("card %q not found", spec.name)
		}
		if spec.inDeck > spec.total {
			return nil, fmt.Errorf("card %q has %d copies in deck but only %d total", spec.name, spec.inDeck, spec.total)
		}
		if spec.inDeck > 0 {
			collection.AddCardToDeck(card, 0, spec.inDeck)
		}
		if available := spec.total - spec.inDeck; available > 0 {
			collection.AddCard(card, available)
		}
	}
	return collection, nil
}

type testGame struct {
	screen        screenui.Screen
	updatePointer func()
}

func (g *testGame) Update() error {
	g.updatePointer()
	next, _, err := g.screen.Update(testScreenWidth, testScreenHeight, 1)
	if err != nil {
		return err
	}
	if next == screenui.CityScr {
		return ebiten.Termination
	}
	return nil
}

func (g *testGame) Draw(screen *ebiten.Image) {
	g.screen.Draw(screen, testScreenWidth, testScreenHeight, 1)
}

func (g *testGame) Layout(_, _ int) (int, int) {
	return testScreenWidth, testScreenHeight
}

func main() {
	if _, err := domain.LoadEmbeddedCardImages(); err != nil {
		log.Fatalf("Failed to load embedded card images: %v", err)
	}

	player, err := domain.NewPlayer("Deck Tester", nil, false, domain.DifficultyEasy, domain.ColorGreen)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	player.CardCollection, err = newTestCollection(testCardSpecs)
	if err != nil {
		log.Fatalf("Failed to create test collection: %v", err)
	}
	player.Gold = 1000

	city := &domain.City{Name: "Test City", Tier: domain.TierTown}
	editDeckScreen, err := screens.NewEditDeckScreen(player, city, testScreenWidth, testScreenHeight)
	if err != nil {
		log.Fatalf("Failed to create edit deck screen: %v", err)
	}

	g := &testGame{screen: editDeckScreen, updatePointer: ui.UpdatePointer}
	ebiten.SetWindowSize(testScreenWidth, testScreenHeight)
	ebiten.SetWindowTitle("Edit Deck Test")
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
