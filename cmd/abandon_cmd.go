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

// abandonCmd discards a colony that is not worth finishing.
//
// This was reachable before only as `aether init "<new goal>" --confirm-reinit`
// — a flag on a different verb, discoverable by reading source. Deciding a goal
// is not worth finishing is an ordinary thing to do, and the ordinary thing
// needed a name. Sealing was the only documented route, which runs the whole
// completion ceremony over work being discarded and promotes its instincts into
// the cross-colony hive: abandoned work teaching every other project.
//
// Two steps on purpose. Without --confirm it prints what would be lost and
// stops, so the destructive form is never the first thing anyone types.
var abandonCmd = &cobra.Command{
	Use:          "abandon",
	Short:        "Discard the current colony and start fresh",
	Long:         `Discard a colony that is not worth finishing. Without --confirm it previews what would be lost. The state is backed up to .aether/data/backups/ and can be restored.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         runAbandon,
}

func init() {
	rootCmd.AddCommand(abandonCmd)
	abandonCmd.Flags().Bool("confirm", false, "Actually discard the colony (a timestamped backup is written first)")
}

func runAbandon(cmd *cobra.Command, args []string) error {
	if store == nil {
		outputErrorMessage("no store initialized")
		return nil
	}

	state, err := loadActiveColonyState()
	if err != nil {
		outputError(1, colonyStateLoadMessage(err), nil)
		return nil
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" || state.State == colony.StateIDLE {
		outputWorkflow(
			map[string]interface{}{"abandoned": false, "reason": "no_active_colony"},
			renderAbandonNothingToDoVisual(),
		)
		return nil
	}

	dataDirForSummary := store.BasePath()
	aetherRootForSummary := resolveAetherRoot()
	summary := abandonColonySummary(state)
	summary["worker_workspaces_with_work"] = countWorktreesHoldingWork(aetherRootForSummary, dataDirForSummary)
	confirmed, _ := cmd.Flags().GetBool("confirm")
	if !confirmed {
		summary["abandoned"] = false
		summary["confirm_required"] = true
		outputWorkflow(summary, renderAbandonPreviewVisual(summary))
		return nil
	}

	dataDir := store.BasePath()
	aetherRoot := resolveAetherRoot()

	// The backup is mandatory, not best-effort. A confirmation that silently
	// destroyed the only copy of the colony's history would be worse than the
	// refusal it replaced.
	backupPath, err := backupColonyStateForAbandon(dataDir)
	if err != nil {
		outputError(1, fmt.Sprintf("%v — refusing to discard the colony without a backup", err), nil)
		return nil
	}

	reset := resetColonyStateForEntomb(state)
	if err := store.SaveJSON("COLONY_STATE.json", reset); err != nil {
		outputError(1, fmt.Sprintf("failed to reset colony state: %v (the previous state is preserved at %s)", err, backupPath), nil)
		return nil
	}
	if err := clearActiveColonyRuntimeFiles(aetherRoot, dataDir); err != nil {
		// Non-fatal: the colony is already reset, and leftover runtime files are
		// stale rather than dangerous. Say so instead of failing the command.
		summary["cleanup_warning"] = err.Error()
	}

	summary["abandoned"] = true
	summary["backup_path"] = displayDataPath(filepath.Join("backups", filepath.Base(backupPath)))
	outputWorkflow(summary, renderAbandonedVisual(summary))
	return nil
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
		b.WriteString("Worker workspaces that still hold unsaved or unmerged work will be kept, not deleted, even after --confirm -- run `aether recover` afterward to see them.\n")
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
