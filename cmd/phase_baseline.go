package cmd

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// phaseBaseline records what the working tree already looked like when a phase
// started dispatching workers.
//
// Aether phases do not git-commit between themselves, so by phase 3 the tree
// legitimately contains phase 1 and phase 2's verified work as uncommitted
// changes. A verifier that runs `git status` and compares against HEAD reads
// that as the current phase touching files far outside its scope. During the
// v1.0.47 acceptance run two separate watchers returned a FAIL verdict for
// exactly this reason, and both reversed it once told what the baseline was.
type phaseBaseline struct {
	HeadSHA     string
	DirtyPaths  []string
	Unavailable string // why no baseline could be captured (not a git repo, etc.)
}

// capturePhaseBaseline snapshots HEAD and the already-modified paths. It never
// fails the build: a repo without git simply yields an explanatory baseline.
func capturePhaseBaseline(root string) phaseBaseline {
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		if strings.TrimSpace(root) != "" {
			cmd.Dir = root
		}
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}

	head, err := run("rev-parse", "HEAD")
	if err != nil || head == "" {
		return phaseBaseline{Unavailable: "no git history available for this workspace"}
	}

	status, err := run("status", "--porcelain")
	if err != nil {
		return phaseBaseline{HeadSHA: head, Unavailable: "working tree status unavailable"}
	}

	var dirty []string
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 3 {
			continue
		}
		path := strings.TrimSpace(line[2:])
		// Rename entries read "old -> new"; the new path is what a verifier sees.
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = strings.TrimSpace(path[idx+4:])
		}
		path = strings.Trim(path, `"`)
		// Aether's own state churns constantly and is never product scope.
		if path == "" || strings.HasPrefix(path, ".aether/") || strings.HasPrefix(path, ".planning/") {
			continue
		}
		dirty = append(dirty, path)
	}
	sort.Strings(dirty)
	return phaseBaseline{HeadSHA: head, DirtyPaths: dirty}
}

// phaseBaselineMaxPaths bounds the listing so a large in-flight tree cannot
// crowd out the rest of a worker's brief.
const phaseBaselineMaxPaths = 25

// renderPhaseBaselineSection renders the baseline for a verifying worker.
// Returns "" when there is nothing useful to say.
func renderPhaseBaselineSection(baseline phaseBaseline) string {
	var b strings.Builder
	b.WriteString("\n## Phase Baseline\n\n")
	if baseline.HeadSHA != "" {
		b.WriteString(fmt.Sprintf("- Workspace HEAD when this phase started: `%s`\n", shortSHA(baseline.HeadSHA)))
	}
	if baseline.Unavailable != "" {
		b.WriteString(fmt.Sprintf("- Baseline note: %s\n", baseline.Unavailable))
	}

	if len(baseline.DirtyPaths) == 0 {
		if baseline.HeadSHA == "" {
			return "" // nothing to anchor a judgement to
		}
		b.WriteString("- No uncommitted product changes existed when this phase started, so every uncommitted change you see now belongs to this phase.\n")
		return b.String()
	}

	shown := baseline.DirtyPaths
	truncated := 0
	if len(shown) > phaseBaselineMaxPaths {
		truncated = len(shown) - phaseBaselineMaxPaths
		shown = shown[:phaseBaselineMaxPaths]
	}
	b.WriteString("- These files were ALREADY modified before this phase began. They are earlier phases' verified work, not this phase's scope:\n")
	for _, path := range shown {
		b.WriteString(fmt.Sprintf("  - %s\n", path))
	}
	if truncated > 0 {
		b.WriteString(fmt.Sprintf("  - ...and %d more\n", truncated))
	}
	b.WriteString("- Aether phases do not commit between phases, so `git status` shows prior phases' work as uncommitted. Judge this phase's scope against the list above, not against HEAD.\n")
	return b.String()
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// baselineAwareCastes are the castes whose job includes judging whether the
// work stayed in scope. Builders do not need the baseline and it would only
// dilute their brief.
var baselineAwareCastes = map[string]bool{
	"watcher":    true,
	"probe":      true,
	"auditor":    true,
	"gatekeeper": true,
	"tracker":    true,
	"chaos":      true,
}
