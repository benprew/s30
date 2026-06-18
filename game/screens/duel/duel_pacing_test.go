package duel

import (
	"testing"

	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func pacingMsg(active string, youLife, oppLife int) *interactive.GameMsg {
	return &interactive.GameMsg{
		State: &interactive.GameState{
			ActivePlayer: active,
			You:          interactive.PlayerState{Name: "You", Life: youLife},
			Opponent:     interactive.PlayerState{Name: "Sorceress", Life: oppLife},
		},
	}
}

func TestPhaseDelay_PlayerTurnNoChange_UsesBase(t *testing.T) {
	got := phaseDelay(pacingMsg("You", 10, 10), pacingMsg("You", 10, 10))
	if got != phaseDisplayDelay {
		t.Errorf("want %v, got %v", phaseDisplayDelay, got)
	}
}

func TestPhaseDelay_EnemyTurn_PausesLonger(t *testing.T) {
	got := phaseDelay(pacingMsg("Sorceress", 10, 10), pacingMsg("Sorceress", 10, 10))
	if got != enemyPhaseDelay {
		t.Errorf("want %v, got %v", enemyPhaseDelay, got)
	}
}

func TestPhaseDelay_LifeChange_PausesLongest(t *testing.T) {
	// You take 6 from Ball Lightning on the enemy's turn.
	got := phaseDelay(pacingMsg("Sorceress", 10, 10), pacingMsg("Sorceress", 4, 10))
	if got != lifeChangeDelay {
		t.Errorf("want %v, got %v", lifeChangeDelay, got)
	}
}

func TestPhaseDelay_OpponentLifeChangeOnPlayerTurn(t *testing.T) {
	got := phaseDelay(pacingMsg("You", 10, 10), pacingMsg("You", 10, 8))
	if got != lifeChangeDelay {
		t.Errorf("want %v, got %v", lifeChangeDelay, got)
	}
}

func TestPhaseDelay_CreatureDamageChange_PausesLongest(t *testing.T) {
	creatureID := uuid.New()
	prev := pacingMsg("You", 10, 10)
	prev.State.You.Battlefield = []interactive.PermanentState{
		{ID: creatureID, Name: "Grizzly Bears", Damage: 0},
	}
	cur := pacingMsg("You", 10, 10)
	cur.State.You.Battlefield = []interactive.PermanentState{
		{ID: creatureID, Name: "Grizzly Bears", Damage: 1},
	}

	got := phaseDelay(prev, cur)
	if got != lifeChangeDelay {
		t.Errorf("want %v, got %v", lifeChangeDelay, got)
	}
}

func TestPhaseDelay_NewDamageLog_PausesLongest(t *testing.T) {
	prev := pacingMsg("You", 10, 10)
	prev.Log = []string{"Resolving Lightning Bolt"}
	cur := pacingMsg("You", 10, 10)
	cur.Log = []string{"Resolving Lightning Bolt", "  Lightning Bolt deals 3 damage to Grizzly Bears"}

	got := phaseDelay(prev, cur)
	if got != lifeChangeDelay {
		t.Errorf("want %v, got %v", lifeChangeDelay, got)
	}
}

func TestPhaseDelay_NilPrev_UsesBase(t *testing.T) {
	if got := phaseDelay(nil, pacingMsg("You", 10, 10)); got != phaseDisplayDelay {
		t.Errorf("want %v, got %v", phaseDisplayDelay, got)
	}
}

func TestPhaseDelay_NilState_UsesBase(t *testing.T) {
	if got := phaseDelay(nil, &interactive.GameMsg{}); got != phaseDisplayDelay {
		t.Errorf("want %v, got %v", phaseDisplayDelay, got)
	}
}

func TestDrainChoiceRequests_SyncsLiveStepBeforeShowingChoice(t *testing.T) {
	toTUI := make(chan interactive.GameMsg, 1)
	fromTUI := make(chan interactive.PriorityAction, 1)
	choiceReqs := make(chan interactive.ChoiceRequest, 1)
	choiceResps := make(chan interactive.ChoiceResponse, 1)
	human := interactive.NewHumanPlayerWithChannels("You", toTUI, fromTUI, choiceReqs, choiceResps)
	opponent := mage.NewBasePlayer("Opponent")
	game := mage.NewGame(human, opponent)
	game.SetStep(core.Upkeep)

	s := &DuelScreen{
		game:  game,
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				Step:         stepPrecombatMain,
				ActivePlayer: "Opponent",
				You:          interactive.PlayerState{Name: "You", Life: 20},
				Opponent:     interactive.PlayerState{Name: "Opponent", Life: 20},
			},
		},
	}
	choiceReqs <- interactive.ChoiceRequest{
		Type:   interactive.ChoiceMay,
		Reason: "pay upkeep",
		Options: []interactive.ChoiceOption{
			{Label: "Yes"},
			{Label: "No"},
		},
	}

	s.drainChoiceRequests()

	if s.choiceRequest == nil {
		t.Fatal("expected pending choice request")
	}
	if got := s.lastMsg.State.Step; got != "Upkeep" {
		t.Fatalf("expected choice to sync visible step to Upkeep, got %q", got)
	}
}
