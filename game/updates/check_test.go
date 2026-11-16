package updates

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCheckUsesSignedPlatformData(t *testing.T) {
	r, key := signedRelease(t, "windows/amd64", []byte("game"), []byte("helper"))
	oldKey := PublicKey
	PublicKey = key
	t.Cleanup(func() { PublicKey = oldKey })
	r.Downloads["windows/amd64"] = "https://example.com/unsigned"
	client := &http.Client{Transport: updateTransport(func(_ *http.Request) (*http.Response, error) {
		data, err := json.Marshal(r)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data)))}, err
	})}
	got, err := Check(context.Background(), client, DefaultURL, "v1.0.0", "windows/amd64")
	if err != nil || got == nil || !got.CanInstall() || got.DownloadURL != testArtifact("s30", []byte("game")).URL {
		t.Fatalf("signed check = %+v, %v", got, err)
	}
	r.Version = "v3.0.0"
	if _, err := Check(context.Background(), client, DefaultURL, "v1.0.0", "windows/amd64"); err == nil {
		t.Fatal("unsigned version replaced signed version")
	}
}

func TestCheckUsesSignedMacDownload(t *testing.T) {
	r, key := signedRelease(t, "darwin/arm64", []byte("game"), nil)
	oldKey := PublicKey
	PublicKey = key
	t.Cleanup(func() { PublicKey = oldKey })
	r.Downloads["darwin/arm64"] = "https://github.com/other/game/releases/download/v2.0.0/game"
	client := &http.Client{Transport: updateTransport(func(_ *http.Request) (*http.Response, error) {
		data, err := json.Marshal(r)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data)))}, err
	})}
	got, err := Check(context.Background(), client, DefaultURL, "v1.0.0", "darwin/arm64")
	if err != nil || got == nil || got.DownloadURL != testArtifact("s30", []byte("game")).URL || got.CanInstall() {
		t.Fatalf("signed Mac check = %+v, %v", got, err)
	}
	r.Manifest = append(r.Manifest, ' ')
	if _, err := Check(context.Background(), client, DefaultURL, "v1.0.0", "darwin/arm64"); err == nil {
		t.Fatal("invalid signature was accepted on macOS")
	}
}

func TestAndroidVersionCodeMustIncrease(t *testing.T) {
	r, key := signedRelease(t, "android", []byte("apk"), nil)
	ConfigureAndroid("", 42, nil)
	t.Cleanup(func() { ConfigureAndroid("", 0, nil) })
	if _, err := r.verifiedPackage(key, "android"); err == nil {
		t.Fatal("same Android version code was accepted")
	}
	ConfigureAndroid("", 41, nil)
	if _, err := r.verifiedPackage(key, "android"); err != nil {
		t.Fatal(err)
	}
}

func TestCheck(t *testing.T) {
	for _, tc := range []struct {
		name, current, body string
		status              int
		want                bool
		wantErr             bool
	}{
		{"newer", "v1.9.0", `{"version":"v1.10.0","downloads":{"darwin/arm64":"https://github.com/benprew/s30/releases/download/v1.10.0/s30_mac-apple-silicon"}}`, 200, true, false},
		{"equal", "1.10.0", `{"version":"v1.10.0"}`, 200, false, false},
		{"older", "1.11.0", `{"version":"v1.10.0"}`, 200, false, false},
		{"prerelease build", "1.10.0-rc.1", `{"version":"v1.10.0","downloads":{"darwin/arm64":"https://github.com/benprew/s30/releases/download/v1.10.0/game"}}`, 200, true, false},
		{"no platform", "1.0.0", `{"version":"v1.10.0","downloads":{"windows/amd64":"https://github.com/game"}}`, 200, false, false},
		{"unsafe URL", "1.0.0", `{"version":"v1.10.0","downloads":{"darwin/arm64":"file:///tmp/game"}}`, 200, false, true},
		{"other repository", "1.0.0", `{"version":"v1.10.0","downloads":{"darwin/arm64":"https://github.com/other/s30/releases/download/v1.10.0/game"}}`, 200, false, true},
		{"other version", "1.0.0", `{"version":"v1.10.0","downloads":{"darwin/arm64":"https://github.com/benprew/s30/releases/download/v1.0.0/game"}}`, 200, false, true},
		{"encoded path", "1.0.0", `{"version":"v1.10.0","downloads":{"darwin/arm64":"https://github.com/benprew/s30/releases/download/v1.10.0/%2e%2e"}}`, 200, false, true},
		{"query", "1.0.0", `{"version":"v1.10.0","downloads":{"darwin/arm64":"https://github.com/benprew/s30/releases/download/v1.10.0/game?download=1"}}`, 200, false, true},
		{"invalid version", "1.0.0", `{"version":"latest"}`, 200, false, true},
		{"invalid JSON", "1.0.0", `{`, 200, false, true},
		{"server error", "1.0.0", `{}`, 503, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: updateTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Errorf("method = %s", r.Method)
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}
			got, err := Check(context.Background(), client, DefaultURL, tc.current, "darwin/arm64")
			if (err != nil) != tc.wantErr || (got != nil) != tc.want {
				t.Fatalf("Check = %+v, %v", got, err)
			}
		})
	}
}

func TestDevelopmentBuildDoesNotCheck(t *testing.T) {
	for _, current := range []string{"dev", "v1.2.3-dirty", "v1.2.3-2-gabcd123", "v1.2.3-2-gabcd123-dirty"} {
		got, err := Check(context.Background(), nil, "invalid", current, "darwin/arm64")
		if got != nil || err != nil {
			t.Fatalf("Check(%q) = %+v, %v", current, got, err)
		}
	}
}
