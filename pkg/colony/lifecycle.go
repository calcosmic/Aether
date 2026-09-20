package colony

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// LifecycleSchemaVersion is the single wire-schema version shared by Phase 199
// lifecycle records. Commands may add optional fields within this version, but
// must not invent command-local versions for the records below.
const LifecycleSchemaVersion = "lifecycle/v1"

// SurveyFreshness describes whether repository territory evidence can be used.
type SurveyFreshness string

const (
	SurveyFreshnessMissing     SurveyFreshness = "missing"
	SurveyFreshnessFresh       SurveyFreshness = "fresh"
	SurveyFreshnessStale       SurveyFreshness = "stale"
	SurveyFreshnessUnavailable SurveyFreshness = "unavailable"
)

func (v SurveyFreshness) Valid() bool {
	switch v {
	case SurveyFreshnessMissing, SurveyFreshnessFresh, SurveyFreshnessStale, SurveyFreshnessUnavailable:
		return true
	default:
		return false
	}
}

func (v SurveyFreshness) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("survey freshness", string(v), v.Valid())
}

func (v *SurveyFreshness) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "survey freshness", func(raw string) bool {
		return SurveyFreshness(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = SurveyFreshness(raw)
	return nil
}

// LifecycleTransactionStage is the durable progress vocabulary for a
// multi-artifact lifecycle transaction.
type LifecycleTransactionStage string

const (
	TransactionStageValidating       LifecycleTransactionStage = "validating"
	TransactionStageStaging          LifecycleTransactionStage = "staging"
	TransactionStageStaged           LifecycleTransactionStage = "staged"
	TransactionStageIntentRecorded   LifecycleTransactionStage = "intent_recorded"
	TransactionStageCommitting       LifecycleTransactionStage = "committing"
	TransactionStageCommitted        LifecycleTransactionStage = "committed"
	TransactionStageVerifying        LifecycleTransactionStage = "verifying"
	TransactionStageVerified         LifecycleTransactionStage = "verified"
	TransactionStageRollingBack      LifecycleTransactionStage = "rolling_back"
	TransactionStageRolledBack       LifecycleTransactionStage = "rolled_back"
	TransactionStageRecoveryRequired LifecycleTransactionStage = "recovery_required"
)

func (v LifecycleTransactionStage) Valid() bool {
	switch v {
	case TransactionStageValidating,
		TransactionStageStaging,
		TransactionStageStaged,
		TransactionStageIntentRecorded,
		TransactionStageCommitting,
		TransactionStageCommitted,
		TransactionStageVerifying,
		TransactionStageVerified,
		TransactionStageRollingBack,
		TransactionStageRolledBack,
		TransactionStageRecoveryRequired:
		return true
	default:
		return false
	}
}

func (v LifecycleTransactionStage) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("transaction stage", string(v), v.Valid())
}

func (v *LifecycleTransactionStage) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "transaction stage", func(raw string) bool {
		return LifecycleTransactionStage(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = LifecycleTransactionStage(raw)
	return nil
}

// RecoveryProvenance identifies how strongly durable evidence supports a
// recovered fact. Unknown is an explicit value; the empty value means the fact
// was not recorded at all.
type RecoveryProvenance string

const (
	RecoveryProvenanceConfirmed     RecoveryProvenance = "confirmed"
	RecoveryProvenanceReconstructed RecoveryProvenance = "reconstructed"
	RecoveryProvenanceConflicting   RecoveryProvenance = "conflicting"
	RecoveryProvenanceUnknown       RecoveryProvenance = "unknown"
)

func (v RecoveryProvenance) Valid() bool {
	switch v {
	case RecoveryProvenanceConfirmed,
		RecoveryProvenanceReconstructed,
		RecoveryProvenanceConflicting,
		RecoveryProvenanceUnknown:
		return true
	default:
		return false
	}
}

func (v RecoveryProvenance) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("recovery provenance", string(v), v.Valid())
}

func (v *RecoveryProvenance) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "recovery provenance", func(raw string) bool {
		return RecoveryProvenance(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = RecoveryProvenance(raw)
	return nil
}

// SignalDelivery states the strongest delivery claim supported by evidence.
type SignalDelivery string

const (
	SignalDeliveryLive             SignalDelivery = "live"
	SignalDeliveryNextSafeBoundary SignalDelivery = "next_safe_boundary"
	SignalDeliveryUnsupported      SignalDelivery = "unsupported"
)

func (v SignalDelivery) Valid() bool {
	switch v {
	case SignalDeliveryLive, SignalDeliveryNextSafeBoundary, SignalDeliveryUnsupported:
		return true
	default:
		return false
	}
}

func (v SignalDelivery) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("signal delivery", string(v), v.Valid())
}

func (v *SignalDelivery) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "signal delivery", func(raw string) bool {
		return SignalDelivery(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = SignalDelivery(raw)
	return nil
}

// SealDisposition separates verified completion from an owner-forced record of
// incomplete closure.
type SealDisposition string

const (
	SealDispositionVerified         SealDisposition = "verified"
	SealDispositionForcedIncomplete SealDisposition = "forced_incomplete"
)

func (v SealDisposition) Valid() bool {
	switch v {
	case SealDispositionVerified, SealDispositionForcedIncomplete:
		return true
	default:
		return false
	}
}

func (v SealDisposition) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("seal disposition", string(v), v.Valid())
}

func (v *SealDisposition) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "seal disposition", func(raw string) bool {
		return SealDisposition(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = SealDisposition(raw)
	return nil
}

// OutcomeKind is the exhaustive command-result vocabulary shared by lifecycle
// renderers and durable receipts.
type OutcomeKind string

const (
	OutcomeKindInProgress              OutcomeKind = "in_progress"
	OutcomeKindCompleted               OutcomeKind = "completed"
	OutcomeKindNoChange                OutcomeKind = "no_change"
	OutcomeKindRefused                 OutcomeKind = "refused"
	OutcomeKindPaused                  OutcomeKind = "paused"
	OutcomeKindResumed                 OutcomeKind = "resumed"
	OutcomeKindArchived                OutcomeKind = "archived"
	OutcomeKindFailed                  OutcomeKind = "failed"
	OutcomeKindRecoveryRequired        OutcomeKind = "recovery_required"
	OutcomeKindVerifiedCompletion      OutcomeKind = "verified_completion"
	OutcomeKindForcedIncompleteClosure OutcomeKind = "forced_incomplete_closure"
)

func (v OutcomeKind) Valid() bool {
	switch v {
	case OutcomeKindInProgress,
		OutcomeKindCompleted,
		OutcomeKindNoChange,
		OutcomeKindRefused,
		OutcomeKindPaused,
		OutcomeKindResumed,
		OutcomeKindArchived,
		OutcomeKindFailed,
		OutcomeKindRecoveryRequired,
		OutcomeKindVerifiedCompletion,
		OutcomeKindForcedIncompleteClosure:
		return true
	default:
		return false
	}
}

func (v OutcomeKind) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("outcome kind", string(v), v.Valid())
}

func (v *OutcomeKind) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "outcome kind", func(raw string) bool {
		return OutcomeKind(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = OutcomeKind(raw)
	return nil
}

// LifecycleStateEffect describes the authoritative mutation effect of a
// lifecycle result.
type LifecycleStateEffect string

const (
	LifecycleStateEffectNone             LifecycleStateEffect = "none"
	LifecycleStateEffectCommitted        LifecycleStateEffect = "committed"
	LifecycleStateEffectRolledBack       LifecycleStateEffect = "rolled_back"
	LifecycleStateEffectRetained         LifecycleStateEffect = "retained"
	LifecycleStateEffectRecoveryRequired LifecycleStateEffect = "recovery_required"
)

func (v LifecycleStateEffect) Valid() bool {
	switch v {
	case LifecycleStateEffectNone,
		LifecycleStateEffectCommitted,
		LifecycleStateEffectRolledBack,
		LifecycleStateEffectRetained,
		LifecycleStateEffectRecoveryRequired:
		return true
	default:
		return false
	}
}

func (v LifecycleStateEffect) MarshalJSON() ([]byte, error) {
	return marshalLifecycleEnum("state effect", string(v), v.Valid())
}

func (v *LifecycleStateEffect) UnmarshalJSON(data []byte) error {
	raw, err := unmarshalLifecycleEnum(data, "state effect", func(raw string) bool {
		return LifecycleStateEffect(raw).Valid()
	})
	if err != nil {
		return err
	}
	*v = LifecycleStateEffect(raw)
	return nil
}

func marshalLifecycleEnum(name, value string, valid bool) ([]byte, error) {
	if !valid {
		return nil, fmt.Errorf("invalid %s %q", name, value)
	}
	return json.Marshal(value)
}

func unmarshalLifecycleEnum(data []byte, name string, valid func(string) bool) (string, error) {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return "", fmt.Errorf("%s must be a JSON string: %w", name, err)
	}
	if !valid(value) {
		return "", fmt.Errorf("invalid %s %q", name, value)
	}
	return value, nil
}

// LifecycleChange records one declared durable mutation without embedding an
// untyped command-local payload.
type LifecycleChange struct {
	Target       string `json:"target"`
	Action       string `json:"action"`
	BeforeDigest string `json:"before_digest,omitempty"`
	AfterDigest  string `json:"after_digest,omitempty"`
}

// LifecycleEvidence names one durable or runtime source supporting a claim.
type LifecycleEvidence struct {
	ID      string `json:"id"`
	Kind    string `json:"kind,omitempty"`
	Source  string `json:"source,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// LifecycleVerification records a deterministic check and the evidence it
// evaluated.
type LifecycleVerification struct {
	Name        string   `json:"name"`
	Passed      bool     `json:"passed"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	Detail      string   `json:"detail,omitempty"`
}

// LifecycleIssue is the shared typed shape for warnings, debt, and blockers.
type LifecycleIssue struct {
	ID          string   `json:"id"`
	Summary     string   `json:"summary"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

// LifecycleDecision preserves one scoped owner/runtime decision and its proof.
type LifecycleDecision struct {
	ID          string   `json:"id"`
	Scope       string   `json:"scope"`
	Summary     string   `json:"summary"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

// LifecycleTransactionReference is the non-recursive link used by outcomes
// and receipts to identify their transaction journal.
type LifecycleTransactionReference struct {
	ID          string                    `json:"id"`
	Stage       LifecycleTransactionStage `json:"stage"`
	JournalPath string                    `json:"journal_path,omitempty"`
}

// LifecycleReceiptReference links another record to a durable receipt.
type LifecycleReceiptReference struct {
	ID     string `json:"id"`
	Digest string `json:"digest,omitempty"`
}

// LifecycleRecoveryFact records one recovery conclusion with its own
// provenance so confirmed and reconstructed facts cannot be flattened.
type LifecycleRecoveryFact struct {
	Name       string              `json:"name"`
	Summary    string              `json:"summary"`
	Provenance RecoveryProvenance  `json:"provenance"`
	Evidence   []LifecycleEvidence `json:"evidence,omitempty"`
}

// LifecycleRecovery groups the provenance-bearing facts used to resume or
// roll back a transaction.
type LifecycleRecovery struct {
	Provenance   RecoveryProvenance             `json:"provenance"`
	Facts        []LifecycleRecoveryFact        `json:"facts,omitempty"`
	Transaction  *LifecycleTransactionReference `json:"transaction,omitempty"`
	Receipt      *LifecycleReceiptReference     `json:"receipt,omitempty"`
	SafeNextStep string                         `json:"safe_next_step,omitempty"`
}

// LifecycleRollback identifies the checkpoint and evidence needed to undo or
// inspect an owner-forced closure.
type LifecycleRollback struct {
	CheckpointID string              `json:"checkpoint_id"`
	Evidence     []LifecycleEvidence `json:"evidence"`
}

// LifecycleReceipt is the canonical durable result shared across lifecycle
// commands. It deliberately uses typed lists instead of map[string]interface{}.
type LifecycleReceipt struct {
	SchemaVersion      string                        `json:"schema_version"`
	ReceiptID          string                        `json:"receipt_id"`
	Command            string                        `json:"command"`
	OutcomeKind        OutcomeKind                   `json:"outcome_kind"`
	ProjectionRevision string                        `json:"projection_revision,omitempty"`
	Changes            []LifecycleChange             `json:"changes,omitempty"`
	Evidence           []LifecycleEvidence           `json:"evidence,omitempty"`
	Verification       []LifecycleVerification       `json:"verification,omitempty"`
	Warnings           []LifecycleIssue              `json:"warnings,omitempty"`
	Debt               []LifecycleIssue              `json:"debt,omitempty"`
	Blockers           []LifecycleIssue              `json:"blockers,omitempty"`
	Decisions          []LifecycleDecision           `json:"decisions,omitempty"`
	StateEffect        LifecycleStateEffect          `json:"state_effect"`
	Transaction        LifecycleTransactionReference `json:"transaction"`
	Recovery           *LifecycleRecovery            `json:"recovery,omitempty"`
	Provenance         RecoveryProvenance            `json:"provenance"`
}

// LifecycleTransactionRecord is the versioned coordinator-journal record for
// a lifecycle mutation.
type LifecycleTransactionRecord struct {
	SchemaVersion      string                     `json:"schema_version"`
	TransactionID      string                     `json:"transaction_id"`
	Command            string                     `json:"command"`
	OutcomeKind        OutcomeKind                `json:"outcome_kind"`
	ProjectionRevision string                     `json:"projection_revision,omitempty"`
	Stage              LifecycleTransactionStage  `json:"stage"`
	BaselineDigest     string                     `json:"baseline_digest,omitempty"`
	Changes            []LifecycleChange          `json:"changes,omitempty"`
	Evidence           []LifecycleEvidence        `json:"evidence,omitempty"`
	Verification       []LifecycleVerification    `json:"verification,omitempty"`
	Warnings           []LifecycleIssue           `json:"warnings,omitempty"`
	Debt               []LifecycleIssue           `json:"debt,omitempty"`
	Blockers           []LifecycleIssue           `json:"blockers,omitempty"`
	Decisions          []LifecycleDecision        `json:"decisions,omitempty"`
	StateEffect        LifecycleStateEffect       `json:"state_effect"`
	Receipt            *LifecycleReceiptReference `json:"receipt,omitempty"`
	Recovery           *LifecycleRecovery         `json:"recovery,omitempty"`
	Provenance         RecoveryProvenance         `json:"provenance"`
}

// LifecycleRepositoryEvidence binds a pause handoff to repository bytes.
type LifecycleRepositoryEvidence struct {
	Root        string `json:"root,omitempty"`
	Head        string `json:"head"`
	DirtyDigest string `json:"dirty_digest"`
}

// LifecycleWorktreeEvidence identifies one worktree present at a pause.
type LifecycleWorktreeEvidence struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Branch string `json:"branch"`
	Head   string `json:"head,omitempty"`
	Digest string `json:"digest,omitempty"`
}

// LifecycleWorkerLineage records an actual worker and its parent relationship.
type LifecycleWorkerLineage struct {
	WorkerID       string `json:"worker_id"`
	ParentWorkerID string `json:"parent_worker_id,omitempty"`
	Caste          string `json:"caste"`
	Status         string `json:"status"`
}

// LifecycleSignalSnapshot binds a handoff to the durable signal state it saw.
type LifecycleSignalSnapshot struct {
	SignalID string `json:"signal_id"`
	Type     string `json:"type"`
	Digest   string `json:"digest,omitempty"`
}

// PauseHandoff is the canonical, versioned safe-boundary handoff.
type PauseHandoff struct {
	SchemaVersion      string                        `json:"schema_version"`
	HandoffID          string                        `json:"handoff_id"`
	Command            string                        `json:"command"`
	OutcomeKind        OutcomeKind                   `json:"outcome_kind"`
	ProjectionRevision string                        `json:"projection_revision,omitempty"`
	PhaseID            int                           `json:"phase_id,omitempty"`
	TaskIDs            []string                      `json:"task_ids,omitempty"`
	AttemptID          string                        `json:"attempt_id,omitempty"`
	RunID              string                        `json:"run_id,omitempty"`
	SafeBoundary       string                        `json:"safe_boundary"`
	RestartPoint       string                        `json:"restart_point"`
	ContextDigest      string                        `json:"context_digest"`
	Repository         LifecycleRepositoryEvidence   `json:"repository"`
	Worktrees          []LifecycleWorktreeEvidence   `json:"worktrees,omitempty"`
	Lineage            []LifecycleWorkerLineage      `json:"lineage,omitempty"`
	Signals            []LifecycleSignalSnapshot     `json:"signals,omitempty"`
	Changes            []LifecycleChange             `json:"changes,omitempty"`
	Evidence           []LifecycleEvidence           `json:"evidence,omitempty"`
	Verification       []LifecycleVerification       `json:"verification,omitempty"`
	Warnings           []LifecycleIssue              `json:"warnings,omitempty"`
	Debt               []LifecycleIssue              `json:"debt,omitempty"`
	Blockers           []LifecycleIssue              `json:"blockers,omitempty"`
	Decisions          []LifecycleDecision           `json:"decisions,omitempty"`
	StateEffect        LifecycleStateEffect          `json:"state_effect"`
	Transaction        LifecycleTransactionReference `json:"transaction"`
	Receipt            *LifecycleReceiptReference    `json:"receipt,omitempty"`
	Recovery           *LifecycleRecovery            `json:"recovery,omitempty"`
	Provenance         RecoveryProvenance            `json:"provenance"`
}

// PauseHandoffReference lets state and session files point at a validated
// handoff without duplicating the entire record.
type PauseHandoffReference struct {
	ID            string `json:"id"`
	TransactionID string `json:"transaction_id"`
	Digest        string `json:"digest"`
	Path          string `json:"path,omitempty"`
}

// SealOutcome is the durable closure result. Its discriminator prevents an
// owner-forced incomplete record from being rendered or counted as verified.
type SealOutcome struct {
	SchemaVersion      string                        `json:"schema_version"`
	OutcomeID          string                        `json:"outcome_id"`
	Command            string                        `json:"command"`
	OutcomeKind        OutcomeKind                   `json:"outcome_kind"`
	ProjectionRevision string                        `json:"projection_revision,omitempty"`
	Disposition        SealDisposition               `json:"disposition"`
	CompletedPhases    []int                         `json:"completed_phases,omitempty"`
	IncompletePhases   []int                         `json:"incomplete_phases,omitempty"`
	IncompleteTaskIDs  []string                      `json:"incomplete_task_ids,omitempty"`
	FailedGates        []LifecycleVerification       `json:"failed_gates,omitempty"`
	SkippedGates       []LifecycleVerification       `json:"skipped_gates,omitempty"`
	MissingEvidence    []LifecycleEvidence           `json:"missing_evidence,omitempty"`
	UnresolvedEvidence []LifecycleEvidence           `json:"unresolved_evidence,omitempty"`
	OwnerReason        string                        `json:"owner_reason,omitempty"`
	Rollback           *LifecycleRollback            `json:"rollback,omitempty"`
	Changes            []LifecycleChange             `json:"changes,omitempty"`
	Evidence           []LifecycleEvidence           `json:"evidence,omitempty"`
	Verification       []LifecycleVerification       `json:"verification,omitempty"`
	Warnings           []LifecycleIssue              `json:"warnings,omitempty"`
	Debt               []LifecycleIssue              `json:"debt,omitempty"`
	Blockers           []LifecycleIssue              `json:"blockers,omitempty"`
	Decisions          []LifecycleDecision           `json:"decisions,omitempty"`
	StateEffect        LifecycleStateEffect          `json:"state_effect"`
	Transaction        LifecycleTransactionReference `json:"transaction"`
	Receipt            *LifecycleReceiptReference    `json:"receipt,omitempty"`
	Recovery           *LifecycleRecovery            `json:"recovery,omitempty"`
	Provenance         RecoveryProvenance            `json:"provenance"`
}

// SealOutcomeReference preserves the closure identity and forced marker in an
// archive without duplicating the full outcome.
type SealOutcomeReference struct {
	OutcomeID     string          `json:"outcome_id"`
	Disposition   SealDisposition `json:"disposition"`
	TransactionID string          `json:"transaction_id"`
	OwnerReason   string          `json:"owner_reason,omitempty"`
}

// ArchiveEntry records source and archived byte digests for one relative path.
type ArchiveEntry struct {
	Path          string             `json:"path"`
	Kind          string             `json:"kind"`
	Size          int64              `json:"size"`
	SourceDigest  string             `json:"source_digest"`
	ArchiveDigest string             `json:"archive_digest"`
	Provenance    RecoveryProvenance `json:"provenance"`
}

// ArchiveCrossReference binds archive/state/report/XML/tombstone identities.
type ArchiveCrossReference struct {
	Kind     string `json:"kind"`
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Digest   string `json:"digest,omitempty"`
}

// ArchiveManifest is the digest-backed, closure-linked archive vocabulary.
type ArchiveManifest struct {
	SchemaVersion      string                        `json:"schema_version"`
	ArchiveID          string                        `json:"archive_id"`
	Command            string                        `json:"command"`
	OutcomeKind        OutcomeKind                   `json:"outcome_kind"`
	ProjectionRevision string                        `json:"projection_revision,omitempty"`
	ManifestDigest     string                        `json:"manifest_digest"`
	Seal               SealOutcomeReference          `json:"seal_outcome"`
	Entries            []ArchiveEntry                `json:"entries"`
	CrossReferences    []ArchiveCrossReference       `json:"cross_references"`
	Changes            []LifecycleChange             `json:"changes,omitempty"`
	Evidence           []LifecycleEvidence           `json:"evidence,omitempty"`
	Verification       []LifecycleVerification       `json:"verification,omitempty"`
	Warnings           []LifecycleIssue              `json:"warnings,omitempty"`
	Debt               []LifecycleIssue              `json:"debt,omitempty"`
	Blockers           []LifecycleIssue              `json:"blockers,omitempty"`
	Decisions          []LifecycleDecision           `json:"decisions,omitempty"`
	StateEffect        LifecycleStateEffect          `json:"state_effect"`
	Transaction        LifecycleTransactionReference `json:"transaction"`
	Receipt            *LifecycleReceiptReference    `json:"receipt,omitempty"`
	Recovery           *LifecycleRecovery            `json:"recovery,omitempty"`
	Provenance         RecoveryProvenance            `json:"provenance"`
}

// ArchiveReference lets active state point to a manifest without copying it.
type ArchiveReference struct {
	ID             string `json:"id"`
	TransactionID  string `json:"transaction_id"`
	ManifestDigest string `json:"manifest_digest"`
	Path           string `json:"path"`
}

// SignalAcknowledgement is present only when a named runtime actor and its
// evidence have acknowledged a signal.
type SignalAcknowledgement struct {
	ActorID    string `json:"actor_id"`
	EvidenceID string `json:"evidence_id"`
}

// SignalDeliveryReceipt distinguishes delivery, acknowledgement, measured
// effect, and fallback evidence without treating prompt presence as influence.
type SignalDeliveryReceipt struct {
	SchemaVersion       string                        `json:"schema_version"`
	ReceiptID           string                        `json:"receipt_id"`
	Command             string                        `json:"command"`
	OutcomeKind         OutcomeKind                   `json:"outcome_kind"`
	ProjectionRevision  string                        `json:"projection_revision,omitempty"`
	SignalID            string                        `json:"signal_id"`
	Delivery            SignalDelivery                `json:"delivery"`
	Acknowledgement     *SignalAcknowledgement        `json:"acknowledgement,omitempty"`
	AffectedJobIDs      []string                      `json:"affected_job_ids,omitempty"`
	AffectedDecisionIDs []string                      `json:"affected_decision_ids,omitempty"`
	EffectEvidence      []LifecycleEvidence           `json:"effect_evidence,omitempty"`
	FallbackEvidence    []LifecycleEvidence           `json:"fallback_evidence,omitempty"`
	Changes             []LifecycleChange             `json:"changes,omitempty"`
	Evidence            []LifecycleEvidence           `json:"evidence,omitempty"`
	Verification        []LifecycleVerification       `json:"verification,omitempty"`
	Warnings            []LifecycleIssue              `json:"warnings,omitempty"`
	Debt                []LifecycleIssue              `json:"debt,omitempty"`
	Blockers            []LifecycleIssue              `json:"blockers,omitempty"`
	Decisions           []LifecycleDecision           `json:"decisions,omitempty"`
	StateEffect         LifecycleStateEffect          `json:"state_effect"`
	Transaction         LifecycleTransactionReference `json:"transaction"`
	Recovery            *LifecycleRecovery            `json:"recovery,omitempty"`
	Provenance          RecoveryProvenance            `json:"provenance"`
}

// Validate checks a lifecycle receipt without mutating it.
func (r LifecycleReceipt) Validate() error {
	if err := validateLifecycleHeader(r.SchemaVersion, r.Command, r.OutcomeKind, r.StateEffect, r.Provenance); err != nil {
		return err
	}
	if strings.TrimSpace(r.ReceiptID) == "" {
		return fmt.Errorf("receipt_id is required")
	}
	if err := r.Transaction.Validate(); err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	if err := validateLifecycleCollections(r.Changes, r.Evidence, r.Verification, r.Warnings, r.Debt, r.Blockers, r.Decisions); err != nil {
		return err
	}
	if r.Recovery != nil {
		if err := r.Recovery.Validate(); err != nil {
			return fmt.Errorf("recovery: %w", err)
		}
	}
	if (r.OutcomeKind == OutcomeKindRecoveryRequired || r.StateEffect == LifecycleStateEffectRecoveryRequired) && r.Recovery == nil {
		return fmt.Errorf("recovery is required for a recovery_required result")
	}
	return nil
}

// Validate checks a transaction record without changing its stage.
func (r LifecycleTransactionRecord) Validate() error {
	if err := validateLifecycleHeader(r.SchemaVersion, r.Command, r.OutcomeKind, r.StateEffect, r.Provenance); err != nil {
		return err
	}
	if strings.TrimSpace(r.TransactionID) == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if !r.Stage.Valid() {
		return fmt.Errorf("stage: invalid transaction stage %q", r.Stage)
	}
	if err := validateLifecycleCollections(r.Changes, r.Evidence, r.Verification, r.Warnings, r.Debt, r.Blockers, r.Decisions); err != nil {
		return err
	}
	if r.Receipt != nil {
		if err := r.Receipt.Validate(); err != nil {
			return fmt.Errorf("receipt: %w", err)
		}
	}
	if r.Recovery != nil {
		if err := r.Recovery.Validate(); err != nil {
			return fmt.Errorf("recovery: %w", err)
		}
	}
	if (r.Stage == TransactionStageRecoveryRequired || r.StateEffect == LifecycleStateEffectRecoveryRequired) && r.Recovery == nil {
		return fmt.Errorf("recovery is required for a recovery_required transaction")
	}
	return nil
}

// Validate checks a recovery result and each independently proven fact.
func (r LifecycleRecovery) Validate() error {
	if !r.Provenance.Valid() {
		return fmt.Errorf("provenance: invalid recovery provenance %q", r.Provenance)
	}
	if r.Transaction != nil {
		if err := r.Transaction.Validate(); err != nil {
			return fmt.Errorf("transaction: %w", err)
		}
	}
	if r.Receipt != nil {
		if err := r.Receipt.Validate(); err != nil {
			return fmt.Errorf("receipt: %w", err)
		}
	}
	if r.Provenance != RecoveryProvenanceUnknown && len(r.Facts) == 0 {
		return fmt.Errorf("facts are required for %s recovery provenance", r.Provenance)
	}
	for i := range r.Facts {
		if err := r.Facts[i].Validate(); err != nil {
			return fmt.Errorf("facts[%d]: %w", i, err)
		}
	}
	return nil
}

// Validate checks one provenance-bearing recovery fact.
func (f LifecycleRecoveryFact) Validate() error {
	if strings.TrimSpace(f.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(f.Summary) == "" {
		return fmt.Errorf("summary is required")
	}
	if !f.Provenance.Valid() {
		return fmt.Errorf("provenance: invalid recovery provenance %q", f.Provenance)
	}
	if f.Provenance != RecoveryProvenanceUnknown && len(f.Evidence) == 0 {
		return fmt.Errorf("evidence is required for %s provenance", f.Provenance)
	}
	return validateLifecycleEvidence(f.Evidence)
}

// Validate checks the safe-boundary and transaction identity of a handoff.
func (h PauseHandoff) Validate() error {
	if err := validateLifecycleHeader(h.SchemaVersion, h.Command, h.OutcomeKind, h.StateEffect, h.Provenance); err != nil {
		return err
	}
	if h.OutcomeKind != OutcomeKindPaused {
		return fmt.Errorf("outcome_kind must be %q for a pause handoff", OutcomeKindPaused)
	}
	if strings.TrimSpace(h.HandoffID) == "" {
		return fmt.Errorf("handoff_id is required")
	}
	if strings.TrimSpace(h.AttemptID) == "" && strings.TrimSpace(h.RunID) == "" {
		return fmt.Errorf("attempt_id or run_id is required")
	}
	if strings.TrimSpace(h.SafeBoundary) == "" {
		return fmt.Errorf("safe_boundary is required")
	}
	if strings.TrimSpace(h.RestartPoint) == "" {
		return fmt.Errorf("restart_point is required")
	}
	if strings.TrimSpace(h.ContextDigest) == "" {
		return fmt.Errorf("context_digest is required")
	}
	if strings.TrimSpace(h.Repository.Head) == "" {
		return fmt.Errorf("repository.head is required")
	}
	if strings.TrimSpace(h.Repository.DirtyDigest) == "" {
		return fmt.Errorf("repository.dirty_digest is required")
	}
	if err := h.Transaction.Validate(); err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	if h.Receipt != nil {
		if err := h.Receipt.Validate(); err != nil {
			return fmt.Errorf("receipt: %w", err)
		}
	}
	if h.Recovery != nil {
		if err := h.Recovery.Validate(); err != nil {
			return fmt.Errorf("recovery: %w", err)
		}
	}
	if err := validateLifecycleCollections(h.Changes, h.Evidence, h.Verification, h.Warnings, h.Debt, h.Blockers, h.Decisions); err != nil {
		return err
	}
	return validateWorkerLineage(h.Lineage)
}

// Validate checks a state/session handoff reference.
func (r PauseHandoffReference) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(r.TransactionID) == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if strings.TrimSpace(r.Digest) == "" {
		return fmt.Errorf("digest is required")
	}
	return nil
}

// Validate enforces the mutually exclusive verified and forced-incomplete
// closure variants.
func (o SealOutcome) Validate() error {
	if err := validateLifecycleHeader(o.SchemaVersion, o.Command, o.OutcomeKind, o.StateEffect, o.Provenance); err != nil {
		return err
	}
	if strings.TrimSpace(o.OutcomeID) == "" {
		return fmt.Errorf("outcome_id is required")
	}
	if !o.Disposition.Valid() {
		return fmt.Errorf("disposition: invalid seal disposition %q", o.Disposition)
	}
	if err := o.Transaction.Validate(); err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	if o.Receipt == nil {
		return fmt.Errorf("receipt is required")
	}
	if err := o.Receipt.Validate(); err != nil {
		return fmt.Errorf("receipt: %w", err)
	}
	if o.Recovery != nil {
		if err := o.Recovery.Validate(); err != nil {
			return fmt.Errorf("recovery: %w", err)
		}
	}
	if err := validateLifecycleCollections(o.Changes, o.Evidence, o.Verification, o.Warnings, o.Debt, o.Blockers, o.Decisions); err != nil {
		return err
	}
	if err := validateLifecycleEvidence(o.MissingEvidence); err != nil {
		return fmt.Errorf("missing_evidence: %w", err)
	}
	if err := validateLifecycleEvidence(o.UnresolvedEvidence); err != nil {
		return fmt.Errorf("unresolved_evidence: %w", err)
	}

	switch o.Disposition {
	case SealDispositionVerified:
		if o.OutcomeKind != OutcomeKindVerifiedCompletion {
			return fmt.Errorf("outcome_kind must be %q for verified seal", OutcomeKindVerifiedCompletion)
		}
		if strings.TrimSpace(o.OwnerReason) != "" || o.Rollback != nil || len(o.UnresolvedEvidence) > 0 || len(o.IncompletePhases) > 0 || len(o.IncompleteTaskIDs) > 0 {
			return fmt.Errorf("verified seal cannot contain forced-incomplete fields")
		}
	case SealDispositionForcedIncomplete:
		if o.OutcomeKind != OutcomeKindForcedIncompleteClosure {
			return fmt.Errorf("outcome_kind must be %q for forced_incomplete seal", OutcomeKindForcedIncompleteClosure)
		}
		if strings.TrimSpace(o.OwnerReason) == "" {
			return fmt.Errorf("owner_reason is required for forced_incomplete seal")
		}
		if len(o.UnresolvedEvidence) == 0 {
			return fmt.Errorf("unresolved_evidence is required for forced_incomplete seal")
		}
		if o.Rollback == nil {
			return fmt.Errorf("rollback is required for forced_incomplete seal")
		}
		if err := o.Rollback.Validate(); err != nil {
			return fmt.Errorf("rollback: %w", err)
		}
	}
	return nil
}

// IsVerifiedCompletion is true only for a fully valid verified outcome.
func (o SealOutcome) IsVerifiedCompletion() bool {
	return o.Disposition == SealDispositionVerified &&
		o.OutcomeKind == OutcomeKindVerifiedCompletion &&
		o.Validate() == nil
}

// Validate checks a rollback reference.
func (r LifecycleRollback) Validate() error {
	if strings.TrimSpace(r.CheckpointID) == "" {
		return fmt.Errorf("checkpoint_id is required")
	}
	if len(r.Evidence) == 0 {
		return fmt.Errorf("evidence is required")
	}
	return validateLifecycleEvidence(r.Evidence)
}

// Validate checks a seal reference retained by an archive.
func (r SealOutcomeReference) Validate() error {
	if strings.TrimSpace(r.OutcomeID) == "" {
		return fmt.Errorf("outcome_id is required")
	}
	if !r.Disposition.Valid() {
		return fmt.Errorf("disposition: invalid seal disposition %q", r.Disposition)
	}
	if strings.TrimSpace(r.TransactionID) == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if r.Disposition == SealDispositionForcedIncomplete && strings.TrimSpace(r.OwnerReason) == "" {
		return fmt.Errorf("owner_reason is required for forced_incomplete seal reference")
	}
	if r.Disposition == SealDispositionVerified && strings.TrimSpace(r.OwnerReason) != "" {
		return fmt.Errorf("verified seal reference cannot contain owner_reason")
	}
	return nil
}

// Validate checks archive digests, closure identity, and cross-references.
func (m ArchiveManifest) Validate() error {
	if err := validateLifecycleHeader(m.SchemaVersion, m.Command, m.OutcomeKind, m.StateEffect, m.Provenance); err != nil {
		return err
	}
	if m.OutcomeKind != OutcomeKindArchived {
		return fmt.Errorf("outcome_kind must be %q for an archive manifest", OutcomeKindArchived)
	}
	if strings.TrimSpace(m.ArchiveID) == "" {
		return fmt.Errorf("archive_id is required")
	}
	if strings.TrimSpace(m.ManifestDigest) == "" {
		return fmt.Errorf("manifest_digest is required")
	}
	if err := m.Seal.Validate(); err != nil {
		return fmt.Errorf("seal_outcome: %w", err)
	}
	if len(m.Entries) == 0 {
		return fmt.Errorf("entries are required")
	}
	for i := range m.Entries {
		if err := m.Entries[i].Validate(); err != nil {
			return fmt.Errorf("entries[%d]: %w", i, err)
		}
	}
	if len(m.CrossReferences) == 0 {
		return fmt.Errorf("cross_references are required")
	}
	for i := range m.CrossReferences {
		if err := m.CrossReferences[i].Validate(); err != nil {
			return fmt.Errorf("cross_references[%d]: %w", i, err)
		}
	}
	if err := m.Transaction.Validate(); err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	if m.Receipt == nil {
		return fmt.Errorf("receipt is required")
	}
	if err := m.Receipt.Validate(); err != nil {
		return fmt.Errorf("receipt: %w", err)
	}
	if m.Recovery != nil {
		if err := m.Recovery.Validate(); err != nil {
			return fmt.Errorf("recovery: %w", err)
		}
	}
	return validateLifecycleCollections(m.Changes, m.Evidence, m.Verification, m.Warnings, m.Debt, m.Blockers, m.Decisions)
}

// Validate checks one archive entry. ArchiveDigest must match SourceDigest so a
// validated manifest cannot claim integrity from source metadata alone.
func (e ArchiveEntry) Validate() error {
	clean := filepath.Clean(e.Path)
	if strings.TrimSpace(e.Path) == "" || filepath.IsAbs(e.Path) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path must be a contained relative path")
	}
	if strings.TrimSpace(e.Kind) == "" {
		return fmt.Errorf("kind is required")
	}
	if e.Size < 0 {
		return fmt.Errorf("size cannot be negative")
	}
	if strings.TrimSpace(e.SourceDigest) == "" {
		return fmt.Errorf("source_digest is required")
	}
	if strings.TrimSpace(e.ArchiveDigest) == "" {
		return fmt.Errorf("archive_digest is required")
	}
	if e.ArchiveDigest != e.SourceDigest {
		return fmt.Errorf("archive_digest does not match source_digest")
	}
	if !e.Provenance.Valid() {
		return fmt.Errorf("provenance: invalid recovery provenance %q", e.Provenance)
	}
	return nil
}

// Validate checks one archive cross-reference.
func (r ArchiveCrossReference) Validate() error {
	if strings.TrimSpace(r.Kind) == "" {
		return fmt.Errorf("kind is required")
	}
	if strings.TrimSpace(r.SourceID) == "" {
		return fmt.Errorf("source_id is required")
	}
	if strings.TrimSpace(r.TargetID) == "" {
		return fmt.Errorf("target_id is required")
	}
	return nil
}

// Validate checks a state/session archive reference.
func (r ArchiveReference) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(r.TransactionID) == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if strings.TrimSpace(r.ManifestDigest) == "" {
		return fmt.Errorf("manifest_digest is required")
	}
	if strings.TrimSpace(r.Path) == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// Validate rejects unsupported delivery claims and live acknowledgements with
// no named actor/evidence.
func (r SignalDeliveryReceipt) Validate() error {
	if err := validateLifecycleHeader(r.SchemaVersion, r.Command, r.OutcomeKind, r.StateEffect, r.Provenance); err != nil {
		return err
	}
	if strings.TrimSpace(r.ReceiptID) == "" {
		return fmt.Errorf("receipt_id is required")
	}
	if strings.TrimSpace(r.SignalID) == "" {
		return fmt.Errorf("signal_id is required")
	}
	if !r.Delivery.Valid() {
		return fmt.Errorf("delivery: invalid signal delivery %q", r.Delivery)
	}
	if err := r.Transaction.Validate(); err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	if r.Acknowledgement != nil {
		if err := r.Acknowledgement.Validate(); err != nil {
			return fmt.Errorf("acknowledgement: %w", err)
		}
	}
	if r.Delivery == SignalDeliveryLive && r.Acknowledgement == nil {
		return fmt.Errorf("acknowledgement is required for live delivery")
	}
	if r.Delivery != SignalDeliveryLive && len(r.FallbackEvidence) == 0 {
		return fmt.Errorf("fallback_evidence is required for %s delivery", r.Delivery)
	}
	if err := validateLifecycleEvidence(r.EffectEvidence); err != nil {
		return fmt.Errorf("effect_evidence: %w", err)
	}
	if err := validateLifecycleEvidence(r.FallbackEvidence); err != nil {
		return fmt.Errorf("fallback_evidence: %w", err)
	}
	if r.Recovery != nil {
		if err := r.Recovery.Validate(); err != nil {
			return fmt.Errorf("recovery: %w", err)
		}
	}
	return validateLifecycleCollections(r.Changes, r.Evidence, r.Verification, r.Warnings, r.Debt, r.Blockers, r.Decisions)
}

// Validate checks a live acknowledgement has both the actor and evidence
// needed to support the claim.
func (a SignalAcknowledgement) Validate() error {
	if strings.TrimSpace(a.ActorID) == "" {
		return fmt.Errorf("actor_id is required")
	}
	if strings.TrimSpace(a.EvidenceID) == "" {
		return fmt.Errorf("evidence_id is required")
	}
	return nil
}

// Validate checks a transaction reference.
func (r LifecycleTransactionReference) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if !r.Stage.Valid() {
		return fmt.Errorf("stage: invalid transaction stage %q", r.Stage)
	}
	return nil
}

// Validate checks a receipt reference.
func (r LifecycleReceiptReference) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

func validateLifecycleHeader(schemaVersion, command string, outcome OutcomeKind, effect LifecycleStateEffect, provenance RecoveryProvenance) error {
	if schemaVersion != LifecycleSchemaVersion {
		return fmt.Errorf("schema_version must be %q", LifecycleSchemaVersion)
	}
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command is required")
	}
	if !outcome.Valid() {
		return fmt.Errorf("outcome_kind: invalid outcome kind %q", outcome)
	}
	if !effect.Valid() {
		return fmt.Errorf("state_effect: invalid state effect %q", effect)
	}
	if !provenance.Valid() {
		return fmt.Errorf("provenance: invalid recovery provenance %q", provenance)
	}
	return nil
}

func validateLifecycleCollections(changes []LifecycleChange, evidence []LifecycleEvidence, verification []LifecycleVerification, warnings, debt, blockers []LifecycleIssue, decisions []LifecycleDecision) error {
	for i := range changes {
		if strings.TrimSpace(changes[i].Target) == "" {
			return fmt.Errorf("changes[%d].target is required", i)
		}
		if strings.TrimSpace(changes[i].Action) == "" {
			return fmt.Errorf("changes[%d].action is required", i)
		}
	}
	if err := validateLifecycleEvidence(evidence); err != nil {
		return fmt.Errorf("evidence: %w", err)
	}
	for i := range verification {
		if strings.TrimSpace(verification[i].Name) == "" {
			return fmt.Errorf("verification[%d].name is required", i)
		}
	}
	for name, issues := range map[string][]LifecycleIssue{
		"warnings": warnings,
		"debt":     debt,
		"blockers": blockers,
	} {
		for i := range issues {
			if strings.TrimSpace(issues[i].ID) == "" {
				return fmt.Errorf("%s[%d].id is required", name, i)
			}
			if strings.TrimSpace(issues[i].Summary) == "" {
				return fmt.Errorf("%s[%d].summary is required", name, i)
			}
		}
	}
	for i := range decisions {
		if strings.TrimSpace(decisions[i].ID) == "" {
			return fmt.Errorf("decisions[%d].id is required", i)
		}
		if strings.TrimSpace(decisions[i].Scope) == "" {
			return fmt.Errorf("decisions[%d].scope is required", i)
		}
		if strings.TrimSpace(decisions[i].Summary) == "" {
			return fmt.Errorf("decisions[%d].summary is required", i)
		}
	}
	return nil
}

func validateLifecycleEvidence(evidence []LifecycleEvidence) error {
	for i := range evidence {
		if strings.TrimSpace(evidence[i].ID) == "" {
			return fmt.Errorf("evidence[%d].id is required", i)
		}
	}
	return nil
}

func validateWorkerLineage(lineage []LifecycleWorkerLineage) error {
	for i := range lineage {
		if strings.TrimSpace(lineage[i].WorkerID) == "" {
			return fmt.Errorf("lineage[%d].worker_id is required", i)
		}
		if strings.TrimSpace(lineage[i].Caste) == "" {
			return fmt.Errorf("lineage[%d].caste is required", i)
		}
		if strings.TrimSpace(lineage[i].Status) == "" {
			return fmt.Errorf("lineage[%d].status is required", i)
		}
	}
	return nil
}
