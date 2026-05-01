package cmd

import (
	"os"
	"strings"
	"sync/atomic"
)

var workerCleanupHandlerInstalled atomic.Bool

func runningInGoTest() bool {
	exe := strings.TrimSpace(os.Args[0])
	return strings.HasSuffix(exe, ".test") || strings.Contains(exe, ".test/")
}
