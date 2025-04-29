package rules

type AbilityType string

const (
	AbilityTypeActivated AbilityType = "Activated"
	AbilityTypeTriggered AbilityType = "Triggered"
	AbilityTypeStatic    AbilityType = "Static"
)

type Ability struct {
	Type    AbilityType
	Cost    string // Mana cost or other activation cost
	Effect  string // Description of the ability's effect
	Trigger string // Condition that triggers the ability (for triggered abilities)
}

func (a *Ability) Resolve() {
	// TODO: Implement ability resolution logic
}
