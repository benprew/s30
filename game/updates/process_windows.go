package updates

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func configureUpdateProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW | windows.CREATE_NEW_PROCESS_GROUP}
}

func replaceExecutable(source, target string) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := os.Rename(source, target)
		if (!errors.Is(err, windows.ERROR_SHARING_VIOLATION) && !errors.Is(err, windows.ERROR_ACCESS_DENIED)) ||
			time.Now().After(deadline) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func parentWaiter(pid int) (func() error, func(), error) {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return nil, nil, err
	}
	return func() error {
		status, err := windows.WaitForSingleObject(handle, 120000)
		if err != nil {
			return err
		}
		if status != windows.WAIT_OBJECT_0 {
			return errors.New("the game did not exit within two minutes")
		}
		return nil
	}, func() { windows.CloseHandle(handle) }, nil
}
