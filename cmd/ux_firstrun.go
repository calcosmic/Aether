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

	// Through writeVisualOutput so the commands are named for the platform the
	// reader is on. This banner is the very first thing a new user sees; naming
	// commands they cannot type is the worst possible first impression.
	writeVisualOutput(stdout, renderWelcomeBanner())

	// Create marker file (non-critical, ignore errors)
	_ = os.WriteFile(markerPath, []byte(""), 0600)
}

// renderWelcomeBanner returns the first-run welcome banner text.
//
// The command column is padded from the resolved names rather than written
// with hand-counted spaces: `/ant-lay-eggs` and `aether lay-eggs` differ in
// width, so fixed padding aligns on exactly one platform and looks broken on
// the other.
func renderWelcomeBanner() string {
	platform := detectPlatform()
	rows := []struct {
		command     string
		description string
	}{
		{platformCommandName("lay-eggs", platform), "Set up Aether in this repo"},
		{platformCommandName("init", platform) + ` "your goal"`, "Start a colony with a goal"},
		{platformCommandName("status", platform), "Check on your colony"},
	}

	width := 0
	for _, row := range rows {
		if len(row.command) > width {
			width = len(row.command)
		}
	}

	var b strings.Builder
	b.WriteString(renderBanner("\U0001F41C", "Welcome to Aether"))
	b.WriteString(visualDividerStr())
	b.WriteString("Aether manages your development colony -- a team of AI workers that plan, build, and verify code together.\n")
	b.WriteString("To get started, set up this repo and create your first colony:\n")
	b.WriteString("\n")
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("  %-*s  %s\n", width, row.command, row.description))
	}
	b.WriteString("\n")
	return b.String()
}
