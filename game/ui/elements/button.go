package elements

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
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
	Text       string
	Font       text.Face
	TextColor  color.Color
	TextOffset image.Point
	HAlign     HorizontalAlignment
	VAlign     VerticalAlignment
}

type Button struct {
	Normal       *ebiten.Image
	Hover        *ebiten.Image
	Pressed      *ebiten.Image
	ButtonText   ButtonText
	HandlerFuncs []func()
	State        ButtonState
	Bounds       image.Rectangle
	Position     *layout.Position
	ID           string
}

func NewButton(normal, hover, pressed *ebiten.Image, x, y int, scale float64) *Button {
	scaledNormal := normal
	scaledHover := hover
	scaledPressed := pressed

	if scale != 0 && scale != 1.0 {
		scaledNormal = imageutil.ScaleImage(normal, scale)
		scaledHover = imageutil.ScaleImage(hover, scale)
		scaledPressed = imageutil.ScaleImage(pressed, scale)
	}

	w := scaledNormal.Bounds().Dx()
	h := scaledNormal.Bounds().Dy()
	bounds := image.Rect(x, y, x+w, y+h)

	return &Button{
		Normal:  scaledNormal,
		Hover:   scaledHover,
		Pressed: scaledPressed,
		Bounds:  bounds,
		State:   StateNormal,
	}
}

func (b *Button) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions, scale float64) {
	var imgToDraw *ebiten.Image

	x, y := b.getPosition(screen, scale)

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Concat(opts.GeoM)
	options.GeoM.Translate(float64(x)*scale, float64(y)*scale)

	switch b.State {
	case StateHover:
		imgToDraw = b.Hover
	case StateClicked, StatePressed:
		imgToDraw = b.Pressed
	default:
		imgToDraw = b.Normal
	}

	screen.DrawImage(imgToDraw, options)

	if b.ButtonText.Text != "" {
		textX, textY := AlignText(imgToDraw, b.ButtonText.Text, b.ButtonText.Font, b.ButtonText.HAlign, b.ButtonText.VAlign)
		textOpts := &ebiten.DrawImageOptions{}
		textOpts.GeoM.Concat(options.GeoM)
		textOpts.GeoM.Translate(textX, textY)
		R, G, B, A := b.ButtonText.TextColor.RGBA()
		textOpts.ColorScale.Scale(float32(R)/65535, float32(G)/65535, float32(B)/65535, float32(A)/65535)
		text.Draw(screen, b.ButtonText.Text, b.ButtonText.Font, &text.DrawOptions{DrawImageOptions: *textOpts})
	}
}

func (b *Button) Update(opts *ebiten.DrawImageOptions, scale float64, screenW, screenH int) {
	mx, my := ui.TouchPosition()
	isTouch := mx > 0
	if mx == 0 {
		mx, my = ebiten.CursorPosition()
	}

	// TODO button position should be set by layout when created and stored in Bounds
	bounds := b.getPositionWithDims(screenW, screenH, scale)
	mp := image.Point{mx, my}

	combinedGeoM := ebiten.GeoM{}
	combinedGeoM.Concat(opts.GeoM)

	if mp.In(bounds) {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fmt.Println("Pressed")
			b.State = StatePressed
		} else if b.State == StatePressed && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || isTouch {
			fmt.Printf("Button Clicked: %s\n", b.ID)
			b.State = StateClicked
		} else {
			fmt.Println("Hover", b.ID)
			b.State = StateHover
		}
	} else {
		b.State = StateNormal
	}
}

func (b *Button) IsClicked() bool {
	return b.State == StateClicked
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

func (b *Button) getPosition(screen *ebiten.Image, scale float64) (int, int) {
	if b.Position != nil {
		w, h := layout.GetBounds(screen)
		return b.Position.Resolve(int(float64(w)/scale), int(float64(h)/scale))
	}
	return b.Bounds.Min.X, b.Bounds.Min.Y
}

func (b *Button) getPositionWithDims(screenW, screenH int, scale float64) image.Rectangle {
	if b.Position != nil {
		x, y := b.Position.Resolve(int(float64(screenW)/scale), int(float64(screenH)/scale))
		w := b.Bounds.Dx()
		h := b.Bounds.Dy()
		return image.Rectangle{Min: image.Point{x, y}, Max: image.Point{x + w, y + h}}
	}
	return b.Bounds
}

func (b *Button) MoveTo(X, Y int) {
	w := b.Normal.Bounds().Dx()
	h := b.Normal.Bounds().Dy()
	r := image.Rect(X, Y, X+w, Y+h)
	// fmt.Println(r)
	b.Bounds = r
}

// func (b *Button) Bounds() image.Rectangle {
// 	return b.Normal.Bounds()
// }
