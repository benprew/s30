package screens

import (
	"bytes"
	"image"
	"testing"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
)

func TestDungeonSheetSelectionUsesOrdinaryThemes(t *testing.T) {
	tests := []struct {
		theme domain.DungeonTheme
		want  []byte
	}{
		{domain.DungeonThemeOne, assets.Dungeon1_png},
		{domain.DungeonThemeTwo, assets.Dungeon2_png},
		{domain.DungeonThemeThree, assets.Dungeon3_png},
	}
	for _, tc := range tests {
		dungeon := &domain.Dungeon{Color: domain.ColorRed, Theme: tc.theme}
		if got := dungeonSheetBytesFor(dungeon); !bytes.Equal(got, tc.want) {
			t.Errorf("theme %d selected the wrong sheet", tc.theme)
		}
		if got := dungeonSheetRowsFor(dungeon); got != 4 {
			t.Errorf("theme %d has %d rows, want 4", tc.theme, got)
		}
	}
}

func TestDungeonBackgroundUsesLogicalScreenSize(t *testing.T) {
	background := loadDungeonBackground()
	if background.Bounds().Dx() != 1024 || background.Bounds().Dy() != 768 {
		t.Fatalf("background size = %dx%d, want 1024x768", background.Bounds().Dx(), background.Bounds().Dy())
	}
}

func TestDungeonSheetSelectionUsesColorOnlyForCastles(t *testing.T) {
	dungeon := &domain.Dungeon{Color: domain.ColorBlue, Theme: domain.DungeonThemeCastle}
	if got := dungeonSheetBytesFor(dungeon); !bytes.Equal(got, assets.DungeonU_png) {
		t.Fatal("blue castle did not select the blue dungeon sheet")
	}
	if got := dungeonSheetRowsFor(dungeon); got != 5 {
		t.Fatalf("castle sheet has %d rows, want 5", got)
	}
}

func TestBossSpriteColumnFacesAdjoiningCorridor(t *testing.T) {
	directions := []struct {
		name  string
		delta image.Point
		want  int
	}{
		{"down", image.Pt(0, 1), 1},
		{"left", image.Pt(-1, 0), 2},
		{"right", image.Pt(1, 0), 3},
		{"up", image.Pt(0, -1), 4},
	}
	for _, tc := range directions {
		t.Run(tc.name, func(t *testing.T) {
			grid := makeWallsForBossTest(3)
			center := image.Pt(1, 1)
			grid[center.Y][center.X].Type = domain.DungeonTileEnemy
			p := center.Add(tc.delta)
			grid[p.Y][p.X].Type = domain.DungeonTileEmpty
			dungeon := &domain.Dungeon{Grid: grid}
			if got := bossSpriteColumn(dungeon, center); got != tc.want {
				t.Fatalf("got column %d, want %d", got, tc.want)
			}
		})
	}
}

func makeWallsForBossTest(size int) [][]domain.DungeonTile {
	grid := make([][]domain.DungeonTile, size)
	for y := range grid {
		grid[y] = make([]domain.DungeonTile, size)
		for x := range grid[y] {
			grid[y][x].Type = domain.DungeonTileWall
		}
	}
	return grid
}

func TestDungeonPointerDirection(t *testing.T) {
	origin := image.Pt(400, 300)
	tests := []struct {
		name    string
		point   image.Point
		clicked bool
		dx, dy  int
	}{
		{name: "not clicked", point: image.Pt(500, 300)},
		{name: "dead zone", point: image.Pt(410, 300), clicked: true},
		{name: "up left", point: image.Pt(368, 284), clicked: true, dx: -1},
		{name: "up right", point: image.Pt(432, 284), clicked: true, dy: -1},
		{name: "down right", point: image.Pt(432, 316), clicked: true, dx: 1},
		{name: "down left", point: image.Pt(368, 316), clicked: true, dy: 1},
		{name: "far right resolves toward down right", point: image.Pt(500, 300), clicked: true, dx: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dx, dy := dungeonPointerDirection(test.point, origin, test.clicked)
			if dx != test.dx || dy != test.dy {
				t.Fatalf("direction = (%d, %d), want (%d, %d)", dx, dy, test.dx, test.dy)
			}
		})
	}
}
