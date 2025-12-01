package domain

import "fmt"

type Amulet struct {
	Color       ColorMask
	Name        string
	Description string
}

func NewAmulet(color ColorMask) Amulet {
	var name, description string

	switch color {
	case ColorWhite:
		name = "Amulet of Order"
		description = "A brilliant white amulet that emanates pure light"
	case ColorBlue:
		name = "Amulet of Knowledge"
		description = "A sapphire amulet that flows with arcane wisdom"
	case ColorBlack:
		name = "Amulet of Power"
		description = "A dark obsidian amulet that pulses with dark energy"
	case ColorRed:
		name = "Amulet of Passion"
		description = "A ruby amulet that burns with inner fire"
	case ColorGreen:
		name = "Amulet of Life"
		description = "An emerald amulet that hums with natural vitality"
	default:
		name = "Unknown Amulet"
		description = "A mysterious amulet of unknown origin"
	}

	return Amulet{
		Color:       color,
		Name:        name,
		Description: description,
	}
}

func (a Amulet) String() string {
	return fmt.Sprintf("%s (%s)", a.Name, a.Description)
}

func GetAllAmuletColors() []ColorMask {
	return []ColorMask{
		ColorWhite,
		ColorBlue,
		ColorBlack,
		ColorRed,
		ColorGreen,
	}
}

func ColorMaskToString(color ColorMask) string {
	switch color {
	case ColorWhite:
		return "White"
	case ColorBlue:
		return "Blue"
	case ColorBlack:
		return "Black"
	case ColorRed:
		return "Red"
	case ColorGreen:
		return "Green"
	default:
		return "Unknown"
	}
}
