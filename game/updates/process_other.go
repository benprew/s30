//go:build !windows

package updates

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureUpdateProcess(_ *exec.Cmd) {}

func replaceExecutable(source, target string) error {
	return os.Rename(source, target)
}

func parentWaiter(pid int) (func() error, func(), error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, nil, err
	}
	return func() error {
		deadline := time.Now().Add(2 * time.Minute)
		for time.Now().Before(deadline) {
			if err := process.Signal(syscall.Signal(0)); err != nil {
				if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
					return nil
				}
				return err
			}
			time.Sleep(100 * time.Millisecond)
		}
		return errors.New("the game did not exit within two minutes")
	}, func() { process.Release() }, nil
}
