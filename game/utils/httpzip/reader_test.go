package httpzip

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func createTestZip() []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	file1, _ := w.Create("test1.txt")
	file1.Write([]byte("Hello, World!"))

	file2, _ := w.CreateHeader(&zip.FileHeader{
		Name:   "test2.txt",
		Method: zip.Deflate,
	})
	file2.Write([]byte("This is a deflated file with some content to compress."))

	file3, _ := w.Create("dir/test3.txt")
	file3.Write([]byte("Nested file"))

	w.Close()
	return buf.Bytes()
}

func TestNewReader(t *testing.T) {
	zipData := createTestZip()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(zipData)))
			return
		}

		http.ServeContent(w, r, "test.zip", time.Time{}, bytes.NewReader(zipData))
	}))
	defer server.Close()

	reader, err := NewReader(server.URL, nil)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	if len(reader.files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(reader.files))
	}
}

func TestReadFile(t *testing.T) {
	zipData := createTestZip()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(zipData)))
			return
		}

		http.ServeContent(w, r, "test.zip", time.Time{}, bytes.NewReader(zipData))
	}))
	defer server.Close()

	reader, err := NewReader(server.URL, nil)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	tests := []struct {
		name     string
		expected string
	}{
		{"test1.txt", "Hello, World!"},
		{"test2.txt", "This is a deflated file with some content to compress."},
		{"dir/test3.txt", "Nested file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := reader.ReadFile(tt.name)
			if err != nil {
				t.Fatalf("ReadFile(%s) failed: %v", tt.name, err)
			}
			if string(data) != tt.expected {
				t.Errorf("ReadFile(%s) = %q, want %q", tt.name, string(data), tt.expected)
			}
		})
	}
}

func TestFileOpen(t *testing.T) {
	zipData := createTestZip()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(zipData)))
			return
		}

		http.ServeContent(w, r, "test.zip", time.Time{}, bytes.NewReader(zipData))
	}))
	defer server.Close()

	reader, err := NewReader(server.URL, nil)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	for _, f := range reader.files {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("Open() failed for %s: %v", f.Name, err)
		}

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(rc)
		rc.Close()

		if err != nil {
			t.Fatalf("ReadFrom() failed for %s: %v", f.Name, err)
		}
	}
}

func TestFileNotFound(t *testing.T) {
	zipData := createTestZip()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(zipData)))
			return
		}

		http.ServeContent(w, r, "test.zip", time.Time{}, bytes.NewReader(zipData))
	}))
	defer server.Close()

	reader, err := NewReader(server.URL, nil)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	_, err = reader.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}
