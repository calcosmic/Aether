//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
)

func configureOracleBackgroundProcess(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
