package screens

import (
	"fmt"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var encounterBgFiles = map[int]string{
	world.TerrainPlains:    "art/sprites/rand_encounter/0569.pic.png",
	world.TerrainMountains: "art/sprites/rand_encounter/0737.pic.png",
	world.TerrainMarsh:     "art/sprites/rand_encounter/0333.pic.png",
	world.TerrainForest:    "art/sprites/rand_encounter/0335.pic.png",
	world.TerrainWater:     "art/sprites/rand_encounter/0873.pic.png",
	world.TerrainSand:      "art/sprites/rand_encounter/0873.pic.png",
}

var landToBgFile = map[string]string{
	"Plains":   "art/sprites/rand_encounter/0569.pic.png",
	"Mountain": "art/sprites/rand_encounter/0737.pic.png",
	"Swamp":    "art/sprites/rand_encounter/0333.pic.png",
	"Forest":   "art/sprites/rand_encounter/0335.pic.png",
	"Island":   "art/sprites/rand_encounter/0873.pic.png",
}

type RandomEncounterScreen struct {
	Card       *domain.Card
	CardImg    *ebiten.Image
	Background *ebiten.Image
	Player     *domain.Player
	LandName   string
	DoneButton *elements.Button
}

func NewRandomEncounterScreen(player *domain.Player, landName string, terrainType int) *RandomEncounterScreen {
	card := domain.FindCardByName(landName)
	if card != nil {
		player.CardCollection.AddCard(card, 1)
	} else {
		fmt.Println("Warning: could not find land card:", landName)
	}

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}

	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}
	btn := elements.NewButton(btnSprites[0][0], btnSprites[0][1], btnSprites[0][2], 0, 0, 1.0)
	btn.ButtonText = elements.ButtonText{
		Text:      "Done",
		Font:      fontFace,
		TextColor: color.White,
		HAlign:    elements.AlignCenter,
		VAlign:    elements.AlignMiddle,
	}

	var cardImg *ebiten.Image
	if card != nil {
		cardImg, _ = card.CardImage(domain.CardViewFull)
	}

	bgFile := encounterBgFiles[terrainType]
	if terrainType == world.TerrainSnow {
		bgFile = landToBgFile[landName]
	}
	var bg *ebiten.Image
	if bgFile != "" {
		data, err := assets.RandEncounterFS.ReadFile(bgFile)
		if err == nil {
			bg, _ = imageutil.LoadImage(data)
		}
	}

	return &RandomEncounterScreen{
		Card:       card,
		CardImg:    cardImg,
		Background: bg,
		Player:     player,
		LandName:   landName,
		DoneButton: btn,
	}
}

func (s *RandomEncounterScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return screenui.WorldScr, nil, nil
	}

	s.DoneButton.MoveTo(W/2-50, H-100)
	opts := &ebiten.DrawImageOptions{}
	s.DoneButton.Update(opts, scale, W, H)
	if s.DoneButton.IsClicked() {
		return screenui.WorldScr, nil, nil
	}

	return screenui.RandomEncounterScr, nil, nil
}

func (s *RandomEncounterScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	if s.Background != nil {
		bgOpts := &ebiten.DrawImageOptions{}
		bgW := float64(s.Background.Bounds().Dx())
		bgH := float64(s.Background.Bounds().Dy())
		scaleX := float64(W) / bgW
		scaleY := float64(H) / bgH
		bgOpts.GeoM.Scale(scaleX, scaleY)
		screen.DrawImage(s.Background, bgOpts)
	}

	centerX := float64(W) / 2
	centerY := float64(H) / 2

	if s.CardImg != nil {
		op := &ebiten.DrawImageOptions{}
		cardScale := scale * 0.75
		op.GeoM.Scale(cardScale, cardScale)
		w := float64(s.CardImg.Bounds().Dx()) * cardScale
		h := float64(s.CardImg.Bounds().Dy()) * cardScale
		op.GeoM.Translate(centerX-w/2, centerY-h/2-30*scale)
		screen.DrawImage(s.CardImg, op)
	}

	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   48,
	}

	textStr := fmt.Sprintf("You found a %s!", s.LandName)
	width, _ := text.Measure(textStr, fontFace, 0)

	x := centerX - width/2
	y := centerY + 200*scale

	opts := &text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(color.White)

	text.Draw(screen, textStr, fontFace, opts)

	btnOpts := &ebiten.DrawImageOptions{}
	btnOpts.GeoM.Scale(scale, scale)
	s.DoneButton.Draw(screen, btnOpts, scale)
}

func (s *RandomEncounterScreen) IsFramed() bool {
	return false
}
