package main

import (
	"os"
	"strings"
	"testing"
)

func TestBrowserLifecycleTriggersSaves(t *testing.T) {
	contents, err := os.ReadFile("save_lifecycle_js.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(contents)
	for _, required := range []string{"visibilitychange", "pagehide", "g.SaveGame()"} {
		if !strings.Contains(source, required) {
			t.Errorf("browser save lifecycle is missing %q", required)
		}
	}
}
