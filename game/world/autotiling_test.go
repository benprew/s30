package world

import (
	"image"
	"testing"
)

func tr(side string, source TileType, open1, open2 bool) Transition {
	return Transition{Side: side, SourceType: source, Open1: open1, Open2: open2}
}

func buildTileMap(t *testing.T, layout []string, expected map[image.Point][]Transition) {
	t.Helper()
	w := NewTileMap(len(layout[0]), len(layout))
	for y := range layout {
		for x, tile := range layout[y] {
			switch tile {
			case 'W':
				w.Set(x, y, TileWater)
			case 'P':
				w.Set(x, y, TilePlains)
			case 'F':
				w.Set(x, y, TileForest)
			default:
				t.Fatalf("unknown tile type: %c", tile)
			}
		}
	}

	for y := range layout {
		for x := range layout[y] {
			p := image.Point{x, y}
			got := GetTransitions(p, w)
			want := expected[p]
			if len(got) != len(want) {
				t.Errorf("%v: expected %d transitions, got %d", p, len(want), len(got))
				for _, tr := range got {
					t.Errorf("  got: %s", tr)
				}
				continue
			}
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("%v[%d]: expected %s, got %s", p, i, want[i], got[i])
				}
			}
		}
	}
}

func TestContinuousCoast(t *testing.T) {
	buildTileMap(t,
		[]string{
			"WW",
			"WP",
			"WP",
		},
		map[image.Point][]Transition{
			{1, 1}: {tr("NW", TileWater, false, true)},
			{1, 2}: {tr("NW", TileWater, true, false)},
		},
	)
}

func TestWaterBlock(t *testing.T) {
	buildTileMap(t,
		[]string{
			"WPPP",
			"PPPP",
			"PWWW",
			"PPPP",
		},
		map[image.Point][]Transition{
			{0, 1}: {tr("NW", TileWater, false, false), tr("SE", TileWater, false, false)},
			{1, 1}: {tr("SE", TileWater, false, false), tr("SW", TileWater, false, false)},
			{2, 1}: {tr("SE", TileWater, false, false), tr("SW", TileWater, false, false)},
			{3, 1}: {tr("SW", TileWater, false, false)},
			{0, 3}: {tr("NE", TileWater, false, false)},
			{1, 3}: {tr("NW", TileWater, false, false), tr("NE", TileWater, false, false)},
			{2, 3}: {tr("NW", TileWater, false, false), tr("NE", TileWater, false, false)},
			{3, 3}: {tr("NW", TileWater, false, false)},
		},
	)
}

func TestWrappedCoast(t *testing.T) {
	buildTileMap(t,
		[]string{
			"WW",
			"WP",
			"WW",
		},
		map[image.Point][]Transition{
			{1, 1}: {tr("NW", TileWater, false, true), tr("SW", TileWater, true, false)},
		},
	)
}

func TestIsland(t *testing.T) {
	buildTileMap(t,
		[]string{
			"WW",
			"WW",
			"WPW",
			"WW",
			"WW",
		},
		map[image.Point][]Transition{
			{1, 2}: {tr("NW", TileWater, true, true), tr("NE", TileWater, true, true), tr("SE", TileWater, true, true), tr("SW", TileWater, true, true)},
		},
	)
}
