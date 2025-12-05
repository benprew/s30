package httpzip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func createZipWithNFiles(n int) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for i := 0; i < n; i++ {
		file, _ := w.Create("file" + strconv.Itoa(i) + ".txt")
		file.Write([]byte("Content for file " + strconv.Itoa(i)))
	}

	w.Close()
	return buf.Bytes()
}

func TestScaling(t *testing.T) {
	tests := []struct {
		fileCount int
	}{
		{10},
		{100},
		{500},
		{1000},
		{1400},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.fileCount)+"files", func(t *testing.T) {
			zipData := createZipWithNFiles(tt.fileCount)
			var requestCount atomic.Int64
			var rangeBytes atomic.Int64

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount.Add(1)

				if r.Method == "HEAD" {
					w.Header().Set("Accept-Ranges", "bytes")
					w.Header().Set("Content-Length", strconv.Itoa(len(zipData)))
					return
				}

				rangeHeader := r.Header.Get("Range")
				if rangeHeader != "" {
					var start, end int
					_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
					rangeBytes.Add(int64(end - start + 1))
				}

				http.ServeContent(w, r, "test.zip", time.Time{}, bytes.NewReader(zipData))
			}))
			defer server.Close()

			requestCount.Store(0)
			reader, err := NewReader(server.URL, nil)
			if err != nil {
				t.Fatalf("NewReader failed: %v", err)
			}

			initRequests := requestCount.Load()
			totalBytes := rangeBytes.Load()

			t.Logf("%d files: NewReader made %d requests, fetched %d bytes (zip size: %d bytes)",
				tt.fileCount, initRequests, totalBytes, len(zipData))

			if len(reader.files) != tt.fileCount {
				t.Errorf("Expected %d files, got %d", tt.fileCount, len(reader.files))
			}

			beforeRead := requestCount.Load()
			_, err = reader.ReadFile("file0.txt")
			if err != nil {
				t.Fatalf("ReadFile failed: %v", err)
			}
			afterRead := requestCount.Load()
			readRequests := afterRead - beforeRead

			t.Logf("  Reading file0 made %d additional requests", readRequests)

			if tt.fileCount <= 100 {
				beforeAll := requestCount.Load()
				for i := 0; i < tt.fileCount; i++ {
					_, err = reader.ReadFile("file" + strconv.Itoa(i) + ".txt")
					if err != nil {
						t.Fatalf("ReadFile file%d failed: %v", i, err)
					}
				}
				afterAll := requestCount.Load()
				allRequests := afterAll - beforeAll
				t.Logf("  Reading all %d files made %d total requests (%.1f per file)",
					tt.fileCount, allRequests, float64(allRequests)/float64(tt.fileCount))
			}
		})
	}
}
