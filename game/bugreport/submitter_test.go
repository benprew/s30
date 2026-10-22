package bugreport

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDefaultSubmitterUsesS30API(t *testing.T) {
	t.Setenv(EnvWorkerURL, "")
	s := NewDefaultSubmitter()
	if s.Worker == nil {
		t.Fatal("default submitter has no API endpoint")
	}
	if got := s.Worker.WorkerURL; got != "https://s30.throwingbones.com/api/bug-reports" {
		t.Fatalf("endpoint = %q", got)
	}
}

func TestSubmitterSendsReportToAPI(t *testing.T) {
	report := &BugReport{ID: "test-report", UserNotes: "Test notes", ActiveScreen: "World"}
	var received bool
	var uploaded []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/bug-reports" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()
		var metadata struct {
			Title   string `json:"title"`
			Body    string `json:"body"`
			IsCrash bool   `json:"is_crash"`
		}
		if err := json.Unmarshal([]byte(r.FormValue("metadata")), &metadata); err != nil {
			t.Error(err)
		}
		if metadata.Title != report.IssueTitle() || metadata.Body != report.ToSummaryMarkdown() || metadata.IsCrash {
			t.Errorf("unexpected metadata: %+v", metadata)
		}
		if strings.Contains(metadata.Body, "Full State JSON") {
			t.Error("summary duplicates full report")
		}
		file, header, err := r.FormFile("report")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		if header.Filename != "test-report.json.gz" {
			t.Errorf("filename = %q", header.Filename)
		}
		compressed, err := io.ReadAll(file)
		uploaded = compressed
		if err != nil {
			t.Error(err)
			return
		}
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			t.Error(err)
			return
		}
		defer reader.Close()
		contents, err := io.ReadAll(reader)
		if err != nil {
			t.Error(err)
			return
		}
		want, err := report.ToJSON()
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(contents, want) {
			t.Error("upload differs from compact report JSON")
		}

		_ = json.NewEncoder(w).Encode(SubmitResult{Success: true, IssueURL: "https://github.test/issues/42", IssueNumber: 42})
	}))
	defer server.Close()
	t.Setenv(EnvWorkerURL, server.URL+"/api/bug-reports")
	s := NewDefaultSubmitter()
	s.Local = NewLocalFileSubmitter(t.TempDir(), func(string) error { return nil })
	result, err := s.Submit(report)
	if err != nil {
		t.Fatal(err)
	}
	if !received || !result.Success || result.IssueNumber != 42 || result.IssueURL != "https://github.test/issues/42" {
		t.Fatalf("unexpected result: %+v, received = %v", result, received)
	}
	backup, err := os.ReadFile(result.LocalFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(result.LocalFilePath, ".json.gz") || !bytes.Equal(backup, uploaded) {
		t.Fatal("backup and uploaded gzip file differ")
	}

}

func TestSubmitterReturnsAPIErrorWithLocalBackup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	s := &CompositeSubmitter{
		Worker: NewWorkerSubmitter(server.URL),
		Local:  NewLocalFileSubmitter(t.TempDir(), func(string) error { return nil }),
	}
	result, err := s.Submit(&BugReport{ID: "failed-report", UserNotes: "Test notes"})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error = %v, want API failure", err)
	}
	if result == nil {
		t.Fatal("local backup result is missing")
	}
	if _, statErr := os.Stat(result.LocalFilePath); statErr != nil {
		t.Fatalf("local backup: %v", statErr)
	}
	if !strings.Contains(err.Error(), result.LocalFilePath) {
		t.Fatalf("error omits backup path: %v", err)
	}
}

func readCompressedReport(t *testing.T, path string) []byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
