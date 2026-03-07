package effects

type TapEffect struct {
	tgt          Targetable
	TapTarget    bool
	DoesNotUntap bool
	UntapCost    string
}

func (t *TapEffect) Name() string                { return "Tap Effect" }
func (t *TapEffect) AddTarget(target Targetable) { t.tgt = target }
func (t *TapEffect) Target() Targetable          { return t.tgt }
func (t *TapEffect) Resolve() {
	if t.TapTarget && t.tgt != nil {
		t.tgt.SetTapped(true)
	}
}
