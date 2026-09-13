package screens

import (
	"fmt"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

// Filter sprite sheet geometry. The sheet is a 9-column by 3-row grid of
// 40x40 icons with separate normal/hover/pressed sheets.
const (
	filterSheetCols = 9
	filterSheetRows = 3
	filterBtnSize   = 40
)

// Filter bar layout. The toggles sit in a single horizontal row just above the
// collection carousel: colors first, then a gap, then card types.
const (
	filterStartX           = 12
	filterBtnGap           = 4  // gap between adjacent buttons
	filterGroupGap         = 24 // gap between the color group and the type group
	filterGapAboveCarousel = 6  // gap between the filter row and the carousel top
	filterColorCount       = 5  // number of color toggles (rest are card types)
)

// filterButton wraps a sprite-sheet button with the filter toggle it controls,
// a predicate reporting whether that toggle is currently active, and the key its
// hover hint has in the game's cue-card table.
type filterButton struct {
	btn      *elements.Button
	cue      string
	toggle   func(f *collectionFilter)
	isActive func(f *collectionFilter) bool
}

// filterCell identifies a single icon within the filter sprite sheet along with
// the collection filter it toggles and the cue-card key for its hover hint.
type filterCell struct {
	row, col int
	cue      string
	toggle   func(f *collectionFilter)
	isActive func(f *collectionFilter) bool
}

func colorCell(row, col int, color string) filterCell {
	return filterCell{
		row: row, col: col,
		cue:      cueKeyForColor(color),
		toggle:   func(f *collectionFilter) { f.toggleColor(color) },
		isActive: func(f *collectionFilter) bool { return f.colors[color] },
	}
}

func typeCell(row, col int, t domain.CardType) filterCell {
	return filterCell{
		row: row, col: col,
		cue:      cueKeyForType(t),
		toggle:   func(f *collectionFilter) { f.toggleType(t) },
		isActive: func(f *collectionFilter) bool { return f.types[t] },
	}
}

// filterColorCells/filterTypeCells map the user-provided sprite-sheet layout
// (row, col) to the filter each icon controls.
var filterColorCells = []filterCell{
	colorCell(2, 8, "W"),
	colorCell(0, 7, "U"),
	colorCell(1, 6, "B"),
	colorCell(2, 4, "R"),
	colorCell(1, 5, "G"),
}

var filterTypeCells = []filterCell{
	typeCell(1, 1, domain.CardTypeCreature),
	typeCell(1, 3, domain.CardTypeSorcery),
	typeCell(1, 7, domain.CardTypeInstant),
	typeCell(2, 7, domain.CardTypeArtifact),
	typeCell(2, 0, domain.CardTypeLand),
}

// createFilterButtons slices the filter sprite sheets and builds the toggle
// buttons (colors first, then card types). Positions are assigned each frame by
// layoutFilterButtons since they depend on the screen height.
func createFilterButtons() ([]*filterButton, error) {
	normal, err := imageutil.LoadSpriteSheet(filterSheetCols, filterSheetRows, assets.EditDeckFilterSheet_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load filter sheet: %w", err)
	}
	hover, err := imageutil.LoadSpriteSheet(filterSheetCols, filterSheetRows, assets.EditDeckFilterSheetHover_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load filter hover sheet: %w", err)
	}
	pressed, err := imageutil.LoadSpriteSheet(filterSheetCols, filterSheetRows, assets.EditDeckFilterSheetPressed_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load filter pressed sheet: %w", err)
	}

	build := func(cells []filterCell) []*filterButton {
		out := make([]*filterButton, 0, len(cells))
		for _, c := range cells {
			btn := elements.NewButton(normal[c.row][c.col], hover[c.row][c.col], pressed[c.row][c.col], 0, 0, 1.0)
			out = append(out, &filterButton{btn: btn, cue: c.cue, toggle: c.toggle, isActive: c.isActive})
		}
		return out
	}

	buttons := build(filterColorCells)
	buttons = append(buttons, build(filterTypeCells)...)
	return buttons, nil
}

// layoutFilterButtons positions the toggles in a horizontal row just above the
// collection carousel: colors, a group gap, then card types.
func (s *EditDeckScreen) layoutFilterButtons(H int) {
	y := H - COLLECTION_HEIGHT - filterBtnSize - filterGapAboveCarousel
	x := filterStartX
	for i, fb := range s.filterButtons {
		if i == filterColorCount {
			x += filterGroupGap
		}
		fb.btn.MoveTo(x, y)
		x += filterBtnSize + filterBtnGap
	}
}

// updateFilterButtons processes clicks on the filter toggles. It returns true
// when a toggle changed, so the caller can rebuild the collection list.
func (s *EditDeckScreen) updateFilterButtons(opts *ebiten.DrawImageOptions, scale float64, W, H int) bool {
	s.layoutFilterButtons(H)
	changed := false
	for _, fb := range s.filterButtons {
		fb.btn.Update(opts, scale, W, H)
		if fb.btn.IsClicked() {
			fb.toggle(&s.filter)
			fb.btn.State = elements.StateNormal
			changed = true
		}
	}
	return changed
}

// filterDrawState returns the visual state a filter toggle must be drawn in. A
// toggle that is on is always drawn pressed, because that sprite is this screen's
// only signal that the filter is active — hovering it must not lift it, or an
// active filter and an inactive one look identical under the cursor.
func filterDrawState(active bool, current elements.ButtonState) elements.ButtonState {
	if active {
		return elements.StatePressed
	}
	return current
}

// drawFilterButtons renders the filter toggles, showing the pressed sprite for
// active toggles so the current filter state is visible at a glance.
func (s *EditDeckScreen) drawFilterButtons(screen *ebiten.Image, scale float64) {
	identity := &ebiten.DrawImageOptions{}
	for _, fb := range s.filterButtons {
		saved := fb.btn.State
		fb.btn.State = filterDrawState(fb.isActive(&s.filter), saved)
		fb.btn.Draw(screen, identity, scale)
		fb.btn.State = saved
	}
}
