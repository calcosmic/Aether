//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
	"time"
)

func configureVerificationCommandProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateVerificationCommandProcessGroup(pid int) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	time.Sleep(250 * time.Millisecond)
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
