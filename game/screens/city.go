package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets/art"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/world"
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
	City    *world.City
}

type ButtonConfig struct {
	Text  string
	Index int
	X     int
	Y     int
}

func NewCityScreen(frame *ebiten.Image, city *world.City) CityScreen {
	return CityScreen{
		Frame:   frame,
		Buttons: mk_buttons(SCALE-0.4, city),
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

	// fonts.DrawText
	// c.drawCityName(screen)
}

func (c *CityScreen) Update() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		fmt.Println("Returned to world map from city")
		return true, nil
	}
	options := &ebiten.DrawImageOptions{}
	for _, b := range c.Buttons {
		b.Update(options)
	}

	return false, nil
}

// Make buttons for City screen
// Iconb and Icons "b" stands for border
func mk_buttons(scale float64, city *world.City) []*elements.Button {
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
		buttons[i] = mk_button(config, fontFace, Icons, Iconb, scale)
	}

	return buttons
}

func mk_button(config ButtonConfig, fontFace *text.GoTextFace, Icons, Iconb [][]*ebiten.Image, scale float64) *elements.Button {
	norm := elements.CombineButton(Iconb[0][0], Icons[0][config.Index], Iconb[0][1], scale)
	hover := elements.CombineButton(Iconb[0][2], Icons[0][config.Index], Iconb[0][3], scale)
	pressed := elements.CombineButton(Iconb[0][0], Icons[0][config.Index], Iconb[0][1], scale)

	return &elements.Button{
		Normal:     norm,
		Hover:      hover,
		Pressed:    pressed,
		Text:       config.Text,
		Font:       fontFace,
		TextColor:  color.White,
		TextOffset: image.Point{X: int(10 * scale), Y: int(60 * scale)},
		State:      elements.StateNormal,
		X:          config.X,
		Y:          config.Y,
	}
}
