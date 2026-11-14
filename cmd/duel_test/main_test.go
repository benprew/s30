package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
)

func TestMain(m *testing.M) {
	if os.Getenv("S30_DUEL_MEMORY_CHILD") != "1" {
		os.Exit(m.Run())
	}
	os.Args = append([]string{os.Args[0]}, os.Args[indexOfSeparator(os.Args)+1:]...)
	img, _, err := image.Decode(bytes.NewReader(assets.CardBlank_png))
	if err != nil {
		panic(err)
	}
	// Fixed images keep network and artwork changes out of the measurements.
	for card := range xTestDeck() {
		domain.CacheCardImage(card.DefaultArtwork().ID, img)
	}
	main()
}

func indexOfSeparator(args []string) int {
	for i, arg := range args {
		if arg == "--" {
			return i
		}
	}
	panic("missing child argument separator")
}

func TestDuelMemoryRegression(t *testing.T) {
	if os.Getenv("S30_DUEL_MEMORY_TEST") != "1" {
		t.Skip("set S30_DUEL_MEMORY_TEST=1 to run graphical AI duel memory checks")
	}
	report := filepath.Join(t.TempDir(), "memory.json")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^$", "--",
		"-headless", "-autoplay", "-fast", "-duels=4", "-rogue=Goblin Warlord", "-ai-test-deck",
		"-load-card-images=false", "-memstatframes=0", "-memory-report="+report)
	cmd.Env = append(os.Environ(), "S30_DUEL_MEMORY_CHILD=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("duel process: %v\n%s", err, output)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v\n%s", err, output)
	}
	var samples []duelMemorySample
	if err := json.Unmarshal(data, &samples); err != nil {
		t.Fatal(err)
	}
	if len(samples) != 4 {
		t.Fatalf("completed %d duels, want 4\n%s", len(samples), output)
	}
	baseline := samples[0]
	for _, sample := range samples[1:] {
		t.Logf("duel %d: allocated=%d allocations=%d heap=%d goroutines=%d", sample.Duel, sample.AllocatedBytes, sample.Allocations, sample.HeapBytes, sample.Goroutines)
		if sample.AllocatedBytes > 256<<20 {
			t.Errorf("duel %d allocated %d bytes, budget %d", sample.Duel, sample.AllocatedBytes, 256<<20)
		}
		if sample.Allocations > 2_000_000 {
			t.Errorf("duel %d used %d allocations, budget 2000000", sample.Duel, sample.Allocations)
		}
		if sample.HeapBytes > baseline.HeapBytes+(16<<20) {
			t.Errorf("duel %d retained heap grew by %d bytes, budget %d", sample.Duel, sample.HeapBytes-baseline.HeapBytes, 16<<20)
		}
		if sample.Goroutines > baseline.Goroutines+4 {
			t.Errorf("duel %d retained goroutines: %d, baseline %d", sample.Duel, sample.Goroutines, baseline.Goroutines)
		}
	}
}
