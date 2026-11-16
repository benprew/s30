package updates

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type androidInstallerStub struct{ calls int }

func (i *androidInstallerStub) InstallUpdate(string) error { i.calls++; return nil }
func (i *androidInstallerStub) OpenDownload(string) error  { return nil }

func TestAndroidRechecksAPKBeforeOpeningInstaller(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s30.apk")
	if err := os.WriteFile(path, []byte("apk"), 0600); err != nil {
		t.Fatal(err)
	}
	installer := &androidInstallerStub{}
	p := &Prepared{payload: path, artifact: testArtifact("s30.apk", []byte("apk")), installer: installer}
	if restart, err := p.Install(); err != nil || restart || installer.calls != 1 {
		t.Fatalf("install = %v, %v, calls = %d", restart, err, installer.calls)
	}
	if err := os.WriteFile(path, []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Install(); err == nil || installer.calls != 1 {
		t.Fatal("changed APK was sent to the installer")
	}
}

func TestAndroidRemovesAlreadyInstalledDownloads(t *testing.T) {
	cache := t.TempDir()
	for _, entry := range []struct{ name, code string }{
		{".s30-update-old", "41"}, {".s30-update-current", "42"}, {".s30-update-new", "43"}, {"other", "41"},
	} {
		dir := filepath.Join(cache, "updates", entry.name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "version-code"), []byte(entry.code), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cleanupAndroidUpdates(cache, 42)
	entries, err := os.ReadDir(filepath.Join(cache, "updates"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("remaining downloads = %v, %v", entries, err)
	}
	for _, name := range []string{".s30-update-new", "other"} {
		if _, err := os.Stat(filepath.Join(cache, "updates", name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPrepareDesktopKeepsInstalledGame(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "verified", true: "corrupt helper"}[corrupt], func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "game")
			if err := os.WriteFile(target, []byte("old game"), 0755); err != nil {
				t.Fatal(err)
			}
			r, key := signedRelease(t, "linux/amd64", []byte("new game"), []byte("helper"))
			r.Platform = "linux/amd64"
			var err error
			r.pkg, err = r.verifiedPackage(key, r.Platform)
			if err != nil {
				t.Fatal(err)
			}
			client := &http.Client{Transport: updateTransport(func(req *http.Request) (*http.Response, error) {
				body := "new game"
				if strings.HasSuffix(req.URL.Path, "s30-updater") {
					body = "helper"
					if corrupt {
						body = "unsafe"
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			var received, total int64
			progress := func(next, size int64) {
				if next < received || (total != 0 && size != total) {
					t.Errorf("progress changed from %d/%d to %d/%d", received, total, next, size)
				}
				received, total = next, size
			}
			p, err := prepareDesktop(context.Background(), client, r, target, progress)
			if (err != nil) != corrupt {
				t.Fatalf("prepare = %+v, %v", p, err)
			}
			if !corrupt && (received != total || total != int64(len("new gamehelper"))) {
				t.Errorf("final progress = %d/%d", received, total)
			}
			data, _ := os.ReadFile(target)
			if string(data) != "old game" {
				t.Fatal("installed game changed during download")
			}
			if p != nil {
				p.Cleanup()
			}
			entries, _ := os.ReadDir(filepath.Dir(target))
			if len(entries) != 1 {
				t.Fatalf("staged files remain: %v", entries)
			}
		})
	}
}

func TestFailedReplacementRestartsUnchangedGame(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory modes do not block replacement on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "game")
	stage := filepath.Join(dir, ".s30-update-test")
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(stage, "s30")
	if err := os.WriteFile(payload, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
	starts := 0
	err := replaceAndRestart(target, payload, func() error {
		starts++
		data, readErr := os.ReadFile(target)
		if readErr != nil || string(data) != "old" {
			t.Fatalf("original game = %q, %v", data, readErr)
		}
		return nil
	})
	if err == nil || starts != 1 {
		t.Fatalf("failed replacement = %v, starts = %d", err, starts)
	}
	if _, err := os.Stat(filepath.Join(stage, "previous")); !os.IsNotExist(err) {
		t.Fatalf("unneeded backup remains: %v", err)
	}
}

func TestReplaceAndRestartRollsBack(t *testing.T) {
	for _, failStart := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "rollback"}[failStart], func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "game")
			stage := filepath.Join(dir, ".s30-update-test")
			if err := os.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
				t.Fatal(err)
			}
			payload := filepath.Join(stage, "s30")
			if err := os.WriteFile(payload, []byte("new"), 0755); err != nil {
				t.Fatal(err)
			}
			starts := 0
			err := replaceAndRestart(target, payload, func() error {
				starts++
				if failStart && starts == 1 {
					return os.ErrPermission
				}
				return nil
			})
			data, _ := os.ReadFile(target)
			want := "new"
			if failStart {
				want = "old"
			}
			if string(data) != want || (err != nil) != failStart {
				t.Fatalf("target = %q, err = %v", data, err)
			}
			if failStart && starts != 2 {
				t.Fatal("old game was not restarted")
			}
		})
	}
}

func TestLinuxManagedLocations(t *testing.T) {
	for _, target := range []string{"/usr/bin/s30", "/usr/local/bin/s30", "/snap/s30/game", "/app/bin/s30"} {
		if desktopLocationAllowed("linux/amd64", target) {
			t.Fatalf("managed location allowed: %s", target)
		}
	}
	if !desktopLocationAllowed("linux/amd64", "/home/player/games/s30") {
		t.Fatal("standalone game was rejected")
	}
}
