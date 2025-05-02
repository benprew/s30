package cards

import (
	"embed"
)

//go:embed testset/cards.json
var CardData embed.FS
