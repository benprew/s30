package core

type Phase int

const (
	PhaseUntap Phase = iota
	PhaseUpkeep
	PhaseDraw
	PhaseMain1
	PhaseCombat
	PhaseMain2
	PhaseEnd
	PhaseCleanup
	PhaseEndTurn // dummy phase that we should never reach
)

type CombatStep string

const (
	CombatStepNone              CombatStep = ""
	CombatStepBeginning         CombatStep = "BeginningOfCombat"
	CombatStepDeclareAttackers  CombatStep = "DeclareAttackers"
	CombatStepDeclareBlockers   CombatStep = "DeclareBlockers"
	CombatStepFirstStrikeDamage CombatStep = "FirstStrikeDamage"
	CombatStepCombatDamage      CombatStep = "CombatDamage"
	CombatStepEndOfCombat       CombatStep = "EndOfCombat"
)

type Turn struct {
	Phase      Phase
	CombatStep CombatStep
	LandPlayed bool
	Discarding bool
}

func (t *Turn) NextPhase() {
	t.Phase = t.Phase % PhaseEndTurn
	t.Phase++
}
