package effects

type Keyword string

const (
	KeywordFlying       Keyword = "Flying"
	KeywordFirstStrike  Keyword = "First Strike"
	KeywordDoubleStrike Keyword = "Double Strike"
	KeywordTrample      Keyword = "Trample"
	KeywordHaste        Keyword = "Haste"
	KeywordVigilance    Keyword = "Vigilance"
	KeywordDeathtouch   Keyword = "Deathtouch"
	KeywordLifelink     Keyword = "Lifelink"
	KeywordReach        Keyword = "Reach"
	KeywordDefender     Keyword = "Defender"
	KeywordIndestructible Keyword = "Indestructible"
	KeywordHexproof     Keyword = "Hexproof"
	KeywordShroud       Keyword = "Shroud"
	KeywordFlash        Keyword = "Flash"
	KeywordBanding      Keyword = "Banding"
	KeywordRampage      Keyword = "Rampage"
	KeywordFear         Keyword = "Fear"
	KeywordIntimidation Keyword = "Intimidate"
	KeywordShadow       Keyword = "Shadow"
	KeywordHorsemanship Keyword = "Horsemanship"
	KeywordProtection   Keyword = "Protection"
	KeywordLandwalk     Keyword = "Landwalk"
	KeywordRegeneration Keyword = "Regeneration"
)

var KeywordMap = map[string]Keyword{
	"flying":        KeywordFlying,
	"first strike":  KeywordFirstStrike,
	"double strike": KeywordDoubleStrike,
	"trample":       KeywordTrample,
	"haste":         KeywordHaste,
	"vigilance":     KeywordVigilance,
	"deathtouch":    KeywordDeathtouch,
	"lifelink":      KeywordLifelink,
	"reach":         KeywordReach,
	"defender":      KeywordDefender,
	"indestructible": KeywordIndestructible,
	"hexproof":      KeywordHexproof,
	"shroud":        KeywordShroud,
	"flash":         KeywordFlash,
	"banding":       KeywordBanding,
	"bands with other": KeywordBanding,
	"rampage":       KeywordRampage,
	"fear":          KeywordFear,
	"intimidate":    KeywordIntimidation,
	"shadow":        KeywordShadow,
	"horsemanship":  KeywordHorsemanship,
}

type KeywordAbility struct {
	tgt      Targetable
	Keywords []Keyword
	Modifier string
}

func (k *KeywordAbility) Name() string {
	if len(k.Keywords) > 0 {
		return string(k.Keywords[0])
	}
	return "Keyword"
}

func (k *KeywordAbility) RequiresTarget() bool        { return false }
func (k *KeywordAbility) AddTarget(target Targetable) { k.tgt = target }
func (k *KeywordAbility) Target() Targetable          { return k.tgt }

func (k *KeywordAbility) Resolve() {}
