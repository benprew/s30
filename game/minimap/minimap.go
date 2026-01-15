package minimap

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type MiniMap struct {
	terrainSprite [][]*ebiten.Image
	frame         *ebiten.Image
	buttons       []*elements.Button
	level         *world.Level
	blinkCounter  int
	fontFace      *text.GoTextFace
}

const (
	SCALE = 1.6
)

func NewMiniMap(l *world.Level) *MiniMap {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   14,
	}

	rowSpecs := []imageutil.RowSpec{
		{Count: 48, Width: 13, Height: 13}, // Row 0: terrain
		{Count: 6, Width: 13, Height: 20},  // Row 1: city/town/castle
		{Count: 1, Width: 15, Height: 15},  // Row 2: colors
		{Count: 10, Width: 16, Height: 16}, // Row 3: characters
		{Count: 10, Width: 23, Height: 23}, // Row 4: mana symbols
	}

	s, err := imageutil.LoadVariableRowSpriteSheet(rowSpecs, assets.MiniMapTerrSpr_png)
	if err != nil {
		panic(fmt.Errorf("failed to load terrain sprite sheet: %w", err))
	}

	frameImg, err := imageutil.LoadImage(assets.MiniMapFrame_png)
	if err != nil {
		panic(err)
	}

	buttonsMap, err := imageutil.LoadMappedSprite(assets.MiniMapFrameSprite_png, assets.MiniMapFrameSprite_json)
	if err != nil {
		panic(err)
	}

	scale := 1.0
	buttons := []*elements.Button{
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 85, 7, scale),
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 85+160, 7, scale),
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 635, 7, scale),
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 635+160, 7, scale),
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
		buttons:       buttons,
		level:         l,
		fontFace:      fontFace,
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

	xref := map[int][2]int{
		world.TerrainWater:     {0, 0},
		world.TerrainForest:    {0, 2},
		world.TerrainMarsh:     {0, 3},
		world.TerrainMountains: {0, 5},
		world.TerrainSand:      {0, 6},
		world.TerrainPlains:    {0, 18},
	}
	city := m.terrainSprite[1][1]
	pLoc := m.level.CharacterTile()
	player := m.terrainSprite[3][0]
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
			spriteCoord := xref[col.TerrainType]
			sprite := m.terrainSprite[spriteCoord[0]][spriteCoord[1]]
			opts.GeoM.Translate(float64(width), 0)
			screen.DrawImage(sprite, opts)

			if col.IsCity {
				cOpts := &ebiten.DrawImageOptions{}
				cOpts.GeoM.Concat(opts.GeoM)
				cOpts.GeoM.Translate(0, -13)
				screen.DrawImage(city, cOpts)
			}
			p := image.Point{X: j, Y: i}
			if pLoc == p && m.blinkCounter%10 < 7 {
				cOpts := &ebiten.DrawImageOptions{}
				cOpts.GeoM.Concat(opts.GeoM)
				cOpts.GeoM.Translate(0, -13)
				screen.DrawImage(player, cOpts)
			}
		}
	}

	for i, row := range m.level.Tiles {
		offset := 0
		if i%2 == 1 {
			offset = width / 2
		}

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Concat(options.GeoM)
		opts.GeoM.Translate(50, 100)
		opts.GeoM.Translate(float64(offset), float64(height*i)/2)
		for _, col := range row {
			opts.GeoM.Translate(float64(width), 0)

			if col.IsCity && col.City.Name != "" {
				cityNameLines := strings.Replace(col.City.Name, " ", "\n", -1)

				textWidth, _ := text.Measure(cityNameLines, m.fontFace, 0)
				textOp := &text.DrawOptions{}
				textOp.GeoM.Concat(opts.GeoM)
				textOp.GeoM.Translate(-float64(textWidth)/2, 8)
				textOp.ColorScale.ScaleWithColor(color.White)
				textOp.LineSpacing = m.fontFace.Size
				text.Draw(screen, cityNameLines, m.fontFace, textOp)
			}
		}
	}
}

func (m *MiniMap) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	m.blinkCounter++

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(SCALE, SCALE) // scale up from 640x480

	for i := range m.buttons {
		b := m.buttons[i]
		b.Update(options, scale, W, H)
		if b.ID == "Done" && b.IsClicked() {
			return screenui.WorldScr, nil, nil
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil, nil
	}
	return screenui.MiniMapScr, nil, nil
}
