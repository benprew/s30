package entities

import (
	"strings"
)

// CharacterName represents a specific character type and role combination
type CharacterName string

const (
	// White enemies
	WhiteArchmage  CharacterName = "W_Amg"
	WhiteFemaleWiz CharacterName = "W_Fwz"
	WhiteKnight    CharacterName = "W_Kht"
	WhiteLord      CharacterName = "W_Lrd"
	WhiteMaleWiz   CharacterName = "W_Mwz"
	WhiteAngel     CharacterName = "W_Wg"

	// Black enemies
	BlackArchmage  CharacterName = "Bk_Amg"
	BlackDjinn     CharacterName = "Bk_Djn"
	BlackFemaleWiz CharacterName = "Bk_Fwz"
	BlackKnight    CharacterName = "Bk_Kht"
	BlackLord      CharacterName = "Bk_Lrd"
	BlackMaleWiz   CharacterName = "Bk_Mwz"
	BlackWingedguy CharacterName = "Bk_Wg"

	// Blue enemies
	BlueArchmage  CharacterName = "Bu_Amg"
	BlueDjinn     CharacterName = "Bu_Djn"
	BlueFemaleWiz CharacterName = "Bu_Fwz"
	BlueLord      CharacterName = "Bu_Lrd"
	BlueMaleWiz   CharacterName = "Bu_Mwz"
	BlueWurm      CharacterName = "Bu_Wrm"
	BlueShifter   CharacterName = "Bu_Sft"

	// Red enemies
	RedArchmage  CharacterName = "R_Amg"
	RedDjinn     CharacterName = "R_Djn"
	RedFemaleWiz CharacterName = "R_Fwz"
	RedLord      CharacterName = "R_Lrd"
	RedMaleWiz   CharacterName = "R_Mwz"
	RedWurm      CharacterName = "R_Wrm"
	Troll        CharacterName = "Troll"

	// Green enemies
	GreenArchmage  CharacterName = "G_Amg"
	GreenDjinn     CharacterName = "G_Djn"
	GreenFemaleWiz CharacterName = "G_Fwz"
	GreenKnight    CharacterName = "G_Kht"
	GreenLord      CharacterName = "G_Lrd"
	GreenMaleWiz   CharacterName = "G_Mwz"
	GreenWurm      CharacterName = "G_Wrm"

	// Dragon enemies
	DragonBRU CharacterName = "Dg_Bru"
	DragonGWR CharacterName = "Dg_Gwr"
	DragonRBG CharacterName = "Dg_Rbg"
	DragonUWB CharacterName = "Dg_Uwb"
	DragonWUG CharacterName = "Dg_Wug"

	// Multi enemies
	MultiApe        CharacterName = "M_Ape"
	MultiCentaur    CharacterName = "M_Cen"
	MultiCentaur2   CharacterName = "M_Cen2"
	MultiFungus     CharacterName = "M_Fng"
	MultiFemaleWiz  CharacterName = "M_Fwz"
	MultiKnight     CharacterName = "M_Kht"
	MultiLord       CharacterName = "M_Lrd"
	MultiSedgeBeast CharacterName = "M_Trl"
	MultiTusk       CharacterName = "M_Tsk"
	MultiWingedguy  CharacterName = "M_Wg"

	// Player characters
	EgoFemale CharacterName = "Ego_F"
	EgoMale   CharacterName = "Ego_M"
)

var Enemies = []CharacterName{
	// White enemies
	WhiteArchmage,
	WhiteFemaleWiz,
	WhiteKnight,
	WhiteLord,
	WhiteMaleWiz,
	WhiteAngel,

	// Black enemies
	BlackArchmage,
	BlackDjinn,
	BlackFemaleWiz,
	BlackKnight,
	BlackLord,
	BlackMaleWiz,
	BlackWingedguy,

	// Blue enemies
	BlueArchmage,
	BlueDjinn,
	BlueFemaleWiz,
	BlueLord,
	BlueMaleWiz,
	BlueWurm,
	BlueShifter,

	// Red enemies
	RedArchmage,
	RedDjinn,
	RedFemaleWiz,
	RedLord,
	RedMaleWiz,
	RedWurm,
	Troll,

	// Green enemies
	GreenArchmage,
	GreenDjinn,
	GreenFemaleWiz,
	GreenKnight,
	GreenLord,
	GreenMaleWiz,
	GreenWurm,

	// Dragon enemies
	DragonBRU,
	DragonGWR,
	DragonRBG,
	DragonUWB,
	DragonWUG,

	// Multi enemies
	MultiApe,
	MultiCentaur,
	MultiCentaur2,
	MultiFungus,
	MultiFemaleWiz,
	MultiKnight,
	MultiLord,
	MultiSedgeBeast,
	MultiTusk,
	MultiWingedguy,
}

// shadowName returns the corresponding shadow sprite name for a character
func shadowName(name CharacterName) string {
	// Special cases first
	xref := map[string]string{
		"Kht":   "Skht",
		"Djn":   "Sdjn",
		"Fwz":   "Sfwz",
		"Mwz":   "Smwz",
		"Trl":   "Strl",
		"Troll": "Strl",
		"Wrm":   "Swrm",
		"Dg_":   "S_Dg",
		"Ego_F": "Sego_F",
		"Ego_M": "Sego_M",
	}
	for str, shadow := range xref {
		if strings.Index(string(name), str) != -1 {
			return shadow
		}
	}

	// fmt.Println(name)

	// Get the prefix and base name
	str := string(name)
	if len(str) < 4 {
		return "S" + str
	}

	parts := strings.Split(string(name), "_")

	prefix := parts[0]
	base := parts[1]

	// fmt.Println(prefix, base)

	// Map prefixes to shadow prefixes
	shadowPrefix := "S"
	switch prefix {
	case "W":
		shadowPrefix = "Sw_"
		base = str[2:]
	case "Bk":
		shadowPrefix = "Sb_"
	case "Bu":
		shadowPrefix = "Su_"
	case "G":
		shadowPrefix = "Sg_"
	case "R":
		shadowPrefix = "Sr_"
	case "M":
		shadowPrefix = "Sm_"
	}

	return shadowPrefix + base
}
