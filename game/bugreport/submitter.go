package bugreport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// EnvWorkerURL is the environment variable used to configure the worker endpoint.
const EnvWorkerURL = "S30_BUG_WORKER_URL"

// DefaultWorkerURL is the default GitHub issues worker endpoint URL.
var DefaultWorkerURL = ""

// SubmitResult summarizes the outcome of filing a bug or crash report.
type SubmitResult struct {
	Success       bool   `json:"success"`
	IssueURL      string `json:"issue_url,omitempty"`
	IssueNumber   int    `json:"issue_number,omitempty"`
	LocalFilePath string `json:"local_file_path,omitempty"`
	Message       string `json:"message"`
}

// Submitter delivers bug and crash reports.
type Submitter interface {
	Submit(report *BugReport) (*SubmitResult, error)
}

// WorkerSubmitter delivers reports via an HTTP worker endpoint that proxies to GitHub Issues.
type WorkerSubmitter struct {
	WorkerURL string
	Client    *http.Client
}

// NewWorkerSubmitter creates a Submitter for the given worker URL.
func NewWorkerSubmitter(workerURL string) *WorkerSubmitter {
	return &WorkerSubmitter{
		WorkerURL: workerURL,
		Client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *WorkerSubmitter) Submit(report *BugReport) (*SubmitResult, error) {
	if s.WorkerURL == "" {
		return nil, fmt.Errorf("worker URL not configured")
	}

	payload := map[string]any{
		"title":    report.IssueTitle(),
		"body":     report.ToMarkdown(),
		"is_crash": report.IsCrash,
		"report":   report,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal worker payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.WorkerURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create worker request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "s30-game-client/"+GameVersion)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send report to worker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("worker returned HTTP status %d", resp.StatusCode)
	}

	var result SubmitResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse worker response: %w", err)
	}

	return &result, nil
}

// LocalFileSubmitter saves reports to local disk and copies markdown to clipboard.
type LocalFileSubmitter struct {
	BaseDir     string
	ClipboardFn func(string) error
}

// NewLocalFileSubmitter creates a LocalFileSubmitter with a specific base directory.
func NewLocalFileSubmitter(baseDir string, clipboardFn func(string) error) *LocalFileSubmitter {
	return &LocalFileSubmitter{
		BaseDir:     baseDir,
		ClipboardFn: clipboardFn,
	}
}

// DefaultBugReportDir returns the user directory where bug reports and crashes are stored.
func DefaultBugReportDir(isCrash bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	subdir := "bug_reports"
	if isCrash {
		subdir = "crashes"
	}
	return filepath.Join(homeDir, ".s30", subdir), nil
}

func (s *LocalFileSubmitter) Submit(report *BugReport) (*SubmitResult, error) {
	dir := s.BaseDir
	if dir == "" {
		defaultDir, err := DefaultBugReportDir(report.IsCrash)
		if err != nil {
			return nil, fmt.Errorf("could not determine report directory: %w", err)
		}
		dir = defaultDir
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create report directory: %w", err)
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.json", report.ID))
	jsonData, err := report.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize report: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write report file: %w", err)
	}

	mdContent := report.ToMarkdown()
	if s.ClipboardFn != nil {
		_ = s.ClipboardFn(mdContent)
	} else {
		_ = CopyToClipboard(mdContent)
	}

	return &SubmitResult{
		Success:       true,
		LocalFilePath: filePath,
		Message:       fmt.Sprintf("Saved report to %s", filePath),
	}, nil
}

// CompositeSubmitter saves locally and sends to the worker if configured.
type CompositeSubmitter struct {
	Worker *WorkerSubmitter
	Local  *LocalFileSubmitter
}

// NewDefaultSubmitter creates the standard submitter with configured worker and local fallback.
func NewDefaultSubmitter() *CompositeSubmitter {
	workerURL := os.Getenv(EnvWorkerURL)
	if workerURL == "" {
		workerURL = DefaultWorkerURL
	}

	var worker *WorkerSubmitter
	if workerURL != "" {
		worker = NewWorkerSubmitter(workerURL)
	}

	return &CompositeSubmitter{
		Worker: worker,
		Local:  NewLocalFileSubmitter("", nil),
	}
}

func (c *CompositeSubmitter) Submit(report *BugReport) (*SubmitResult, error) {
	localRes, localErr := c.Local.Submit(report)
	if localErr != nil {
		return nil, fmt.Errorf("failed to save report locally: %w", localErr)
	}

	if c.Worker == nil || c.Worker.WorkerURL == "" {
		return localRes, nil
	}

	workerRes, workerErr := c.Worker.Submit(report)
	if workerErr != nil {
		localRes.Message = fmt.Sprintf("Saved locally to %s (worker error: %v)", localRes.LocalFilePath, workerErr)
		return localRes, nil
	}

	workerRes.LocalFilePath = localRes.LocalFilePath
	return workerRes, nil
}
