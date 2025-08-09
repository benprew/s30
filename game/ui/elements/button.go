package elements

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/game/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type HorizontalAlignment int
type VerticalAlignment int

const (
	AlignCenter HorizontalAlignment = iota
	AlignLeft
	AlignRight
)

const (
	AlignMiddle VerticalAlignment = iota
	AlignTop
	AlignBottom
)

type ButtonState int

const (
	StateNormal ButtonState = iota
	StateHover
	StatePressed
	StateClicked  // Click is registered on mouseup when button already pressed
	StateDisabled // Added for completeness, though not used in Draw yet
)

type ButtonText struct {
	Text             string
	Font             text.Face
	TextColor        color.Color
	TextOffset       image.Point // offset relative to button 0,0
	HorizontalCenter HorizontalAlignment
	VerticalCenter   VerticalAlignment
}

type Button struct {
	Hover        *ebiten.Image
	Normal       *ebiten.Image
	Pressed      *ebiten.Image
	ButtonText   ButtonText
	HandlerFuncs []func() // handle click
	State        ButtonState
	X            int
	Y            int
	Scale        float64
	ButtonID     string
}

func scaleImage(img *ebiten.Image, scale float64) *ebiten.Image {
	X := img.Bounds().Dx()
	Y := img.Bounds().Dy()

	newImg := ebiten.NewImage(int(float64(X)*scale), int(float64(Y)*scale))

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	newImg.DrawImage(img, opts)
	return newImg
}

// Draw renders the button onto the screen.
// It draws the appropriate button image based on its state and overlays the text.
func (b *Button) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions, scale float64) {
	var imgToDraw *ebiten.Image

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Concat(opts.GeoM)
	options.GeoM.Translate(float64(b.X)*scale, float64(b.Y)*scale)
	switch b.State {
	case StateHover:
		imgToDraw = b.Hover
	case StateClicked:
		fallthrough
	case StatePressed:
		imgToDraw = b.Pressed
	case StateNormal:
		fallthrough
	default:
		imgToDraw = b.Normal
	}

	if b.Scale != 0 {
		imgToDraw = scaleImage(imgToDraw, b.Scale)
	}
	screen.DrawImage(imgToDraw, options)

	if b.ButtonText.Text != "" {
		textX, textY := AlignText(imgToDraw, b.ButtonText.Text, b.ButtonText.Font, b.ButtonText.HorizontalCenter, b.ButtonText.VerticalCenter)
		options.GeoM.Translate(textX, textY)
		R, G, B, A := b.ButtonText.TextColor.RGBA()
		options.ColorScale.Scale(float32(R)/65535, float32(G)/65535, float32(B)/65535, float32(A)/65535)
		textOpts := text.DrawOptions{DrawImageOptions: *options}
		text.Draw(screen, b.ButtonText.Text, b.ButtonText.Font, &textOpts)
	}
}

// Update checks the button's state based on mouse interaction. Button box is button size + scale. Scale is passed in opts
func (b *Button) Update(opts *ebiten.DrawImageOptions, scale float64) {
	mx, my := ui.TouchPosition()
	isTouch := mx > 0
	if mx == 0 {
		mx, my = ebiten.CursorPosition()
	}

	bounds := b.Normal.Bounds()
	buttonWidth := float64(bounds.Dx()) * scale
	buttonHeight := float64(bounds.Dy()) * scale
	combinedGeoM := ebiten.GeoM{}
	combinedGeoM.Concat(opts.GeoM)
	if b.Scale != 0 {
		combinedGeoM.Scale(b.Scale, b.Scale)
	}

	scaledWidth, scaledHeight := combinedGeoM.Apply(float64(buttonWidth), float64(buttonHeight))

	if mx >= b.X && mx < b.X+int(scaledWidth) && my >= b.Y && my < b.Y+int(scaledHeight) {
		// if b.State == StatePressed && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		//  for _, handler := range b.HandlerFuncs {
		//      if handler != nil {
		//          handler()
		//      }
		//  }
		// }

		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fmt.Println("Pressed")
			b.State = StatePressed
		} else if b.State == StatePressed && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || isTouch {
			fmt.Printf("Button Clicked: %s\n", b.ButtonText.Text)
			b.State = StateClicked
		} else {
			fmt.Println("Hover", b.ButtonText.Text)
			b.State = StateHover
		}
	} else {
		b.State = StateNormal
	}
}

// CombineButton combines the 3 images into a single button image
// Moved from game/screens/city.go
func CombineButton(btnFrame, btnIcon, txtBox *ebiten.Image, scale float64) *ebiten.Image {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	combinedImage := ebiten.NewImage(120, 100)
	combinedImage.DrawImage(btnFrame, op)
	op.GeoM.Translate(8.0*scale, 5.0*scale)
	combinedImage.DrawImage(btnIcon, op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale+1.2, scale+0.6)
	op.GeoM.Translate(1*scale, 55.0*scale)
	combinedImage.DrawImage(txtBox, op)
	return combinedImage
}

func AlignText(imgToDraw *ebiten.Image, txt string, font text.Face, hAlign HorizontalAlignment, vAlign VerticalAlignment) (float64, float64) {
	// Get button dimensions
	bounds := imgToDraw.Bounds()
	buttonWidth := float64(bounds.Dx())
	buttonHeight := float64(bounds.Dy())

	// Measure text dimensions
	textWidth, textHeight := text.Measure(txt, font, 0)

	// Calculate horizontal position
	var textX float64
	switch hAlign {
	case AlignLeft:
		textX = 0
	case AlignCenter:
		textX = (buttonWidth - textWidth) / 2
	case AlignRight:
		textX = buttonWidth - textWidth
	}

	// Calculate vertical position
	var textY float64
	switch vAlign {
	case AlignTop:
		textY = 0
	case AlignMiddle:
		textY = (buttonHeight - textHeight) / 2
	case AlignBottom:
		textY = buttonHeight - textHeight
	}

	return textX, textY
}
