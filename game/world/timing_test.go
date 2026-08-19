package world

import "testing"

func TestEnemySpawnTimingPreservesTenTPSCadence(t *testing.T) {
	tests := []struct {
		name                  string
		totalTicks            int
		ticksSinceInteraction int
		want                  bool
	}{
		{name: "before grace period", totalTicks: 360, ticksSinceInteraction: 360},
		{name: "grace elapsed between checks", totalTicks: 479, ticksSinceInteraction: 479},
		{name: "first check after grace", totalTicks: 480, ticksSinceInteraction: 480, want: true},
		{name: "later check", totalTicks: 600, ticksSinceInteraction: 600, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldSpawnEnemy(test.totalTicks, test.ticksSinceInteraction); got != test.want {
				t.Fatalf("shouldSpawnEnemy(%d, %d) = %v, want %v", test.totalTicks, test.ticksSinceInteraction, got, test.want)
			}
		})
	}
}

func TestRandomEncounterSpawnTimingPreservesTenTPSCadence(t *testing.T) {
	if EncounterSpawnRate != 600 {
		t.Fatalf("EncounterSpawnRate = %d, want 600", EncounterSpawnRate)
	}
}
