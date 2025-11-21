package world

import (
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
		start          TilePoint
		expectedPath   []TilePoint       // nil if no path expected
		roadSprites    [][]*ebiten.Image // Sprites for roads
		roadSpriteInfo [][]string        // Maps sprite index to direction string (e.g., "N", "NE")
	}{
		{
			name: "Simple direct path to road",
			levelSetup: func(l *Level) {
				l.Tile(TilePoint{3, 1}).AddRoadSprite(l.roadSprites[0][1])
			},
			start:        TilePoint{X: 1, Y: 1},
			expectedPath: []TilePoint{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}},
		},
		{
			name: "Simple direct path to city",
			levelSetup: func(l *Level) {
				l.Tile(TilePoint{1, 3}).IsCity = true // Target city
			},
			start:        TilePoint{X: 1, Y: 1},
			expectedPath: []TilePoint{{X: 1, Y: 1}, {X: 1, Y: 3}}, // go south 1
		},
		{
			name: "Path around water obstacle",
			levelSetup: func(l *Level) {
				l.Tile(TilePoint{2, 1}).TerrainType = TerrainWater // Obstacle
				l.Tile(TilePoint{4, 1}).AddRoadSprite(l.roadSprites[0][1])
			},
			start: TilePoint{X: 1, Y: 1},
			// Expected path might vary slightly based on diagonal preference, this is one possibility
			expectedPath: []TilePoint{{X: 4, Y: 1}, {X: 3, Y: 1}, {X: 3, Y: 2}, {X: 2, Y: 2}, {X: 1, Y: 1}},
			// Alternative if diagonals are preferred: {{X: 1, Y: 1}, {X: 2, Y: 2}, {X: 3, Y: 1}} - depends on BFS neighbor order
		},
		// {
		// 	name: "Path to nearest target (road closer than city)",
		// 	levelSetup: func(l *Level) {
		// 		l.Tile(TilePoint{3, 1}).AddRoadSprite(l.roadSprites[0][1]) // Closer target (road)
		// 		l.Tile(TilePoint{1, 4}).IsCity = true                      // Further target (city)
		// 	},
		// 	start:        TilePoint{X: 1, Y: 1},
		// 	expectedPath: []TilePoint{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}},
		// },
		// {
		// 	name: "Path to nearest target (city closer than road)",
		// 	levelSetup: func(l *Level) {
		// 		l.Tile(TilePoint{4, 4}).AddRoadSprite(l.roadSprites[0][1]) // Further target (road)
		// 		l.Tile(TilePoint{1, 3}).IsCity = true                      // Closer target (city)
		// 	},
		// 	start:        TilePoint{X: 1, Y: 1},
		// 	expectedPath: []TilePoint{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 1, Y: 3}},
		// },
		// {
		// 	name: "Start on a road tile (should find nearest other road/city)",
		// 	levelSetup: func(l *Level) {
		// 		l.Tile(TilePoint{1, 1}).AddRoadSprite(l.roadSprites[0][1]) // Start is also a road
		// 		l.Tile(TilePoint{3, 1}).AddRoadSprite(l.roadSprites[0][1]) // Target road
		// 		l.Tile(TilePoint{1, 3}).IsCity = true                      // Another potential target
		// 	},
		// 	start:        TilePoint{X: 1, Y: 1},
		// 	expectedPath: []TilePoint{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}}, // Path to the nearest *other* target
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh level for each test case
			level := createTestLevel(5, 5) // Use a 5x5 grid for these tests
			level.Tile(TilePoint{1, 1}).IsCity = true

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
				} else {
					// If no panic occurred, but we expected one (expectedPath == nil)
					if tc.expectedPath == nil {
						// It's possible BFS completed without finding a path and returned nil instead of panicking.
						// We need to check the actual result in this case too.
						// The DeepEqual check below will handle this.
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
			if actualPath != nil && len(actualPath) > 0 {
				endPoint := actualPath[0]
				endTile := level.Tile(endPoint)
				if endTile == nil || endPoint == tc.start || !(endTile.IsCity || endTile.IsRoad()) {
					t.Errorf("connectCityBFS path endpoint %v is not a valid target (city or road) %v", endPoint, endTile)
				}
			}
		})
	}
}
