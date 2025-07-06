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

	//go:embed sprites/world/land/Cstline2.spr.png
	Cstline2_png []byte

	//go:embed sprites/world/land/Cstline1.spr.png
	Cstline_png []byte

	//go:embed sprites/world/land/Castles1.spr.png
	Castles1_png []byte

	// 6x2 city + shadow x2
	// Total 6x4
	//go:embed sprites/world/land/Locatn01.spr.png
	Cities1_png []byte

	// 6x2 city + shadow x2
	// Total 6x4
	//go:embed sprites/world/land/Locatn03.spr.png
	Dungeons_png []byte

	// world frame
	//go:embed sprites/world/Advinter1024.pic.png
	WorldFrame_png []byte

	//go:embed sprites/world/land/Roads.spr.png
	Roads_png []byte

	//go:embed sprites/world/City.png
	City_png []byte

	//go:embed sprites/world/Village.png
	Village_png []byte

	// Character sprites
	//go:embed sprites/world/characters/*.spr.png
	CharacterFS embed.FS

	//go:embed sprites/mini_map/*.png
	MiniMapFS embed.FS

	//go:embed sprites/mini_map/Ttsprite.spr.png
	MiniMapTerrSpr_png []byte

	//go:embed sprites/mini_map/Mapback.pic.png
	MiniMapFrame_png []byte

	//go:embed sprites/mini_map/Mapbttns.pic.png
	MiniMapFrameSprite_png []byte

	//go:embed sprites/mini_map/Mapbttns_map.json
	MiniMapFrameSprite_json []byte
)
