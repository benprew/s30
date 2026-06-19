package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/benprew/mage-go/cards"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

// testGame drives the dungeon screen and follows screen transitions the same
// way game.go does, so walking into an enemy actually starts (and resolves) a
// duel before returning to the dungeon. It terminates once the player leaves
// the dungeon — either by pressing ESC or by losing a duel and being expelled.
type testGame struct {
	screens map[screenui.ScreenName]screenui.Screen
	current screenui.ScreenName
}

func (g *testGame) Update() error {
	name, screen, err := g.screens[g.current].Update(1024, 768, 1.0)
	if err != nil {
		return err
	}
	if screen != nil {
		g.screens[name] = screen
	}
	if name != g.current {
		switch name {
		case screenui.DuelScr:
			fmt.Println("Duel started!")
		case screenui.DuelWinScr:
			fmt.Println("You won the duel!")
		case screenui.DuelLoseScr:
			fmt.Println("You lost the duel!")
		case screenui.WorldScr:
			fmt.Println("Left the dungeon.")
			return ebiten.Termination
		}
	}
	g.current = name
	return nil
}

func (g *testGame) Draw(screen *ebiten.Image) {
	if s, ok := g.screens[g.current]; ok {
		s.Draw(screen, 1024, 768, 1.0)
	}
}

func (g *testGame) Layout(_, _ int) (int, int) {
	return 1024, 768
}

func colorFromName(name string) domain.ColorMask {
	switch name {
	case "white", "W":
		return domain.ColorWhite
	case "blue", "U":
		return domain.ColorBlue
	case "black", "B":
		return domain.ColorBlack
	case "red", "R":
		return domain.ColorRed
	case "green", "G":
		return domain.ColorGreen
	default:
		return domain.ColorRed
	}
}

func selectRestrictedCards(cards []*domain.Card, rng *rand.Rand) []*domain.Card {
	seen := make(map[string]bool)
	pool := make([]*domain.Card, 0)
	for _, card := range cards {
		if !card.VintageRestricted || seen[card.CardName] {
			continue
		}
		seen[card.CardName] = true
		pool = append(pool, card)
	}
	if len(pool) == 0 {
		return nil
	}

	rng.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})

	maxCount := min(len(pool), 4)
	count := rng.Intn(maxCount) + 1
	return pool[:count]
}

func main() {
	showOpponentHand := flag.Bool("show-opponent-hand", false, "reveal the opponent's hand in dungeon duels (debug)")
	colorName := flag.String("color", "red", "dungeon color (white/blue/black/red/green)")
	enemies := flag.Int("enemies", 3, "number of enemies to place in the dungeon")
	flag.Parse()

	interactive.RevealOpponentHand = *showOpponentHand

	logging.Enable(logging.Duel)

	color := colorFromName(*colorName)

	player, err := domain.NewPlayer("Test", nil, false, domain.DifficultyEasy, color)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}

	seed := time.Now().UnixNano()
	restrictedCards := selectRestrictedCards(domain.CARDS, rand.New(rand.NewSource(seed)))

	dungeon := domain.GenerateDungeon(domain.DungeonGenOptions{
		Name:            "Test Dungeon",
		Level:           1,
		Color:           color,
		CreatureSize:    domain.CreatureSizeSmall,
		GridSize:        11,
		NumEnemies:      *enemies,
		NumDice:         10,
		NumScrolls:      1,
		NumGoldChests:   2,
		RestrictedCards: restrictedCards,
		EnemyPool:       domain.DungeonEnemyPool(color),
		DiceCardPool:    player.GetDuelDeck().NonLandCards(),
		Seed:            seed,
	})

	player.DungeonState = &domain.DungeonState{
		CurrentDungeon: dungeon,
		Position:       dungeon.Entrance,
		DungeonLife:    player.Life,
	}
	dungeon.RevealFrom(dungeon.Entrance)

	lvl := &world.Level{
		Player:   player,
		Dungeons: []*domain.Dungeon{dungeon},
	}

	g := &testGame{
		screens: map[screenui.ScreenName]screenui.Screen{
			screenui.DungeonScr: screens.NewDungeonScreen(player, lvl),
		},
		current: screenui.DungeonScr,
	}

	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("Dungeon Test")
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
