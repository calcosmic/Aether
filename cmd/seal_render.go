package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

type SealMemoryProvenance struct {
	ID                 string   `json:"id"`
	Kind               string   `json:"kind"`
	Source             string   `json:"source"`
	Store              string   `json:"store"`
	RepositoryIdentity string   `json:"repository_identity"`
	Scope              string   `json:"scope"`
	Digest             string   `json:"digest"`
	EvidenceIDs        []string `json:"evidence_ids,omitempty"`
	DecisionIDs        []string `json:"decision_ids,omitempty"`
	Sanitization       string   `json:"sanitization"`
	Sensitive          bool     `json:"sensitive"`
	TransactionID      string   `json:"transaction_id"`
}

type SealSignalDecision struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Source         string `json:"source"`
	Scope          string `json:"scope"`
	Digest         string `json:"digest"`
	Classification string `json:"classification"`
	Privacy        string `json:"privacy"`
	Sensitive      bool   `json:"sensitive"`
	TransactionID  string `json:"transaction_id"`
}

type SealWisdomDecision struct {
	ID                 string  `json:"id"`
	Source             string  `json:"source"`
	Scope              string  `json:"scope"`
	Digest             string  `json:"digest"`
	Domain             string  `json:"domain,omitempty"`
	Confidence         float64 `json:"confidence,omitempty"`
	Policy             string  `json:"policy"`
	Sanitization       string  `json:"sanitization"`
	Sensitive          bool    `json:"sensitive"`
	Eligible           bool    `json:"eligible"`
	PromotionAllowed   bool    `json:"promotion_allowed"`
	Promoted           bool    `json:"promoted"`
	Reason             string  `json:"reason"`
	PromotionReceiptID string  `json:"promotion_receipt_id,omitempty"`
	TransactionID      string  `json:"transaction_id"`
}

type SealClosureEvidence struct {
	SchemaVersion      string                  `json:"schema_version"`
	TransactionID      string                  `json:"transaction_id"`
	OutcomeKind        colony.OutcomeKind      `json:"outcome_kind"`
	Disposition        colony.SealDisposition  `json:"disposition"`
	OwnerReason        string                  `json:"owner_reason,omitempty"`
	Memory             []SealMemoryProvenance  `json:"memory"`
	Signals            []SealSignalDecision    `json:"signals"`
	Wisdom             []SealWisdomDecision    `json:"wisdom"`
	Checkpoints        []SealOwnerCheckpoint   `json:"owner_checkpoints"`
	ResidualRisks      []colony.LifecycleIssue `json:"residual_risks"`
	UncompletedWork    []SealUnresolvedItem    `json:"uncompleted_work"`
	RetainedContents   []SealPreservedContent  `json:"retained_contents"`
	PrimaryNext        string                  `json:"primary_next"`
	OptionalNext       string                  `json:"optional_next"`
	ProjectionRevision string                  `json:"projection_revision"`
}

type SealPromotionReceipt struct {
	SchemaVersion string `json:"schema_version"`
	ReceiptID     string `json:"receipt_id"`
	TransactionID string `json:"transaction_id"`
	WisdomID      string `json:"wisdom_id"`
	Policy        string `json:"policy"`
	SourceDigest  string `json:"source_digest"`
	Promoted      bool   `json:"promoted"`
	Reason        string `json:"reason,omitempty"`
}

type SealTransactionResult struct {
	TransactionID string                  `json:"transaction_id"`
	Outcome       colony.SealOutcome      `json:"outcome"`
	Receipt       colony.LifecycleReceipt `json:"receipt"`
	Evidence      SealClosureEvidence     `json:"closure_evidence"`
	SummaryPath   string                  `json:"summary_path"`
	PrimaryNext   string                  `json:"primary_next"`
	OptionalNext  string                  `json:"optional_next"`
	Replay        bool                    `json:"replay,omitempty"`
	Promotions    []SealPromotionReceipt  `json:"promotion_receipts,omitempty"`
}

// RenderSealOutcome is the sole final seal renderer. It branches on the
// durable disposition, never on a loose force flag. The forced branch avoids
// every verified-success discriminator by construction.
func RenderSealOutcome(result SealTransactionResult) string {
	structured, closeoutErr := sealLifecycleCloseoutResult(result)
	var body string
	if result.Outcome.Disposition == colony.SealDispositionForcedIncomplete {
		body = renderForcedSealOutcome(result)
	} else {
		body = renderVerifiedSealOutcome(result)
	}
	if closeoutErr != nil {
		return strings.TrimRight(body, "\n") + "\n\nCloseout unavailable: " + closeoutErr.Error() + "\n"
	}
	return appendLifecycleCloseoutVisual(body, structured, "codex")
}

func renderVerifiedSealOutcome(result SealTransactionResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("seal"), "Verified Seal"))
	b.WriteString(visualDividerStr())
	b.WriteString(crownedAnthillArt)
	b.WriteString("\n\n")
	b.WriteString(spacedTitle("Crowned Anthill"))
	b.WriteString("\n\n")
	b.WriteString("Verified completion recorded. Every phase, task, required gate, and evidence source passed the seal preflight.\n")
	b.WriteString("Receipt: ")
	b.WriteString(result.Receipt.ReceiptID)
	b.WriteString("\nRecord: ")
	b.WriteString(result.SummaryPath)
	b.WriteString("\n")
	return b.String()
}

func renderForcedSealOutcome(result SealTransactionResult) string {
	var b strings.Builder
	b.WriteString("⛔ FORCED SEAL — COMPLETION NOT VERIFIED\n")
	b.WriteString(visualDividerStr())
	b.WriteString("The colony was force-sealed for recordkeeping. Completion was not verified.\n")
	b.WriteString("Owner reason: ")
	b.WriteString(strings.TrimSpace(result.Outcome.OwnerReason))
	b.WriteString("\n")
	if len(result.Evidence.UncompletedWork) > 0 {
		b.WriteString("Unresolved at closure:\n")
		for _, item := range result.Evidence.UncompletedWork {
			b.WriteString("  - ")
			b.WriteString(formatSealUnresolvedItem(item))
			b.WriteString("\n")
		}
	}
	b.WriteString("Rollback checkpoint: ")
	if result.Outcome.Rollback != nil {
		b.WriteString(result.Outcome.Rollback.CheckpointID)
	} else {
		b.WriteString("not recorded")
	}
	b.WriteString("\n")
	return b.String()
}

// sealLifecycleCloseoutResult adapts the already-committed seal artifacts to
// the shared terminal grammar. It deliberately does not resolve a next action:
// the durable closure evidence supplies status and optional entomb authority.
func sealLifecycleCloseoutResult(result SealTransactionResult) (map[string]interface{}, error) {
	revision, err := sealCloseoutProjectionRevision(result)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.PrimaryNext) != "aether status" {
		return nil, fmt.Errorf("seal closeout requires aether status as the primary action")
	}
	if strings.TrimSpace(result.OptionalNext) != "aether entomb" {
		return nil, fmt.Errorf("seal closeout requires aether entomb as the optional action")
	}

	closure := LifecycleClosureProjection{
		Status:       "verified",
		Inspectable:  true,
		ArchiveReady: true,
	}
	if result.Outcome.Disposition == colony.SealDispositionForcedIncomplete {
		closure.Status = "forced_incomplete"
		closure.Forced = true
		closure.OwnerReason = strings.TrimSpace(result.Outcome.OwnerReason)
	}
	outcomeEvidence := append([]colony.LifecycleEvidence(nil), result.Outcome.Evidence...)
	if closure.Forced {
		for index := range outcomeEvidence {
			if strings.Contains(strings.ToLower(outcomeEvidence[index].Summary), "crowned") {
				outcomeEvidence[index].Summary = "Retained forced-incomplete closure evidence"
			}
		}
	}
	blockers := append([]colony.LifecycleIssue(nil), result.Outcome.Blockers...)
	for _, item := range result.Evidence.UncompletedWork {
		blockers = append(blockers, colony.LifecycleIssue{
			ID:          strings.TrimSpace(item.ID),
			Summary:     strings.TrimSpace(item.Summary),
			EvidenceIDs: append([]string(nil), item.EvidenceIDs...),
		})
	}
	projection := LifecycleProjection{
		SchemaVersion:      LifecycleResultSchemaVersion,
		Command:            "seal",
		OutcomeKind:        result.Outcome.OutcomeKind,
		ProjectionRevision: revision,
		View:               LifecycleViewFocused,
		Platform:           "codex",
		Standing: LifecycleFact[string]{
			Value:  "The sealed colony remains active and inspectable.",
			Source: lifecycleSource("standing", result.SummaryPath, LifecycleFactConfirmed, ""),
		},
		Changes:        append([]colony.LifecycleChange(nil), result.Outcome.Changes...),
		Evidence:       outcomeEvidence,
		Verification:   append([]colony.LifecycleVerification(nil), result.Outcome.Verification...),
		Warnings:       append([]colony.LifecycleIssue(nil), result.Outcome.Warnings...),
		Debt:           append([]colony.LifecycleIssue(nil), result.Outcome.Debt...),
		Blockers:       blockers,
		OwnerDecisions: append([]colony.LifecycleDecision(nil), result.Outcome.Decisions...),
		NextAction: LifecycleProjectedAction{
			ID:             "inspect_sealed",
			RuntimeCommand: result.PrimaryNext,
			DisplayCommand: result.PrimaryNext,
			Reason:         "The sealed colony remains active and inspectable.",
		},
		Alternatives: []LifecycleActionChoice{{
			ID:             "entomb",
			RuntimeCommand: result.OptionalNext,
			DisplayCommand: result.OptionalNext,
			Reason:         "Optionally archive and clear the sealed colony.",
		}},
		StateEffect: result.Outcome.StateEffect,
		Transaction: &result.Outcome.Transaction,
		Receipt:     &result.Receipt,
		Closure:     closure,
	}
	closureRecordSummary := "Retained Crowned Anthill closure record"
	if closure.Forced {
		closureRecordSummary = "Retained forced-incomplete closure record"
	}
	evidence := []colony.LifecycleEvidence{
		{ID: "seal-closure-record", Kind: "closure_record", Source: result.SummaryPath, Summary: closureRecordSummary},
		{ID: "seal-transaction-receipt", Kind: "receipt", Source: result.Receipt.Transaction.JournalPath, Summary: "Committed seal transaction receipt"},
	}
	evidence = append(evidence, result.Receipt.Evidence...)
	details := LifecycleCloseoutDetails{
		Summary:      "Verified completion was sealed and retained for inspection.",
		Evidence:     evidence,
		Verification: append([]colony.LifecycleVerification(nil), result.Receipt.Verification...),
	}
	if closure.Forced {
		details.Summary = "The owner force-sealed an incomplete colony for recordkeeping; completion was not verified."
	}
	structured := map[string]interface{}{
		"sealed":              true,
		"outcome_kind":        result.Outcome.OutcomeKind,
		"state_effect":        result.Outcome.StateEffect,
		"projection_revision": revision,
		"disposition":         result.Outcome.Disposition,
		"seal_outcome":        result.Outcome,
		"lifecycle_receipt":   result.Receipt,
		"summary":             result.SummaryPath,
		"next":                result.PrimaryNext,
		"optional_next":       result.OptionalNext,
	}
	applyNextActionToResult(structured, nextAction{
		Command:        result.PrimaryNext,
		Recommendation: projection.NextAction.Reason,
		Alternatives:   []nextActionAlternative{{Command: result.OptionalNext, Explanation: projection.Alternatives[0].Reason}},
		Projection:     &projection,
	})
	if err := applyLifecycleCloseout(structured, "seal", details); err != nil {
		return nil, err
	}
	return structured, nil
}

func sealCloseoutProjectionRevision(result SealTransactionResult) (string, error) {
	revisions := []string{
		strings.TrimSpace(result.Outcome.ProjectionRevision),
		strings.TrimSpace(result.Evidence.ProjectionRevision),
		strings.TrimSpace(result.Receipt.ProjectionRevision),
	}
	revision := ""
	for _, candidate := range revisions {
		if candidate == "" {
			continue
		}
		if revision != "" && candidate != revision {
			return "", fmt.Errorf("seal closeout projection revisions disagree: %q and %q", revision, candidate)
		}
		revision = candidate
	}
	if revision == "" {
		return "", fmt.Errorf("seal closeout requires a projection revision")
	}
	return revision, nil
}
