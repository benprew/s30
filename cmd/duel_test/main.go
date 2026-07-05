package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"

	_ "github.com/benprew/mage-go/cards"
	"github.com/benprew/mage-go/pkg/mage/interactive"
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
			panic(fmt.Sprintf("card %q not found\n", name))
		}
		deck[card] = count
	}

	// add("Fireball", 4)
	// add("Earthquake", 2)
	// add("Lightning Bolt", 4)
	// add("Rod of Ruin", 3)
	// add("Jade Statue", 2)
	// add("Kird Ape", 4)
	// add("Fire Elemental", 3)
	// add("Sol Ring", 2)
	// add("Mishra's Factory", 3)
	// add("Mountain", 14)
	// add("Forest", 6)
	add("Wall of Brambles", 20)
	add("Llanowar Elves", 10)
	add("Forest", 30)

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
	maxFrames  int
	frames     int
}

func (g *testGame) Update() error {
	g.frames++
	if g.maxFrames > 0 && g.frames > g.maxFrames {
		return ebiten.Termination
	}

	name, _, err := g.duelScreen.Update(1024, 768, 1.0)
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
	return nil
}

func (g *testGame) Draw(screen *ebiten.Image) {
	g.duelScreen.Draw(screen, 1024, 768, 1.0)
}

func (g *testGame) Layout(_, _ int) (int, int) {
	return 1024, 768
}

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to file")
	profileFrames := flag.Int("profileframes", 0, "terminate after this many update frames")
	rogue := flag.String("rogue", "", "fight this rogue instead of picking randomly")
	showOpponentHand := flag.Bool("show-opponent-hand", false, "reveal the opponent's hand (debug)")
	aiTestDeck := flag.Bool("ai-test-deck", false, "AI opponent plays xTestDeck() instead of its rogue deck")
	flag.Parse()

	interactive.RevealOpponentHand = *showOpponentHand

	if *memprofile != "" {
		runtime.MemProfileRate = 1
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalf("Failed to create CPU profile: %v", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("Failed to start CPU profile: %v", err)
		}
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	logging.Enable(logging.Duel)

	rogueName := *rogue
	if rogueName == "" {
		rogueName = pickRandomRogue()
	}
	fmt.Printf("Fighting: %s\n", rogueName)

	enemy, err := domain.NewEnemy(rogueName)
	if err != nil {
		log.Fatalf("Failed to create enemy %s: %v", rogueName, err)
	}

	if *aiTestDeck {
		// Copy the Character so we don't mutate the shared Rogues registry entry.
		enemyCharacter := *enemy.Character
		enemyCharacter.CardCollection = domain.NewCardCollection()
		for card, count := range xTestDeck() {
			enemyCharacter.CardCollection.AddCardToDeck(card, 0, count)
		}
		enemy.Character = &enemyCharacter
	}

	player, err := domain.NewPlayer("Test", nil, false, domain.DifficultyEasy, domain.ColorColorless)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	player.Life = 999

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

	g := &testGame{duelScreen: duelScreen, maxFrames: *profileFrames}

	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("Duel Test - X Spells")
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatalf("Failed to create memory profile: %v", err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalf("Failed to write memory profile: %v", err)
		}
	}
}
