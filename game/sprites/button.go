package sprites

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

// ButtonState defines the possible states of a button.
type ButtonState int

const (
	StateNormal ButtonState = iota
	StateHover
	StatePressed
	StateDisabled // Added for completeness, though not used in Draw yet
)

type Button struct {
	Hover        *ebiten.Image
	Normal       *ebiten.Image
	Pressed      *ebiten.Image
	Text         string
	Font         *font.Face
	TextColor    color.Color
	TextOffset   image.Point           // offset relative to button 0,0
	HandlerFuncs []*func(ebiten.Image) // handle hover, click, etc
	State        ButtonState           // Current state of the button
	X            int
	Y            int
}

// Draw renders the button onto the screen.
// It draws the appropriate button image based on its state and overlays the text.
func (b *Button) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	var imgToDraw *ebiten.Image

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Concat(options.GeoM)

	fmt.Println(options.GeoM)

	// Select the image based on the button's state
	switch b.State {
	case StateHover:
		imgToDraw = b.Hover
	case StatePressed:
		imgToDraw = b.Pressed
	case StateNormal:
		fallthrough // Explicit fallthrough for clarity
	default: // Includes StateDisabled or any unexpected state
		imgToDraw = b.Normal
	}

	options.GeoM.Translate(float64(b.X), float64(b.Y))
	screen.DrawImage(imgToDraw, options)

	// Draw the text if font and text are provided
	if b.Font != nil && b.Text != "" {
		// Calculate text position:
		// options.GeoM contains the translation for the button's top-left corner.
		// We need to extract this translation.
		tx := options.GeoM.Element(0, 2) // Translation X
		ty := options.GeoM.Element(1, 2) // Translation Y

		// Add the button's position and the text's relative offset
		textX := int(tx) + b.TextOffset.X
		textY := int(ty) + b.TextOffset.Y

		// Draw the text onto the screen
		text.Draw(screen, b.Text, *b.Font, textX, textY, b.TextColor)
	}
}
