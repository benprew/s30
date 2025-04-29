package minimap

import (
	"bytes"
	"image"

	"github.com/benprew/s30/art"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
)

type MiniMap struct {
	terrainSprite [][]*ebiten.Image
	frame         *ebiten.Image
	frameSprite   []*ebiten.Image
	buttons       []*sprites.Button
}

func NewMiniMap() MiniMap {
	// data :=
	// data, err := art.MiniMapFS.ReadFile("Ttsprite.spr.png")
	// if err != nil {
	// 	panic(err)
	// }
	s, err := sprites.LoadSpriteSheet(75, 1, art.MiniMapTerrSpr_png)
	if err != nil {
		panic(err)
	}

	img, _, err := image.Decode(bytes.NewReader(art.MiniMapFrame_png))
	if err != nil {
		panic(err)
	}

	sprInfo, err := sprites.LoadSprInfoFromJSON(art.MiniMapFrameSprite_json)
	if err != nil {
		panic(err)
	}

	frameSprite, err := sprites.LoadSubimages(art.MiniMapFrameSprite_png, &sprInfo)
	if err != nil {
		panic(err)
	}

	buttons := []*sprites.Button{
		&sprites.Button{
			Normal:  frameSprite[0],
			Hover:   frameSprite[1],
			Pressed: frameSprite[2],
			Text:    "Plain Map",
			// font:    Mmtg,
			//		textColor: color.Color{:write},
			State: sprites.StateNormal,
			X:     85,
			Y:     7,
		},
		&sprites.Button{
			Normal:  frameSprite[0],
			Hover:   frameSprite[1],
			Pressed: frameSprite[2],
			Text:    "Info Map",
			// font:    Mmtg,
			//		textColor: color.Color{:write},
			State: sprites.StateNormal,
			X:     85 + 160,
			Y:     7,
		},
		&sprites.Button{
			Normal:  frameSprite[0],
			Hover:   frameSprite[1],
			Pressed: frameSprite[2],
			Text:    "City Map",
			// font:    Mmtg,
			//		textColor: color.Color{:write},
			State: sprites.StateNormal,
			X:     635,
			Y:     7,
		},
		&sprites.Button{
			Normal:  frameSprite[0],
			Hover:   frameSprite[1],
			Pressed: frameSprite[2],
			Text:    "Done",
			// font:    Mmtg,
			//		textColor: color.Color{:write},
			State: sprites.StateNormal,
			X:     635 + 160,
			Y:     7,
		},
	}
	return MiniMap{
		terrainSprite: s,
		frame:         ebiten.NewImageFromImage(img),
		frameSprite:   frameSprite, buttons: buttons,
	}
}

func (m *MiniMap) Draw(screen *ebiten.Image, scale float64, l *world.Level) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	options.GeoM.Scale(1.6, 1.6)       // scale up from 640x480
	screen.DrawImage(m.frame, options) // draw background frame

	for _, b := range m.buttons {
		b.Draw(screen, options)
	}

	xref := map[int]int{
		world.TerrainWater:     0,
		world.TerrainForest:    2,
		world.TerrainMarsh:     3,
		world.TerrainMountains: 5,
		world.TerrainSand:      6,
		world.TerrainPlains:    18,
	}
	city := m.terrainSprite[0][49]
	pLoc := l.CharacterTile()
	player := m.terrainSprite[0][54]
	width := m.terrainSprite[0][0].Bounds().Dx() - 1
	height := m.terrainSprite[0][0].Bounds().Dy() - 1
	//Draw level from T
	for i, row := range l.Tiles {
		offset := 0
		if i%2 == 1 {
			offset = width / 2
		}

		options := &ebiten.DrawImageOptions{}
		options.GeoM.Scale(scale, scale)
		options.GeoM.Translate(125, 150)
		options.GeoM.Translate(float64(offset), float64(height*i)/2)
		for j, col := range row {
			sprite := m.terrainSprite[0][xref[col.TerrainType]]
			options.GeoM.Translate(float64(width), 0)
			screen.DrawImage(sprite, options)

			if col.IsCity {
				screen.DrawImage(city, options)
			}
			p := world.TilePoint{X: j, Y: i}
			if pLoc == p {
				screen.DrawImage(player, options)
			}
		}
	}

}

func (m *MiniMap) Update() error {
	return nil
}
