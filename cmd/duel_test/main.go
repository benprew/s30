package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"

	_ "github.com/benprew/mage-go/cards"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/imageutil"
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

	add("Fireball", 4)
	add("Earthquake", 2)
	add("Lightning Bolt", 4)
	add("Rod of Ruin", 3)
	add("Jade Statue", 2)
	add("Kird Ape", 4)
	add("Fire Elemental", 3)
	add("Sol Ring", 2)
	add("Mishra's Factory", 3)
	add("Mountain", 14)
	add("Forest", 6)
	// add("Wall of Brambles", 20)
	// add("Llanowar Elves", 10)
	// add("Forest", 30)

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
	duelScreen    *screens.DuelScreen
	newDuel       func() (*screens.DuelScreen, error)
	maxFrames     int
	frames        int
	targetDuels   int
	finishedDuel  int
	memstatEvery  int
	lastMemstat   int
	memoryReport  string
	memorySamples []duelMemorySample
	memoryStart   runtime.MemStats
}

type duelMemorySample struct {
	Duel           int
	AllocatedBytes uint64
	Allocations    uint64
	HeapBytes      uint64
	Goroutines     int
}

func (g *testGame) beginMemorySample() {
	runtime.GC()
	runtime.ReadMemStats(&g.memoryStart)
}

func (g *testGame) finishMemorySample() {
	g.duelScreen.Close()
	g.duelScreen = nil
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	g.memorySamples = append(g.memorySamples, duelMemorySample{
		Duel:           g.finishedDuel,
		AllocatedBytes: stats.TotalAlloc - g.memoryStart.TotalAlloc,
		Allocations:    stats.Mallocs - g.memoryStart.Mallocs,
		HeapBytes:      stats.HeapAlloc,
		Goroutines:     runtime.NumGoroutine(),
	})
}

func (g *testGame) Update() error {
	ui.UpdatePointer()
	g.frames++
	if g.maxFrames > 0 && g.frames > g.maxFrames {
		return ebiten.Termination
	}
	if g.memstatEvery > 0 && g.frames%g.memstatEvery == 0 {
		g.reportMemoryStats()
	}

	name, _, err := g.duelScreen.Update(1024, 768, 1.0)
	if err != nil {
		return err
	}
	if name == screenui.DuelWinScr || name == screenui.DuelLoseScr {
		g.finishedDuel++
		if name == screenui.DuelWinScr {
			fmt.Printf("Duel %d: player AI won\n", g.finishedDuel)
		} else {
			fmt.Printf("Duel %d: opponent AI won\n", g.finishedDuel)
		}
		g.reportMemoryStats()
		if g.memoryReport != "" {
			g.finishMemorySample()
		} else {
			g.duelScreen.Close()
		}
		if g.finishedDuel >= g.targetDuels {
			return ebiten.Termination
		}
		if g.memoryReport != "" {
			g.beginMemorySample()
		}
		g.duelScreen, err = g.newDuel()
		return err
	}
	return nil
}

func (g *testGame) Draw(screen *ebiten.Image) {
	if g.duelScreen == nil { // needed for finishMemorySample
		return
	}
	g.duelScreen.Draw(screen, 1024, 768, 1.0)
}

func (g *testGame) Layout(_, _ int) (int, int) {
	return 1024, 768
}

func (g *testGame) reportMemoryStats() {
	if g.lastMemstat == g.frames {
		return
	}
	printMemoryStats(g.frames, g.finishedDuel)
	g.lastMemstat = g.frames
}

func residentSetBytes() uint64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return pages * uint64(os.Getpagesize())
}

func printMemoryStats(frame, duel int) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	cardImages, labeledCards := domain.CardImageCacheStats()
	fmt.Printf("MEM frame=%d duels=%d heap_alloc=%d heap_inuse=%d total_alloc=%d mallocs=%d frees=%d live_objects=%d sys=%d rss=%d gc=%d image_registry=%d card_images=%d labeled_cards=%d\n",
		frame, duel, stats.HeapAlloc, stats.HeapInuse, stats.TotalAlloc, stats.Mallocs, stats.Frees, stats.Mallocs-stats.Frees,
		stats.Sys, residentSetBytes(), stats.NumGC, imageutil.RegistryLen(), cardImages, labeledCards)
}

func writeProfile(name, profileName string, gc bool) error {
	if name == "" {
		return nil
	}
	if gc {
		runtime.GC()
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	profile := pprof.Lookup(profileName)
	if profile == nil {
		return fmt.Errorf("profile %q is unavailable", profileName)
	}
	return profile.WriteTo(f, 0)
}

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to file")
	memprofile := flag.String("memprofile", "", "write retained heap profile to file")
	allocprofile := flag.String("allocprofile", "", "write cumulative allocation profile to file")
	memProfileRate := flag.Int("memprofilerate", runtime.MemProfileRate, "bytes allocated per heap-profile sample (1 records every allocation)")
	profileFrames := flag.Int("profileframes", 0, "terminate after this many update frames")
	memstatFrames := flag.Int("memstatframes", 600, "print memory and image-cache statistics every N frames (0 disables)")
	duels := flag.Int("duels", 1, "number of automated duels to run")
	autoplay := flag.Bool("autoplay", false, "let an AI drive the player seat")
	rogue := flag.String("rogue", "", "fight this rogue instead of picking randomly")
	showOpponentHand := flag.Bool("show-opponent-hand", false, "reveal the opponent's hand (debug)")
	aiTestDeck := flag.Bool("ai-test-deck", false, "AI opponent plays xTestDeck() instead of its rogue deck")
	duelLog := flag.Bool("duel-log", false, "enable verbose duel logging (skews allocation profiles)")
	loadCardImages := flag.Bool("load-card-images", true, "load embedded card art before profiling")
	headless := flag.Bool("headless", false, "run without opening a visible window")
	fast := flag.Bool("fast", false, "run up to 1000 updates per second for automated checks")
	memoryReport := flag.String("memory-report", "", "write per-duel allocation and post-GC heap counters as JSON (forces GC between duels)")
	flag.Parse()
	if *duels < 1 {
		log.Fatal("-duels must be at least 1")
	}
	if *memProfileRate < 1 {
		log.Fatal("-memprofilerate must be at least 1")
	}

	interactive.RevealOpponentHand = *showOpponentHand
	if *loadCardImages {
		loaded, err := domain.LoadEmbeddedCardImages()
		if err != nil {
			log.Fatalf("Failed to load embedded card images: %v", err)
		}
		fmt.Printf("Loaded %d embedded card images\n", loaded)
	}

	if *memprofile != "" || *allocprofile != "" {
		runtime.MemProfileRate = *memProfileRate
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

	if *duelLog {
		logging.Enable(logging.Duel)
	}

	rogueName := *rogue
	if rogueName == "" {
		rogueName = pickRandomRogue()
	}
	fmt.Printf("Fighting: %s (duels=%d autoplay=%v)\n", rogueName, *duels, *autoplay)

	newDuel := func() (*screens.DuelScreen, error) {
		enemy, err := domain.NewEnemy(rogueName)
		if err != nil {
			return nil, fmt.Errorf("create enemy %s: %w", rogueName, err)
		}
		if *aiTestDeck {
			enemyCharacter := *enemy.Character
			enemyCharacter.CardCollection = domain.NewCardCollection()
			for card, count := range xTestDeck() {
				enemyCharacter.CardCollection.AddCardToDeck(card, 0, count)
			}
			enemy.Character = &enemyCharacter
		}

		player, err := domain.NewPlayer("Test", nil, false, domain.DifficultyEasy, domain.ColorColorless)
		if err != nil {
			return nil, fmt.Errorf("create player: %w", err)
		}
		if !*autoplay {
			player.Life = 999
		}
		player.CardCollection = domain.NewCardCollection()
		for card, count := range xTestDeck() {
			player.CardCollection.AddCardToDeck(card, 0, count)
		}

		lvl := &world.Level{Player: player, Enemies: []domain.Enemy{enemy}}
		duelScreen := screens.NewDuelScreen(player, &enemy, lvl, 0, nil, nil)
		if *autoplay {
			duelScreen.EnableAutoPlay()
		}
		return duelScreen, nil
	}

	g := &testGame{
		memoryReport: *memoryReport,
		newDuel:      newDuel,
		maxFrames:    *profileFrames,
		targetDuels:  *duels,
		memstatEvery: *memstatFrames,
		lastMemstat:  -1,
	}
	if g.memoryReport != "" {
		g.beginMemorySample()
	}
	var err error
	g.duelScreen, err = newDuel()
	if err != nil {
		log.Fatal(err)
	}
	g.reportMemoryStats()
	defer func() {
		if g.duelScreen != nil {
			g.duelScreen.Close()
		}
	}()

	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("Duel Test - X Spells")
	if *headless {
		ebiten.SetWindowVisible(false)
	}
	if *fast {
		ebiten.SetTPS(1000)
		ebiten.SetVsyncEnabled(false)
	}
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}

	g.reportMemoryStats()
	if g.memoryReport != "" {
		data, err := json.MarshalIndent(g.memorySamples, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(g.memoryReport, data, 0o600); err != nil {
			log.Fatal(err)
		}
		if g.finishedDuel != g.targetDuels {
			log.Fatalf("completed %d of %d duels", g.finishedDuel, g.targetDuels)
		}
	}
	if err := writeProfile(*allocprofile, "allocs", false); err != nil {
		log.Fatalf("Failed to write allocation profile: %v", err)
	}
	if err := writeProfile(*memprofile, "heap", true); err != nil {
		log.Fatalf("Failed to write heap profile: %v", err)
	}
}
