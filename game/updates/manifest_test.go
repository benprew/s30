package updates

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func signedRelease(t *testing.T, platform string, payload, helper []byte) (*Release, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	artifact := testArtifact("s30", payload)
	pkg := Package{Artifact: artifact, VersionCode: 42}
	if helper != nil {
		a := testArtifact("s30-updater", helper)
		pkg.Helper = &a
	}
	m := Manifest{Schema: 1, Version: "v2.0.0", Platforms: map[string]Package{platform: pkg}}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return &Release{
		Version: m.Version, Manifest: data,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data)),
		Downloads: map[string]string{platform: artifact.URL},
	}, base64.StdEncoding.EncodeToString(pub)
}

func testArtifact(name string, data []byte) Artifact {
	digest := sha256.Sum256(data)
	return Artifact{URL: "https://github.com/benprew/s30/releases/download/v2.0.0/" + name, SHA256: hex.EncodeToString(digest[:]), Size: int64(len(data))}
}

func TestManifestVerification(t *testing.T) {
	r, key := signedRelease(t, "windows/amd64", []byte("game"), []byte("helper"))
	m, err := VerifyManifest(r.Manifest, r.Signature, key)
	if err != nil || m.Version != r.Version || m.Platforms["windows/amd64"].Helper == nil {
		t.Fatalf("VerifyManifest = %+v, %v", m, err)
	}
	for _, tc := range []struct {
		name, key, signature string
		data                 []byte
	}{
		{"changed payload", key, r.Signature, append(append([]byte{}, r.Manifest...), ' ')},
		{"missing key", "", r.Signature, r.Manifest},
		{"wrong key", base64.StdEncoding.EncodeToString(make([]byte, 32)), r.Signature, r.Manifest},
		{"missing signature", key, "", r.Manifest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := VerifyManifest(tc.data, tc.signature, tc.key); err == nil {
				t.Fatal("untrusted manifest was accepted")
			}
		})
	}
}

func TestArtifactValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Artifact)
	}{
		{"http", func(a *Artifact) { a.URL = "http://github.com/benprew/s30/releases/download/v2.0.0/s30" }},
		{"other repository", func(a *Artifact) { a.URL = "https://github.com/other/s30/releases/download/v2.0.0/s30" }},
		{"other version", func(a *Artifact) { a.URL = "https://github.com/benprew/s30/releases/download/v1.0.0/s30" }},
		{"query", func(a *Artifact) { a.URL += "?download=1" }},
		{"digest", func(a *Artifact) { a.SHA256 = "invalid" }},
		{"empty file", func(a *Artifact) { a.Size = 0 }},
		{"large file", func(a *Artifact) { a.Size = 1 << 32 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := testArtifact("s30", []byte("game"))
			tc.mutate(&a)
			if err := a.validate("v2.0.0"); err == nil {
				t.Fatal("invalid artifact was accepted")
			}
		})
	}
}
