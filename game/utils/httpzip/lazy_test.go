package httpzip

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func createLargeTestZip() []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for i := 0; i < 50; i++ {
		file, _ := w.Create("file" + strconv.Itoa(i) + ".txt")
		file.Write([]byte("Content for file " + strconv.Itoa(i)))
	}

	bigFile, _ := w.Create("bigfile.txt")
	bigData := make([]byte, 100*1024)
	for i := range bigData {
		bigData[i] = byte('A' + (i % 26))
	}
	bigFile.Write(bigData)

	w.Close()
	return buf.Bytes()
}

func TestLazyLoading(t *testing.T) {
	zipData := createLargeTestZip()
	var requestCount atomic.Int64
	var headCount atomic.Int64
	var getCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		if r.Method == http.MethodHead {
			headCount.Add(1)
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(zipData)))
			return
		}

		if r.Method == http.MethodGet {
			getCount.Add(1)
		}

		http.ServeContent(w, r, "test.zip", time.Time{}, bytes.NewReader(zipData))
	}))
	defer server.Close()

	t.Logf("Created test zip with %d bytes", len(zipData))

	requestCount.Store(0)
	reader, err := NewReader(server.URL, nil)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	initRequests := requestCount.Load()
	t.Logf("NewReader made %d requests (HEAD: %d, GET: %d)", initRequests, headCount.Load(), getCount.Load())

	if initRequests > 10 {
		t.Errorf("NewReader made too many requests: %d (expected < 10)", initRequests)
	}

	if len(reader.files) != 51 {
		t.Errorf("Expected 51 files, got %d", len(reader.files))
	}

	beforeOpen := requestCount.Load()
	data, err := reader.ReadFile("file0.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	afterFirstOpen := requestCount.Load()
	firstOpenRequests := afterFirstOpen - beforeOpen

	t.Logf("First open of file0.txt made %d requests", firstOpenRequests)
	t.Logf("Content: %s", string(data))

	if firstOpenRequests > 5 {
		t.Errorf("First open made too many requests: %d (expected < 5)", firstOpenRequests)
	}

	beforeSecondOpen := requestCount.Load()
	data2, err := reader.ReadFile("file0.txt")
	if err != nil {
		t.Fatalf("Second ReadFile failed: %v", err)
	}
	afterSecondOpen := requestCount.Load()
	secondOpenRequests := afterSecondOpen - beforeSecondOpen

	t.Logf("Second open of file0.txt made %d requests", secondOpenRequests)

	if string(data) != string(data2) {
		t.Errorf("Data mismatch between first and second read")
	}

	if secondOpenRequests > firstOpenRequests {
		t.Errorf("Second open should not make more requests than first open (got %d vs %d)", secondOpenRequests, firstOpenRequests)
	}

	beforeBatch := requestCount.Load()
	for i := 1; i < 10; i++ {
		_, err := reader.ReadFile("file" + strconv.Itoa(i) + ".txt")
		if err != nil {
			t.Fatalf("ReadFile file%d failed: %v", i, err)
		}
	}
	afterBatch := requestCount.Load()
	batchRequests := afterBatch - beforeBatch

	t.Logf("Opening 9 additional files made %d requests (avg %.1f per file)", batchRequests, float64(batchRequests)/9.0)

	totalRequests := requestCount.Load()
	t.Logf("Total requests for entire test: %d", totalRequests)

	if totalRequests > 100 {
		t.Errorf("Total requests exceeded reasonable limit: %d", totalRequests)
	}

	beforeBig := requestCount.Load()
	bigData, err := reader.ReadFile("bigfile.txt")
	if err != nil {
		t.Fatalf("ReadFile bigfile failed: %v", err)
	}
	afterBig := requestCount.Load()
	bigFileRequests := afterBig - beforeBig

	t.Logf("Reading 100KB file made %d requests", bigFileRequests)
	t.Logf("Big file size: %d bytes", len(bigData))

	if bigFileRequests > 5 {
		t.Errorf("Reading 100KB file made too many requests: %d (should be 1-2)", bigFileRequests)
	}
}
