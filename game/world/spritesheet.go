package world

import (
	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
)

type SpriteSheet struct {
	Plains *ebiten.Image
	Water  *ebiten.Image
	Sand   *ebiten.Image
	Forest *ebiten.Image
	Marsh  *ebiten.Image
	Ice    *ebiten.Image
}

// LoadWorldTileSheet loads the embedded SpriteSheet.
func LoadWorldTileSheet(tileWidth, tileHeight int) (*SpriteSheet, error) {
	sheet, err := sprites.LoadSpriteSheet(6, 1, assets.Landtile_png)
	if err != nil {
		return nil, err
	}

	return &SpriteSheet{
		Plains: sheet[0][0],
		Water:  sheet[0][1],
		Sand:   sheet[0][2],
		Forest: sheet[0][3],
		Marsh:  sheet[0][4],
		Ice:    sheet[0][5],
	}, nil
}
