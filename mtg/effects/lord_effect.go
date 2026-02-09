package effects

type LordEffect struct {
	tgt            Targetable
	Subtype        string
	PowerBoost     int
	ToughnessBoost int
	GrantedKeyword  *Keyword
	GrantedModifier string
	ExcludeSelf     bool
}

func (l *LordEffect) Name() string {
	if l.Subtype != "" {
		return l.Subtype + " Lord"
	}
	return "Lord Effect"
}

func (l *LordEffect) RequiresTarget() bool        { return false }
func (l *LordEffect) AddTarget(target Targetable) { l.tgt = target }
func (l *LordEffect) Target() Targetable          { return l.tgt }

func (l *LordEffect) Resolve() {}
