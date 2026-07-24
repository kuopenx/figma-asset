//go:build !windows

package figmaasset

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var errProcessNotFound = errors.New("process not found")

func processStartTime(pid int) (string, error) {
	command := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=")
	command.Env = append(os.Environ(), "LC_ALL=C")
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
