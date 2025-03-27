package game

import (
	"fmt"
	"math"

	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game is an isometric demo game.
type Game struct {
	w, h         int
	currentLevel *Level

	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	offscreen  *ebiten.Image
	worldFrame *ebiten.Image

	playerSprite *sprites.Character
	enemies      []*sprites.Character
}

// NewGame returns a new isometric demo Game.
func NewGame() (*Game, error) {
	l, err := NewLevel()
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	playerSprite, err := sprites.LoadCharacter(sprites.EgoFemale)
	if err != nil {
		return nil, fmt.Errorf("failed to load player sprite: %s", err)
	}

	worldFrame, err := LoadWorldFrame()
	if err != nil {
		return nil, fmt.Errorf("failed to load world frame: %s", err)
	}

	// Set initial player position at center of map
	playerSprite.X = float64(0)                       // Divide by 4 due to isometric projection
	playerSprite.Y = -float64(l.h * l.tileHeight / 2) // Negative because Y increases downward

	// Initialize the player's position
	fmt.Printf("Starting player at position: %.1f, %.1f\n", playerSprite.X, playerSprite.Y)

	g := &Game{
		currentLevel: l,
		camScale:     1.25,
		camScaleTo:   1.25,
		mousePanX:    math.MinInt32,
		mousePanY:    math.MinInt32,
		playerSprite: playerSprite,
		worldFrame:   worldFrame,
	}
	return g, nil
}

// Update reads current user input and updates the Game state.
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

	// Clamp target zoom level.
	if g.camScaleTo < 0.75 {
		g.camScaleTo = 0.75
	} else if g.camScaleTo > 1.25 {
		g.camScaleTo = 1.25
	}

	// Smooth zoom transition.
	div := 10.0
	if g.camScaleTo > g.camScale {
		g.camScale += (g.camScaleTo - g.camScale) / div
	} else if g.camScaleTo < g.camScale {
		g.camScale -= (g.camScale - g.camScaleTo) / div
	}

	// Move player and update direction via keyboard
	left := ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA)
	right := ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD)
	down := ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS)
	up := ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW)

	g.playerSprite.Update(up, down, left, right)

	for _, e := range g.enemies {
		// update enemy location and direction, moving randomly
		e.Update(up, down, left, right)
	}

	// Clamp player position to world boundaries
	worldWidth := float64(g.currentLevel.w * g.currentLevel.tileWidth / 2) // because tiles are 2x wide as tall
	worldHeight := float64(g.currentLevel.h * g.currentLevel.tileHeight)
	if g.playerSprite.X < -worldWidth {
		g.playerSprite.X = -worldWidth
	} else if g.playerSprite.X > worldWidth {
		g.playerSprite.X = worldWidth
	}
	if g.playerSprite.Y < -worldHeight {
		g.playerSprite.Y = -worldHeight
	} else if g.playerSprite.Y > 0 {
		g.playerSprite.Y = 0
	}

	// Randomize level.
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		l, err := NewLevel()
		if err != nil {
			return fmt.Errorf("failed to create new level: %s", err)
		}

		g.currentLevel = l
	}

	return nil
}

// Draw draws the Game on the screen.
func (g *Game) Draw(screen *ebiten.Image) {
	// Render level.
	g.renderLevel(screen)

	// Print game info.
	ebitenutil.DebugPrint(screen, fmt.Sprintf("KEYS WASD EC R\nFPS  %0.0f\nTPS  %0.0f\nSCA  %0.2f\nPOS  %0.2f,%0.2f", ebiten.ActualFPS(), ebiten.ActualTPS(), g.camScale, g.playerSprite.X, g.playerSprite.Y))
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.w, g.h = outsideWidth, outsideHeight
	return g.w, g.h
}

// cartesianToIso transforms cartesian coordinates into isometric coordinates.
func (g *Game) cartesianToIso(x, y float64) (float64, float64) {
	// Adjust for isometric projection
	ix := (x - y) * float64(g.currentLevel.tileWidth/2)
	iy := (x + y) * float64(g.currentLevel.tileHeight/2)
	return ix, iy
}

// isoToCartesian transforms isometric coordinates into cartesian coordinates.
func (g *Game) isoToCartesian(x, y float64) (float64, float64) {
	tileW := g.currentLevel.tileWidth
	tileH := g.currentLevel.tileHeight
	cx := (x/float64(tileW/2) + y/float64(tileH/4)) / 2
	cy := (y/float64(tileH/4) - (x / float64(tileW/2))) / 2
	return cx, cy
}

// renderLevel draws the current Level on the screen.
func (g *Game) renderLevel(screen *ebiten.Image) {
	// Calculate camera position based on player position
	playerIsoX, playerIsoY := g.cartesianToIso(g.playerSprite.X, g.playerSprite.Y)

	playerIsoX = g.playerSprite.X
	playerIsoY = g.playerSprite.Y

	op := &ebiten.DrawImageOptions{}
	xPadding := float64(g.currentLevel.tileHeight) * g.camScale
	yPadding := float64(g.currentLevel.tileHeight) * g.camScale
	cx, cy := float64(g.w/2), float64(g.h/2)

	scaleLater := g.camScale > 1
	target := screen
	scale := g.camScale

	// When zooming in, tiles can have slight bleeding edges.
	// To avoid them, render the result on an offscreen first and then scale it later.
	if scaleLater {
		if g.offscreen != nil {
			if g.offscreen.Bounds().Size() != screen.Bounds().Size() {
				g.offscreen.Deallocate()
				g.offscreen = nil
			}
		}
		if g.offscreen == nil {
			s := screen.Bounds().Size()
			g.offscreen = ebiten.NewImage(s.X, s.Y)
		}
		target = g.offscreen
		target.Clear()
		scale = 1
	}

	// Draw from back to front for proper overlapping
	for y := range g.currentLevel.h {
		for x := range g.currentLevel.w {
			xi, yi := g.cartesianToIso(float64(x), float64(y))

			// Skip drawing tiles that are out of the screen (with padding for smooth edges)
			drawX, drawY := ((xi-playerIsoX)*g.camScale)+cx, ((yi+playerIsoY)*g.camScale)+cy
			if drawX+xPadding < -xPadding || drawY+yPadding < -yPadding || drawX-xPadding > float64(g.w) || drawY-yPadding > float64(g.h) {
				continue
			}

			t := g.currentLevel.tiles[y][x]
			if t == nil {
				continue // No tile at this position.
			}

			op.GeoM.Reset()
			// Move to current isometric position.
			op.GeoM.Translate(xi, yi)
			// Translate camera position based on player.
			op.GeoM.Translate(-playerIsoX, playerIsoY)
			// Zoom.
			op.GeoM.Scale(scale, scale)
			// Center.
			op.GeoM.Translate(cx, cy)

			t.Draw(target, op)
		}
	}

	if scaleLater {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-cx, -cy)
		op.GeoM.Scale(float64(g.camScale), float64(g.camScale))
		op.GeoM.Translate(cx, cy)
		screen.DrawImage(target, op)
	}

	// Draw player and world frame
	g.playerSprite.Draw(screen, g.w, g.h, g.camScale)

	// for _, e := range g.enemies {
	// 	e.Draw(screen, g.w, g.h, g.camScale)
	// }

	// Draw the world frame over everything
	frameOp := &ebiten.DrawImageOptions{}
	screen.DrawImage(g.worldFrame, frameOp)
}
