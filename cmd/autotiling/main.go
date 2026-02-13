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
	return Plains
}

func (w *World) Has(p TilePoint) bool {
	_, ok := w.Grid[p]
	return ok
}

func (w *World) Set(x, y int, t TileType) {
	w.Grid[TilePoint{x, y}] = t
}

// Transition describes the edge graphic needed
type Transition struct {
	Side       string   // NW, NE, SE, SW
	SourceType TileType // The dominant tile type causing the transition (e.g., Water)
	Open1      bool     // First corner of Side (e.g., N for "NW")
	Open2      bool     // Second corner of Side (e.g., W for "NW")
}

func (t Transition) String() string {
	c1 := "Closed"
	if t.Open1 {
		c1 = "Open"
	}
	c2 := "Closed"
	if t.Open2 {
		c2 = "Open"
	}
	return fmt.Sprintf("[%s Edge] Type: %s | %c: %s | %c: %s", t.Side, t.SourceType, t.Side[0], c1, t.Side[1], c2)
}

// GetTransitions calculates all needed overlays for a specific tile.
// Each tile is a diamond with vertices N, E, S, W and edges NW, NE, SE, SW.
// Each vertex is shared by 4 tiles. A corner is "open" if the tile across the
// vertex also has a transition on the same edge direction.
//
// Corner adjacency (the tile across each vertex for a given edge):
//
//	NW: N corner → NE neighbor, W corner → SW neighbor
//	NE: N corner → NW neighbor, E corner → SE neighbor
//	SE: S corner → SW neighbor, E corner → NE neighbor
//	SW: S corner → SE neighbor, W corner → NW neighbor
func GetTransitions(currentPos TilePoint, world *World) []Transition {
	transitions := []Transition{}
	myType := world.Get(currentPos)
	d := Directions[currentPos.Y%2]

	isCornerOpen := func(adjDirIdx, sideDirIdx DirIdx, sourceType TileType) bool {
		adjPos := currentPos.Add(d[adjDirIdx])
		if !world.Has(adjPos) {
			return false
		}
		adjType := world.Get(adjPos)
		adjD := Directions[adjPos.Y%2]
		adjNeighborType := world.Get(adjPos.Add(adjD[sideDirIdx]))
		return adjNeighborType == sourceType && sourceType >= adjType
	}

	checkEdge := func(sideVec TilePoint, sideDirIdx DirIdx, sideName string, corner1Adj, corner2Adj DirIdx) {
		neighborPos := currentPos.Add(sideVec)
		neighborType := world.Get(neighborPos)

		if neighborType > myType {
			transitions = append(transitions, Transition{
				Side:       sideName,
				SourceType: neighborType,
				Open1:      isCornerOpen(corner1Adj, sideDirIdx, neighborType),
				Open2:      isCornerOpen(corner2Adj, sideDirIdx, neighborType),
			})
		}
	}

	checkEdge(d[NWIdx], NWIdx, "NW", NEIdx, SWIdx)
	checkEdge(d[NEIdx], NEIdx, "NE", NWIdx, SEIdx)
	checkEdge(d[SEIdx], SEIdx, "SE", SWIdx, NEIdx)
	checkEdge(d[SWIdx], SWIdx, "SW", SEIdx, NWIdx)

	fixVertexCorners := func(cardinalIdx DirIdx, edge1Name, edge2Name string) {
		if len(transitions) == 4 {
			return
		}
		var e1, e2 int = -1, -1
		for i, t := range transitions {
			if t.Side == edge1Name {
				e1 = i
			}
			if t.Side == edge2Name {
				e2 = i
			}
		}
		if e1 < 0 || e2 < 0 || transitions[e1].SourceType != transitions[e2].SourceType {
			return
		}
		cardinalType := world.Get(currentPos.Add(d[cardinalIdx]))
		if cardinalType != transitions[e1].SourceType {
			return
		}
		transitions[e1].Open2 = true
		transitions[e2].Open1 = true
		transitions[e2].Open2 = false
	}
	fixVertexCorners(WIdx, "NW", "SW")
	fixVertexCorners(EIdx, "NE", "SE")

	return transitions
}

func tr(side string, source TileType, open1, open2 bool) Transition {
	return Transition{Side: side, SourceType: source, Open1: open1, Open2: open2}
}

func main() {
	fmt.Println("--- Autotiling POC ---")

	fmt.Println("\nScenario 1: Continuous Coast")
	buildWorld(
		[]string{
			"WW",
			"WP",
			"WP",
		},
		map[TilePoint][]Transition{
			{1, 1}: {tr("NW", Water, false, true)},
			{1, 2}: {tr("NW", Water, true, false)},
		},
	)

	fmt.Println("\nScenario 2: Water block")
	buildWorld(
		[]string{
			"WPPP",
			"PPPP",
			"PWWW",
			"PPPP",
		},
		map[TilePoint][]Transition{
			{0, 1}: {tr("NW", Water, false, false), tr("SE", Water, false, false)},
			{1, 1}: {tr("SE", Water, false, false), tr("SW", Water, false, false)},
			{2, 1}: {tr("SE", Water, false, false), tr("SW", Water, false, false)},
			{3, 1}: {tr("SW", Water, false, false)},
			{0, 3}: {tr("NE", Water, false, false)},
			{1, 3}: {tr("NW", Water, false, false), tr("NE", Water, false, false)},
			{2, 3}: {tr("NW", Water, false, false), tr("NE", Water, false, false)},
			{3, 3}: {tr("NW", Water, false, false)},
		},
	)

	fmt.Println("\nScenario 3: Wrapped Coast")
	buildWorld(
		[]string{
			"WW",
			"WP",
			"WW",
		},
		map[TilePoint][]Transition{
			{1, 1}: {tr("NW", Water, false, true), tr("SW", Water, true, false)},
		},
	)

	fmt.Println("\nScenario 4: island")
	buildWorld(
		[]string{
			"WW",
			"WW",
			"WPW",
			"WW",
			"WW",
		},
		map[TilePoint][]Transition{
			{1, 2}: {tr("NW", Water, true, true), tr("NE", Water, true, true), tr("SE", Water, true, true), tr("SW", Water, true, true)},
		},
	)
}

func buildWorld(layout []string, expected map[TilePoint][]Transition) {
	w := NewWorld(len(layout[0]), len(layout))
	for y := range layout {
		for x, tile := range layout[y] {
			switch tile {
			case 'W':
				w.Set(x, y, Water)
			case 'P':
				w.Set(x, y, Plains)
			case 'F':
				w.Set(x, y, Forest)
			default:
				panic("unknown tile type")
			}
		}
	}

	failed := false
	for y := range layout {
		for x := range layout[y] {
			p := TilePoint{x, y}
			got := GetTransitions(p, w)
			want := expected[p]
			if len(got) != len(want) {
				fmt.Printf("FAIL %v: expected %d transitions, got %d\n", p, len(want), len(got))
				for _, t := range got {
					fmt.Printf("  got: %s\n", t)
				}
				failed = true
				continue
			}
			for i := range got {
				if got[i] != want[i] {
					fmt.Printf("FAIL %v[%d]: expected %s, got %s\n", p, i, want[i], got[i])
					failed = true
				}
			}
		}
	}
	if failed {
		fmt.Println("FAIL")
	} else {
		fmt.Println("PASS")
	}
}
