package updates

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func lockUpdateFile(t *testing.T, path string) func() {
	t.Helper()
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	unlocked := false
	unlock := func() {
		if !unlocked {
			_ = windows.CloseHandle(handle)
			unlocked = true
		}
	}
	t.Cleanup(unlock)
	return unlock
}

func TestWindowsReplacementRetriesSharingViolation(t *testing.T) {
	dir := t.TempDir()
	target, payload := filepath.Join(dir, "game.exe"), filepath.Join(dir, "new.exe")
	for path, data := range map[string]string{target: "old", payload: "new"} {
		if err := os.WriteFile(path, []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	unlock := lockUpdateFile(t, target)
	done := make(chan error, 1)
	go func() { done <- replaceExecutable(payload, target) }()
	select {
	case err := <-done:
		t.Fatalf("replacement stopped while the file was locked: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	unlock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("replacement did not finish after the lock was removed")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "new" {
		t.Fatalf("installed file = %q, %v", data, err)
	}
}

func TestWindowsCleanupRetainsMarkerUntilHelperExits(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".s30-update-completed")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"complete", "helper.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0700); err != nil {
			t.Fatal(err)
		}
	}
	unlock := lockUpdateFile(t, filepath.Join(dir, "helper.exe"))
	cleanupDesktopUpdates(root)
	if _, err := os.Stat(filepath.Join(dir, "complete")); err != nil {
		t.Fatalf("completion marker was lost while the helper was locked: %v", err)
	}
	unlock()
	cleanupDesktopUpdates(root)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("completed update remains after the helper exits: %v", err)
	}
}
