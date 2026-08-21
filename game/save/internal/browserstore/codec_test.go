package browserstore

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	data := bytes.Repeat([]byte(`{"tiles":[{"terrain":"forest"}],"enemies":[]}`), 1000)

	encoded, err := Encode(data)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(encoded) >= len(data) {
		t.Fatalf("encoded save is not smaller: got %d bytes, raw is %d", len(encoded), len(data))
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatal("decoded save differs from original")
	}
}

func TestDecodeAcceptsLegacyPlainJSON(t *testing.T) {
	legacy := `{"name":"legacy","version":1}`
	decoded, err := Decode(legacy)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(decoded) != legacy {
		t.Fatalf("Decode = %q, want %q", decoded, legacy)
	}
}
