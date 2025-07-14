package screenui

import "github.com/hajimehoshi/ebiten/v2"

type ScreenName int

const (
	StartScr ScreenName = iota
	WorldScr
	MiniMapScr
	CityScr
	BuyCardsScr
)

type Screen interface {
	Update(W, H int) (ScreenName, error)
	Draw(screen *ebiten.Image, W, H int, scale float64)
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
	default:
		return "Unknown"
	}
}
