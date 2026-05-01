//go:build !windows

package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/calcosmic/Aether/pkg/codex"
)

func setupWorkerCleanupHandler() {
	if runningInGoTest() {
		return
	}
	if !workerCleanupHandlerInstalled.CompareAndSwap(false, true) {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		signal.Stop(signals)
		codex.GlobalProcessTracker().KillAll("")
		if unixSig, ok := sig.(syscall.Signal); ok {
			_ = syscall.Kill(os.Getpid(), unixSig)
		}
		os.Exit(128)
	}()
}
