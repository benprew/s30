package save

import "testing"

func TestDeserializeSaveNormalizesLegacyMovementSpeedWithoutChangingVersion(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"world": {
			"Player": {"MoveSpeed": 10},
			"Enemies": [{"MoveSpeed": 6}]
		}
	}`)

	got, err := deserializeSave(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 {
		t.Fatalf("Version = %d, want 1", got.Version)
	}
	if got.World.Player.MoveSpeed != 10.0/6.0 {
		t.Errorf("player MoveSpeed = %v, want %v", got.World.Player.MoveSpeed, 10.0/6.0)
	}
	if got.World.Enemies[0].MoveSpeed != 1 {
		t.Errorf("enemy MoveSpeed = %v, want 1", got.World.Enemies[0].MoveSpeed)
	}
}

func TestDeserializeSaveLeavesNewMovementSpeedUnchanged(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"world": {
			"Player": {"MoveSpeed": 1.6666666666666667},
			"Enemies": [{"MoveSpeed": 1.8333333333333333}, {"MoveSpeed": 12}]
		}
	}`)

	got, err := deserializeSave(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.World.Player.MoveSpeed != 1.6666666666666667 {
		t.Errorf("player MoveSpeed = %v, want unchanged", got.World.Player.MoveSpeed)
	}
	if got.World.Enemies[0].MoveSpeed != 1.8333333333333333 {
		t.Errorf("enemy MoveSpeed = %v, want unchanged", got.World.Enemies[0].MoveSpeed)
	}
	if got.World.Enemies[1].MoveSpeed != 12 {
		t.Errorf("out-of-range enemy MoveSpeed = %v, want unchanged", got.World.Enemies[1].MoveSpeed)
	}
}
