//go:build ios || (!darwin && !windows && !linux && !android)

package updates

import "errors"

func Supported() bool { return false }

func OpenDownload(_ string) error {
	return errors.New("release downloads are not supported on this platform")
}
