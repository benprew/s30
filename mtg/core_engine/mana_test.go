package core_engine

import (
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
