package version

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Current is set with -ldflags for release builds.
var Current = "dev"

var pattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
var gitDescription = regexp.MustCompile(`-[0-9]+-g[0-9a-f]+$`)

func parse(value string) ([3]uint64, bool, bool) {
	var parts [3]uint64
	match := pattern.FindStringSubmatch(value)
	if match == nil {
		return parts, false, false
	}
	for i := range parts {
		n, err := strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return parts, false, false
		}
		parts[i] = n
	}
	return parts, match[4] != "", true
}

// Valid reports whether value has a release version number.
func Valid(value string) bool {
	if strings.HasSuffix(value, "-dirty") || gitDescription.MatchString(value) {
		return false
	}
	_, _, ok := parse(value)
	return ok
}

// AndroidVersionCode returns the version code for a stable release.
func AndroidVersionCode(value string) (int64, error) {
	parts, prerelease, ok := parse(value)
	if !ok || prerelease || strings.Contains(value, "+") {
		return 0, errors.New("a stable release version is required")
	}
	major, minor, patch := parts[0], parts[1], parts[2]
	if major > 2099 || minor > 999 || patch > 999 || major+minor+patch == 0 {
		return 0, errors.New("release version is outside the Android version code range")
	}
	return int64(major*1_000_000 + minor*1_000 + patch), nil
}

// Newer reports whether latest is a newer stable release.
func Newer(latest, current string) bool {
	if !Valid(current) {
		return false
	}
	a, preA, okA := parse(latest)
	b, preB, okB := parse(current)
	if !okA || !okB || preA {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return preB
}
