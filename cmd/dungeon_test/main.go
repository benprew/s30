package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	_ "git.sr.ht/~cdcarter/mage-go/cards"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

// dungeonEnemyPool returns a small set of rogue characters with sprites loaded
// so the dungeon generator has real enemies to place.
func dungeonEnemyPool(n int) []*domain.Character {
	names := make([]string, 0, len(domain.Rogues))
	for name := range domain.Rogues {
		names = append(names, name)
	}
	sort.Strings(names)
	if n > len(names) {
		n = len(names)
	}
	out := make([]*domain.Character, 0, n)
	for _, name := range names[:n] {
		c := domain.Rogues[name]
		if err := c.LoadImages(); err != nil {
			log.Fatalf("load images for %s: %v", name, err)
		}
		out = append(out, c)
	}
	return out
}

type testGame struct {
	dungeonScreen *screens.DungeonScreen
}

func (g *testGame) Update() error {
	name, _, err := g.dungeonScreen.Update(1024, 768, 1.0)
	if err != nil {
		return err
	}
	if name == screenui.WorldScr {
		fmt.Println("Left dungeon.")
		return ebiten.Termination
	}
	return nil
}

func (g *testGame) Draw(screen *ebiten.Image) {
	g.dungeonScreen.Draw(screen, 1024, 768, 1.0)
}

func (g *testGame) Layout(_, _ int) (int, int) {
	return 1024, 768
}

func main() {
	logging.Enable(logging.Duel)

	player, err := domain.NewPlayer("Test", nil, false, domain.DifficultyEasy)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}

	seed := time.Now().UnixNano()
	dungeon := domain.GenerateDungeon(domain.DungeonGenOptions{
		Name:          "Test Dungeon",
		Level:         1,
		Color:         domain.ColorRed,
		CreatureSize:  domain.CreatureSizeSmall,
		GridSize:      11,
		NumEnemies:    3,
		NumDice:       2,
		NumScrolls:    1,
		NumGoldChests: 2,
		EnemyPool:     dungeonEnemyPool(4),
		Seed:          seed,
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

	dungeonScreen := screens.NewDungeonScreen(player, lvl)

	g := &testGame{dungeonScreen: dungeonScreen}

	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("Dungeon Test")
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
