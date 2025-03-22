package art

import (
	_ "embed"
)

var (
	// basic land tiles

	//go:embed Landtile.spr.png
	Landtile_png []byte

	// grass and shrubs

	//go:embed Land.spr.png
	Land_png []byte

	// trees

	//go:embed Land2.spr.png
	Land2_png []byte

	// world frame
	//go:embed Advinter1024.pic.png
	WorldFrame_png []byte

	// player characters
	//go:embed Ego_M.spr.png
	Ego_M_png []byte

	//go:embed Ego_F.spr.png
	Ego_F_png []byte
)
