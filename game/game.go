package game

import (
	"fmt"
	"math"
	"time"

	"github.com/benprew/s30/game/domain"
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
	camScale             float64
	camScaleTo           float64
	mousePanX, mousePanY int
	worldFrame           *screens.WorldFrame
	currentScreenName    screenui.ScreenName
	screenMap            map[screenui.ScreenName]screenui.Screen
	player               *domain.Player
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

	player, err := domain.NewPlayer("Player", nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load player sprite: %s", err)
	}

	l, err := world.NewLevel(player)
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	m := minimap.NewMiniMap(l)

	wf, err := screens.NewWorldFrame(player)
	if err != nil {
		return nil, fmt.Errorf("failed to create world frame: %s", err)
	}

	scale := 1.0

	g := &Game{
		screenW:           int(1024 * scale),
		screenH:           int(768 * scale),
		camScale:          scale,
		camScaleTo:        1,
		mousePanX:         math.MinInt32,
		mousePanY:         math.MinInt32,
		worldFrame:        wf,
		currentScreenName: screenui.WorldScr,
		screenMap: map[screenui.ScreenName]screenui.Screen{
			screenui.WorldScr:    l,
			screenui.MiniMapScr:  m,
			screenui.DuelAnteScr: screens.NewDuelAnteScreen(),
		},
		player: player,
	}

	ebiten.SetWindowSize(g.screenW, g.screenH)

	fmt.Printf("NewGame execution time: %s\n", time.Since(startTime))
	return g, err
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		l, err := world.NewLevel(g.player)
		if err != nil {
			return fmt.Errorf("failed to create new level: %s", err)
		}
		g.screenMap[screenui.WorldScr] = l
	}

	name, err := g.CurrentScreen().Update(g.screenW, g.screenH, g.camScale)
	if err != nil {
		panic(fmt.Errorf("err updating %s: %s", screenui.ScreenNameToString(name), err))
	}
	// If the world level signaled a duel ante, construct the duel screen with the encountered enemy
	if name == screenui.DuelAnteScr && g.currentScreenName != screenui.DuelAnteScr {
		// try to take encounter from level
		if lvl, ok := g.screenMap[screenui.WorldScr].(*world.Level); ok {
			if lvl.EncounterPending() {
				if e, idx, ok := lvl.TakeEncounter(); ok {
					g.screenMap[name] = screens.NewDuelAnteScreenWithEnemy(lvl, idx)
					// mark the enemy engaged so it is ignored until resolved
					e.SetEngaged(true)
				}
			}
		}
	}
	if name == screenui.CityScr && g.currentScreenName != screenui.CityScr {
		tile := g.Level().Tile(g.Level().CharacterTile())
		g.screenMap[name] = screens.NewCityScreen(&tile.City, g.player)
	}
	if name == screenui.BuyCardsScr && g.currentScreenName != screenui.BuyCardsScr {
		tile := g.Level().Tile(g.Level().CharacterTile())
		g.screenMap[name] = screens.NewBuyCardsScreen(&tile.City, g.player)
	}
	g.currentScreenName = name

	if g.CurrentScreen().IsFramed() {
		name, err = g.worldFrame.Update(g.screenW, g.screenH, g.camScale)
		if err != nil {
			panic(fmt.Errorf("err updating %s: %s", screenui.ScreenNameToString(name), err))
		}
		if name != -1 {
			g.currentScreenName = name
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.CurrentScreen().Draw(screen, g.screenW, g.screenH, g.camScale)
	if g.CurrentScreen().IsFramed() {
		g.worldFrame.Draw(screen, g.camScale)
	}

	// Print game info.
	charT := g.Level().CharacterTile()
	charP := g.Level().CharacterPos()
	closestCityTile, closestCityDistance := g.Level().FindClosestCity()

	// Get mouse cursor screen position
	mouseX, mouseY := ebiten.CursorPosition()

	debugText := fmt.Sprintf(
		"Screen: %s\nKEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nPOS  %d,%d\nTILE  %d,%d\nMOUSE %d,%d",
		screenui.ScreenNameToString(g.currentScreenName), ebiten.ActualFPS(), ebiten.ActualTPS(), charP.X, charP.Y, charT.X, charT.Y, mouseX, mouseY,
	)

	// Add closest city info if a city exists
	if closestCityTile.X != -1 && closestCityTile.Y != -1 {
		cityTile := g.Level().Tile(closestCityTile)
		debugText += fmt.Sprintf("\nCLOSEST CITY: %s at (%d,%d) dist: %.0f",
			cityTile.City.Name, closestCityTile.X, closestCityTile.Y, closestCityDistance)
	}

	ebitenutil.DebugPrint(screen, debugText)
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenW, g.screenH = outsideWidth, outsideHeight
	return g.screenW, g.screenH
}
