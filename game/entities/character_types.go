package entities

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
	BlueSafer      CharacterName = "B_Sfr"
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

// shadowName returns the corresponding shadow sprite name for a character
func shadowName(name CharacterName) string {
	// Special cases first
	switch name {
	case Troll, MultiTroll:
		return "Strl"
	case EgoFemale:
		return "Sego_F"
	case EgoMale:
		return "Sego_M"
	case DragonBRU, DragonGWR, DragonRBG, DragonUWB, DragonWUG:
		return "S_Dg"
	case BlackDjinn, BlueDjinn:
		return "Sdjn"
	}

	// Get the prefix and base name
	str := string(name)
	if len(str) < 4 {
		return "S" + str
	}

	prefix := str[:2]
	base := str[3:]

	// Map prefixes to shadow prefixes
	shadowPrefix := "S"
	switch prefix {
	case "W_":
		shadowPrefix = "Sw_"
		base = str[2:]
	case "Bk":
		shadowPrefix = "Sb_"
	case "Bu":
		shadowPrefix = "Su_"
	case "G_":
		shadowPrefix = "Sg_"
	case "R_":
		shadowPrefix = "Sr_"
	case "M_":
		shadowPrefix = "Sm_"
	}

	return shadowPrefix + base
}
