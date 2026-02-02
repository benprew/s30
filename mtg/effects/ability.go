package effects

type AbilityType string

const (
	AbilityTypeActivated AbilityType = "Activated"
	AbilityTypeTriggered AbilityType = "Triggered"
	AbilityTypeStatic    AbilityType = "Static"
	AbilityTypeMana      AbilityType = "Mana"
)

type Ability struct {
	Type    AbilityType
	Cost    string
	Effect  string
	Trigger string
}

func (a *Ability) Resolve() {
}
