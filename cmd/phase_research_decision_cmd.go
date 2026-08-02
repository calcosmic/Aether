package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// parsePhaseResearchDecisionDescription reconstructs the phaseResearchRecommendation
// newPhaseResearchDecision (cmd/phase_research_decision.go) encoded into a
// PendingDecision.Description string of the shape
// "<recommend> phase <id> (<name>): <reason>". Returns ok=false when the
// description does not match that shape.
func parsePhaseResearchDecisionDescription(description string) (phaseResearchRecommendation, bool) {
	const marker = " phase "
	idx := strings.Index(description, marker)
	if idx < 0 {
		return phaseResearchRecommendation{}, false
	}
	recommend := description[:idx]
	if recommend != "research" && recommend != "skip" {
		return phaseResearchRecommendation{}, false
	}
	rest := description[idx+len(marker):]

	openParen := strings.Index(rest, " (")
	if openParen < 0 {
		return phaseResearchRecommendation{}, false
	}
	id, err := strconv.Atoi(strings.TrimSpace(rest[:openParen]))
	if err != nil {
		return phaseResearchRecommendation{}, false
	}
	rest = rest[openParen+len(" ("):]

	closeParen := strings.Index(rest, "): ")
	if closeParen < 0 {
		return phaseResearchRecommendation{}, false
	}
	name := rest[:closeParen]
	reason := rest[closeParen+len("): "):]

	return phaseResearchRecommendation{
		PhaseID:   id,
		PhaseName: name,
		Recommend: recommend,
		Reason:    reason,
	}, true
}

// parseFlipPhaseIDs parses a comma-separated --flip value into the set of
// phase IDs to flip. T-164-13: each token is parsed with strconv.Atoi and
// matched only against candidateIDs (the phase IDs present in the current
// unresolved research-decision set); unparsable or unknown tokens are
// reported back in invalid but never create or mutate a record.
func parseFlipPhaseIDs(flip string, candidateIDs map[int]bool) (flipped map[int]bool, invalid []string) {
	flipped = map[int]bool{}
	flip = strings.TrimSpace(flip)
	if flip == "" {
		return flipped, nil
	}
	for _, token := range strings.Split(flip, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		id, err := strconv.Atoi(token)
		if err != nil || !candidateIDs[id] {
			invalid = append(invalid, token)
			continue
		}
		flipped[id] = true
	}
	return flipped, invalid
}

// renderPlanResearchApproveLogLine builds the single log line D-16 requires
// autopilot to print. Under --auto it is prefixed "auto-accepted:"; otherwise
// it names the researched phase IDs and, when any exist, the skipped ones.
func renderPlanResearchApproveLogLine(approved, skipped []int, auto bool) string {
	if len(approved) == 0 {
		if auto {
			return "auto-accepted: no phases require research"
		}
		return "no phases require research"
	}
	prefix := "research phases"
	if auto {
		prefix = "auto-accepted: research phases"
	}
	line := fmt.Sprintf("%s %s", prefix, joinInts(approved))
	if len(skipped) > 0 {
		line += fmt.Sprintf("; skipping %s", joinInts(skipped))
	}
	return line
}

func joinInts(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

// planResearchApproveCmd answers the Queen's whole research batch (D-01) in
// one invocation. It follows the exact structure of the three verbs in
// cmd/pending_decision.go: nil-store guard, LoadJSON, mutate, SaveJSON,
// outputOK. No new JSON file or struct is introduced (RESEARCH-06).
var planResearchApproveCmd = &cobra.Command{
	Use:   "plan-research-approve",
	Short: "Answer the Queen's phase research proposal batch",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		approveAll, _ := cmd.Flags().GetBool("approve-all")
		flip, _ := cmd.Flags().GetString("flip")
		auto, _ := cmd.Flags().GetBool("auto")
		// The card's default action is acceptance -- the user should never be
		// forced to type a flag to agree with the Queen.
		if !approveAll && strings.TrimSpace(flip) == "" && !auto {
			approveAll = true
		}

		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			file = PendingDecisionFile{Decisions: []PendingDecision{}}
		}

		scope := loadCurrentPendingDecisionScope()

		candidateIDs := map[int]bool{}
		for _, d := range file.Decisions {
			if d.Type != phaseResearchDecisionType || d.Resolved {
				continue
			}
			if !pendingDecisionMatchesScope(d, scope) {
				continue
			}
			if d.Phase != nil {
				candidateIDs[*d.Phase] = true
			}
		}

		flippedSet, invalidFlips := parseFlipPhaseIDs(flip, candidateIDs)

		approved := []int{}
		skipped := []int{}
		overrides := []int{}
		now := time.Now().UTC().Format(time.RFC3339)

		for i := range file.Decisions {
			d := &file.Decisions[i]
			if d.Type != phaseResearchDecisionType || d.Resolved {
				continue
			}
			if !pendingDecisionMatchesScope(*d, scope) {
				continue
			}
			rec, ok := parsePhaseResearchDecisionDescription(d.Description)
			if !ok {
				continue
			}

			flipped := flippedSet[rec.PhaseID]
			d.Resolved = true
			d.ResolvedAt = now
			d.Resolution = phaseResearchDecisionResolution(rec, flipped, auto)
			stampPendingDecisionScope(d, scope)

			direction := rec.Recommend
			if flipped {
				direction = oppositeRecommend(rec.Recommend)
				overrides = append(overrides, rec.PhaseID)
			}
			if direction == "research" {
				approved = append(approved, rec.PhaseID)
			} else {
				skipped = append(skipped, rec.PhaseID)
			}
		}

		if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
			outputError(2, fmt.Sprintf("failed to save decisions: %v", err), nil)
			return nil
		}

		sort.Ints(approved)
		sort.Ints(skipped)
		sort.Ints(overrides)

		result := map[string]interface{}{
			"approved":  approved,
			"skipped":   skipped,
			"overrides": overrides,
			"log_line":  renderPlanResearchApproveLogLine(approved, skipped, auto),
		}
		if len(invalidFlips) > 0 {
			result["invalid_flips"] = invalidFlips
		}
		outputOK(result)
		return nil
	},
}

func init() {
	planResearchApproveCmd.Flags().Bool("approve-all", false, "Accept the Queen's recommendation for every phase")
	planResearchApproveCmd.Flags().String("flip", "", "Comma-separated phase IDs to flip")
	planResearchApproveCmd.Flags().Bool("auto", false, "Autopilot: approve all and mark records auto-accepted")

	rootCmd.AddCommand(planResearchApproveCmd)
}
