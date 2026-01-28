package core_engine

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
	if card, ok := s.tgt.(*Card); ok {
		card.PowerBoost += s.PowerBoost
		card.ToughnessBoost += s.ToughnessBoost
	}
}
