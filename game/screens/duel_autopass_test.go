package screens

import (
	"image"
	"sync"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/core"
)

func setupMain2Players() (*core.Player, *core.Player) {
	player := &core.Player{
		ID:          0,
		LifeTotal:   20,
		ManaPool:    core.ManaPool{},
		Hand:        []*core.Card{},
		Library:     []*core.Card{},
		Battlefield: []*core.Card{},
		Graveyard:   []*core.Card{},
		Exile:       []*core.Card{},
		Turn:        &core.Turn{},
		InputChan:   make(chan core.PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
	}
	opponent := &core.Player{
		ID:          1,
		LifeTotal:   20,
		ManaPool:    core.ManaPool{},
		Hand:        []*core.Card{},
		Library:     []*core.Card{},
		Battlefield: []*core.Card{},
		Graveyard:   []*core.Card{},
		Exile:       []*core.Card{},
		Turn:        &core.Turn{},
		InputChan:   make(chan core.PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
		IsAI:        true,
	}
	return player, opponent
}

func addTestLandToBattlefield(player *core.Player, name string, id core.EntityID) *core.Card {
	card := core.NewCardFromDomain(domain.FindCardByName(name), id, player)
	card.Active = true
	card.Tapped = false
	card.CurrentZone = core.ZoneBattlefield
	player.Battlefield = append(player.Battlefield, card)
	return card
}

func addTestCardToHand(player *core.Player, name string, id core.EntityID) *core.Card {
	card := core.NewCardFromDomain(domain.FindCardByName(name), id, player)
	card.CurrentZone = core.ZoneHand
	player.Hand = append(player.Hand, card)
	return card
}

// Regression test: active player should have sorcery-speed actions in main
// phase 2 (play lands, cast creatures/sorceries). The autopass goroutine
// must not treat main phase 2 as a phase with no meaningful actions.
func TestAutoPass_DoesNotPassMain2WithPlayableCards(t *testing.T) {
	player, opponent := setupMain2Players()
	gs := core.NewGame([]*core.Player{player, opponent}, false)
	gs.ActivePlayer = 0
	gs.CurrentPlayer = 0

	addTestLandToBattlefield(player, "Forest", 100)
	addTestCardToHand(player, "Forest", 101)
	addTestCardToHand(player, "Llanowar Elves", 102)

	player.Turn.Phase = core.PhaseMain2
	player.Turn.LandPlayed = false

	s := &DuelScreen{
		gameState:        gs,
		self:             &duelPlayer{core: player},
		opponent:         &duelPlayer{core: opponent},
		pendingAttackers: make(map[core.EntityID]bool),
		pendingBlockers:  make(map[core.EntityID]core.EntityID),
		cardActions:      make(map[core.EntityID][]core.PlayerAction),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[core.EntityID]image.Point),
	}

	actions := gs.AvailableActions(player)

	onlyPass := true
	for _, a := range actions {
		if a.Type != core.ActionPassPriority {
			onlyPass = false
			break
		}
	}
	if onlyPass {
		t.Error("autopass incorrectly fires in main phase 2: player has playable land and castable creature")
	}

	hasPlayLand := false
	hasCastSpell := false
	for _, a := range actions {
		switch a.Type {
		case core.ActionPlayLand:
			hasPlayLand = true
		case core.ActionCastSpell:
			hasCastSpell = true
		}
	}
	if !hasPlayLand {
		t.Error("expected PlayLand in main phase 2")
	}
	if !hasCastSpell {
		t.Error("expected CastSpell in main phase 2 (Llanowar Elves with Forest on battlefield)")
	}

	s.refreshCardActions()
	if len(s.cardActions) == 0 {
		t.Error("refreshCardActions should populate card actions in main phase 2")
	}
}

// Integration test: run a full turn with autopass-style logic that tracks
// whether the "only pass" branch fires during main phase 2. The player never
// plays a land, so by main phase 2 PlayLand should still be available and
// the autopass should NOT fire.
func TestMain2_AutoPassDoesNotFireDuringFullTurn(t *testing.T) {
	player, opponent := setupMain2Players()
	gs := core.NewGame([]*core.Player{player, opponent}, false)

	for i := range 10 {
		card := core.NewCardFromDomain(domain.FindCardByName("Forest"), core.EntityID(300+i), player)
		player.Library = append(player.Library, card)
	}
	for i := range 10 {
		card := core.NewCardFromDomain(domain.FindCardByName("Forest"), core.EntityID(400+i), opponent)
		opponent.Library = append(opponent.Library, card)
	}

	gs.StartGame()

	done := make(chan struct{})
	var mu sync.Mutex
	autoPassFiredInMain2 := false

	go func() {
		for {
			select {
			case <-done:
				return
			case <-player.WaitingChan:
				if player.Turn.Discarding {
					actions := gs.AvailableActions(player)
					for _, a := range actions {
						if a.Type == core.ActionDiscard {
							player.InputChan <- a
							break
						}
					}
					continue
				}
				actions := gs.AvailableActions(player)
				onlyPass := true
				for _, a := range actions {
					if a.Type != core.ActionPassPriority {
						onlyPass = false
						break
					}
				}
				if onlyPass {
					if player.Turn.Phase == core.PhaseMain2 {
						mu.Lock()
						autoPassFiredInMain2 = true
						mu.Unlock()
					}
				}
				player.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}
			case <-opponent.WaitingChan:
				opponent.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}
			}
		}
	}()

	gs.ActivePlayer = 0
	gs.CurrentPlayer = 0
	gs.NextTurn()

	close(done)

	mu.Lock()
	defer mu.Unlock()

	if autoPassFiredInMain2 {
		t.Error("autopass fired during main phase 2 when player has lands in hand (PlayLand should be available)")
	}
}

// Integration test: run through a full turn using NextTurn and verify the
// active player receives priority in main phase 2 with non-pass actions.
// The player's input goroutine always passes priority (never plays a land),
// so by main phase 2 they still have an unplayed land in hand.
func TestMain2_PriorityDuringFullTurn(t *testing.T) {
	player, opponent := setupMain2Players()
	gs := core.NewGame([]*core.Player{player, opponent}, false)

	for i := range 10 {
		card := core.NewCardFromDomain(domain.FindCardByName("Forest"), core.EntityID(100+i), player)
		player.Library = append(player.Library, card)
	}
	for i := range 10 {
		card := core.NewCardFromDomain(domain.FindCardByName("Forest"), core.EntityID(200+i), opponent)
		opponent.Library = append(opponent.Library, card)
	}

	gs.StartGame()

	type phaseRecord struct {
		phase   core.Phase
		nonPass bool
	}

	var mu sync.Mutex
	var records []phaseRecord

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-player.WaitingChan:
				actions := gs.AvailableActions(player)
				if player.Turn.Discarding {
					for _, a := range actions {
						if a.Type == core.ActionDiscard {
							player.InputChan <- a
							break
						}
					}
					continue
				}
				nonPass := false
				for _, a := range actions {
					if a.Type != core.ActionPassPriority {
						nonPass = true
						break
					}
				}
				mu.Lock()
				records = append(records, phaseRecord{
					phase:   player.Turn.Phase,
					nonPass: nonPass,
				})
				mu.Unlock()
				player.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}
			case <-opponent.WaitingChan:
				opponent.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}
			}
		}
	}()

	gs.ActivePlayer = 0
	gs.CurrentPlayer = 0
	gs.NextTurn()

	close(done)

	mu.Lock()
	defer mu.Unlock()

	foundMain2 := false
	main2NonPass := false
	for _, r := range records {
		if r.phase == core.PhaseMain2 {
			foundMain2 = true
			if r.nonPass {
				main2NonPass = true
			}
		}
	}

	if !foundMain2 {
		t.Error("active player never received priority in main phase 2")
	}
	if foundMain2 && !main2NonPass {
		t.Error("active player had only PassPriority in main phase 2; turn would be autopassed but player has land in hand")
	}
}
