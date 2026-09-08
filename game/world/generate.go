package world

// generates the overworld map and terrain

import (
	"container/heap"
	"fmt"
	"image"
	"math/rand"

	"github.com/benprew/s30/game/domain"
	"github.com/hajimehoshi/ebiten/v2"
)

// Terrain type constants
const (
	TerrainUndefined = iota
	TerrainWater
	TerrainSand
	TerrainMarsh
	TerrainPlains
	TerrainForest
	TerrainMountains
	TerrainSnow
)

// different directions based on which y row you're on, because it's a zigzag pattern
var Directions = [2][8]image.Point{
	{{0, 2}, {0, -2}, {1, 0}, {-1, 0}, {0, -1}, {-1, -1}, {0, 1}, {-1, 1}},
	{{0, 2}, {0, -2}, {1, 0}, {-1, 0}, {1, -1}, {0, -1}, {1, 1}, {0, 1}},
}
var DirNames = []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}

// mapTerrainTypes assigns terrain based on noise values and returns potential city locations.
func (l *Level) mapTerrainTypes(bands [][]TerrainBand) []image.Point {
	l.Tiles = make([][]*Tile, l.H)
	validCityLocations := []image.Point{}

	for y := range l.H {
		l.Tiles[y] = make([]*Tile, l.W)
		for x := range l.W {
			band := bands[y][x]
			t := &Tile{TerrainType: band.properties().terrain, TerrainBand: band}
			l.Tiles[y][x] = t

			if t.TerrainType != TerrainWater {
				validCityLocations = append(validCityLocations, image.Point{x, y})
			}
		}
	}
	return validCityLocations // Return potential locations
}

// placeCities places cities and returns their locations.
func (l *Level) placeCities(validLocations []image.Point, citySprites [][]*ebiten.Image, numCities, minDistance int, rng *rand.Rand) {
	if len(validLocations) == 0 || numCities <= 0 {
		fmt.Println("Warning: No valid locations provided or numCities <= 0.")
	}

	placedCities := []image.Point{}
	castleLocs := l.castleTileLocations()
	dungeonLocs := l.dungeonTileLocations()

	// Shuffle valid locations for random placement
	rng.Shuffle(len(validLocations), func(i, j int) {
		validLocations[i], validLocations[j] = validLocations[j], validLocations[i]
	})

	for _, loc := range validLocations {
		if len(placedCities) >= numCities {
			break // Reached the desired number of cities
		}

		tile := l.Tile(loc)
		if tile == nil || tile.IsCastle || tile.IsDungeon {
			continue
		}

		// Check distance from already placed cities
		isValidPlacement := true
		for _, city := range placedCities {
			dist := absInt(loc.X-city.X) + absInt(loc.Y-city.Y)
			if dist <= minDistance {
				isValidPlacement = false
				break
			}
		}

		if isValidPlacement && !farFrom(loc, castleLocs, minDistance) {
			isValidPlacement = false
		}
		if isValidPlacement && !farFrom(loc, dungeonLocs, max(2, minDistance/2)) {
			isValidPlacement = false
		}

		if !isValidPlacement {
			continue
		}

		if tile != nil {
			tile.suppressTerrainDecorations()
			cityIdx := deterministicIndex(l.GenerationSeed, loc.X, loc.Y, 12)
			cityX := cityIdx % 6
			cityY := 0
			if cityIdx > 5 {
				cityY = 2
			}

			tier := domain.TierCapital

			if citySprites != nil {
				tile.AddCitySprite(citySprites[cityY][cityX])
				tile.AddCitySprite(citySprites[cityY+1][cityX])
			}
			amuletColor := assignAmuletColor(len(placedCities))

			city := &domain.City{
				Tier:            tier,
				Name:            genCityNameWithRNG(rng),
				X:               loc.X,
				Y:               loc.Y,
				BackgroundImage: cityBgImage(int(tier)),
				AmuletColor:     amuletColor,
				CardsForSale:    domain.MkCards(),
			}

			tile.City = city
			placedCities = append(placedCities, loc)
		}
	}

	if len(placedCities) < numCities {
		fmt.Printf("Warning: Could only place %d out of %d requested cities with min distance %d.\n", len(placedCities), numCities, minDistance)
	}
	// Randomly assign world magics to cities
	if len(placedCities) > 0 {
		shuffledCities := make([]image.Point, len(placedCities))
		copy(shuffledCities, placedCities)
		rng.Shuffle(len(shuffledCities), func(i, j int) {
			shuffledCities[i], shuffledCities[j] = shuffledCities[j], shuffledCities[i]
		})

		availableWorldMagics := make([]*domain.WorldMagic, len(domain.AllWorldMagics))
		copy(availableWorldMagics, domain.AllWorldMagics)
		rng.Shuffle(len(availableWorldMagics), func(i, j int) {
			availableWorldMagics[i], availableWorldMagics[j] = availableWorldMagics[j], availableWorldMagics[i]
		})

		// Assign each world magic to a random city
		numToAssign := min(len(availableWorldMagics), len(shuffledCities))
		for i := range numToAssign {
			tile := l.Tile(shuffledCities[i])
			if tile != nil && tile.IsCity() {
				tile.City.AssignedWorldMagic = availableWorldMagics[i]
			}
		}
	}
}

type roadSearchNode struct {
	position image.Point
	cost     int
	order    int
}

type roadSearchQueue []roadSearchNode

func (q roadSearchQueue) Len() int { return len(q) }
func (q roadSearchQueue) Less(i, j int) bool {
	if q[i].cost == q[j].cost {
		return q[i].order < q[j].order
	}
	return q[i].cost < q[j].cost
}
func (q roadSearchQueue) Swap(i, j int)   { q[i], q[j] = q[j], q[i] }
func (q *roadSearchQueue) Push(value any) { *q = append(*q, value.(roadSearchNode)) }
func (q *roadSearchQueue) Pop() any {
	old := *q
	last := old[len(old)-1]
	*q = old[:len(old)-1]
	return last
}

// connectCityBFS finds the lowest-cost route from a city to the road network.
// The old name remains because saved-map rebuilds and tests use it.
func (l *Level) connectCityBFS(start image.Point) []image.Point {
	queue := &roadSearchQueue{{position: start}}
	heap.Init(queue)
	parents := map[image.Point]image.Point{start: start}
	costs := map[image.Point]int{start: 0}
	order := 0

	for queue.Len() > 0 {
		node := heap.Pop(queue).(roadSearchNode)
		current := node.position
		if node.cost != costs[current] {
			continue
		}
		tile := l.Tile(current)
		if current != start && (tile.IsCity() || tile.IsRoad()) {
			path := []image.Point{}
			temp := current
			for temp != start {
				path = append(path, temp)
				parent, ok := parents[temp]
				if !ok || parent == temp {
					return nil
				}
				temp = parent
			}
			path = append(path, start)
			return path
		}

		dirs := Directions[current.Y%2]
		for _, n := range dirs {
			neighborPos := image.Point{X: current.X + n.X, Y: current.Y + n.Y}
			if neighborPos.X < 0 || neighborPos.X >= l.W || neighborPos.Y < 0 || neighborPos.Y >= l.H {
				continue
			}
			neighbor := l.Tile(neighborPos)
			stepCost, passable := roadTravelCost(neighbor)
			if !passable {
				continue
			}
			newCost := node.cost + stepCost
			oldCost, seen := costs[neighborPos]
			if seen && newCost >= oldCost {
				continue
			}
			costs[neighborPos] = newCost
			parents[neighborPos] = current
			order++
			heap.Push(queue, roadSearchNode{position: neighborPos, cost: newCost, order: order})
		}
	}

	panic(fmt.Sprintf("Warning: BFS from %v found no target road or city.\n", start))
}

func roadTravelCost(tile *Tile) (int, bool) {
	if tile == nil || tile.TerrainType == TerrainWater {
		return 0, false
	}
	if tile.IsRoad() {
		return 1, true
	}
	switch tile.TerrainType {
	case TerrainPlains:
		return 10, true
	case TerrainSand, TerrainMarsh:
		return 14, true
	case TerrainForest:
		return 18, true
	case TerrainMountains, TerrainSnow:
		return 40, true
	default:
		return 20, true
	}
}

// getDirection determines the compass direction from one tile to an adjacent tile.
// because we render tiles in a zigzag pattern, it's not a simple X/Y grid
func getDirection(from, to image.Point) string {
	dx := to.X - from.X
	dy := to.Y - from.Y

	dir := image.Point{dx, dy}

	dirs := Directions[from.Y%2]
	for i, d := range dirs {
		if d == dir {
			return DirNames[i]
		}
	}
	panic(fmt.Sprintf("unknown dir: %+v, from: %v, to: %v", dir, from, to))
}

// getRoadSprite finds the road sprite corresponding to a specific exit direction.
func (l *Level) getRoadSprite(direction string) *ebiten.Image {
	for r, row := range l.roadSpriteInfo {
		for c, dir := range row {
			if dir == direction {
				// Ensure the sprite exists at this index
				if r < len(l.roadSprites) && c < len(l.roadSprites[r]) && l.roadSprites[r][c] != nil {
					return l.roadSprites[r][c]
				}
				panic(fmt.Sprintf("Warning: Road sprite for direction %s at [%d][%d] not found or is nil.\n", direction, r, c))
			}
		}
	}
	panic(fmt.Sprintf("Warning: No road sprite definition found for direction %s.\n", direction))
}

// drawRoadAlongPath adds road sprites to tiles along a given path.
func (l *Level) drawRoadAlongPath(path []image.Point) {
	// fmt.Println("path:", path)
	if len(path) < 2 {
		return // Need at least two points for a path segment
	}

	for i, currentPos := range path {
		tile := l.Tile(currentPos)
		if tile == nil {
			fmt.Printf("Warning: Tile not found at %v during road drawing.\n", currentPos)
			continue
		}

		// Determine incoming and outgoing directions relative to the current tile
		var incomingDirFromPrev, outgoingDirToNext string

		if i > 0 { // Has a previous node
			// Direction from previous node *to* current node
			incomingDirFromPrev = getDirection(path[i-1], currentPos)
		}
		if i < len(path)-1 { // Has a next node
			// Direction from current node *to* next node
			outgoingDirToNext = getDirection(currentPos, path[i+1])
		}

		// Add road sprites based on directions
		// We need the sprite representing the segment leaving the *current* tile

		// If there's a previous tile, add the road segment pointing back to it.
		if incomingDirFromPrev != "" {
			// The sprite needed is the one exiting the *current* tile towards the *previous* tile.
			exitDirTowardsPrev := getDirection(currentPos, path[i-1])
			sprite := l.getRoadSprite(exitDirTowardsPrev)
			if sprite != nil {
				tile.AddRoadSprite(sprite)
			}
		}
		// If there's a next tile, add the road segment pointing towards it.
		if outgoingDirToNext != "" {
			// The sprite needed is the one exiting the *current* tile towards the *next* tile.
			sprite := l.getRoadSprite(outgoingDirToNext)
			if sprite != nil {
				tile.AddRoadSprite(sprite)
			}
		}
	}
}

// --- End Road Generation Logic ---

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func assignAmuletColor(cityIndex int) domain.ColorMask {
	amuletColors := domain.GetAllAmuletColors()
	return amuletColors[cityIndex%len(amuletColors)]
}
