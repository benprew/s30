package game

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game is an isometric demo game.
type Game struct {
	screenW, screenH int
	currentLevel     *Level

	camScale   float64
	camScaleTo float64

	mousePanX, mousePanY int

	offscreen  *ebiten.Image
	worldFrame *ebiten.Image

	playerSprite *sprites.Character
	enemies      []*sprites.Character
	// widht and height of the level in pixels
	levelW, levelH int
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

	g := &Game{
		screenW:      1024,
		screenH:      768,
		currentLevel: l,
		camScale:     1.25,
		camScaleTo:   1.25,
		mousePanX:    math.MinInt32,
		mousePanY:    math.MinInt32,
		playerSprite: playerSprite,
		worldFrame:   worldFrame,
		enemies:      make([]*sprites.Character, 0),
	}
	g.levelW = l.tileWidth * l.w
	g.levelH = l.tileHeight * l.h / 2 // because tiles are packed in a zigzag pattern

	// Set initial player position at center of map
	playerSprite.X, playerSprite.Y = g.levelW/2, g.levelH/2

	// Initialize the player's position
	fmt.Printf("Starting player at position: %d, %d\n", playerSprite.X, playerSprite.Y)

	// Spawn initial enemies
	if err := g.spawnEnemies(3); err != nil {
		return nil, fmt.Errorf("failed to spawn enemies: %s", err)
	}

	return g, nil
}

// spawnEnemies creates a specified number of enemies at random positions
func (g *Game) spawnEnemies(count int) error {
	// Enemy character types to choose from
	enemyTypes := []sprites.CharacterName{
		sprites.WhiteArchmage, sprites.BlackKnight, sprites.BlueDjinn,
		sprites.DragonBRU, sprites.MultiTroll, sprites.Troll,
	}

	for i := 0; i < count; i++ {
		// Choose a random enemy type
		enemyType := enemyTypes[rand.Intn(len(enemyTypes))]

		// Load the enemy sprite
		enemy, err := sprites.LoadCharacter(enemyType)
		if err != nil {
			return fmt.Errorf("failed to load enemy sprite %s: %w", enemyType, err)
		}
		var x, y int

		// Set random position (avoiding player's immediate area)
		minDistance := 5000.0 // Minimum distance from player
		for {
			fmt.Println(g.levelH, g.levelW)
			// Random position within world bounds
			x = rand.Intn(g.levelW)
			y = rand.Intn(g.levelH)

			// Check distance from player
			dx := x - g.playerSprite.X
			dy := y - g.playerSprite.Y
			distance := math.Sqrt(float64(dx*dx + dy*dy))

			if distance < minDistance {
				break
			}
		}

		enemy.SetPosition(x, y)
		enemy.MoveSpeed = 1 + rand.Intn(3)

		g.enemies = append(g.enemies, enemy)
	}

	return nil
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
	g.camScaleTo = 1
	g.camScale = 1

	// // Clamp target zoom level.
	// if g.camScaleTo < 0.075 {
	// 	g.camScaleTo = 0.075
	// } else if g.camScaleTo > 2.25 {
	// 	g.camScaleTo = 2.25
	// }

	// // Smooth zoom transition.
	// div := 10.0
	// if g.camScaleTo > g.camScale {
	// 	g.camScale += (g.camScaleTo - g.camScale) / div
	// } else if g.camScaleTo < g.camScale {
	// 	g.camScale -= (g.camScale - g.camScaleTo) / div
	// }

	// Move player and update direction via keyboard using bit flags
	var dirBits int = 0
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		dirBits |= sprites.DirLeft
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		dirBits |= sprites.DirRight
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		dirBits |= sprites.DirDown
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		dirBits |= sprites.DirUp
	}
	g.playerSprite.Update(dirBits)

	// Update enemies with AI movement
	for _, e := range g.enemies {
		// Update enemy AI to move toward player when in range
		dirbits := e.UpdateAI(g.playerSprite.X, g.playerSprite.Y)
		e.Update(dirbits)
	}

	// Clamp player position to world boundaries
	if g.playerSprite.X < g.screenW/2 {
		g.playerSprite.X = g.screenW / 2
	}
	if g.playerSprite.X > g.levelW-g.screenW/2 {
		g.playerSprite.X = g.levelW - g.screenW/2
	}
	if g.playerSprite.Y < g.screenH/2 {
		g.playerSprite.Y = g.screenH / 2
	} else if g.playerSprite.Y > g.levelH-g.screenH/2 {
		g.playerSprite.Y = g.levelH - g.screenH/2
	}

	// Randomize level.
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		l, err := NewLevel()
		if err != nil {
			return fmt.Errorf("failed to create new level: %s", err)
		}

		g.currentLevel = l
		g.enemies = make([]*sprites.Character, 0)

		// Respawn enemies on new level
		if err := g.spawnEnemies(3); err != nil {
			return fmt.Errorf("failed to spawn enemies: %s", err)
		}
	}

	// Add more enemies with the 'N' key
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if err := g.spawnEnemies(5); err != nil {
			return fmt.Errorf("failed to spawn additional enemies: %s", err)
		}
	}

	return nil
}

// Draw draws the Game on the screen.
func (g *Game) Draw(screen *ebiten.Image) {
	// Render level.
	g.renderLevel(screen)

	// Print game info.
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"KEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nSCA  %0.2f\nPOS  %d,%d\nEPOS  %d,%d",
			ebiten.ActualFPS(), ebiten.ActualTPS(), g.camScale, g.playerSprite.X, g.playerSprite.Y, g.enemies[0].X, g.enemies[0].Y,
		),
	)
}

// Layout is called when the Game's layout changes.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenW, g.screenH = outsideWidth, outsideHeight
	return g.screenW, g.screenH
}

// cartesianToIso transforms cartesian coordinates into isometric coordinates.
func (g *Game) cartesianToIso(x, y int) (int, int) {
	// Adjust for isometric projection
	ix := (x - y) * (g.currentLevel.tileWidth / 2)
	iy := (x + y) * (g.currentLevel.tileHeight / 2)
	return ix, iy
}

// renderLevel draws the current Level on the screen.
func (g *Game) renderLevel(screen *ebiten.Image) {
	padding := 400
	scale := 1.0

	g.currentLevel.RenderZigzag(screen, g.playerSprite.X, g.playerSprite.Y, (g.screenW/2)+padding, (g.screenH/2)+padding)

	// if scaleLater {
	// 	op := &ebiten.DrawImageOptions{}
	// 	op.GeoM.Translate(-cx, -cy)
	// 	op.GeoM.Scale(float64(g.camScale), float64(g.camScale))
	// 	op.GeoM.Translate(cx, cy)
	// 	screen.DrawImage(target, op)
	// }

	// Draw player
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	options.GeoM.Translate(float64(g.screenW)/2, float64(g.screenH)/2)
	options.GeoM.Translate(-float64(sprites.CharSprW/2)*scale, -float64(sprites.CharSprH/2)*scale)
	g.playerSprite.Draw(screen, options)

	// Draw enemies
	for _, e := range g.enemies {
		if !g.isVisible(e.X, e.Y, e.Animations[0][0].Bounds().Dx(), e.Animations[0][0].Bounds().Dy()) {
			continue // Skip if not visible
		}

		screenX, screenY := g.screenOffset(e.X, e.Y)
		// Draw enemy
		enemyOp := &ebiten.DrawImageOptions{}
		enemyOp.GeoM.Scale(scale, scale)
		enemyOp.GeoM.Translate(-float64(sprites.CharSprW/2)*scale, -float64(sprites.CharSprH/2)*scale)
		enemyOp.GeoM.Translate(float64(screenX), float64(screenY))
		e.Draw(screen, enemyOp)
	}

	// Draw the worldFrame over everything
	frameOp := &ebiten.DrawImageOptions{}
	screen.DrawImage(g.worldFrame, frameOp)
}

// x,y is the position of the tile, width and height are the dimensions of the tile
// Check if the tile is visible on the screen
// return the position of the tile in the screen
func (g *Game) isVisible(x, y, width, height int) bool {
	// convert screenW and screenH based on the player position
	screenX := g.playerSprite.X - (g.screenW / 2)
	screenY := g.playerSprite.Y - (g.screenH / 2)

	// Check if the object is within the screen bounds
	if x+width < screenX || x > screenX+g.screenW || y+height < screenY || y > screenY+g.screenH {
		return false
	}

	return true
}

func (g *Game) screenOffset(x, y int) (int, int) {
	// Calculate screen position based on player position
	screenX := g.playerSprite.X - g.screenW/2
	screenY := g.playerSprite.Y - g.screenH/2

	// Calculate the position relative to the screen
	return x - screenX, y - screenY
}
