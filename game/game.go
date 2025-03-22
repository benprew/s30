package game

import (
	"fmt"
	"math"
	"time"

	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game is an isometric demo game.
type Game struct {
	w, h         int
	currentLevel *Level

	camX, camY float64
	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	offscreen  *ebiten.Image
	worldFrame *ebiten.Image

	// Player state
	playerSprite *sprites.PlayerSprite
	playerDir    int
	playerFrame  int
	lastUpdate   time.Time
	isMoving     bool
}

// NewGame returns a new isometric demo Game.
func NewGame() (*Game, error) {
	l, err := NewLevel()
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	playerSprite, err := sprites.LoadSpriteSheet(248, 174)
	if err != nil {
		return nil, fmt.Errorf("failed to load player sprite: %s", err)
	}

	worldFrame, err := LoadWorldFrame()
	if err != nil {
		return nil, fmt.Errorf("failed to load world frame: %s", err)
	}

	g := &Game{
		currentLevel: l,
		camScale:     1.25,
		camScaleTo:   1.25,
		mousePanX:    math.MinInt32,
		mousePanY:    math.MinInt32,
		playerSprite: playerSprite,
		worldFrame:   worldFrame,
		playerDir:    0,
		playerFrame:  0,
		lastUpdate:   time.Now(),
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
	if g.camScaleTo < 0.01 {
		g.camScaleTo = 0.01
	} else if g.camScaleTo > 100 {
		g.camScaleTo = 100
	}

	// Smooth zoom transition.
	div := 10.0
	if g.camScaleTo > g.camScale {
		g.camScale += (g.camScaleTo - g.camScale) / div
	} else if g.camScaleTo < g.camScale {
		g.camScale -= (g.camScale - g.camScaleTo) / div
	}

	// Pan camera and update player direction via keyboard
	pan := 7.0 / g.camScale
	left := ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA)
	right := ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD)
	down := ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS)
	up := ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW)

	if left {
		g.camX -= pan
	}
	if right {
		g.camX += pan
	}
	if down {
		g.camY -= pan
	}
	if up {
		g.camY += pan
	}

	// Update player direction and movement state based on input
	g.isMoving = up || down || left || right

	if g.isMoving {
		if up && right {
			g.playerDir = 5 // upRight
		} else if up && left {
			g.playerDir = 3 // upLeft
		} else if down && right {
			g.playerDir = 7 // downRight
		} else if down && left {
			g.playerDir = 1 // downLeft
		} else if up {
			g.playerDir = 4 // up
		} else if down {
			g.playerDir = 0 // down
		} else if left {
			g.playerDir = 2 // left
		} else if right {
			g.playerDir = 6 // right
		}
	}

	// Pan camera via mouse.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if g.mousePanX == math.MinInt32 && g.mousePanY == math.MinInt32 {
			g.mousePanX, g.mousePanY = ebiten.CursorPosition()
		} else {
			x, y := ebiten.CursorPosition()
			dx, dy := float64(g.mousePanX-x)*(pan/100), float64(g.mousePanY-y)*(pan/100)
			g.camX, g.camY = g.camX-dx, g.camY+dy
		}
	} else if g.mousePanX != math.MinInt32 || g.mousePanY != math.MinInt32 {
		g.mousePanX, g.mousePanY = math.MinInt32, math.MinInt32
	}

	// Clamp camera position.
	worldWidth := float64(g.currentLevel.w * g.currentLevel.tileWidth / 2)
	worldHeight := float64(g.currentLevel.h * g.currentLevel.tileHeight / 2)
	if g.camX < -worldWidth {
		g.camX = -worldWidth
	} else if g.camX > worldWidth {
		g.camX = worldWidth
	}
	if g.camY < -worldHeight {
		g.camY = -worldHeight
	} else if g.camY > 0 {
		g.camY = 0
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
	ebitenutil.DebugPrint(screen, fmt.Sprintf("KEYS WASD EC R\nFPS  %0.0f\nTPS  %0.0f\nSCA  %0.2f\nPOS  %0.0f,%0.0f", ebiten.ActualFPS(), ebiten.ActualTPS(), g.camScale, g.camX, g.camY))
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.w, g.h = outsideWidth, outsideHeight
	return g.w, g.h
}

// cartesianToIso transforms cartesian coordinates into isometric coordinates.
func (g *Game) cartesianToIso(x, y float64) (float64, float64) {
	ix := (x - y) * float64(g.currentLevel.tileWidth/2)
	iy := (x + y) * float64(g.currentLevel.tileHeight/4)
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

	for y := 0; y < g.currentLevel.h; y++ {
		for x := 0; x < g.currentLevel.w; x++ {
			xi, yi := g.cartesianToIso(float64(x), float64(y))

			// Skip drawing tiles that are out of the screen (with padding for smooth edges)
			drawX, drawY := ((xi-g.camX)*g.camScale)+cx, ((yi+g.camY)*g.camScale)+cy
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
			// Translate camera position.
			op.GeoM.Translate(-g.camX, g.camY)
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

	// Draw player
	if g.isMoving && time.Since(g.lastUpdate) > time.Millisecond*100 {
		g.playerFrame = (g.playerFrame + 1) % 5
		g.lastUpdate = time.Now()
	} else if !g.isMoving {
		g.playerFrame = 0 // Reset to standing frame when not moving
	}

	playerOp := &ebiten.DrawImageOptions{}
	playerOp.GeoM.Translate(-float64(124), -float64(87)) // Center the sprite
	playerOp.GeoM.Scale(g.camScale, g.camScale)          // Apply camera zoom
	playerOp.GeoM.Translate(float64(g.w)/2, float64(g.h)/2)
	screen.DrawImage(g.playerSprite.Animations[g.playerDir][g.playerFrame], playerOp)

	// Draw the world frame over everything
	frameOp := &ebiten.DrawImageOptions{}
	screen.DrawImage(g.worldFrame, frameOp)
}
