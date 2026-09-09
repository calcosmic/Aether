package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

var errPlanningTimelineInjected = errors.New("injected planning timeline failure")

func TestPlanningTimelineAppendStoresContentAddressedCardAndIndex(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.September, 7, 14, 5, 0, 0, time.UTC)
	card := validPlanningIterationCardForTest(t, 1, now)
	original := clonePlanningTimelineTestCard(t, card)

	receipt, err := appendPlanningIterationCard(root, card, planningTimelineAppendOptions{
		ReceiptID: "route-pass-1",
	})
	if err != nil {
		t.Fatalf("append planning iteration: %v", err)
	}
	if receipt.CardID == "" || !strings.HasSuffix(receipt.CardID, "-"+receipt.CardHash[:12]) {
		t.Fatalf("card is not content addressed: %#v", receipt)
	}
	if receipt.WriteReceipt.ReceiptID == "" || receipt.WriteReceipt.StateEffect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("write receipt is not a committed lifecycle receipt: %#v", receipt.WriteReceipt)
	}

	storedCard := readPlanningTimelineTestCard(t, root, receipt.CardPath)
	original.ID = receipt.CardID
	original.ContentHash = receipt.CardHash
	if !reflect.DeepEqual(storedCard, original) {
		t.Fatalf("stored card did not preserve the complete pass\nwant: %#v\n got: %#v", original, storedCard)
	}

	index := readPlanningTimelineTestIndex(t, root, receipt.IndexPath)
	if index.RunID != card.RunID || len(index.Entries) != 1 {
		t.Fatalf("timeline index = %#v", index)
	}
	entry := index.Entries[0]
	if entry.Iteration != 1 || entry.CardID != receipt.CardID || entry.CardHash != receipt.CardHash || entry.PreviousCardHash != "" {
		t.Fatalf("timeline entry = %#v", entry)
	}
	if index.TimelineDigest != receipt.TimelineDigest || index.LastCardHash != receipt.CardHash {
		t.Fatalf("timeline digest/last hash = %q/%q, receipt = %q/%q", index.TimelineDigest, index.LastCardHash, receipt.TimelineDigest, receipt.CardHash)
	}

	// The caller retains mutable slices. Changing them after append must not
	// rewrite or alias the immutable on-disk pass.
	card.EvidenceIDs[0] = "mutated-evidence"
	card.DimensionAssessments[0].Rationale = "mutated rationale"
	card.SemanticDelta.AuthorityImpacts[0].Rationale = "mutated authority impact"
	if got := readPlanningTimelineTestCard(t, root, receipt.CardPath); !reflect.DeepEqual(got, original) {
		t.Fatalf("caller mutation changed stored card\nwant: %#v\n got: %#v", original, got)
	}
}

func TestPlanningTimelineAppendRoundTripPreservesCausalDetails(t *testing.T) {
	root := t.TempDir()
	card := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 14, 6, 0, 0, time.UTC))
	card.EvidenceIDs = []string{"evidence-scout", "evidence-route"}
	card.DimensionAssessments[0].Before = 41
	card.DimensionAssessments[0].After = 58
	card.DimensionAssessments[0].FreshEvidenceIDs = []string{"evidence-scout"}
	card.DimensionAssessments[0].ResolvedGapIDs = []string{"gap-old"}
	if err := colony.AddressPlanningDimensionAssessment(&card.DimensionAssessments[0]); err != nil {
		t.Fatalf("re-address changed knowledge assessment: %v", err)
	}
	card.WeakestGap = card.DimensionAssessments[2].RemainingGap
	card.Decision.SelectedGapID = card.WeakestGap.ID
	card.Decision.ResidualGapIDs = []string{card.WeakestGap.ID}
	card.Decision.Reason = colony.PlanningStopContinue
	if err := addressPlanningStopDecision(&card.Decision); err != nil {
		t.Fatalf("re-address changed stop decision: %v", err)
	}
	card.SemanticDelta.AuthorityImpacts[0].AffectedSemanticIDs = []string{"phase-grounded", "task-grounded"}
	if err := colony.AddressPlanningAuthorityImpact(&card.SemanticDelta.AuthorityImpacts[0]); err != nil {
		t.Fatalf("re-address changed authority impact: %v", err)
	}
	if err := colony.AddressPlanningSemanticDelta(&card.SemanticDelta); err != nil {
		t.Fatalf("re-address semantic delta after authority change: %v", err)
	}

	receipt, err := appendPlanningIterationCard(root, card, planningTimelineAppendOptions{ReceiptID: "route-pass-details"})
	if err != nil {
		t.Fatalf("append planning iteration: %v", err)
	}
	stored := readPlanningTimelineTestCard(t, root, receipt.CardPath)
	if stored.DimensionAssessments[0].Before != 41 || stored.DimensionAssessments[0].After != 58 {
		t.Fatalf("before-to-after score was not preserved: %#v", stored.DimensionAssessments[0])
	}
	if !reflect.DeepEqual(stored.EvidenceIDs, []string{"evidence-scout", "evidence-route"}) || !reflect.DeepEqual(stored.DimensionAssessments[0].ResolvedGapIDs, []string{"gap-old"}) {
		t.Fatalf("evidence or resolved gaps were not preserved: %#v", stored)
	}
	if stored.WeakestGap.ID != card.WeakestGap.ID || stored.Decision.Reason != colony.PlanningStopContinue {
		t.Fatalf("weakest gap or disposition was not preserved: %#v", stored)
	}
	if len(stored.SemanticDelta.AuthorityImpacts) != 1 || len(stored.SemanticDelta.AuthorityImpacts[0].AffectedSemanticIDs) != 2 {
		t.Fatalf("semantic delta authority impacts were not preserved: %#v", stored.SemanticDelta)
	}
}

func TestPlanningTimelineAtomicFailureLeavesCardAndIndexUnchanged(t *testing.T) {
	t.Run("failure after staging and before commit", func(t *testing.T) {
		root := t.TempDir()
		card := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 14, 7, 0, 0, time.UTC))
		_, err := appendPlanningIterationCard(root, card, planningTimelineAppendOptions{
			ReceiptID: "route-pass-staged-fault",
			Fault: func(point string) error {
				if point == "after_intent" {
					return errPlanningTimelineInjected
				}
				return nil
			},
		})
		if !errors.Is(err, errPlanningTimelineInjected) {
			t.Fatalf("append error = %v, want injected fault", err)
		}
		assertPlanningTimelineTestArtifactsAbsent(t, root, card.RunID)
	})

	t.Run("second target replacement rolls the first target back", func(t *testing.T) {
		root := t.TempDir()
		card := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 14, 8, 0, 0, time.UTC))
		_, err := appendPlanningIterationCard(root, card, planningTimelineAppendOptions{
			ReceiptID: "route-pass-rename-fault",
			Rename: func(oldPath, newPath string) error {
				if strings.HasSuffix(filepath.ToSlash(newPath), "/timeline.json") {
					return errPlanningTimelineInjected
				}
				return os.Rename(oldPath, newPath)
			},
		})
		if !errors.Is(err, errPlanningTimelineInjected) {
			t.Fatalf("append error = %v, want injected replacement fault", err)
		}
		assertPlanningTimelineTestArtifactsAbsent(t, root, card.RunID)
	})
}

func TestPlanningTimelinePathAndSequenceRejectionsAreZeroMutation(t *testing.T) {
	t.Run("run path escape", func(t *testing.T) {
		root := t.TempDir()
		card := validPlanningIterationCardForTest(t, 1, time.Now().UTC())
		card.RunID = "../escape"
		if _, err := appendPlanningIterationCard(root, card, planningTimelineAppendOptions{ReceiptID: "escape"}); err == nil || !strings.Contains(err.Error(), "run_id") {
			t.Fatalf("path escape error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "escape")); !os.IsNotExist(err) {
			t.Fatalf("path escape created content outside planning root: %v", err)
		}
	})

	t.Run("missing dimension", func(t *testing.T) {
		root := t.TempDir()
		card := validPlanningIterationCardForTest(t, 1, time.Now().UTC())
		card.DimensionAssessments = card.DimensionAssessments[:4]
		if _, err := appendPlanningIterationCard(root, card, planningTimelineAppendOptions{ReceiptID: "missing-dimension"}); err == nil || !strings.Contains(err.Error(), "five") {
			t.Fatalf("missing dimension error = %v", err)
		}
		assertPlanningTimelineTestArtifactsAbsent(t, root, card.RunID)
	})

	t.Run("skipped initial ordinal", func(t *testing.T) {
		root := t.TempDir()
		card := validPlanningIterationCardForTest(t, 2, time.Now().UTC())
		if _, err := appendPlanningIterationCard(root, card, planningTimelineAppendOptions{ReceiptID: "skipped-initial"}); err == nil || !strings.Contains(err.Error(), "iteration") {
			t.Fatalf("skipped ordinal error = %v", err)
		}
		assertPlanningTimelineTestArtifactsAbsent(t, root, card.RunID)
	})

	t.Run("wrong predecessor and skipped next ordinal", func(t *testing.T) {
		root := t.TempDir()
		first := validPlanningIterationCardForTest(t, 1, time.Now().UTC())
		firstReceipt, err := appendPlanningIterationCard(root, first, planningTimelineAppendOptions{ReceiptID: "first"})
		if err != nil {
			t.Fatalf("append first: %v", err)
		}
		indexPath := filepath.Join(root, filepath.FromSlash(firstReceipt.IndexPath))
		before, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatal(err)
		}

		second := validPlanningIterationCardForTest(t, 2, time.Now().UTC().Add(time.Minute))
		if _, err := appendPlanningIterationCard(root, second, planningTimelineAppendOptions{ReceiptID: "wrong-prior", PreviousCardHash: strings.Repeat("0", 64)}); err == nil || !strings.Contains(err.Error(), "previous_card_hash") {
			t.Fatalf("wrong predecessor error = %v", err)
		}
		third := validPlanningIterationCardForTest(t, 3, time.Now().UTC().Add(2*time.Minute))
		if _, err := appendPlanningIterationCard(root, third, planningTimelineAppendOptions{ReceiptID: "skipped-next", PreviousCardHash: firstReceipt.CardHash}); err == nil || !strings.Contains(err.Error(), "iteration") {
			t.Fatalf("skipped next ordinal error = %v", err)
		}
		after, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Fatal("rejected append changed the timeline index")
		}
		entries, err := filepath.Glob(filepath.Join(root, ".aether", "data", "planning", first.RunID, "iterations", "*.json"))
		if err != nil || len(entries) != 1 {
			t.Fatalf("card files after rejected appends = %v, err=%v", entries, err)
		}
	})
}

func TestPlanningTimelineReplayExactIsNoOpAndDivergenceConflicts(t *testing.T) {
	root := t.TempDir()
	card := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 14, 9, 0, 0, time.UTC))
	opts := planningTimelineAppendOptions{ReceiptID: "route-pass-replay"}
	first, err := appendPlanningIterationCard(root, card, opts)
	if err != nil {
		t.Fatalf("append first: %v", err)
	}
	cardPath := filepath.Join(root, filepath.FromSlash(first.CardPath))
	indexPath := filepath.Join(root, filepath.FromSlash(first.IndexPath))
	cardBefore, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatal(err)
	}
	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}

	replayed, err := appendPlanningIterationCard(root, card, opts)
	if err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	if replayed.CardID != first.CardID || replayed.CardHash != first.CardHash || replayed.TimelineDigest != first.TimelineDigest {
		t.Fatalf("exact replay returned a different binding\nfirst: %#v\nreplay: %#v", first, replayed)
	}
	if !reflect.DeepEqual(replayed.WriteReceipt, first.WriteReceipt) {
		t.Fatal("exact replay returned a different lifecycle write receipt")
	}
	cardAfter, _ := os.ReadFile(cardPath)
	indexAfter, _ := os.ReadFile(indexPath)
	if !bytes.Equal(cardBefore, cardAfter) || !bytes.Equal(indexBefore, indexAfter) {
		t.Fatal("exact replay rewrote card or index bytes")
	}

	divergent := clonePlanningTimelineTestCard(t, card)
	divergent.DimensionAssessments[0].Rationale = "different evidence interpretation"
	if err := colony.AddressPlanningDimensionAssessment(&divergent.DimensionAssessments[0]); err != nil {
		t.Fatalf("re-address divergent assessment: %v", err)
	}
	_, err = appendPlanningIterationCard(root, divergent, opts)
	var conflict *planningTimelineConflictError
	if !errors.As(err, &conflict) || !strings.Contains(err.Error(), "receipt") {
		t.Fatalf("divergent replay error = %v, want typed receipt conflict", err)
	}
	cardAfterConflict, _ := os.ReadFile(cardPath)
	indexAfterConflict, _ := os.ReadFile(indexPath)
	if !bytes.Equal(cardBefore, cardAfterConflict) || !bytes.Equal(indexBefore, indexAfterConflict) {
		t.Fatal("divergent replay changed the original chain")
	}
}

func TestPlanningTimelineReplayEarlierReceiptAfterLaterAppendIsNoOp(t *testing.T) {
	root := t.TempDir()
	first, second := appendTwoPlanningTimelineTestCards(t, root)
	firstCard := readPlanningTimelineTestCard(t, root, first.CardPath)
	indexPath := filepath.Join(root, filepath.FromSlash(second.IndexPath))
	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}

	replayed, err := appendPlanningIterationCard(root, firstCard, planningTimelineAppendOptions{ReceiptID: first.ReceiptID})
	if err != nil {
		t.Fatalf("replay earlier append: %v", err)
	}
	if replayed.CardID != first.CardID || replayed.CardHash != first.CardHash || replayed.TimelineDigest != first.TimelineDigest {
		t.Fatalf("earlier replay did not return its original binding\nwant: %#v\n got: %#v", first, replayed)
	}
	if !reflect.DeepEqual(replayed.WriteReceipt, first.WriteReceipt) {
		t.Fatal("earlier replay did not return its original lifecycle receipt")
	}
	indexAfter, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Fatal("earlier replay rewrote the advanced timeline index")
	}
}

func TestPlanningTimelineReplayResumesMatchingStagedIntent(t *testing.T) {
	root := t.TempDir()
	card := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 14, 10, 0, 0, time.UTC))
	opts := planningTimelineAppendOptions{
		ReceiptID: "route-pass-resume",
		Fault: func(point string) error {
			if point == "after_intent" {
				return errPlanningTimelineInjected
			}
			return nil
		},
	}
	if _, err := appendPlanningIterationCard(root, card, opts); !errors.Is(err, errPlanningTimelineInjected) {
		t.Fatalf("staged append error = %v, want injected fault", err)
	}
	assertPlanningTimelineTestArtifactsAbsent(t, root, card.RunID)

	opts.Fault = nil
	receipt, err := appendPlanningIterationCard(root, card, opts)
	if err != nil {
		t.Fatalf("resume exact staged append: %v", err)
	}
	if receipt.WriteReceipt.StateEffect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("resumed write receipt = %#v", receipt.WriteReceipt)
	}
	loaded, err := loadPlanningTimeline(root, card.RunID)
	if err != nil {
		t.Fatalf("load resumed timeline: %v", err)
	}
	if len(loaded.Cards) != 1 || loaded.Cards[0].ID != receipt.CardID {
		t.Fatalf("resumed timeline = %#v", loaded)
	}
}

func TestPlanningTimelineDigestLoadIsDeterministicAndDetectsTamper(t *testing.T) {
	root := t.TempDir()
	first, second := appendTwoPlanningTimelineTestCards(t, root)

	loaded, err := loadPlanningTimeline(root, first.RunID)
	if err != nil {
		t.Fatalf("load timeline: %v", err)
	}
	again, err := loadPlanningTimeline(root, first.RunID)
	if err != nil {
		t.Fatalf("load timeline again: %v", err)
	}
	if loaded.Binding == nil || again.Binding == nil || !reflect.DeepEqual(loaded.Binding, again.Binding) {
		t.Fatalf("timeline binding was not deterministic\nfirst: %#v\nagain: %#v", loaded.Binding, again.Binding)
	}
	if loaded.Binding.TimelineDigest != second.TimelineDigest || loaded.Binding.LastCardHash != second.CardHash {
		t.Fatalf("loaded binding = %#v, final append = %#v", loaded.Binding, second)
	}

	tampered := readPlanningTimelineTestCard(t, root, first.CardPath)
	tampered.EvidenceThatWouldChange = "tampered evidence guidance"
	tamperedBytes, err := marshalPlanningTimelineJSON(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(first.CardPath)), tamperedBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPlanningTimeline(root, first.RunID); err == nil || !strings.Contains(err.Error(), "mutated") {
		t.Fatalf("tampered timeline load error = %v", err)
	}
}

func TestPlanningTimelineDigestRejectsReorderedIndexEvenWhenReaddressed(t *testing.T) {
	root := t.TempDir()
	first, second := appendTwoPlanningTimelineTestCards(t, root)
	index := readPlanningTimelineTestIndex(t, root, second.IndexPath)
	index.Entries[0], index.Entries[1] = index.Entries[1], index.Entries[0]
	index.ID = ""
	index.ContentHash = ""
	hash, err := planningTimelineIndexContentHash(index)
	if err != nil {
		t.Fatal(err)
	}
	index.ContentHash = hash
	index.ID = "planning-timeline-index-" + hash[:12]
	content, err := marshalPlanningTimelineJSON(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(second.IndexPath)), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPlanningTimeline(root, first.RunID); err == nil || !strings.Contains(err.Error(), "iteration") {
		t.Fatalf("reordered index load error = %v", err)
	}
}

func TestPlanningTimelineBoundAcceptedHistoryCannotBePrunedOrReordered(t *testing.T) {
	root := t.TempDir()
	first, _ := appendTwoPlanningTimelineTestCards(t, root)
	loaded, err := loadPlanningTimeline(root, first.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Binding == nil {
		t.Fatal("validated current timeline did not expose a binding")
	}
	revisions := []colony.PlanRevision{{
		ID:                     "plan-r1-accepted",
		PlanningTimelineID:     loaded.Binding.ID,
		PlanningTimelineDigest: loaded.Binding.TimelineDigest,
	}}
	protection, err := planningTimelineProtectionFor(loaded, revisions)
	if err != nil {
		t.Fatalf("classify accepted timeline: %v", err)
	}
	if protection.Classification != planningTimelineAccepted || protection.Prunable || protection.Reorderable || protection.BoundRevisionID != revisions[0].ID {
		t.Fatalf("accepted protection = %#v", protection)
	}
	for _, mutation := range []planningTimelineMutation{planningTimelineMutationPrune, planningTimelineMutationReorder} {
		err := authorizePlanningTimelineMutation(loaded, revisions, mutation)
		var protected *planningTimelineProtectedError
		if !errors.As(err, &protected) {
			t.Fatalf("accepted %s error = %v, want typed protection error", mutation, err)
		}
	}

	candidate, err := planningTimelineProtectionFor(loaded, nil)
	if err != nil {
		t.Fatalf("classify candidate-only timeline: %v", err)
	}
	if candidate.Classification != planningTimelineCandidateOnly || !candidate.Prunable || candidate.Reorderable {
		t.Fatalf("candidate-only protection = %#v", candidate)
	}
	if err := authorizePlanningTimelineMutation(loaded, nil, planningTimelineMutationPrune); err != nil {
		t.Fatalf("candidate-only retention prune was blocked: %v", err)
	}
}

func TestPlanningTimelineBoundRejectsDigestConflictAndClassifiesLegacyEvidence(t *testing.T) {
	root := t.TempDir()
	first, _ := appendTwoPlanningTimelineTestCards(t, root)
	loaded, err := loadPlanningTimeline(root, first.RunID)
	if err != nil {
		t.Fatal(err)
	}
	conflicting := []colony.PlanRevision{{
		ID:                     "plan-r1-conflict",
		PlanningTimelineID:     loaded.Binding.ID,
		PlanningTimelineDigest: strings.Repeat("0", 64),
	}}
	if _, err := planningTimelineProtectionFor(loaded, conflicting); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("accepted binding digest conflict = %v", err)
	}

	legacyRun := "legacy-run"
	legacyDir := filepath.Join(root, ".aether", "data", "planning", legacyRun, "iterations")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "iteration-01-scout.json"), []byte("{\"legacy\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy, err := loadPlanningTimeline(root, legacyRun)
	if err != nil {
		t.Fatalf("load legacy evidence: %v", err)
	}
	if legacy.Classification != planningTimelineLegacyUnbound || legacy.Binding != nil || len(legacy.LegacyArtifacts) != 1 {
		t.Fatalf("legacy timeline classification = %#v", legacy)
	}
	legacyProtection, err := planningTimelineProtectionFor(legacy, conflicting)
	if err != nil {
		t.Fatalf("classify legacy evidence: %v", err)
	}
	if legacyProtection.Classification != planningTimelineLegacyUnbound || legacyProtection.BoundRevisionID != "" {
		t.Fatalf("legacy protection = %#v", legacyProtection)
	}
}

func appendTwoPlanningTimelineTestCards(t *testing.T, root string) (planningTimelineAppendReceipt, planningTimelineAppendReceipt) {
	t.Helper()
	firstCard := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 14, 11, 0, 0, time.UTC))
	first, err := appendPlanningIterationCard(root, firstCard, planningTimelineAppendOptions{ReceiptID: "route-pass-one"})
	if err != nil {
		t.Fatalf("append first card: %v", err)
	}
	secondCard := validPlanningIterationCardForTest(t, 2, time.Date(2026, time.September, 7, 14, 12, 0, 0, time.UTC))
	second, err := appendPlanningIterationCard(root, secondCard, planningTimelineAppendOptions{
		ReceiptID:        "route-pass-two",
		PreviousCardHash: first.CardHash,
	})
	if err != nil {
		t.Fatalf("append second card: %v", err)
	}
	return first, second
}

func readPlanningTimelineTestCard(t *testing.T, root, relativePath string) colony.PlanningIterationCard {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("read card %s: %v", relativePath, err)
	}
	var card colony.PlanningIterationCard
	if err := json.Unmarshal(content, &card); err != nil {
		t.Fatalf("decode card %s: %v", relativePath, err)
	}
	return card
}

func readPlanningTimelineTestIndex(t *testing.T, root, relativePath string) planningTimelineIndex {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("read index %s: %v", relativePath, err)
	}
	var index planningTimelineIndex
	if err := json.Unmarshal(content, &index); err != nil {
		t.Fatalf("decode index %s: %v", relativePath, err)
	}
	return index
}

func assertPlanningTimelineTestArtifactsAbsent(t *testing.T, root, runID string) {
	t.Helper()
	base := filepath.Join(root, ".aether", "data", "planning", runID)
	if _, err := os.Stat(filepath.Join(base, "timeline.json")); !os.IsNotExist(err) {
		t.Fatalf("timeline index exists after rejected append: %v", err)
	}
	cards, err := filepath.Glob(filepath.Join(base, "iterations", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("card artifacts exist after rejected append: %v", cards)
	}
}

func clonePlanningTimelineTestCard(t *testing.T, card colony.PlanningIterationCard) colony.PlanningIterationCard {
	t.Helper()
	content, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	var clone colony.PlanningIterationCard
	if err := json.Unmarshal(content, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}
