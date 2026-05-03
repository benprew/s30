package domain

import (
	"image"
	"testing"
)

// makeRow builds a single-row dungeon for visibility tests where each rune
// maps to a tile type: '.' empty, '#' wall, 'E' enemy, 'P' player origin.
func makeRow(t *testing.T, layout string) (*Dungeon, image.Point) {
	t.Helper()
	d := &Dungeon{Grid: makeWalls(len(layout), 1)}
	var origin image.Point
	for x, r := range layout {
		switch r {
		case '.':
			d.Grid[0][x].Type = DungeonTileEmpty
		case '#':
			d.Grid[0][x].Type = DungeonTileWall
		case 'E':
			d.Grid[0][x].Type = DungeonTileEnemy
		case 'P':
			d.Grid[0][x].Type = DungeonTileEmpty
			origin = image.Point{X: x, Y: 0}
		default:
			t.Fatalf("unknown rune %q", r)
		}
	}
	return d, origin
}

func TestRevealFromStopsAtWall(t *testing.T) {
	d, origin := makeRow(t, "..P..#..")

	d.RevealFrom(origin)

	expectSeen := []int{0, 1, 2, 3, 4, 5}
	expectHidden := []int{6, 7}
	for _, x := range expectSeen {
		if !d.Grid[0][x].Seen {
			t.Errorf("tile %d should be Seen", x)
		}
	}
	for _, x := range expectHidden {
		if d.Grid[0][x].Seen {
			t.Errorf("tile %d should be hidden (past wall)", x)
		}
	}
}

func TestRevealFromStopsAtEnemyButRevealsTheEnemy(t *testing.T) {
	d, origin := makeRow(t, "P..E..")

	d.RevealFrom(origin)

	for _, x := range []int{0, 1, 2, 3} {
		if !d.Grid[0][x].Seen {
			t.Errorf("tile %d should be Seen (including the enemy)", x)
		}
	}
	for _, x := range []int{4, 5} {
		if d.Grid[0][x].Seen {
			t.Errorf("tile %d should be hidden (behind enemy)", x)
		}
	}
}

func TestRevealFromIsCumulative(t *testing.T) {
	d, _ := makeRow(t, "....P....")

	d.RevealFrom(image.Point{X: 4, Y: 0})

	// Move "player" one step left and reveal again — previously seen tiles
	// must remain seen.
	d.RevealFrom(image.Point{X: 3, Y: 0})

	for x := 0; x < d.Width(); x++ {
		if !d.Grid[0][x].Seen {
			t.Errorf("tile %d should be Seen after cumulative reveals", x)
		}
	}
}

func TestRevealFromCardinalOnly(t *testing.T) {
	// Build a small grid where a diagonal cell should NOT be revealed when
	// the player is at center and walls block cardinal lines from reaching
	// it.
	//
	//   . # .
	//   # P #
	//   . # .
	d := &Dungeon{Grid: makeWalls(3, 3)}
	for _, p := range []image.Point{{0, 0}, {2, 0}, {0, 2}, {2, 2}, {1, 1}} {
		d.Grid[p.Y][p.X].Type = DungeonTileEmpty
	}
	// Walls at (1,0), (1,2), (0,1), (2,1) are already walls from makeWalls.

	d.RevealFrom(image.Point{X: 1, Y: 1})

	for _, p := range []image.Point{{0, 0}, {2, 0}, {0, 2}, {2, 2}} {
		if d.Grid[p.Y][p.X].Seen {
			t.Errorf("diagonal tile %v should not be visible from center (no cardinal LoS)", p)
		}
	}
	// Walls themselves get revealed because the ray hits them.
	for _, p := range []image.Point{{1, 0}, {1, 2}, {0, 1}, {2, 1}} {
		if !d.Grid[p.Y][p.X].Seen {
			t.Errorf("blocking wall %v should be revealed", p)
		}
	}
}
