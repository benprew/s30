package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Prepared struct {
	dir            string
	payload        string
	helper         string
	plan           installPlan
	installer      AndroidInstaller
	artifact       Artifact
	helperArtifact Artifact
}

func automaticPlatform(platform string) bool {
	return platform == "android" || strings.HasPrefix(platform, "windows/") || strings.HasPrefix(platform, "linux/")
}

// CanInstall reports whether this release has a verified automatic update.
func (r *Release) CanInstall() bool {
	if r == nil || r.pkg == nil || !automaticPlatform(r.Platform) || os.Getenv("S30_DISABLE_AUTO_UPDATE") == "1" {
		return false
	}
	if strings.HasPrefix(r.Platform, "linux/") {
		target, err := executablePath()
		if err != nil || !desktopLocationAllowed(r.Platform, target) {
			return false
		}
	}
	return true
}

// Prepare downloads the signed update into a private directory.
func Prepare(ctx context.Context, release *Release, progress Progress) (*Prepared, error) {
	if !release.CanInstall() {
		return nil, errors.New("automatic updates are not available")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	client := &http.Client{}
	if runtime.GOOS == "android" {
		return prepareAndroid(ctx, client, release, progress)
	}
	if release.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		return nil, errors.New("update is for another platform")
	}
	target, err := executablePath()
	if err != nil {
		return nil, err
	}
	return prepareDesktop(ctx, client, release, target, progress)
}

func executablePath() (string, error) {
	target, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(target)
}

func desktopLocationAllowed(platform, target string) bool {
	if !strings.HasPrefix(platform, "linux/") {
		return true
	}
	for _, prefix := range []string{"/usr/", "/bin/", "/sbin/", "/lib/", "/lib64/", "/snap/", "/app/", "/var/lib/flatpak/"} {
		if strings.HasPrefix(target, prefix) {
			return false
		}
	}
	return os.Getenv("APPIMAGE") == "" && os.Getenv("FLATPAK_ID") == "" && os.Getenv("SNAP") == ""
}

func prepareDesktop(ctx context.Context, client *http.Client, release *Release, target string, progress Progress) (_ *Prepared, err error) {
	platform := release.Platform
	if !desktopLocationAllowed(platform, target) {
		return nil, errors.New("use your package manager to update this installation")
	}
	pkg := release.pkg
	if pkg == nil || pkg.Helper == nil {
		return nil, errors.New("automatic update has not been verified")
	}
	info, err := os.Lstat(target)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return nil, errors.New("the installed executable is not a regular file")
	}
	oldHash, err := fileChecksum(target)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(filepath.Dir(target), ".s30-update-")
	if err != nil {
		return nil, fmt.Errorf("the game directory must be writable: %w", err)
	}
	p := &Prepared{dir: dir, helperArtifact: *pkg.Helper}
	defer func() {
		if err != nil {
			p.Cleanup()
		}
	}()
	suffix := ""
	if strings.HasPrefix(platform, "windows/") {
		suffix = ".exe"
	}
	p.payload = filepath.Join(dir, "s30"+suffix)
	p.helper = filepath.Join(dir, "s30-updater"+suffix)
	total := pkg.Size + pkg.Helper.Size
	report := func(offset int64) Progress {
		return func(received, _ int64) {
			if progress != nil {
				progress(offset+received, total)
			}
		}
	}
	if err = download(ctx, client, pkg.Artifact, release.Version, p.payload, report(0)); err != nil {
		return nil, err
	}
	if err = download(ctx, client, *pkg.Helper, release.Version, p.helper, report(pkg.Size)); err != nil {
		return nil, err
	}
	if err = os.Chmod(p.payload, info.Mode().Perm()); err != nil {
		return nil, err
	}
	if err = os.Chmod(p.helper, 0700); err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	p.plan = installPlan{Release: *release, Platform: platform, Target: target, ParentPID: os.Getpid(), OldSHA256: oldHash, Args: os.Args[1:], Workdir: cwd}
	data, err := json.Marshal(p.plan)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), data, 0600); err != nil {
		return nil, err
	}
	return p, nil
}

func prepareAndroid(ctx context.Context, client *http.Client, release *Release, progress Progress) (_ *Prepared, err error) {
	pkg := release.pkg
	if pkg == nil {
		return nil, errors.New("automatic update has not been verified")
	}
	androidUpdate.RLock()
	cacheDir, installer := androidUpdate.dir, androidUpdate.installer
	androidUpdate.RUnlock()
	if cacheDir == "" || installer == nil {
		return nil, errors.New("android installer is not available")
	}
	root := filepath.Join(cacheDir, "updates")
	if err = os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(root, ".s30-update-")
	if err != nil {
		return nil, err
	}
	p := &Prepared{dir: dir, payload: filepath.Join(dir, "s30.apk"), installer: installer, artifact: pkg.Artifact}
	defer func() {
		if err != nil {
			p.Cleanup()
		}
	}()
	if err := download(ctx, client, pkg.Artifact, release.Version, p.payload, progress); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "version-code"), []byte(strconv.FormatInt(pkg.VersionCode, 10)), 0600); err != nil {
		return nil, err
	}
	return p, nil
}

// Install starts the system installer. A true result requires the game to exit.
func (p *Prepared) Install() (bool, error) {
	if p.installer != nil {
		if err := verifyFile(p.payload, p.artifact); err != nil {
			return false, err
		}
		return false, p.installer.InstallUpdate(p.payload)
	}
	if err := launchHelper(p); err != nil {
		return false, err
	}
	return true, nil
}

// Cleanup removes files from an update that has not started installation.
func (p *Prepared) Cleanup() {
	if p != nil && p.dir != "" {
		os.RemoveAll(p.dir)
	}
}
