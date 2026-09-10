package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/calcosmic/Aether/pkg/colony"
)

// lifecycleWorkOutcomeFixtureCloseout builds one closeout, populated with
// participants, evidence, and a standing instruction so the fixture can
// distinguish "slot present with content" from "slot present but empty" --
// carrying the given work verdict.
func lifecycleWorkOutcomeFixtureCloseout(t *testing.T, verdict colony.WorkOutcome) LifecycleCloseout {
	t.Helper()
	projection := lifecycleCloseout199Projection(colony.StateEXECUTING, "continue")
	projection.Actors.Value = []LifecycleActorFact{{Name: "Mason-67", Caste: "builder", Status: "completed"}}
	projection.Evidence = []colony.LifecycleEvidence{{ID: "receipt-1", Kind: "receipt", Source: "receipts/continue.json", Summary: "Check receipt verified"}}
	projection.Signals.Value = []colony.PheromoneSignal{{ID: "focus-1", Type: "FOCUS", Active: true, Content: []byte(`{"text":"Keep it focused"}`)}}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindInProgress, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "continue", LifecycleCloseoutDetails{WorkOutcome: verdict}); err != nil {
		t.Fatalf("apply closeout for verdict %q: %v", verdict, err)
	}
	return lifecycleCloseout199MustResult(t, result)
}

// lifecycleWorkOutcomeSparseFixtureCloseout mirrors the fixture above but
// deliberately supplies NOTHING for the optional slots (no participants, no
// evidence, no standing instructions) -- used to prove a non-success verdict
// still renders every canonical slot rather than dropping the empty ones.
func lifecycleWorkOutcomeSparseFixtureCloseout(t *testing.T, verdict colony.WorkOutcome) LifecycleCloseout {
	t.Helper()
	projection := lifecycleCloseout199Projection(colony.StateEXECUTING, "continue")
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindInProgress, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "continue", LifecycleCloseoutDetails{WorkOutcome: verdict}); err != nil {
		t.Fatalf("apply sparse closeout for verdict %q: %v", verdict, err)
	}
	return lifecycleCloseout199MustResult(t, result)
}

// lifecycleWorkOutcomeTokens splits s into lowercase word tokens on any
// non-letter/non-digit boundary, so a token comparison can never be fooled
// by a substring match inside an unrelated longer word.
func lifecycleWorkOutcomeTokens(s string) map[string]bool {
	tokens := map[string]bool{}
	for _, word := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if word != "" {
			tokens[word] = true
		}
	}
	return tokens
}

// TestEveryWorkVerdictRendersTheSameCeremonySlots is the D-05 equal-ceremony
// proof: a closeout built with any of the six verdicts (from one shared
// fixture, content held constant, only the verdict varying) carries the
// same slot set -- the full canonical set, none dropped -- and each
// verdict's rendered card carries that verdict's own label from
// colony.WorkOutcomeLabels(), never a string written a second time at the
// render site.
func TestEveryWorkVerdictRendersTheSameCeremonySlots(t *testing.T) {
	labels := colony.WorkOutcomeLabels()
	var first []LifecycleCloseoutSlot
	for i, verdict := range colony.AllWorkOutcomes() {
		closeout := lifecycleWorkOutcomeFixtureCloseout(t, verdict)
		if i == 0 {
			first = closeout.Slots
		} else if !reflect.DeepEqual(closeout.Slots, first) {
			t.Fatalf("verdict %q slots = %#v, want %#v (from verdict %q)", verdict, closeout.Slots, first, colony.AllWorkOutcomes()[0])
		}

		wantLabel := labels[verdict]
		if closeout.WhatHappened.Verdict != wantLabel {
			t.Errorf("verdict %q WhatHappened.Verdict = %q, want %q (from colony.WorkOutcomeLabels())", verdict, closeout.WhatHappened.Verdict, wantLabel)
		}
		visual := renderLifecycleCloseout(closeout, "claude")
		if !strings.Contains(visual, wantLabel) {
			t.Errorf("verdict %q rendered card missing its label %q:\n%s", verdict, wantLabel, visual)
		}

		// A sparse fixture (nothing to put in the optional slots) must still
		// carry the full canonical slot set -- an empty slot is rendered,
		// never dropped.
		sparse := lifecycleWorkOutcomeSparseFixtureCloseout(t, verdict)
		if !reflect.DeepEqual(sparse.Slots, first) {
			t.Errorf("sparse verdict %q slots = %#v, want the same full set %#v -- an empty slot must render, not disappear", verdict, sparse.Slots, first)
		}
	}
	if len(first) != len(lifecycleCloseoutCanonicalSlots) {
		t.Fatalf("full-ceremony slot count = %d, want %d (every canonical slot, D-05)", len(first), len(lifecycleCloseoutCanonicalSlots))
	}
}

// TestNonSuccessVerdictBorrowsNoSuccessWording proves a non-success verdict
// never dresses itself in the success verdict's language. The success
// verdict's token set is derived from colony.WorkOutcomeLabels() inside the
// test -- never a copy of the wording -- so this assertion cannot go stale
// when the label wording changes.
func TestNonSuccessVerdictBorrowsNoSuccessWording(t *testing.T) {
	labels := colony.WorkOutcomeLabels()
	successTokens := lifecycleWorkOutcomeTokens(labels[colony.WorkOutcomeSuccess])
	if len(successTokens) == 0 {
		t.Fatalf("success label produced no tokens to check against: %q", labels[colony.WorkOutcomeSuccess])
	}
	for _, verdict := range colony.AllWorkOutcomes() {
		if verdict == colony.WorkOutcomeSuccess {
			continue
		}
		closeout := lifecycleWorkOutcomeFixtureCloseout(t, verdict)
		visual := renderLifecycleCloseout(closeout, "claude")
		visualTokens := lifecycleWorkOutcomeTokens(visual)
		for token := range successTokens {
			if visualTokens[token] {
				t.Errorf("verdict %q rendered card shares success-verdict token %q:\n%s", verdict, token, visual)
			}
		}
	}
}

// TestCloseoutWithoutAVerdictIsUnchanged proves a closeout built with no
// work verdict (LifecycleCloseoutDetails.WorkOutcome left at its zero
// value) behaves exactly as it did before this field existed: the new
// WorkOutcome/Verdict fields stay nil/empty, are omitted from JSON entirely
// (never even an empty key), and the closeout is stable across identical
// inputs.
func TestCloseoutWithoutAVerdictIsUnchanged(t *testing.T) {
	build := func(t *testing.T) (LifecycleCloseout, []byte) {
		t.Helper()
		projection := lifecycleCloseout199Projection(colony.StateEXECUTING, "continue")
		projection.Actors.Value = []LifecycleActorFact{{Name: "Mason-67", Caste: "builder", Status: "completed"}}
		projection.Evidence = []colony.LifecycleEvidence{{ID: "receipt-1", Kind: "receipt", Source: "receipts/continue.json", Summary: "Check receipt verified"}}
		projection.Signals.Value = []colony.PheromoneSignal{{ID: "focus-1", Type: "FOCUS", Active: true, Content: []byte(`{"text":"Keep it focused"}`)}}
		result := lifecycleCloseout199Result(projection, colony.OutcomeKindInProgress, colony.LifecycleStateEffectCommitted)
		if err := applyLifecycleCloseout(result, "continue", LifecycleCloseoutDetails{Summary: "Phase check finished."}); err != nil {
			t.Fatalf("apply closeout: %v", err)
		}
		closeout := lifecycleCloseout199MustResult(t, result)
		data, err := json.Marshal(closeout)
		if err != nil {
			t.Fatalf("marshal closeout: %v", err)
		}
		return closeout, data
	}

	first, firstJSON := build(t)
	second, secondJSON := build(t)
	if !reflect.DeepEqual(first, second) || string(firstJSON) != string(secondJSON) {
		t.Fatalf("closeout without a verdict is not stable across identical inputs:\n%s\n---\n%s", firstJSON, secondJSON)
	}
	if first.WorkOutcome != nil {
		t.Fatalf("closeout without a verdict carries a non-nil WorkOutcome: %#v", first.WorkOutcome)
	}
	if first.WhatHappened.Verdict != "" {
		t.Fatalf("closeout without a verdict carries a non-empty WhatHappened.Verdict: %q", first.WhatHappened.Verdict)
	}
	if strings.Contains(string(firstJSON), `"work_outcome"`) {
		t.Fatalf("closeout without a verdict serialised a work_outcome key:\n%s", firstJSON)
	}
	if strings.Contains(string(firstJSON), `"verdict"`) {
		t.Fatalf("closeout without a verdict serialised a verdict key:\n%s", firstJSON)
	}
}
