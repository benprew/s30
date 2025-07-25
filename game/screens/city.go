package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets/art"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	SCALE = 1.6
)

type CityScreen struct {
	Frame   *ebiten.Image
	Buttons []*elements.Button
	City    *domain.City
}

type ButtonConfig struct {
	Text  string
	Index int
	X     int
	Y     int
}

func NewCityScreen(frame *ebiten.Image, city *domain.City) *CityScreen {
	return &CityScreen{
		Frame:   frame,
		Buttons: mkButtons(SCALE-0.4, city),
		City:    city,
	}
}

func (c *CityScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	cityOpts := &ebiten.DrawImageOptions{}
	cityOpts.GeoM.Scale(SCALE, SCALE)
	cityOpts.GeoM.Translate(100.0, 75.0) // Offset the image
	screen.DrawImage(c.City.BackgroundImage, cityOpts)

	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(c.Frame, frameOpts)

	for _, b := range c.Buttons {
		b.Draw(screen, frameOpts)
	}

	c.drawCityName(screen)
}

func (c *CityScreen) Update(W, H int) (screenui.ScreenName, error) {
	options := &ebiten.DrawImageOptions{}
	for i := range c.Buttons {
		b := c.Buttons[i]
		b.Update(options)
		if b.ButtonText.Text == "Leave Village" && b.State == elements.StateClicked {
			return screenui.WorldScr, nil
		}
		if b.ButtonText.Text == "Buy Cards" && b.State == elements.StateClicked {
			return screenui.BuyCardsScr, nil
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil
	}
	return screenui.CityScr, nil
}

func (c *CityScreen) drawCityName(screen *ebiten.Image) {
	// Use the same font face as the buttons for consistency
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   30, // Adjust size as needed
	}

	// Position the text at the top center of the screen
	// We need the screen width to center it. Assuming W is available.
	// For now, let's hardcode a position or use a fixed offset.
	// A better approach would be to pass screen dimensions to Draw.

	textX := 512 - (len(c.City.Name) * 6) // Centered around the middle of the screen (adjust as needed)
	textY := 100                          // Near the top

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(textX), float64(textY))
	R, G, B, A := color.White.RGBA()
	options.ColorScale.Scale(float32(R)/65535, float32(G)/65535, float32(B)/65535, float32(A)/65535)

	textOpts := text.DrawOptions{DrawImageOptions: *options}

	text.Draw(screen, c.City.Name, fontFace, &textOpts)
}

// Make buttons for City screen
// Iconb and Icons "b" stands for border
func mkButtons(scale float64, city *domain.City) []*elements.Button {
	Icons, err := sprites.LoadSpriteSheet(12, 2, art.Icons_png)
	if err != nil {
		panic(fmt.Errorf("failed to load icons sprite sheet: %w", err))
	}
	Iconb, err := sprites.LoadSpriteSheet(8, 1, art.Iconb_png)
	if err != nil {
		panic(fmt.Errorf("failed to load iconb sprite sheet: %w", err))
	}

	// Create a font face using ebiten's text v2
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   20,
	}

	buttonConfigs := []ButtonConfig{
		{"Buy Cards", 3, 200, 125},
		{"Begin Quest", 2, 450, 250},
		{fmt.Sprintf("Buy food %d gold", city.FoodCost()), 0, 200, 400},
		{"Leave Village", 1, 700, 400},
		{"Edit Deck", 4, 700, 125},
	}

	buttons := make([]*elements.Button, len(buttonConfigs))
	for i, config := range buttonConfigs {
		btn := mkButton(config, fontFace, Icons, Iconb, scale)
		buttons[i] = &btn
	}

	return buttons
}

func mkButton(config ButtonConfig, fontFace *text.GoTextFace, Icons, Iconb [][]*ebiten.Image, scale float64) elements.Button {
	norm := elements.CombineButton(Iconb[0][0], Icons[0][config.Index], Iconb[0][1], scale)
	hover := elements.CombineButton(Iconb[0][2], Icons[0][config.Index], Iconb[0][3], scale)
	pressed := elements.CombineButton(Iconb[0][0], Icons[0][config.Index], Iconb[0][1], scale)

	return elements.Button{
		Normal:  norm,
		Hover:   hover,
		Pressed: pressed,
		ButtonText: elements.ButtonText{
			Text:           config.Text,
			Font:           fontFace,
			TextColor:      color.White,
			TextOffset:     image.Point{X: int(10 * scale), Y: int(60 * scale)},
			VerticalCenter: elements.AlignBottom,
		},
		State: elements.StateNormal,
		X:     config.X,
		Y:     config.Y,
	}
}
