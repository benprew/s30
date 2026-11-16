package updates

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/benprew/s30/game/version"
)

// PublicKey is the base64 Ed25519 public key set when a release is built.
var PublicKey string

const MaxArtifactSize int64 = 2 << 30

type Artifact struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Package struct {
	Artifact
	Helper      *Artifact `json:"helper,omitempty"`
	VersionCode int64     `json:"version_code,omitempty"`
}

type Manifest struct {
	Schema    int                `json:"schema"`
	Version   string             `json:"version"`
	Platforms map[string]Package `json:"platforms"`
}

// VerifyManifest checks the signature before it reads the release data.
func VerifyManifest(data []byte, signature, publicKey string) (*Manifest, error) {
	key, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, errors.New("update signing key is not configured")
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil || !ed25519.Verify(key, data, sig) {
		return nil, errors.New("update signature is invalid")
	}
	var manifest Manifest
	if len(data) > 64<<10 {
		return nil, errors.New("update manifest is too large")
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Schema != 1 || !version.Valid(manifest.Version) || len(manifest.Platforms) == 0 {
		return nil, errors.New("update manifest is invalid")
	}
	for platform, pkg := range manifest.Platforms {
		if err := pkg.validate(manifest.Version); err != nil {
			return nil, fmt.Errorf("%s: %w", platform, err)
		}
		if pkg.Helper != nil {
			if err := pkg.Helper.validate(manifest.Version); err != nil {
				return nil, err
			}
		}
	}
	return &manifest, nil
}

func (a Artifact) validate(releaseVersion string) error {
	if !validDownloadURL(a.URL, releaseVersion) {
		return errors.New("update artifact URL is invalid")
	}
	digest, err := hex.DecodeString(a.SHA256)
	if err != nil || len(digest) != 32 || a.Size <= 0 || a.Size > MaxArtifactSize {
		return errors.New("update artifact size or checksum is invalid")
	}
	return nil
}

func validDownloadURL(raw, releaseVersion string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" {
		return false
	}
	prefix := "/benprew/s30/releases/download/" + releaseVersion + "/"
	name := strings.TrimPrefix(u.Path, prefix)
	return version.Valid(releaseVersion) && strings.HasPrefix(u.Path, prefix) &&
		name != "" && name != "." && name != ".." && !strings.Contains(name, "/")
}

func (r *Release) verifiedPackage(publicKey, platform string) (*Package, error) {
	m, err := VerifyManifest(r.Manifest, r.Signature, publicKey)
	if err != nil {
		return nil, err
	}
	if m.Version != r.Version {
		return nil, errors.New("update version does not match its signature")
	}
	pkg, ok := m.Platforms[platform]
	if !ok {
		return nil, errors.New("update has no file for this platform")
	}
	if platform == "android" {
		if pkg.VersionCode <= androidVersionCode() {
			return nil, errors.New("android update version code is not newer")
		}
	} else if automaticPlatform(platform) && pkg.Helper == nil {
		return nil, errors.New("update has no installation helper")
	}
	return &pkg, nil
}
