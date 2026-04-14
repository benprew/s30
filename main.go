package main

import (
	"flag"
	"log"
	"strings"

	"github.com/benprew/s30/game"
	"github.com/benprew/s30/internal/pprofutil"
	"github.com/benprew/s30/logging"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	verbose := flag.String("v", "", "enable verbose logging for subsystems (comma-separated: mtg,world,duel)")
	pprofAddr := flag.String("pprof", "", "enable pprof HTTP server at the given listen address, e.g. 127.0.0.1:6060")
	flag.Parse()

	if *verbose != "" {
		for _, s := range strings.Split(*verbose, ",") {
			logging.Enable(logging.Subsystem(strings.TrimSpace(s)))
		}
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
	ebiten.SetTPS(10)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	// ebiten.SetFullscreen(true)

	g, err := game.NewGame()
	if err != nil {
		log.Fatal(err)
	}

	if err = ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
