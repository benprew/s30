package world

import (
	"image"
	"math/rand"
)

// TerrainBand records the visual elevation band used to build a tile.
// Transitional bands use Land2 and Sland2 artwork.
type TerrainBand int

const (
	TerrainBandWater TerrainBand = iota
	TerrainBandSand
	TerrainBandSandMarsh
	TerrainBandMarsh
	TerrainBandMarshPlains
	TerrainBandPlains
	TerrainBandPlainsForest
	TerrainBandForest
	TerrainBandForestMountains
	TerrainBandMountains
	TerrainBandIce
)

type terrainBase int

const (
	terrainBaseWater terrainBase = iota
	terrainBaseSand
	terrainBaseMarsh
	terrainBasePlains
	terrainBaseIce
)

type terrainBandProperties struct {
	terrain   int
	base      terrainBase
	column    int
	secondary bool
}

func (b TerrainBand) properties() terrainBandProperties {
	switch b {
	case TerrainBandWater:
		return terrainBandProperties{terrain: TerrainWater, base: terrainBaseWater, column: -1}
	case TerrainBandSand:
		return terrainBandProperties{terrain: TerrainSand, base: terrainBaseSand, column: 1}
	case TerrainBandSandMarsh:
		return terrainBandProperties{terrain: TerrainSand, base: terrainBaseWater, column: 0, secondary: true}
	case TerrainBandMarsh:
		return terrainBandProperties{terrain: TerrainMarsh, base: terrainBaseMarsh, column: 0}
	case TerrainBandMarshPlains:
		return terrainBandProperties{terrain: TerrainPlains, base: terrainBasePlains, column: 1, secondary: true}
	case TerrainBandPlains:
		return terrainBandProperties{terrain: TerrainPlains, base: terrainBasePlains, column: 4}
	case TerrainBandPlainsForest:
		return terrainBandProperties{terrain: TerrainPlains, base: terrainBasePlains, column: 2, secondary: true}
	case TerrainBandForest:
		return terrainBandProperties{terrain: TerrainForest, base: terrainBasePlains, column: 2}
	case TerrainBandForestMountains:
		return terrainBandProperties{terrain: TerrainForest, base: terrainBasePlains, column: 3, secondary: true}
	case TerrainBandMountains:
		return terrainBandProperties{terrain: TerrainMountains, base: terrainBasePlains, column: 3}
	case TerrainBandIce:
		return terrainBandProperties{terrain: TerrainSnow, base: terrainBaseIce, column: -1}
	default:
		return terrainBandProperties{terrain: TerrainWater, base: terrainBaseWater, column: -1}
	}
}

type terrainDecoration struct {
	Variant   int
	Column    int
	Secondary bool
	OffsetX   float64
	OffsetY   float64
}

func terrainDecorationPlan(_ int64, p image.Point, band TerrainBand, tileWidth int) []terrainDecoration {
	props := band.properties()
	if props.column < 0 {
		return nil
	}

	hash := absInt(p.X)*7 + absInt(p.Y)*3
	xOffset := float64(tileWidth / ((hash%5)*2 + 4))
	yJitter := float64(((hash % 4) * 2) + 5)
	if band == TerrainBandSand {
		xOffset = 0
		yJitter = 0
	}
	return []terrainDecoration{
		{
			Variant:   hash % 11,
			Column:    props.column,
			Secondary: props.secondary,
			OffsetX:   xOffset,
			OffsetY:   -40 - yJitter,
		},
		{
			Variant:   (p.X + hash) % 11,
			Column:    props.column,
			Secondary: props.secondary,
			OffsetX:   -xOffset,
			OffsetY:   -40 + yJitter,
		},
	}
}

func generateTerrainBands(w, h int, seed int64) [][]TerrainBand {
	const maxAttempts = 32
	var best [][]TerrainBand
	bestLand := -1
	minimumLand := w * h * 35 / 100

	for attempt := range maxAttempts {
		bands := generateTerrainBandAttempt(w, h, seed+int64(attempt)*7919)
		fillEnclosedWater(bands)
		retainMainLandmass(bands)
		land := countLandBands(bands)
		if land > bestLand {
			best = bands
			bestLand = land
		}
		if land >= minimumLand {
			return bands
		}
	}
	return best
}

func generateTerrainBandAttempt(w, h int, seed int64) [][]TerrainBand {
	rng := rand.New(rand.NewSource(seed))
	var lattice [19][19]int
	for y := range lattice {
		for x := range lattice[y] {
			lattice[y][x] = rng.Intn(16)
		}
	}
	for i := range 18 {
		lattice[i][18] = lattice[i][0]
		lattice[18][i] = lattice[0][i]
	}
	lattice[18][18] = lattice[0][0]

	bands := make([][]TerrainBand, h)
	for y := range h {
		bands[y] = make([]TerrainBand, w)
		for x := range w {
			ox := x * 64 / max(1, w)
			oy := y * 64 / max(1, h)
			if isOutsideOriginalMapShape(ox, oy) || x < 2 || y < 2 || x >= w-2 || y >= h-2 {
				bands[y][x] = TerrainBandWater
				continue
			}
			bands[y][x] = quantizeTerrain(originalTerrainValue(lattice, ox, oy))
		}
	}
	return bands
}

func isOutsideOriginalMapShape(x, y int) bool {
	projectedX := (x+y)*3 - 32
	projectedY := (y-x)*3 + 100
	return projectedX < 4 || projectedY < 4 || projectedX > 315 || projectedY > 195
}

func originalTerrainValue(lattice [19][19]int, x, y int) int {
	distance := absInt(y-32) + absInt(x-32)
	falloff := clampInt((distance*distance)/512+absInt(x-y)/16, 0, 12)
	value := (sampleOriginalNoise(lattice, x<<5, y<<5)*4 +
		sampleOriginalNoise(lattice, x<<8, y<<8)*2 +
		sampleOriginalNoise(lattice, x<<9, y<<9) - falloff*512) * 7
	return clampInt(value/256, 0, 100)
}

func sampleOriginalNoise(lattice [19][19]int, x, y int) int {
	x -= 128
	y -= 128
	xCell, xRemainder := floorDivMod(x, 256)
	yCell, yRemainder := floorDivMod(y, 256)
	xCell = positiveMod(xCell, 16)
	yCell = positiveMod(yCell, 16)
	xFraction := xRemainder / 8
	yFraction := yRemainder / 8

	return (lattice[yCell][xCell]*(32-xFraction)*(32-yFraction) +
		lattice[yCell][xCell+1]*xFraction*(32-yFraction) +
		lattice[yCell+1][xCell]*(32-xFraction)*yFraction +
		lattice[yCell+1][xCell+1]*xFraction*yFraction) / 32
}

func floorDivMod(value, divisor int) (int, int) {
	quotient := value / divisor
	remainder := value % divisor
	if remainder < 0 {
		quotient--
		remainder += divisor
	}
	return quotient, remainder
}

func positiveMod(value, divisor int) int {
	value %= divisor
	if value < 0 {
		value += divisor
	}
	return value
}

func quantizeTerrain(value int) TerrainBand {
	switch {
	case value < 16:
		return TerrainBandWater
	case value < 22:
		return TerrainBandSand
	case value < 24:
		return TerrainBandSandMarsh
	case value < 32:
		return TerrainBandMarsh
	case value < 34:
		return TerrainBandMarshPlains
	case value < 40:
		return TerrainBandPlains
	case value < 43:
		return TerrainBandPlainsForest
	case value < 56:
		return TerrainBandForest
	case value < 60:
		return TerrainBandForestMountains
	default:
		return TerrainBandMountains
	}
}

func fillEnclosedWater(bands [][]TerrainBand) {
	if len(bands) < 3 || len(bands[0]) < 3 {
		return
	}
	copyBands := cloneTerrainBands(bands)
	for y := 1; y < len(bands)-1; y++ {
		for x := 1; x < len(bands[y])-1; x++ {
			if copyBands[y][x] != TerrainBandWater {
				continue
			}
			if copyBands[y-1][x-1] != TerrainBandWater && copyBands[y-1][x+1] != TerrainBandWater &&
				copyBands[y+1][x-1] != TerrainBandWater && copyBands[y+1][x+1] != TerrainBandWater {
				bands[y][x] = TerrainBandPlains
			}
		}
	}
}

func retainMainLandmass(bands [][]TerrainBand) {
	start, ok := landBandNearestCenter(bands)
	if !ok {
		return
	}
	visited := map[image.Point]bool{start: true}
	queue := []image.Point{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, direction := range Directions[current.Y%2] {
			next := current.Add(direction)
			if next.Y < 0 || next.Y >= len(bands) || next.X < 0 || next.X >= len(bands[next.Y]) || visited[next] {
				continue
			}
			if bands[next.Y][next.X] == TerrainBandWater {
				continue
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}
	for y := range bands {
		for x := range bands[y] {
			if bands[y][x] != TerrainBandWater && !visited[image.Point{X: x, Y: y}] {
				bands[y][x] = TerrainBandWater
			}
		}
	}
}

func landBandNearestCenter(bands [][]TerrainBand) (image.Point, bool) {
	if len(bands) == 0 {
		return image.Point{}, false
	}
	cx, cy := len(bands[0])/2, len(bands)/2
	best := image.Point{}
	bestDistance := int(^uint(0) >> 1)
	found := false
	for y := range bands {
		for x, band := range bands[y] {
			if band == TerrainBandWater {
				continue
			}
			distance := absInt(x-cx) + absInt(y-cy)
			if distance < bestDistance {
				best = image.Point{X: x, Y: y}
				bestDistance = distance
				found = true
			}
		}
	}
	return best, found
}

func countLandBands(bands [][]TerrainBand) int {
	count := 0
	for y := range bands {
		for _, band := range bands[y] {
			if band != TerrainBandWater {
				count++
			}
		}
	}
	return count
}

func firstLandBand(bands [][]TerrainBand) (image.Point, bool) {
	for y := range bands {
		for x, band := range bands[y] {
			if band != TerrainBandWater {
				return image.Point{X: x, Y: y}, true
			}
		}
	}
	return image.Point{}, false
}

func reachableLandBands(bands [][]TerrainBand, start image.Point) int {
	visited := map[image.Point]bool{start: true}
	queue := []image.Point{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, direction := range Directions[current.Y%2] {
			next := current.Add(direction)
			if next.Y < 0 || next.Y >= len(bands) || next.X < 0 || next.X >= len(bands[next.Y]) || visited[next] {
				continue
			}
			if bands[next.Y][next.X] == TerrainBandWater {
				continue
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}
	return len(visited)
}

func cloneTerrainBands(source [][]TerrainBand) [][]TerrainBand {
	clone := make([][]TerrainBand, len(source))
	for y := range source {
		clone[y] = append([]TerrainBand(nil), source[y]...)
	}
	return clone
}

func clampInt(value, low, high int) int {
	return min(max(value, low), high)
}
