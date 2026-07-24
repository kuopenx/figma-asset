//go:build !windows

package figmaasset

import (
	"errors"
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

func forceKillProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}

func isProcessNotFound(err error) bool {
	return errors.Is(err, errProcessNotFound) || errors.Is(err, syscall.ESRCH)
}
