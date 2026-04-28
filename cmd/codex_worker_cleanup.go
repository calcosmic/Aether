package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
)

func cleanupStaleWorkersBeforeDispatch(root string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return
	}
	result, err := codex.CleanupStaleWorkers(root)
	if err != nil {
		emitVisualProgress(fmt.Sprintf("Worker cleanup warning: %v", err))
		return
	}
	if len(result.Stale) == 0 && len(result.Failures) == 0 {
		return
	}
	emitVisualProgress(fmt.Sprintf(
		"Worker cleanup: %d stale worker(s), %d terminated, %d force-killed",
		len(result.Stale),
		len(result.Terminated),
		len(result.Killed),
	))
	if len(result.Failures) > 0 {
		emitVisualProgress(fmt.Sprintf("Worker cleanup warning: %s", strings.Join(result.Failures, "; ")))
	}
}
