package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/benprew/s30/game/version"
)

const DefaultURL = "https://s30.throwingbones.com/api/releases/latest"

type Release struct {
	Version     string            `json:"version"`
	Downloads   map[string]string `json:"downloads"`
	DownloadURL string            `json:"-"`
	Manifest    []byte            `json:"manifest,omitempty"`
	Signature   string            `json:"signature,omitempty"`
	Platform    string            `json:"-"`
	pkg         *Package
}

// Check returns an update only when a newer release has a download for platform.
func Check(ctx context.Context, client *http.Client, endpoint, current, platform string) (*Release, error) {
	if !version.Valid(current) {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "s30-game-client/"+current)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update service returned %d", res.StatusCode)
	}
	var release Release
	if err = json.NewDecoder(io.LimitReader(res.Body, 128<<10)).Decode(&release); err != nil {
		return nil, err
	}
	if !version.Valid(release.Version) {
		return nil, fmt.Errorf("invalid release version %q", release.Version)
	}
	if !version.Newer(release.Version, current) {
		return nil, nil
	}
	if len(release.Manifest) != 0 && release.Signature != "" && PublicKey != "" {
		pkg, verifyErr := release.verifiedPackage(PublicKey, platform)
		if verifyErr != nil {
			return nil, verifyErr
		}
		release.DownloadURL = pkg.URL
		release.pkg = pkg
	} else {
		release.DownloadURL = release.Downloads[platform]
	}
	if release.DownloadURL == "" {
		return nil, nil
	}
	if !validDownloadURL(release.DownloadURL, release.Version) {
		return nil, fmt.Errorf("invalid release download URL")
	}
	release.Platform = platform
	return &release, nil
}

// Start checks once in the background. Unsupported platforms do not check.
func Start() <-chan *Release {
	result := make(chan *Release, 1)
	go func() {
		defer close(result)
		cleanupCompletedUpdates()
		androidUpdate.RLock()
		cacheDir, versionCode := androidUpdate.dir, androidUpdate.versionCode
		androidUpdate.RUnlock()
		cleanupAndroidUpdates(cacheDir, versionCode)
		if !Supported() {
			return
		}
		endpoint := os.Getenv("S30_UPDATE_URL")
		if endpoint == "" {
			endpoint = DefaultURL
		}
		platform := runtime.GOOS + "/" + runtime.GOARCH
		if runtime.GOOS == "android" {
			platform = "android"
		}
		release, err := Check(context.Background(), &http.Client{Timeout: 5 * time.Second}, endpoint, version.Current, platform)
		if err == nil && release != nil {
			result <- release
		}
	}()
	return result
}
