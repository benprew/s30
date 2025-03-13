package main

type Camera struct {
	X, Y int // Camera position in world coordinates (now using integers)
}

// Update the camera position to follow the player with integer-based movement
func (c *Camera) Update(g *Game, screenWidth, screenHeight int) {
	// Set the camera position directly to the player's position
	// For smoother movement, we can implement a "lerp-like" approach with integers

	// Calculate difference between player and camera
	deltaX := g.playerX - c.X
	deltaY := g.playerY - c.Y

	// Move camera a portion of the way to the player (integer division)
	// Adjusting the divisor controls the smoothness (higher = smoother)
	smoothnessDivisor := 5

	// Only move if the difference is large enough
	if abs(deltaX) > 0 {
		c.X += deltaX / smoothnessDivisor
		// Ensure camera catches up to player when very close
		if abs(deltaX) < smoothnessDivisor && c.X != g.playerX {
			if deltaX > 0 {
				c.X++
			} else {
				c.X--
			}
		}
	}

	if abs(deltaY) > 0 {
		c.Y += deltaY / smoothnessDivisor
		// Ensure camera catches up to player when very close
		if abs(deltaY) < smoothnessDivisor && c.Y != g.playerY {
			if deltaY > 0 {
				c.Y++
			} else {
				c.Y--
			}
		}
	}
}

// Helper function for absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// func (c *Camera) Update(g *Game, ScreenWidth, ScreenHeight int) {
// 	// The camera follows the player, keeping the player centered on the screen
// 	c.x = g.playerX - ScreenWidth/2
// 	c.y = g.playerY - ScreenHeight/2

// 	// Prevent the camera from going outside the world bounds
// 	if c.x < 0 {
// 		c.x = 0
// 	}
// 	if c.y < 0 {
// 		c.y = 0
// 	}
// 	// if c.x > g.worldWidth-g.ScreenWidth {
// 	// 	c.x = g.worldWidth - g.ScreenWidth
// 	// }
// 	// if c.y > g.worldHeight-g.ScreenHeight {
// 	// 	c.y = g.worldHeight - g.ScreenHeight
// 	// }
// }

// func (c *Camera) Transform(x, y int) (int, int) {
// 	// Transform world coordinates into camera coordinates
// 	return x - c.x, y - c.y
// }

// func update(screen *ebiten.Image) error {
// 	// Update camera based on player position
// 	camera := &Camera{}
// 	camera.Update(playerX, playerY)
//
// }
