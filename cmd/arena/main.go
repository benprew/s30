package main

import (
	"flag"
	"fmt"
	"image"
	"log"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	arenaScreenWidth  = 1024
	arenaScreenHeight = 768
)

type arenaGame struct {
	screens       map[screenui.ScreenName]screenui.Screen
	current       screenui.ScreenName
	updatePointer func()
}

func (g *arenaGame) Update() error {
	g.updatePointer()
	current := g.screens[g.current]
	next, screen, err := current.Update(arenaScreenWidth, arenaScreenHeight, 1)
	if err != nil {
		return err
	}
	if next == screenui.PopScr || next == screenui.NoScr {
		return nil
	}
	if next != g.current || screen != nil {
		g.Close()
	}
	if screen != nil {
		g.screens[next] = screen
	}
	g.current = next
	if next == screenui.WorldScr || next == screenui.QuitScr {
		return ebiten.Termination
	}
	return nil
}

func (g *arenaGame) Draw(screen *ebiten.Image) {
	if current := g.screens[g.current]; current != nil {
		current.Draw(screen, arenaScreenWidth, arenaScreenHeight, 1)
	}
}

func (g *arenaGame) Layout(int, int) (int, int) {
	return arenaScreenWidth, arenaScreenHeight
}

// Close stops a duel when the launcher changes screens or exits.
func (g *arenaGame) Close() {
	if current, ok := g.screens[g.current].(interface{ Close() }); ok {
		current.Close()
	}
}

func main() {
	life := flag.Int("life", 10, "player starting life for each arena duel")
	gold := flag.Int("gold", 1000, fmt.Sprintf("player starting gold (entry costs %d)", domain.ArenaEntryCost))

	showOpponentHand := flag.Bool("show-opponent-hand", false, "reveal the opponent's hand (debug)")
	duelLog := flag.Bool("duel-log", false, "enable verbose duel logging")
	champion := flag.Bool("champion", false, "launch directly to the champion screen (debug)")
	flag.Parse()
	if *life < 1 || *gold < 0 || flag.NArg() != 0 {
		log.Fatal("-life must be positive, -gold must be nonnegative, and no positional arguments are accepted")
	}
	interactive.RevealOpponentHand = *showOpponentHand
	if *duelLog {
		logging.Enable(logging.Duel)
	}
	if _, err := domain.LoadEmbeddedCardImages(); err != nil {
		log.Fatalf("Failed to load embedded card images: %v", err)
	}
	player, err := domain.NewPlayer("Arena Test", nil, false, domain.DifficultyEasy, domain.ColorGreen)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	player.Life, player.Gold = *life, *gold
	player.CardCollection = domain.NewCardCollection()
	encounter := world.RandomEncounter{Tile: image.Pt(0, 0), Type: world.EncounterArena}
	level := &world.Level{Player: player, W: 1, H: 1,
		Tiles: [][]*world.Tile{{{}}}, RandomEncounters: []world.RandomEncounter{encounter}}
	initialScreen := screenui.Screen(screens.NewArenaEntryScreen(level, encounter))
	if *champion {
		initialScreen = screens.NewArenaChampionScreen()
	}
	g := &arenaGame{screens: map[screenui.ScreenName]screenui.Screen{
		screenui.RandomEncounterScr: initialScreen,
	}, current: screenui.RandomEncounterScr, updatePointer: ui.UpdatePointer}

	defer g.Close()
	ebiten.SetWindowSize(arenaScreenWidth, arenaScreenHeight)
	ebiten.SetWindowTitle("Arena Test")
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
	fmt.Printf("Arena ended. Gold: %d. Cards earned: %d.\n", player.Gold, player.CardCollection.NumCards())
}
