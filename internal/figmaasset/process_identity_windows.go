//go:build windows

package figmaasset

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

var errProcessNotFound = errors.New("process not found")

func processStartTime(pid int) (string, error) {
	command := exec.Command(
		"powershell",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		"(Get-Process -Id "+strconv.Itoa(pid)+" -ErrorAction Stop).StartTime.ToUniversalTime().Ticks",
	)
	output, err := command.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", errProcessNotFound
		}
		return "", err
	}
	startTime := strings.TrimSpace(string(output))
	if startTime == "" {
		return "", errProcessNotFound
	}
	return startTime, nil
}
