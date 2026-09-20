package cmd

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// phaseEndHiveRepoName resolves the current repository's display name the
// same way seal's promotion loop does (cmd/codex_workflow_cmds.go,
// runSealWisdomReview): the origin remote's basename, falling back to the
// working directory's basename. This is a deliberate, narrow duplication --
// the seal call site is inline inside a much larger function and extracting
// a shared helper there is out of this plan's scope -- so the two copies
// must be kept in step by hand if the resolution logic ever changes.
func phaseEndHiveRepoName() string {
	var repoName string
	if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		remote := strings.TrimSpace(string(out))
		parts := strings.Split(remote, "/")
		if len(parts) > 0 {
			repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
	}
	if repoName == "" {
		if cwd, err := os.Getwd(); err == nil {
			repoName = filepath.Base(cwd)
		}
	}
	return repoName
}

// promotePhaseEndInstinctsToHive offers every non-archived instinct at
// confidence >= 0.8 with a non-empty action to the shared cross-project
// store, at the end of every check -- not only at project close (198.1-05,
// FEED-05). It calls the exact same gate and the exact same writer seal's
// own promotion loop calls (automaticHivePromotionEnabled,
// promoteToHiveWithReference), so the one policy switch (see
// cmd/hive_policy.go) and the one hive writer govern both paths
// identically. The policy itself is never read directly here -- only through
// automaticHivePromotionEnabled(), the one gate.
//
// eligible counts every instinct that clears the confidence/action bar,
// regardless of the policy switch, mirroring seal's own hiveEligibleCount.
// promoted counts only the ones the writer actually accepted (a reinforced
// duplicate counts as promoted; a hub write failure does not).
//
// This never blocks the phase: a hub write failure is logged to stderr and
// counted as not-promoted, never returned as an error or a panic.
func promotePhaseEndInstinctsToHive(phaseID int) (eligible int, promoted int) {
	if store == nil {
		return 0, 0
	}

	file := loadInstinctFileOrEmpty(store)
	repoName := phaseEndHiveRepoName()

	for _, entry := range file.Instincts {
		if entry.Archived {
			continue
		}
		if entry.Confidence < 0.8 || entry.Action == "" {
			continue
		}
		eligible++
		if !automaticHivePromotionEnabled() {
			continue
		}
		domain := entry.Domain
		if domain == "" {
			domain = "general"
		}
		if err := promoteToHiveWithReference(entry.Action, domain, repoName, entry.Confidence, "phase-end:"+entry.ID); err != nil {
			log.Printf("phase-end: hive-promote failed for %s: %v", entry.ID, err)
			continue
		}
		promoted++
	}

	return eligible, promoted
}

// attachHivePromotionSummary stores phase-end hive promotion counts under
// result["hive_promotion"], mirroring attachConsolidationSummary's shape so
// both continue lanes report the promotion the same way.
func attachHivePromotionSummary(result map[string]interface{}, eligible, promoted int) {
	if result == nil {
		return
	}
	result["hive_promotion"] = map[string]interface{}{
		"eligible": eligible,
		"promoted": promoted,
	}
}
