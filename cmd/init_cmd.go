package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
)

// initCmd implements the `aether init` command.
// It creates the colony directory structure, COLONY_STATE.json, session.json,
// CONTEXT.md, and activity.log. It is idempotent -- if a colony is already
// initialized, it reports the existing state without overwriting.
//
// Sealed colony detection: if a sealed colony is detected, the command checks
// for uncommitted changes (in-progress seal) before allowing overwrite.
var initCmd = &cobra.Command{
	Use:   "init <goal>",
	Short: "Initialize a new colony in the current directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		scopeRaw, _ := cmd.Flags().GetString("scope")
		scope, err := colony.ParseColonyScope(scopeRaw)
		if err != nil {
			outputError(1, fmt.Sprintf("invalid scope %q (must be project or meta)", scopeRaw), nil)
			return nil
		}
		colonyModeRaw, _ := cmd.Flags().GetString("colony-mode")
		colonyMode, err := parseInitColonyMode(colonyModeRaw)
		if err != nil {
			outputError(1, fmt.Sprintf("invalid colony mode %q (must be colony or orchestrator)", colonyModeRaw), nil)
			return nil
		}

		goal := strings.TrimSpace(args[0])
		if goal == "" {
			outputError(1, "goal must not be empty", nil)
			return nil
		}

		promoteShelfRaw, _ := cmd.Flags().GetString("promote-shelf")
		dismissShelfRaw, _ := cmd.Flags().GetString("dismiss-shelf")

		dataDir := store.BasePath()
		aetherDir := filepath.Dir(dataDir)

		// Check idempotency: if COLONY_STATE.json exists, inspect it
		statePath := filepath.Join(dataDir, "COLONY_STATE.json")
		if _, err := os.Stat(statePath); err == nil {
			// Colony already initialized -- load and inspect
			var existing colony.ColonyState
			if loadErr := store.LoadJSON("COLONY_STATE.json", &existing); loadErr == nil {
				// An entombed/reset colony leaves the state scaffold in place but
				// clears the goal. Treat that as no active colony.
				if existing.Goal == nil || strings.TrimSpace(ptrStr(existing.Goal)) == "" || existing.State == colony.StateIDLE {
					goto createFreshColony
				}
				// If colony is sealed, check for in-progress seal (uncommitted changes).
				// Completion is keyed on state OR milestone: the review found a
				// state marked complete without the milestone string hit the generic
				// refusal that never mentions --confirm-reinit — and silently
				// ignored the flag when given. A COMPLETED colony carries its whole
				// history just like a sealed one.
				if existing.Milestone == "Crowned Anthill" || existing.State == colony.StateCOMPLETED {
					if sealInProgress(dataDir) {
						outputError(1, "a seal operation appears to be in progress (COLONY_STATE.json has uncommitted changes with Crowned Anthill milestone). Wait for the seal to complete, commit the seal state, or run `aether entomb` first.", nil)
						return nil
					}
					// A sealed colony's state carries its whole history — phases,
					// instincts, decisions, errors. Re-init used to overwrite it
					// silently with only a stderr note about a .bak nobody was
					// told how to restore. Destroying a colony's memory requires
					// saying so out loud.
					if confirmed, _ := cmd.Flags().GetBool("confirm-reinit"); !confirmed {
						outputError(1, fmt.Sprintf(
							"this repository has a sealed colony (goal: %q, %d phases). Re-initializing replaces its state. "+
								"The preferred path is `aether entomb` to archive it properly. "+
								"To proceed anyway, rerun with --confirm-reinit; the old state will be backed up under .aether/data/backups/ and can be restored by copying the .bak file back over COLONY_STATE.json.",
							ptrStr(existing.Goal), len(existing.Plan.Phases)), nil)
						return nil
					}
					// Confirmed — fall through; the backup below preserves the state.
				} else {
					// Active (non-sealed) colony. Abandoning one mid-flight is a
					// real thing to want — a goal turns out not to be worth
					// finishing — and there was no way to do it. Entomb requires
					// Crowned Anthill, so the only route was to seal work you had
					// just decided to bin, which runs the full ceremony over it
					// and promotes its instincts to the cross-colony hive.
					//
					// Same confirmation and same timestamped backup as the sealed
					// path: destroying a colony's memory requires saying so out
					// loud, but it must be possible to say.
					if confirmed, _ := cmd.Flags().GetBool("confirm-reinit"); !confirmed {
						outputError(1, fmt.Sprintf(
							"this repository has an active colony (goal: %q, state: %s, phase %d). "+
								"Starting a new one replaces it. If the work is finished, `aether seal` then `aether entomb` archives it properly. "+
								"To abandon it and start fresh, rerun with --confirm-reinit; the old state is backed up under .aether/data/backups/ and can be restored by copying the .bak file back over COLONY_STATE.json.",
							ptrStr(existing.Goal), existing.State, existing.CurrentPhase), nil)
						return nil
					}
					// Confirmed — fall through; the backup below preserves the state.
				}
			}
		}

	createFreshColony:
		var charter *colony.Charter
		if charterJSON, _ := cmd.Flags().GetString("charter-json"); charterJSON != "" {
			var ch colony.Charter
			if err := json.Unmarshal([]byte(charterJSON), &ch); err != nil {
				outputError(1, fmt.Sprintf("invalid charter JSON: %v", err), nil)
				return nil
			}
			if err := validateCharterFieldLength(ch); err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			charter = &ch
		}

		// --research records a pointer to work the operator already had done,
		// most often a saved Oracle run. It is stored on state rather than
		// folded into the charter: charter fields are capped at 2000 characters
		// and reach workers as hard rules, while research is evidence a worker
		// may argue with.
		researchDocs, _ := cmd.Flags().GetStringArray("research")
		researchDocs, researchErr := validateColonyResearchDocs(skillWorkspaceRoot(), researchDocs)
		if researchErr != nil {
			outputError(1, researchErr.Error(), nil)
			return nil
		}

		// Rotate trace file if it has grown too large
		if rotated, rotateErr := trace.RotateTraceFile(store, 50); rotateErr == nil && rotated {
			fmt.Fprintf(os.Stderr, "warning: rotated trace.jsonl before init\n")
		}

		now := time.Now()
		nowStr := now.Format(time.RFC3339)

		// Generate session ID: first word of goal + timestamp
		sanitizedGoal := strings.ToLower(strings.Fields(goal)[0])
		sessionID := fmt.Sprintf("%s_%d", sanitizedGoal, now.Unix())

		// Generate run ID for trace logging
		runID := fmt.Sprintf("%s_%d_%s", sanitizedGoal, now.Unix(), randomHex(4))

		// Create directory structure
		if err := os.MkdirAll(filepath.Join(aetherDir, "dreams"), 0755); err != nil {
			outputError(1, fmt.Sprintf("failed to create directory structure: %v", err), nil)
			return nil
		}

		// Clear the prior colony's conversational residue so it cannot leak
		// into the new colony's workers. This is not bookkeeping:
		// pending-decisions.json renders into every worker prompt as
		// CLARIFIED INTENT, and handoffs/worker-handoffs.json renders as
		// Previous Worker Handoffs — leaving them behind briefs workers on a
		// new goal with the previous project's decisions (RUNTIME-01, locked
		// by TestInitClearsPriorColonyDecisionResidue).
		_ = os.Remove(filepath.Join(dataDir, "session.json"))
		_ = os.Remove(filepath.Join(dataDir, "pending-decisions.json"))
		_ = os.Remove(filepath.Join(dataDir, "assumptions.json"))
		_ = os.RemoveAll(filepath.Join(dataDir, "handoffs"))

		// Backup old colony state before overwriting (sealed colony fresh-init).
		// The backup is mandatory, not best-effort: if it cannot be written, the
		// re-init stops rather than destroying the only copy of the colony's
		// history. The restore command ships in the error/result so recovery
		// never requires reading source code.
		priorStateBackup := ""
		if _, err := os.Stat(statePath); err == nil {
			backupDir := filepath.Join(dataDir, "backups")
			if err := os.MkdirAll(backupDir, 0755); err != nil {
				outputError(1, fmt.Sprintf("cannot create backup directory before re-init: %v — refusing to overwrite the previous colony state without a backup", err), nil)
				return nil
			}
			backupFile := filepath.Join(backupDir, fmt.Sprintf("COLONY_STATE.pre-init.%s.bak", time.Now().Format("20060102-150405")))
			if err := copyFile(statePath, backupFile); err != nil {
				outputError(1, fmt.Sprintf("cannot back up previous colony state: %v — refusing to overwrite without a backup", err), nil)
				return nil
			}
			priorStateBackup = backupFile
			fmt.Fprintf(os.Stderr, "backed up previous colony state to %s\nrestore with: cp %q %q\n", backupFile, backupFile, statePath)
		}

		// Check any leftover worktrees from a previous colony. Nothing here
		// is destroyed automatically (D-01) — dirty or unmerged work is
		// kept, not deleted, and every occurrence is reported (D-02).
		// gcOrphanedWorktrees itself already names the deliberate-removal
		// command per entry via reportWorktreePreservation; this summary
		// line intentionally says "aether recover" rather than repeating
		// the destructive command's own name, since TestWorktreeReapHasNoLifecycleCaller
		// (cmd/worktree_crash_safety_test.go) fails the build if this file
		// contains that literal string — a lifecycle path must not even
		// mention the destruction command by name, let alone call it.
		var wtPreserved int
		if cleaned, preserved, err := gcOrphanedWorktrees(); err == nil {
			wtPreserved = preserved
			if cleaned > 0 || preserved > 0 {
				fmt.Fprintf(os.Stderr, "worker workspaces from a previous colony: %d forgotten (already gone), %d kept because they still hold work — run `aether recover` to see them\n", cleaned, preserved)
			}
		} else {
			fmt.Fprintf(os.Stderr, "warning: could not check previous colony's worker workspaces for leftover work: %v\n", err)
		}
		// wtPreserved only counts entries gcOrphanedWorktrees actually saw,
		// and it only ever iterates state.Worktrees. A worktree created by
		// `git worktree add` but killed before its state entry was appended
		// (the exact crash window this phase exists for) is invisible to
		// that count — it looks like "nothing preserved" even though it may
		// hold uncommitted or unmerged work. Scan the directory on disk,
		// independent of state, before trusting wtPreserved == 0 (WR-06,
		// 187-VERIFICATION.md GAP-3).
		worktreesDir := filepath.Join(aetherDir, "worktrees")
		gitRoot := filepath.Dir(aetherDir)
		knownPaths := map[string]bool{}
		var wtScanState colony.ColonyState
		if loadErr := store.LoadJSON("COLONY_STATE.json", &wtScanState); loadErr == nil {
			for _, wt := range wtScanState.Worktrees {
				p := wt.Path
				if !filepath.IsAbs(p) {
					p = filepath.Join(gitRoot, p)
				}
				knownPaths[p] = true
			}
		}
		unrecorded := scanUnrecordedWorktrees(gitRoot, worktreesDir, knownPaths)
		var unrecordedUnsafe []worktreeSafety
		for _, safety := range unrecorded {
			if !safety.Safe {
				unrecordedUnsafe = append(unrecordedUnsafe, safety)
			}
		}
		if len(unrecordedUnsafe) > 0 {
			wtPreserved += len(unrecordedUnsafe)
			for _, safety := range unrecordedUnsafe {
				reportWorktreePreservation(safety, fmt.Sprintf("found on disk but not yet recorded (likely interrupted mid-creation): %s", safety.Reason))
			}
		}

		// Remove the worktrees directory entirely to ensure a clean slate,
		// but only when nothing was preserved. Removing it unconditionally
		// would silently undo every preservation gcOrphanedWorktrees just
		// made — init would become the new data-loss path the moment
		// gcOrphanedWorktrees stopped being one. Do not remove this guard;
		// doing so reintroduces the exact defect this phase was created to
		// fix.
		if wtPreserved == 0 {
			_ = os.RemoveAll(worktreesDir)
		} else {
			fmt.Fprintf(os.Stderr, "the previous colony's worker workspaces were left in place because they still hold work — run `aether recover` to see them\n")
		}

		// Clean up reviews from any prior colony
		_ = os.RemoveAll(filepath.Join(dataDir, "reviews"))

		// Create COLONY_STATE.json v3.0
		state := colony.ColonyState{
			Version:       "3.0",
			Goal:          &goal,
			Scope:         scope,
			ColonyMode:    colonyMode,
			ColonyVersion: 0,
			State:         colony.StateREADY,
			CurrentPhase:  0,
			SessionID:     &sessionID,
			RunID:         &runID,
			InitializedAt: &now,
			Plan:          colony.Plan{Phases: []colony.Phase{}},
			Memory: colony.Memory{
				PhaseLearnings: []colony.PhaseLearning{},
				Decisions:      []colony.Decision{},
				Instincts:      []colony.Instinct{},
			},
			Errors: colony.Errors{
				Records:         []colony.ErrorRecord{},
				FlaggedPatterns: []colony.FlaggedPattern{},
			},
			Signals:      []colony.Signal{},
			Graveyards:   []colony.Graveyard{},
			Events:       []string{},
			ParallelMode: colony.ModeInRepo,
		}
		state.Charter = charter
		state.ResearchDocs = researchDocs

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(1, fmt.Sprintf("failed to create COLONY_STATE.json: %v", err), nil)
			return nil
		}

		// Phase 165 gap CR-01: promotion must stay below the state save and
		// above the session build. Every refusal branch above this line
		// (empty goal, active colony, sealed colony without --confirm-reinit,
		// in-progress seal, invalid scope/colony-mode/charter JSON, failed
		// state save) returns before this ever runs, so a failed `aether init`
		// writes nothing to shelf.json. Do not move this call upward.
		shelfPromoted, shelfDismissed, shelfFailed := applyInitShelfSelections(store, promoteShelfRaw, dismissShelfRaw, goal)
		if len(shelfFailed) > 0 {
			fmt.Fprintf(os.Stderr, "warning: could not apply shelf selection(s): %s\n", strings.Join(shelfFailed, ", "))
		}

		// Ranked next-move proposals, computed from what the repo actually
		// contains — the runtime proposes, the wrapper asks, the user picks.
		// The top proposal replaces the old hardcoded "aether plan" as the
		// recorded suggestion.
		repoRoot := filepath.Dir(aetherDir)
		proposals := computeInitProposals(repoRoot, goal, priorStateBackup != "")
		suggestedNext := "aether plan"
		if len(proposals) > 0 {
			suggestedNext = proposals[0].Command
		}

		// Create session.json
		session := colony.SessionFile{
			SessionID:        sessionID,
			StartedAt:        nowStr,
			ColonyGoal:       goal,
			ColonyMode:       colonyMode,
			CurrentPhase:     0,
			CurrentMilestone: "",
			SuggestedNext:    suggestedNext,
			ActiveTodos:      promotedShelfTodos(store, goal),
			Summary:          "Colony initialized",
		}

		if err := store.SaveJSON("session.json", session); err != nil {
			outputError(1, fmt.Sprintf("failed to create session.json: %v", err), nil)
			return nil
		}

		if _, err := syncColonyArtifacts(state, colonyArtifactOptions{
			CommandName:   "init",
			SuggestedNext: suggestedNext,
			Summary:       "Colony initialized",
			HandoffTitle:  "Initialized Colony",
			WriteHandoff:  true,
		}); err != nil {
			outputError(1, fmt.Sprintf("failed to create recovery artifacts: %v", err), nil)
			return nil
		}

		// Initialize activity.log with first entry
		activityEntry := map[string]interface{}{
			"timestamp": nowStr,
			"action":    "COLONY_INITIALIZED",
			"detail":    fmt.Sprintf("goal=%q session=%s", goal, sessionID),
		}

		if err := store.AppendJSONL("activity.log", activityEntry); err != nil {
			outputError(1, fmt.Sprintf("failed to create activity.log: %v", err), nil)
			return nil
		}

		// Cross-colony bookkeeping (RECLAIM-02/09) — both NON-BLOCKING,
		// matching the seal-time hive-promotion precedent: registry and hive
		// failures warn, never stop an init. The registry entry carries the
		// domain tags that scope hive wisdom retrieval for this repo.
		registryDomains := detectColonyDomains(repoRoot)
		if _, regErr := upsertColonyRegistryEntry(repoRoot, goal, registryDomains, true); regErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not register colony in hub registry: %v\n", regErr)
		}
		hiveSeeded := 0
		if automaticHiveReadEnabled() {
			if seeded, _, _, seedErr := seedQueenFromHive(); seedErr == nil {
				hiveSeeded = seeded
			}
		}

		// Load active shelf for wrapper consumption
		shelfEntries, _ := loadActiveShelf(store)
		result := map[string]interface{}{
			"state":               string(colony.StateREADY),
			"goal":                goal,
			"scope":               string(scope),
			"colony_mode":         string(colonyMode),
			"version":             "3.0",
			"phase":               0,
			"session":             sessionID,
			"data_dir":            dataDir,
			"shelf_backlog":       shelfEntries,
			"shelf_backlog_count": len(shelfEntries),
			"shelf_promoted":      shelfPromoted,
			"shelf_dismissed":     shelfDismissed,
			"shelf_failed":        shelfFailed,
		}
		if len(researchDocs) > 0 {
			// Echo what was accepted so the operator can see the pointer landed
			// rather than trusting that it did.
			result["research_docs"] = researchDocs
		}
		result["registry_domains"] = registryDomains
		result["hive_seeded"] = hiveSeeded
		result["proposals"] = proposals
		result["suggested_next"] = suggestedNext
		if priorStateBackup != "" {
			result["prior_state_backup"] = priorStateBackup
			result["prior_state_restore"] = fmt.Sprintf("cp %q %q", priorStateBackup, statePath)
		}
		// One closing answer: the card the owner reads and the fields a wrapper
		// reads come from the same resolve, so they cannot name different
		// commands (Phase 197 plan 04).
		closeLifecycleCommand(result, "init", "", "")
		outputWorkflow(result, renderInitVisual(goal, string(scope), sessionID, dataDir, charter, hiveSeeded, proposals, researchDocs...))
		return nil
	},
}

// ptrStr safely dereferences a *string, returning "" if nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseInitColonyMode(raw string) (colony.ColonyMode, error) {
	mode := colony.ColonyMode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return colony.ColonyModeColony, nil
	}
	if !mode.Valid() {
		return "", colony.ErrInvalidColonyMode
	}
	return mode, nil
}

func init() {
	initCmd.Flags().String("scope", string(colony.ScopeProject), "Colony scope: project or meta")
	initCmd.Flags().String("colony-mode", string(colony.ColonyModeColony), "Colony mode: colony or orchestrator")
	initCmd.Flags().String("charter-json", "", "Approved charter data as JSON string")
	initCmd.Flags().StringArray("research", nil, "Repository-relative path to a research document this colony should be planned from, e.g. a saved Oracle run under .aether/research (repeatable)")
	initCmd.Flags().Bool("confirm-reinit", false, "Confirm replacing an existing colony's state, sealed or active (a timestamped backup is written to .aether/data/backups/)")
	initCmd.Flags().String("promote-shelf", "", "Comma-separated shelf entry IDs to promote into this colony as todos")
	initCmd.Flags().String("dismiss-shelf", "", "Comma-separated shelf entry IDs to dismiss from the backlog")
	rootCmd.AddCommand(initCmd)
}

// sealInProgress checks whether COLONY_STATE.json has uncommitted changes
// that indicate a seal is in progress (working tree has Crowned Anthill milestone
// but the HEAD commit does not). This prevents a new colony init from overwriting
// a seal that hasn't been committed yet.
func sealInProgress(dataDir string) bool {
	// Resolve the repo root from dataDir (strip .aether/data to get project root)
	projectRoot := filepath.Dir(filepath.Dir(dataDir))

	// Check if we're in a git repo
	gitCmd := exec.Command("git", "rev-parse", "--git-dir")
	gitCmd.Dir = projectRoot
	if _, err := gitCmd.CombinedOutput(); err != nil {
		return false
	}

	// Get the repo root so we can compute a path relative to it
	topCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	topCmd.Dir = projectRoot
	rootOut, err := topCmd.CombinedOutput()
	if err != nil {
		return false
	}
	gitRoot := strings.TrimSpace(string(rootOut))

	// Build path relative to git root.
	// Use gitRoot (resolved by git) instead of dataDir to avoid
	// macOS /var -> /private/var symlink mismatch.
	stateRelPath := filepath.Join(".aether", "data", "COLONY_STATE.json")

	// Check if COLONY_STATE.json has uncommitted changes
	diffCmd := exec.Command("git", "diff", "--name-only", "HEAD", "--", stateRelPath)
	diffCmd.Dir = gitRoot
	diffOut, err := diffCmd.CombinedOutput()
	if err != nil || len(strings.TrimSpace(string(diffOut))) == 0 {
		return false
	}

	// The working tree differs from HEAD — check if HEAD has the seal milestone
	showCmd := exec.Command("git", "show", "HEAD:"+stateRelPath)
	showCmd.Dir = gitRoot
	showOut, err := showCmd.CombinedOutput()
	if err != nil {
		// File doesn't exist in HEAD — it's a new file, not an in-progress seal
		return false
	}

	// Check if the committed version has Crowned Anthill
	committed := string(showOut)
	if !strings.Contains(committed, "Crowned Anthill") {
		// HEAD does NOT have the seal — the seal is uncommitted
		return true
	}

	return false
}

// validateCharterFieldLength checks that no charter field exceeds 2000 characters.
// This prevents unreasonably large state files from user-controlled input (T-72-01).
func validateCharterFieldLength(ch colony.Charter) error {
	const maxLen = 2000
	fields := []struct {
		name  string
		value string
	}{
		{"intent", ch.Intent},
		{"vision", ch.Vision},
		{"governance", ch.Governance},
		{"goals", ch.Goals},
		{"tech_stack", ch.TechStack},
		{"key_risks", ch.KeyRisks},
		{"constraints", ch.Constraints},
	}
	for _, f := range fields {
		if len(f.value) > maxLen {
			return fmt.Errorf("charter field %q exceeds %d characters (%d)", f.name, maxLen, len(f.value))
		}
	}
	return nil
}
