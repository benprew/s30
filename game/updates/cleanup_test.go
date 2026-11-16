package updates

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benprew/s30/game/version"
)

func TestDesktopCleanupKeepsActiveUpdatesAndRecoveryFiles(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name, marker  string
		stale, remove bool
	}{
		{".s30-update-completed", "complete", false, true},
		{".s30-update-interrupted", "", true, true},
		{".s30-update-active", "", false, false},
		{".s30-update-failed", "update.log", true, false},
		{".s30-update-recovery", "previous", true, false},
		{"other", "complete", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(root, tc.name)
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if tc.marker != "" {
				if err := os.WriteFile(filepath.Join(dir, tc.marker), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.stale {
				old := time.Now().Add(-72 * time.Hour)
				if err := os.Chtimes(dir, old, old); err != nil {
					t.Fatal(err)
				}
			}
			cleanupDesktopUpdates(root)
			_, err := os.Stat(dir)
			if os.IsNotExist(err) != tc.remove {
				t.Fatalf("cleanup %s: %v, remove = %v", dir, err, tc.remove)
			}
		})
	}
}

func TestAndroidStartupCleansDownloadsInBackground(t *testing.T) {
	cache := t.TempDir()
	dir := filepath.Join(cache, "updates", ".s30-update-installed")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "version-code"), []byte("42"), 0600); err != nil {
		t.Fatal(err)
	}
	ConfigureAndroid(cache, 42, nil)
	t.Cleanup(func() { ConfigureAndroid("", 0, nil) })
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("configuration removed the APK on its caller's thread: %v", err)
	}
	oldVersion := version.Current
	version.Current = "dev"
	t.Cleanup(func() { version.Current = oldVersion })
	for range Start() {
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("startup did not remove the installed APK: %v", err)
	}
}
