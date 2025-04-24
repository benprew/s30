package game

import (
    "fmt"
    "math"

    "github.com/benprew/s30/game/world"
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game is an isometric demo game.
type Game struct {
    screenW, screenH int
    level            *world.Level

    camScale   float64
    camScaleTo float64

    mousePanX, mousePanY int

    worldFrame *ebiten.Image
}

// NewGame returns a new isometric demo Game.
func NewGame() (*Game, error) {
    l, err := world.NewLevel()
    if err != nil {
        return nil, fmt.Errorf("failed to create new level: %s", err)
    }

    worldFrame, err := LoadWorldFrame()
    if err != nil {
        return nil, fmt.Errorf("failed to load world frame: %s", err)
    }

    g := &Game{
        screenW:    1024,
        screenH:    768,
        level:      l,
        camScale:   1.25,
        camScaleTo: 1.25,
        mousePanX:  math.MinInt32,
        mousePanY:  math.MinInt32,
        worldFrame: worldFrame,
    }

    return g, nil
}

func (g *Game) Update() error {
    // Update target zoom level.
    var scrollY float64
    if ebiten.IsKeyPressed(ebiten.KeyC) || ebiten.IsKeyPressed(ebiten.KeyPageDown) {
        scrollY = -0.25
    } else if ebiten.IsKeyPressed(ebiten.KeyE) || ebiten.IsKeyPressed(ebiten.KeyPageUp) {
        scrollY = .25
    } else {
        _, scrollY = ebiten.Wheel()
        if scrollY < -1 {
            scrollY = -1
        } else if scrollY > 1 {
            scrollY = 1
        }
    }
    g.camScaleTo += scrollY * (g.camScaleTo / 7)
    g.camScaleTo = 1
    g.camScale = 1

    // // Clamp target zoom level.
    // if g.camScaleTo < 0.075 {
    //  g.camScaleTo = 0.075
    // } else if g.camScaleTo > 2.25 {
    //  g.camScaleTo = 2.25
    // }

    // // Smooth zoom transition.
    // div := 10.0
    // if g.camScaleTo > g.camScale {
    //  g.camScale += (g.camScaleTo - g.camScale) / div
    // } else if g.camScaleTo < g.camScale {
    //  g.camScale -= (g.camScale - g.camScaleTo) / div
    // }

    // Randomize level.
    if inpututil.IsKeyJustPressed(ebiten.KeyR) {
        l, err := world.NewLevel()
        if err != nil {
            return fmt.Errorf("failed to create new level: %s", err)
        }
        g.level = l
    }

    if err := g.level.Update(g.screenW, g.screenH); err != nil {
        return err
    }
    return nil
}

// Draw draws the Game on the screen.
func (g *Game) Draw(screen *ebiten.Image) {
    g.level.Draw(screen, g.screenW, g.screenH)

    // Draw the worldFrame over everything
    frameOp := &ebiten.DrawImageOptions{}
    screen.DrawImage(g.worldFrame, frameOp)

    // Print game info.
    charT := g.level.CharacterTile()
    charP := g.level.CharacterPos()
    ebitenutil.DebugPrint(
        screen,
        fmt.Sprintf(
            "KEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nPOS  %d,%d\nTILE  %d,%d",
            ebiten.ActualFPS(), ebiten.ActualTPS(), charP.X, charP.Y, charT.X, charT.Y,
        ),
    )
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
    g.screenW, g.screenH = outsideWidth, outsideHeight
    return g.screenW, g.screenH
}
