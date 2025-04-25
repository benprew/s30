package entities

import (
	"strings"
)

// CharacterName represents a specific character type and role combination
type CharacterName string

const (
	// White enemies
	WhiteArchmage   CharacterName = "W_Amg"
	WhiteFirewizard CharacterName = "W_Fwz"
	WhiteKnight     CharacterName = "W_Kht"
	WhiteLord       CharacterName = "W_Lrd"
	WhiteMagicwiz   CharacterName = "W_Mwz"
	WhiteWingedguy  CharacterName = "W_Wg"

	// Black enemies
	BlackArchmage   CharacterName = "Bk_Amg"
	BlackDjinn      CharacterName = "Bk_Djn"
	BlackFirewizard CharacterName = "Bk_Fwz"
	BlackKnight     CharacterName = "Bk_Kht"
	BlackLord       CharacterName = "Bk_Lrd"
	BlackMagicwiz   CharacterName = "Bk_Mwz"
	BlackWingedguy  CharacterName = "Bk_Wg"

	// Blue enemies
	BlueShifter    CharacterName = "Bu_Sft"
	BlueArchmage   CharacterName = "Bu_Amg"
	BlueDjinn      CharacterName = "Bu_Djn"
	BlueFirewizard CharacterName = "Bu_Fwz"
	BlueLord       CharacterName = "Bu_Lrd"
	BlueMagicwiz   CharacterName = "Bu_Mwz"
	BlueWyrm       CharacterName = "Bu_Wrm"

	// Dragon enemies
	DragonBRU CharacterName = "Dg_Bru"
	DragonGWR CharacterName = "Dg_Gwr"
	DragonRBG CharacterName = "Dg_Rbg"
	DragonUWB CharacterName = "Dg_Uwb"
	DragonWUG CharacterName = "Dg_Wug"

	Troll CharacterName = "Troll"

	// Multi enemies
	MultiApe        CharacterName = "M_Ape"
	MultiCentaur    CharacterName = "M_Cen"
	MultiCentaur2   CharacterName = "M_Cen2"
	MultiFang       CharacterName = "M_Fng"
	MultiFirewizard CharacterName = "M_Fwz"
	MultiKnight     CharacterName = "M_Kht"
	MultiLord       CharacterName = "M_Lrd"
	MultiTroll      CharacterName = "M_Trl"
	MultiTusk       CharacterName = "M_Tsk"
	MultiWingedguy  CharacterName = "M_Wg"

	// Player characters
	EgoFemale CharacterName = "Ego_F"
	EgoMale   CharacterName = "Ego_M"
)

var CharacterNames = []CharacterName{
	WhiteArchmage,
	WhiteFirewizard,
	WhiteKnight,
	WhiteLord,
	WhiteMagicwiz,
	WhiteWingedguy,

	// Black enemies
	BlackArchmage,
	BlackDjinn,
	BlackFirewizard,
	BlackKnight,
	BlackLord,
	BlackMagicwiz,
	BlackWingedguy,

	// Blue enemies
	BlueSafer,
	BlueArchmage,
	BlueDjinn,
	BlueFirewizard,
	BlueLord,
	BlueMagicwiz,
	BlueWyrm,

	// Dragon enemies
	DragonBRU,
	DragonGWR,
	DragonRBG,
	DragonUWB,
	DragonWUG,

	Troll,

	// Multi enemies
	MultiApe,
	MultiCentaur,
	MultiCentaur2,
	MultiFang,
	MultiFirewizard,
	MultiKnight,
	MultiLord,
	MultiTroll,
	MultiTusk,
	MultiWingedguy,

	// Player characters
	EgoFemale,
	EgoMale,
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
