package game

import (
	"fmt"
	"math"
	"time"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/minimap"
	"github.com/benprew/s30/game/save"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	ScreenW, ScreenH     int
	camScale             float64
	camScaleTo           float64
	mousePanX, mousePanY int
	worldFrame           *screens.WorldFrame
	currentScreenName    screenui.ScreenName
	screenMap            map[screenui.ScreenName]screenui.Screen
	player               *domain.Player
}

func (g *Game) CurrentScreen() screenui.Screen {
	return g.screenMap[g.currentScreenName]
}

func (g *Game) Level() *world.Level {
	ls := g.screenMap[screenui.WorldScr].(*screens.LevelScreen)
	return ls.Level
}

func NewGame() (*Game, error) {
	startTime := time.Now()

	player, err := domain.NewPlayer("Player", nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load player sprite: %s", err)
	}

	l, err := world.NewLevel(player)
	if err != nil {
		return nil, fmt.Errorf("failed to create new level: %s", err)
	}

	m := minimap.NewMiniMap(l)

	wf, err := screens.NewWorldFrame(player)
	if err != nil {
		return nil, fmt.Errorf("failed to create world frame: %s", err)
	}

	scale := 1.0
	screenW := int(1024 * scale)
	screenH := int(768 * scale)

	g := &Game{
		ScreenW:           screenW,
		ScreenH:           screenH,
		camScale:          scale,
		camScaleTo:        1,
		mousePanX:         math.MinInt32,
		mousePanY:         math.MinInt32,
		worldFrame:        wf,
		currentScreenName: screenui.WorldScr,
		screenMap: map[screenui.ScreenName]screenui.Screen{
			screenui.WorldScr:    screens.NewLevelScreen(l),
			screenui.MiniMapScr:  m,
			screenui.DuelAnteScr: screens.NewDuelAnteScreen(),
		},
		player: player,
	}

	ebiten.SetWindowSize(g.ScreenW, g.ScreenH)

	go domain.PreloadCardImages(domain.CollectPriorityCards(player))

	fmt.Printf("NewGame execution time: %s\n", time.Since(startTime))
	return g, err
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if err := g.SaveGame("quicksave"); err != nil {
			fmt.Printf("Error saving game: %v\n", err)
		} else {
			fmt.Println("Game saved!")
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		l, err := world.NewLevel(g.player)
		if err != nil {
			return fmt.Errorf("failed to create new level: %s", err)
		}
		g.screenMap[screenui.WorldScr] = screens.NewLevelScreen(l)
	}

	name, screen, err := g.CurrentScreen().Update(g.ScreenW, g.ScreenH, g.camScale)
	if err != nil {
		return fmt.Errorf("err updating %s: %s", screenui.ScreenNameToString(name), err)
	}

	// If the screen returned a new screen instance, use it
	if screen != nil {
		g.screenMap[name] = screen
	}

	// If the world level signaled a duel ante, construct the duel screen with the encountered enemy
	if name == screenui.DuelAnteScr && g.currentScreenName != screenui.DuelAnteScr && screen == nil {
		lvl := g.Level()
		if lvl.EncounterPending() {
			if e, idx, ok := lvl.TakeEncounter(); ok {
				g.screenMap[name] = screens.NewDuelAnteScreenWithEnemy(lvl, idx)
				e.SetEngaged(true)
			}
		}
	}

	// Check for Random Encounter
	if name == screenui.WorldScr {
		lvl := g.Level()
		if lvl.RandomEncounterPending() {
			if _, terrainType, ok := lvl.TakeRandomEncounter(); ok {
				landName := world.TerrainToLandName(terrainType)
				g.screenMap[screenui.RandomEncounterScr] = screens.NewRandomEncounterScreen(g.player, landName, terrainType)
				name = screenui.RandomEncounterScr
			}
		}
	}

	g.currentScreenName = name

	if g.CurrentScreen().IsFramed() {
		var wfScreen screenui.Screen
		name, wfScreen, err = g.worldFrame.Update(g.ScreenW, g.ScreenH, g.camScale)
		if err != nil {
			return fmt.Errorf("err updating %s: %s", screenui.ScreenNameToString(name), err)
		}
		if wfScreen != nil {
			g.screenMap[name] = wfScreen
		}
		if name != -1 {
			g.currentScreenName = name
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.CurrentScreen().Draw(screen, g.ScreenW, g.ScreenH, g.camScale)
	if g.CurrentScreen().IsFramed() {
		g.worldFrame.Draw(screen, g.camScale)
	}

	// Print game info.
	charT := g.Level().CharacterTile()
	charP := g.Level().Player.Loc()
	closestCityTile, closestCityDistance := g.Level().FindClosestCity()

	// Get mouse cursor screen position
	mouseX, mouseY := ebiten.CursorPosition()

	debugText := fmt.Sprintf(
		"Screen: %s\nKEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nPOS  %d,%d\nTILE  %d,%d\nMOUSE %d,%d",
		screenui.ScreenNameToString(g.currentScreenName), ebiten.ActualFPS(), ebiten.ActualTPS(), charP.X, charP.Y, charT.X, charT.Y, mouseX, mouseY,
	)

	// Add closest city info if a city exists
	if closestCityTile.X != -1 && closestCityTile.Y != -1 {
		cityTile := g.Level().Tile(closestCityTile)
		debugText += fmt.Sprintf("\nCLOSEST CITY: %s at (%d,%d) dist: %.0f",
			cityTile.City.Name, closestCityTile.X, closestCityTile.Y, closestCityDistance)
	}

	ebitenutil.DebugPrint(screen, debugText)
}

// Layout returns the virtual screen resolution. Ebiten automatically scales
// the rendered output to fit the actual window size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1024, 768
}

func (g *Game) SaveGame(saveName string) error {
	level := g.Level()
	savePath, err := save.SaveGame(level, saveName)
	if err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}
	fmt.Printf("Game saved to: %s\n", savePath)
	return nil
}

func LoadSavedGame(savePath string) (*Game, error) {
	level, err := save.LoadGame(savePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load game: %w", err)
	}

	m := minimap.NewMiniMap(level)
	wf, err := screens.NewWorldFrame(level.Player)
	if err != nil {
		return nil, fmt.Errorf("failed to create world frame: %w", err)
	}

	scale := 1.0
	screenW := int(1024 * scale)
	screenH := int(768 * scale)

	g := &Game{
		ScreenW:           screenW,
		ScreenH:           screenH,
		camScale:          scale,
		camScaleTo:        1,
		mousePanX:         math.MinInt32,
		mousePanY:         math.MinInt32,
		worldFrame:        wf,
		currentScreenName: screenui.WorldScr,
		screenMap: map[screenui.ScreenName]screenui.Screen{
			screenui.WorldScr:    screens.NewLevelScreen(level),
			screenui.MiniMapScr:  m,
			screenui.DuelAnteScr: screens.NewDuelAnteScreen(),
		},
		player: level.Player,
	}

	ebiten.SetWindowSize(g.ScreenW, g.ScreenH)

	go domain.PreloadCardImages(domain.CollectPriorityCards(level.Player))

	return g, nil
}
