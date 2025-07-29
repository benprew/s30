package card_info

import (
	_ "embed"
)

var (
	//go:embed scryfall_cards.json.zst
	Cards_json []byte
)
