package events

import "fmt"

type DirectDamage struct {
	Target Targetable
	Amount int // Amount of damage dealt
}

func (c *DirectDamage) Name() string         { return "Direct Damage" }
func (c *DirectDamage) RequiresTarget() bool { return true }
func (c *DirectDamage) AddTarget(target Targetable) {
	c.Target = target
}

func (c *DirectDamage) Resolve() {
	fmt.Printf("Lightning Bolt deals 3 damage to %s\n", c.Target.Name())
	c.Target.ReceiveDamage(c.Amount)
}
