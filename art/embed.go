package art

import (
	_ "embed"
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

	// player characters
	//go:embed sprites/world/characters/Ego_M.spr.png
	Ego_M_png []byte
	//go:embed sprites/world/characters/Sego_M.spr.png
	Sego_M_png []byte

	//go:embed sprites/world/characters/Ego_F.spr.png
	Ego_F_png []byte
	//go:embed sprites/world/characters/Sego_F.spr.png
	Sego_F_png []byte
)
