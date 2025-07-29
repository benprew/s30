package core_engine

import (
	"fmt"
	"slices"
	"testing"
)

func TestManaPool_AddMana(t *testing.T) {
	pool := ManaPool{}
	pool.AddMana([]rune{'W'})

	if len(pool) != 1 {
		t.Errorf("Expected mana pool to have length 1, but got %d", len(pool))
	}

	if len(pool[0]) != 1 || pool[0][0] != 'W' {
		t.Errorf("Expected mana pool to contain 'W', but got %v", pool)
	}
}

func TestManaPool_RemoveMana(t *testing.T) {
	pool := ManaPool{{'W'}, {'U'}, {'B'}}

	pool.RemoveMana('U')

	if len(pool) != 2 {
		t.Errorf("Expected mana pool to have length 2, but got %d", len(pool))
	}

	expected := ManaPool{{'W'}, {'B'}}
	for i, mana := range pool {
		if len(mana) != len(expected[i]) {
			t.Errorf("Expected mana pool to contain %v, but got %v", expected, pool)
			return
		}
		if mana[0] != expected[i][0] {
			t.Errorf("Expected mana pool to contain %v, but got %v", expected, pool)
			return
		}
	}
}

func TestManaPool_CanPay(t *testing.T) {
	pool := ManaPool{{'W'}, {'U'}, {'B'}, {'R'}, {'G'}}

	if !pool.CanPay("WUBRG") {
		t.Errorf("Expected to be able to pay WUBRG, but could not")
	}

	if pool.CanPay("WWWWWW") {
		t.Errorf("Expected to not be able to pay WWWWWW, but could")
	}
}

func TestManaPool_Pay(t *testing.T) {
	pool := ManaPool{{'W'}, {'U'}, {'B'}, {'R'}, {'G'}}

	err := pool.Pay("UB")
	if err != nil {
		t.Errorf("Expected to be able to pay UB, but got error: %v", err)
	}

	if !pool.CanPay("WRG") {
		t.Errorf("Expected to be able to pay WRG, but could not")
	}

	err = pool.Pay("WWWWWW")
	if err == nil {
		t.Errorf("Expected to not be able to pay WWWWWW, but could")
	}
}

func TestAvailableMana(t *testing.T) {
	// test a player has 2 available mana with an untapped land and elf
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	// Find a Forest card and put it on the battlefield
	var forest *Card
	for _, card := range player.Hand {
		if card.Name() == "Forest" {
			forest = card
			break
		}
	}
	game.PlayLand(player, forest)

	// Check available mana
	manaPool := game.AvailableMana(player, player.ManaPool)
	expected := ManaPool{}
	expected.AddMana([]rune{'G'})
	if !slices.Equal(manaPool[0], expected[0]) {
		t.Fatalf("Mana pool should == 'G'")
	}

	// Find a Llanowar Elves card and put it on the battlefield
	var elvesCard *Card
	for _, card := range player.Hand {
		if card.Name() == "Llanowar Elves" {
			elvesCard = card
			break
		}
	}
	err := game.Resolve(&StackItem{Events: nil, Player: player, Card: elvesCard})
	if err != nil {
		t.Fatalf("Can't cast elves")
	}

	elvesCard.Active = true
	forest.Tapped = false

	// Check available mana
	manaPool = game.AvailableMana(player, player.ManaPool)
	expectedPool := ManaPool{[]rune{'G'}, []rune{'G'}}

	fmt.Println(expectedPool)
	fmt.Println(manaPool)
	// Check that both mana are green
	for i, mana := range expectedPool {
		if !slices.Equal(mana, manaPool[i]) {
			t.Errorf("Expected green mana, got %v", string(mana))
		}
	}
}

func TestCanPayColorless(t *testing.T) {
	cost := "2WW"
	pool := ManaPool{[]rune{'W'}, []rune{'W'}, []rune{'R'}, []rune{'B'}}

	if !pool.CanPay(cost) {
		t.Errorf("Unable to pay colorless cost")
	}
}

func TestParseCostCurlyBraceFormat(t *testing.T) {
	pool := ManaPool{}

	tests := []struct {
		cost     string
		expected map[rune]int
	}{
		{"{3}{G}{R}", map[rune]int{'C': 3, 'G': 1, 'R': 1}},
		{"{2}{W}{W}", map[rune]int{'C': 2, 'W': 2}},
		{"{1}{U}{B}", map[rune]int{'C': 1, 'U': 1, 'B': 1}},
		{"{5}", map[rune]int{'C': 5}},
		{"{W}{U}{B}{R}{G}", map[rune]int{'W': 1, 'U': 1, 'B': 1, 'R': 1, 'G': 1}},
		{"{0}", map[rune]int{'C': 0}},
	}

	for _, test := range tests {
		result := pool.ParseCost(test.cost)

		// Check that all expected keys are present with correct values
		for expectedRune, expectedCount := range test.expected {
			if result[expectedRune] != expectedCount {
				t.Errorf("For cost %s, expected %c: %d, got %d", test.cost, expectedRune, expectedCount, result[expectedRune])
			}
		}

		// Check that no unexpected keys are present
		for resultRune, resultCount := range result {
			if test.expected[resultRune] != resultCount {
				t.Errorf("For cost %s, unexpected %c: %d", test.cost, resultRune, resultCount)
			}
		}
	}
}
