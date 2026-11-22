package screens

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/benprew/s30/game/updates"
)

func TestBrowserDownloadErrorExplainsBrowserFailure(t *testing.T) {
	result := make(chan updateResult, 1)
	result <- updateResult{err: errors.New("browser failed")}
	s := &StartScreen{updateChecked: true, availableUpdate: &updates.Release{}, downloadResult: result}
	s.pollUpdate()
	if !strings.Contains(s.updateError, "open") || strings.Contains(s.updateError, "install") {
		t.Fatalf("browser failure = %q", s.updateError)
	}
}

func TestDismissedDownloadCannotStartInstallation(t *testing.T) {
	result := make(chan updateResult, 1)
	result <- updateResult{prepared: &updates.Prepared{}}
	s := &StartScreen{updateChecked: true, downloadResult: result}
	s.pollUpdate()
	if s.downloadResult != nil || s.restartAfterUpdate || s.preparedUpdate != nil {
		t.Fatal("dismissed download started installation")
	}
}

func TestDismissUpdateUnblocksMenuBeforeDownloadStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan updateResult)
	s := &StartScreen{
		updateChecked: true, availableUpdate: &updates.Release{},
		downloadResult: result, updateCancel: cancel, preparedUpdate: &updates.Prepared{},
	}
	s.dismissUpdate()
	if ctx.Err() == nil || s.availableUpdate != nil || s.downloadResult != nil || s.preparedUpdate != nil || s.updateCancel != nil {
		t.Fatal("dismissed update still blocks the menu or retains prepared files")
	}
	select {
	case result <- updateResult{prepared: &updates.Prepared{}}:
	case <-time.After(time.Second):
		t.Fatal("late download result was not collected")
	}
}

func TestCancelUpdateDownloadWaitsForCleanup(t *testing.T) {
	for _, tc := range []struct {
		name      string
		dismissed bool
	}{
		{"active download", false},
		{"dismissed download", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				result := make(chan updateResult, 1)
				s := &StartScreen{updateCancel: cancel, downloadResult: result}
				if tc.dismissed {
					s.dismissUpdate()
				}
				done := make(chan struct{})
				go func() {
					s.CancelUpdateDownload()
					close(done)
				}()
				synctest.Wait()
				if ctx.Err() != context.Canceled {
					t.Error("shutdown did not cancel the download")
				}
				select {
				case <-done:
					t.Error("shutdown returned before the download was cleaned up")
				default:
				}

				result <- updateResult{err: context.Canceled}
				<-done
				if len(result) != 0 {
					t.Error("shutdown did not collect the download result")
				}
			})
		})
	}
}

func TestCancelUpdateDownloadLeavesInstallerRunning(t *testing.T) {
	result := make(chan updateResult, 1)
	result <- updateResult{restart: true}
	prepared := &updates.Prepared{}
	s := &StartScreen{preparedUpdate: prepared, downloadResult: result}
	s.CancelUpdateDownload()
	if s.preparedUpdate != prepared || s.downloadResult != result || len(result) != 1 {
		t.Fatal("shutdown discarded files after handing them to the installer")
	}
}

func TestUpdateProgressKeepsLatestValue(t *testing.T) {
	progress := make(chan string, 1)
	reportUpdateProgress(progress, 1, 100)
	reportUpdateProgress(progress, 100, 100)
	s := &StartScreen{updateChecked: true, updateProgress: progress}
	s.pollUpdate()
	if s.updateStatus != "Downloading update... 100%" {
		t.Fatalf("progress = %q", s.updateStatus)
	}
}

func TestUpdateRestartRequiresSuccessfulInstallation(t *testing.T) {
	for _, failure := range []bool{false, true} {
		result := make(chan updateResult, 1)
		r := updateResult{restart: true}
		if failure {
			r.err = errors.New("helper failed")
		}
		result <- r
		s := &StartScreen{updateChecked: true, availableUpdate: &updates.Release{}, downloadResult: result}
		s.pollUpdate()
		if s.restartAfterUpdate == failure {
			t.Fatalf("restart = %v, failure = %v", s.restartAfterUpdate, failure)
		}
	}
}

func TestStartScreenReceivesUpdateWithoutWaiting(t *testing.T) {
	results := make(chan *updates.Release, 1)
	s := &StartScreen{updateResults: results, updateChecked: true}
	s.pollUpdate()
	if s.availableUpdate != nil {
		t.Fatal("unexpected update")
	}
	results <- &updates.Release{Version: "v2.0.0", DownloadURL: "https://github.com/benprew/s30/releases/download/v2.0.0/game"}
	close(results)
	s.pollUpdate()
	if s.availableUpdate == nil || s.availableUpdate.Version != "v2.0.0" {
		t.Fatal("update was not received")
	}
	s.availableUpdate = nil
	s.pollUpdate()
	if s.availableUpdate != nil || s.updateResults != nil {
		t.Fatal("dismissed update returned")
	}
}
