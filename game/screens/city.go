package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	SCALE = 1.6
)

type CityScreen struct {
	Buttons []*elements.Button
	City    *domain.City
	Player  *domain.Player
}

type ButtonConfig struct {
	ID       string
	Text     string
	Index    int
	Position *layout.Position
}

func NewCityScreen(city *domain.City, player *domain.Player) *CityScreen {
	return &CityScreen{
		Buttons: mkButtons(SCALE-0.4, city),
		City:    city,
		Player:  player,
	}
}

func (c *CityScreen) IsFramed() bool {
	return true
}

func (c *CityScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	cityOpts := &ebiten.DrawImageOptions{}
	cityOpts.GeoM.Scale(scale, scale)
	cityOpts.GeoM.Scale(SCALE, SCALE)
	cityOpts.GeoM.Translate(FrameOffsetX*scale, FrameOffsetY*scale)
	screen.DrawImage(c.City.BackgroundImage, cityOpts)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	for _, b := range c.Buttons {
		b.Draw(screen, opts, scale)
	}

	c.drawCityName(screen)
}

func (c *CityScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	options := &ebiten.DrawImageOptions{}
	for i := range c.Buttons {
		b := c.Buttons[i]
		b.Update(options, scale, W, H)
		switch b.ID {
		case "leave":
			if b.IsClicked() {
				return screenui.WorldScr, nil
			}
		case "buycards":
			if b.IsClicked() {
				return screenui.BuyCardsScr, nil
			}
		case "buyfood":
			if b.IsClicked() {
				cost := c.City.FoodCost()
				if c.Player.Gold >= cost {
					c.Player.Gold -= cost
					c.Player.Food += 10
				}
				b.State = elements.StateNormal
			}
		case "editdeck":
			if b.IsClicked() {
				return screenui.EditDeckScr, nil
			}
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
	Icons, err := imageutil.LoadSpriteSheet(12, 2, assets.Icons_png)
	if err != nil {
		panic(fmt.Errorf("failed to load icons sprite sheet: %w", err))
	}
	Iconb, err := imageutil.LoadSpriteSheet(8, 1, assets.Iconb_png)
	if err != nil {
		panic(fmt.Errorf("failed to load iconb sprite sheet: %w", err))
	}

	// Create a font face using ebiten's text v2
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   20,
	}

	buttonConfigs := []ButtonConfig{
		{ID: "buycards", Text: "Buy Cards", Index: 3, Position: &layout.Position{Anchor: layout.WFTopLeft, OffsetX: 100, OffsetY: 50}},
		{ID: "quest", Text: "Begin Quest", Index: 2, Position: &layout.Position{Anchor: layout.WFCenter, OffsetX: -50, OffsetY: 0}},
		{ID: "buyfood", Text: fmt.Sprintf("%d gold = 10 food", city.FoodCost()), Index: 0, Position: &layout.Position{Anchor: layout.WFBottomLeft, OffsetX: 100, OffsetY: -125}},
		{ID: "leave", Text: "Leave Village", Index: 1, Position: &layout.Position{Anchor: layout.WFBottomRight, OffsetX: -250, OffsetY: -125}},
		{ID: "editdeck", Text: "Edit Deck", Index: 4, Position: &layout.Position{Anchor: layout.WFTopRight, OffsetX: -250, OffsetY: 50}},
	}

	buttons := make([]*elements.Button, len(buttonConfigs))
	for i, config := range buttonConfigs {
		btn := mkButton(config, fontFace, Icons, Iconb, scale)
		btn.ID = config.ID
		buttons[i] = &btn
	}

	return buttons
}

func mkButton(config ButtonConfig, fontFace *text.GoTextFace, Icons, Iconb [][]*ebiten.Image, scale float64) elements.Button {
	norm := elements.CombineButton(Iconb[0][0], Icons[0][config.Index], Iconb[0][1], scale)
	hover := elements.CombineButton(Iconb[0][2], Icons[0][config.Index], Iconb[0][3], scale)
	pressed := elements.CombineButton(Iconb[0][0], Icons[0][config.Index], Iconb[0][1], scale)

	btn := elements.NewButton(norm, hover, pressed, 0, 0, 1.0)
	btn.ButtonText = elements.ButtonText{
		Text:       config.Text,
		Font:       fontFace,
		TextColor:  color.White,
		TextOffset: image.Point{X: int(10 * scale), Y: int(60 * scale)},
		VAlign:     elements.AlignBottom,
	}
	btn.Position = config.Position

	return *btn
}
