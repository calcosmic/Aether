//go:build windows

package cmd

import (
	"os"
	"os/exec"
)

func configureVerificationCommandProcessGroup(cmd *exec.Cmd) {
	_ = cmd
}

func terminateVerificationCommandProcessGroup(pid int) {
	if pid <= 0 {
		return
	}
	process, err := os.FindProcess(pid)
	if err == nil {
		_ = process.Kill()
	}
}
