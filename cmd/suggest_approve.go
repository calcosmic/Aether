package cmd

import (
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// suggestApproveCmd is the ONE tick-to-approve surface for everything that
// needs the owner's decision before it can take effect: a runtime-proposed
// pheromone suggestion (from suggest-analyze) and a cross-project import
// held quarantined on arrival both wait in the same queue and are cleared
// by this same command (D-07/D-10). There is deliberately no second
// approval command -- TestOneApprovalSurface fails by name if one appears.
var suggestApproveCmd = &cobra.Command{
	Use:   "suggest-approve",
	Short: "Review, edit, and approve pheromone suggestions and quarantined imports",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		approveID, _ := cmd.Flags().GetString("approve")
		editText, _ := cmd.Flags().GetString("edit")
		dismissID, _ := cmd.Flags().GetString("dismiss")
		dismissAll, _ := cmd.Flags().GetBool("dismiss-all")

		// Load active colony state. Non-blocking: return ok:true with empty
		// list on error (e.g. no colony initialized yet), matching the
		// pre-existing "empty queue prints an honest empty result" behavior.
		cs, err := loadActiveColonyState()
		if err != nil {
			outputOK(map[string]interface{}{
				"suggestions": []interface{}{},
				"total":       0,
			})
			return nil
		}

		// --- Dismiss-all mode ---
		if dismissAll {
			count := 0
			if dryRun {
				if cs.PendingSuggestions != nil {
					for i := range *cs.PendingSuggestions {
						if !(*cs.PendingSuggestions)[i].Dismissed {
							count++
						}
					}
				}
			} else {
				// Re-read under the store's own lock and count what is
				// actually dismissed there, rather than marshalling the
				// snapshot loaded above over the whole file: a bare
				// SaveJSON discarded any unrelated colony-state change
				// committed since that read, and could leave the file torn
				// if interrupted. Caught by
				// TestColonyStateWriteAllowlistOnlyShrinks.
				var fresh colony.ColonyState
				if err := store.UpdateJSONAtomically("COLONY_STATE.json", &fresh, func() error {
					count = 0
					if fresh.PendingSuggestions == nil {
						return nil
					}
					pending := *fresh.PendingSuggestions
					for i := range pending {
						if !pending[i].Dismissed {
							pending[i].Dismissed = true
							count++
						}
					}
					fresh.PendingSuggestions = &pending
					return nil
				}); err != nil {
					outputErrorMessage("failed to save: " + err.Error())
					return nil
				}
			}
			outputOK(map[string]interface{}{
				"dismissed":       true,
				"dismissed_count": count,
				"dry_run":         dryRun,
			})
			return nil
		}

		// --- Dismiss single (reject) ---
		if dismissID != "" {
			result, err := rejectPendingNote(dismissID, dryRun)
			if err != nil {
				outputErrorMessage(err.Error())
				return nil
			}
			if !result.Found {
				outputOK(map[string]interface{}{
					"not_found": true,
					"message":   "suggestion not found: " + dismissID,
				})
				return nil
			}
			outputOK(map[string]interface{}{
				"dismissed": true,
				"dry_run":   dryRun,
			})
			return nil
		}

		// --- Approve single (optionally preceded by an edit) ---
		if approveID != "" {
			if editText != "" {
				editResult, err := editPendingNote(approveID, editText, dryRun)
				if err != nil {
					outputErrorMessage(err.Error())
					return nil
				}
				if !editResult.Found {
					outputOK(map[string]interface{}{
						"not_found": true,
						"message":   "suggestion not found: " + approveID,
					})
					return nil
				}
			}

			result, err := approvePendingNote(approveID, dryRun)
			if err != nil {
				outputErrorMessage(err.Error())
				return nil
			}
			if !result.Found {
				outputOK(map[string]interface{}{
					"not_found": true,
					"message":   "suggestion not found: " + approveID,
				})
				return nil
			}

			if dryRun {
				outputOK(map[string]interface{}{
					"would_approve": true,
					"suggestion": map[string]interface{}{
						"id":           result.Item.ID,
						"type":         result.Item.Type,
						"content":      result.Item.Content,
						"reason":       result.Item.Reason,
						"content_hash": result.Item.ContentHash,
						"origin":       pendingNoteOrigin(result.Item),
					},
					"dry_run": dryRun,
				})
				return nil
			}

			response := map[string]interface{}{
				"approved": true,
				"origin":   pendingNoteOrigin(result.Item),
			}
			if result.Signal != nil {
				response["signal"] = map[string]interface{}{
					"id":       result.Signal.ID,
					"type":     result.Signal.Type,
					"priority": result.Signal.Priority,
					"source":   result.Signal.Source,
				}
			}
			outputOK(response)
			return nil
		}

		// --- List mode (no flags) ---
		active := filterActiveSuggestions(cs.PendingSuggestions)
		maps := pendingNotesToMap(active)
		outputOK(map[string]interface{}{
			"suggestions": maps,
			"total":       len(maps),
		})
		return nil
	},
}

func init() {
	suggestApproveCmd.Flags().Bool("dry-run", false, "Preview without persisting approvals")
	suggestApproveCmd.Flags().String("approve", "", "Approve a suggestion or quarantined import by ID")
	suggestApproveCmd.Flags().String("edit", "", "Replace the wording of the item named by --approve before approving it")
	suggestApproveCmd.Flags().String("dismiss", "", "Reject a suggestion or quarantined import by ID")
	suggestApproveCmd.Flags().Bool("dismiss-all", false, "Dismiss all pending suggestions")
	rootCmd.AddCommand(suggestApproveCmd)
}

// filterActiveSuggestions returns only non-dismissed suggestions from the slice.
func filterActiveSuggestions(suggestions *[]colony.PendingSuggestion) []colony.PendingSuggestion {
	if suggestions == nil {
		return []colony.PendingSuggestion{}
	}
	var active []colony.PendingSuggestion
	for _, s := range *suggestions {
		if !s.Dismissed {
			active = append(active, s)
		}
	}
	if active == nil {
		active = []colony.PendingSuggestion{}
	}
	return active
}
