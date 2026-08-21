package bugreport

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestHandleCrashOutputsDiagnostics(t *testing.T) {
	tempDir := t.TempDir()
	origLocal := DefaultBugReportDir
	_ = origLocal

	submitter := NewLocalFileSubmitter(tempDir, nil)
	report := CollectCrashReport(nil, "World", nil, "test panic", []byte("goroutine 1 [running]"))

	res, err := submitter.Submit(report)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if !strings.Contains(res.LocalFilePath, "crash_") {
		t.Errorf("Expected crash prefix in filename: %s", res.LocalFilePath)
	}

	content, err := os.ReadFile(res.LocalFilePath)
	if err != nil {
		t.Fatalf("Failed to read crash file: %v", err)
	}

	if !bytes.Contains(content, []byte("test panic")) {
		t.Errorf("Crash file content missing panic message")
	}
}
