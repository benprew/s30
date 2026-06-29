package game

import (
	"fmt"
	"math"
	"time"

	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/minimap"
	"github.com/benprew/s30/game/save"
	"github.com/benprew/s30/game/screens"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/benprew/s30/logging"
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
	currentScreen        screenui.ScreenName
	prevScreen           screenui.ScreenName
	screenMap            map[screenui.ScreenName]screenui.Screen
	player               *domain.Player
	Difficulty           domain.Difficulty
	audio                *gameaudio.AudioManager
}

func (g *Game) CurrentScreen() screenui.Screen {
	return g.screenMap[g.currentScreen]
}

// navigate switches the active screen. PopScr returns to the previous screen;
// NoScr and the current name are no-ops. Any other name is remembered as the
// screen to pop back to.
func (g *Game) navigate(name screenui.ScreenName) {
	switch name {
	case screenui.NoScr, g.currentScreen:
		// no-op
	case screenui.PopScr:
		g.currentScreen = g.prevScreen
	default:
		g.prevScreen = g.currentScreen
		g.currentScreen = name
	}
}

func (g *Game) Level() *world.Level {
	ls := g.screenMap[screenui.WorldScr].(*screens.LevelScreen)
	return ls.Level
}

func NewGame() (*Game, error) {
	loadedCardImages, err := domain.LoadEmbeddedCardImages()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded card images: %w", err)
	}
	if loadedCardImages > 0 {
		fmt.Printf("Loaded %d embedded card images\n", loadedCardImages)
	}

	scale := 1.0
	screenW := int(1024 * scale)
	screenH := int(768 * scale)

	am := gameaudio.NewAudioManager()

	g := &Game{
		ScreenW:           screenW,
		ScreenH:           screenH,
		camScale:          scale,
		camScaleTo:        1,
		mousePanX:         math.MinInt32,
		mousePanY:         math.MinInt32,
		currentScreen: screenui.StartScr,
		prevScreen:    screenui.StartScr,
		screenMap: map[screenui.ScreenName]screenui.Screen{
			screenui.StartScr: screens.NewStartScreen(),
		},
		Difficulty: domain.DifficultyEasy,
		audio:      am,
	}

	ebiten.SetWindowSize(g.ScreenW, g.ScreenH)
	ebiten.SetWindowClosingHandled(true)
	return g, nil
}

func (g *Game) initWorld(level *world.Level) error {
	m := minimap.NewMiniMap(level)

	wf, err := screens.NewWorldFrame(level.Player)
	if err != nil {
		return fmt.Errorf("failed to create world frame: %w", err)
	}

	g.worldFrame = wf
	g.player = level.Player
	g.screenMap[screenui.WorldScr] = screens.NewLevelScreen(level)
	g.screenMap[screenui.MiniMapScr] = m
	g.screenMap[screenui.QuestScrollScr] = screens.NewQuestScrollScreen(level.Player)
	g.screenMap[screenui.DuelAnteScr] = screens.NewDuelAnteScreen()

	go domain.PreloadCardImages(domain.CollectPriorityCards(level.Player))
	g.audio.PlayBGM(gameaudio.RandomWorldBGM())

	return nil
}

func (g *Game) handleStartTransition() error {
	startScr := g.screenMap[screenui.StartScr].(*screens.StartScreen)
	if startScr.SelectedSave != "" {
		level, err := save.LoadGame(startScr.SelectedSave)
		if err != nil {
			return fmt.Errorf("failed to load save: %w", err)
		}
		if err := level.RebuildSprites(); err != nil {
			return fmt.Errorf("failed to rebuild sprites: %w", err)
		}
		return g.initWorld(level)
	}

	startTime := time.Now()
	g.Difficulty = startScr.SelectedDifficulty
	player, err := domain.NewPlayer("Player", nil, false, g.Difficulty, startScr.SelectedColor)
	if err != nil {
		return fmt.Errorf("failed to load player sprite: %s", err)
	}
	level, err := world.NewLevel(player)
	if err != nil {
		return fmt.Errorf("failed to create new level: %s", err)
	}
	level.SetIdentity(world.NewGameID(), startScr.SelectedDifficulty, startScr.SelectedColor)
	if err := g.initWorld(level); err != nil {
		return err
	}
	fmt.Printf("New game creation time: %s\n", time.Since(startTime))
	return nil
}

func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() {
		if g.player != nil {
			if err := g.SaveGame(); err != nil {
				fmt.Printf("Error auto-saving: %v\n", err)
			}
		}
		return ebiten.Termination
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.audio.ToggleMute()
	}

	if g.player != nil {
		if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
			if err := g.SaveGame(); err != nil {
				fmt.Printf("Error saving game: %v\n", err)
			} else {
				fmt.Println("Game saved!")
			}
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			cur := g.Level()
			l, err := world.NewLevel(g.player)
			if err != nil {
				return fmt.Errorf("failed to create new level: %s", err)
			}
			l.SetIdentity(cur.GameID, cur.Difficulty, cur.PlayerColor)
			g.screenMap[screenui.WorldScr] = screens.NewLevelScreen(l)
		}
	}

	prevScreen := g.currentScreen
	name, screen, err := g.CurrentScreen().Update(g.ScreenW, g.ScreenH, g.camScale)
	if err != nil {
		return fmt.Errorf("err updating %s: %s", screenui.ScreenNameToString(prevScreen), err)
	}

	// Entering the world from the start screen builds the world first.
	if prevScreen == screenui.StartScr && name == screenui.WorldScr {
		if transitionErr := g.handleStartTransition(); transitionErr != nil {
			return transitionErr
		}
	}

	// If the screen returned a new instance, register it.
	if screen != nil && name != screenui.PopScr && name != screenui.NoScr {
		g.screenMap[name] = screen
	}

	// If the world level signaled a duel ante, construct the duel screen with the encountered enemy.
	if name == screenui.DuelAnteScr && prevScreen != screenui.DuelAnteScr && screen == nil {
		lvl := g.Level()
		if lvl.EncounterPending() {
			if e, idx, ok := lvl.TakeEncounter(); ok {
				g.screenMap[screenui.DuelAnteScr] = screens.NewDuelAnteScreenWithEnemy(lvl, idx)
				e.SetEngaged(true)
			}
		}
	}

	// Check for a random encounter while remaining on the world screen.
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

	g.navigate(name)
	screenChanged := g.currentScreen != prevScreen
	if screenChanged {
		g.updateBGM(g.currentScreen)
	}

	// Skip the world frame on the tick the active screen changed so a click that
	// dismissed an overlay (e.g. the quest scroll popping back to the world) is
	// not re-processed by the frame and used to reopen it.
	if !screenChanged && g.CurrentScreen().IsFramed() {
		wfName, wfScreen, wfErr := g.worldFrame.Update(g.ScreenW, g.ScreenH, g.camScale)
		if wfErr != nil {
			return fmt.Errorf("err updating world frame: %s", wfErr)
		}
		if wfScreen != nil && wfName != screenui.PopScr && wfName != screenui.NoScr {
			g.screenMap[wfName] = wfScreen
		}
		g.navigate(wfName)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	cur := g.CurrentScreen()
	if cur.IsOverlay() {
		if below := g.screenMap[g.prevScreen]; below != nil {
			below.Draw(screen, g.ScreenW, g.ScreenH, g.camScale)
		}
	}
	cur.Draw(screen, g.ScreenW, g.ScreenH, g.camScale)
	if cur.IsFramed() {
		g.worldFrame.Draw(screen, g.camScale)
	}

	if logging.Enabled(logging.World) && g.screenMap[screenui.WorldScr] != nil {
		charT := g.Level().CharacterTile()
		charP := g.Level().Player.Loc()
		closestCityTile, closestCityDistance := g.Level().FindClosestCity()

		mouseX, mouseY := ebiten.CursorPosition()

		debugText := fmt.Sprintf(
			"Screen: %s\nKEYS WASD N R\nFPS  %0.0f\nTPS  %0.0f\nPOS  %d,%d\nTILE  %d,%d\nMOUSE %d,%d",
			screenui.ScreenNameToString(g.currentScreen), ebiten.ActualFPS(), ebiten.ActualTPS(), charP.X, charP.Y, charT.X, charT.Y, mouseX, mouseY,
		)

		if closestCityTile.X != -1 && closestCityTile.Y != -1 {
			cityTile := g.Level().Tile(closestCityTile)
			debugText += fmt.Sprintf("\nCLOSEST CITY: %s at (%d,%d) dist: %.0f",
				cityTile.City.Name, closestCityTile.X, closestCityTile.Y, closestCityDistance)
		}

		ebitenutil.DebugPrint(screen, debugText)
	}
}

// Layout returns the virtual screen resolution. Ebiten automatically scales
// the rendered output to fit the actual window size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1024, 768
}

// Audio returns the game's AudioManager for use by screens.
func (g *Game) Audio() *gameaudio.AudioManager {
	return g.audio
}

func (g *Game) updateBGM(screen screenui.ScreenName) {
	switch screen {
	case screenui.WorldScr, screenui.MiniMapScr:
		if !gameaudio.IsWorldBGM(g.audio.CurrentBGM()) {
			g.audio.PlayBGM(gameaudio.RandomWorldBGM())
		}
	case screenui.DuelScr:
		g.audio.PlayBGM(gameaudio.BGMBattle)
	case screenui.CityScr, screenui.BuyCardsScr, screenui.EditDeckScr, screenui.WisemanScr:
		g.audio.PlayBGM(gameaudio.BGMCity)
	case screenui.DuelWinScr:
		g.audio.PlaySFX(gameaudio.SFXVictory)
		g.audio.StopBGM()
	case screenui.DuelLoseScr:
		g.audio.PlaySFX(gameaudio.SFXDefeat)
		g.audio.StopBGM()
	case screenui.DuelAnteScr:
		g.audio.StopBGM()
	}
}

func (g *Game) SaveGame() error {
	level := g.Level()
	savePath, err := save.SaveGame(level)
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

	if err := level.RebuildSprites(); err != nil {
		return nil, fmt.Errorf("failed to rebuild sprites: %w", err)
	}

	am := gameaudio.NewAudioManager()

	scale := 1.0
	screenW := int(1024 * scale)
	screenH := int(768 * scale)

	g := &Game{
		ScreenW:    screenW,
		ScreenH:    screenH,
		camScale:   scale,
		camScaleTo: 1,
		mousePanX:  math.MinInt32,
		mousePanY:  math.MinInt32,
		screenMap:  make(map[screenui.ScreenName]screenui.Screen),
		audio:      am,
	}

	if err := g.initWorld(level); err != nil {
		return nil, err
	}

	g.currentScreen = screenui.WorldScr
	g.prevScreen = screenui.WorldScr
	ebiten.SetWindowSize(g.ScreenW, g.ScreenH)

	return g, nil
}
