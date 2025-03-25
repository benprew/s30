package art

import (
	"embed"
)

var (
	// basic land tiles

	//go:embed sprites/world/land/Landtile.spr.png
	Landtile_png []byte

	// grass and shrubs
	//go:embed sprites/world/land/Land.spr.png
	Land_png []byte
	// grass and shrubs shadows
	//go:embed sprites/world/land/Sland.spr.png
	Sland_png []byte

	// trees
	//go:embed sprites/world/land/Land2.spr.png
	Land2_png []byte
	// trees
	//go:embed sprites/world/land/Sland2.spr.png
	Sland2_png []byte

	// world frame
	//go:embed Advinter1024.pic.png
	WorldFrame_png []byte

	// Character sprites
	//go:embed sprites/world/characters/*.spr.png
	CharacterFS embed.FS
)
