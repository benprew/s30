package duel

import "testing"

// Tokens never pass through a deck, so the image map must know them up front or
// a Wasp from The Hive is drawn as an empty frame.
func TestBuildCardImageMapIncludesTokens(t *testing.T) {
	m := buildCardImageMap()
	wasp := m["Wasp"]
	if wasp == nil {
		t.Fatal("buildCardImageMap() has no entry for the Wasp token")
	}
	if wasp.BorderCropURL == "" {
		t.Error("Wasp entry has no BorderCropURL")
	}
}

// Tokens without a printing need an entry too, or they fall back to the
// board's empty frame instead of a labeled card.
func TestBuildCardImageMapIncludesArtlessTokens(t *testing.T) {
	m := buildCardImageMap()
	for _, name := range []string{"Bird", "Djinn", "Tetravite", "Spawn of Azar"} {
		if m[name] == nil {
			t.Errorf("buildCardImageMap() has no entry for the %s token", name)
		}
	}
}
