package screenui

import "github.com/hajimehoshi/ebiten/v2"

type ScreenName int

const (
	StartScr ScreenName = iota
	WorldScr
	MiniMapScr
	CityScr
	BuyCardsScr
	EditDeckScr
	DuelAnteScr
	DuelWinScr
	DuelLoseScr
	WisemanScr
	RandomEncounterScr
)

type Screen interface {
	Update(W, H int, scale float64) (ScreenName, Screen, error)
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
	case BuyCardsScr:
		return "BuyCards"
	case EditDeckScr:
		return "EditDeck"
	case DuelAnteScr:
		return "DuelAnte"
	case DuelWinScr:
		return "DuelWin"
	case DuelLoseScr:
		return "DuelLose"
	case WisemanScr:
		return "Wiseman"
	case RandomEncounterScr:
		return "RandomEncounter"
	default:
		return "Unknown"
	}
}
