package game

import (
	"fmt"
	"math"
	"time"

	"github.com/benprew/s30/game/minimap"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game is an isometric demo game.
type Game struct {
	screenW, screenH int
	level            *world.Level
	miniMap          *minimap.MiniMap

	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	currentScreen int
}

const (
	StartScr = iota
	WorldScr
	MiniMapScr
)

// NewGame returns a new isometric demo Game.
func NewGame() (*Game, error) {
	startTime := time.Now()
	fmt.Println("NewGame start")

	l, err := world.NewLevel()
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
		currentScreen: MiniMapScr,
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
		return g.level.Update(g.screenW, g.screenH)
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

// Draw draws the Game on the screen.
func (g *Game) Draw(screen *ebiten.Image) {
	switch g.currentScreen {
	case WorldScr:
		g.level.Draw(screen, g.screenW, g.screenH, g.camScale)
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
			"KEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nPOS  %d,%d\nTILE  %d,%d\nMOUSE %d,%d",
			ebiten.ActualFPS(), ebiten.ActualTPS(), charP.X, charP.Y, charT.X, charT.Y, mouseX, mouseY,
		),
	)
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenW, g.screenH = outsideWidth, outsideHeight
	return g.screenW, g.screenH
}
