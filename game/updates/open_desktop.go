//go:build (darwin && !ios) || windows || (linux && !android)

package updates

import (
	"os/exec"
	"runtime"
)

func Supported() bool { return true }

// OpenDownload opens the release download in the default browser.
func OpenDownload(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	default:
		return exec.Command("xdg-open", url).Run()
	}
}
