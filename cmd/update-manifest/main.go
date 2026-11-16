package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/benprew/s30/game/updates"
	"github.com/benprew/s30/game/version"
)

var releaseFiles = []struct{ platform, game, helper string }{
	{"windows/amd64", "s30_windows.exe", "s30-updater_windows-amd64.exe"},
	{"windows/arm64", "s30_windows-arm.exe", "s30-updater_windows-arm64.exe"},
	{"linux/amd64", "s30_linux", "s30-updater_linux-amd64"},
	{"linux/arm64", "s30_linux-arm", "s30-updater_linux-arm64"},
	{"android", "s30_android.apk", ""},
	{"darwin/amd64", "s30_mac-intel", ""},
	{"darwin/arm64", "s30_mac-apple-silicon", ""},
}

func main() {
	keygen := flag.Bool("keygen", false, "create a signing seed and public key")
	code := flag.Bool("android-version-code", false, "print the Android version code for a stable release")
	releaseVersion := flag.String("version", "", "stable release tag")
	artifacts := flag.String("artifacts", "artifacts", "directory of downloaded build artifacts")
	out := flag.String("out", "s30-update-manifest.json", "output file or key file prefix")
	flag.Parse()
	var err error
	switch {
	case *keygen:
		err = generateKey(*out)
	case *code:
		var value int64
		value, err = version.AndroidVersionCode(*releaseVersion)
		if err == nil {
			fmt.Println(value)
		}
	default:
		err = signRelease(*releaseVersion, *artifacts, *out)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func generateKey(prefix string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	for _, file := range []struct {
		suffix, value string
		mode          os.FileMode
	}{
		{".key", base64.StdEncoding.EncodeToString(priv.Seed()), 0600},
		{".pub", base64.StdEncoding.EncodeToString(pub), 0644},
	} {
		f, err := os.OpenFile(prefix+file.suffix, os.O_WRONLY|os.O_CREATE|os.O_EXCL, file.mode)
		if err != nil {
			return err
		}
		_, writeErr := fmt.Fprintln(f, file.value)
		if err := errors.Join(writeErr, f.Close()); err != nil {
			return err
		}
	}
	return nil
}

func releaseArtifact(root, folder, name, tag string) (updates.Artifact, error) {
	f, err := os.Open(filepath.Join(root, folder, name))
	if err != nil {
		return updates.Artifact{}, err
	}
	defer f.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(f, updates.MaxArtifactSize+1))
	if err != nil || size <= 0 || size > updates.MaxArtifactSize {
		return updates.Artifact{}, fmt.Errorf("invalid release file %s: %v", name, err)
	}
	return updates.Artifact{
		URL:    "https://github.com/benprew/s30/releases/download/" + tag + "/" + name,
		SHA256: hex.EncodeToString(hash.Sum(nil)), Size: size,
	}, nil
}

func signRelease(tag, root, output string) error {
	expectedCode, err := version.AndroidVersionCode(tag)
	if err != nil {
		return err
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("UPDATE_SIGNING_KEY")))
	if err != nil || len(seed) != ed25519.SeedSize {
		return errors.New("UPDATE_SIGNING_KEY must contain a base64 Ed25519 seed")
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey))
	if public != strings.TrimSpace(os.Getenv("UPDATE_PUBLIC_KEY")) {
		return errors.New("UPDATE_PUBLIC_KEY does not match UPDATE_SIGNING_KEY")
	}
	codeData, err := os.ReadFile(filepath.Join(root, "s30_android.apk", "s30_android.version"))
	if err != nil {
		return err
	}
	code, err := strconv.ParseInt(strings.TrimSpace(string(codeData)), 10, 64)
	if err != nil || code != expectedCode {
		return errors.New("android build version code does not match the release version")
	}
	manifest := updates.Manifest{Schema: 1, Version: tag, Platforms: make(map[string]updates.Package)}
	for _, entry := range releaseFiles {
		artifact, readErr := releaseArtifact(root, entry.game, entry.game, tag)
		if readErr != nil {
			return readErr
		}
		pkg := updates.Package{Artifact: artifact}
		if entry.helper != "" {
			helper, helperErr := releaseArtifact(root, entry.game, entry.helper, tag)
			if helperErr != nil {
				return helperErr
			}
			pkg.Helper = &helper
		}
		if entry.platform == "android" {
			pkg.VersionCode = code
		}
		manifest.Platforms[entry.platform] = pkg
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, data))
	if _, err := updates.VerifyManifest(data, signature, public); err != nil {
		return err
	}
	if err := os.WriteFile(output, data, 0644); err != nil {
		return err
	}
	return os.WriteFile(output+".sig", []byte(signature+"\n"), 0644)
}
