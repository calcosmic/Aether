package colony

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestLifecycleEnums(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		wire  string
	}{
		{name: "survey missing", value: SurveyFreshnessMissing, wire: "missing"},
		{name: "survey fresh", value: SurveyFreshnessFresh, wire: "fresh"},
		{name: "survey stale", value: SurveyFreshnessStale, wire: "stale"},
		{name: "survey unavailable", value: SurveyFreshnessUnavailable, wire: "unavailable"},
		{name: "transaction validating", value: TransactionStageValidating, wire: "validating"},
		{name: "transaction staging", value: TransactionStageStaging, wire: "staging"},
		{name: "transaction staged", value: TransactionStageStaged, wire: "staged"},
		{name: "transaction intent", value: TransactionStageIntentRecorded, wire: "intent_recorded"},
		{name: "transaction committing", value: TransactionStageCommitting, wire: "committing"},
		{name: "transaction committed", value: TransactionStageCommitted, wire: "committed"},
		{name: "transaction verifying", value: TransactionStageVerifying, wire: "verifying"},
		{name: "transaction verified", value: TransactionStageVerified, wire: "verified"},
		{name: "transaction rolling back", value: TransactionStageRollingBack, wire: "rolling_back"},
		{name: "transaction rolled back", value: TransactionStageRolledBack, wire: "rolled_back"},
		{name: "transaction recovery", value: TransactionStageRecoveryRequired, wire: "recovery_required"},
		{name: "recovery confirmed", value: RecoveryProvenanceConfirmed, wire: "confirmed"},
		{name: "recovery reconstructed", value: RecoveryProvenanceReconstructed, wire: "reconstructed"},
		{name: "recovery conflicting", value: RecoveryProvenanceConflicting, wire: "conflicting"},
		{name: "recovery unknown", value: RecoveryProvenanceUnknown, wire: "unknown"},
		{name: "signal live", value: SignalDeliveryLive, wire: "live"},
		{name: "signal boundary", value: SignalDeliveryNextSafeBoundary, wire: "next_safe_boundary"},
		{name: "signal unsupported", value: SignalDeliveryUnsupported, wire: "unsupported"},
		{name: "seal verified", value: SealDispositionVerified, wire: "verified"},
		{name: "seal forced", value: SealDispositionForcedIncomplete, wire: "forced_incomplete"},
		{name: "outcome in progress", value: OutcomeKindInProgress, wire: "in_progress"},
		{name: "outcome completed", value: OutcomeKindCompleted, wire: "completed"},
		{name: "outcome no change", value: OutcomeKindNoChange, wire: "no_change"},
		{name: "outcome refused", value: OutcomeKindRefused, wire: "refused"},
		{name: "outcome paused", value: OutcomeKindPaused, wire: "paused"},
		{name: "outcome resumed", value: OutcomeKindResumed, wire: "resumed"},
		{name: "outcome archived", value: OutcomeKindArchived, wire: "archived"},
		{name: "outcome failed", value: OutcomeKindFailed, wire: "failed"},
		{name: "outcome recovery", value: OutcomeKindRecoveryRequired, wire: "recovery_required"},
		{name: "outcome verified completion", value: OutcomeKindVerifiedCompletion, wire: "verified_completion"},
		{name: "outcome forced closure", value: OutcomeKindForcedIncompleteClosure, wire: "forced_incomplete_closure"},
		{name: "effect none", value: LifecycleStateEffectNone, wire: "none"},
		{name: "effect committed", value: LifecycleStateEffectCommitted, wire: "committed"},
		{name: "effect rolled back", value: LifecycleStateEffectRolledBack, wire: "rolled_back"},
		{name: "effect retained", value: LifecycleStateEffectRetained, wire: "retained"},
		{name: "effect recovery", value: LifecycleStateEffectRecoveryRequired, wire: "recovery_required"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal valid enum: %v", err)
			}
			if got, want := string(encoded), `"`+tt.wire+`"`; got != want {
				t.Fatalf("wire value = %s, want %s", got, want)
			}

			target := reflect.New(reflect.TypeOf(tt.value))
			if err := json.Unmarshal(encoded, target.Interface()); err != nil {
				t.Fatalf("unmarshal valid enum: %v", err)
			}
			if got := target.Elem().Interface(); !reflect.DeepEqual(got, tt.value) {
				t.Fatalf("round trip = %#v, want %#v", got, tt.value)
			}
		})
	}

	invalidTargets := []any{
		new(SurveyFreshness),
		new(LifecycleTransactionStage),
		new(RecoveryProvenance),
		new(SignalDelivery),
		new(SealDisposition),
		new(OutcomeKind),
		new(LifecycleStateEffect),
	}
	for _, target := range invalidTargets {
		if err := json.Unmarshal([]byte(`"future_value"`), target); err == nil {
			t.Errorf("%T accepted an unknown wire value", target)
		}
	}
}

func TestLifecycleReceipt(t *testing.T) {
	t.Parallel()

	receipt := validLifecycleReceipt()
	if err := receipt.Validate(); err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}

	first, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal receipt: %v", err)
	}
	second, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal receipt again: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("receipt serialization is not deterministic:\n%s\n%s", first, second)
	}
	for _, key := range []string{
		`"schema_version"`, `"command"`, `"outcome_kind"`, `"projection_revision"`,
		`"changes"`, `"evidence"`, `"verification"`, `"warnings"`, `"debt"`,
		`"blockers"`, `"decisions"`, `"state_effect"`, `"transaction"`,
		`"receipt_id"`, `"recovery"`, `"provenance"`,
	} {
		if !strings.Contains(string(first), key) {
			t.Errorf("serialized receipt missing %s: %s", key, first)
		}
	}

	missingSchema := receipt
	missingSchema.SchemaVersion = ""
	assertLifecycleValidationError(t, missingSchema.Validate(), "schema_version")

	missingTransaction := receipt
	missingTransaction.Transaction.ID = ""
	assertLifecycleValidationError(t, missingTransaction.Validate(), "transaction")

	transaction := validLifecycleTransactionRecord()
	if err := transaction.Validate(); err != nil {
		t.Fatalf("valid transaction rejected: %v", err)
	}
	transaction.TransactionID = ""
	assertLifecycleValidationError(t, transaction.Validate(), "transaction_id")

	recovery := LifecycleRecovery{
		Provenance: RecoveryProvenanceReconstructed,
		Facts: []LifecycleRecoveryFact{{
			Name:       "restart_point",
			Summary:    "Phase 3 can restart after task 1",
			Provenance: RecoveryProvenanceReconstructed,
			Evidence:   []LifecycleEvidence{validLifecycleEvidence("evidence-recovery")},
		}},
	}
	if err := recovery.Validate(); err != nil {
		t.Fatalf("valid recovery rejected: %v", err)
	}
	recovery.Facts[0].Provenance = RecoveryProvenance("guessed")
	assertLifecycleValidationError(t, recovery.Validate(), "provenance")

	t.Run("signal delivery requires evidence for claims", func(t *testing.T) {
		boundary := validSignalDeliveryReceipt()
		if err := boundary.Validate(); err != nil {
			t.Fatalf("valid boundary receipt rejected: %v", err)
		}

		live := boundary
		live.Delivery = SignalDeliveryLive
		live.Acknowledgement = &SignalAcknowledgement{
			ActorID:    "worker-1",
			EvidenceID: "evidence-ack",
		}
		live.FallbackEvidence = nil
		if err := live.Validate(); err != nil {
			t.Fatalf("evidenced live delivery rejected: %v", err)
		}

		live.Acknowledgement = nil
		assertLifecycleValidationError(t, live.Validate(), "acknowledgement")

		boundary.FallbackEvidence = nil
		assertLifecycleValidationError(t, boundary.Validate(), "fallback_evidence")
	})
}

func TestLifecyclePauseHandoff(t *testing.T) {
	t.Parallel()

	handoff := validPauseHandoff()
	if err := handoff.Validate(); err != nil {
		t.Fatalf("valid handoff rejected: %v", err)
	}

	missingID := handoff
	missingID.HandoffID = ""
	assertLifecycleValidationError(t, missingID.Validate(), "handoff_id")

	missingTransaction := handoff
	missingTransaction.Transaction.ID = ""
	assertLifecycleValidationError(t, missingTransaction.Validate(), "transaction")

	missingAttempt := handoff
	missingAttempt.AttemptID = ""
	missingAttempt.RunID = ""
	assertLifecycleValidationError(t, missingAttempt.Validate(), "attempt_id")
}

func TestLifecycleSealOutcome(t *testing.T) {
	t.Parallel()

	verified := validVerifiedSealOutcome()
	if err := verified.Validate(); err != nil {
		t.Fatalf("valid verified outcome rejected: %v", err)
	}
	if !verified.IsVerifiedCompletion() {
		t.Fatal("verified outcome did not report verified completion")
	}

	forced := validForcedSealOutcome()
	if err := forced.Validate(); err != nil {
		t.Fatalf("valid forced outcome rejected: %v", err)
	}
	if forced.IsVerifiedCompletion() {
		t.Fatal("forced-incomplete outcome reported verified completion")
	}

	withoutReason := forced
	withoutReason.OwnerReason = ""
	assertLifecycleValidationError(t, withoutReason.Validate(), "owner_reason")

	withoutEvidence := forced
	withoutEvidence.UnresolvedEvidence = nil
	assertLifecycleValidationError(t, withoutEvidence.Validate(), "unresolved_evidence")

	withoutRollback := forced
	withoutRollback.Rollback = nil
	assertLifecycleValidationError(t, withoutRollback.Validate(), "rollback")

	mismatchedKind := forced
	mismatchedKind.OutcomeKind = OutcomeKindVerifiedCompletion
	assertLifecycleValidationError(t, mismatchedKind.Validate(), "outcome_kind")
}

func TestLifecycleArchiveManifest(t *testing.T) {
	t.Parallel()

	manifest := validArchiveManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}

	missingDigest := manifest
	missingDigest.Entries = append([]ArchiveEntry(nil), manifest.Entries...)
	missingDigest.Entries[0].SourceDigest = ""
	assertLifecycleValidationError(t, missingDigest.Validate(), "source_digest")

	mismatchedDigest := manifest
	mismatchedDigest.Entries = append([]ArchiveEntry(nil), manifest.Entries...)
	mismatchedDigest.Entries[0].ArchiveDigest = "sha256:different"
	assertLifecycleValidationError(t, mismatchedDigest.Validate(), "archive_digest")

	withoutCrossReference := manifest
	withoutCrossReference.CrossReferences = nil
	assertLifecycleValidationError(t, withoutCrossReference.Validate(), "cross_references")
}

func validLifecycleEvidence(id string) LifecycleEvidence {
	return LifecycleEvidence{
		ID:      id,
		Kind:    "fixture",
		Source:  ".aether/data/fixture.json",
		Digest:  "sha256:fixture",
		Summary: "fixture evidence",
	}
}

func validLifecycleTransactionReference() LifecycleTransactionReference {
	return LifecycleTransactionReference{
		ID:          "transaction-1",
		Stage:       TransactionStageVerified,
		JournalPath: ".aether/data/transactions/transaction-1/intent.json",
	}
}

func validLifecycleReceiptReference() *LifecycleReceiptReference {
	return &LifecycleReceiptReference{ID: "receipt-1", Digest: "sha256:receipt"}
}

func validLifecycleReceipt() LifecycleReceipt {
	return LifecycleReceipt{
		SchemaVersion:      LifecycleSchemaVersion,
		ReceiptID:          "receipt-1",
		Command:            "pause",
		OutcomeKind:        OutcomeKindPaused,
		ProjectionRevision: "revision-7",
		Changes: []LifecycleChange{{
			Target:      ".aether/data/COLONY_STATE.json",
			Action:      "update",
			AfterDigest: "sha256:after",
		}},
		Evidence:     []LifecycleEvidence{validLifecycleEvidence("evidence-1")},
		Verification: []LifecycleVerification{{Name: "state-handoff-cross-reference", Passed: true, EvidenceIDs: []string{"evidence-1"}}},
		Warnings:     []LifecycleIssue{{ID: "warning-1", Summary: "non-blocking warning"}},
		Debt:         []LifecycleIssue{{ID: "debt-1", Summary: "follow-up debt"}},
		Blockers:     []LifecycleIssue{{ID: "blocker-1", Summary: "preserved blocker"}},
		Decisions:    []LifecycleDecision{{ID: "decision-1", Scope: "phase-3", Summary: "keep partial work"}},
		StateEffect:  LifecycleStateEffectCommitted,
		Transaction:  validLifecycleTransactionReference(),
		Recovery: &LifecycleRecovery{
			Provenance: RecoveryProvenanceConfirmed,
			Facts: []LifecycleRecoveryFact{{
				Name:       "handoff",
				Summary:    "handoff and state agree",
				Provenance: RecoveryProvenanceConfirmed,
				Evidence:   []LifecycleEvidence{validLifecycleEvidence("evidence-recovery")},
			}},
		},
		Provenance: RecoveryProvenanceConfirmed,
	}
}

func validLifecycleTransactionRecord() LifecycleTransactionRecord {
	return LifecycleTransactionRecord{
		SchemaVersion:      LifecycleSchemaVersion,
		TransactionID:      "transaction-1",
		Command:            "pause",
		OutcomeKind:        OutcomeKindInProgress,
		ProjectionRevision: "revision-7",
		Stage:              TransactionStageStaged,
		BaselineDigest:     "sha256:baseline",
		Changes:            []LifecycleChange{{Target: ".aether/data/COLONY_STATE.json", Action: "update"}},
		Evidence:           []LifecycleEvidence{validLifecycleEvidence("evidence-transaction")},
		StateEffect:        LifecycleStateEffectNone,
		Provenance:         RecoveryProvenanceConfirmed,
	}
}

func validPauseHandoff() PauseHandoff {
	return PauseHandoff{
		SchemaVersion:      LifecycleSchemaVersion,
		HandoffID:          "handoff-1",
		Command:            "pause",
		OutcomeKind:        OutcomeKindPaused,
		ProjectionRevision: "revision-7",
		PhaseID:            3,
		TaskIDs:            []string{"task-1"},
		AttemptID:          "attempt-1",
		RunID:              "run-1",
		SafeBoundary:       "after task-1",
		RestartPoint:       "phase-3/task-2",
		ContextDigest:      "sha256:context",
		Repository: LifecycleRepositoryEvidence{
			Head:        "abc123",
			DirtyDigest: "sha256:dirty",
		},
		Lineage: []LifecycleWorkerLineage{{
			WorkerID: "worker-1",
			Caste:    "builder",
			Status:   "completed",
		}},
		Changes:     []LifecycleChange{{Target: ".aether/data/handoff.json", Action: "create", AfterDigest: "sha256:handoff"}},
		Evidence:    []LifecycleEvidence{validLifecycleEvidence("evidence-handoff")},
		Decisions:   []LifecycleDecision{{ID: "decision-1", Scope: "task-1", Summary: "resume task-2"}},
		StateEffect: LifecycleStateEffectCommitted,
		Transaction: validLifecycleTransactionReference(),
		Receipt:     validLifecycleReceiptReference(),
		Recovery: &LifecycleRecovery{
			Provenance: RecoveryProvenanceConfirmed,
			Facts: []LifecycleRecoveryFact{{
				Name:       "safe_boundary",
				Summary:    "task-1 completed before pause",
				Provenance: RecoveryProvenanceConfirmed,
				Evidence:   []LifecycleEvidence{validLifecycleEvidence("evidence-boundary")},
			}},
		},
		Provenance: RecoveryProvenanceConfirmed,
	}
}

func validVerifiedSealOutcome() SealOutcome {
	return SealOutcome{
		SchemaVersion:      LifecycleSchemaVersion,
		OutcomeID:          "seal-1",
		Command:            "seal",
		OutcomeKind:        OutcomeKindVerifiedCompletion,
		ProjectionRevision: "revision-7",
		Disposition:        SealDispositionVerified,
		CompletedPhases:    []int{1, 2, 3},
		Evidence:           []LifecycleEvidence{validLifecycleEvidence("evidence-seal")},
		Verification:       []LifecycleVerification{{Name: "required-gates", Passed: true, EvidenceIDs: []string{"evidence-seal"}}},
		StateEffect:        LifecycleStateEffectCommitted,
		Transaction:        validLifecycleTransactionReference(),
		Receipt:            validLifecycleReceiptReference(),
		Provenance:         RecoveryProvenanceConfirmed,
	}
}

func validForcedSealOutcome() SealOutcome {
	return SealOutcome{
		SchemaVersion:      LifecycleSchemaVersion,
		OutcomeID:          "seal-forced-1",
		Command:            "seal",
		OutcomeKind:        OutcomeKindForcedIncompleteClosure,
		ProjectionRevision: "revision-7",
		Disposition:        SealDispositionForcedIncomplete,
		CompletedPhases:    []int{1, 2},
		IncompletePhases:   []int{3},
		OwnerReason:        "deployment window closed",
		UnresolvedEvidence: []LifecycleEvidence{validLifecycleEvidence("evidence-unresolved")},
		Rollback: &LifecycleRollback{
			CheckpointID: "checkpoint-before-seal",
			Evidence:     []LifecycleEvidence{validLifecycleEvidence("evidence-rollback")},
		},
		Blockers:    []LifecycleIssue{{ID: "blocker-1", Summary: "phase 3 remains incomplete"}},
		StateEffect: LifecycleStateEffectCommitted,
		Transaction: validLifecycleTransactionReference(),
		Receipt:     validLifecycleReceiptReference(),
		Provenance:  RecoveryProvenanceConfirmed,
	}
}

func validArchiveManifest() ArchiveManifest {
	return ArchiveManifest{
		SchemaVersion:      LifecycleSchemaVersion,
		ArchiveID:          "archive-1",
		Command:            "entomb",
		OutcomeKind:        OutcomeKindArchived,
		ProjectionRevision: "revision-7",
		ManifestDigest:     "sha256:manifest",
		Seal: SealOutcomeReference{
			OutcomeID:     "seal-1",
			Disposition:   SealDispositionVerified,
			TransactionID: "transaction-seal-1",
		},
		Entries: []ArchiveEntry{{
			Path:          "COLONY_STATE.json",
			Kind:          "state",
			Size:          128,
			SourceDigest:  "sha256:state",
			ArchiveDigest: "sha256:state",
			Provenance:    RecoveryProvenanceConfirmed,
		}},
		CrossReferences: []ArchiveCrossReference{{
			Kind:     "seal_outcome",
			SourceID: "archive-1",
			TargetID: "seal-1",
			Digest:   "sha256:seal",
		}},
		Evidence:    []LifecycleEvidence{validLifecycleEvidence("evidence-archive")},
		StateEffect: LifecycleStateEffectCommitted,
		Transaction: validLifecycleTransactionReference(),
		Receipt:     validLifecycleReceiptReference(),
		Provenance:  RecoveryProvenanceConfirmed,
	}
}

func validSignalDeliveryReceipt() SignalDeliveryReceipt {
	return SignalDeliveryReceipt{
		SchemaVersion:       LifecycleSchemaVersion,
		ReceiptID:           "signal-receipt-1",
		Command:             "focus",
		OutcomeKind:         OutcomeKindCompleted,
		ProjectionRevision:  "revision-7",
		SignalID:            "signal-1",
		Delivery:            SignalDeliveryNextSafeBoundary,
		AffectedJobIDs:      []string{"job-1"},
		AffectedDecisionIDs: []string{"decision-1"},
		FallbackEvidence:    []LifecycleEvidence{validLifecycleEvidence("evidence-fallback")},
		Evidence:            []LifecycleEvidence{validLifecycleEvidence("evidence-signal")},
		StateEffect:         LifecycleStateEffectCommitted,
		Transaction:         validLifecycleTransactionReference(),
		Provenance:          RecoveryProvenanceConfirmed,
	}
}

func assertLifecycleValidationError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error containing %q", field)
	}
	if !strings.Contains(err.Error(), field) {
		t.Fatalf("validation error %q does not contain %q", err, field)
	}
}
