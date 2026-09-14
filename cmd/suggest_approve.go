package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/storage"
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

		// BIO-08 (CR-03): accept/edit/reject record their acting identity
		// through the SAME --actor/--actor-name mechanism
		// pheromoneDisplayCmd's five other influence actions already use,
		// defaulting to the owner exactly as that command does.
		actorFlag, _ := cmd.Flags().GetString("actor")
		actorName, _ := cmd.Flags().GetString("actor-name")
		actor := pheromoneActorOwner
		if strings.TrimSpace(actorFlag) != "" {
			actor = actorFlag
		}

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
			result, err := rejectPendingNote(dismissID, actor, actorName, dryRun)
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
				editResult, err := editPendingNote(approveID, actor, actorName, editText, dryRun)
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

			result, err := approvePendingItem(approveID, actor, actorName, dryRun)
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
	suggestApproveCmd.Flags().String("actor", "", "Actor performing --approve/--edit/--dismiss: owner (default), runtime, or learning")
	suggestApproveCmd.Flags().String("actor-name", "", "Name of the actor performing the action (optional, recorded in the history)")
	rootCmd.AddCommand(suggestApproveCmd)
}

// approvePendingItem is suggest-approve's ONE dispatch point for --approve
// (D-07/D-10, LEARN-07): a runtime-proposed pheromone suggestion and a
// cross-project import (the two pre-existing origins) still go through
// approvePendingNote unchanged; a difficulty-triggered skill proposal and a
// quarantined canary candidate (LEARN-07's two additions) are each handled
// by their own function below. There is deliberately no second approval
// command for either new kind -- TestOneApprovalSurface still passes
// because all four kinds share this ONE dispatch point and this ONE CLI
// surface (suggest-approve).
func approvePendingItem(id, actorKind, actorName string, dryRun bool) (pendingNoteActionResult, error) {
	if store == nil {
		return pendingNoteActionResult{}, fmt.Errorf("no store initialized")
	}
	_, item, _, ok := findPendingNote(id)
	if !ok {
		return pendingNoteActionResult{Found: false}, nil
	}
	switch pendingNoteOrigin(item) {
	case colony.PendingOriginSkillProposal:
		return approveSkillProposal(item, actorKind, actorName, dryRun)
	case colony.PendingOriginCanaryCandidate:
		return approveCanaryQuarantineRelease(item, actorKind, actorName, dryRun)
	default:
		return approvePendingNote(id, actorKind, actorName, dryRun)
	}
}

// colonySkillProposalSink is the real implementation of
// learn.SkillProposalSink (pkg/learn/difficulty.go): it enqueues a
// difficulty-triggered skill candidate into the SAME owner tick-to-approve
// queue every other pending decision already uses -- it never creates an
// active skill directly. cmd/codex_continue_finalize.go's
// captureContinueLearning passes this on both continue lanes
// (TestSkillProposalSinkIsWiredFromTheCheckPath).
type colonySkillProposalSink struct{}

// ProposeSkill implements learn.SkillProposalSink.
func (colonySkillProposalSink) ProposeSkill(proposal learn.SkillProposal) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	contentHash := "sha256:" + sha256Sum(proposal.Name+"|"+proposal.Content)
	now := time.Now().UTC().Format(time.RFC3339)
	origin := colony.PendingOriginSkillProposal
	name := proposal.Name
	sourceRunID := proposal.SourceRunID
	learningEntryID := proposal.LearningEntryID
	confidence := proposal.Confidence

	item := colony.PendingSuggestion{
		ID:                   generateSignalID(),
		Type:                 "SKILL",
		Content:              proposal.Content,
		Reason:               fmt.Sprintf("A difficulty-triggered skill candidate derived from run %s", sourceRunID),
		ContentHash:          contentHash,
		CreatedAt:            now,
		Origin:               &origin,
		SkillName:            &name,
		SkillSourceRunID:     &sourceRunID,
		SkillLearningEntryID: &learningEntryID,
		SkillConfidence:      &confidence,
	}

	var cs colony.ColonyState
	return store.UpdateJSONAtomically("COLONY_STATE.json", &cs, func() error {
		existing := []colony.PendingSuggestion{}
		if cs.PendingSuggestions != nil {
			existing = *cs.PendingSuggestions
		}
		for _, e := range existing {
			if !e.Dismissed && e.ContentHash == contentHash {
				return nil // identical candidate already queued -- not a duplicate proposal
			}
		}
		merged := append(existing, item)
		cs.PendingSuggestions = &merged
		return nil
	})
}

// approveSkillProposal creates the real skill through the skill service's
// own creation function (learn.SkillService.CreateSkill, pkg/learn/skills.go,
// which stays exactly where it is) and marks the queued item accepted.
// Dismissing a skill proposal (rejectPendingNote, unchanged) creates
// nothing.
func approveSkillProposal(item colony.PendingSuggestion, actorKind, actorName string, dryRun bool) (pendingNoteActionResult, error) {
	if dryRun {
		return pendingNoteActionResult{Found: true, WouldApply: true, Item: item}, nil
	}
	if item.SkillName == nil || strings.TrimSpace(*item.SkillName) == "" {
		return pendingNoteActionResult{}, fmt.Errorf("queued skill proposal %q has no skill name", item.ID)
	}

	before := pendingNoteActionSummary(item)

	dbPath := filepath.Join(store.BasePath(), "colony.db")
	sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
	if err != nil {
		return pendingNoteActionResult{}, fmt.Errorf("open skill store: %w", err)
	}
	defer sqliteStore.Close()

	baseDir := storage.ResolveAetherRoot(context.Background())
	svc := learn.NewSkillService(sqliteStore.DB(), baseDir)
	meta := learn.SkillMetadata{
		Name:        *item.SkillName,
		Stage:       learn.SkillStageActive,
		AutoCreated: true,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if item.SkillSourceRunID != nil {
		meta.SourceRunID = *item.SkillSourceRunID
	}
	if item.SkillConfidence != nil {
		meta.Confidence = *item.SkillConfidence
	}
	if err := svc.CreateSkill(meta, item.Content); err != nil {
		return pendingNoteActionResult{}, fmt.Errorf("create skill %q: %w", meta.Name, err)
	}

	stampPendingNoteAction(&item, colony.PendingActionAccepted)
	if err := savePendingNoteAtomically(item); err != nil {
		return pendingNoteActionResult{}, err
	}
	pendingNoteWriteCount++
	if _, err := appendInfluenceHistory(item.ID, pheromoneActionAccepted, actorKind, actorName, before, pendingNoteActionSummary(item), ""); err != nil {
		return pendingNoteActionResult{}, err
	}
	return pendingNoteActionResult{Found: true, Item: item}, nil
}

// approveCanaryQuarantineRelease is the queue-side half of releasing a
// quarantined canary candidate: it is reached ONLY through this one owner
// approval surface, and it is the only caller of releaseCanaryQuarantine
// (cmd/rollback.go) -- Task 2's own structural check
// (TestNoAutomaticPathReleasesAQuarantine) proves nothing else clears the
// flag.
func approveCanaryQuarantineRelease(item colony.PendingSuggestion, actorKind, actorName string, dryRun bool) (pendingNoteActionResult, error) {
	if item.CanaryCandidateID == nil || strings.TrimSpace(*item.CanaryCandidateID) == "" {
		return pendingNoteActionResult{}, fmt.Errorf("queued canary candidate %q has no linked candidate id", item.ID)
	}
	if dryRun {
		return pendingNoteActionResult{Found: true, WouldApply: true, Item: item}, nil
	}

	before := pendingNoteActionSummary(item)
	releasedBy := actorName
	if strings.TrimSpace(releasedBy) == "" {
		releasedBy = actorKind
	}
	if _, err := releaseCanaryQuarantine(*item.CanaryCandidateID, releasedBy); err != nil {
		return pendingNoteActionResult{}, err
	}

	stampPendingNoteAction(&item, colony.PendingActionAccepted)
	if err := savePendingNoteAtomically(item); err != nil {
		return pendingNoteActionResult{}, err
	}
	pendingNoteWriteCount++
	if _, err := appendInfluenceHistory(item.ID, pheromoneActionAccepted, actorKind, actorName, before, pendingNoteActionSummary(item), ""); err != nil {
		return pendingNoteActionResult{}, err
	}
	return pendingNoteActionResult{Found: true, Item: item}, nil
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
