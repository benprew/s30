package updates

import "errors"

func Supported() bool {
	androidUpdate.RLock()
	defer androidUpdate.RUnlock()
	return androidUpdate.installer != nil
}

func OpenDownload(url string) error {
	androidUpdate.RLock()
	installer := androidUpdate.installer
	androidUpdate.RUnlock()
	if installer == nil {
		return errors.New("android installer is not available")
	}
	return installer.OpenDownload(url)
}
