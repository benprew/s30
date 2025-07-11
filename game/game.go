package game

import (
	"fmt"
	"math"
	"time"

	"github.com/benprew/s30/game/minimap"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	screenW, screenH     int
	level                *world.Level
	miniMap              *minimap.MiniMap
	cityScreen           screens.CityScreen
	camScale             float64
	camScaleTo           float64
	mousePanX, mousePanY int
	currentScreen        int
}

const (
	StartScr = iota
	WorldScr
	MiniMapScr
	CityScr
)

// TODO: World should be a screen, level shouldn't be part of world
// Build a "Screen" interface that has an Draw and Update function
// Then game.draw can be game.CurrentScreen.Draw()
// same for game.Update()

// NewGame returns a new isometric demo Game.
func NewGame() (*Game, error) {
	startTime := time.Now()
	fmt.Println("NewGame start")

	l, err := world.NewLevel()
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	m := minimap.NewMiniMap()

	g := &Game{
		screenW:       1024,
		screenH:       768,
		level:         l,
		miniMap:       &m,
		camScale:      1,
		camScaleTo:    1,
		mousePanX:     math.MinInt32,
		mousePanY:     math.MinInt32,
		currentScreen: WorldScr, // Start on the world map
	}

	ebiten.SetWindowSize(g.screenW, g.screenH)

	fmt.Printf("NewGame execution time: %s\n", time.Since(startTime))
	return g, err
}

func (g *Game) Update() error {
	switch g.currentScreen {
	case WorldScr:
		// Randomize level.
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			l, err := world.NewLevel()
			if err != nil {
				return fmt.Errorf("failed to create new level: %s", err)
			}
			g.level = l
		}

		// Check if player entered a city or village
		err := g.level.Update(g.screenW, g.screenH)
		if err != nil {
			// Check for specific errors indicating screen change
			if err == world.ErrEnteredCity {
				g.currentScreen = CityScr
				fmt.Println("Entered city")
				tile := g.level.Tile(g.level.CharacterTile())
				g.cityScreen = screens.NewCityScreen(g.level.Frame, &tile.City)
				return nil // Consume the error, screen has changed
			}
			return fmt.Errorf("error updating world map: %w", err)
		}
		return nil // No screen change, continue in world map

	case CityScr:
		done, err := g.cityScreen.Update()
		if err != nil {
			return err
		}
		if done {
			g.currentScreen = WorldScr
		}
	case MiniMapScr:
		done, err := g.miniMap.Update()
		if err != nil {
			return err
		}
		if done {
			g.currentScreen = WorldScr
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.currentScreen {
	case WorldScr:
		g.level.Draw(screen, g.screenW, g.screenH, g.camScale)
	case CityScr:
		g.cityScreen.Draw(screen, g.screenW, g.screenH, g.camScale)
	case MiniMapScr:
		g.miniMap.Draw(screen, g.camScale, g.level)
	}

	// Print game info.
	charT := g.level.CharacterTile()
	charP := g.level.CharacterPos()
	// Get mouse cursor screen position
	mouseX, mouseY := ebiten.CursorPosition()

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"Screen: %d\nKEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nPOS  %d,%d\nTILE  %d,%d\nMOUSE %d,%d",
			g.currentScreen, ebiten.ActualFPS(), ebiten.ActualTPS(), charP.X, charP.Y, charT.X, charT.Y, mouseX, mouseY,
		),
	)
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenW, g.screenH = outsideWidth, outsideHeight
	return g.screenW, g.screenH
}
