package elements

import (
	"fmt"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/hajimehoshi/ebiten/v2"
)

// Orientation constants for ScrollableList
const (
	OrientationHorizontal = 0
	OrientationVertical   = 1
)

// ScrollableList is used to display a list of buttons.
// Buttons were used so that you can handle click events and also buttons contain text + and image.
//
// ScrollableList's by default show buttons on the both sides of the list based on orientation (vertical lists have vertical buttons)
//
// The buttons are "fwd" and "ff". Fwd buttons move the list one item at a time, and ff moves the list a full page.
//
// - items - List of buttons to display
// - background - the background image to use, will be scaled to width/height
// - width - list width
// - height - list height
// - orientation - enum of vertical or horizontal
type ScrollableList struct {
	items         []*Button
	background    *ebiten.Image
	width         int
	height        int
	orientation   int
	itemPadding   int // Pixel space between items in the list (defaults to 5)
	currentOffset int // Index of the first visible item
	visibleCount  int // Number of items that fit in the viewport

	// Navigation buttons
	fwdBackBtn *Button // Forward backward (one item back)
	ffBackBtn  *Button // Fast-forward backward (one page back)
	fwdBtn     *Button // Forward (one item forward)
	ffBtn      *Button // Fast-forward (one page forward)

	Position *layout.Position
}

// NewScrollableList creates a new ScrollableList UI element.
//
// ScrollableList is used to display a list of buttons.
// Buttons were used so that you can handle click events and also buttons contain text + and image.
//
// ScrollableList's by default show buttons on the both sides of the list based on orientation (vertical lists have vertical buttons)
//
// The buttons are "fwd" and "ff". Fwd buttons move the list one item at a time, and ff moves the list a full page.
//
// Parameters:
//   - items - List of buttons to display
//   - background - the background image to use, will be scaled to width/height
//   - width - list width
//   - height - list height
//   - orientation - enum of vertical or horizontal (OrientationHorizontal or OrientationVertical)
//   - itemPadding - pixel space between items (defaults to 5 if set to 0)
func NewScrollableList(
	items []*Button,
	background *ebiten.Image,
	width int,
	height int,
	orientation int,
	position *layout.Position,
) (*ScrollableList, error) {
	// Load the sprite map for navigation buttons
	sprites, err := imageutil.LoadMappedSprite(assets.MiniMapFrameSprite_png, assets.MiniMapFrameSprite_json)
	if err != nil {
		return nil, fmt.Errorf("failed to load map button sprites: %w", err)
	}

	// Scale the background to the desired dimensions
	scaledBg := background
	if background != nil {
		bgBounds := background.Bounds()
		scaleX := float64(width) / float64(bgBounds.Dx())
		scaleY := float64(height) / float64(bgBounds.Dy())
		scaledBg = imageutil.ScaleImageInd(background, scaleX, scaleY)
	}

	sl := &ScrollableList{
		items:         items,
		background:    scaledBg,
		width:         width,
		height:        height,
		orientation:   orientation,
		itemPadding:   15, // Default padding
		currentOffset: 0,
		Position:      position,
	}

	// Create navigation buttons based on orientation
	if orientation == OrientationVertical {
		// Vertical orientation: buttons at top and bottom
		// Top buttons (backward navigation)
		sl.ffBackBtn = NewButton(
			sprites["ff_norm"],
			sprites["ff_hover"],
			sprites["ff_press"],
			0, 0, 1.0,
		)
		// Set anchor relative to list position
		sl.ffBackBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX,
			OffsetY: position.OffsetY,
		}

		ffBackBtnWidth := sl.ffBackBtn.Bounds.Dx()
		sl.fwdBackBtn = NewButton(
			sprites["fwd_norm"],
			sprites["fwd_hover"],
			sprites["fwd_press"],
			ffBackBtnWidth, 0, 1.0,
		)
		sl.fwdBackBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX + ffBackBtnWidth,
			OffsetY: position.OffsetY,
		}

		// Bottom buttons (forward navigation)
		btnHeight := sl.ffBackBtn.Bounds.Dy()
		bottomY := height - btnHeight
		sl.fwdBtn = NewButton(
			sprites["fwd_norm"],
			sprites["fwd_hover"],
			sprites["fwd_press"],
			0, bottomY, 1.0,
		)
		sl.fwdBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX,
			OffsetY: position.OffsetY + bottomY,
		}

		fwdBtnWidth := sl.fwdBtn.Bounds.Dx()
		sl.ffBtn = NewButton(
			sprites["ff_norm"],
			sprites["ff_hover"],
			sprites["ff_press"],
			fwdBtnWidth, bottomY, 1.0,
		)
		sl.ffBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX + fwdBtnWidth,
			OffsetY: position.OffsetY + bottomY,
		}
	} else {
		// Horizontal orientation: buttons at bottom left and right
		// Use the same buttons as vertical, but rotate them 90 degrees clockwise

		// use Dy for W and Dx for H because they haven't been rotated
		btnWidth := sprites["fwd_norm"].Bounds().Dy()
		btnHeight := sprites["fwd_norm"].Bounds().Dx()

		// Rotate sprites 90 degrees clockwise for horizontal orientation
		rotateSprite := func(img *ebiten.Image, amt float64) *ebiten.Image {
			bounds := img.Bounds()
			rotated := ebiten.NewImage(bounds.Dy(), bounds.Dx())
			op := &ebiten.DrawImageOptions{}
			// Rotate 90 degrees clockwise: translate then rotate
			op.GeoM.Translate(-float64(bounds.Dx())/2, -float64(bounds.Dy())/2)
			op.GeoM.Rotate(amt * 3.14159 / 180) // 90 degrees in radians
			op.GeoM.Translate(float64(bounds.Dy())/2, float64(bounds.Dx())/2)
			rotated.DrawImage(img, op)
			return rotated
		}

		// Left buttons (backward navigation) at bottom left - rotated vertical backward buttons
		bottomY := height - btnHeight

		sl.ffBackBtn = NewButton(
			rotateSprite(sprites["ff_norm"], 90),
			rotateSprite(sprites["ff_hover"], 90),
			rotateSprite(sprites["ff_press"], 90),
			0, bottomY, 1.0,
		)
		sl.ffBackBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX,
			OffsetY: position.OffsetY + bottomY,
		}

		ffBackBtnWidth := sl.ffBackBtn.Bounds.Dx()
		sl.fwdBackBtn = NewButton(
			rotateSprite(sprites["fwd_norm"], 90),
			rotateSprite(sprites["fwd_hover"], 90),
			rotateSprite(sprites["fwd_press"], 90),
			ffBackBtnWidth, bottomY, 1.0,
		)
		sl.fwdBackBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX + ffBackBtnWidth,
			OffsetY: position.OffsetY + bottomY,
		}

		// Right buttons (forward navigation) at bottom right - rotated vertical forward buttons
		rightX := width - (2 * btnWidth)

		sl.fwdBtn = NewButton(
			rotateSprite(sprites["fwd_norm"], -90),
			rotateSprite(sprites["fwd_hover"], -90),
			rotateSprite(sprites["fwd_press"], -90),
			rightX, bottomY, 1.0,
		)
		sl.fwdBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX + rightX,
			OffsetY: position.OffsetY + bottomY,
		}

		sl.ffBtn = NewButton(
			rotateSprite(sprites["ff_norm"], -90),
			rotateSprite(sprites["ff_hover"], -90),
			rotateSprite(sprites["ff_press"], -90),
			rightX+btnWidth, bottomY, 1.0,
		)
		sl.ffBtn.Position = &layout.Position{
			Anchor:  position.Anchor,
			OffsetX: position.OffsetX + rightX + btnWidth,
			OffsetY: position.OffsetY + bottomY,
		}
	}

	// Position item buttons based on orientation
	// sl.positionItems() // Deprecated, positioning handled in Update

	// Calculate how many items can be visible at once
	sl.visibleCount = sl.calculateVisibleCount()
	fmt.Println("visibleCount:", sl.visibleCount)

	return sl, nil
}

// positionItems is deprecated and removed. Positioning is now handled dynamically in Update.

// calculateVisibleCount determines how many items fit in the viewport
func (sl *ScrollableList) calculateVisibleCount() int {
	if len(sl.items) == 0 {
		return 0
	}

	// Get the size of a single item
	firstItem := sl.items[0]
	itemBounds := firstItem.Bounds

	if sl.orientation == OrientationVertical {
		// Calculate based on height
		itemHeight := itemBounds.Dy() + sl.itemPadding
		// Reserve space for navigation buttons
		btnHeight := sl.fwdBtn.Bounds.Dy()
		availableHeight := sl.height - (2 * btnHeight)
		return availableHeight / itemHeight
	} else {
		// Calculate based on width
		itemWidth := itemBounds.Dx() + sl.itemPadding
		// Reserve space for navigation buttons
		btnWidth := sl.fwdBtn.Bounds.Dx()
		availableWidth := sl.width - (2 * btnWidth)
		return availableWidth / itemWidth
	}
}

// scrollForward moves the list forward by one item
func (sl *ScrollableList) scrollForward() {
	if sl.currentOffset+sl.visibleCount < len(sl.items) {
		sl.currentOffset++
	}
}

// scrollFastForward moves the list forward by one page
func (sl *ScrollableList) scrollFastForward() {
	newOffset := sl.currentOffset + sl.visibleCount
	if newOffset >= len(sl.items) {
		// Go to the last page
		sl.currentOffset = len(sl.items) - sl.visibleCount
		if sl.currentOffset < 0 {
			sl.currentOffset = 0
		}
	} else {
		sl.currentOffset = newOffset
	}
}

// scrollBackward moves the list backward by one item
func (sl *ScrollableList) scrollBackward() {
	if sl.currentOffset > 0 {
		sl.currentOffset--
	}
}

// scrollFastBackward moves the list backward by one page
func (sl *ScrollableList) scrollFastBackward() {
	newOffset := sl.currentOffset - sl.visibleCount
	if newOffset < 0 {
		sl.currentOffset = 0
	} else {
		sl.currentOffset = newOffset
	}
}

// Update handles button clicks and scrolling
func (sl *ScrollableList) Update(opts *ebiten.DrawImageOptions, scale float64, screenW, screenH int) {
	// Resolve list position
	offsetX, offsetY := sl.Position.Resolve(int(float64(screenW)/scale), int(float64(screenH)/scale))

	// Update navigation buttons state
	sl.fwdBtn.Update(opts, scale, screenW, screenH)
	sl.ffBtn.Update(opts, scale, screenW, screenH)
	sl.fwdBackBtn.Update(opts, scale, screenW, screenH)
	sl.ffBackBtn.Update(opts, scale, screenW, screenH)

	// Check for navigation button clicks
	if sl.fwdBtn.IsClicked() {
		sl.scrollForward()
		sl.fwdBtn.State = StateNormal
	}
	if sl.ffBtn.IsClicked() {
		sl.scrollFastForward()
		sl.ffBtn.State = StateNormal
	}
	if sl.fwdBackBtn.IsClicked() {
		sl.scrollBackward()
		sl.fwdBackBtn.State = StateNormal
	}
	if sl.ffBackBtn.IsClicked() {
		sl.scrollFastBackward()
		sl.ffBackBtn.State = StateNormal
	}

	// Update visible items positions
	endIdx := sl.currentOffset + sl.visibleCount
	if endIdx > len(sl.items) {
		endIdx = len(sl.items)
	}

	if sl.orientation == OrientationVertical {
		// Vertical: Stack items starting after top buttons
		currentY := sl.ffBackBtn.Bounds.Dy()

		for i := sl.currentOffset; i < endIdx; i++ {
			sl.items[i].MoveTo(offsetX, offsetY+currentY)
			sl.items[i].Update(opts, scale, screenW, screenH)
			currentY += sl.items[i].Bounds.Dy() + sl.itemPadding
		}
	} else {
		// Horizontal: Stack items starting after left buttons
		currentX := sl.ffBackBtn.Bounds.Dx() + sl.fwdBackBtn.Bounds.Dx()

		for i := sl.currentOffset; i < endIdx; i++ {
			sl.items[i].MoveTo(offsetX+currentX, offsetY)
			sl.items[i].Update(opts, scale, screenW, screenH)
			currentX += sl.items[i].Bounds.Dx() + sl.itemPadding
		}
	}
}

// Draw renders the list and visible items
func (sl *ScrollableList) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions, scale float64) {
	// Draw background using the passed options (which likely include translation)
	if sl.background != nil {
		screen.DrawImage(sl.background, opts)
	}

	// For buttons, they have been moved to absolute positions in Update().
	// So we should draw them with Identity options (no translation),
	// otherwise they will be double-translated.
	identityOpts := &ebiten.DrawImageOptions{}

	// Draw navigation buttons
	sl.ffBackBtn.Draw(screen, identityOpts, scale)
	sl.fwdBackBtn.Draw(screen, identityOpts, scale)
	sl.fwdBtn.Draw(screen, identityOpts, scale)
	sl.ffBtn.Draw(screen, identityOpts, scale)

	// Draw visible items
	endIdx := sl.currentOffset + sl.visibleCount
	if endIdx > len(sl.items) {
		endIdx = len(sl.items)
	}

	for i := sl.currentOffset; i < endIdx; i++ {
		sl.items[i].Draw(screen, identityOpts, scale)
	}
}

// drawVertical is no longer used as Draw handles everything
func (sl *ScrollableList) drawVertical(screen *ebiten.Image, opts *ebiten.DrawImageOptions, scale float64) {
	// Deprecated
}

// drawHorizontal is no longer used as Draw handles everything
func (sl *ScrollableList) drawHorizontal(screen *ebiten.Image, opts *ebiten.DrawImageOptions, scale float64) {
	// Deprecated
}

// GetItems returns the list of buttons in the scrollable list
func (sl *ScrollableList) GetItems() []*Button {
	return sl.items
}
