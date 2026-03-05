package effects

type DestroyPermanent struct {
	tgt     Targetable
	Destroy bool
}

func (d *DestroyPermanent) Name() string                { return "Destroy Permanent" }
func (d *DestroyPermanent) AddTarget(target Targetable) { d.tgt = target }
func (d *DestroyPermanent) Target() Targetable          { return d.tgt }
func (d *DestroyPermanent) Resolve() {
	d.tgt.MarkDestroyed()
}
