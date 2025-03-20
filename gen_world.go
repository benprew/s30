package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/aquilax/go-perlin"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	WorldSize    = 1000 // 1000x1000 tiles
	TileSize     = 64   // 64x64 pixels per tile
	ScreenWidth  = 1280
	ScreenHeight = 1024
	Seed         = 12345 // Fixed seed for consistency

	// Terrain thresholds
	Water  = 0.3
	Marsh  = 0.4
	Plains = 0.6
	Desert = 0.7
	Forest = 0.8
	Ice    = 0.9
)

type Game struct {
	world       [][]string
	spriteSheet *ebiten.Image
	spriteMap   map[string]*ebiten.Image
	camera      Camera
	playerX     int
	playerY     int
}

func loadSpriteSheet() (*ebiten.Image, map[string]*ebiten.Image) {
	// Load image using standard image package
	file, err := os.Open("art/Landtile.spr.png")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		log.Fatal("decoding image - ", err)
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	// Convert indexed color to RGBA and set transparency
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.(*image.Paletted).ColorIndexAt(x, y)
			if c == 255 { // Assuming index 255 is the transparent color
				rgba.Set(x, y, color.RGBA{0, 0, 0, 0}) // Transparent
			} else {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}

	processedImg := ebiten.NewImageFromImage(rgba)

	w := 206
	h := 102

	sprites := map[string]*ebiten.Image{
		"Water":  processedImg.SubImage(image.Rect(0, 0, w, h)).(*ebiten.Image),
		"Marsh":  processedImg.SubImage(image.Rect(w, 0, w*2, h)).(*ebiten.Image),
		"Plains": processedImg.SubImage(image.Rect(w*2, 0, w*3, h)).(*ebiten.Image),
		"Desert": processedImg.SubImage(image.Rect(w*3, 0, w*4, h)).(*ebiten.Image),
		"Forest": processedImg.SubImage(image.Rect(w*4, 0, w*5, h)).(*ebiten.Image),
		"Ice":    processedImg.SubImage(image.Rect(w*5, 0, w*6, h)).(*ebiten.Image),
	}

	return processedImg, sprites
}

func generateTerrain() [][]float64 {
	p := perlin.NewPerlin(2, 2, 3, Seed) // Perlin noise generator
	terrain := make([][]float64, WorldSize)

	for y := 0; y < WorldSize; y++ {
		terrain[y] = make([]float64, WorldSize)
		for x := 0; x < WorldSize; x++ {
			nx := float64(x) / float64(WorldSize) // Normalize coordinates
			ny := float64(y) / float64(WorldSize)
			noiseValue := p.Noise2D(nx*10, ny*10) // Adjust scale for variation
			terrain[y][x] = (noiseValue + 1) / 2  // Normalize to 0-1 range
		}
	}
	return terrain
}

func mapTerrainTypes(terrain [][]float64) [][]string {
	world := make([][]string, WorldSize)
	for y := 0; y < WorldSize; y++ {
		world[y] = make([]string, WorldSize)
		for x := 0; x < WorldSize; x++ {
			value := terrain[y][x]
			latitude := float64(y) / float64(WorldSize) // Determine latitude

			// Assign terrain type based on noise value and latitude
			switch {
			case latitude < 0.1 || latitude > 0.9:
				world[y][x] = "Ice"
			case value < Water:
				world[y][x] = "Water"
			case value < Marsh:
				world[y][x] = "Marsh"
			case value < Plains:
				world[y][x] = "Plains"
			case value < Desert:
				world[y][x] = "Desert"
			case value < Forest:
				world[y][x] = "Forest"
			default:
				world[y][x] = "Plains" // Default fallback
			}
		}
	}
	return world
}

func (g *Game) Update() error {
	// Example movement (WASD keys or arrow keys)
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.playerY -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.playerY += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.playerX -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.playerX += 1
	}

	// Transform player position to camera coordinates
	// cameraX, cameraY := g.camera.Transform(g.playerX, g.playerY)

	// // Clear the screen
	// screen.Fill(color.RGBA{0, 0, 0, 255})

	// // Draw a simple red square as the player
	// ebitenutil.DrawRect(screen, cameraX-10, cameraY-10, 20, 20, color.RGBA{255, 0, 0, 255})

	g.camera.Update(g, ScreenWidth, ScreenHeight)
	return nil // No game logic yet
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Calculate the center of the screen
	centerX := ScreenWidth / 2
	centerY := ScreenHeight / 2

	// Calculate the world-to-screen offset based on camera position
	offsetX := centerX - g.camera.X*(TileSize/2)
	offsetY := centerY - g.camera.Y*(TileSize/2)

	// Determine visible range of tiles
	visibleRadius := max(ScreenWidth, ScreenHeight) / TileSize * 2
	startX := max(0, g.camera.X-visibleRadius)
	endX := min(WorldSize-1, g.camera.X+visibleRadius)
	startY := max(0, g.camera.Y-visibleRadius)
	endY := min(WorldSize-1, g.camera.Y+visibleRadius)

	// Draw tiles from back to front for proper overlap
	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			// Skip rendering if out of world bounds
			if x < 0 || y < 0 || x >= WorldSize || y >= WorldSize {
				continue
			}

			terrain := g.world[y][x]
			img := g.spriteMap[terrain]
			if img != nil {
				// Convert grid position to isometric screen position
				screenX, screenY := gridToScreen(x, y)

				// Apply camera offset
				screenX += float64(offsetX)
				screenY += float64(offsetY)

				// Skip drawing tiles that are definitely off-screen
				if screenX < -TileSize*2 || screenY < -TileSize*2 ||
					screenX > float64(ScreenWidth+TileSize) || screenY > float64(ScreenHeight+TileSize) {
					continue
				}

				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(screenX, screenY)
				screen.DrawImage(img, op)
			}
		}
	}

	// Draw the player at the center of the screen
	playerOp := &ebiten.DrawImageOptions{}
	playerOp.GeoM.Translate(float64(centerX-10), float64(centerY-10))
	// Draw player sprite here
	// screen.DrawImage(g.playerSprite, playerOp)

	// Draw diagnostic information
	diagnostics := fmt.Sprintf("Camera: (%d, %d)\nPlayer: (%d, %d)", 
		g.camera.x, g.camera.y,
		g.playerX, g.playerY)
	ebitenutil.DebugPrint(screen, diagnostics)
}

// Helper function to convert grid coordinates to screen coordinates
func gridToScreen(x, y int) (float64, float64) {
	tileWidth := 206.0
	tileHeight := 102.0

	x_screen := float64((x - y) * int(tileWidth/2))
	y_screen := float64((x + y) * int(tileHeight/2))

	return x_screen, y_screen
}

// func (g *Game) Draw(screen *ebiten.Image) {
// 	// Calculate visible range based on camera position
// 	startX := max(0, g.camera.X-ScreenWidth/2/TileSize-1)
// 	endX := min(WorldSize, g.camera.X+ScreenWidth/2/TileSize+2)
// 	startY := max(0, g.camera.Y-ScreenHeight/2/TileSize-1)
// 	endY := min(WorldSize, g.camera.Y+ScreenHeight/2/TileSize+2)

// 	// Offset for centering the view on the camera
// 	offsetX := float64(ScreenWidth/2) - float64(g.camera.X*TileSize)
// 	offsetY := float64(ScreenHeight/2) - float64(g.camera.Y*TileSize)

// 	for y := int(startY); y < int(endY); y++ {
// 		for x := int(startX); x < int(endX); x++ {
// 			terrain := g.world[y][x]
// 			img := g.spriteMap[terrain]
// 			if img != nil {
// 				// Convert grid position to isometric screen position
// 				screenX, screenY := gridToScreen(x, y)

// 				// Apply camera offset
// 				screenX += offsetX
// 				screenY += offsetY

// 				op := &ebiten.DrawImageOptions{}
// 				op.GeoM.Translate(screenX, screenY)
// 				screen.DrawImage(img, op)
// 			}
// 		}
// 	}
// }

// func gridToScreen(x, y int) (float64, float64) {
// 	// For isometric tiles that are 206×102 pixels
// 	tileWidth := 206.0
// 	tileHeight := 102.0

// 	// Calculate screen coordinates
// 	// In isometric projection:
// 	// - Each step in x direction moves half a tile width to the right and half a tile height down
// 	// - Each step in y direction moves half a tile width to the left and half a tile height down
// 	x_screen := float64((x - y) * int(tileWidth/2))
// 	y_screen := float64((x + y) * int(tileHeight/2))

// 	return x_screen, y_screen
// }

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	// rand.Seed(time.Now().UnixNano())
	terrain := generateTerrain()
	world := mapTerrainTypes(terrain)

	spriteSheet, spriteMap := loadSpriteSheet()

	g := &Game{
		world:       world,
		spriteSheet: spriteSheet,
		spriteMap:   spriteMap,
		playerX:     500,
		playerY:     500,
		camera:      Camera{500, 500},
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
