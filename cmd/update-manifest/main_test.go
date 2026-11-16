package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/s30/game/updates"
)

func TestSignReleaseBindsBuildArtifacts(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key := base64.StdEncoding.EncodeToString(public)
	t.Setenv("UPDATE_SIGNING_KEY", base64.StdEncoding.EncodeToString(private.Seed()))
	t.Setenv("UPDATE_PUBLIC_KEY", key)
	root := t.TempDir()
	for _, entry := range releaseFiles {
		dir := filepath.Join(root, entry.game)
		if err = os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{entry.game, entry.helper} {
			if name == "" {
				continue
			}
			if err = os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err = os.WriteFile(filepath.Join(root, "s30_android.apk", "s30_android.version"), []byte("1002003"), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "manifest.json")
	if err = signRelease("v1.2.3", root, output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := os.ReadFile(output + ".sig")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := updates.VerifyManifest(data, strings.TrimSpace(string(sig)), key)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Platforms) != 7 || manifest.Platforms["android"].VersionCode != 1002003 {
		t.Fatalf("manifest = %+v", manifest)
	}
	for _, platform := range []string{"linux/amd64", "linux/arm64", "windows/amd64", "windows/arm64"} {
		if manifest.Platforms[platform].Helper == nil {
			t.Fatalf("missing helper for %s", platform)
		}
	}
	t.Setenv("UPDATE_PUBLIC_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err = signRelease("v1.2.3", root, filepath.Join(root, "bad.json")); err == nil {
		t.Fatal("mismatched signing key was accepted")
	}
}
