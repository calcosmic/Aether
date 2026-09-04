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
	if result.Outcome.Disposition == colony.SealDispositionForcedIncomplete {
		return renderForcedSealOutcome(result)
	}
	return renderVerifiedSealOutcome(result)
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
	b.WriteString(renderSealNextActions(result))
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
	b.WriteString(renderSealNextActions(result))
	return b.String()
}

func renderSealNextActions(result SealTransactionResult) string {
	primary := strings.TrimSpace(result.PrimaryNext)
	if primary == "" {
		primary = "aether status"
	}
	optional := strings.TrimSpace(result.OptionalNext)
	if optional == "" {
		optional = "aether entomb"
	}
	return fmt.Sprintf("\nNext Up\nRun `%s` to inspect the retained closure record.\nOptional: run `%s` only when you choose to archive it.\n", primary, optional)
}
