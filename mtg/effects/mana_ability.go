package effects

type ManaAbility struct {
	tgt       Targetable
	ManaTypes []string
	AnyColor  bool
}

func (m *ManaAbility) Name() string {
	if len(m.ManaTypes) == 1 {
		return "Add {" + m.ManaTypes[0] + "}"
	}
	if m.AnyColor {
		return "Add one mana of any color"
	}
	return "Mana Ability"
}

func (m *ManaAbility) RequiresTarget() bool        { return false }
func (m *ManaAbility) AddTarget(target Targetable) { m.tgt = target }
func (m *ManaAbility) Target() Targetable          { return m.tgt }

func (m *ManaAbility) Resolve() {}
