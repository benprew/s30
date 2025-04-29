package rules

type Phase string

const (
	PhaseUntap   Phase = "Untap"
	PhaseUpkeep  Phase = "Upkeep"
	PhaseDraw    Phase = "Draw"
	PhaseMain1   Phase = "Main1"
	PhaseCombat  Phase = "Combat"
	PhaseMain2   Phase = "Main2"
	PhaseEnd     Phase = "End"
	PhaseCleanup Phase = "Cleanup"
)

type Turn struct {
	Phase      Phase
	LandPlayed bool
}

func (t *Turn) NextPhase() {
	switch t.Phase {
	case PhaseUntap:
		t.Phase = PhaseUpkeep
	case PhaseUpkeep:
		t.Phase = PhaseDraw
	case PhaseDraw:
		t.Phase = PhaseMain1
	case PhaseMain1:
		t.Phase = PhaseCombat
	case PhaseCombat:
		t.Phase = PhaseMain2
	case PhaseMain2:
		t.Phase = PhaseEnd
	case PhaseEnd:
		t.Phase = PhaseCleanup
	case PhaseCleanup:
		t.Phase = PhaseUntap // New turn
	}
}
