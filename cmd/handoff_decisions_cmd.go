package cmd

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// handoffDecision is one open decision a worker recorded in its handoff,
// awaiting an answer from the project owner. Workers already write these
// (open_decisions is a mandatory handoff field), but until this command
// existed the only reader was the next worker's prompt — a human-answerable
// question was handed to a sibling agent instead of the human.
type handoffDecision struct {
	ID        string `json:"id"`
	Question  string `json:"question"`
	Worker    string `json:"worker,omitempty"`
	Caste     string `json:"caste,omitempty"`
	Phase     int    `json:"phase,omitempty"`
	Workflow  string `json:"workflow,omitempty"`
	Freshness string `json:"freshness,omitempty"`
}

func normalizeDecisionText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func handoffDecisionID(text string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(normalizeDecisionText(text))))[:12]
}

// answeredDecisionTexts is the normalized set of questions already resolved
// in pending-decisions.json, matched against either the parsed clarification
// question or the raw description.
func answeredDecisionTexts() map[string]bool {
	answered := map[string]bool{}
	file := loadPendingDecisionFile()
	for _, decision := range file.Decisions {
		if !decision.Resolved || strings.TrimSpace(decision.Resolution) == "" {
			continue
		}
		question, _ := parseClarificationDescription(decision.Description)
		if text := normalizeDecisionText(question); text != "" {
			answered[text] = true
		}
		if text := normalizeDecisionText(decision.Description); text != "" {
			answered[text] = true
		}
	}
	return answered
}

// pendingHandoffDecisions lists open decisions from persisted worker
// handoffs that no resolved pending-decision entry answers yet. Read-only:
// it never writes to the handoff store or the decision file.
func pendingHandoffDecisions(phaseFilter int) []handoffDecision {
	records, err := loadWorkerHandoffRecords()
	if err != nil || len(records) == 0 {
		return nil
	}
	answered := answeredDecisionTexts()
	byText := map[string]handoffDecision{}
	for _, record := range records {
		if phaseFilter > 0 && record.Phase > 0 && record.Phase != phaseFilter {
			continue
		}
		for _, question := range record.OpenDecisions {
			question = strings.TrimSpace(question)
			norm := normalizeDecisionText(question)
			if norm == "" || answered[norm] {
				continue
			}
			// Later records win the dedupe so the freshest provenance shows.
			byText[norm] = handoffDecision{
				ID:        handoffDecisionID(question),
				Question:  question,
				Worker:    record.WorkerName,
				Caste:     record.Caste,
				Phase:     record.Phase,
				Workflow:  record.Workflow,
				Freshness: record.Freshness,
			}
		}
	}
	decisions := make([]handoffDecision, 0, len(byText))
	for _, decision := range byText {
		decisions = append(decisions, decision)
	}
	sort.SliceStable(decisions, func(i, j int) bool {
		return handoffFreshnessTime(decisions[i].Freshness).After(handoffFreshnessTime(decisions[j].Freshness))
	})
	return decisions
}

// recordDecisionAnswer stores an owner's answer as a resolved clarification,
// which the colony-prime capsule already renders into every subsequent
// worker prompt as CLARIFIED INTENT — no new injection path.
func recordDecisionAnswer(question, answer string, phase int, source string) (PendingDecision, error) {
	decision := PendingDecision{
		ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()),
		Type:        clarificationDecisionType,
		Description: formatClarificationDescription(question, nil),
		Source:      source,
		Resolved:    true,
		Resolution:  answer,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		ResolvedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if phase > 0 {
		decision.Phase = &phase
	}
	stampPendingDecisionScope(&decision, loadCurrentPendingDecisionScope())

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		file = PendingDecisionFile{Decisions: []PendingDecision{}}
	}
	file.Decisions = append(file.Decisions, decision)
	if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
		return PendingDecision{}, fmt.Errorf("failed to save decision answer: %w", err)
	}
	return decision, nil
}

// renderClarifiedIntentSection renders the CLARIFIED INTENT block exactly as
// the colony-prime capsule delivers it, so a wrapper can append the same
// runtime-rendered section to later-wave worker prompts mid-build.
func renderClarifiedIntentSection() string {
	result := clarifiedIntentPromptRenderResult()
	if len(result.Lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## CLARIFIED INTENT\n\n")
	for _, line := range result.Lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

var handoffDecisionsCmd = &cobra.Command{
	Use:   "handoff-decisions",
	Short: "List workers' open decisions awaiting an owner answer (read-only)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		phase, _ := cmd.Flags().GetInt("phase")
		decisions := pendingHandoffDecisions(phase)
		outputOK(map[string]interface{}{
			"count":     len(decisions),
			"decisions": decisions,
		})
		return nil
	},
}

var decisionAnswerCmd = &cobra.Command{
	Use:   "decision-answer",
	Short: "Record the owner's answer to a worker's open decision",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		question := mustGetString(cmd, "question")
		if question == "" {
			return nil
		}
		answer := mustGetString(cmd, "answer")
		if answer == "" {
			return nil
		}
		phase, _ := cmd.Flags().GetInt("phase")
		source, _ := cmd.Flags().GetString("source")
		waiverCapability, _ := cmd.Flags().GetString("waiver-capability")

		// CR-01 (194-REVIEW.md): a --question shaped like a forced-reviewer
		// decline (forcedReviewerWaiverQuestionText) may ONLY resolve a
		// pending row the runtime itself already created when it showed a
		// LIVE forced reviewer on the check-in card
		// (ensureForcedReviewerWaiverPendingDecision,
		// cmd/ceremony_team_checkin.go) -- it can never create a brand-new
		// resolved entry the way an ordinary clarification answer does
		// below. This closes the forgery path 194-REVIEW.md's CR-01 named:
		// anything able to compute the deterministic sentence (public
		// information -- phaseID plus one of five fixed strings, readable
		// in queen_risk_signals.go or the rendered card) could otherwise
		// silently waive a reviewer by simply calling this command with no
		// prior state at all. The phase used below is parsed OUT OF the
		// question text itself, not the --phase flag, so a forger cannot
		// dodge the check by passing a mismatched or absent --phase.
		if waiverPhase, _, isWaiver := forcedReviewerWaiverSignalForQuestion(question); isWaiver {
			resolved, found, err := resolveForcedReviewerWaiverPendingDecision(question, answer, waiverPhase, waiverCapability)
			if err != nil {
				outputError(2, err.Error(), nil)
				return nil
			}
			if !found {
				outputErrorMessage(fmt.Sprintf(
					"Nothing was recorded. Phase %d is not currently waiting on an answer for that reviewer -- the program only accepts a decline for a reviewer it has actually shown on the check-in card. Run `aether ceremony team-checkin` again to see the current decline command.",
					waiverPhase,
				))
				return nil
			}
			outputOK(map[string]interface{}{
				"id":             resolved.ID,
				"recorded":       true,
				"prompt_section": renderClarifiedIntentSection(),
			})
			return nil
		}

		decision, err := recordDecisionAnswer(question, answer, phase, source)
		if err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}
		outputOK(map[string]interface{}{
			"id":             decision.ID,
			"recorded":       true,
			"prompt_section": renderClarifiedIntentSection(),
		})
		return nil
	},
}

func init() {
	handoffDecisionsCmd.Flags().Int("phase", 0, "Only list decisions recorded during this phase (0 = all)")

	decisionAnswerCmd.Flags().String("question", "", "The worker's open decision being answered (required)")
	decisionAnswerCmd.Flags().String("answer", "", "The owner's answer (required)")
	decisionAnswerCmd.Flags().Int("phase", 0, "Phase the decision belongs to")
	decisionAnswerCmd.Flags().String("source", "worker-handoff", "Where the question came from")
	decisionAnswerCmd.Flags().String("waiver-capability", "", "Single-use capability from the owner-facing reviewer decline card")

	rootCmd.AddCommand(handoffDecisionsCmd)
	rootCmd.AddCommand(decisionAnswerCmd)
}
