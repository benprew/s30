package game

import "testing"

func TestLoadingDotsAdvanceOncePerSecond(t *testing.T) {
	loading := &LoadingGame{}

	for range loadingDotInterval - 1 {
		if err := loading.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if loading.dots != 0 {
		t.Fatalf("dots = %d before one second, want 0", loading.dots)
	}

	if err := loading.Update(); err != nil {
		t.Fatal(err)
	}
	if loading.dots != 1 {
		t.Fatalf("dots = %d after one second, want 1", loading.dots)
	}
}

func TestLoadingGameSaveBeforeInitializationIsSafe(t *testing.T) {
	loading := &LoadingGame{}
	if err := loading.SaveGame(); err != nil {
		t.Fatalf("SaveGame before initialization: %v", err)
	}
}
