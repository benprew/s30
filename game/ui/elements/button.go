package elements

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

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
	Font         text.Face // Changed from *font.Face to text.Face
	TextColor    color.Color
	TextOffset   image.Point // offset relative to button 0,0
	HandlerFuncs []func()    // handle click
	State        ButtonState
	X            int
	Y            int
}

// Draw renders the button onto the screen.
// It draws the appropriate button image based on its state and overlays the text.
func (b *Button) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	var imgToDraw *ebiten.Image

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Concat(opts.GeoM)
	options.GeoM.Translate(float64(b.X), float64(b.Y))

	switch b.State {
	case StateHover:
		imgToDraw = b.Hover
	case StatePressed:
		imgToDraw = b.Pressed
	case StateNormal:
		fallthrough
	default:
		imgToDraw = b.Normal
	}

	screen.DrawImage(imgToDraw, options)

	if b.Text != "" {
		options.GeoM.Translate(float64(b.TextOffset.X), float64(b.TextOffset.Y))
		R, G, B, A := b.TextColor.RGBA()
		options.ColorScale.Scale(float32(R)/65535, float32(G)/65535, float32(B)/65535, float32(A)/65535)
		textOpts := text.DrawOptions{DrawImageOptions: *options}
		text.Draw(screen, b.Text, b.Font, &textOpts)
	}
}

// Update checks the button's state based on mouse interaction and the provided drawing options.
func (b *Button) Update(opts *ebiten.DrawImageOptions) {
	mx, my := ebiten.CursorPosition()

	bounds := b.Normal.Bounds()
	buttonWidth := bounds.Dx()
	buttonHeight := bounds.Dy()
	combinedGeoM := ebiten.GeoM{}
	combinedGeoM.Concat(opts.GeoM)

	scaledWidth, scaledHeight := combinedGeoM.Apply(float64(buttonWidth), float64(buttonHeight))

	bx := b.X
	by := b.Y

	if mx >= bx && mx < bx+int(scaledWidth) && my >= by && my < by+int(scaledHeight) {
		if b.State == StatePressed && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fmt.Printf("Button Clicked: %s\n", b.Text)
			for _, handler := range b.HandlerFuncs {
				if handler != nil {
					handler()
				}
			}
		}

		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fmt.Println("Pressed")
			b.State = StatePressed
		} else {
			fmt.Println("Hover", b.Text)
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
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(1*scale, 55.0*scale)
	combinedImage.DrawImage(txtBox, op)
	return combinedImage
}
