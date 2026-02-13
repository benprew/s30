package world

import (
	"fmt"
	"image"
)

// TileType represents the type of terrain for autotiling.
// Tile ordering: lowest value never has a transition overlay.
type TileType int

const (
	TilePlains TileType = iota
	TileForest
	TileWater
)

func (t TileType) String() string {
	switch t {
	case TilePlains:
		return "Plains"
	case TileForest:
		return "Forest"
	case TileWater:
		return "Water"
	default:
		return "Unknown"
	}
}

type dirIdx int

const (
	_ dirIdx = iota // N (unused in autotiling)
	_               // S (unused in autotiling)
	eIdx
	wIdx
	neIdx
	nwIdx
	seIdx
	swIdx
)

// TileMap represents a grid of tiles for autotiling calculations.
type TileMap struct {
	Grid          map[image.Point]TileType
	Width, Height int
}

func NewTileMap(width, height int) *TileMap {
	return &TileMap{
		Grid:   make(map[image.Point]TileType),
		Width:  width,
		Height: height,
	}
}

func (w *TileMap) Get(p image.Point) TileType {
	if t, ok := w.Grid[p]; ok {
		return t
	}
	return TilePlains
}

func (w *TileMap) Has(p image.Point) bool {
	_, ok := w.Grid[p]
	return ok
}

func (w *TileMap) Set(x, y int, t TileType) {
	w.Grid[image.Point{x, y}] = t
}

// Transition describes the edge graphic needed for a tile.
type Transition struct {
	Side       string   // NW, NE, SE, SW
	SourceType TileType // The dominant tile type causing the transition
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
//	NW: N corner -> NE neighbor, W corner -> SW neighbor
//	NE: N corner -> NW neighbor, E corner -> SE neighbor
//	SE: S corner -> SW neighbor, E corner -> NE neighbor
//	SW: S corner -> SE neighbor, W corner -> NW neighbor
func GetTransitions(currentPos image.Point, world *TileMap) []Transition {
	transitions := []Transition{}
	myType := world.Get(currentPos)
	d := Directions[currentPos.Y%2]

	isCornerOpen := func(adjDirIdx, sideDirIdx dirIdx, sourceType TileType) bool {
		adjPos := currentPos.Add(d[adjDirIdx])
		if !world.Has(adjPos) {
			return false
		}
		adjType := world.Get(adjPos)
		adjD := Directions[adjPos.Y%2]
		adjNeighborType := world.Get(adjPos.Add(adjD[sideDirIdx]))
		return adjNeighborType == sourceType && sourceType >= adjType
	}

	checkEdge := func(sideVec image.Point, sideDirIdx dirIdx, sideName string, corner1Adj, corner2Adj dirIdx) {
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

	checkEdge(d[nwIdx], nwIdx, "NW", neIdx, swIdx)
	checkEdge(d[neIdx], neIdx, "NE", nwIdx, seIdx)
	checkEdge(d[seIdx], seIdx, "SE", swIdx, neIdx)
	checkEdge(d[swIdx], swIdx, "SW", seIdx, nwIdx)

	fixVertexCorners := func(cardinalIdx dirIdx, edge1Name, edge2Name string) {
		if len(transitions) == 4 {
			return
		}
		var e1, e2 = -1, -1
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
	fixVertexCorners(wIdx, "NW", "SW")
	fixVertexCorners(eIdx, "NE", "SE")

	return transitions
}
