package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const legacyAbandonMigrationSchemaVersion = "legacy-abandon-migration/v1"

type legacyAbandonMigrationResult struct {
	SchemaVersion string                      `json:"schema_version"`
	Command       string                      `json:"command"`
	OutcomeKind   colony.OutcomeKind          `json:"outcome_kind"`
	Explanation   string                      `json:"explanation"`
	OwnerAction   string                      `json:"owner_action"`
	Invoked       bool                        `json:"invoked"`
	StateEffect   colony.LifecycleStateEffect `json:"state_effect"`
	NextAction    string                      `json:"next_action"`
}

// abandonCmd remains parseable only to explain the direct-owner forced-close
// route. It never invokes that route and never changes colony state.
var abandonCmd = &cobra.Command{
	Use:          "abandon",
	Short:        "Legacy forced-close migration route",
	Hidden:       true,
	Args:         cobra.NoArgs,
	Annotations:  map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true", "aether.io/internal-only": "true"},
	SilenceUsage: true,
	RunE:         runAbandon,
}

func init() {
	rootCmd.AddCommand(abandonCmd)
	abandonCmd.Flags().Bool("confirm", false, "legacy compatibility flag; no state is discarded")
	_ = abandonCmd.Flags().MarkHidden("confirm")
}

func runAbandon(_ *cobra.Command, _ []string) error {
	result := legacyAbandonMigrationResult{
		SchemaVersion: legacyAbandonMigrationSchemaVersion,
		Command:       "abandon",
		OutcomeKind:   colony.OutcomeKindNoChange,
		Explanation:   "The standalone abandon command is retired. An incomplete colony can be closed only by the direct owner through the explicit forced-seal path.",
		OwnerAction:   `aether seal --force --reason "why this incomplete colony is being closed"`,
		Invoked:       false,
		StateEffect:   colony.LifecycleStateEffectNone,
		NextAction:    "aether status",
	}
	outputWorkflow(result, renderLegacyAbandonMigration(result))
	return nil
}

func renderLegacyAbandonMigration(result legacyAbandonMigrationResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("abandon"), "Abandon Retired"))
	b.WriteString(visualDividerStr())
	b.WriteString(result.Explanation)
	b.WriteString("\nThis compatibility command did not invoke the forced close and did not change state.\n")
	b.WriteString(renderNextUp(
		fmt.Sprintf("If you are the direct owner, run `%s` yourself with a specific reason.", result.OwnerAction),
		"Run `aether status` first to review the durable colony evidence.",
	))
	return b.String()
}

// abandonColonySummary describes what discarding this colony would cost, so the
// preview is specific rather than a generic "are you sure".
func abandonColonySummary(state colony.ColonyState) map[string]interface{} {
	completed := 0
	for _, phase := range state.Plan.Phases {
		if phase.Status == colony.PhaseCompleted {
			completed++
		}
	}
	return map[string]interface{}{
		"goal":             ptrStr(state.Goal),
		"state":            string(state.State),
		"current_phase":    state.CurrentPhase,
		"phases_total":     len(state.Plan.Phases),
		"phases_completed": completed,
		"instincts":        len(state.Memory.Instincts),
		"decisions":        len(state.Memory.Decisions),
	}
}

// countWorktreesHoldingWork reports how many worker workspaces (git
// worktrees) under this colony currently hold uncommitted or unmerged work,
// so abandon's confirmation preview can say so in plain language before a
// human types --confirm. A destructive confirmation that hides the thing
// being destroyed is not informed consent -- 187-VERIFICATION.md GAP-5 found
// abandon's preview never mentioned worktrees at all.
//
// This checks BOTH worktrees tracked in COLONY_STATE.json (via
// worktreeDestructionSafety) and worktrees present on disk but not yet
// recorded (via scanUnrecordedWorktrees, the same crash window GAP-3 closed
// for init) -- state.Worktrees is still populated here since this runs
// before the colony state reset.
func countWorktreesHoldingWork(aetherRoot, dataDir string) int {
	count := 0
	knownPaths := map[string]bool{}

	var state colony.ColonyState
	if store != nil {
		if err := store.LoadJSON("COLONY_STATE.json", &state); err == nil {
			for _, wt := range state.Worktrees {
				p := wt.Path
				if !filepath.IsAbs(p) {
					p = filepath.Join(aetherRoot, p)
				}
				knownPaths[p] = true
				if safety := worktreeDestructionSafety(aetherRoot, wt); !safety.Safe {
					count++
				}
			}
		}
	}

	worktreesDir := filepath.Join(aetherRoot, ".aether", "worktrees")
	for _, safety := range scanUnrecordedWorktrees(aetherRoot, worktreesDir, knownPaths) {
		if !safety.Safe {
			count++
		}
	}
	return count
}

func backupColonyStateForAbandon(dataDir string) (string, error) {
	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
	raw, err := os.ReadFile(statePath)
	if err != nil {
		return "", fmt.Errorf("cannot read colony state to back it up: %w", err)
	}
	backupDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create backup directory: %w", err)
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("COLONY_STATE.pre-abandon.%s.bak", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(backupPath, raw, 0600); err != nil {
		return "", fmt.Errorf("cannot write backup: %w", err)
	}
	return backupPath, nil
}

func renderAbandonNothingToDoVisual() string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("abandon"), "Abandon Colony"))
	b.WriteString(visualDividerStr())
	b.WriteString("There is no active colony in this repo, so there is nothing to discard.\n")
	b.WriteString(renderNextUp(
		"Run `aether init \"your goal\"` to start one.",
		"Run `aether status` to check what this repo currently has.",
	))
	return b.String()
}

func renderAbandonPreviewVisual(summary map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("abandon"), "Abandon Colony"))
	b.WriteString(visualDividerStr())
	b.WriteString("This would discard the current colony:\n\n")
	b.WriteString(fmt.Sprintf("  Goal:      %s\n", stringValue(summary["goal"])))
	b.WriteString(fmt.Sprintf("  State:     %s\n", stringValue(summary["state"])))
	b.WriteString(fmt.Sprintf("  Progress:  %d of %d phases completed\n",
		intValue(summary["phases_completed"]), intValue(summary["phases_total"])))
	if learned := intValue(summary["instincts"]) + intValue(summary["decisions"]); learned > 0 {
		b.WriteString(fmt.Sprintf("  Learned:   %d instinct(s) and decision(s) recorded\n", learned))
	}
	if withWork := intValue(summary["worker_workspaces_with_work"]); withWork > 0 {
		b.WriteString(fmt.Sprintf("  Warning:   %d worker workspace(s) still hold unsaved or unmerged work\n", withWork))
	}
	b.WriteString("\nThe state is backed up first and can be restored, but the colony stops here.\n")
	if withWork := intValue(summary["worker_workspaces_with_work"]); withWork > 0 {
		b.WriteString("Worker workspaces that still hold unsaved or unmerged work will be kept, not deleted. Run `aether maintenance recovery-inspect` to inspect preserved work; run `aether resume` only if you choose to restore a runnable lifecycle.\n")
	}
	b.WriteString("If this work is actually finished, `aether seal` then `aether entomb` archives it properly instead.\n")
	b.WriteString(renderNextUp(
		"Run `aether abandon --confirm` to discard it and start fresh.",
		"Run `aether status` to look at the colony again before deciding.",
	))
	return b.String()
}

func renderAbandonedVisual(summary map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("abandon"), "Colony Abandoned"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Discarded: %s\n", stringValue(summary["goal"])))
	if backup := strings.TrimSpace(stringValue(summary["backup_path"])); backup != "" {
		b.WriteString(fmt.Sprintf("Backed up to: %s\n", backup))
		b.WriteString("Restore by copying that file back over .aether/data/COLONY_STATE.json.\n")
	}
	if warning := strings.TrimSpace(stringValue(summary["cleanup_warning"])); warning != "" {
		b.WriteString(fmt.Sprintf("Note: some runtime files could not be cleared (%s). They are stale, not harmful.\n", warning))
	}
	b.WriteString(renderNextUp(
		"Run `aether init \"your new goal\"` to start a fresh colony.",
		"Run `aether status` to confirm the repo is clear.",
	))
	return b.String()
}
