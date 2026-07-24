package figmaasset

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"
)

func TestOwnedDaemonStateMatchesCurrentProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process start-time assertion is covered by the Windows runtime path")
	}
	t.Setenv("HOME", t.TempDir())

	state, err := writeDaemonState(os.Getpid())
	if err != nil {
		t.Fatalf("writeDaemonState() error = %v", err)
	}
	got, owned, err := ownedDaemonState()
	if err != nil {
		t.Fatalf("ownedDaemonState() error = %v", err)
	}
	if !owned {
		t.Fatal("ownedDaemonState() owned = false, want true")
	}
	if got != state {
		t.Fatalf("ownedDaemonState() = %#v, want %#v", got, state)
	}
}

func TestOwnedDaemonStateRejectsMismatchedProcessIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process start-time assertion is covered by the Windows runtime path")
	}
	t.Setenv("HOME", t.TempDir())

	state, err := writeDaemonState(os.Getpid())
	if err != nil {
		t.Fatalf("writeDaemonState() error = %v", err)
	}
	state.ProcessStartTime = "different-process"
	payload, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(daemonStatePath(), payload, 0o600); err != nil {
		t.Fatalf("write stale state: %v", err)
	}

	_, owned, err := ownedDaemonState()
	if err != nil {
		t.Fatalf("ownedDaemonState() error = %v", err)
	}
	if owned {
		t.Fatal("ownedDaemonState() owned = true, want false")
	}
	if _, err := os.Stat(daemonStatePath()); !os.IsNotExist(err) {
		t.Fatalf("daemon state still exists, stat error = %v", err)
	}
}
