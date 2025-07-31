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

	if !pool.CanPay("{W}{U}{B}{R}{G}") {
		t.Errorf("Expected to be able to pay {W}{U}{B}{R}{G}, but could not")
	}

	if pool.CanPay("{W}{W}{W}{W}{W}{W}") {
		t.Errorf("Expected to not be able to pay {W}{W}{W}{W}{W}{W}, but could")
	}
}

func TestManaPool_Pay(t *testing.T) {
	pool := ManaPool{{'W'}, {'U'}, {'B'}, {'R'}, {'G'}}

	err := pool.Pay("{U}{B}")
	if err != nil {
		t.Errorf("Expected to be able to pay {U}{B}, but got error: %v", err)
	}

	if !pool.CanPay("{W}{R}{G}") {
		t.Errorf("Expected to be able to pay {W}{R}{G}, but could not")
	}

	err = pool.Pay("{W}{W}{W}{W}{W}{W}")
	if err == nil {
		t.Errorf("Expected to not be able to pay {W}{W}{W}{W}{W}{W}, but could")
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
	cost := "{2}{W}{W}"
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

func TestCanPayCurlyBraceFormat(t *testing.T) {
	tests := []struct {
		name     string
		pool     ManaPool
		cost     string
		expected bool
	}{
		{
			name:     "Can pay simple colored cost",
			pool:     ManaPool{[]rune{'G'}, []rune{'R'}},
			cost:     "{G}{R}",
			expected: true,
		},
		{
			name:     "Can pay colorless and colored cost",
			pool:     ManaPool{[]rune{'W'}, []rune{'W'}, []rune{'R'}, []rune{'B'}},
			cost:     "{2}{W}{W}",
			expected: true,
		},
		{
			name:     "Can pay with excess mana",
			pool:     ManaPool{[]rune{'G'}, []rune{'G'}, []rune{'G'}, []rune{'R'}},
			cost:     "{1}{G}",
			expected: true,
		},
		{
			name:     "Cannot pay insufficient colored mana",
			pool:     ManaPool{[]rune{'G'}, []rune{'R'}},
			cost:     "{W}{U}",
			expected: false,
		},
		{
			name:     "Cannot pay insufficient colorless mana",
			pool:     ManaPool{[]rune{'G'}},
			cost:     "{3}{G}",
			expected: false,
		},
		{
			name:     "Can pay zero cost",
			pool:     ManaPool{},
			cost:     "{0}",
			expected: true,
		},
		{
			name:     "Can pay large colorless cost with many sources",
			pool:     ManaPool{[]rune{'2'}, []rune{'W'}, []rune{'G'}},
			cost:     "{3}{G}",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.pool.CanPay(test.cost)
			if result != test.expected {
				t.Errorf("Expected CanPay(%s) to be %v, but got %v", test.cost, test.expected, result)
			}
		})
	}
}
