package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// PendingDecision represents a pending decision that needs resolution.
type PendingDecision struct {
	ID          string `json:"id"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description"`
	Phase       *int   `json:"phase,omitempty"`
	Source      string `json:"source,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	GoalHash    string `json:"goal_hash,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	Resolved    bool   `json:"resolved"`
	CreatedAt   string `json:"created_at"`
	ResolvedAt  string `json:"resolved_at,omitempty"`
	// AttemptID and WaiverCapabilitySHA256 bind an owner-only forced-reviewer
	// decline to the exact build attempt whose check-in card created it. The
	// raw single-use capability is shown only on that card and is never stored.
	AttemptID              string `json:"attempt_id,omitempty"`
	WaiverCapabilitySHA256 string `json:"waiver_capability_sha256,omitempty"`
	// HardConstraint marks a clarification whose answer must become a
	// REDIRECT signal (a hard "never do this"). Typed per the
	// prose-to-control-flow decision — the legacy ":hard" source suffix is
	// still honored as a fallback, but new writers set this field.
	HardConstraint bool `json:"hard_constraint,omitempty"`
	// Grounding records what a composed question is BASED ON (the scan
	// fact, state datum, or survey finding that motivated it). Required for
	// wrapper-composed questions — a question that cannot cite its basis is
	// the canned-question problem wearing a new coat.
	Grounding string `json:"grounding,omitempty"`
}

// PendingDecisionFile is the JSON structure for pending-decisions.json.
type PendingDecisionFile struct {
	Decisions []PendingDecision `json:"decisions"`
}

const pendingDecisionsFile = "pending-decisions.json"

var pendingDecisionAddCmd = &cobra.Command{
	Use:   "pending-decision-add",
	Short: "Record a pending decision",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		description := mustGetString(cmd, "description")
		if description == "" {
			return nil
		}

		decisionType, _ := cmd.Flags().GetString("type")
		phase, _ := cmd.Flags().GetInt("phase")
		source, _ := cmd.Flags().GetString("source")

		decision := PendingDecision{
			ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()),
			Type:        decisionType,
			Description: description,
			Source:      source,
			Resolved:    false,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		}

		if phase > 0 {
			decision.Phase = &phase
		}
		stampPendingDecisionScope(&decision, loadCurrentPendingDecisionScope())

		// Load existing decisions
		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			file = PendingDecisionFile{Decisions: []PendingDecision{}}
		}

		file.Decisions = append(file.Decisions, decision)

		if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
			outputError(2, fmt.Sprintf("failed to save decisions: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"id":    decision.ID,
			"added": true,
		})
		return nil
	},
}

var pendingDecisionListCmd = &cobra.Command{
	Use:   "pending-decision-list",
	Short: "List pending decisions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		filterUnresolved, _ := cmd.Flags().GetBool("unresolved")
		filterType, _ := cmd.Flags().GetString("type")

		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			outputOK(map[string]interface{}{
				"total":      0,
				"unresolved": 0,
				"decisions":  []PendingDecision{},
			})
			return nil
		}

		// Scope: only show decisions matching current session/goal
		scope := loadCurrentPendingDecisionScope()
		active, stale := filterPendingDecisionFileForScope(file, scope)

		// Filter decisions
		filtered := []PendingDecision{}
		for _, d := range active.Decisions {
			if filterUnresolved && d.Resolved {
				continue
			}
			if filterType != "" && d.Type != filterType {
				continue
			}
			// Forced-reviewer waivers are observable here, but their attempt
			// and capability bindings are implementation details of the
			// owner-only authorization path. The generic list must not expose
			// those values to callers that cannot legitimately use them.
			if d.Source == "forced-reviewer-waiver" {
				d.AttemptID = ""
				d.WaiverCapabilitySHA256 = ""
			}
			filtered = append(filtered, d)
		}

		unresolved := 0
		for _, d := range active.Decisions {
			if !d.Resolved {
				unresolved++
			}
		}

		outputOK(map[string]interface{}{
			"total":      len(file.Decisions),
			"unresolved": unresolved,
			"stale":      len(stale.Decisions),
			"decisions":  filtered,
		})
		return nil
	},
}

var pendingDecisionResolveCmd = &cobra.Command{
	Use:   "pending-decision-resolve",
	Short: "Resolve a pending decision",
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
		resolution := mustGetString(cmd, "resolution")
		if resolution == "" {
			return nil
		}

		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			outputError(1, "pending-decisions.json not found", nil)
			return nil
		}

		found := false
		scope := loadCurrentPendingDecisionScope()
		for i := range file.Decisions {
			if file.Decisions[i].ID == id {
				if !pendingDecisionMatchesScope(file.Decisions[i], scope) {
					outputError(1, fmt.Sprintf("decision %q is stale for the current goal/session", id), nil)
					return nil
				}
				// This generic command has no owner capability, build-attempt
				// binding, or dispatch-window check. Letting it resolve a
				// protected row would preserve that row's authentic metadata and
				// manufacture a valid waiver. The capability-aware
				// decision-answer path is the only resolver for this source.
				if file.Decisions[i].Source == "forced-reviewer-waiver" {
					outputError(1, "forced reviewer decisions must be answered with decision-answer and the displayed capability", nil)
					return nil
				}
				file.Decisions[i].Resolved = true
				file.Decisions[i].Resolution = resolution
				file.Decisions[i].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
				stampPendingDecisionScope(&file.Decisions[i], scope)
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("decision %q not found", id), nil)
			return nil
		}

		if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"id":       id,
			"resolved": true,
		})
		return nil
	},
}

func init() {
	pendingDecisionAddCmd.Flags().String("type", "", "Decision type (e.g., architectural, technical)")
	pendingDecisionAddCmd.Flags().String("description", "", "Description of the decision (required)")
	pendingDecisionAddCmd.Flags().Int("phase", 0, "Phase number")
	pendingDecisionAddCmd.Flags().String("source", "", "Source of the decision")

	pendingDecisionListCmd.Flags().Bool("unresolved", false, "Show only unresolved decisions")
	pendingDecisionListCmd.Flags().String("type", "", "Filter by decision type")

	pendingDecisionResolveCmd.Flags().String("id", "", "Decision ID to resolve (required)")
	pendingDecisionResolveCmd.Flags().String("resolution", "", "Resolution text (required)")

	rootCmd.AddCommand(pendingDecisionAddCmd)
	rootCmd.AddCommand(pendingDecisionListCmd)
	rootCmd.AddCommand(pendingDecisionResolveCmd)
}
