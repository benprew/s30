package game

import (
	"fmt"
	"math"
	"time"

	"github.com/benprew/s30/game/minimap"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	screenW, screenH     int
	cityScreen           screens.CityScreen
	camScale             float64
	camScaleTo           float64
	mousePanX, mousePanY int
	currentScreenName    screenui.ScreenName
	screenMap            map[screenui.ScreenName]screenui.Screen
}

func (g *Game) CurrentScreen() screenui.Screen {
	return g.screenMap[g.currentScreenName]
}

// helper function to run the level
func (g *Game) Level() *world.Level {
	l := g.screenMap[screenui.WorldScr]
	return l.(*world.Level)
}

func NewGame() (*Game, error) {
	startTime := time.Now()
	fmt.Println("NewGame start")

	l, err := world.NewLevel()
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	m := minimap.NewMiniMap(l)

	g := &Game{
		screenW:           1024,
		screenH:           768,
		camScale:          1,
		camScaleTo:        1,
		mousePanX:         math.MinInt32,
		mousePanY:         math.MinInt32,
		currentScreenName: screenui.WorldScr,
		screenMap: map[screenui.ScreenName]screenui.Screen{
			screenui.WorldScr:   l,
			screenui.MiniMapScr: m,
		},
	}

	ebiten.SetWindowSize(g.screenW, g.screenH)

	fmt.Printf("NewGame execution time: %s\n", time.Since(startTime))
	return g, err
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		l, err := world.NewLevel()
		if err != nil {
			return fmt.Errorf("failed to create new level: %s", err)
		}
		g.screenMap[screenui.WorldScr] = l
	}

	name, err := g.CurrentScreen().Update(g.screenW, g.screenH)
	if err != nil {
		panic(fmt.Errorf("err updating %s: %s", screenui.ScreenNameToString(name), err))
	}
	if name == screenui.CityScr && g.currentScreenName != screenui.CityScr {
		tile := g.Level().Tile(g.Level().CharacterTile())
		g.screenMap[name] = screens.NewCityScreen(g.Level().Frame, &tile.City)
	}
	if name == screenui.BuyCardsScr && g.currentScreenName != screenui.BuyCardsScr {
		tile := g.Level().Tile(g.Level().CharacterTile())
		g.screenMap[name] = screens.NewBuyCardsScreen(g.Level().Frame, &tile.City)
	}
	g.currentScreenName = name

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.CurrentScreen().Draw(screen, g.screenW, g.screenH, g.camScale)

	// Print game info.
	charT := g.Level().CharacterTile()
	charP := g.Level().CharacterPos()
	// Get mouse cursor screen position
	mouseX, mouseY := ebiten.CursorPosition()

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"Screen: %s\nKEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nPOS  %d,%d\nTILE  %d,%d\nMOUSE %d,%d",
			screenui.ScreenNameToString(g.currentScreenName), ebiten.ActualFPS(), ebiten.ActualTPS(), charP.X, charP.Y, charT.X, charT.Y, mouseX, mouseY,
		),
	)
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenW, g.screenH = outsideWidth, outsideHeight
	return g.screenW, g.screenH
}
