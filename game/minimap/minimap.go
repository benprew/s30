package minimap

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
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
	SCALE           = 1.6
	miniMapFontSize = 14 * SCALE
	doneButtonID    = "Done"
)

func NewMiniMap(l *world.Level) *MiniMap {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   miniMapFontSize,
	}

	rowSpecs := []imageutil.RowSpec{
		{Count: 48, Width: 13, Height: 13}, // Row 0: terrain
		{Count: 6, Width: 13, Height: 20},  // Row 1: city/town/castle
		{Count: 1, Width: 15, Height: 15},  // Row 2: colors
		{Count: 10, Width: 16, Height: 16}, // Row 3: characters
		{Count: 10, Width: 23, Height: 23}, // Row 4: mana symbols
	}

	terrain, err := imageutil.LoadVariableRowSpriteSheet(rowSpecs, assets.MiniMapTerrSpr_png)
	if err != nil {
		panic(fmt.Errorf("failed to load terrain sprite sheet: %w", err))
	}
	scaledTerrain := make([][]*ebiten.Image, len(terrain))
	for row := range terrain {
		scaledTerrain[row] = make([]*ebiten.Image, len(terrain[row]))
		for column := range terrain[row] {
			scaledTerrain[row][column] = imageutil.ScaleImage(terrain[row][column], SCALE)
		}
	}

	frameImg, err := imageutil.LoadImage(assets.MiniMapFrame_png)
	if err != nil {
		panic(err)
	}

	buttonsMap, err := imageutil.LoadMappedSprite(assets.MiniMapFrameSprite_png, assets.MiniMapFrameSprite_json)
	if err != nil {
		panic(err)
	}

	buttons := []*elements.Button{
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 85, 7, SCALE),
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 85+160, 7, SCALE),
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 635, 7, SCALE),
		elements.NewButton(buttonsMap["btn1_norm"], buttonsMap["btn1_hover"], buttonsMap["btn1_press"], 635+160, 7, SCALE),
	}

	buttons[0].ButtonText = elements.ButtonText{
		Text:      "World Map",
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignMiddle,
	}
	buttons[0].ID = "World Map"

	buttons[1].ButtonText = elements.ButtonText{
		Text:      "Info Map",
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignMiddle,
	}
	buttons[1].ID = "Info Map"

	buttons[2].ButtonText = elements.ButtonText{
		Text:      "City Map",
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignMiddle,
	}
	buttons[2].ID = "City Map"

	buttons[3].ButtonText = elements.ButtonText{
		Text:      doneButtonID,
		Font:      fontFace,
		TextColor: color.White,
		VAlign:    elements.AlignMiddle,
	}
	buttons[3].ID = doneButtonID
	return &MiniMap{
		terrainSprite: scaledTerrain,
		frame:         imageutil.ScaleImage(frameImg, SCALE),
		buttons:       buttons,
		level:         l,
		fontFace:      fontFace,
	}
}

func (m *MiniMap) IsFramed() bool {
	return false
}

func (m *MiniMap) IsOverlay() bool { return true }

func (m *MiniMap) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.DrawImage(m.frame, &ebiten.DrawImageOptions{})

	for _, b := range m.buttons {
		b.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}

	xref := map[int][2]int{
		world.TerrainWater:     {0, 0},
		world.TerrainForest:    {0, 2},
		world.TerrainMarsh:     {0, 3},
		world.TerrainMountains: {0, 5},
		world.TerrainSand:      {0, 6},
		world.TerrainPlains:    {0, 18},
	}
	options := &ebiten.DrawImageOptions{}
	city := m.terrainSprite[1][1]
	castle := m.terrainSprite[1][2]
	pLoc := m.level.CharacterTile()
	player := m.terrainSprite[3][0]
	width := m.terrainSprite[0][0].Bounds().Dx() - 1
	height := m.terrainSprite[0][0].Bounds().Dy() - 1
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

			if col.IsCity() {
				cOpts := &ebiten.DrawImageOptions{}
				cOpts.GeoM.Concat(opts.GeoM)
				cOpts.GeoM.Translate(0, -13)
				screen.DrawImage(city, cOpts)
			}
			if col.IsCastle && col.Castle != nil {
				cOpts := &ebiten.DrawImageOptions{}
				cOpts.GeoM.Concat(opts.GeoM)
				cOpts.GeoM.Translate(0, -13)
				if col.Castle.Defeated {
					cOpts.ColorScale.ScaleAlpha(0.4)
				}
				screen.DrawImage(castle, cOpts)
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

	// draw minimap image overlays (castles & cities)
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

			if col.IsCity() && col.City.Name != "" {
				cityNameLines := strings.ReplaceAll(col.City.Name, " ", "\n")
				cityText := elements.NewText(miniMapFontSize, cityNameLines, 0, 8)
				cityText.LineSpacing = float64(m.fontFace.Size)
				textWidth, _ := cityText.Measure()
				cityText.X = -int(textWidth / 2)
				if m.isQuestTarget(col.City.Name) && m.blinkCounter%10 < 7 {
					cityText.Color = color.RGBA{R: 255, G: 215, B: 0, A: 255}
				}
				cityText.Draw(screen, opts, 1.0)
			}
			if col.IsCastle && col.Castle != nil {
				label := domain.ColorMaskToString(col.Castle.Color) + "\nCastle"
				castleText := elements.NewText(miniMapFontSize, label, 0, 8)
				castleText.LineSpacing = float64(m.fontFace.Size)
				textWidth, _ := castleText.Measure()
				castleText.X = -int(textWidth / 2)
				castleText.Draw(screen, opts, 1.0)
			}
		}
	}
}

func (m *MiniMap) isQuestTarget(cityName string) bool {
	for _, q := range m.level.Player.ActiveQuests {
		if q.Type == domain.QuestTypeDelivery && q.TargetCity != nil && q.TargetCity.Name == cityName {
			return true
		}
	}
	return false
}

func (m *MiniMap) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	m.blinkCounter++

	options := &ebiten.DrawImageOptions{}

	for i := range m.buttons {
		b := m.buttons[i]
		b.Update(options, scale, W, H)
		if b.ID == doneButtonID && b.IsClicked() {
			return screenui.PopScr, nil, nil
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return screenui.PopScr, nil, nil
	}
	return screenui.MiniMapScr, nil, nil
}
