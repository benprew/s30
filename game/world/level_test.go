package world

import (
	"image"
	"testing"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

// Helper function to create a simple level for testing
func createTestLevel(w, h int) *Level {
	l := &Level{
		w:     w,
		h:     h,
		Tiles: make([][]*Tile, h),
		// Initialize other fields if necessary for the function being tested
		// roadSprites, roadSpriteInfo might not be needed for BFS logic itself
	}
	for y := 0; y < h; y++ {
		l.Tiles[y] = make([]*Tile, w)
		for x := 0; x < w; x++ {
			// Default to Plains, can be overridden in tests
			l.Tiles[y][x] = &Tile{TerrainType: TerrainPlains}
		}
	}
	return l
}

func TestConnectCityBFS(t *testing.T) {
	testCases := []struct {
		name           string
		levelSetup     func(*Level) // Function to modify the base level for the test case
		start          image.Point
		expectedPath   []image.Point       // nil if no path expected
		roadSprites    [][]*ebiten.Image // Sprites for roads
		roadSpriteInfo [][]string        // Maps sprite index to direction string (e.g., "N", "NE")
	}{
		{
			name: "Simple direct path to road",
			levelSetup: func(l *Level) {
				l.Tile(image.Point{3, 1}).AddRoadSprite(l.roadSprites[0][1])
			},
			start:        image.Point{X: 1, Y: 1},
			expectedPath: []image.Point{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}},
		},
		{
			name: "Simple direct path to city",
			levelSetup: func(l *Level) {
				l.Tile(image.Point{1, 3}).IsCity = true // Target city
			},
			start:        image.Point{X: 1, Y: 1},
			expectedPath: []image.Point{{X: 1, Y: 1}, {X: 1, Y: 3}}, // go south 1
		},
		{
			name: "Path around water obstacle",
			levelSetup: func(l *Level) {
				l.Tile(image.Point{2, 1}).TerrainType = TerrainWater // Obstacle
				l.Tile(image.Point{4, 1}).AddRoadSprite(l.roadSprites[0][1])
			},
			start: image.Point{X: 1, Y: 1},
			// Expected path might vary slightly based on diagonal preference, this is one possibility
			expectedPath: []image.Point{{X: 4, Y: 1}, {X: 3, Y: 1}, {X: 3, Y: 2}, {X: 2, Y: 2}, {X: 1, Y: 1}},
			// Alternative if diagonals are preferred: {{X: 1, Y: 1}, {X: 2, Y: 2}, {X: 3, Y: 1}} - depends on BFS neighbor order
		},
		// {
		// 	name: "Path to nearest target (road closer than city)",
		// 	levelSetup: func(l *Level) {
		// 		l.Tile(image.Point{3, 1}).AddRoadSprite(l.roadSprites[0][1]) // Closer target (road)
		// 		l.Tile(image.Point{1, 4}).IsCity = true                      // Further target (city)
		// 	},
		// 	start:        image.Point{X: 1, Y: 1},
		// 	expectedPath: []image.Point{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}},
		// },
		// {
		// 	name: "Path to nearest target (city closer than road)",
		// 	levelSetup: func(l *Level) {
		// 		l.Tile(image.Point{4, 4}).AddRoadSprite(l.roadSprites[0][1]) // Further target (road)
		// 		l.Tile(image.Point{1, 3}).IsCity = true                      // Closer target (city)
		// 	},
		// 	start:        image.Point{X: 1, Y: 1},
		// 	expectedPath: []image.Point{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 1, Y: 3}},
		// },
		// {
		// 	name: "Start on a road tile (should find nearest other road/city)",
		// 	levelSetup: func(l *Level) {
		// 		l.Tile(image.Point{1, 1}).AddRoadSprite(l.roadSprites[0][1]) // Start is also a road
		// 		l.Tile(image.Point{3, 1}).AddRoadSprite(l.roadSprites[0][1]) // Target road
		// 		l.Tile(image.Point{1, 3}).IsCity = true                      // Another potential target
		// 	},
		// 	start:        image.Point{X: 1, Y: 1},
		// 	expectedPath: []image.Point{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}}, // Path to the nearest *other* target
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh level for each test case
			level := createTestLevel(5, 5) // Use a 5x5 grid for these tests
			level.Tile(image.Point{1, 1}).IsCity = true

			roads, _ := imageutil.LoadSpriteSheet(6, 2, assets.Roads_png)
			// Store roads and info in the level struct
			level.roadSprites = roads
			// Define the mapping from sprite sheet index to compass direction exit point.
			// Based on Roads.spr.png layout (6 columns, 2 rows)
			level.roadSpriteInfo = [][]string{
				// Row 0: Primarily diagonal and S/SW exits
				{"", "NE", "E", "SE", "S", "SW"}, // Indices 0,0 to 0,5
				// Row 1: Primarily cardinal (except E) and NW exits
				{"W", "NW", "N", "", "", ""}, // Indices 1,0 to 1,5 (Note: some might be empty based on the 6x2 sheet)
			}

			// Apply the specific setup for this test case
			if tc.levelSetup != nil {
				tc.levelSetup(level)
			}

			PrintLevel(level)

			// Override the panic behavior for testing "no path" cases
			defer func() {
				if r := recover(); r != nil {
					// If we expected no path (nil), a panic is acceptable ONLY IF it's the specific "no target" panic
					if tc.expectedPath != nil {
						t.Errorf("connectCityBFS panicked unexpectedly: %v", r)
					} else {
						// Check if the panic message matches the expected one
						expectedPanicMsg := "Warning: BFS from" // Check prefix
						if panicMsg, ok := r.(string); !ok || len(panicMsg) < len(expectedPanicMsg) || panicMsg[:len(expectedPanicMsg)] != expectedPanicMsg {
							t.Errorf("connectCityBFS panicked with unexpected message: %v", r)
						}
						// If expected path is nil and panic occurred as expected, test passes for this aspect.
					}
				}
			}()

			// Run the BFS
			actualPath := level.connectCityBFS(tc.start)

			// Compare the actual path with the expected path
			if len(actualPath) != len(tc.expectedPath) {
				t.Errorf("connectCityBFS(%v) = %v, want %v", tc.start, actualPath, tc.expectedPath)
			}

			// Additional check: If a path was found, ensure the end tile is actually a target
			if len(actualPath) > 0 {
				endPoint := actualPath[0]
				endTile := level.Tile(endPoint)
				if endTile == nil || endPoint == tc.start || (!endTile.IsCity && !endTile.IsRoad()) {
					t.Errorf("connectCityBFS path endpoint %v is not a valid target (city or road) %v", endPoint, endTile)
				}
			}
		})
	}
}

func TestTileToPixel(t *testing.T) {
	// Initialize a level with known tile dimensions
	l := &Level{
		tileWidth:  200,
		tileHeight: 100,
	}

	testCases := []struct {
		x, y         int
		wantX, wantY int
	}{
		{0, 0, 100, 50},   // Row 0, Col 0: px=0, py=0 -> center (100, 50)
		{1, 0, 300, 50},   // Row 0, Col 1: px=200, py=0 -> center (300, 50)
		{0, 1, 200, 100},  // Row 1, Col 0: px=0+100, py=50 -> center (200, 100)
		{1, 1, 400, 100},  // Row 1, Col 1: px=200+100, py=50 -> center (400, 100)
		{0, 2, 100, 150},  // Row 2, Col 0: px=0, py=100 -> center (100, 150)
		{2, 3, 600, 200},  // Row 3, Col 2: px=400+100, py=150 -> center (600, 200)
	}

	for _, tc := range testCases {
		got := l.TileToPixel(image.Point{X: tc.x, Y: tc.y})
		if got.X != tc.wantX || got.Y != tc.wantY {
			t.Errorf("TileToPixel(%d, %d) = (%d, %d), want (%d, %d)", tc.x, tc.y, got.X, got.Y, tc.wantX, tc.wantY)
		}
	}
}
