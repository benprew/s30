package minimap

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets/art"
	"github.com/benprew/s30/assets/fonts"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type MiniMap struct {
	terrainSprite [][]*ebiten.Image
	frame         *ebiten.Image
	frameSprite   []*ebiten.Image
	buttons       []*elements.Button
}

const (
	SCALE = 1.6
)

func NewMiniMap() MiniMap {
	// Create a font face using ebiten's text v2
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.Magic_ttf))
	if err != nil {
		panic(fmt.Errorf("failed to create font source: %w", err))
	}

	fontFace := &text.GoTextFace{
		Source: fontSource,
		Size:   14,
	}

	s, err := sprites.LoadSpriteSheet(75, 1, art.MiniMapTerrSpr_png)
	if err != nil {
		panic(fmt.Errorf("failed to load terrain sprite sheet: %w", err))
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

	buttons := []*elements.Button{
		&elements.Button{
			Normal:     frameSprite[0],
			Hover:      frameSprite[1],
			Pressed:    frameSprite[2],
			Text:       "World Map",
			Font:       fontFace,
			TextColor:  color.White,
			TextOffset: image.Point{X: 25, Y: 14},
			State:      elements.StateNormal,
			X:          85,
			Y:          7,
		},
		&elements.Button{
			Normal:     frameSprite[0],
			Hover:      frameSprite[1],
			Pressed:    frameSprite[2],
			Text:       "Info Map",
			Font:       fontFace,
			TextColor:  color.White,
			TextOffset: image.Point{X: 25, Y: 14},
			State:      elements.StateNormal,
			X:          85 + 160,
			Y:          7,
		},
		&elements.Button{
			Normal:     frameSprite[0],
			Hover:      frameSprite[1],
			Pressed:    frameSprite[2],
			Text:       "City Map",
			Font:       fontFace,
			TextColor:  color.White,
			TextOffset: image.Point{X: 40, Y: 14},
			State:      elements.StateNormal,
			X:          635,
			Y:          7,
		},
		&elements.Button{
			Normal:     frameSprite[0],
			Hover:      frameSprite[1],
			Pressed:    frameSprite[2],
			Text:       "Done",
			Font:       fontFace,
			TextColor:  color.White,
			TextOffset: image.Point{X: 60, Y: 14},
			State:      elements.StateNormal,
			X:          635 + 160,
			Y:          7,
		},
	}
	return MiniMap{
		terrainSprite: s,
		frame:         ebiten.NewImageFromImage(img),
		frameSprite:   frameSprite,
		buttons:       buttons,
	}
}

func (m *MiniMap) Draw(screen *ebiten.Image, scale float64, l *world.Level) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	options.GeoM.Scale(SCALE, SCALE)   // scale up from 640x480
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
	width := int(float64(m.terrainSprite[0][0].Bounds().Dx())*SCALE) - 1
	height := int(float64(m.terrainSprite[0][0].Bounds().Dy())*SCALE) - 1
	//Draw level from T
	for i, row := range l.Tiles {
		offset := 0
		if i%2 == 1 {
			offset = width / 2
		}

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Concat(options.GeoM)
		opts.GeoM.Translate(50, 100)
		opts.GeoM.Translate(float64(offset), float64(height*i)/2)
		for j, col := range row {
			sprite := m.terrainSprite[0][xref[col.TerrainType]]
			opts.GeoM.Translate(float64(width), 0)
			screen.DrawImage(sprite, opts)

			if col.IsCity {
				screen.DrawImage(city, opts)
			}
			p := world.TilePoint{X: j, Y: i}
			if pLoc == p {
				screen.DrawImage(player, opts)
			}
		}
	}
}

func (m *MiniMap) Update() (bool, error) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(SCALE, SCALE) // scale up from 640x480

	donePressed := false
	for _, b := range m.buttons {
		b.Update(options)
		if b.Text == "Done" && b.State == elements.StatePressed {
			donePressed = true
		}
	}
	return donePressed, nil
}
