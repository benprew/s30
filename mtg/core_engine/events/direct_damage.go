package events

import (
	"fmt"
)

type DirectDamage struct {
	tgt     Targetable
	Amount  int // Amount of damage dealt
	tgtType string
}

func (c *DirectDamage) Name() string                { return "Direct Damage" }
func (c *DirectDamage) RequiresTarget() bool        { return true }
func (c *DirectDamage) AddTarget(target Targetable) { c.tgt = target }
func (c *DirectDamage) Target() Targetable          { return c.tgt }
func (c *DirectDamage) Resolve() {
	fmt.Printf("Lightning Bolt deals 3 damage to %s\n", c.tgt.Name())
	c.tgt.ReceiveDamage(c.Amount)
}
