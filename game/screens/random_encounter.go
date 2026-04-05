package screens

import (
	"fmt"
	"image/color"

	"github.com/benprew/s30/assets"
	gameaudio "github.com/benprew/s30/game/audio"
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
	world.TerrainPlains:    "art/screens/rand_encounter/0569.pic.png",
	world.TerrainMountains: "art/screens/rand_encounter/0737.pic.png",
	world.TerrainMarsh:     "art/screens/rand_encounter/0333.pic.png",
	world.TerrainForest:    "art/screens/rand_encounter/0335.pic.png",
	world.TerrainWater:     "art/screens/rand_encounter/0873.pic.png",
	world.TerrainSand:      "art/screens/rand_encounter/0873.pic.png",
}

var landToBgFile = map[string]string{
	"Plains":   "art/screens/rand_encounter/0569.pic.png",
	"Mountain": "art/screens/rand_encounter/0737.pic.png",
	"Swamp":    "art/screens/rand_encounter/0333.pic.png",
	"Forest":   "art/screens/rand_encounter/0335.pic.png",
	"Island":   "art/screens/rand_encounter/0873.pic.png",
}

type RandomEncounterScreen struct {
	Card       *domain.Card
	CardImg    *ebiten.Image
	Background *ebiten.Image
	Player     *domain.Player
	LandName   string
	DoneButton *elements.Button
}

// ==============================================================================
// 1.22 What are Lairs?
// ==============================================================================
//
//   Lairs are one of five different buildings that appear on the world map.
//   All you need to do is walk onto the lair and you get the prize in question.
//   'named' areas are lairs that put up a dialog box that say something
//   like 'You Found a Thieves Hideout!'.
//
//   Unnamed areas:
//     Gain one land of the appropriate color (island/plains...etc)
//     Gain a card of the appropriate color (usually a rare)
//     Gain an artifact
//
//   Named areas: Sometimes you can also gain a card or artifact from these.
//     Thieves Hideout: +500 Gold
//     Oasis of Mouldoon: Trade a card in your deck for +5 lives in the
//         next duel
//     Ruined Tower: Gain a Dungeon Clue.
//     Diamond Mine: trade random amulets for cards.
//     Gem Cutter Guild: Buy amulets for 200 gold each.
//     Guardian Ghost: Trade a random amulet for a look at a random color
//     Spectral Arena: Duel a creature for rare/astral cards.
//     Lost City of El'Arkan: Gain 1 amulet of each color
//     Nomad's Bazaar: Purchase cards at normal cost.
//
//   The Lost City and Thieves Hideout have a chance of taking gold/amulets
//   depending on the level of difficulty.

func NewRandomEncounterScreen(player *domain.Player, landName string, terrainType int) *RandomEncounterScreen {
	card := domain.FindCardByName(landName)
	if card != nil {
		player.CardCollection.AddCard(card, 1)
		if am := gameaudio.Get(); am != nil {
			am.PlaySFX(gameaudio.SFXFindCard)
		}
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

	textStr := fmt.Sprintf("You found a %s!", s.LandName)
	foundText := elements.NewText(48, textStr, 0, int(centerY+200*scale))
	foundText.HAlign = elements.AlignCenter
	foundText.BoundsW = float64(W)
	foundText.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	btnOpts := &ebiten.DrawImageOptions{}
	btnOpts.GeoM.Scale(scale, scale)
	s.DoneButton.Draw(screen, btnOpts, scale)
}

func (s *RandomEncounterScreen) IsFramed() bool {
	return false
}
