package screenui

import "testing"

func TestSentinelsAreDistinct(t *testing.T) {
	if PopScr == NoScr {
		t.Errorf("PopScr and NoScr must be distinct")
	}
	// Sentinels must not collide with real screen names.
	for _, s := range []ScreenName{StartScr, WorldScr, MiniMapScr, QuestRewardScr} {
		if s == PopScr || s == NoScr {
			t.Errorf("sentinel collides with real screen %d", s)
		}
	}
}
