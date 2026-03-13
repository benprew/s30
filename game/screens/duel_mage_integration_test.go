package screens

import (
	"fmt"
	"testing"
	"time"

	_ "git.sr.ht/~cdcarter/mage-go/cards"
	mage "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
)

func TestMageIntegration_PlayLandShowsOnBattlefield(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	ai := ai.NewAIPlayer("Opponent")

	// Give human known cards
	for range 5 {
		c, err := mage.CreateCard("Mountain")
		if err != nil {
			t.Fatalf("Failed to create Mountain: %v", err)
		}
		human.AddToLibrary(c)
	}
	for range 5 {
		c, err := mage.CreateCard("Forest")
		if err != nil {
			t.Fatalf("Failed to create Forest: %v", err)
		}
		ai.AddToLibrary(c)
	}

	human.ShuffleLibrary()
	ai.ShuffleLibrary()

	g := mage.NewGame(human, ai)

	// Draw opening hands
	for range 4 {
		human.DrawCard()
	}
	for range 4 {
		ai.DrawCard()
	}

	t.Logf("Human hand size: %d", len(human.Hand()))
	for _, c := range human.Hand() {
		t.Logf("  Hand card: %s (ID: %s)", c.Name(), c.ID())
	}
	t.Logf("Human library size: %d", len(human.Library()))
	t.Logf("AI hand size: %d", len(ai.Hand()))

	// Start game loop
	go interactive.RunGameLoop(g, 0, 0)

	// Wait for first message (should be main phase)
	var msg interactive.GameMsg
	select {
	case msg = <-human.ToTUI():
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for first message")
	}

	t.Logf("First message - Step: %s, Prompt: %v, Options: %d", msg.State.Step, msg.Prompt, len(msg.Options))
	t.Logf("  You.Hand: %d cards, You.Battlefield: %d perms", len(msg.State.You.Hand), len(msg.State.You.Battlefield))
	t.Logf("  You.Life: %d, You.LibraryCount: %d", msg.State.You.Life, msg.State.You.LibraryCount)

	for _, opt := range msg.Options {
		t.Logf("  Option: type=%v label=%q cardName=%q cardID=%s permID=%s needsTarget=%v",
			opt.Type, opt.Label, opt.CardName, opt.CardID, opt.PermanentID, opt.NeedsTarget)
	}

	// Find a PlayLand action for Mountain
	var playLandAction *interactive.ActionOption
	for _, opt := range msg.Options {
		if opt.Type == interactive.ActionPlayLand {
			playLandAction = &opt
			break
		}
	}

	if playLandAction == nil {
		t.Fatal("No PlayLand action available in main phase with Mountain in hand")
	}

	t.Logf("Playing land: %s (CardID: %s)", playLandAction.CardName, playLandAction.CardID)

	// Send play land action
	human.FromTUI() <- interactive.PriorityAction{
		Type:   interactive.ActionPlayLand,
		CardID: playLandAction.CardID,
	}

	// Drain messages until we get one with the land on battlefield or we time out
	deadline := time.After(5 * time.Second)
	found := false
	for !found {
		select {
		case msg = <-human.ToTUI():
			t.Logf("Message - Step: %s, Prompt: %v, GameOver: %v", msg.State.Step, msg.Prompt, msg.GameOver)
			t.Logf("  You.Battlefield: %d perms", len(msg.State.You.Battlefield))
			for _, p := range msg.State.You.Battlefield {
				t.Logf("    Perm: %s (ID: %s) land=%v creature=%v tapped=%v",
					p.Name, p.ID, p.IsLand, p.IsCreature, p.Tapped)
			}
			t.Logf("  Options: %d", len(msg.Options))
			for _, opt := range msg.Options {
				t.Logf("    Option: type=%v label=%q", opt.Type, opt.Label)
			}

			if len(msg.State.You.Battlefield) > 0 {
				found = true
			}

			if msg.GameOver {
				t.Fatal("Game ended before land appeared on battlefield")
			}

			// Keep passing to advance the game if we get priority
			if len(msg.Options) > 0 && !found {
				human.FromTUI() <- interactive.PriorityAction{Type: interactive.ActionPass}
			}
		case <-deadline:
			t.Fatal("Timed out waiting for land on battlefield")
		}
	}

	if len(msg.State.You.Battlefield) == 0 {
		t.Fatal("Mountain should be on battlefield after playing it")
	}

	landFound := false
	for _, p := range msg.State.You.Battlefield {
		if p.Name == "Mountain" && p.IsLand {
			landFound = true
			break
		}
	}
	if !landFound {
		t.Error("Expected Mountain on battlefield")
	}
}

func TestMageIntegration_CastArtifactAfterPlayingLands(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	ai := ai.NewAIPlayer("Opponent")

	// Give human Sol Ring + Mountains
	solRing, err := mage.CreateCard("Sol Ring")
	if err != nil {
		t.Fatalf("Failed to create Sol Ring: %v", err)
	}
	solRing.SetOwner(human.PlayerID())
	human.AddToHand(solRing)

	for range 3 {
		c, err := mage.CreateCard("Mountain")
		if err != nil {
			t.Fatalf("Failed to create Mountain: %v", err)
		}
		c.SetOwner(human.PlayerID())
		human.AddToHand(c)
	}

	// Pad libraries
	for range 10 {
		c, _ := mage.CreateCard("Mountain")
		human.AddToLibrary(c)
	}
	for range 10 {
		c, _ := mage.CreateCard("Forest")
		ai.AddToLibrary(c)
		ai.AddToHand(c)
	}

	g := mage.NewGame(human, ai)

	t.Logf("Human hand: %d cards", len(human.Hand()))
	for _, c := range human.Hand() {
		t.Logf("  %s (ID: %s, cost: %s)", c.Name(), c.ID(), c.ManaCost())
	}

	go interactive.RunGameLoop(g, 0, 0)

	// Helper to get next message with priority
	getMsg := func() interactive.GameMsg {
		t.Helper()
		select {
		case msg := <-human.ToTUI():
			return msg
		case <-time.After(5 * time.Second):
			t.Fatal("Timed out waiting for message")
			return interactive.GameMsg{}
		}
	}

	// Get initial main phase message
	msg := getMsg()
	t.Logf("Step: %s, Prompt: %v", msg.State.Step, msg.Prompt)

	// Find and play a Mountain
	var playLand *interactive.ActionOption
	for _, opt := range msg.Options {
		if opt.Type == interactive.ActionPlayLand && opt.CardName == "Mountain" {
			playLand = &opt
			break
		}
	}
	if playLand == nil {
		for _, opt := range msg.Options {
			t.Logf("  Available: type=%v label=%q cardName=%q", opt.Type, opt.Label, opt.CardName)
		}
		t.Fatal("No PlayLand action for Mountain")
	}

	t.Logf("Playing Mountain (ID: %s)", playLand.CardID)
	human.FromTUI() <- interactive.PriorityAction{
		Type:   interactive.ActionPlayLand,
		CardID: playLand.CardID,
	}

	// Read messages until we get one with options (next priority)
	var castSolRing *interactive.ActionOption
	deadline := time.After(5 * time.Second)
	for castSolRing == nil {
		select {
		case msg = <-human.ToTUI():
			t.Logf("Step: %s, Prompt: %v, Battlefield: %d, Hand: %d",
				msg.State.Step, msg.Prompt, len(msg.State.You.Battlefield), len(msg.State.You.Hand))
			for _, p := range msg.State.You.Battlefield {
				t.Logf("  Battlefield: %s tapped=%v", p.Name, p.Tapped)
			}

			// Check mana pool
			mp := msg.State.You.ManaPool
			t.Logf("  ManaPool: W=%d U=%d B=%d R=%d G=%d C=%d",
				mp.White, mp.Blue, mp.Black, mp.Red, mp.Green, mp.Colorless)

			for _, opt := range msg.Options {
				t.Logf("  Option: type=%v label=%q cardName=%q cost=%q",
					opt.Type, opt.Label, opt.CardName, opt.ManaCost)
				if opt.Type == interactive.ActionCastSpell && opt.CardName == "Sol Ring" {
					castSolRing = &opt
				}
			}

			if msg.GameOver {
				t.Fatal("Game ended unexpectedly")
			}

			// If we have priority but no cast option, tap the mountain first then pass
			if len(msg.Options) > 0 && castSolRing == nil {
				// Look for activate ability on Mountain (tap for mana)
				var tapMtn *interactive.ActionOption
				for _, opt := range msg.Options {
					if opt.Type == interactive.ActionActivateAbility {
						tapMtn = &opt
						break
					}
				}
				if tapMtn != nil {
					t.Logf("Activating: %s (PermanentID: %s)", tapMtn.Label, tapMtn.PermanentID)
					human.FromTUI() <- interactive.PriorityAction{
						Type:         interactive.ActionActivateAbility,
						PermanentID:  tapMtn.PermanentID,
						AbilityIndex: tapMtn.AbilityIndex,
					}
				} else {
					human.FromTUI() <- interactive.PriorityAction{Type: interactive.ActionPass}
				}
			}
		case <-deadline:
			t.Logf("Final state - Step: %s, Battlefield: %d", msg.State.Step, len(msg.State.You.Battlefield))
			for _, opt := range msg.Options {
				t.Logf("  Option: type=%v label=%q", opt.Type, opt.Label)
			}
			t.Fatal("Timed out - never got CastSpell option for Sol Ring")
		}
	}

	t.Logf("Found CastSpell for Sol Ring! CardID: %s", castSolRing.CardID)
	fmt.Println("SUCCESS: Sol Ring is castable after playing Mountain and tapping for mana")
}
