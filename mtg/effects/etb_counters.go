package effects

type ETBCounters struct {
	tgt            Targetable
	ETBCounters    int
	CounterPower   int
	CounterTough   int
}

func (e *ETBCounters) Name() string                { return "ETB Counters" }
func (e *ETBCounters) AddTarget(target Targetable) { e.tgt = target }
func (e *ETBCounters) Target() Targetable          { return e.tgt }
func (e *ETBCounters) Resolve()                    {}
