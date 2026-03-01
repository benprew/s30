package logging

import "fmt"

type Subsystem string

const (
	MTG   Subsystem = "mtg"
	World Subsystem = "world"
	Duel  Subsystem = "duel"
)

var enabled = map[Subsystem]bool{}

func Enable(s Subsystem) {
	enabled[s] = true
}

func Enabled(s Subsystem) bool {
	return enabled[s]
}

func Printf(s Subsystem, format string, args ...any) {
	if enabled[s] {
		fmt.Printf(format, args...)
	}
}
