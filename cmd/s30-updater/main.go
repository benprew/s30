package main

import (
	"flag"
	"log"

	"github.com/benprew/s30/game/updates"
)

func main() {
	plan := flag.String("plan", "", "path to the verified update installation plan")
	flag.Parse()
	if *plan == "" {
		log.Fatal("an installation plan is required")
	}
	if err := updates.Apply(*plan); err != nil {
		log.Fatal(err)
	}
}
