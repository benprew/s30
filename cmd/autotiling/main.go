package main

import (
	"fmt"
	"image"
)

// TileType represents the type of terrain
type TileType int

// Tile ordering, lowest value never has transition
const (
	Plains TileType = iota
	Forest
	Water
)

func (t TileType) String() string {
	switch t {
	case Plains:
		return "Plains"
	case Forest:
		return "Forest"
	case Water:
		return "Water"
	default:
		return "Unknown"
	}
}

// TilePoint represents a coordinate in the grid
type TilePoint image.Point

func (p TilePoint) Add(other TilePoint) TilePoint {
	return TilePoint{p.X + other.X, p.Y + other.Y}
}

var Directions = [2][8]TilePoint{
	{{0, -2}, {0, 2}, {1, 0}, {-1, 0}, {0, -1}, {-1, -1}, {0, 1}, {-1, 1}},
	{{0, -2}, {0, 2}, {1, 0}, {-1, 0}, {1, -1}, {0, -1}, {1, 1}, {0, 1}},
}
var DirNames = []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}

type DirIdx int

const (
	NIdx DirIdx = iota
	SIdx
	EIdx
	WIdx
	NEIdx
	NWIdx
	SEIdx
	SWIdx
)

// World represents the game map
type World struct {
	Grid   map[TilePoint]TileType
	Width  int
	Height int
}

func NewWorld(width, height int) *World {
	return &World{
		Grid:   make(map[TilePoint]TileType),
		Width:  width,
		Height: height,
	}
}

func (w *World) Get(p TilePoint) TileType {
	if t, ok := w.Grid[p]; ok {
		return t
	}
	return Plains // Default to lowest level if out of bounds/empty
}

func (w *World) Set(x, y int, t TileType) {
	w.Grid[TilePoint{x, y}] = t
}

// Transition describes the edge graphic needed
type Transition struct {
	Side       string   // North, South, East, West
	SourceType TileType // The dominant tile type causing the transition (e.g., Water)
	OpenStart  bool     // Is the transition connected at the "start" (Top for W/E, Left for N/S)?
	OpenEnd    bool     // Is the transition connected at the "end" (Bottom for W/E, Right for N/S)?
}

func (t Transition) String() string {
	startStr := "Closed"
	if t.OpenStart {
		startStr = "Open"
	}
	endStr := "Closed"
	if t.OpenEnd {
		endStr = "Open"
	}
	return fmt.Sprintf("[%s Edge] Type: %s | Start: %s | End: %s", t.Side, t.SourceType, startStr, endStr)
}

// GetTransitions calculates all needed overlays for a specific tile
func GetTransitions(currentPos TilePoint, world *World) []Transition {
	transitions := []Transition{}
	myType := world.Get(currentPos)

	// We check for any neighbor types that have a higher precedence than us.
	// For this POC, let's just assume Water > Plains.
	// In a full system, you'd loop from myType+1 to MaxType.

	// Helper to reduce code duplication
	checkEdge := func(sideVec TilePoint, sideName string, startDiag, endDiag TilePoint) {
		neighborPos := currentPos.Add(sideVec)
		neighborType := world.Get(neighborPos)

		// Rule: Only draw transition if neighbor is "above" me in hierarchy
		if neighborType > myType {
			// Check Diagonals to see if the edge connects

			// For West/East edges: Start is Top (Y-1), End is Bottom (Y+1)
			// For North/South edges: Start is Left (X-1), End is Right (X+1)

			// Note: We check the neighbor of the neighbor!
			// Actually, per the "Open/Closed" logic discussed:
			// We check the diagonal neighbor relative to US (the current tile).
			// If that diagonal is ALSO the higher type, then the edge "continues".

			startVal := world.Get(currentPos.Add(startDiag))
			endVal := world.Get(currentPos.Add(endDiag))

			isOpenStart := startVal == neighborType
			isOpenEnd := endVal == neighborType

			fmt.Println("neighborPos", neighborPos, "start:", currentPos.Add(startDiag), "end:", currentPos.Add(endDiag))
			transitions = append(transitions, Transition{
				Side:       sideName,
				SourceType: neighborType,
				OpenStart:  isOpenStart,
				OpenEnd:    isOpenEnd,
			})
		}
	}

	d := Directions[currentPos.Y%2]

	// 1. Check NW (Vertical Edge)
	// Start: NorthWest, End: SouthWest
	fmt.Println("current:", currentPos)
	checkEdge(d[NWIdx], "NW", d[WIdx], d[NIdx])

	// 2. Check NE (Vertical Edge)
	// Start: NorthEast, End: SouthEast
	checkEdge(d[NEIdx], "NE", d[NIdx], d[EIdx])

	// 3. Check SE (Horizontal Edge)
	// Start: SouthEast, End: NorthEast
	checkEdge(d[SEIdx], "SE", d[EIdx], d[SIdx])

	// 4. Check South (Horizontal Edge)
	// Start: SouthWest, End: SouthEast
	checkEdge(d[SWIdx], "SW", d[WIdx], d[SIdx])

	return transitions
}

func main() {
	fmt.Println("--- Autotiling POC ---")

	// Scenario 1: Continuous Coast
	// W (0,0)   W (1,0)
	// W (0,1)   P1 (1,1)
	// W (0,2)   P2 (1,2)
	fmt.Println("\nScenario 1: Continuous Coast")
	world1 := NewWorld(2, 2)
	m := []string{
		"WW",
		"WP",
		"WP",
	}
	buildWorld(world1, m)

	printTileTransitions(world1, TilePoint{1, 1}, "P1 (1,1)")
	printTileTransitions(world1, TilePoint{1, 2}, "P2 (1,2)")

	fmt.Println("\nScenario 2: layout1")
	w2 := NewWorld(4, 4)
	m2 := []string{
		"WPPP",
		"PPPP",
		"PWWW",
		"PPPP",
	}
	buildWorld(w2, m2)
	printTileTransitions(w2, TilePoint{0, 1}, "0,1")
	printTileTransitions(w2, TilePoint{1, 1}, "1,1")
	printTileTransitions(w2, TilePoint{1, 3}, "1,3")
}

func buildWorld(w *World, layout []string) {
	for y := range layout {
		for x, tile := range layout[y] {
			switch tile {
			case 'W':
				w.Set(x, y, Water)
			case 'P':
				w.Set(x, y, Plains)
			default:
				panic("unknown tile type")
			}
		}
	}
}

func printTileTransitions(w *World, p TilePoint, label string) {
	trans := GetTransitions(p, w)
	fmt.Printf("Transitions for %s:\n", label)
	if len(trans) == 0 {
		fmt.Println("  (None)")
	}
	for _, t := range trans {
		fmt.Printf("  -> %s\n", t)
	}
}
