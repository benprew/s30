package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/benprew/s30/game"
	"github.com/benprew/s30/internal/pprofutil"
	"github.com/benprew/s30/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

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
	verbose := flag.String("v", "", "enable verbose logging for subsystems (comma-separated: mtg,world,duel)")
	pprofAddr := flag.String("pprof", "", "enable pprof HTTP server at the given listen address, e.g. 127.0.0.1:6060")
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to file on exit")
	memprofile := flag.String("memprofile", "", "write retained heap profile to file on exit")
	allocprofile := flag.String("allocprofile", "", "write cumulative allocation profile to file on exit")
	memProfileRate := flag.Int("memprofilerate", runtime.MemProfileRate, "bytes allocated per heap-profile sample (1 records every allocation)")
	debug := flag.Bool("debug", false, "use the debug burn deck and start enemies at 1 life")
	showOpponentHand := flag.Bool("show-opponent-hand", false, "reveal the opponent's hand")
	flag.Parse()

	if *verbose != "" {
		for s := range strings.SplitSeq(*verbose, ",") {
			logging.Enable(logging.Subsystem(strings.TrimSpace(s)))
		}
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

	pprofLn, err := pprofutil.Start(*pprofAddr, log.Printf)
	if err != nil {
		log.Fatal(err)
	}
	if pprofLn != nil {
		defer pprofLn.Close()
	}

	ebiten.SetWindowTitle("Shandalar 30")
	// ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	// ebiten.SetFullscreen(true)

	g, err := game.NewGameWithOptions(game.Options{
		Debug:            *debug,
		ShowOpponentHand: *showOpponentHand,
	})
	if err != nil {
		log.Fatal(err)
	}
	registerSaveLifecycle(g)

	if err = ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}

	if err := writeProfile(*allocprofile, "allocs", false); err != nil {
		log.Fatalf("Failed to write allocation profile: %v", err)
	}
	if err := writeProfile(*memprofile, "heap", true); err != nil {
		log.Fatalf("Failed to write heap profile: %v", err)
	}
}
