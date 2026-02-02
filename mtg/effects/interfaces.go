package effects

type TargetType int

const (
	TargetTypeCard TargetType = iota
	TargetTypePlayer
)

type Targetable interface {
	Name() string
	EntityID() int
	ReceiveDamage(amount int)
	TargetType() TargetType
	IsDead() bool
	AddPowerBoost(int)
	AddToughnessBoost(int)
}

type Event interface {
	Name() string
	Resolve()
	Target() Targetable
	AddTarget(Targetable)
}
