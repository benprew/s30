package updates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type installPlan struct {
	Release   Release  `json:"release"`
	Platform  string   `json:"platform"`
	Target    string   `json:"target"`
	ParentPID int      `json:"parent_pid"`
	OldSHA256 string   `json:"old_sha256"`
	Args      []string `json:"args"`
	Workdir   string   `json:"workdir"`
}

func fileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyFile(path string, artifact Artifact) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != artifact.Size {
		return errors.New("staged update file is missing or has changed")
	}
	digest, err := fileChecksum(path)
	if err != nil || digest != artifact.SHA256 {
		return errors.New("staged update checksum is invalid")
	}
	return nil
}

func launchHelper(p *Prepared) error {
	if err := verifyFile(p.helper, p.helperArtifact); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(p.dir, "update.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	cmd := exec.Command(p.helper, "--plan", filepath.Join(p.dir, "plan.json"))
	configureUpdateProcess(cmd)
	cmd.Stdout, cmd.Stderr = log, log
	if err = cmd.Start(); err != nil {
		log.Close()
		return err
	}
	log.Close()
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if _, err := os.Stat(filepath.Join(p.dir, "ready")); err == nil {
				return nil
			}
		case err := <-done:
			return fmt.Errorf("update helper stopped before installation: %v", err)
		case <-deadline.C:
			cmd.Process.Kill()
			<-done
			return errors.New("update helper did not start in time")
		}
	}
}

// Apply installs an update after the game process exits.
func Apply(planPath string) error {
	f, err := os.Open(planPath)
	if err != nil {
		return err
	}
	var plan installPlan
	err = json.NewDecoder(io.LimitReader(f, 128<<10)).Decode(&plan)
	f.Close()
	if err != nil {
		return err
	}
	dir := filepath.Dir(planPath)
	if !filepath.IsAbs(plan.Target) || !filepath.IsAbs(dir) || filepath.Clean(plan.Target) != plan.Target ||
		filepath.Dir(dir) != filepath.Dir(plan.Target) || !strings.HasPrefix(filepath.Base(dir), ".s30-update-") ||
		plan.Platform != runtime.GOOS+"/"+runtime.GOARCH || !desktopLocationAllowed(plan.Platform, plan.Target) ||
		plan.ParentPID <= 1 || plan.ParentPID == os.Getpid() {
		return errors.New("update installation plan is invalid")
	}
	pkg, err := plan.Release.verifiedPackage(PublicKey, plan.Platform)
	if err != nil {
		return err
	}
	name := "s30"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	payload := filepath.Join(dir, name)
	if err = verifyFile(payload, pkg.Artifact); err != nil {
		return err
	}
	wait, closeParent, err := parentWaiter(plan.ParentPID)
	if err != nil {
		return err
	}
	defer closeParent()
	if err = os.WriteFile(filepath.Join(dir, "ready"), nil, 0600); err != nil {
		return err
	}
	if err = wait(); err != nil {
		return err
	}
	info, err := os.Lstat(plan.Target)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("installed executable is no longer a regular file")
	}
	oldHash, err := fileChecksum(plan.Target)
	if err != nil || oldHash != plan.OldSHA256 {
		return errors.New("installed executable changed during the update")
	}
	if err = verifyFile(payload, pkg.Artifact); err != nil {
		return err
	}
	return replaceAndRestart(plan.Target, payload, func() error {
		cmd := exec.Command(plan.Target, plan.Args...)
		cmd.Dir = plan.Workdir
		if err := cmd.Start(); err != nil {
			return err
		}
		cmd.Process.Release()
		return nil
	})
}

func replaceAndRestart(target, payload string, start func() error) error {
	dir := filepath.Dir(payload)
	backup := filepath.Join(dir, "previous")
	if err := backupExecutable(target, backup); err != nil {
		return errors.Join(err, start())
	}
	if err := replaceExecutable(payload, target); err != nil {
		_ = os.Remove(backup)
		return errors.Join(err, start())
	}
	if err := start(); err != nil {
		return rollback(target, backup, start, err)
	}
	// Windows keeps the helper file open until this process stops.
	_ = os.WriteFile(filepath.Join(dir, "complete"), nil, 0600)
	return nil
}

func rollback(target, backup string, start func() error, cause error) error {
	if err := replaceExecutable(backup, target); err != nil {
		return fmt.Errorf("update failed: %v; restore %s manually: %w", cause, backup, err)
	}
	return errors.Join(cause, start())
}

func backupExecutable(target, backup string) (err error) {
	if os.Link(target, backup) == nil {
		return nil
	}
	source, err := os.Open(target)
	if err != nil {
		return err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return err
	}
	dest, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() {
		dest.Close()
		if err != nil {
			os.Remove(backup)
		}
	}()
	if _, err = io.Copy(dest, source); err != nil {
		return err
	}
	if err = dest.Sync(); err != nil {
		return err
	}
	return dest.Close()
}

func cleanupCompletedUpdates() {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return
	}
	target, err := executablePath()
	if err != nil {
		return
	}
	cleanupDesktopUpdates(filepath.Dir(target))
}

func cleanupDesktopUpdates(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".s30-update-") {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		_, completeErr := os.Stat(filepath.Join(dir, "complete"))
		_, logErr := os.Stat(filepath.Join(dir, "update.log"))
		_, backupErr := os.Stat(filepath.Join(dir, "previous"))
		info, statErr := entry.Info()
		abandoned := statErr == nil && time.Since(info.ModTime()) > 48*time.Hour &&
			os.IsNotExist(logErr) && os.IsNotExist(backupErr)
		if completeErr == nil || abandoned {
			if err := os.RemoveAll(dir); err != nil && completeErr == nil {
				_ = os.WriteFile(filepath.Join(dir, "complete"), nil, 0600)
			}
		}
	}
}
