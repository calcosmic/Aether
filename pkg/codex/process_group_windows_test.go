//go:build windows

package codex

import "testing"

func TestWorkerSysProcAttrWindowsNoop(t *testing.T) {
	if attr := workerSysProcAttr(); attr != nil {
		t.Fatalf("workerSysProcAttr() = %+v, want nil on Windows", attr)
	}
}
