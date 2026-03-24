package screens

import (
	"fmt"
	"image"
	"math"
	"math/rand"

	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type LevelScreen struct {
	Level *world.Level
}

func NewLevelScreen(level *world.Level) *LevelScreen {
	return &LevelScreen{
		Level: level,
	}
}

func (s *LevelScreen) IsFramed() bool {
	return true
}

func (s *LevelScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	s.Level.Draw(screen, W, H, scale)
}

func (s *LevelScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	prevTile := s.Level.CharacterTile()

	if err := s.Level.UpdateWorld(W, H); err != nil {
		return screenui.WorldScr, nil, err
	}

	currentTile := s.Level.CharacterTile()
	tile := s.Level.Tile(currentTile)

	if am := gameaudio.Get(); am != nil && currentTile != (image.Point{X: -1, Y: -1}) {
		if tile != nil {
			terrainColor := gameaudio.TerrainTypeToColor(tile.TerrainType)

			if prevTile != currentTile {
				am.PlayFootstep(terrainColor)
			}

			if s.Level.TotalTicks()%50 == 0 && rand.Intn(3) == 0 {
				am.PlayBird(terrainColor)
			}
		}
	}

	if currentTile != (image.Point{X: -1, Y: -1}) {
		if tile != nil {
			if tile.IsCity && prevTile != currentTile {
				if am := gameaudio.Get(); am != nil {
					am.PlaySFX(gameaudio.CastleSFXForColor(domain.ColorMaskToString(tile.City.AmuletColor)))
				}
				return screenui.CityScr, NewCityScreen(&tile.City, s.Level.Player, s.Level), nil
			}
		}
	}

	pLoc := s.Level.Player.Loc()
	for i, e := range s.Level.Enemies() {
		if e.Engaged {
			continue
		}
		eLoc := e.Loc()

		pCenterX := float64(pLoc.X)
		pCenterY := float64(pLoc.Y)
		eCenterX := float64(eLoc.X)
		eCenterY := float64(eLoc.Y)

		dx := pCenterX - eCenterX
		dy := pCenterY - eCenterY
		dist := math.Hypot(dx, dy)

		if dist <= 20.0 {
			s.Level.SetEncounter(i)
			return screenui.DuelAnteScr, nil, nil
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if err := s.Level.SpawnEnemies(5); err != nil {
			return screenui.WorldScr, nil, fmt.Errorf("failed to spawn additional enemies: %s", err)
		}
	}

	return screenui.WorldScr, nil, nil
}
