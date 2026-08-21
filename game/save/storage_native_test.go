//go:build !js

package save

import (
	"path/filepath"
	"testing"
)

func TestConfiguredSaveDir(t *testing.T) {
	configuredDir := filepath.Join(t.TempDir(), "android-files", "saves")
	SetSaveDir(configuredDir)
	t.Cleanup(func() { SetSaveDir("") })

	got, err := SaveDir()
	if err != nil {
		t.Fatalf("SaveDir: %v", err)
	}
	if got != configuredDir {
		t.Fatalf("SaveDir = %q, want %q", got, configuredDir)
	}
}
