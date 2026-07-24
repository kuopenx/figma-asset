package figmaasset

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

type daemonState struct {
	PID              int    `json:"pid"`
	ProcessStartTime string `json:"processStartTime"`
	Nonce            string `json:"nonce"`
}

func daemonStatePath() string {
	return filepath.Join(pluginDir(), "daemon-state.json")
}

func writeDaemonState(pid int) (daemonState, error) {
	startTime, err := processStartTime(pid)
	if err != nil {
		return daemonState{}, fmt.Errorf("read daemon process identity: %w", err)
	}
	state := daemonState{
		PID:              pid,
		ProcessStartTime: startTime,
		Nonce:            randomDaemonNonce(),
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return daemonState{}, err
	}

	path := daemonStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return daemonState{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".daemon-state-*")
	if err != nil {
		return daemonState{}, err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return daemonState{}, err
	}
	if _, err := temporary.Write(append(payload, '\n')); err != nil {
		temporary.Close()
		return daemonState{}, err
	}
	if err := temporary.Close(); err != nil {
		return daemonState{}, err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return daemonState{}, err
	}
	return state, nil
}

func ownedDaemonState() (daemonState, bool, error) {
	payload, err := os.ReadFile(daemonStatePath())
	if errors.Is(err, os.ErrNotExist) {
		return daemonState{}, false, nil
	}
	if err != nil {
		return daemonState{}, false, fmt.Errorf("read daemon state: %w", err)
	}

	var state daemonState
	if err := json.Unmarshal(payload, &state); err != nil || state.PID <= 0 || state.ProcessStartTime == "" || state.Nonce == "" {
		_ = os.Remove(daemonStatePath())
		return daemonState{}, false, nil
	}

	startTime, err := processStartTime(state.PID)
	if isProcessNotFound(err) {
		removeDaemonState(state.Nonce)
		return daemonState{}, false, nil
	}
	if err != nil {
		return daemonState{}, false, fmt.Errorf("read daemon process identity: %w", err)
	}
	if startTime != state.ProcessStartTime {
		removeDaemonState(state.Nonce)
		return daemonState{}, false, nil
	}
	return state, true, nil
}

func removeDaemonState(nonce string) {
	payload, err := os.ReadFile(daemonStatePath())
	if err != nil {
		return
	}
	var state daemonState
	if json.Unmarshal(payload, &state) == nil && state.Nonce == nonce {
		_ = os.Remove(daemonStatePath())
	}
}

func waitForOwnedDaemonExit(state daemonState, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		startTime, err := processStartTime(state.PID)
		if isProcessNotFound(err) || (err == nil && startTime != state.ProcessStartTime) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func isDaemonPortOccupied() bool {
	connection, err := net.DialTimeout("tcp", daemonListenAddress(DefaultPort), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = connection.Close()
	return true
}

func randomDaemonNonce() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes[:])
}
