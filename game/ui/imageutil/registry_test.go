package imageutil

import (
	"testing"
)

func TestCacheKey(t *testing.T) {
	data1 := []byte{1, 2, 3}
	data2 := []byte{4, 5, 6}

	key1a := cacheKey(data1, "test")
	key1b := cacheKey(data1, "test")
	key2 := cacheKey(data2, "test")
	key1diff := cacheKey(data1, "other")

	if key1a != key1b {
		t.Error("same data and extra should produce same key")
	}
	if key1a == key2 {
		t.Error("different data should produce different keys")
	}
	if key1a == key1diff {
		t.Error("different extra should produce different keys")
	}
}

func TestRegistryGetSet(t *testing.T) {
	ResetRegistry()

	val := "test_value"
	key := "test_key"

	got, ok := registryGet(key)
	if ok {
		t.Error("expected miss on empty registry")
	}
	if got != nil {
		t.Error("expected nil on miss")
	}

	registrySet(key, val)

	got, ok = registryGet(key)
	if !ok {
		t.Error("expected hit after set")
	}
	if got != val {
		t.Errorf("expected %v, got %v", val, got)
	}
}

func TestResetRegistry(t *testing.T) {
	registrySet("key", "val")
	ResetRegistry()

	_, ok := registryGet("key")
	if ok {
		t.Error("expected miss after reset")
	}
}

func TestRegistryStats(t *testing.T) {
	ResetRegistry()

	registrySet("a", 1)
	registrySet("b", 2)

	count := RegistryLen()
	if count != 2 {
		t.Errorf("expected 2 entries, got %d", count)
	}
}
