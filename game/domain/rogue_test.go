package domain

import (
	"testing"
)

// TestLoadRogues loads the TOML rogues from the repository and performs
// basic sanity checks to ensure they parsed correctly.
func TestLoadRogues(t *testing.T) {
	if len(Rogues) == 0 {
		t.Fatal("no rogues were loaded from configs/rogues")
	}

	for key, r := range Rogues {
		if r == nil {
			t.Fatalf("rogue %s is nil", key)
		}
		if r.Name == "" {
			t.Fatalf("rogue %s has empty Name field", key)
		}
		// Expect the TOML Name to match the map key in normal operation.
		if r.Name != key {
			t.Logf("warning: map key %q does not match Rogue.Name %q", key, r.Name)
		}

		if len(r.Deck) == 0 {
			t.Fatalf("rogue %s (%s) has empty Deck", key, r.Name)
		}
		for i, de := range r.Deck {
			if de.Name == "" {
				t.Fatalf("rogue %s deck entry %d has empty Name", key, i)
			}
			if de.Count <= 0 {
				t.Fatalf("rogue %s deck entry %s has non-positive Count %d", key, de.Name, de.Count)
			}
		}
	}
}

func TestLoadImages(t *testing.T) {
	r, ok := Rogues["Lord of Fate"]
	if !ok || r == nil {
		t.Fatalf("rogue 'Lord of Fate' not found in loaded rogues")
	}

	if err := r.LoadImages(); err != nil {
		t.Fatalf("LoadImages failed for 'Lord of Fate': %v", err)
	}
}
