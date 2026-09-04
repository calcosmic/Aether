package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestAgencyReceipt199DeliveryEvidence(t *testing.T) {
	signal := agencySignal199("FOCUS", "Keep authentication changes small")

	unsupported, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{})
	if err != nil {
		t.Fatalf("build unsupported receipt: %v", err)
	}
	if got := unsupported.Receipt.Delivery; got != colony.SignalDeliveryUnsupported {
		t.Fatalf("delivery = %q, want unsupported", got)
	}
	if got := unsupported.Delivery; got != "unsupported" {
		t.Fatalf("display delivery = %q, want unsupported", got)
	}

	boundary := colony.LifecycleEvidence{ID: "boundary-build-1", Kind: "worker_context_boundary", Source: "build-manifest"}
	nextBoundary, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{LifecycleBoundary: &boundary})
	if err != nil {
		t.Fatalf("build next-boundary receipt: %v", err)
	}
	if got := nextBoundary.Receipt.Delivery; got != colony.SignalDeliveryNextSafeBoundary {
		t.Fatalf("delivery = %q, want next_safe_boundary", got)
	}

	liveEvidence := colony.LifecycleEvidence{ID: "delivery-live-1", Kind: "runtime_delivery", Source: "typed-event"}
	if _, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{LiveDelivery: &liveEvidence}); err == nil || !strings.Contains(err.Error(), "live delivery") {
		t.Fatalf("live delivery without acknowledgement error = %v", err)
	}

	ackEvidence := colony.LifecycleEvidence{ID: "ack-1", Kind: "runtime_acknowledgement", Source: "typed-event"}
	live, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{
		LiveDelivery:            &liveEvidence,
		Acknowledgement:         &colony.SignalAcknowledgement{ActorID: "builder-1", EvidenceID: ackEvidence.ID},
		AcknowledgementEvidence: &ackEvidence,
	})
	if err != nil {
		t.Fatalf("build evidenced live receipt: %v", err)
	}
	if got := live.Receipt.Delivery; got != colony.SignalDeliveryLive {
		t.Fatalf("delivery = %q, want live", got)
	}
	if err := live.Receipt.Validate(); err != nil {
		t.Fatalf("live receipt validation: %v", err)
	}
}

func TestAgencyReceipt199Acknowledgement(t *testing.T) {
	signal := agencySignal199("FEEDBACK", "Prefer the smaller refactor")
	result, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Acknowledgement; got != AgencyAcknowledgementPending {
		t.Fatalf("acknowledgement = %q, want %q", got, AgencyAcknowledgementPending)
	}
	if result.Receipt.Acknowledgement != nil {
		t.Fatalf("receipt invented acknowledgement: %#v", result.Receipt.Acknowledgement)
	}

	liveEvidence := colony.LifecycleEvidence{ID: "delivery-live-2", Kind: "runtime_delivery"}
	ackEvidence := colony.LifecycleEvidence{ID: "ack-2", Kind: "runtime_acknowledgement"}
	_, err = BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{
		LiveDelivery:            &liveEvidence,
		Acknowledgement:         &colony.SignalAcknowledgement{ActorID: "builder-2", EvidenceID: "not-linked"},
		AcknowledgementEvidence: &ackEvidence,
	})
	if err == nil || !strings.Contains(err.Error(), "linked acknowledgement evidence") {
		t.Fatalf("unlinked acknowledgement error = %v", err)
	}
}

func TestAgencyReceipt199MeasuredEffect(t *testing.T) {
	signal := agencySignal199("FOCUS", "Protect the lifecycle boundary")
	result, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.MeasuredEffect; got != AgencyMeasuredEffectPending {
		t.Fatalf("measured effect = %q, want %q", got, AgencyMeasuredEffectPending)
	}
	if len(result.Receipt.EffectEvidence) != 0 || len(result.Receipt.AffectedDecisionIDs) != 0 {
		t.Fatalf("receipt invented measured effect: %#v", result.Receipt)
	}

	effect := colony.LifecycleEvidence{ID: "effect-1", Kind: "changed_decision"}
	decision := colony.LifecycleDecision{ID: "decision-1", Scope: "job-1", Summary: "Changed the implementation choice", EvidenceIDs: []string{"other-evidence"}}
	if _, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{ChangedDecision: &decision, EffectEvidence: &effect}); err == nil || !strings.Contains(err.Error(), "linked changed-decision evidence") {
		t.Fatalf("unlinked measured effect error = %v", err)
	}

	decision.EvidenceIDs = []string{effect.ID}
	measured, err := BuildAgencySignalResult(signal, false, AgencyReceiptEvidence{ChangedDecision: &decision, EffectEvidence: &effect})
	if err != nil {
		t.Fatalf("build measured receipt: %v", err)
	}
	if got := measured.Receipt.AffectedDecisionIDs; len(got) != 1 || got[0] != decision.ID {
		t.Fatalf("affected decisions = %#v", got)
	}
	if got := measured.Receipt.EffectEvidence; len(got) != 1 || got[0].ID != effect.ID {
		t.Fatalf("effect evidence = %#v", got)
	}
}

func TestAgencyReceipt199WorkEffect(t *testing.T) {
	active := []string{"job-auth", "job-docs"}
	cases := []struct {
		name       string
		signalType string
		evidence   AgencyReceiptEvidence
		want       AgencyWorkEffect
		wantPaused string
		wantErr    string
	}{
		{name: "focus continues independent work", signalType: "FOCUS", evidence: AgencyReceiptEvidence{ActiveJobIDs: active}, want: AgencyWorkContinuedIndependent},
		{name: "feedback continues independent work", signalType: "FEEDBACK", evidence: AgencyReceiptEvidence{ActiveJobIDs: active}, want: AgencyWorkContinuedIndependent},
		{name: "redirect without proven conflict continues work", signalType: "REDIRECT", evidence: AgencyReceiptEvidence{ActiveJobIDs: active}, want: AgencyWorkContinuedIndependent},
		{name: "no active work", signalType: "FOCUS", want: AgencyWorkNoActive},
		{name: "unrelated redirect cannot pause", signalType: "REDIRECT", evidence: AgencyReceiptEvidence{ActiveJobIDs: active, ConflictingJobID: "job-unrelated", ConflictEvidence: &colony.LifecycleEvidence{ID: "conflict-unrelated"}}, wantErr: "not an active job"},
		{name: "proven conflict pauses one named job", signalType: "REDIRECT", evidence: AgencyReceiptEvidence{ActiveJobIDs: active, ConflictingJobID: "job-auth", ConflictEvidence: &colony.LifecycleEvidence{ID: "conflict-auth", Kind: "declared_scope_conflict"}}, want: AgencyWorkConflictingJobPaused, wantPaused: "job-auth"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := BuildAgencySignalResult(agencySignal199(tc.signalType, "Owner wording"), false, tc.evidence)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.WorkEffect != tc.want {
				t.Fatalf("work effect = %q, want %q", result.WorkEffect, tc.want)
			}
			if result.PausedJobID != tc.wantPaused {
				t.Fatalf("paused job = %q, want %q", result.PausedJobID, tc.wantPaused)
			}
		})
	}
}

func agencySignal199(signalType, wording string) colony.PheromoneSignal {
	content, _ := json.Marshal(map[string]string{"text": wording})
	return colony.PheromoneSignal{
		ID:       "sig-199-12",
		Type:     signalType,
		Priority: "normal",
		Source:   "user",
		Active:   true,
		Content:  content,
	}
}
