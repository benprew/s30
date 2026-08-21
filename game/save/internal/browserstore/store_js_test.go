//go:build js

package browserstore

import (
	"syscall/js"
	"testing"
)

func TestStoreRoundTripEntriesAndRemove(t *testing.T) {
	js.Global().Call("eval", `
		globalThis.localStorage = (() => {
			const data = new Map();
			return {
				get length() { return data.size; },
				key(index) { return Array.from(data.keys())[index] ?? null; },
				getItem(key) { return data.has(key) ? data.get(key) : null; },
				setItem(key, value) { data.set(String(key), String(value)); },
				removeItem(key) { data.delete(key); }
			};
		})();
	`)

	store, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Set("s30.save.one", "first"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Set("unrelated", "ignored"); err != nil {
		t.Fatalf("Set unrelated: %v", err)
	}
	value, found, err := store.Get("s30.save.one")
	if err != nil || !found || value != "first" {
		t.Fatalf("Get = %q, %t, %v", value, found, err)
	}
	entries, err := store.Entries("s30.save.")
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Key != "s30.save.one" || entries[0].Value != "first" {
		t.Fatalf("Entries = %+v", entries)
	}
	if err := store.Remove("s30.save.one"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, found, err := store.Get("s30.save.one"); err != nil || found {
		t.Fatalf("removed Get found = %t, err = %v", found, err)
	}
}
