package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var flagAddCmd = &cobra.Command{
	Use:   "flag-add [title] | flag-add <type> <title> <description> [source] [phase]",
	Short: "Create a new flag",
	Args:  cobra.MaximumNArgs(5),
	Aliases: []string{
		"flag",
		"flag-create",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		title, _ := cmd.Flags().GetString("title")
		if title == "" && len(args) == 1 {
			title = args[0]
		}
		severity, _ := cmd.Flags().GetString("severity")
		if severity == "" {
			severity = "high"
		}
		source, _ := cmd.Flags().GetString("source")
		flagType, _ := cmd.Flags().GetString("type")
		description, _ := cmd.Flags().GetString("description")
		phaseNum, _ := cmd.Flags().GetInt("phase")

		if len(args) >= 3 {
			if flagType == "" {
				flagType = args[0]
			}
			if title == "" {
				title = args[1]
			}
			if description == "" {
				description = args[2]
			}
			if source == "" && len(args) >= 4 {
				source = args[3]
			}
			if phaseNum == 0 && len(args) >= 5 {
				if parsed, err := strconv.Atoi(args[4]); err == nil {
					phaseNum = parsed
				}
			}
		}
		if title == "" {
			outputError(1, "flag title is required", nil)
			return nil
		}

		if flagType == "" {
			flagType = "issue"
		}
		if description == "" {
			description = title
		}

		// Validate severity
		switch severity {
		case "critical", "high", "low":
		default:
			outputError(1, fmt.Sprintf("invalid severity %q: must be critical, high, or low", severity), nil)
			return nil
		}

		var phasePtr *int
		if phaseNum > 0 {
			phasePtr = &phaseNum
		}

		flag := colony.FlagEntry{
			ID:          generateFlagID(),
			Type:        flagType,
			Description: description,
			Source:      source,
			Phase:       phasePtr,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			Resolved:    false,
		}

		// Load existing flags
		var ff colony.FlagsFile
		loaded := false
		if err := store.LoadJSON("pending-decisions.json", &ff); err == nil {
			loaded = true
		} else if err2 := store.LoadJSON("flags.json", &ff); err2 == nil {
			loaded = true
		}
		if !loaded {
			ff = colony.FlagsFile{Decisions: []colony.FlagEntry{}}
		}
		err := updateFlagFile(&ff, func() error {
			if ff.Decisions == nil {
				ff.Decisions = []colony.FlagEntry{}
			}

			if existing, ok := matchingEnvironmentBlockedLaunchFlag(ff.Decisions, flag); ok {
				result := map[string]interface{}{
					"created": false,
					"deduped": true,
					"flag":    existing,
					"total":   len(ff.Decisions),
				}
				outputWorkflow(result, renderFlagActionVisual("flag", "Flag Already Active", result))
				return errFlagNoMutation
			}

			ff.Decisions = append(ff.Decisions, flag)

			return nil
		})

		if err != nil {
			if errors.Is(err, errFlagNoMutation) {
				return nil
			}
			outputError(2, fmt.Sprintf("failed to save flags: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"created": true,
			"flag":    flag,
			"total":   len(ff.Decisions),
		}
		outputWorkflow(result, renderFlagActionVisual("flag", "Flag Created", result))
		return nil
	},
}

func matchingEnvironmentBlockedLaunchFlag(flags []colony.FlagEntry, candidate colony.FlagEntry) (colony.FlagEntry, bool) {
	if candidate.Resolved || candidate.Type != "blocker" || !isEnvironmentBlockedLaunchVerification(candidate.Description) {
		return colony.FlagEntry{}, false
	}
	for _, existing := range flags {
		if existing.Resolved || existing.Type != "blocker" {
			continue
		}
		if !sameFlagPhase(existing.Phase, candidate.Phase) {
			continue
		}
		if !isEnvironmentBlockedLaunchVerification(existing.Description) {
			continue
		}
		return existing, true
	}
	return colony.FlagEntry{}, false
}

func sameFlagPhase(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

var flagResolveCmd = &cobra.Command{
	Use:   "flag-resolve",
	Short: "Resolve a flag by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		id := mustGetString(cmd, "id")
		if id == "" {
			return nil
		}
		message, _ := cmd.Flags().GetString("message")

		var ff colony.FlagsFile
		if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
			if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
				outputError(1, "flags file not found", nil)
				return nil
			}
		}

		var resolvedAt string
		err := updateFlagFile(&ff, func() error {
			found := false
			for i := range ff.Decisions {
				if ff.Decisions[i].ID == id {
					if flagRequiresBoundDecisionAnswer(ff.Decisions[i]) {
						outputError(1, "protected decision requires its displayed bound decision-answer command", nil)
						return errFlagNoMutation
					}
					ff.Decisions[i].Resolved = true
					ff.Decisions[i].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
					ff.Decisions[i].Resolution = message
					resolvedAt = ff.Decisions[i].ResolvedAt
					found = true
					break
				}
			}

			if !found {
				outputError(1, fmt.Sprintf("flag %q not found", id), nil)
				return errFlagNoMutation
			}

			return nil
		})

		if err != nil {
			if errors.Is(err, errFlagNoMutation) {
				return nil
			}
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"resolved":  true,
			"id":        id,
			"message":   message,
			"timestamp": resolvedAt,
		}
		outputWorkflow(result, renderFlagActionVisual("flags", "Flag Resolved", result))
		return nil
	},
}

var flagCheckBlockersCmd = &cobra.Command{
	Use:   "flag-check-blockers",
	Short: "Check for active critical flags (blockers)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		snapshot, err := readBlockerSnapshot(store)
		if err != nil {
			detail := blockerSnapshotErrorDetail(store, err)
			outputError(2, "blocker truth unavailable", map[string]interface{}{
				"blocker_snapshot_available": false,
				"blocker_snapshot_error":     detail,
			})
			return nil
		}
		issues := 0
		notes := 0
		if ff, ok := loadFlagsFile(store); ok {
			for _, f := range ff.Decisions {
				if f.Resolved || f.Type == "blocker" {
					continue
				}
				switch f.Type {
				case "note":
					notes++
				default:
					// Preserve the diagnostic command's compatibility rule:
					// unknown non-blocker records are issues, not notes.
					issues++
				}
			}
		}

		result := map[string]interface{}{
			"issues":                     issues,
			"notes":                      notes,
			"has_blockers":               snapshot.Count > 0,
			"blocker_snapshot_available": true,
		}
		addBlockerSnapshotFields(result, snapshot)
		outputOK(result)
		return nil
	},
}

var flagAcknowledgeCmd = &cobra.Command{
	Use:   "flag-acknowledge",
	Short: "Acknowledge a flag by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		id := mustGetString(cmd, "id")
		if id == "" {
			return nil
		}

		var ff colony.FlagsFile
		if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
			if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
				outputError(1, "flags file not found", nil)
				return nil
			}
		}

		err := updateFlagFile(&ff, func() error {
			found := false
			for i := range ff.Decisions {
				if ff.Decisions[i].ID == id {
					if flagRequiresBoundDecisionAnswer(ff.Decisions[i]) {
						outputError(1, "protected decision requires its displayed bound decision-answer command", nil)
						return errFlagNoMutation
					}
					// Classic flag lifecycle: "Blockers CANNOT be acknowledged —
					// they must be resolved before phase advancement." Parking a
					// blocker would let the Iron Law gate be waved through.
					if strings.EqualFold(strings.TrimSpace(ff.Decisions[i].Type), "blocker") {
						outputError(1, fmt.Sprintf("flag %q is a blocker and cannot be acknowledged — resolve it: aether flag-resolve --id %s --message \"what fixed it\"", id, id), nil)
						return errFlagNoMutation
					}
					ff.Decisions[i].Acknowledged = true
					ff.Decisions[i].AcknowledgedAt = time.Now().UTC().Format(time.RFC3339)
					found = true
					break
				}
			}

			if !found {
				outputError(1, fmt.Sprintf("flag %q not found", id), nil)
				return errFlagNoMutation
			}

			return nil
		})

		if err != nil {
			if errors.Is(err, errFlagNoMutation) {
				return nil
			}
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"acknowledged": true,
			"id":           id,
			"at":           time.Now().UTC().Format(time.RFC3339),
		}
		outputWorkflow(result, renderFlagActionVisual("flags", "Flag Acknowledged", result))
		return nil
	},
}

// flagAutoResolveCmd is age-based housekeeping ONLY, and only under an
// explicit --max-days. It used to default to resolving anything older than 7
// days — resolving problems by aging, regardless of whether they were fixed,
// which inverted the classic semantics (classic auto-resolve was
// evidence-based: blockers cleared on build_pass). The evidence-based path
// is runtime-owned now — autoResolveVerificationBlockers runs inside
// continue where the real verification report exists — so this CLI cannot
// fake the evidence.
var flagAutoResolveCmd = &cobra.Command{
	Use:   "flag-auto-resolve",
	Short: "Resolve flags older than an explicit --max-days (age-based housekeeping only)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		maxDays, _ := cmd.Flags().GetInt("max-days")
		if maxDays <= 0 {
			outputError(1, "flag-auto-resolve requires an explicit --max-days: age is not evidence, so aging out flags is an operator decision, never a default (blockers auto-resolve on verification evidence inside continue instead)", nil)
			return nil
		}

		var ff colony.FlagsFile
		if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
			if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
				result := map[string]interface{}{"resolved": 0}
				outputWorkflow(result, renderFlagActionVisual("flags", "Flags Auto-Resolved", result))
				return nil
			}
		}

		cutoff := time.Now().UTC().AddDate(0, 0, -maxDays)
		resolved := 0

		err := updateFlagFile(&ff, func() error {
			for i := range ff.Decisions {
				if ff.Decisions[i].Resolved || flagRequiresBoundDecisionAnswer(ff.Decisions[i]) {
					continue
				}
				createdAt, err := time.Parse(time.RFC3339, ff.Decisions[i].CreatedAt)
				if err != nil {
					continue
				}
				if createdAt.Before(cutoff) {
					ff.Decisions[i].Resolved = true
					ff.Decisions[i].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
					ff.Decisions[i].Resolution = fmt.Sprintf("aged out by flag-auto-resolve --max-days %d", maxDays)
					resolved++
				}
			}

			if resolved == 0 {
				return errFlagNoMutation
			}
			return nil
		})
		if err != nil && !errors.Is(err, errFlagNoMutation) {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"resolved": resolved,
			"max_days": maxDays,
		}
		outputWorkflow(result, renderFlagActionVisual("flags", "Flags Auto-Resolved", result))
		return nil
	},
}

// autoResolveMachineSources are the flag sources the runtime itself raises
// when verification fails. When a LATER verification run comes back green,
// these clear automatically — the classic `auto_resolve_on: "build_pass"`
// semantics. Deliberately absent: chaos (the v2.4.3 exemption — a Chaos
// finding always demands a human), and user-raised flags (empty or other
// sources) — the owner's "don't advance until I say" is never waved through
// by a green build.
var autoResolveMachineSources = map[string]bool{
	"verification": true,
	"watcher":      true,
	"escalation":   true,
	"build":        true,
	"continue":     true,
	"medic":        true,
}

// autoResolveVerificationBlockers clears machine-raised blocker flags when
// verification has genuinely passed. Called by BOTH real continue paths
// strictly BEFORE the gates evaluate — the failed run raised the flag, the
// green run is the evidence that clears it, and without this ordering the
// restored Iron Law gate would deadlock on its own stale flags. NEVER called
// from the plan-only/inspection path (dry-run must not mutate).
func autoResolveVerificationBlockers(verificationPassed bool, phaseID int) int {
	if !verificationPassed || store == nil {
		return 0
	}
	var ff colony.FlagsFile
	if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
		if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
			return 0
		}
	}
	resolved := 0
	now := time.Now().UTC().Format(time.RFC3339)
	err := updateFlagFile(&ff, func() error {
		for i := range ff.Decisions {
			flag := &ff.Decisions[i]
			if flag.Resolved || flagRequiresBoundDecisionAnswer(*flag) || !strings.EqualFold(strings.TrimSpace(flag.Type), "blocker") {
				continue
			}
			source := strings.ToLower(strings.TrimSpace(flag.Source))
			if strings.Contains(source, "chaos") {
				continue
			}
			if !autoResolveMachineSources[source] {
				continue
			}
			flag.Resolved = true
			flag.ResolvedAt = now
			flag.Resolution = fmt.Sprintf("auto-resolved: phase %d verification passed", phaseID)
			resolved++
		}
		if resolved == 0 {
			return errFlagNoMutation
		}
		return nil
	})
	if err != nil && !errors.Is(err, errFlagNoMutation) {
		return 0
	}
	return resolved
}

func init() {
	flagAddCmd.Flags().String("title", "", "Flag title/description (required)")
	flagAddCmd.Flags().String("severity", "", "Severity: critical, high, low (required)")
	flagAddCmd.Flags().String("source", "", "Source of the flag")
	flagAddCmd.Flags().String("type", "", "Flag type: blocker, issue, note (default: issue)")
	flagAddCmd.Flags().String("description", "", "Detailed description (defaults to title)")
	flagAddCmd.Flags().Int("phase", 0, "Phase number (0 means no phase)")

	flagResolveCmd.Flags().String("id", "", "Flag ID to resolve (required)")
	flagResolveCmd.Flags().String("message", "", "Resolution message")
	flagAcknowledgeCmd.Flags().String("id", "", "Flag ID to acknowledge (required)")

	// No default: age-based resolution is an explicit operator decision
	// (age is not evidence that anything was fixed).
	flagAutoResolveCmd.Flags().Int("max-days", 0, "Maximum age in days for auto-resolution (required; no default)")

	rootCmd.AddCommand(flagAddCmd)
	rootCmd.AddCommand(flagResolveCmd)
	rootCmd.AddCommand(flagCheckBlockersCmd)
	rootCmd.AddCommand(flagAcknowledgeCmd)
	rootCmd.AddCommand(flagAutoResolveCmd)
}

// The generic flag lifecycle has no native binding or owner capability. These
// rows share its file, but only their existing bound decision route may resolve
// them. Recognize typed provenance and retained fields, never question prose.
func flagRequiresBoundDecisionAnswer(flag colony.FlagEntry) bool {
	if flag.Source == codexNativeDecisionSource || flag.Source == "forced-reviewer-waiver" || isAutopilotCheckpointType(flag.Type) {
		return true
	}
	for _, key := range []string{"native_binding", "waiver_capability_sha256", "waiver_capability_sha256s", "checkpoint_key", "checkpoint_capability_sha256", "checkpoint_capability_sha256s", "work_generation"} {
		if flag.HasAdditionalField(key) {
			return true
		}
	}
	return false
}

var errFlagNoMutation = errors.New("flag mutation declined or unchanged")

// updateFlagFile reloads the current shared decision file under its existing
// store lock before applying an ordinary flag mutation. A caller's earlier
// flags.json fallback remains available only when the canonical file is absent;
// an intervening native question/answer commit always replaces that stale read.
func updateFlagFile(file *colony.FlagsFile, mutate func() error) error {
	return store.UpdateJSONAtomically(pendingDecisionsFile, file, mutate)
}
