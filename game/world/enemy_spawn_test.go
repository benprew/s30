package world

import (
	"image"
	"math/rand"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestProgressionEnemyMaxLevelStartsLow(t *testing.T) {
	player := &domain.Player{
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
		MinDeckSize: 30,
	}

	if got := progressionEnemyMaxLevel(player, 0, 0); got != 2 {
		t.Fatalf("progressionEnemyMaxLevel() = %d, want 2", got)
	}
}

func TestProgressionEnemyMaxLevelScalesWithCardsCombatsDaysAndCastles(t *testing.T) {
	player := &domain.Player{
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
		MinDeckSize: 30,
		Days:        40,
	}
	card := domain.FindCardByName("Forest")
	if card == nil {
		t.Fatal("test setup: Forest not found")
	}
	player.CardCollection.AddCard(card, 45)

	got := progressionEnemyMaxLevel(player, 7, 1)
	want := 8
	if got != want {
		t.Fatalf("progressionEnemyMaxLevel() = %d, want %d", got, want)
	}
}

func TestProgressionEnemyMaxLevelScalesWithPowerfulCards(t *testing.T) {
	player := &domain.Player{
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
		MinDeckSize: 30,
	}

	var powerfulCard *domain.Card
	for _, card := range domain.CardsByTier[domain.TierMandatory] {
		if card.CardType != domain.CardTypeLand {
			powerfulCard = card
			break
		}
	}
	if powerfulCard == nil {
		t.Fatal("test setup: no mandatory non-land card found")
	}
	player.CardCollection.AddCard(powerfulCard, 2)

	got := progressionEnemyMaxLevel(player, 0, 0)
	want := 3
	if got != want {
		t.Fatalf("progressionEnemyMaxLevel() = %d, want %d", got, want)
	}
}

func TestIsWithinEnemySpawnRadiusAllowsNearbyEnemies(t *testing.T) {
	if !isWithinEnemySpawnRadius(499.9) {
		t.Fatal("expected enemies to spawn within 500px of the player")
	}
	if !isWithinEnemySpawnRadius(500.0) {
		t.Fatal("expected enemies to spawn at the 500px boundary")
	}
	if isWithinEnemySpawnRadius(500.1) {
		t.Fatal("expected enemies outside 500px to be rejected")
	}
}

func TestEnemySpawnTilesFindsSparseValidTile(t *testing.T) {
	level := createTestLevel(3, 3)
	level.TileWidth = 200
	level.TileHeight = 100
	for y := 0; y < level.H; y++ {
		for x := 0; x < level.W; x++ {
			level.Tile(image.Point{X: x, Y: y}).TerrainType = TerrainWater
		}
	}

	validTile := image.Point{X: 1, Y: 1}
	level.Tile(validTile).TerrainType = TerrainPlains
	level.Player = &domain.Player{}
	level.Player.SetLoc(level.TileToPixel(validTile))

	tiles := level.enemySpawnTiles(level.Player.Loc())
	if len(tiles) != 1 {
		t.Fatalf("enemySpawnTiles() returned %d tiles, want 1: %v", len(tiles), tiles)
	}
	if tiles[0] != validTile {
		t.Fatalf("enemySpawnTiles()[0] = %v, want %v", tiles[0], validTile)
	}
}

func TestRandomEnemySpawnPositionWithinTileMapsBackToTile(t *testing.T) {
	level := createTestLevel(3, 3)
	level.TileWidth = 200
	level.TileHeight = 100
	rng := rand.New(rand.NewSource(1))

	for _, tile := range []image.Point{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 2}} {
		for range 100 {
			position := level.randomEnemySpawnPositionInTile(rng, tile)
			if got := level.PixelToTile(position); got != tile {
				t.Fatalf("PixelToTile(%v) = %v, want %v", position, got, tile)
			}
		}
	}
}

func TestEnemySpawnProfileNearCastlePrefersCastleColorAndStrongerEnemies(t *testing.T) {
	level := &Level{
		CombatsWon: 0,
		Player: &domain.Player{
			Character: domain.Character{
				CardCollection: domain.NewCardCollection(),
			},
			MinDeckSize: 30,
		},
		Castles: []*domain.Castle{
			{Color: domain.ColorRed, MapTile: image.Point{X: 10, Y: 10}},
		},
	}

	profile := level.enemySpawnProfileAt(image.Point{X: 12, Y: 12})
	if profile.maxLevel != 4 {
		t.Fatalf("profile.maxLevel = %d, want 4", profile.maxLevel)
	}
	if profile.preferredColor != "Red" {
		t.Fatalf("profile.preferredColor = %q, want Red", profile.preferredColor)
	}
}

func TestChooseEnemyNameFiltersByProgressionAndWeightsCastleColor(t *testing.T) {
	rogues := map[string]*domain.Character{
		"Blue Low":  {Name: "Blue Low", Level: 2, PrimaryColor: "Blue"},
		"Red Low":   {Name: "Red Low", Level: 2, PrimaryColor: "Red"},
		"Red High":  {Name: "Red High", Level: 4, PrimaryColor: "Red"},
		"Boss":      {Name: "Boss", Level: 11, PrimaryColor: "Red"},
		"Too Early": {Name: "Too Early", Level: 5, PrimaryColor: "Blue"},
	}

	names := weightedEnemyNames(rogues, enemySpawnProfile{maxLevel: 4, preferredColor: "Red"})
	counts := map[string]int{}
	for _, name := range names {
		counts[name]++
	}

	if counts["Too Early"] != 0 {
		t.Fatalf("included too-strong enemy %q", "Too Early")
	}
	if counts["Boss"] != 0 {
		t.Fatalf("included castle boss")
	}
	if counts["Red Low"] != castleColorWeight || counts["Red High"] != castleColorWeight {
		t.Fatalf("red weights = low:%d high:%d, want %d", counts["Red Low"], counts["Red High"], castleColorWeight)
	}
	if counts["Blue Low"] != normalEnemyWeight {
		t.Fatalf("blue weight = %d, want %d", counts["Blue Low"], normalEnemyWeight)
	}
}
