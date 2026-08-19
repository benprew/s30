package screens

import "testing"

func TestCityEntranceAnimationCompletes(t *testing.T) {
	animation := newCityEntranceAnimation()
	if animation.progress != 0 || animation.complete {
		t.Fatalf("new animation = progress %v, complete %v; want 0, false", animation.progress, animation.complete)
	}

	for range cityEntranceAnimationFrames - 1 {
		animation.update()
	}
	if animation.complete {
		t.Fatal("animation completed before its configured duration")
	}

	animation.update()
	if !animation.complete {
		t.Fatal("animation did not complete after its configured duration")
	}
	if animation.progress != 1 {
		t.Fatalf("completed progress = %v, want 1", animation.progress)
	}
}

func TestCityEntranceCellThresholdsAreStableAndVaried(t *testing.T) {
	first := cityEntranceCellThreshold(2, 3)
	if first != cityEntranceCellThreshold(2, 3) {
		t.Fatal("cell threshold changed for the same coordinates")
	}

	allSame := true
	for y := range 5 {
		for x := range 5 {
			if cityEntranceCellThreshold(x, y) != first {
				allSame = false
			}
		}
	}
	if allSame {
		t.Fatal("cell thresholds do not vary across the dissolve grid")
	}
}
