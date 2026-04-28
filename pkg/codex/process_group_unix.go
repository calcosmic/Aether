//go:build !windows

package codex

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func workerSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func terminateWorkerProcess(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}

func killWorkerProcess(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

func workerProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	cmd := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	status := strings.TrimSpace(string(output))
	return status != "" && !strings.HasPrefix(status, "Z")
}

func workerProcessCommandLine(pid int) string {
	if pid <= 0 {
		return ""
	}
	cmd := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
