package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
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

// lifecycleCeremonySlotViolations is the pure comparison at the heart of the
// equal-ceremony invariant: given the slot set recorded for each verdict, it
// reports -- by slot and by the exact verdicts missing it -- any canonical
// slot present for at least one verdict and absent for at least one other.
// It has no dependency on *testing.T so it can be exercised directly against
// both real production output and a synthetic broken fixture.
func lifecycleCeremonySlotViolations(bySlot map[colony.WorkOutcome][]LifecycleCloseoutSlot, canonical []LifecycleCloseoutSlot) []string {
	verdicts := make([]colony.WorkOutcome, 0, len(bySlot))
	for verdict := range bySlot {
		verdicts = append(verdicts, verdict)
	}
	sort.Slice(verdicts, func(i, j int) bool { return verdicts[i] < verdicts[j] })

	var violations []string
	for _, slot := range canonical {
		var present, missing []colony.WorkOutcome
		for _, verdict := range verdicts {
			has := false
			for _, s := range bySlot[verdict] {
				if s == slot {
					has = true
					break
				}
			}
			if has {
				present = append(present, verdict)
			} else {
				missing = append(missing, verdict)
			}
		}
		if len(present) > 0 && len(missing) > 0 {
			violations = append(violations, fmt.Sprintf("slot %q is present for %v but missing for %v", slot, present, missing))
		}
	}
	return violations
}

// lifecycleWorkOutcomeLabelCoverageViolations reports, by name, any declared
// verdict with no entry in labels, and any label entry whose key is not a
// declared verdict -- so a seventh verdict can never ship silently
// unlabelled, and a stray label entry can never silently orphan itself.
func lifecycleWorkOutcomeLabelCoverageViolations(labels map[colony.WorkOutcome]string, verdicts []colony.WorkOutcome) []string {
	declared := make(map[colony.WorkOutcome]bool, len(verdicts))
	for _, verdict := range verdicts {
		declared[verdict] = true
	}
	var violations []string
	for _, verdict := range verdicts {
		if strings.TrimSpace(labels[verdict]) == "" {
			violations = append(violations, fmt.Sprintf("declared verdict %q has no entry in WorkOutcomeLabels()", verdict))
		}
	}
	for verdict := range labels {
		if !declared[verdict] {
			violations = append(violations, fmt.Sprintf("WorkOutcomeLabels() carries %q, which is not a declared verdict", verdict))
		}
	}
	sort.Strings(violations)
	return violations
}

// TestCloseoutCeremonyIsEqualAcrossVerdicts is the invariant Task 2's
// equality proof cannot hold on its own against a FUTURE regression: it
// enumerates every declared WorkOutcome (colony.AllWorkOutcomes()) and every
// declared LifecycleCloseoutSlot (lifecycleCloseoutCanonicalSlots) from the
// runtime's own vocabularies -- never a list this test maintains -- builds a
// closeout for each verdict from one shared fixture, and fails, naming the
// slot and the verdicts missing it, the moment a future slot is added for
// only some of them. It also proves a declared verdict can never ship
// without a label, and a label entry can never silently point at an
// undeclared verdict.
func TestCloseoutCeremonyIsEqualAcrossVerdicts(t *testing.T) {
	canonical := append([]LifecycleCloseoutSlot(nil), lifecycleCloseoutCanonicalSlots[:]...)
	verdicts := colony.AllWorkOutcomes()

	t.Run("real package holds the invariant", func(t *testing.T) {
		bySlot := make(map[colony.WorkOutcome][]LifecycleCloseoutSlot, len(verdicts))
		for _, verdict := range verdicts {
			bySlot[verdict] = lifecycleWorkOutcomeFixtureCloseout(t, verdict).Slots
		}
		if violations := lifecycleCeremonySlotViolations(bySlot, canonical); len(violations) > 0 {
			t.Fatalf("equal-ceremony invariant violated:\n%s", strings.Join(violations, "\n"))
		}

		if violations := lifecycleWorkOutcomeLabelCoverageViolations(colony.WorkOutcomeLabels(), verdicts); len(violations) > 0 {
			t.Fatalf("label coverage invariant violated:\n%s", strings.Join(violations, "\n"))
		}
	})

	t.Run("a slot populated for one verdict only fails by name", func(t *testing.T) {
		bySlot := make(map[colony.WorkOutcome][]LifecycleCloseoutSlot, len(verdicts))
		for _, verdict := range verdicts {
			bySlot[verdict] = append([]LifecycleCloseoutSlot(nil), canonical...)
		}
		// Break the invariant: strip LifecycleCloseoutNextUp from every
		// verdict except the first, simulating a future slot a later change
		// only wires up for one verdict.
		broken := verdicts[0]
		for _, verdict := range verdicts[1:] {
			var without []LifecycleCloseoutSlot
			for _, slot := range bySlot[verdict] {
				if slot != LifecycleCloseoutNextUp {
					without = append(without, slot)
				}
			}
			bySlot[verdict] = without
		}

		violations := lifecycleCeremonySlotViolations(bySlot, canonical)
		if len(violations) == 0 {
			t.Fatalf("expected the invariant to fail against a fixture where %q is present for %q only", LifecycleCloseoutNextUp, broken)
		}
		var found bool
		for _, violation := range violations {
			if strings.Contains(violation, string(LifecycleCloseoutNextUp)) && strings.Contains(violation, string(broken)) {
				found = true
			}
		}
		if !found {
			t.Fatalf("violations do not name both the slot %q and the verdict %q: %v", LifecycleCloseoutNextUp, broken, violations)
		}
	})

	t.Run("a verdict with no label fails by name", func(t *testing.T) {
		labels := colony.WorkOutcomeLabels()
		delete(labels, colony.WorkOutcomeTimeout)
		violations := lifecycleWorkOutcomeLabelCoverageViolations(labels, verdicts)
		if len(violations) == 0 {
			t.Fatalf("expected the invariant to fail when %q has no label", colony.WorkOutcomeTimeout)
		}
		var found bool
		for _, violation := range violations {
			if strings.Contains(violation, string(colony.WorkOutcomeTimeout)) {
				found = true
			}
		}
		if !found {
			t.Fatalf("violations do not name the unlabelled verdict %q: %v", colony.WorkOutcomeTimeout, violations)
		}
	})
}
