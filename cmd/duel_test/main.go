package main

import (
	"fmt"
	"log"
	"math/rand"

	_ "git.sr.ht/~cdcarter/mage-go/cards"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

// xTestDeck returns a deck with X spells (Fireball, Earthquake) plus
// enough Mountains to cast them.
func xTestDeck() domain.Deck {
	deck := make(domain.Deck)
	add := func(name string, count int) {
		card := domain.FindCardByName(name)
		if card == nil {
			fmt.Printf("WARNING: card %q not found, skipping\n", name)
			return
		}
		deck[card] = count
	}

	add("Fireball", 4)
	add("Earthquake", 2)
	add("Drain Life", 2)
	add("Lightning Bolt", 4)
	add("Kird Ape", 4)
	add("Mountain", 14)
	add("Swamp", 4)
	add("Forest", 2)

	return deck
}

func pickRandomRogue() string {
	names := make([]string, 0, len(domain.Rogues))
	for name := range domain.Rogues {
		names = append(names, name)
	}
	return names[rand.Intn(len(names))]
}

type testGame struct {
	duelScreen *screens.DuelScreen
}

func (g *testGame) Update() error {
	name, screen, err := g.duelScreen.Update(1024, 768, 1.0)
	if err != nil {
		return err
	}
	if name == screenui.DuelWinScr || name == screenui.DuelLoseScr {
		if name == screenui.DuelWinScr {
			fmt.Println("You won!")
		} else {
			fmt.Println("You lost!")
		}
		return ebiten.Termination
	}
	_ = screen
	return nil
}

func (g *testGame) Draw(screen *ebiten.Image) {
	g.duelScreen.Draw(screen, 1024, 768, 1.0)
}

func (g *testGame) Layout(_, _ int) (int, int) {
	return 1024, 768
}

func main() {
	logging.Enable(logging.Duel)

	rogueName := pickRandomRogue()
	fmt.Printf("Fighting: %s\n", rogueName)

	enemy, err := domain.NewEnemy(rogueName)
	if err != nil {
		log.Fatalf("Failed to create enemy %s: %v", rogueName, err)
	}

	player, err := domain.NewPlayer("Test", nil, false, domain.DifficultyEasy)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}

	// Replace player's deck with our X-spell test deck
	player.CardCollection = domain.NewCardCollection()
	for card, count := range xTestDeck() {
		player.CardCollection.AddCardToDeck(card, 0, count)
	}

	// Minimal level for win/lose handling
	lvl := &world.Level{
		Player:  player,
		Enemies: []domain.Enemy{enemy},
	}

	duelScreen := screens.NewDuelScreen(player, &enemy, lvl, 0, nil, nil)

	g := &testGame{duelScreen: duelScreen}

	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("Duel Test - X Spells")
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
