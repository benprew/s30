package assets

import (
	"embed"
	_ "embed"
)

var (
	// Fonts
	//go:embed fonts/Planewalker-38m6.ttf
	Magic_ttf []byte

	//go:embed fonts/PlanewalkerDings-Pl9d.ttf
	MagicSym_ttf []byte

	// Card data
	//go:embed card_info/scryfall_cards.json.zst
	Cards_json []byte

	// Basic land tiles
	//go:embed art/sprites/world/land/Landtile.spr.png
	Landtile_png []byte

	// Grass and shrubs
	//go:embed art/sprites/world/land/Land.spr.png
	Land_png []byte
	// Grass and shrubs shadows
	//go:embed art/sprites/world/land/Sland.spr.png
	Sland_png []byte

	//go:embed art/sprites/Icons.spr.png
	Icons_png []byte
	//go:embed art/sprites/Iconb.spr.png
	Iconb_png []byte

	// Trees
	//go:embed art/sprites/world/land/Land2.spr.png
	Land2_png []byte
	// Trees shadows
	//go:embed art/sprites/world/land/Sland2.spr.png
	Sland2_png []byte

	//go:embed art/sprites/world/land/Cstline2.spr.png
	Cstline2_png []byte

	//go:embed art/sprites/world/land/Cstline1.spr.png
	Cstline_png []byte

	//go:embed art/sprites/world/land/Castles1.spr.png
	Castles1_png []byte

	// 6x2 city + shadow x2
	// Total 6x4
	//go:embed art/sprites/world/land/Locatn01.spr.png
	Cities1_png []byte

	// 6x2 city + shadow x2
	// Total 6x4
	//go:embed art/sprites/world/land/Locatn03.spr.png
	Dungeons_png []byte

	// World frame
	//go:embed art/sprites/world/Advinter1024.pic.png
	WorldFrame_png []byte

	//go:embed art/sprites/world/land/Roads.spr.png
	Roads_png []byte

	// Character sprites
	//go:embed art/sprites/world/characters/*.png
	CharacterFS embed.FS

	//////////////////////
	// Visit City Screens
	//////////////////////

	//go:embed art/sprites/city/City.png
	City_png []byte
	//go:embed art/sprites/city/Village.png
	Village_png []byte
	//go:embed art/sprites/city/Buycards.png
	BuyCards_png []byte
	//go:embed art/sprites/city/Smbuybttn.png
	BuyCardsSprite_png []byte
	//go:embed art/sprites/city/Smbuybttn_map.json
	BuyCardsSpriteMap_json []byte

	//////////
	// MiniMap
	//////////

	//go:embed art/sprites/mini_map/*.png
	MiniMapFS embed.FS
	//go:embed art/sprites/mini_map/Ttsprite.spr.png
	MiniMapTerrSpr_png []byte
	//go:embed art/sprites/mini_map/Mapback.pic.png
	MiniMapFrame_png []byte
	//go:embed art/sprites/mini_map/Mapbttns.pic.png
	MiniMapFrameSprite_png []byte
	//go:embed art/sprites/mini_map/Mapbttns_map.json
	MiniMapFrameSprite_json []byte
)
