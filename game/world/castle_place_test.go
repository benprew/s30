package world

import (
	"encoding/json"
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestPlaceCastlesPlacesOnePerColor(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(1, nil, nil, nil, nil, nil)

	if got := len(l.Castles); got != 5 {
		t.Fatalf("expected 5 castles, got %d", got)
	}

	seen := map[domain.ColorMask]bool{}
	for _, c := range l.Castles {
		if seen[c.Color] {
			t.Errorf("color %s placed twice", domain.ColorMaskToString(c.Color))
		}
		seen[c.Color] = true
	}
	for _, color := range domain.GetAllAmuletColors() {
		if !seen[color] {
			t.Errorf("color %s missing", domain.ColorMaskToString(color))
		}
	}
}

func TestPlaceCastlesAttachesCastleToTile(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(2, nil, nil, nil, nil, nil)

	for _, c := range l.Castles {
		tile := l.Tile(c.MapTile)
		if tile == nil {
			t.Fatalf("tile missing at %v", c.MapTile)
		}
		if !tile.IsCastle {
			t.Errorf("tile at %v not marked IsCastle", c.MapTile)
		}
		if tile.Castle != c {
			t.Errorf("tile.Castle at %v does not point back to castle", c.MapTile)
		}
		if c.RogueName != castleRogues[c.Color] {
			t.Errorf("color %s: rogue name %q, want %q",
				domain.ColorMaskToString(c.Color), c.RogueName, castleRogues[c.Color])
		}
	}
}

func TestPlaceCastlesHonorsMinDistance(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(3, nil, nil, nil, nil, nil)

	for i, a := range l.Castles {
		for j, b := range l.Castles {
			if i == j {
				continue
			}
			d := absInt(a.MapTile.X-b.MapTile.X) + absInt(a.MapTile.Y-b.MapTile.Y)
			if d < 6 {
				t.Errorf("castles %d (%s) and %d (%s) only %d apart",
					i, domain.ColorMaskToString(a.Color),
					j, domain.ColorMaskToString(b.Color), d)
			}
		}
	}
}

func TestPlaceCastlesAvoidsPlayerStart(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(4, nil, nil, nil, nil, nil)

	cx, cy := l.W/2, l.H/2
	for _, c := range l.Castles {
		d := absInt(c.MapTile.X-cx) + absInt(c.MapTile.Y-cy)
		if d <= castlePlayerSafeRange {
			t.Errorf("castle %s at %v within %d tiles of player start",
				domain.ColorMaskToString(c.Color), c.MapTile, castlePlayerSafeRange)
		}
	}
}

func TestPlaceCastlesAvoidsWater(t *testing.T) {
	l := createTestLevel(40, 40)
	for x := 0; x < l.W; x++ {
		l.Tile(image.Point{x, 0}).TerrainType = TerrainWater
		l.Tile(image.Point{x, 1}).TerrainType = TerrainWater
		l.Tile(image.Point{x, l.H - 1}).TerrainType = TerrainWater
	}
	l.placeCastles(5, nil, nil, nil, nil, nil)

	for _, c := range l.Castles {
		tile := l.Tile(c.MapTile)
		if tile.TerrainType == TerrainWater {
			t.Errorf("castle %s placed on water at %v",
				domain.ColorMaskToString(c.Color), c.MapTile)
		}
	}
}

func TestPlaceCastlesStampsZoneTerrain(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(6, nil, nil, nil, nil, nil)

	for _, c := range l.Castles {
		expected := castleZoneTerrain[c.Color]
		// Sample a tile one step away from the castle center; it should
		// have been re-painted to the zone terrain (unless it was on or
		// near water, which the stamp skips).
		center := c.MapTile
		var found bool
		for dy := -castleZoneRadius; dy <= castleZoneRadius && !found; dy++ {
			for dx := -castleZoneRadius; dx <= castleZoneRadius && !found; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				p := image.Point{X: center.X + dx, Y: center.Y + dy}
				tile := l.Tile(p)
				if tile == nil {
					continue
				}
				if tile.TerrainType == expected {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("zone for %s castle has no %d-typed tile around %v",
				domain.ColorMaskToString(c.Color), expected, center)
		}
	}
}

func TestCastleRoguesCoversAllColors(t *testing.T) {
	for _, color := range domain.GetAllAmuletColors() {
		name, ok := castleRogues[color]
		if !ok || name == "" {
			t.Errorf("missing rogue mapping for %s", domain.ColorMaskToString(color))
			continue
		}
		if _, exists := domain.Rogues[name]; !exists {
			t.Errorf("rogue %q for %s not found in domain.Rogues",
				name, domain.ColorMaskToString(color))
		}
	}
}

func TestHandleCastleDuelOutcomeMarksDefeatedOnWin(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(7, nil, nil, nil, nil, nil)
	if len(l.Castles) == 0 {
		t.Fatal("no castles placed")
	}
	c := l.Castles[0]
	l.SetPendingCastle(c, c.MapTile)
	l.HandleCastleDuelOutcome(true)
	if !c.Defeated {
		t.Errorf("castle %s not marked defeated after win", domain.ColorMaskToString(c.Color))
	}
	if l.pendingCastle != nil {
		t.Errorf("pendingCastle not cleared after outcome")
	}
}

func TestCastleStateSurvivesJSONRoundTrip(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(11, nil, nil, nil, nil, nil)
	if len(l.Castles) == 0 {
		t.Fatal("no castles placed")
	}
	// Mark one castle defeated to ensure that bit persists.
	defeatedColor := l.Castles[0].Color
	defeatedTile := l.Castles[0].MapTile
	l.Castles[0].Defeated = true
	if tile := l.Tile(defeatedTile); tile != nil {
		tile.Castle.Defeated = true
	}

	data, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Level
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Castles) != len(l.Castles) {
		t.Fatalf("castle count: got %d, want %d", len(got.Castles), len(l.Castles))
	}

	var foundDefeated bool
	for _, c := range got.Castles {
		if c.Color != defeatedColor {
			continue
		}
		if !c.Defeated {
			t.Errorf("castle %s should be defeated after round-trip", domain.ColorMaskToString(c.Color))
		}
		if c.MapTile != defeatedTile {
			t.Errorf("castle %s MapTile: got %v, want %v",
				domain.ColorMaskToString(c.Color), c.MapTile, defeatedTile)
		}
		foundDefeated = true
	}
	if !foundDefeated {
		t.Errorf("did not find restored castle for color %s",
			domain.ColorMaskToString(defeatedColor))
	}

	tile := got.Tile(defeatedTile)
	if tile == nil || !tile.IsCastle || tile.Castle == nil {
		t.Fatalf("tile %v lost IsCastle/Castle after round-trip", defeatedTile)
	}
	if !tile.Castle.Defeated {
		t.Errorf("tile.Castle.Defeated lost after round-trip")
	}
}

func TestHandleCastleDuelOutcomeKeepsCastleOnLoss(t *testing.T) {
	l := createTestLevel(40, 40)
	l.placeCastles(8, nil, nil, nil, nil, nil)
	if len(l.Castles) == 0 {
		t.Fatal("no castles placed")
	}
	c := l.Castles[0]
	l.SetPendingCastle(c, c.MapTile)
	l.HandleCastleDuelOutcome(false)
	if c.Defeated {
		t.Errorf("castle %s should not be defeated after loss", domain.ColorMaskToString(c.Color))
	}
	if l.pendingCastle != nil {
		t.Errorf("pendingCastle not cleared after outcome")
	}
}
