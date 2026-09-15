package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/ui/screenui"
)

// The menu has to fit on the screen it floats over, and every row of both faces
// has to sit inside the panel, or a row would be drawn where no click can reach it.
func TestGameMenuFitsOnScreenAndRowsSitInsideIt(t *testing.T) {
	const W = 1024
	panel := gameMenuBounds(W)
	screenRect := image.Rect(0, 0, W, 768)
	if !panel.In(screenRect) {
		t.Fatalf("panel %v does not fit on the screen %v", panel, screenRect)
	}
	if len(gameMenuRows) == 0 {
		t.Fatal("the menu has no rows")
	}
	for i := range gameMenuRows {
		if row := gameMenuRowBounds(i, W); !row.In(panel) {
			t.Errorf("row %d of the list %v is not inside the panel %v", i, row, panel)
		}
	}
	for i := range gameMenuConfirmRows {
		if row := gameMenuRowBounds(i, W); !row.In(panel) {
			t.Errorf("row %d of the question %v is not inside the panel %v", i, row, panel)
		}
	}
}

// Rows must not overlap, or a click between two labels would be ambiguous.
func TestGameMenuRowsDoNotOverlap(t *testing.T) {
	const W = 1024
	rows := len(gameMenuRows)
	if len(gameMenuConfirmRows) > rows {
		rows = len(gameMenuConfirmRows)
	}
	for i := 1; i < rows; i++ {
		if gameMenuRowBounds(i, W).Overlaps(gameMenuRowBounds(i-1, W)) {
			t.Errorf("rows %d and %d overlap", i-1, i)
		}
	}
}

// Every row of the list has to lead somewhere else: one pointing at the menu
// itself, or at no screen, would be a button that does nothing.
func TestGameMenuRowsLeadToOtherScreens(t *testing.T) {
	for _, row := range gameMenuRows {
		switch row.target {
		case screenui.NoScr, screenui.PopScr, screenui.GameMenuScr:
			t.Errorf("row %q opens %v, which is not a screen to go to", row.label, row.target)
		}
		if row.label == "" {
			t.Error("a row has no label")
		}
	}
}

// Quit asks first. The original puts "Ready to Quit? No. Yes." behind it, and a
// row that left the game on the first click would be the wrong answer to that.
func TestGameMenuAsksBeforeQuitting(t *testing.T) {
	const W = 1024
	quit := gameMenuRowBounds(0, W).Min.Add(image.Pt(5, 5))

	step, name := gameMenuChoice(quit, true, W, menuStepRows)
	if step != menuStepConfirmQuit {
		t.Errorf("clicking Quit showed face %v, want the question", step)
	}
	if name != screenui.GameMenuScr {
		t.Errorf("clicking Quit went to %v, want to stay in the menu", name)
	}

	yes := gameMenuRowBounds(1, W).Min.Add(image.Pt(5, 5))
	if step, name := gameMenuChoice(yes, true, W, menuStepConfirmQuit); name != screenui.StartScr || step != menuStepRows {
		t.Errorf("answering Yes gave (%v, %v), want the list and StartScr", step, name)
	}

	no := gameMenuRowBounds(0, W).Min.Add(image.Pt(5, 5))
	if _, name := gameMenuChoice(no, true, W, menuStepConfirmQuit); name != screenui.GameMenuScr {
		t.Errorf("answering No went to %v, want to stay in the menu", name)
	}
}

// The question uses the game's own words (Advstrings.txt line 9).
func TestGameMenuConfirmUsesTheOriginalsWords(t *testing.T) {
	if gameMenuConfirmTitle != "Ready to Quit?" {
		t.Errorf("confirm title = %q, want %q", gameMenuConfirmTitle, "Ready to Quit?")
	}
	want := []string{"No.", "Yes."}
	if len(gameMenuConfirmRows) != len(want) {
		t.Fatalf("the question has %d answers, want %d", len(gameMenuConfirmRows), len(want))
	}
	for i, label := range want {
		if gameMenuConfirmRows[i].label != label {
			t.Errorf("answer %d = %q, want %q", i, gameMenuConfirmRows[i].label, label)
		}
	}
}

// The list face: a row opens its screen, the panel's dead space keeps the menu,
// and anywhere else pops back to the world.
func TestGameMenuChoice(t *testing.T) {
	const W = 1024
	panel := gameMenuBounds(W)
	seeMap := gameMenuRowBounds(1, W).Min.Add(image.Pt(5, 5))
	titleArea := image.Pt(panel.Min.X+5, panel.Min.Y+2)
	outside := image.Pt(panel.Min.X-20, panel.Min.Y+5)

	cases := []struct {
		name    string
		pos     image.Point
		clicked bool
		want    screenui.ScreenName
	}{
		{"a click on See Map opens the map", seeMap, true, screenui.MiniMapScr},
		{"a click on the panel's title area keeps the menu", titleArea, true, screenui.GameMenuScr},
		{"a click outside pops back to the world", outside, true, screenui.PopScr},
		{"no click this frame leaves the menu open", seeMap, false, screenui.GameMenuScr},
	}
	for _, c := range cases {
		if _, got := gameMenuChoice(c.pos, c.clicked, W, menuStepRows); got != c.want {
			t.Errorf("%s: gameMenuChoice(%v, %v) = %v, want %v", c.name, c.pos, c.clicked, got, c.want)
		}
	}
}

// The button has to be reachable by touch, on the screen, and inside the frame's
// playable area rather than on its border.
func TestWorldMenuButtonSitsInTheFrameCorner(t *testing.T) {
	b := worldMenuButtonBounds()
	if !b.In(image.Rect(0, 0, 1024, 768)) {
		t.Fatalf("button %v is not on the screen", b)
	}
	if b.Dx() < 44 || b.Dy() < 44 {
		t.Errorf("button is %dx%d, too small to hit with a finger", b.Dx(), b.Dy())
	}
	// The frame's art ends at x=922 and its top strip at y=66; a button outside
	// that corner sits on the border, not on the map.
	if b.Max.X > worldMenuButtonRight {
		t.Errorf("button ends at x=%d, past the frame's playable area (%d)", b.Max.X, worldMenuButtonRight)
	}
	if b.Min.Y < worldMenuButtonTop {
		t.Errorf("button starts at y=%d, above the frame's playable area (%d)", b.Min.Y, worldMenuButtonTop)
	}
}

// Either input opens the menu: the button for touch and mouse, Escape for the
// keyboard path the original used.
func TestWorldMenuOpensOnClickOrEscape(t *testing.T) {
	cases := []struct{ clicked, escape, want bool }{
		{false, false, false},
		{true, false, true},
		{false, true, true},
		{true, true, true},
	}
	for _, c := range cases {
		if got := worldMenuOpens(c.clicked, c.escape); got != c.want {
			t.Errorf("worldMenuOpens(%v, %v) = %v, want %v", c.clicked, c.escape, got, c.want)
		}
	}
}
