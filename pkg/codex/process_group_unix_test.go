//go:build !windows

package codex

import "testing"

func TestWorkerSysProcAttrSetsProcessGroup(t *testing.T) {
	attr := workerSysProcAttr()
	if attr == nil {
		t.Fatal("workerSysProcAttr() = nil, want process group attrs")
	}
	if !attr.Setpgid {
		t.Fatal("workerSysProcAttr().Setpgid = false, want true")
	}
}
