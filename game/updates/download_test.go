package updates

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type updateTransport func(*http.Request) (*http.Response, error)

func (f updateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloadVerifiesBeforeKeepingFile(t *testing.T) {
	for _, body := range []string{"game", "evil", "gam", "game-extra"} {
		t.Run(body, func(t *testing.T) {
			client := &http.Client{Transport: updateTransport(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			file := filepath.Join(t.TempDir(), "update")
			err := download(context.Background(), client, testArtifact("s30", []byte("game")), "v2.0.0", file, nil)
			if body == "game" {
				data, readErr := os.ReadFile(file)
				if err != nil || readErr != nil || string(data) != body {
					t.Fatalf("download = %v, %q, %v", err, data, readErr)
				}
			} else {
				if err == nil {
					t.Fatal("corrupt download was accepted")
				}
				if _, err := os.Stat(file); !os.IsNotExist(err) {
					t.Fatal("partial download remains")
				}
			}
		})
	}
}

func TestDownloadRejectsUnsafeRedirect(t *testing.T) {
	for _, target := range []string{"http://github.com/file", "https://example.com/file", "https://github.com.evil.test/file"} {
		client := &http.Client{Transport: updateTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == target {
				t.Fatal("unsafe redirect was followed")
			}
			return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{"Location": {target}}}, nil
		})}
		if err := download(context.Background(), client, testArtifact("s30", []byte("game")), "v2.0.0", filepath.Join(t.TempDir(), "update"), nil); err == nil {
			t.Fatal("unsafe redirect was accepted")
		}
	}
}

func TestDownloadReportsProgressByPercent(t *testing.T) {
	writer := &progressWriter{total: 10_000}
	var reports int
	var last int64
	writer.report = func(received, total int64) {
		reports++
		if received < last || total != writer.total {
			t.Fatalf("progress = %d/%d after %d", received, total, last)
		}
		last = received
	}
	for range writer.total {
		_, _ = writer.Write([]byte{0})
	}
	if reports > 101 || last != writer.total {
		t.Fatalf("progress reports = %d, last = %d", reports, last)
	}
}
