package events

type Targetable interface {
	Name() string
	ReceiveDamage(amount int)
}
