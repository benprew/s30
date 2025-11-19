package minimap

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type MiniMap struct {
	terrainSprite [][]*ebiten.Image
	frame         *ebiten.Image
	frameSprite   []*ebiten.Image
	buttons       []*elements.Button
	level         *world.Level
}

const (
	SCALE = 1.6
)

func NewMiniMap(l *world.Level) *MiniMap {
	// Create a font face using ebiten's text v2
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Magic_ttf))
	if err != nil {
		panic(fmt.Errorf("failed to create font source: %w", err))
	}

	fontFace := &text.GoTextFace{
		Source: fontSource,
		Size:   14,
	}

	s, err := sprites.LoadSpriteSheet(75, 1, assets.MiniMapTerrSpr_png)
	if err != nil {
		panic(fmt.Errorf("failed to load terrain sprite sheet: %w", err))
	}

	frameImg, err := elements.LoadImage(assets.MiniMapFrame_png)
	if err != nil {
		panic(err)
	}

	sprInfo, err := sprites.LoadSprInfoFromJSON(assets.MiniMapFrameSprite_json)
	if err != nil {
		panic(err)
	}

	frameSprite, err := sprites.LoadSubimages(assets.MiniMapFrameSprite_png, &sprInfo)
	if err != nil {
		panic(err)
	}

	buttons := []*elements.Button{
		elements.NewButton(frameSprite[0], frameSprite[1], frameSprite[2], 85, 7, SCALE),
		elements.NewButton(frameSprite[0], frameSprite[1], frameSprite[2], 85+160, 7, SCALE),
		elements.NewButton(frameSprite[0], frameSprite[1], frameSprite[2], 635, 7, SCALE),
		elements.NewButton(frameSprite[0], frameSprite[1], frameSprite[2], 635+160, 7, SCALE),
	}

	buttons[0].ButtonText = elements.ButtonText{
		Text:      "World Map",
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignBottom,
	}
	buttons[0].ID = "World Map"

	buttons[1].ButtonText = elements.ButtonText{
		Text:      "Info Map",
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignBottom,
	}
	buttons[1].ID = "Info Map"

	buttons[2].ButtonText = elements.ButtonText{
		Text:      "City Map",
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignBottom,
	}
	buttons[2].ID = "City Map"

	buttons[3].ButtonText = elements.ButtonText{
		Text:      "Done",
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignBottom,
	}
	buttons[3].ID = "Done"
	return &MiniMap{
		terrainSprite: s,
		frame:         frameImg,
		frameSprite:   frameSprite,
		buttons:       buttons,
		level:         l,
	}
}

func (m *MiniMap) IsFramed() bool {
	return false
}

func (m *MiniMap) Draw(screen *ebiten.Image, W, H int, scale float64) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	options.GeoM.Scale(SCALE, SCALE)   // scale up from 640x480
	screen.DrawImage(m.frame, options) // draw background frame

	for _, b := range m.buttons {
		b.Draw(screen, options, scale)
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
	pLoc := m.level.CharacterTile()
	player := m.terrainSprite[0][54]
	width := int(float64(m.terrainSprite[0][0].Bounds().Dx())*SCALE) - 1
	height := int(float64(m.terrainSprite[0][0].Bounds().Dy())*SCALE) - 1
	//Draw level from T
	for i, row := range m.level.Tiles {
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

func (m *MiniMap) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(SCALE, SCALE) // scale up from 640x480

	for i := range m.buttons {
		b := m.buttons[i]
		b.Update(options, scale, W, H)
		if b.ID == "Done" && b.IsClicked() {
			return screenui.WorldScr, nil
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil
	}
	return screenui.MiniMapScr, nil
}
