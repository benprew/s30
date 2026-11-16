package updates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Progress func(received, total int64)

type progressWriter struct {
	received, total int64
	report          Progress
	lastPercent     int64
}

func (p *progressWriter) Write(data []byte) (int, error) {
	p.received += int64(len(data))
	percent := p.received * 100 / p.total
	if p.report != nil && percent != p.lastPercent {
		p.lastPercent = percent
		p.report(p.received, p.total)
	}
	return len(data), nil
}

func artifactClient(client *http.Client) *http.Client {
	c := *client
	c.Timeout = 30 * time.Minute
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		host := req.URL.Host
		if len(via) >= 10 || req.URL.Scheme != "https" || req.URL.User != nil ||
			(host != "github.com" && host != "release-assets.githubusercontent.com" &&
				host != "objects.githubusercontent.com" && host != "github-releases.githubusercontent.com") {
			return errors.New("update download redirected to an untrusted address")
		}
		return nil
	}
	return &c
}

func download(ctx context.Context, client *http.Client, artifact Artifact, releaseVersion, dest string, progress Progress) (err error) {
	if !validDownloadURL(artifact.URL, releaseVersion) || artifact.Size <= 0 || artifact.Size > MaxArtifactSize {
		return errors.New("invalid download")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")
	res, err := artifactClient(client).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("update download returned %d", res.StatusCode)
	}
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(dest)
		}
	}()
	hash := sha256.New()
	p := &progressWriter{total: artifact.Size, report: progress, lastPercent: -1}
	n, err := io.Copy(io.MultiWriter(f, hash, p), io.LimitReader(res.Body, artifact.Size+1))
	if err != nil {
		return err
	}
	if n != artifact.Size || hex.EncodeToString(hash.Sum(nil)) != artifact.SHA256 {
		return errors.New("update download failed its size or checksum check")
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}
