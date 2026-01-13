package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
)

type TileGrid struct {
	width  int
	height int
	tiles  [][]int
}

type CoastlineMap struct {
	sprites map[string]*SpriteData
}

type SpriteData struct {
	full    []string
	connect map[string][]string
}

const (
	Plains = iota
	Water
	Sand
	Forest
	Marsh
	Ice
)

func ebitenToImage(eimg *ebiten.Image) image.Image {
	return eimg.SubImage(eimg.Bounds())
}

type Game struct {
	grid       *TileGrid
	cstMap     *CoastlineMap
	cstline1   [][]*ebiten.Image
	cstline2   [][]*ebiten.Image
	landtile   *world.SpriteSheet
	outputImgs []image.Image // Store multiple outputs
	rendered   bool
	err        error
}

func (g *Game) Update() error {
	if g.rendered {
		return ebiten.Termination
	}

	// Generate layout with seed 100 for debugging
	seeds := []int64{100}
	for _, seed := range seeds {
		fmt.Printf("Generating layout with seed %d...\n", seed)
		g.grid.createRandomPattern(seed)
		g.grid.printGrid()
		img := g.grid.renderWithTransitions(g.landtile, g.cstline1, g.cstline2, g.cstMap)
		g.outputImgs = append(g.outputImgs, img)
	}

	g.rendered = true
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Off-screen rendering
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func main() {
	fmt.Println("Tile Transition Demo (Random Layouts)")

	// Larger grid for random layouts
	grid := &TileGrid{
		width:  5,
		height: 5,
	}

	game := &Game{
		grid: grid,
	}

	var err error
	game.cstMap, err = parseCoastlineMap()
	if err != nil {
		fmt.Printf("Error parsing coastline map: %v\n", err)
		return
	}

	fmt.Printf("Loaded coastline map with %d sprites\n", len(game.cstMap.sprites))

	// Load Assets
	game.cstline1, err = imageutil.LoadSpriteSheet(4, 28, assets.Cstline_png)
	if err != nil {
		fmt.Printf("Error loading Cstline1: %v\n", err)
		return
	}

	game.cstline2, err = imageutil.LoadSpriteSheet(4, 14, assets.Cstline2_png)
	if err != nil {
		fmt.Printf("Error loading Cstline2: %v\n", err)
		return
	}

	game.landtile, err = world.LoadWorldTileSheet(206, 102)
	if err != nil {
		fmt.Printf("Error loading land tiles: %v\n", err)
		return
	}

	// Run Game
	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		fmt.Printf("Ebiten error: %v\n", err)
		return
	}

	if game.err != nil {
		fmt.Printf("Rendering error: %v\n", game.err)
		return
	}

	// Save Outputs
	for i, img := range game.outputImgs {
		filename := fmt.Sprintf("layout_%d.png", i)
		f, err := os.Create(filename)
		if err != nil {
			fmt.Printf("Error creating %s: %v\n", filename, err)
			continue
		}

		if err := png.Encode(f, img); err != nil {
			fmt.Printf("Error encoding %s: %v\n", filename, err)
			f.Close()
			continue
		}
		f.Close()
		fmt.Printf("Saved %s\n", filename)
	}
}

func (g *TileGrid) createRandomPattern(seed int64) {
	r := rand.New(rand.NewSource(seed))
	g.tiles = make([][]int, g.height)
	for y := 0; y < g.height; y++ {
		g.tiles[y] = make([]int, g.width)
		for x := 0; x < g.width; x++ {
			g.tiles[y][x] = Plains
		}
	}

	// Create random clusters of water
	numWater := (g.width * g.height) / 3 // Approx 33% water
	for i := 0; i < numWater; i++ {
		rx := r.Intn(g.width)
		ry := r.Intn(g.height)
		g.tiles[ry][rx] = Water
	}
}

func (g *TileGrid) printGrid() {
	terrainNames := []string{"P", "W", "S", "F", "M"}
	fmt.Println("\nTerrain Grid:")
	for y := 0; y < g.height; y++ {
		if y%2 == 1 {
			fmt.Print(" ")
		}
		for x := 0; x < g.width; x++ {
			fmt.Printf("%s ", terrainNames[g.tiles[y][x]])
		}
		fmt.Println()
	}
}

func parseCoastlineMap() (*CoastlineMap, error) {
	content, err := os.ReadFile("assets/art/sprites/world/land/Cstline_map.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read Cstline_map.json: %w", err)
	}

	type jsonSpriteData struct {
		Full    []string
		Connect map[string][]string
	}

	rawMap := make(map[string]jsonSpriteData)
	if err := json.Unmarshal(content, &rawMap); err != nil {
		return nil, err
	}

	cstMap := &CoastlineMap{
		sprites: make(map[string]*SpriteData),
	}

	for key, data := range rawMap {
		cstMap.sprites[key] = &SpriteData{
			full:    data.Full,
			connect: data.Connect,
		}
	}

	return cstMap, nil
}

func (g *TileGrid) renderWithTransitions(landtile *world.SpriteSheet, cstline1, cstline2 [][]*ebiten.Image, cstMap *CoastlineMap) image.Image {
	tileWidth := 206
	tileHeight := 102

	imgWidth := g.width * tileWidth
	imgHeight := (g.height * tileHeight / 2) + tileHeight + 50

	outputImg := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Pass 1: Draw Base Tiles
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			pixelX := x * tileWidth
			pixelY := y * tileHeight / 2

			if y%2 != 0 {
				pixelX += tileWidth / 2
			}

			var baseSprite *ebiten.Image
			switch g.tiles[y][x] {
			case Water:
				baseSprite = landtile.Water
			case Plains:
				baseSprite = landtile.Plains
			case Forest:
				baseSprite = landtile.Forest
			case Marsh:
				baseSprite = landtile.Marsh
			case Sand:
				baseSprite = landtile.Sand
			}

			if baseSprite != nil {
				bounds := baseSprite.Bounds()
				baseSpriteImg := ebitenToImage(baseSprite)
				draw.Draw(outputImg, image.Rect(pixelX, pixelY, pixelX+bounds.Dx(), pixelY+bounds.Dy()),
					baseSpriteImg, bounds.Min, draw.Over)
			}
		}
	}

	// Pass 2: Draw Transitions
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			if g.tiles[y][x] == Plains {
				pixelX := x * tileWidth
				pixelY := y * tileHeight / 2

				if y%2 != 0 {
					pixelX += tileWidth / 2
				}

				transitions := g.getEdgeTransitions(x, y, cstMap, cstline1)
				for _, pt := range transitions {
					if pt.sprite != nil {
						if x == 0 && y == 1 {
							fmt.Printf("DEBUG (0,1) Drawing Transition Sprite at Row %d Col %d Offset (%d,%d)\n", pt.row, pt.col, pt.offsetX, pt.offsetY)
						}
						bounds := pt.sprite.Bounds()
						spriteImg := ebitenToImage(pt.sprite)
						drawX := pixelX + pt.offsetX
						drawY := pixelY + pt.offsetY
						draw.Draw(outputImg, image.Rect(drawX, drawY, drawX+bounds.Dx(), drawY+bounds.Dy()),
							spriteImg, bounds.Min, draw.Over)
					}
				}
			}
		}
	}

	return outputImg
}

func (g *TileGrid) getNeighbors(x, y int) [8]int {
	neighbors := [8]int{-1, -1, -1, -1, -1, -1, -1, -1}
	dirs := world.Directions[y%2]
	for i := 0; i < 8; i++ {
		nx := x + dirs[i].X
		ny := y + dirs[i].Y
		if nx >= 0 && nx < g.width && ny >= 0 && ny < g.height {
			neighbors[i] = g.tiles[ny][nx]
		}
	}
	return neighbors
}

type TransitionSprite struct {
	row int
	col int
}

type PositionedTransition struct {
	sprite  *ebiten.Image
	offsetX int
	offsetY int
	row     int
	col     int
}

type EdgeDef struct {
	Name        string
	NeighborIdx int
	StartIdx    int
	EndIdx      int
	PrevEdge    string
	NextEdge    string
}

// getEdgeTransitions implements the Autotiling Algorithm using "Open/Closed" logic.
// Logic adapted from cmd/autotiling/main.go.
func (g *TileGrid) getEdgeTransitions(x, y int, cstMap *CoastlineMap, cstline1 [][]*ebiten.Image) []PositionedTransition {
	neighbors := g.getNeighbors(x, y)
	transitions := []PositionedTransition{}

	// Indices based on world.Directions: N=0, S=1, E=2, W=3, NE=4, NW=5, SE=6, SW=7
	edges := []EdgeDef{
		{"3->0", 5, 3, 0, "2->3", "0->1"}, // NW Edge: Neighbor=NW(5), Start=W(3), End=N(0)
		{"0->1", 4, 0, 2, "3->0", "1->2"}, // NE Edge: Neighbor=NE(4), Start=N(0), End=E(2)
		{"1->2", 6, 2, 1, "0->1", "2->3"}, // SE Edge: Neighbor=SE(6), Start=E(2), End=S(1)
		{"2->3", 7, 1, 3, "1->2", "3->0"}, // SW Edge: Neighbor=SW(7), Start=S(1), End=W(3)
	}

	tileCornerPositions := map[int][2]int{
		0: {103, 0},   // Top
		1: {205, 50},  // Right
		2: {103, 100}, // Bottom
		3: {0, 50},    // Left
	}
	transitionCorner0 := [2]int{51, 0}

	for _, e := range edges {
		// 1. Activation Check (Diagonal Neighbor)
		if neighbors[e.NeighborIdx] != Water {
			continue
		}

		// 2. Open/Closed Check (Orthogonal Neighbors)
		openStart := neighbors[e.StartIdx] == Water
		openEnd := neighbors[e.EndIdx] == Water

		if x == 0 && y == 1 {
			fmt.Printf("DEBUG (0,1) Edge %s: Active (Neighbor %d is Water). Start(Idx %d)=%v, End(Idx %d)=%v\n",
				e.Name, e.NeighborIdx, e.StartIdx, openStart, e.EndIdx, openEnd)
		}

		// 3. Find Pair
		transitionPair := g.findSmartTransitionPair(e.Name, openStart, openEnd, e.PrevEdge, e.NextEdge, cstMap)

		if x == 0 && y == 1 && len(transitionPair) > 0 {
			fmt.Printf("DEBUG (0,1) Edge %s: Selected Pair %v\n", e.Name, transitionPair)
		}

		if len(transitionPair) > 0 {
			parts := strings.Split(e.Name, "->")
			if len(parts) == 2 {
				c1, _ := strconv.Atoi(parts[0])
				c2, _ := strconv.Atoi(parts[1])

				tileCorner1Pos := tileCornerPositions[c1]
				tileCorner2Pos := tileCornerPositions[c2]

				midpointX := (tileCorner1Pos[0] + tileCorner2Pos[0]) / 2
				midpointY := (tileCorner1Pos[1] + tileCorner2Pos[1]) / 2

				t1OffsetX := tileCorner2Pos[0] - transitionCorner0[0]
				t1OffsetY := tileCorner2Pos[1] - transitionCorner0[1]

				t2OffsetX := midpointX - transitionCorner0[0]
				t2OffsetY := midpointY - transitionCorner0[1]

				offsets := [2][2]int{
					{t1OffsetX, t1OffsetY},
					{t2OffsetX, t2OffsetY},
				}

				for i, ts := range transitionPair {
					sprite := g.getSpriteFromRowCol(ts.row, ts.col, cstline1, Water)
					if sprite != nil {
						transitions = append(transitions, PositionedTransition{
							sprite:  sprite,
							offsetX: offsets[i][0],
							offsetY: offsets[i][1],
							row:     ts.row,
							col:     ts.col,
						})
					}
				}
			}
		}
	}

	return transitions
}

type CandidatePair struct {
	pair  []TransitionSprite
	score int
}

func (g *TileGrid) findSmartTransitionPair(edge string, openStart, openEnd bool, prevEdge, nextEdge string, cstMap *CoastlineMap) []TransitionSprite {
	candidates := []TransitionSprite{}
	reverseEdge := getReverseEdge(edge)

	for key, spriteData := range cstMap.sprites {
		hasEdge := false
		hasOtherEdges := false

		for _, fullEdge := range spriteData.full {
			if fullEdge == edge || fullEdge == reverseEdge {
				hasEdge = true
			} else {
				hasOtherEdges = true
			}
		}

		if hasEdge && !hasOtherEdges {
			parts := strings.Split(key, ",")
			if len(parts) == 2 {
				row, _ := strconv.Atoi(parts[0])
				col, _ := strconv.Atoi(parts[1])
				candidates = append(candidates, TransitionSprite{row: row, col: col})
			}
		}
	}

	if len(candidates) < 2 {
		return nil
	}

	// Sort candidates for determinism
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].row != candidates[j].row {
			return candidates[i].row < candidates[j].row
		}
		return candidates[i].col < candidates[j].col
	})

	possiblePairs := []CandidatePair{}

	for _, c1 := range candidates {
		key1 := fmt.Sprintf("%d,%d", c1.row, c1.col)
		spriteData1 := cstMap.sprites[key1]

		for _, connectList := range spriteData1.connect {
			for _, c2 := range candidates {
				if c1.row == c2.row && c1.col == c2.col {
					continue
				}
				spriteKey2 := fmt.Sprintf("%d,%d", c2.row, c2.col)

				isConnected := false
				for _, connectedKey := range connectList {
					if connectedKey == spriteKey2 {
						isConnected = true
						break
					}
				}

				if isConnected {
					score := 0
					spriteData2 := cstMap.sprites[spriteKey2]

					// Check Open End
					c1ConnectsNext := hasConnection(spriteData1.connect, nextEdge)
					if openEnd {
						if c1ConnectsNext {
							score += 2
						} else {
							score -= 2
						}
					} else {
						if !c1ConnectsNext {
							score += 2
						} else {
							score -= 2
						}
					}

					// Check Open Start
					c2ConnectsPrev := hasConnection(spriteData2.connect, prevEdge)
					if openStart {
						if c2ConnectsPrev {
							score += 2
						} else {
							score -= 2
						}
					} else {
						if !c2ConnectsPrev {
							score += 2
						} else {
							score -= 2
						}
					}

					possiblePairs = append(possiblePairs, CandidatePair{
						pair:  []TransitionSprite{c1, c2},
						score: score,
					})
				}
			}
		}
	}

	if len(possiblePairs) > 0 {
		// Use SliceStable for deterministic output when scores are tied
		sort.SliceStable(possiblePairs, func(i, j int) bool {
			return possiblePairs[i].score > possiblePairs[j].score
		})
		return possiblePairs[0].pair
	}

	return nil
}

func hasConnection(connectMap map[string][]string, targetEdge string) bool {
	reverseTarget := getReverseEdge(targetEdge)
	if _, ok := connectMap[targetEdge]; ok {
		return true
	}
	if _, ok := connectMap[reverseTarget]; ok {
		return true
	}
	return false
}

func getReverseEdge(edge string) string {
	parts := strings.Split(edge, "->")
	if len(parts) == 2 {
		return parts[1] + "->" + parts[0]
	}
	return edge
}

func (g *TileGrid) getSpriteFromRowCol(row, col int, cstline1 [][]*ebiten.Image, terrain int) *ebiten.Image {
	setIdx := 0
	switch terrain {
	case Water:
		setIdx = 0
	case Marsh:
		setIdx = 1
	case Forest:
		setIdx = 2
	default:
		return nil
	}

	spriteRow := row + (setIdx * 7)
	if spriteRow >= len(cstline1) {
		return nil
	}

	if col >= len(cstline1[spriteRow]) {
		return nil
	}

	return cstline1[spriteRow][col]
}
