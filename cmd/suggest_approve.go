package cmd

import (
	"encoding/json"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

var suggestApproveCmd = &cobra.Command{
	Use:   "suggest-approve",
	Short: "Review and approve pheromone suggestions from suggest-analyze",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		approveID, _ := cmd.Flags().GetString("approve")
		dismissID, _ := cmd.Flags().GetString("dismiss")
		dismissAll, _ := cmd.Flags().GetBool("dismiss-all")

		// Load active colony state. Non-blocking: return ok:true with empty on error.
		cs, err := loadActiveColonyState()
		if err != nil {
			outputOK(map[string]interface{}{
				"suggestions": []interface{}{},
				"total":       0,
			})
			return nil
		}

		// --- Flag-based modes (approve/dismiss operate even if pending list is empty) ---
		if approveID != "" || dismissID != "" || dismissAll {
			// Ensure PendingSuggestions slice exists.
			if cs.PendingSuggestions == nil {
				empty := []colony.PendingSuggestion{}
				cs.PendingSuggestions = &empty
			}

			// --- Dismiss-all mode ---
			if dismissAll {
				count := 0
				for i := range *cs.PendingSuggestions {
					if !(*cs.PendingSuggestions)[i].Dismissed {
						(*cs.PendingSuggestions)[i].Dismissed = true
						count++
					}
				}
				if !dryRun {
					stateData, _ := json.Marshal(cs)
					_ = store.AtomicWrite("COLONY_STATE.json", stateData)
				}
				outputOK(map[string]interface{}{
					"dismissed":       true,
					"dismissed_count": count,
					"dry_run":         dryRun,
				})
				return nil
			}

			// --- Dismiss single ---
			if dismissID != "" {
				found := false
				for i := range *cs.PendingSuggestions {
					if (*cs.PendingSuggestions)[i].ID == dismissID {
						found = true
						if !dryRun {
							(*cs.PendingSuggestions)[i].Dismissed = true
							stateData, _ := json.Marshal(cs)
							_ = store.AtomicWrite("COLONY_STATE.json", stateData)
						}
						break
					}
				}
				if !found {
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

			// --- Approve single ---
			var suggestion *colony.PendingSuggestion
			idx := -1
			for i := range *cs.PendingSuggestions {
				if (*cs.PendingSuggestions)[i].ID == approveID {
					s := (*cs.PendingSuggestions)[i]
					suggestion = &s
					idx = i
					break
				}
			}
			if suggestion == nil {
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
						"id":           suggestion.ID,
						"type":         suggestion.Type,
						"content":      suggestion.Content,
						"reason":       suggestion.Reason,
						"content_hash": suggestion.ContentHash,
					},
					"dry_run": dryRun,
				})
				return nil
			}

			// Sanitize content before creating pheromone signal.
			sanitized, err := colony.SanitizeSignalContent(suggestion.Content)
			if err != nil {
				outputOK(map[string]interface{}{
					"approved": false,
					"message":  "suggestion content failed sanitization: " + err.Error(),
				})
				return nil
			}

			// Build content JSON.
			contentJSON, _ := json.Marshal(map[string]string{"text": sanitized})

			// Compute content hash.
			contentHash := "sha256:" + sha256Sum(suggestion.Content)

			// Determine priority from type.
			priority := "normal"
			switch suggestion.Type {
			case "REDIRECT":
				priority = "high"
			case "FEEDBACK":
				priority = "low"
			}

			// Create pheromone signal.
			strength := 0.7
			now := time.Now().UTC().Format(time.RFC3339)
			signal := colony.PheromoneSignal{
				ID:          generateSignalID(),
				Type:        suggestion.Type,
				Content:     json.RawMessage(contentJSON),
				Priority:    priority,
				Source:      "aether-suggest",
				CreatedAt:   now,
				Active:      true,
				Strength:    &strength,
				ContentHash: &contentHash,
				Tags:        make([]colony.PheromoneTag, 0),
			}

			// Load existing pheromones, run dedup, save.
			var pf colony.PheromoneFile
			if err := store.LoadJSON("pheromones.json", &pf); err != nil {
				pf = colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
			}
			if pf.Signals == nil {
				pf.Signals = []colony.PheromoneSignal{}
			}

			// Dedup: check for existing active signal with same type + content_hash.
			replaced := false
			for i := range pf.Signals {
				sig := &pf.Signals[i]
				if !sig.Active {
					continue
				}
				if sig.Type == suggestion.Type && sig.ContentHash != nil && *sig.ContentHash == contentHash {
					// Reinforce existing signal.
					sig.CreatedAt = now
					if sig.ReinforcementCount == nil {
						rc := 0
						sig.ReinforcementCount = &rc
					}
					*sig.ReinforcementCount++
					maxStr := 1.0
					sig.Strength = &maxStr
					replaced = true
					break
				}
			}

			if !replaced {
				pf.Signals = append(pf.Signals, signal)
			}

			if err := store.SaveJSON("pheromones.json", pf); err != nil {
				outputOK(map[string]interface{}{
					"approved": false,
					"message":  "failed to save pheromone: " + err.Error(),
				})
				return nil
			}

			// Emit lifecycle event.
			emitLifecycleCeremony(events.CeremonyTopicPheromoneEmit, events.CeremonyPayload{
				PheromoneType: suggestion.Type,
				Strength:      strength,
				Status:        "created",
				Message:       sanitized,
			}, "aether-suggest")

			// Remove the suggestion from pending list.
			pending := *cs.PendingSuggestions
			pending = append(pending[:idx], pending[idx+1:]...)
			cs.PendingSuggestions = &pending

			stateData, _ := json.Marshal(cs)
			_ = store.AtomicWrite("COLONY_STATE.json", stateData)

			outputOK(map[string]interface{}{
				"approved": true,
				"signal": map[string]interface{}{
					"id":           signal.ID,
					"type":         signal.Type,
					"priority":     signal.Priority,
					"source":       signal.Source,
					"content_hash": contentHash,
				},
				"replaced": replaced,
			})
			return nil
		}

		// --- List mode (no flags) ---
		// If no pending suggestions, return empty list.
		if cs.PendingSuggestions == nil || len(*cs.PendingSuggestions) == 0 {
			active := filterActiveSuggestions(cs.PendingSuggestions)
			maps := pendingSuggestionsToMap(&active)
			outputOK(map[string]interface{}{
				"suggestions": maps,
				"total":       len(maps),
			})
			return nil
		}

		// --- Dismiss-all mode ---
		if dismissAll {
			count := 0
			for i := range *cs.PendingSuggestions {
				if !(*cs.PendingSuggestions)[i].Dismissed {
					(*cs.PendingSuggestions)[i].Dismissed = true
					count++
				}
			}
			if !dryRun {
				stateData, _ := json.Marshal(cs)
				_ = store.AtomicWrite("COLONY_STATE.json", stateData)
			}
			outputOK(map[string]interface{}{
				"dismissed":      true,
				"dismissed_count": count,
				"dry_run":        dryRun,
			})
			return nil
		}

		// --- Dismiss single ---
		if dismissID != "" {
			found := false
			for i := range *cs.PendingSuggestions {
				if (*cs.PendingSuggestions)[i].ID == dismissID {
					found = true
					if !dryRun {
						(*cs.PendingSuggestions)[i].Dismissed = true
						stateData, _ := json.Marshal(cs)
						_ = store.AtomicWrite("COLONY_STATE.json", stateData)
					}
					break
				}
			}
			if !found {
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

		// --- Approve single ---
		if approveID != "" {
			var suggestion *colony.PendingSuggestion
			idx := -1
			for i := range *cs.PendingSuggestions {
				if (*cs.PendingSuggestions)[i].ID == approveID {
					s := (*cs.PendingSuggestions)[i]
					suggestion = &s
					idx = i
					break
				}
			}
			if suggestion == nil {
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
						"id":           suggestion.ID,
						"type":         suggestion.Type,
						"content":      suggestion.Content,
						"reason":       suggestion.Reason,
						"content_hash": suggestion.ContentHash,
					},
					"dry_run": dryRun,
				})
				return nil
			}

			// Sanitize content before creating pheromone signal.
			sanitized, err := colony.SanitizeSignalContent(suggestion.Content)
			if err != nil {
				outputOK(map[string]interface{}{
					"approved": false,
					"message":  "suggestion content failed sanitization: " + err.Error(),
				})
				return nil
			}

			// Build content JSON.
			contentJSON, _ := json.Marshal(map[string]string{"text": sanitized})

			// Compute content hash.
			contentHash := "sha256:" + sha256Sum(suggestion.Content)

			// Determine priority from type.
			priority := "normal"
			switch suggestion.Type {
			case "REDIRECT":
				priority = "high"
			case "FEEDBACK":
				priority = "low"
			}

			// Create pheromone signal.
			strength := 0.7
			now := time.Now().UTC().Format(time.RFC3339)
			signal := colony.PheromoneSignal{
				ID:          generateSignalID(),
				Type:        suggestion.Type,
				Content:     json.RawMessage(contentJSON),
				Priority:    priority,
				Source:      "aether-suggest",
				CreatedAt:   now,
				Active:      true,
				Strength:    &strength,
				ContentHash: &contentHash,
				Tags:        make([]colony.PheromoneTag, 0),
			}

			// Load existing pheromones, run dedup, save.
			var pf colony.PheromoneFile
			if err := store.LoadJSON("pheromones.json", &pf); err != nil {
				pf = colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
			}
			if pf.Signals == nil {
				pf.Signals = []colony.PheromoneSignal{}
			}

			// Dedup: check for existing active signal with same type + content_hash.
			replaced := false
			for i := range pf.Signals {
				sig := &pf.Signals[i]
				if !sig.Active {
					continue
				}
				if sig.Type == suggestion.Type && sig.ContentHash != nil && *sig.ContentHash == contentHash {
					// Reinforce existing signal.
					sig.CreatedAt = now
					if sig.ReinforcementCount == nil {
						rc := 0
						sig.ReinforcementCount = &rc
					}
					*sig.ReinforcementCount++
					maxStr := 1.0
					sig.Strength = &maxStr
					replaced = true
					break
				}
			}

			if !replaced {
				pf.Signals = append(pf.Signals, signal)
			}

			if err := store.SaveJSON("pheromones.json", pf); err != nil {
				outputOK(map[string]interface{}{
					"approved": false,
					"message":  "failed to save pheromone: " + err.Error(),
				})
				return nil
			}

			// Emit lifecycle event.
			emitLifecycleCeremony(events.CeremonyTopicPheromoneEmit, events.CeremonyPayload{
				PheromoneType: suggestion.Type,
				Strength:      strength,
				Status:        "created",
				Message:       sanitized,
			}, "aether-suggest")

			// Remove the suggestion from pending list.
			pending := *cs.PendingSuggestions
			pending = append(pending[:idx], pending[idx+1:]...)
			cs.PendingSuggestions = &pending

			stateData, _ := json.Marshal(cs)
			_ = store.AtomicWrite("COLONY_STATE.json", stateData)

			outputOK(map[string]interface{}{
				"approved": true,
				"signal": map[string]interface{}{
					"id":           signal.ID,
					"type":         signal.Type,
					"priority":     signal.Priority,
					"source":       signal.Source,
					"content_hash": contentHash,
				},
				"replaced": replaced,
			})
			return nil
		}

		// Default: list mode.
		active := filterActiveSuggestions(cs.PendingSuggestions)
		maps := pendingSuggestionsToMap(&active)
		outputOK(map[string]interface{}{
			"suggestions": maps,
			"total":       len(maps),
		})
		return nil
	},
}

func init() {
	suggestApproveCmd.Flags().Bool("dry-run", false, "Preview without persisting approvals")
	suggestApproveCmd.Flags().String("approve", "", "Approve a suggestion by ID")
	suggestApproveCmd.Flags().String("dismiss", "", "Dismiss a suggestion by ID")
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
