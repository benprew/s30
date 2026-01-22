package core_engine

type DirectDamage struct {
	tgt    Targetable
	Amount int
}

func (c *DirectDamage) Name() string                { return "Direct Damage" }
func (c *DirectDamage) RequiresTarget() bool        { return true }
func (c *DirectDamage) AddTarget(target Targetable) { c.tgt = target }
func (c *DirectDamage) Target() Targetable          { return c.tgt }
func (c *DirectDamage) Cast() Event {
	return c
}
func (c *DirectDamage) Resolve() {
	c.tgt.ReceiveDamage(c.Amount)
}
