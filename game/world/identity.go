package world

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/benprew/s30/game/domain"
)

// NewGameID returns a short, filesystem-safe random identifier for a new game.
func NewGameID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate game id: %v", err))
	}
	return hex.EncodeToString(b)
}

// SetIdentity records the choices that define this playthrough. Difficulty and
// color never change once a game starts, so together with the id they give the
// game a stable name used to group and prune its saves.
func (l *Level) SetIdentity(gameID string, difficulty domain.Difficulty, color domain.ColorMask) {
	l.GameID = gameID
	l.Difficulty = difficulty
	l.PlayerColor = color
}

// SaveName is the human-readable, stable name for this game, e.g.
// "Apprentice-Black-1a2b3c4d5e6f".
func (l *Level) SaveName() string {
	return fmt.Sprintf("%s-%s-%s",
		domain.DifficultyToString(l.Difficulty),
		domain.ColorMaskToString(l.PlayerColor),
		l.GameID)
}
