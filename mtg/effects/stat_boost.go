package effects

type StatBoost struct {
	tgt            Targetable
	PowerBoost     int
	ToughnessBoost int
}

func (s *StatBoost) Name() string                { return "Stat Boost" }
func (s *StatBoost) RequiresTarget() bool        { return true }
func (s *StatBoost) AddTarget(target Targetable) { s.tgt = target }
func (s *StatBoost) Target() Targetable          { return s.tgt }

func (s *StatBoost) Resolve() {
	s.tgt.AddPowerBoost(s.PowerBoost)
	s.tgt.AddToughnessBoost(s.ToughnessBoost)
}
