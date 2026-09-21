package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
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
	rows := max(len(gameMenuConfirmRows), len(gameMenuRows))
	for i := 1; i < rows; i++ {
		if gameMenuRowBounds(i, W).Overlaps(gameMenuRowBounds(i-1, W)) {
			t.Errorf("rows %d and %d overlap", i-1, i)
		}
	}
}

func findGameMenuRow(t *testing.T, label string) int {
	t.Helper()
	for i, r := range gameMenuRows {
		if r.label == label {
			return i
		}
	}
	t.Fatalf("row %q not found in gameMenuRows", label)
	return -1
}

// Every row of the list has to lead somewhere: Quit exits the game (QuitScr),
// See Map opens the minimap, Load opens the load overlay, and Save saves and continues (PopScr).
func TestGameMenuRowsLeadToOtherScreens(t *testing.T) {
	for _, row := range gameMenuRows {
		if row.label == "Save" {
			if row.target != screenui.PopScr {
				t.Errorf("row %q opens %v, want PopScr", row.label, row.target)
			}
			continue
		}
		if row.label == "Quit" {
			if row.target != screenui.QuitScr {
				t.Errorf("row %q opens %v, want QuitScr", row.label, row.target)
			}
			continue
		}
		switch row.target {
		case screenui.NoScr, screenui.PopScr, screenui.GameMenuScr, screenui.QuitScr:
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
	quitIdx := findGameMenuRow(t, "Quit")
	quit := gameMenuRowBounds(quitIdx, W).Min.Add(image.Pt(5, 5))

	step, name := gameMenuChoice(quit, true, W, menuStepRows)
	if step != menuStepConfirmQuit {
		t.Errorf("clicking Quit showed face %v, want the question", step)
	}
	if name != screenui.GameMenuScr {
		t.Errorf("clicking Quit went to %v, want to stay in the menu", name)
	}

	yes := gameMenuRowBounds(1, W).Min.Add(image.Pt(5, 5))
	if step, name := gameMenuChoice(yes, true, W, menuStepConfirmQuit); name != screenui.QuitScr || step != menuStepRows {
		t.Errorf("answering Yes gave (%v, %v), want the list and QuitScr", step, name)
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

// The list face: a row opens its screen, Save continues, the panel's dead space keeps the menu,
// and anywhere else pops back to the world.
func TestGameMenuChoice(t *testing.T) {
	const W = 1024
	panel := gameMenuBounds(W)
	seeMapIdx := findGameMenuRow(t, "See Map")
	seeMap := gameMenuRowBounds(seeMapIdx, W).Min.Add(image.Pt(5, 5))
	saveIdx := findGameMenuRow(t, "Save")
	savePt := gameMenuRowBounds(saveIdx, W).Min.Add(image.Pt(5, 5))
	loadIdx := findGameMenuRow(t, "Load")
	loadPt := gameMenuRowBounds(loadIdx, W).Min.Add(image.Pt(5, 5))
	titleArea := image.Pt(panel.Min.X+5, panel.Min.Y+2)
	outside := image.Pt(panel.Min.X-20, panel.Min.Y+5)

	cases := []struct {
		name    string
		pos     image.Point
		clicked bool
		want    screenui.ScreenName
	}{
		{"a click on See Map opens the map", seeMap, true, screenui.MiniMapScr},
		{"a click on Save continues the game", savePt, true, screenui.PopScr},
		{"a click on Load opens load game", loadPt, true, screenui.LoadGameScr},
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

// The button has to be reachable by touch, on the screen, and in the screen's
// upper right corner clear of the playable map frame.
func TestWorldMenuButtonSitsInTheScreenCorner(t *testing.T) {
	const W = 1024
	b := worldMenuButtonBounds(W)
	if !b.In(image.Rect(0, 0, W, 768)) {
		t.Fatalf("button %v is not on the screen", b)
	}
	if b.Dx() < 44 || b.Dy() < 44 {
		t.Errorf("button is %dx%d, too small to hit with a finger", b.Dx(), b.Dy())
	}
	if frameRight := FrameOffsetX + FrameWidth; b.Min.X < frameRight {
		t.Errorf("button starts at x=%d, inside the map frame's right edge (%d)", b.Min.X, frameRight)
	}
	if b.Max.Y > FrameOffsetY {
		t.Errorf("button ends at y=%d, below the map frame's top edge (%d)", b.Max.Y, FrameOffsetY)
	}
}

func TestGameMenuTitleIsLeftJustified(t *testing.T) {
	const W = 1024
	panel := gameMenuBounds(W)
	for _, titleText := range []string{gameMenuTitle, gameMenuConfirmTitle} {
		title := gameMenuTitleElement(panel, titleText)
		if title.HAlign != elements.AlignLeft {
			t.Errorf("title %q HAlign = %v, want AlignLeft (%v)", titleText, title.HAlign, elements.AlignLeft)
		}
		if title.X != panel.Min.X+gameMenuTitlePadX {
			t.Errorf("title %q X = %d, want %d", titleText, title.X, panel.Min.X+gameMenuTitlePadX)
		}
	}
}

func TestGameMenuSaveTriggersCallback(t *testing.T) {
	const W = 1024
	saved := false
	menu := NewGameMenuScreen(func() error {
		saved = true
		return nil
	})
	saveIdx := findGameMenuRow(t, "Save")
	savePt := gameMenuRowBounds(saveIdx, W).Min.Add(image.Pt(5, 5))

	name, _ := menu.handleChoice(savePt, true, W)
	if !saved {
		t.Error("clicking Save did not invoke onSave callback")
	}
	if name != screenui.PopScr {
		t.Errorf("name = %v, want PopScr", name)
	}
}

func TestGameMenuLoadReturnsLoadGameScreen(t *testing.T) {
	const W = 1024
	menu := NewGameMenuScreen()
	loadIdx := findGameMenuRow(t, "Load")
	loadPt := gameMenuRowBounds(loadIdx, W).Min.Add(image.Pt(5, 5))

	name, screen := menu.handleChoice(loadPt, true, W)
	if name != screenui.LoadGameScr {
		t.Errorf("name = %v, want LoadGameScr", name)
	}
	if screen == nil {
		t.Error("expected non-nil screen for LoadGameScr")
	}
}

func TestGameMenuYesReturnsQuitScr(t *testing.T) {
	const W = 1024
	menu := NewGameMenuScreen()
	quitIdx := findGameMenuRow(t, "Quit")
	quitPt := gameMenuRowBounds(quitIdx, W).Min.Add(image.Pt(5, 5))

	// First click Quit to show confirmation
	menu.handleChoice(quitPt, true, W)
	if menu.step != menuStepConfirmQuit {
		t.Fatalf("step = %v, want menuStepConfirmQuit", menu.step)
	}

	// Click "Yes." (row 1)
	yesPt := gameMenuRowBounds(1, W).Min.Add(image.Pt(5, 5))
	name, _ := menu.handleChoice(yesPt, true, W)
	if name != screenui.QuitScr {
		t.Errorf("name = %v, want QuitScr", name)
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

func TestGameMenuOpenAndClose(t *testing.T) {
	menu := NewGameMenuScreen()
	if menu.IsOpen() {
		t.Error("menu should start closed")
	}
	menu.Open()
	if !menu.IsOpen() {
		t.Error("menu should be open after Open()")
	}
	menu.Close()
	if menu.IsOpen() {
		t.Error("menu should be closed after Close()")
	}
}

func TestGameMenuNotFramed(t *testing.T) {
	menu := NewGameMenuScreen()
	if menu.IsFramed() {
		t.Error("game menu should not be framed by WorldFrame")
	}
}

func TestGameMenuUpdateClosedReturnsNoScr(t *testing.T) {
	menu := NewGameMenuScreen()
	name, scr, err := menu.UpdateMenu(1024, 768, 1.0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != screenui.NoScr {
		t.Errorf("name = %v, want NoScr when closed without interaction", name)
	}
	if scr != nil {
		t.Errorf("scr = %v, want nil", scr)
	}
}

func TestGameMenuDrawDoesNotPanic(t *testing.T) {
	menu := NewGameMenuScreen()
	screen := ebiten.NewImage(1024, 768)

	// Draw while closed
	menu.Draw(screen, 1024, 768, 1.0)

	// Draw while open
	menu.Open()
	menu.Draw(screen, 1024, 768, 1.0)
}

func TestMenuSquareImageLoads(t *testing.T) {
	img := getMenuSquareImage()
	if img == nil {
		t.Fatal("menu square image is nil")
	}
	if img.Bounds().Dx() != 24 || img.Bounds().Dy() != 24 {
		t.Errorf("menu square size = %dx%d, want 24x24", img.Bounds().Dx(), img.Bounds().Dy())
	}
}
