package save

import (
	"time"

	"github.com/benprew/s30/game/world"
)

type SaveData struct {
	Name    string       `json:"name"`
	Version int          `json:"version"`
	SavedAt time.Time    `json:"saved_at"`
	World   *world.Level `json:"world"`
}
