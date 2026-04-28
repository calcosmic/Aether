//go:build windows

package codex

import "syscall"

func workerSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func terminateWorkerProcess(pid int) error {
	_ = pid
	return nil
}

func killWorkerProcess(pid int) error {
	_ = pid
	return nil
}

func workerProcessExists(pid int) bool {
	_ = pid
	return false
}

func workerProcessCommandLine(pid int) string {
	_ = pid
	return ""
}
