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
