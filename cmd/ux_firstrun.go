package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// checkAndEmitFirstRun displays a welcome banner for first-time users.
// It creates a marker file to suppress future displays. The banner is
// skipped in JSON output mode and when a colony already exists.
func checkAndEmitFirstRun(dataDir string) {
	markerPath := filepath.Join(dataDir, ".welcomed")

	// Already welcomed -- skip
	if _, err := os.Stat(markerPath); err == nil {
		return
	}

	// Has a colony -- not first run
	if _, err := os.Stat(filepath.Join(dataDir, "COLONY_STATE.json")); err == nil {
		return
	}

	// Skip in JSON mode
	if !shouldRenderVisualOutput(stdout) {
		return
	}

	fmt.Fprint(stdout, renderWelcomeBanner())

	// Create marker file (non-critical, ignore errors)
	_ = os.WriteFile(markerPath, []byte(""), 0600)
}

// renderWelcomeBanner returns the first-run welcome banner text.
func renderWelcomeBanner() string {
	var b strings.Builder
	b.WriteString(renderBanner("\U0001F41C", "Welcome to Aether"))
	b.WriteString(visualDivider)
	b.WriteString("Aether manages your development colony -- a team of AI workers that plan, build, and verify code together.\n")
	b.WriteString("To get started, set up this repo and create your first colony:\n")
	b.WriteString("\n")
	b.WriteString("  aether lay-eggs          Set up Aether in this repo\n")
	b.WriteString("  aether init \"your goal\"  Start a colony with a goal\n")
	b.WriteString("  aether status            Check on your colony\n")
	b.WriteString("\n")
	return b.String()
}
