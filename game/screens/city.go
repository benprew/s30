package screens

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
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
	Level   *world.Level
}

type ButtonConfig struct {
	ID       string
	Text     string
	Index    int
	Position *layout.Position
}

func NewCityScreen(city *domain.City, player *domain.Player, level *world.Level) *CityScreen {
	if city.WisemanBoon == domain.BoonNone {
		city.WisemanBoon = pickBoon(city, player, level)
	}
	return &CityScreen{
		Buttons: mkButtons(SCALE-0.4, city),
		City:    city,
		Player:  player,
		Level:   level,
	}
}

func pickBoon(city *domain.City, player *domain.Player, level *world.Level) domain.BoonType {
	allowQuest := player.ActiveQuest == nil

	var options []domain.BoonType
	if allowQuest {
		for range 5 {
			options = append(options, domain.BoonQuest)
		}
	}
	for range 2 {
		options = append(options, domain.BoonBonusLife)
	}
	for range 2 {
		options = append(options, domain.BoonEnemyDeckInfo)
	}
	if hasWorldMagicCities(city, level) {
		for range 2 {
			options = append(options, domain.BoonWorldMagicLocation)
		}
	}
	if len(domain.CARDS) > 0 {
		for range 2 {
			options = append(options, domain.BoonBonusCard)
		}
	}

	if len(options) == 0 {
		return domain.BoonBonusLife
	}
	return options[rand.Intn(len(options))]
}

func hasWorldMagicCities(city *domain.City, level *world.Level) bool {
	if level == nil {
		return false
	}
	w, h := level.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			tile := level.Tile(image.Point{X: x, Y: y})
			if tile != nil && tile.IsCity && tile.City.Name != city.Name && tile.City.HasWorldMagic() {
				return true
			}
		}
	}
	return false
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

func (c *CityScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	options := &ebiten.DrawImageOptions{}
	for i := range c.Buttons {
		b := c.Buttons[i]
		b.Update(options, scale, W, H)
		switch b.ID {
		case "leave":
			if b.IsClicked() {
				return screenui.WorldScr, nil, nil
			}
		case "buycards":
			if b.IsClicked() {
				return screenui.BuyCardsScr, NewBuyCardsScreen(c.City, c.Player, W, H), nil
			}
		case "quest":
			if b.IsClicked() {
				return screenui.WisemanScr, NewWisemanScreen(c.City, c.Player, c.Level), nil
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
				s, err := NewEditDeckScreen(c.Player, c.City, W, H)
				return screenui.EditDeckScr, s, err
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.WorldScr, nil, nil
	}
	return screenui.CityScr, nil, nil
}

func (c *CityScreen) drawCityName(screen *ebiten.Image) {
	W := screen.Bounds().Dx()
	cityName := elements.NewText(30, c.City.Name, 0, 100)
	cityName.HAlign = elements.AlignCenter
	cityName.BoundsW = float64(W)
	cityName.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
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

	questText := "Talk to Wiseman"
	if city.WisemanBoon.IsQuest() && !city.BoonGranted {
		questText = "Begin Quest"
	}

	buttonConfigs := []ButtonConfig{
		{ID: "buycards", Text: "Buy Cards", Index: 3, Position: &layout.Position{Anchor: layout.WFTopLeft, OffsetX: 100, OffsetY: 50}},
		{ID: "quest", Text: questText, Index: 2, Position: &layout.Position{Anchor: layout.WFCenter, OffsetX: -50, OffsetY: 0}},
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
