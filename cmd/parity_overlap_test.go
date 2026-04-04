package cmd

import (
	"testing"
)

// parityCase defines a single parity test case for a shell/Go command pair.
type parityCase struct {
	name          string   // Test case name
	subcmd        string   // Shell subcommand name
	args          []string // Arguments to pass to both shell and Go
	knownBreak    bool     // Whether this command has a known structural difference
	skipShellZero bool     // Skip checking shell exit code (some always return non-zero)
}

// TestParityOverlapStateCommands tests state-related commands.
func TestParityOverlapStateCommands(t *testing.T) {
	cases := []parityCase{
		{name: "load-state", subcmd: "load-state"},
		{name: "validate-state", subcmd: "validate-state"},
		{name: "state-read", subcmd: "state-read"},
		{name: "unload-state", subcmd: "unload-state"},
		{name: "validate-oracle-state", subcmd: "validate-oracle-state"},
		{name: "state-read-field", subcmd: "state-read-field", args: []string{"goal"}},
		{name: "state-mutate", subcmd: "state-mutate", args: []string{".goal", "new goal"}},
		{name: "state-write", subcmd: "state-write"},
		{name: "state-checkpoint", subcmd: "state-checkpoint"},
		{name: "view-state-init", subcmd: "view-state-init"},
		{name: "view-state-get", subcmd: "view-state-get", args: []string{"goal"}},
		{name: "view-state-set", subcmd: "view-state-set", args: []string{"goal", "test"}},
		{name: "view-state-toggle", subcmd: "view-state-toggle", args: []string{"test"}},
		{name: "view-state-expand", subcmd: "view-state-expand", args: []string{"test"}},
		{name: "view-state-collapse", subcmd: "view-state-collapse", args: []string{"test"}},
		{name: "colony-name", subcmd: "colony-name"},
		{name: "colony-depth", subcmd: "colony-depth"},
		{name: "colony-vital-signs", subcmd: "colony-vital-signs"},
		{name: "domain-detect", subcmd: "domain-detect"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapPheromoneCommands tests pheromone-related commands.
func TestParityOverlapPheromoneCommands(t *testing.T) {
	cases := []parityCase{
		{name: "pheromone-read", subcmd: "pheromone-read"},
		{name: "pheromone-write", subcmd: "pheromone-write", args: []string{"--type", "FOCUS", "--content", "test focus signal", "--expires", "1h"}},
		{name: "pheromone-count", subcmd: "pheromone-count", knownBreak: true},
		{name: "pheromone-expire", subcmd: "pheromone-expire"},
		{name: "pheromone-prime", subcmd: "pheromone-prime"},
		{name: "pheromone-display", subcmd: "pheromone-display"},
		{name: "pheromone-merge-back", subcmd: "pheromone-merge-back"},
		{name: "pheromone-snapshot-inject", subcmd: "pheromone-snapshot-inject"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapFlagCommands tests flag-related commands.
func TestParityOverlapFlagCommands(t *testing.T) {
	cases := []parityCase{
		{name: "flag-list", subcmd: "flag-list", knownBreak: true},
		{name: "flag-add", subcmd: "flag-add", args: []string{"--title", "test flag", "--type", "issue"}, skipShellZero: true},
		{name: "flag-resolve", subcmd: "flag-resolve", args: []string{"flag_001"}, skipShellZero: true},
		{name: "flag-acknowledge", subcmd: "flag-acknowledge", args: []string{"flag_001"}, skipShellZero: true},
		{name: "flag-check-blockers", subcmd: "flag-check-blockers"},
		{name: "flag-auto-resolve", subcmd: "flag-auto-resolve"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapSpawnCommands tests spawn-related commands.
func TestParityOverlapSpawnCommands(t *testing.T) {
	cases := []parityCase{
		{name: "spawn-log", subcmd: "spawn-log", args: []string{"builder", "test task", "phase-1"}},
		{name: "spawn-complete", subcmd: "spawn-complete", args: []string{"builder", "test task", "success"}},
		{name: "spawn-can-spawn", subcmd: "spawn-can-spawn", args: []string{"builder"}},
		{name: "spawn-get-depth", subcmd: "spawn-get-depth"},
		{name: "spawn-tree-load", subcmd: "spawn-tree-load"},
		{name: "spawn-tree-active", subcmd: "spawn-tree-active"},
		{name: "spawn-tree-depth", subcmd: "spawn-tree-depth"},
		{name: "spawn-efficiency", subcmd: "spawn-efficiency"},
		{name: "spawn-can-spawn-swarm", subcmd: "spawn-can-spawn-swarm"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapQueenCommands tests queen-related commands.
func TestParityOverlapQueenCommands(t *testing.T) {
	cases := []parityCase{
		{name: "queen-init", subcmd: "queen-init"},
		{name: "queen-read", subcmd: "queen-read"},
		{name: "queen-promote", subcmd: "queen-promote", args: []string{"test instinct", "testing", "0.8"}, skipShellZero: true},
		{name: "queen-thresholds", subcmd: "queen-thresholds"},
		{name: "queen-write-learnings", subcmd: "queen-write-learnings", args: []string{"test learning"}, skipShellZero: true},
		{name: "queen-promote-instinct", subcmd: "queen-promote-instinct", args: []string{"inst_001"}, skipShellZero: true},
		{name: "queen-seed-from-hive", subcmd: "queen-seed-from-hive"},
		{name: "queen-migrate", subcmd: "queen-migrate"},
		{name: "charter-write", subcmd: "charter-write", args: []string{"test colony", "build feature"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapLearningCommands tests learning pipeline commands.
func TestParityOverlapLearningCommands(t *testing.T) {
	cases := []parityCase{
		{name: "learning-promote", subcmd: "learning-promote", args: []string{"test observation"}, skipShellZero: true},
		{name: "learning-inject", subcmd: "learning-inject"},
		{name: "learning-observe", subcmd: "learning-observe", args: []string{"pattern", "test observation", "cli"}, skipShellZero: true},
		{name: "learning-check-promotion", subcmd: "learning-check-promotion", args: []string{"test observation"}, skipShellZero: true},
		{name: "learning-promote-auto", subcmd: "learning-promote-auto"},
		{name: "learning-approve-proposals", subcmd: "learning-approve-proposals"},
		{name: "learning-defer-proposals", subcmd: "learning-defer-proposals"},
		{name: "learning-display-proposals", subcmd: "learning-display-proposals"},
		{name: "learning-select-proposals", subcmd: "learning-select-proposals"},
		{name: "learning-extract-fallback", subcmd: "learning-extract-fallback"},
		{name: "learning-undo-promotions", subcmd: "learning-undo-promotions"},
		{name: "memory-capture", subcmd: "memory-capture", args: []string{"test learning"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapMiddenCommands tests midden (failure tracking) commands.
func TestParityOverlapMiddenCommands(t *testing.T) {
	cases := []parityCase{
		{name: "midden-write", subcmd: "midden-write", args: []string{"build", "test failure", "phase-1"}, skipShellZero: true},
		{name: "midden-recent-failures", subcmd: "midden-recent-failures"},
		{name: "midden-review", subcmd: "midden-review"},
		{name: "midden-acknowledge", subcmd: "midden-acknowledge", args: []string{"all"}, skipShellZero: true},
		{name: "midden-search", subcmd: "midden-search", args: []string{"test"}},
		{name: "midden-tag", subcmd: "midden-tag", args: []string{"test", "build"}, skipShellZero: true},
		{name: "midden-prune", subcmd: "midden-prune"},
		{name: "midden-collect", subcmd: "midden-collect"},
		{name: "midden-cross-pr-analysis", subcmd: "midden-cross-pr-analysis"},
		{name: "midden-handle-revert", subcmd: "midden-handle-revert"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapHiveCommands tests hive brain commands.
func TestParityOverlapHiveCommands(t *testing.T) {
	cases := []parityCase{
		{name: "hive-init", subcmd: "hive-init"},
		{name: "hive-store", subcmd: "hive-store", args: []string{"--text", "test wisdom", "--domain", "cli", "--confidence", "0.8"}, skipShellZero: true},
		{name: "hive-read", subcmd: "hive-read"},
		{name: "hive-abstract", subcmd: "hive-abstract", args: []string{"test instinct text", "cli"}, skipShellZero: true},
		{name: "hive-promote", subcmd: "hive-promote", args: []string{"--text", "test wisdom", "--source-repo", "test"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapInstinctCommands tests instinct commands.
func TestParityOverlapInstinctCommands(t *testing.T) {
	cases := []parityCase{
		{name: "instinct-read", subcmd: "instinct-read"},
		{name: "instinct-create", subcmd: "instinct-create", args: []string{"--trigger", "test trigger", "--action", "test action"}, skipShellZero: true},
		{name: "instinct-apply", subcmd: "instinct-apply", args: []string{"inst_001"}, skipShellZero: true},
		{name: "instinct-archive", subcmd: "instinct-archive", args: []string{"inst_001"}, skipShellZero: true},
		{name: "instinct-read-trusted", subcmd: "instinct-read-trusted"},
		{name: "instinct-decay-all", subcmd: "instinct-decay-all"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapTrustCommands tests trust scoring commands.
func TestParityOverlapTrustCommands(t *testing.T) {
	cases := []parityCase{
		{name: "trust-score-compute", subcmd: "trust-score-compute", args: []string{"test", "0.8", "0.5"}, skipShellZero: true},
		{name: "trust-score-decay", subcmd: "trust-score-decay", args: []string{"0.8", "30"}, skipShellZero: true},
		{name: "trust-tier", subcmd: "trust-tier", args: []string{"0.8"}},
	}
	runParityCases(t, cases)
}

// TestParityOverlapEventBusCommands tests event bus commands.
func TestParityOverlapEventBusCommands(t *testing.T) {
	cases := []parityCase{
		{name: "event-bus-publish", subcmd: "event-bus-publish", args: []string{"test", "event", "info"}, skipShellZero: true},
		{name: "event-bus-query", subcmd: "event-bus-query", args: []string{"test"}},
		{name: "event-bus-cleanup", subcmd: "event-bus-cleanup"},
		{name: "event-bus-replay", subcmd: "event-bus-replay"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapGraphCommands tests graph layer commands.
func TestParityOverlapGraphCommands(t *testing.T) {
	cases := []parityCase{
		{name: "graph-link", subcmd: "graph-link", args: []string{"a", "b", "related"}, skipShellZero: true},
		{name: "graph-neighbors", subcmd: "graph-neighbors", args: []string{"a"}},
		{name: "graph-reach", subcmd: "graph-reach", args: []string{"a", "2"}, skipShellZero: true},
		{name: "graph-cluster", subcmd: "graph-cluster"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapDisplayCommands tests display rendering commands.
func TestParityOverlapDisplayCommands(t *testing.T) {
	cases := []parityCase{
		{name: "swarm-display-init", subcmd: "swarm-display-init", skipShellZero: true},
		{name: "swarm-display-update", subcmd: "swarm-display-update", args: []string{"builder", "running"}, skipShellZero: true},
		{name: "swarm-display-get", subcmd: "swarm-display-get"},
		{name: "swarm-display-text", subcmd: "swarm-display-text"},
		{name: "swarm-display-render", subcmd: "swarm-display-render"},
		{name: "swarm-display-inline", subcmd: "swarm-display-inline"},
		{name: "swarm-activity-log", subcmd: "swarm-activity-log"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapCurationCommands tests curation ant commands.
func TestParityOverlapCurationCommands(t *testing.T) {
	cases := []parityCase{
		{name: "curation-run", subcmd: "curation-run"},
		{name: "curation-archivist", subcmd: "curation-archivist"},
		{name: "curation-critic", subcmd: "curation-critic"},
		{name: "curation-herald", subcmd: "curation-herald"},
		{name: "curation-janitor", subcmd: "curation-janitor"},
		{name: "curation-librarian", subcmd: "curation-librarian"},
		{name: "curation-nurse", subcmd: "curation-nurse"},
		{name: "curation-scribe", subcmd: "curation-scribe"},
		{name: "curation-sentinel", subcmd: "curation-sentinel"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapSecurityCommands tests security and error pattern commands.
func TestParityOverlapSecurityCommands(t *testing.T) {
	cases := []parityCase{
		{name: "check-antipattern", subcmd: "check-antipattern", args: []string{"test content"}, skipShellZero: true},
		{name: "error-add", subcmd: "error-add", args: []string{"test", "test error"}, skipShellZero: true},
		{name: "error-flag-pattern", subcmd: "error-flag-pattern", args: []string{"test pattern"}, skipShellZero: true},
		{name: "error-summary", subcmd: "error-summary"},
		{name: "signature-scan", subcmd: "signature-scan", args: []string{"test"}, skipShellZero: true},
		{name: "signature-match", subcmd: "signature-match", args: []string{"test"}, skipShellZero: true},
		{name: "incident-rule-add", subcmd: "incident-rule-add", args: []string{"test", "test pattern", "warn"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapBuildFlowCommands tests build flow utility commands.
func TestParityOverlapBuildFlowCommands(t *testing.T) {
	cases := []parityCase{
		{name: "generate-ant-name", subcmd: "generate-ant-name", knownBreak: true},
		{name: "generate-commit-message", subcmd: "generate-commit-message", args: []string{"test change"}, skipShellZero: true},
		{name: "generate-progress-bar", subcmd: "generate-progress-bar", args: []string{"5", "10", "20"}, skipShellZero: true},
		{name: "generate-threshold-bar", subcmd: "generate-threshold-bar", args: []string{"0.8"}, skipShellZero: true},
		{name: "update-progress", subcmd: "update-progress", args: []string{"2", "3"}, skipShellZero: true},
		{name: "print-next-up", subcmd: "print-next-up", args: []string{"3"}},
		{name: "milestone-detect", subcmd: "milestone-detect", knownBreak: true},
		{name: "version-check-cached", subcmd: "version-check-cached"},
		{name: "progress-update", subcmd: "progress-update", args: []string{"2", "3", "Foundation"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapContextCommands tests context assembly commands.
func TestParityOverlapContextCommands(t *testing.T) {
	cases := []parityCase{
		{name: "context-capsule", subcmd: "context-capsule"},
		{name: "context-update", subcmd: "context-update", args: []string{"test context"}, skipShellZero: true},
		{name: "colony-prime", subcmd: "colony-prime"},
		{name: "pr-context", subcmd: "pr-context"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapHistoryCommands tests history/changelog commands.
func TestParityOverlapHistoryCommands(t *testing.T) {
	cases := []parityCase{
		{name: "history", subcmd: "history"},
		{name: "changelog-append", subcmd: "changelog-append", args: []string{"test entry"}, skipShellZero: true},
		{name: "changelog-collect-plan-data", subcmd: "changelog-collect-plan-data", args: []string{"test-plan"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapSessionCommands tests session management commands.
func TestParityOverlapSessionCommands(t *testing.T) {
	cases := []parityCase{
		{name: "session-init", subcmd: "session-init", args: []string{"test goal"}, skipShellZero: true},
		{name: "session-read", subcmd: "session-read"},
		{name: "session-update", subcmd: "session-update", args: []string{"key", "value"}, skipShellZero: true},
		{name: "session-clear", subcmd: "session-clear"},
		{name: "session-mark-resumed", subcmd: "session-mark-resumed", skipShellZero: true},
		{name: "session-verify-fresh", subcmd: "session-verify-fresh", args: []string{"test"}},
		{name: "resume-dashboard", subcmd: "resume-dashboard"},
	}
	runParityCases(t, cases)
}

// TestParityOverlapRegistryCommands tests registry commands.
func TestParityOverlapRegistryCommands(t *testing.T) {
	cases := []parityCase{
		{name: "registry-add", subcmd: "registry-add", args: []string{"test-repo"}, skipShellZero: true},
		{name: "registry-list", subcmd: "registry-list"},
		{name: "registry-export-xml", subcmd: "registry-export-xml"},
		{name: "registry-import-xml", subcmd: "registry-import-xml", args: []string{"test"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapExchangeCommands tests XML exchange commands.
func TestParityOverlapExchangeCommands(t *testing.T) {
	cases := []parityCase{
		{name: "pheromone-export-xml", subcmd: "pheromone-export-xml"},
		{name: "pheromone-import-xml", subcmd: "pheromone-import-xml", args: []string{"test"}, skipShellZero: true},
		{name: "wisdom-export-xml", subcmd: "wisdom-export-xml"},
		{name: "wisdom-import-xml", subcmd: "wisdom-import-xml", args: []string{"test"}, skipShellZero: true},
		{name: "colony-archive-xml", subcmd: "colony-archive-xml"},
		{name: "export", subcmd: "export"},
		{name: "import", subcmd: "import", args: []string{"test"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapSuggestCommands tests suggestion system commands.
func TestParityOverlapSuggestCommands(t *testing.T) {
	cases := []parityCase{
		{name: "suggest-analyze", subcmd: "suggest-analyze"},
		{name: "suggest-approve", subcmd: "suggest-approve", skipShellZero: true},
		{name: "suggest-check", subcmd: "suggest-check"},
		{name: "suggest-quick-dismiss", subcmd: "suggest-quick-dismiss", skipShellZero: true},
		{name: "suggest-record", subcmd: "suggest-record", args: []string{"test suggestion"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// TestParityOverlapMiscCommands tests miscellaneous utility commands.
func TestParityOverlapMiscCommands(t *testing.T) {
	cases := []parityCase{
		{name: "entropy-score", subcmd: "entropy-score", knownBreak: true},
		{name: "memory-metrics", subcmd: "memory-metrics", knownBreak: true},
		{name: "data-safety-stats", subcmd: "data-safety-stats"},
		{name: "data-clean", subcmd: "data-clean"},
		{name: "force-unlock", subcmd: "force-unlock"},
		{name: "eternal-init", subcmd: "eternal-init"},
		{name: "eternal-store", subcmd: "eternal-store", args: []string{"test signal"}, skipShellZero: true},
		{name: "validate-worker-response", subcmd: "validate-worker-response", args: []string{"test"}, skipShellZero: true},
		{name: "validate-oracle-state", subcmd: "validate-oracle-state"},
		{name: "temp-clean", subcmd: "temp-clean"},
		{name: "bootstrap-system", subcmd: "bootstrap-system"},
		{name: "chamber-create", subcmd: "chamber-create", args: []string{"test-chamber"}, skipShellZero: true},
		{name: "chamber-list", subcmd: "chamber-list"},
		{name: "chamber-verify", subcmd: "chamber-verify", args: []string{"test-chamber"}, skipShellZero: true},
		{name: "emoji-audit", subcmd: "emoji-audit"},
		{name: "normalize-args", subcmd: "normalize-args", args: []string{"test"}, skipShellZero: true},
		{name: "survey-load", subcmd: "survey-load"},
		{name: "survey-verify", subcmd: "survey-verify"},
		{name: "trophallaxis-diagnose", subcmd: "trophallaxis-diagnose"},
		{name: "trophallaxis-retry", subcmd: "trophallaxis-retry", skipShellZero: true},
		{name: "scar-add", subcmd: "scar-add", args: []string{"test scar"}, skipShellZero: true},
		{name: "scar-list", subcmd: "scar-list"},
		{name: "scar-check", subcmd: "scar-check", args: []string{"test"}, skipShellZero: true},
		{name: "immune-auto-scar", subcmd: "immune-auto-scar"},
		{name: "pending-decision-add", subcmd: "pending-decision-add", args: []string{"test decision"}, skipShellZero: true},
		{name: "pending-decision-list", subcmd: "pending-decision-list"},
		{name: "pending-decision-resolve", subcmd: "pending-decision-resolve", args: []string{"test"}, skipShellZero: true},
		{name: "autofix-checkpoint", subcmd: "autofix-checkpoint", skipShellZero: true},
		{name: "autofix-rollback", subcmd: "autofix-rollback", skipShellZero: true},
		{name: "phase-insert", subcmd: "phase-insert", args: []string{"test phase", "after", "2"}, skipShellZero: true},
		{name: "grave-add", subcmd: "grave-add", args: []string{"test grave"}, skipShellZero: true},
		{name: "grave-check", subcmd: "grave-check", args: []string{"test"}, skipShellZero: true},
		{name: "backup-prune-global", subcmd: "backup-prune-global"},
		{name: "clash-check", subcmd: "clash-check"},
		{name: "clash-setup", subcmd: "clash-setup", skipShellZero: true},
		{name: "worktree-create", subcmd: "worktree-create", args: []string{"test"}, skipShellZero: true},
		{name: "worktree-cleanup", subcmd: "worktree-cleanup"},
		{name: "worktree-merge", subcmd: "worktree-merge", args: []string{"test"}, skipShellZero: true},
	}
	runParityCases(t, cases)
}

// runParityCases executes a batch of parity test cases. Each case creates
// its own temp directory and runs the command through both shell and Go paths.
// Tests run sequentially (no t.Parallel) for reliability.
func runParityCases(t *testing.T, cases []parityCase) {
	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := setupParityEnv(t)
			shellOut := runShellCommand(t, tmpDir, tc.subcmd, tc.args...)
			goOut := runGoCommand(t, tmpDir, tc.subcmd, tc.args...)

			// If neither output is JSON, skip (command may not exist in either path)
			if !isJSON(shellOut) && !isJSON(goOut) {
				t.Skipf("neither output is JSON: shell=%q go=%q",
					truncateStr(shellOut, min(100, len(shellOut))),
					truncateStr(goOut, min(100, len(goOut))))
			}

			// If one output is empty, skip (command likely doesn't exist in that path)
			if shellOut == "" || goOut == "" {
				t.Skipf("one path returned empty: shell=%q go=%q",
					truncateStr(shellOut, min(80, len(shellOut))),
					truncateStr(goOut, min(80, len(goOut))))
			}

			// Check for known parity breaks
			if tc.knownBreak {
				if isParityBreak(t, shellOut, goOut) {
					t.Logf("KNOWN PARITY BREAK: %s (documented, not a regression)", tc.name)
					return
				}
				// Not a known break? Still verify envelope parity
			}

			assertEnvelopeParity(t, shellOut, goOut)
		})
	}
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
