package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
)

type exactTwoTarget struct {
	mage.Target
}

func (exactTwoTarget) Min() int { return 2 }
func (exactTwoTarget) Max() int { return 2 }

func TestTargetingExactTwoRequiresAndSubmitsTwoDistinctTargets(t *testing.T) {
	cardID := uuid.New()
	first := uuid.New()
	second := uuid.New()
	fromTUI := make(chan interactive.PriorityAction, 1)
	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       cardID,
		CardName:     "Ashes to Ashes",
		NeedsTarget:  true,
		TargetType:   exactTwoTarget{Target: mage.TargetCreature()},
		ValidTargets: []uuid.UUID{first, second},
	}
	human := interactive.NewHumanPlayerWithChannels("Test",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)
	s := &DuelScreen{human: human}

	s.enterTargetingMode(cardID, action.CardName, []interactive.ActionOption{action})
	s.selectTarget(first)
	if s.targetSelectionComplete() {
		t.Fatal("one target must not complete an exact-two selection")
	}
	if s.submitTargetingAction() {
		t.Fatal("incomplete target selection must not be submitted")
	}

	s.selectTarget(first)
	if len(s.selectedTargetIDs) != 0 {
		t.Fatal("selecting an already selected target should deselect it")
	}

	s.selectTarget(first)
	s.selectTarget(second)
	if !s.targetSelectionComplete() {
		t.Fatal("two targets should complete an exact-two selection")
	}
	if !s.submitTargetingAction() {
		t.Fatal("complete target selection should be submitted")
	}

	select {
	case got := <-fromTUI:
		if len(got.Targets) != 2 || got.Targets[0] != first || got.Targets[1] != second {
			t.Fatalf("submitted targets = %v, want [%v %v]", got.Targets, first, second)
		}
	default:
		t.Fatal("expected targeted action to be sent")
	}
}

func TestSingleTargetSelectionStillReplacesPreviousTarget(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	action := interactive.ActionOption{
		NeedsTarget:  true,
		TargetType:   mage.TargetCreature(),
		ValidTargets: []uuid.UUID{first, second},
	}
	s := &DuelScreen{}

	s.enterTargetingMode(uuid.New(), "Test", []interactive.ActionOption{action})
	s.selectTarget(first)
	s.selectTarget(second)

	if len(s.selectedTargetIDs) != 1 || s.selectedTargetIDs[0] != second {
		t.Fatalf("selected targets = %v, want [%v]", s.selectedTargetIDs, second)
	}
}

func TestTargetingPromptDescribesExactTargetCount(t *testing.T) {
	s := &DuelScreen{
		targetingAction: interactive.ActionOption{
			TargetType: exactTwoTarget{Target: mage.TargetCreature()},
		},
	}

	if got := s.targetingPrompt("Ashes to Ashes"); got != "Choose 2 targets for Ashes to Ashes" {
		t.Fatalf("targeting prompt = %q", got)
	}
}

func TestTargetingRaiseDead_AutoOpensGraveyardAndSelectsGraveyardCreature(t *testing.T) {
	cardID := uuid.New()
	graveCreatureID := uuid.New()
	graveInstantID := uuid.New()
	battlefieldCreatureID := uuid.New()
	fromTUI := make(chan interactive.PriorityAction, 1)

	state := &interactive.GameState{
		Step:         stepPrecombatMain,
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "You",
			Graveyard: []interactive.CardState{
				{ID: graveCreatureID, Name: "Grizzly Bears"},
				{ID: graveInstantID, Name: "Dark Ritual"},
			},
			Battlefield: []interactive.PermanentState{
				{ID: battlefieldCreatureID, Name: "Air Elemental", IsCreature: true},
			},
		},
		Opponent: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "Opponent",
		},
	}

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       cardID,
		CardName:     "Raise Dead",
		NeedsTarget:  true,
		TargetType:   mage.TargetCreatureInYourGraveyard(),
		ValidTargets: []uuid.UUID{graveCreatureID},
	}

	human := interactive.NewHumanPlayerWithChannels("Test",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)

	self := &duelPlayer{name: "You"}
	opponent := &duelPlayer{name: "Opponent"}
	s := &DuelScreen{
		human:            human,
		self:             self,
		opponent:         opponent,
		lastMsg:          &interactive.GameMsg{State: state},
		doneBtn:          [3]*ebiten.Image{ebiten.NewImage(50, 20), ebiten.NewImage(50, 20), ebiten.NewImage(50, 20)},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.enterTargetingMode(cardID, action.CardName, []interactive.ActionOption{action})

	if s.viewingGraveyard != self {
		t.Fatalf("expected viewingGraveyard to automatically open self graveyard, got %v", s.viewingGraveyard)
	}

	// Clicking battlefield creature should not target it
	s.handleTargetClick(duelBoardX+50, duelPlayerBoardY+50)
	if len(s.selectedTargetIDs) != 0 {
		t.Fatalf("battlefield creature must not be selected, got %v", s.selectedTargetIDs)
	}

	// Click on the valid creature in the graveyard view (index 0)
	bounds := graveyardCardBounds(0, 1024, 210)
	clickX := bounds.Min.X + 10
	clickY := bounds.Min.Y + 10

	s.updateTargetingMouse(clickX, clickY, true, 1024)

	if s.viewingGraveyard != nil {
		t.Fatalf("expected viewingGraveyard to close after selecting single target, got %v", s.viewingGraveyard)
	}
	if s.targetingCardID != uuid.Nil {
		t.Fatalf("expected targeting mode to exit after single target selection")
	}

	select {
	case got := <-fromTUI:
		if len(got.Targets) != 1 || got.Targets[0] != graveCreatureID {
			t.Fatalf("submitted targets = %v, want [%v]", got.Targets, graveCreatureID)
		}
		if got.CardID != cardID {
			t.Fatalf("submitted cardID = %v, want %v", got.CardID, cardID)
		}
	default:
		t.Fatal("expected Raise Dead action to be submitted")
	}
}

func TestTargetingRaiseDead_IgnoresInvalidGraveyardCardClick(t *testing.T) {
	cardID := uuid.New()
	graveCreatureID := uuid.New()
	graveInstantID := uuid.New()
	fromTUI := make(chan interactive.PriorityAction, 1)

	state := &interactive.GameState{
		Step:         stepPrecombatMain,
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "You",
			Graveyard: []interactive.CardState{
				{ID: graveCreatureID, Name: "Grizzly Bears"},
				{ID: graveInstantID, Name: "Dark Ritual"},
			},
		},
		Opponent: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "Opponent",
		},
	}

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       cardID,
		CardName:     "Raise Dead",
		NeedsTarget:  true,
		TargetType:   mage.TargetCreatureInYourGraveyard(),
		ValidTargets: []uuid.UUID{graveCreatureID},
	}

	human := interactive.NewHumanPlayerWithChannels("Test",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)

	self := &duelPlayer{name: "You"}
	s := &DuelScreen{
		human:            human,
		self:             self,
		opponent:         &duelPlayer{name: "Opponent"},
		lastMsg:          &interactive.GameMsg{State: state},
		doneBtn:          [3]*ebiten.Image{ebiten.NewImage(50, 20), ebiten.NewImage(50, 20), ebiten.NewImage(50, 20)},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.enterTargetingMode(cardID, action.CardName, []interactive.ActionOption{action})

	// Click on Dark Ritual (index 1), which is NOT a valid target for Raise Dead
	bounds := graveyardCardBounds(1, 1024, 210)
	clickX := bounds.Min.X + 10
	clickY := bounds.Min.Y + 10

	s.updateTargetingMouse(clickX, clickY, true, 1024)

	// Should still be in targeting mode and viewing graveyard
	if s.viewingGraveyard != self {
		t.Fatalf("expected viewingGraveyard to remain open after clicking non-target card, got %v", s.viewingGraveyard)
	}
	if len(s.selectedTargetIDs) != 0 {
		t.Fatalf("non-target card should not be selected, got %v", s.selectedTargetIDs)
	}
}

func TestTargetingAnimateDead_ShowsCreaturesInAllGraveyards(t *testing.T) {
	cardID := uuid.New()
	youCreatureID := uuid.New()
	oppCreatureID := uuid.New()
	fromTUI := make(chan interactive.PriorityAction, 1)

	state := &interactive.GameState{
		Step:         stepPrecombatMain,
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "You",
			Graveyard: []interactive.CardState{
				{ID: youCreatureID, Name: "Grizzly Bears"},
			},
		},
		Opponent: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "Opponent",
			Graveyard: []interactive.CardState{
				{ID: oppCreatureID, Name: "Serra Angel"},
			},
		},
	}

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       cardID,
		CardName:     "Animate Dead",
		NeedsTarget:  true,
		TargetType:   mage.TargetCreatureCardInAnyGraveyard(),
		ValidTargets: []uuid.UUID{youCreatureID, oppCreatureID},
	}

	human := interactive.NewHumanPlayerWithChannels("Test",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)

	self := &duelPlayer{name: "You"}
	opponent := &duelPlayer{name: "Opponent"}
	s := &DuelScreen{
		human:            human,
		self:             self,
		opponent:         opponent,
		lastMsg:          &interactive.GameMsg{State: state},
		doneBtn:          [3]*ebiten.Image{ebiten.NewImage(50, 20), ebiten.NewImage(50, 20), ebiten.NewImage(50, 20)},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.enterTargetingMode(cardID, action.CardName, []interactive.ActionOption{action})

	if !s.viewingAllGraveyards {
		t.Fatalf("expected viewingAllGraveyards to be true for Animate Dead with targets in both graveyards")
	}

	players := s.displayedGraveyardPlayers()
	if len(players) != 2 {
		t.Fatalf("expected 2 displayed graveyard players, got %d", len(players))
	}

	sections := s.graveyardViewLayout(1024)
	if len(sections) != 2 {
		t.Fatalf("expected 2 graveyard sections, got %d", len(sections))
	}

	var oppCardRect image.Rectangle
	foundOpp := false
	for _, sec := range sections {
		if sec.Player == opponent {
			for _, gc := range sec.Cards {
				if gc.Card.ID == oppCreatureID {
					oppCardRect = gc.Rect
					foundOpp = true
				}
			}
		}
	}
	if !foundOpp {
		t.Fatalf("did not find opponent creature in graveyard sections")
	}

	// Click on opponent's Serra Angel
	s.updateTargetingMouse(oppCardRect.Min.X+5, oppCardRect.Min.Y+5, true, 1024)

	if s.isViewingGraveyard() {
		t.Fatalf("expected graveyard view to close after submitting single target, got isViewingGraveyard=true")
	}
	if s.targetingCardID != uuid.Nil {
		t.Fatalf("expected targeting mode to exit after single target selection")
	}

	select {
	case got := <-fromTUI:
		if len(got.Targets) != 1 || got.Targets[0] != oppCreatureID {
			t.Fatalf("submitted targets = %v, want [%v]", got.Targets, oppCreatureID)
		}
	default:
		t.Fatal("expected Animate Dead action with opponent target to be submitted")
	}
}
