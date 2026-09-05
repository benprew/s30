package world

import "image"

func (l *Level) rebuildTerrainTransitions(sa *spriteAssets) error {
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			tile := l.Tile(image.Point{X: x, Y: y})
			if tile != nil {
				tile.transitionSprites = nil
			}
		}
	}

	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			position := image.Point{X: x, Y: y}
			tile := l.Tile(position)
			if tile == nil {
				continue
			}
			waterMask, marshMask := originalTerrainTransitionMasks(l, position)
			for _, transition := range []struct {
				source TileType
				mask   uint8
			}{
				{source: TileMarsh, mask: marshMask},
				{source: TileWater, mask: waterMask},
			} {
				for _, ref := range cornerTransitionPlan(transition.source, transition.mask) {
					if ref.Row < 0 || ref.Row >= len(sa.cstline1) ||
						ref.Col < 0 || ref.Col >= len(sa.cstline1[ref.Row]) {
						continue
					}
					tile.AddTransitionSprite(sa.cstline1[ref.Row][ref.Col], ref.OffsetX, ref.OffsetY)
				}
			}
		}
	}
	return nil
}

type transitionSpriteBank struct {
	sheet      int
	flatOffset int
}

func transitionSpriteBankFor(terrain TileType) (transitionSpriteBank, bool) {
	switch terrain {
	case TileWater:
		return transitionSpriteBank{sheet: 1, flatOffset: 0}, true
	case TileMarsh:
		return transitionSpriteBank{sheet: 1, flatOffset: 28}, true
	case TileForest:
		return transitionSpriteBank{sheet: 1, flatOffset: 56}, true
	case TileSand:
		return transitionSpriteBank{sheet: 2, flatOffset: 0}, true
	case TileIce:
		return transitionSpriteBank{sheet: 2, flatOffset: 28}, true
	default:
		return transitionSpriteBank{}, false
	}
}

type transitionSpriteRef struct {
	Row int
	Col int
}

type positionedTransitionRef struct {
	transitionSpriteRef
	Sheet   int
	OffsetX float64
	OffsetY float64
}

func originalTerrainTransitionMasks(level *Level, position image.Point) (waterMask, marshMask uint8) {
	current := level.Tile(position)
	if current == nil {
		return 0, 0
	}
	currentBand := current.TerrainBand
	if currentBand == TerrainBandWater || currentBand == TerrainBandSandMarsh {
		return 0, 0
	}
	for index, direction := range originalTerrainDirections() {
		neighbor := level.Tile(position.Add(direction))
		if neighbor == nil {
			continue
		}
		neighborBand := neighbor.TerrainBand
		if neighborBand == TerrainBandWater || neighborBand == TerrainBandSandMarsh ||
			(currentBand != neighborBand && (isCoastTerrainBand(currentBand) || isCoastTerrainBand(neighborBand))) {
			waterMask |= 1 << index
		}
		if currentBand != TerrainBandMarsh && neighborBand == TerrainBandMarsh {
			marshMask |= 1 << index
		}
	}
	return waterMask, marshMask
}

func originalTerrainDirections() [8]image.Point {
	return [8]image.Point{
		{X: 0, Y: -1},
		{X: 1, Y: -1},
		{X: 1, Y: 0},
		{X: 1, Y: 1},
		{X: 0, Y: 1},
		{X: -1, Y: 1},
		{X: -1, Y: 0},
		{X: -1, Y: -1},
	}
}

func isCoastTerrainBand(band TerrainBand) bool {
	return band == TerrainBandSand || band == TerrainBandSandMarsh
}

func cornerTransitionPatterns(mask uint8) [4]int {
	wrapped := uint16(mask) | uint16(mask)<<8
	return [4]int{
		int(mask & 7),
		int(mask>>4) & 7,
		int(wrapped>>6) & 7,
		int(mask>>2) & 7,
	}
}

func cornerTransitionPlan(sourceType TileType, mask uint8) []positionedTransitionRef {
	bank, ok := transitionSpriteBankFor(sourceType)
	if !ok || mask == 0 {
		return nil
	}
	patterns := cornerTransitionPatterns(mask)
	positions := [4][2]float64{
		{52, 0},
		{52, 100},
		{-51, 50},
		{154, 50},
	}
	result := make([]positionedTransitionRef, 0, 4)
	for corner, pattern := range patterns {
		if pattern == 0 {
			continue
		}
		flatIndex := bank.flatOffset + corner*7 + pattern - 1
		result = append(result, positionedTransitionRef{
			transitionSpriteRef: transitionSpriteRef{Row: flatIndex / 4, Col: flatIndex % 4},
			Sheet:               bank.sheet,
			OffsetX:             positions[corner][0],
			OffsetY:             positions[corner][1],
		})
	}
	return result
}

func expectedTerrainTransitionCount(level *Level, position image.Point) int {
	waterMask, marshMask := originalTerrainTransitionMasks(level, position)
	return len(cornerTransitionPlan(TileMarsh, marshMask)) + len(cornerTransitionPlan(TileWater, waterMask))
}
