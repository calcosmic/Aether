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

		// Filter decisions
		filtered := []PendingDecision{}
		for _, d := range file.Decisions {
			if filterUnresolved && d.Resolved {
				continue
			}
			if filterType != "" && d.Type != filterType {
				continue
			}
			filtered = append(filtered, d)
		}

		unresolved := 0
		for _, d := range file.Decisions {
			if !d.Resolved {
				unresolved++
			}
		}

		outputOK(map[string]interface{}{
			"total":      len(file.Decisions),
			"unresolved": unresolved,
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
		for i := range file.Decisions {
			if file.Decisions[i].ID == id {
				file.Decisions[i].Resolved = true
				file.Decisions[i].Resolution = resolution
				file.Decisions[i].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
				stampPendingDecisionScope(&file.Decisions[i], loadCurrentPendingDecisionScope())
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
