//go:build windows

package figmaasset

import (
	"errors"
	"os"
	"os/exec"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// Windows doesn't support Unix process groups.
}

func terminateProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

func forceKillProcess(pid int) error {
	return terminateProcess(pid)
}

func isProcessNotFound(err error) bool {
	return errors.Is(err, errProcessNotFound) || errors.Is(err, os.ErrProcessDone)
}
