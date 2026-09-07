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
	card.WeakestGap = card.DimensionAssessments[2].RemainingGap
	card.Decision.SelectedGapID = card.WeakestGap.ID
	card.Decision.ResidualGapIDs = []string{card.WeakestGap.ID}
	card.Decision.Reason = colony.PlanningStopContinue
	card.SemanticDelta.AuthorityImpacts[0].AffectedSemanticIDs = []string{"phase-grounded", "task-grounded"}

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
