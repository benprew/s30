package screenui

import "github.com/hajimehoshi/ebiten/v2"

type ScreenName int

const (
	StartScr ScreenName = iota
	WorldScr
	MiniMapScr
	CityScr
	BuyCardsScr
	DuelAnteScr
)

type Screen interface {
	Update(W, H int, scale float64) (ScreenName, error)
	Draw(screen *ebiten.Image, W, H int, scale float64)
	IsFramed() bool // True if we should draw the world frame
}

// screenNameToString converts a ScreenName to its string representation for debugging.
func ScreenNameToString(sn ScreenName) string {
	switch sn {
	case StartScr:
		return "Start"
	case WorldScr:
		return "World"
	case MiniMapScr:
		return "MiniMap"
	case CityScr:
		return "City"
	case DuelAnteScr:
		return "DuelAnte"
	default:
		return "Unknown"
	}
}
