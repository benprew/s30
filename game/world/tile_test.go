package world

import (
	"encoding/json"
	"testing"
)

func TestTileUnmarshalJSONLegacyNonCityDropsPlaceholderCity(t *testing.T) {
	data := []byte(`{
		"IsCity": false,
		"City": {
			"Tier": 0,
			"Name": "",
			"X": 0,
			"Y": 0
		},
		"TerrainType": 1
	}`)

	var tile Tile
	if err := json.Unmarshal(data, &tile); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if tile.City != nil {
		t.Fatalf("City = %#v, want nil for legacy non-city tile", tile.City)
	}
}

func TestTileUnmarshalJSONLegacyCityKeepsCity(t *testing.T) {
	data := []byte(`{
		"IsCity": true,
		"City": {
			"Tier": 3,
			"Name": "Azar's Steading",
			"X": 26,
			"Y": 8
		},
		"TerrainType": 4
	}`)

	var tile Tile
	if err := json.Unmarshal(data, &tile); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if tile.City == nil {
		t.Fatal("City = nil, want legacy city preserved")
	}
	if tile.City.Name != "Azar's Steading" {
		t.Fatalf("City.Name = %q, want %q", tile.City.Name, "Azar's Steading")
	}
}
