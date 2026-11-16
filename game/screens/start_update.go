package screens

import (
	"context"
	"errors"
	"fmt"

	"github.com/benprew/s30/game/updates"
	"github.com/hajimehoshi/ebiten/v2"
)

type updateResult struct {
	prepared *updates.Prepared
	restart  bool
	err      error
}

func (s *StartScreen) setAutomaticUpdate(automatic bool) {
	s.automaticUpdate = automatic
	label := "Download update"
	if automatic {
		label = "Update and restart"
		if s.availableUpdate.Platform == "android" {
			label = "Install update"
		}
	}
	if s.updateActionBtn != nil {
		s.updateActionBtn.ButtonText.Text = label
	}
}

func (s *StartScreen) pollUpdate() {
	if !s.updateChecked {
		s.updateChecked = true
		s.updateResults = updates.Start()
	}
	if s.updateResults != nil {
		select {
		case release := <-s.updateResults:
			s.availableUpdate = release
			s.setAutomaticUpdate(release.CanInstall())
			if release != nil && s.updateNoticeLabel != nil {
				s.updateNoticeLabel.Text = "Version " + release.Version + " is available"
			}
			s.updateResults = nil
		default:
		}
	}
	if s.updateProgress != nil {
		select {
		case status := <-s.updateProgress:
			s.updateStatus = status
		default:
		}
	}
	if s.downloadResult == nil {
		return
	}
	select {
	case result := <-s.downloadResult:
		s.downloadResult = nil
		s.updateProgress = nil
		if s.updateCancel != nil {
			s.updateCancel()
			s.updateCancel = nil
		}
		if s.availableUpdate == nil {
			result.prepared.Cleanup()
			s.updateStatus = ""
			return
		}
		if result.err != nil {
			s.updateStatus = ""
			if errors.Is(result.err, context.Canceled) {
				s.availableUpdate = nil
				return
			}
			fmt.Printf("Update failed: %v\n", result.err)
			if s.automaticUpdate {
				s.updateError = "Could not install the update. You can download it instead."
			} else {
				s.updateError = "Could not open the download. Please try again."
			}
			s.setAutomaticUpdate(false)
			s.preparedUpdate.Cleanup()
			s.preparedUpdate = nil
			return
		}
		if result.prepared != nil {
			s.preparedUpdate = result.prepared
			s.startInstallation(result.prepared)
			return
		}
		s.restartAfterUpdate = result.restart
		if s.preparedUpdate != nil && !result.restart {
			s.updateStatus = "Approve the update in Android. Tap Install update to try again."
		} else {
			s.updateStatus = ""
		}
	default:
	}
}

func (s *StartScreen) startInstallation(prepared *updates.Prepared) {
	s.updateStatus = "Starting the installer..."
	result := make(chan updateResult, 1)
	s.downloadResult = result
	go func() {
		restart, err := prepared.Install()
		result <- updateResult{restart: restart, err: err}
	}()
}

func (s *StartScreen) dismissUpdate() {
	s.availableUpdate = nil
	if s.updateCancel != nil {
		s.updateCancel()
		s.updateCancel = nil
	}
	if s.downloadResult != nil {
		result := s.downloadResult
		s.downloadResult = nil
		go func() {
			pending := <-result
			pending.prepared.Cleanup()
		}()
	}
	s.preparedUpdate.Cleanup()
	s.preparedUpdate = nil
	s.updateProgress = nil
	s.updateStatus = ""
	s.updateError = ""
}

func (s *StartScreen) updateDownload(w, h int, scale float64) {
	if s.availableUpdate == nil {
		return
	}
	opts := &ebiten.DrawImageOptions{}
	if s.downloadResult != nil && s.updateCancel == nil {
		return
	}
	s.laterBtn.Update(opts, scale, w, h)
	if s.laterBtn.IsClicked() {
		s.dismissUpdate()
		return
	}
	if s.downloadResult != nil {
		return
	}
	s.updateActionBtn.Update(opts, scale, w, h)
	if !s.updateActionBtn.IsClicked() {
		return
	}
	s.updateError = ""
	if s.automaticUpdate && s.preparedUpdate != nil {
		s.startInstallation(s.preparedUpdate)
		return
	}
	result := make(chan updateResult, 1)
	s.downloadResult = result
	release := s.availableUpdate
	if !s.automaticUpdate {
		go func() { result <- updateResult{err: updates.OpenDownload(release.DownloadURL)} }()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.updateCancel = cancel
	progress := make(chan string, 1)
	s.updateProgress = progress
	s.updateStatus = "Downloading update..."
	go func() {
		prepared, err := updates.Prepare(ctx, release, func(received, total int64) {
			reportUpdateProgress(progress, received, total)
		})
		result <- updateResult{prepared: prepared, err: err}
	}()
}

func reportUpdateProgress(progress chan string, received, total int64) {
	status := fmt.Sprintf("Downloading update... %d%%", received*100/total)
	select {
	case progress <- status:
		return
	default:
	}
	select {
	case <-progress:
	default:
	}
	select {
	case progress <- status:
	default:
	}
}

func (s *StartScreen) drawUpdate(screen *ebiten.Image, scale float64) {
	opts := &ebiten.DrawImageOptions{}
	s.versionLabel.Draw(screen, opts, scale)
	if s.availableUpdate == nil {
		return
	}
	s.updateNoticeLabel.Draw(screen, opts, scale)
	s.updateActionBtn.Draw(screen, opts, scale)
	s.laterBtn.Draw(screen, opts, scale)
	status := s.updateStatus
	if s.updateError != "" {
		status = s.updateError
	}
	if status != "" {
		s.updateStatusLabel.Text = status
		s.updateStatusLabel.Draw(screen, opts, scale)
	}
}
