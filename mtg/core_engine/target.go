package core_engine

type TargetType int

const (
	TargetTypeCard TargetType = iota
	TargetTypePlayer
)

type Targetable interface {
	Name() string
	ReceiveDamage(amount int)
	TargetType() TargetType
	IsDead() bool
}
