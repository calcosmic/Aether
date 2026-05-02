//go:build !windows

package codex

import (
	"testing"
)

func TestWorkerSysProcAttrSetsProcessGroup(t *testing.T) {
	attr := workerSysProcAttr()
	if attr == nil {
		t.Fatal("workerSysProcAttr() = nil, want process group attrs")
	}
	if !attr.Setpgid {
		t.Fatal("workerSysProcAttr().Setpgid = false, want true")
	}
}

// TestProcessGroupTerminateKillSignatures verifies the terminate and kill
// functions have correct signatures and can be called (actual process killing
// is tested via mocks in process_tracker_test.go). These functions call
// syscall.Kill(-pid, signal) which sends signals to the process group.
func TestProcessGroupTerminateKillSignatures(t *testing.T) {
	// The functions terminateWorkerProcess and killWorkerProcess exist
	// and have the correct signature (func(int) error). They are called
	// via the mockable function variables terminateWorkerFunc and
	// killWorkerFunc. Actual kill behavior is tested through the
	// ProcessTracker tests with mocked functions.

	// Verify the function variables are assigned (not nil)
	if terminateWorkerFunc == nil {
		t.Fatal("terminateWorkerFunc is nil, expected syscall.Kill(-pid, SIGTERM)")
	}
	if killWorkerFunc == nil {
		t.Fatal("killWorkerFunc is nil, expected syscall.Kill(-pid, SIGKILL)")
	}
}
