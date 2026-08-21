package cmd

import (
	"fmt"
	"strings"
)

// renderVerificationCommandSection tells a worker how to check its own work.
//
// resolveTestCommand has existed since the gate was written, and its only
// caller was the gate itself — which runs *after* the worker has finished. So
// every builder and every reviewer worked out how to run the project's tests
// from scratch, every phase, by reading CLAUDE.md or guessing. That is one of
// the cheapest questions in the codebase to answer and one of the most
// frequently re-asked.
//
// Roughly 60 characters against a 23,700-char ceiling.
func renderVerificationCommandSection() string {
	command := strings.TrimSpace(resolveTestCommand())
	if command == "" {
		return ""
	}
	return fmt.Sprintf("\n## Verification Command\n\nRun this to check your work: `%s`\n", command)
}
