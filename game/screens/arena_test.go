package screens

import (
	"image"
	"math/rand"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
)

func TestArenaEditorCannotSellAndReturnsToAnte(t *testing.T) {
	card := domain.FindCardByName("Lightning Bolt")
	player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}, MinDeckSize: 15}
	player.CardCollection.AddCardToDeck(card, 0, 1)
	editor := &EditDeckScreen{Player: player, DisableSelling: true, ReturnScr: screenui.DuelAnteScr}
	if editor.sellCard(card, true) || player.CardCollection.NumCards() != 1 || player.Gold != 0 {
		t.Fatal("arena editor sold a card")
	}
	if editor.leaveEditor() != screenui.EditDeckScr || !editor.leaveWarning {
		t.Fatal("undersized deck must show padding warning")
	}
	player.MinDeckSize = 1
	if editor.leaveEditor() != screenui.DuelAnteScr {
		t.Fatal("arena editor must return to arena ante")
	}
}

func TestArenaVictoryReturnsToEditorThenChampion(t *testing.T) {
	player := &domain.Player{Character: domain.Character{Life: 10, CardCollection: domain.NewCardCollection()}, Gold: 300}
	tile := image.Pt(0, 0)
	level := &world.Level{W: 1, H: 1, Tiles: [][]*world.Tile{{{}}}, Player: player, RandomEncounters: []world.RandomEncounter{{Tile: tile, Type: world.EncounterArena}}}
	run, err := domain.NewArenaRun(player, rand.New(rand.NewSource(4)))
	if err != nil {
		t.Fatal(err)
	}
	session := &arenaSession{run: run, level: level, tile: tile}
	for round := 1; round <= 7; round++ {
		name, screen, outcomeErr := session.outcome(true)
		if outcomeErr != nil || name != screenui.DuelWinScr {
			t.Fatalf("round %d outcome: %v", round, outcomeErr)
		}
		win := screen.(*DuelWinScreen)
		if round < 7 {
			if win.ReturnScr != screenui.EditDeckScr {
				t.Fatal("win must return to arena editor")
			}
			editor := win.ReturnScreen.(*EditDeckScreen)
			if !editor.DisableSelling || editor.Player != run.Player || editor.ReturnScreen == nil {
				t.Fatal("arena editor is not linked to next opponent")
			}
		} else if win.ReturnScr != screenui.RandomEncounterScr {
			t.Fatal("final win must show champion screen")
		}
	}
	if len(level.RandomEncounters) != 0 || !run.Champion {
		t.Fatal("championship did not complete the encounter")
	}
}

func TestArenaLossCompletesEncounter(t *testing.T) {
	player := &domain.Player{Character: domain.Character{CardCollection: domain.NewCardCollection()}, Gold: 300}
	tile := image.Pt(0, 0)
	level := &world.Level{W: 1, H: 1, Tiles: [][]*world.Tile{{{}}}, Player: player, RandomEncounters: []world.RandomEncounter{{Tile: tile, Type: world.EncounterArena}}}
	run, err := domain.NewArenaRun(player, rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatal(err)
	}
	session := &arenaSession{run: run, level: level, tile: tile}
	name, screen, err := session.outcome(false)
	if err != nil || name != screenui.DuelLoseScr || screen == nil || !run.Finished || len(level.RandomEncounters) != 0 {
		t.Fatal("loss did not end arena and show loss screen")
	}
	if player.CardCollection.NumCards() != 0 {
		t.Fatal("loss retained temporary cards")
	}
}
