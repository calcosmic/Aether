package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// newCreditTrophallaxisFixture drives the real 203-10 public path (pack,
// acknowledge, record decision) to produce a genuine trophallaxisPacket with
// a recorded colony.LifecycleDecision, so credit tests attach to real
// recorded facts rather than a hand-built struct.
func newCreditTrophallaxisFixture(t *testing.T, suffix string) (packetID string, decisionID string) {
	t.Helper()
	seedTrophallaxisReceiver(t, "Mason-"+suffix)
	result := newTrophallaxisResultFixture("credit-" + suffix)
	packet, err := packTrophallaxisPacket(trophallaxisPackInput{
		Result:         result,
		Handoff:        newTrophallaxisHandoffFixture(),
		ParentReceiver: "Mason-" + suffix,
		Scope:          []string{trophallaxisSectionSummary},
	})
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if _, err := acknowledgeTrophallaxisPacket(packet.PacketID, colony.SignalAcknowledgement{ActorID: "Mason-" + suffix, EvidenceID: "evidence-" + suffix}, ""); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	decided, err := recordTrophallaxisDecision(packet.PacketID, "Mason-"+suffix, "did the thing for "+suffix, nil)
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	if decided.Decision == nil {
		t.Fatal("fixture is broken: no decision recorded")
	}
	return packet.PacketID, decided.Decision.ID
}

// --- Task 1: a credit record that can say helped, did nothing, or made it worse ---

func TestRecruitmentCreditOutcomeVocabulary(t *testing.T) {
	names := recruitmentCreditOutcomeNames()
	if len(names) != 4 {
		t.Fatalf("recruitmentCreditOutcomeNames() = %v, want exactly 4 members", names)
	}
	for _, want := range []string{"helpful", "neutral", "harmful", "pending"} {
		found := false
		for _, got := range names {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("recruitmentCreditOutcomeNames() = %v, missing declared member %q", names, want)
		}
	}

	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "vocab")

	_, credited, err := recordRecruitmentCredit("contrib-vocab", recruitmentContributionNote, decisionID, "evidence-vocab", recruitmentCreditOutcome("bogus"), "")
	if err == nil {
		t.Fatal("expected an undeclared outcome to be refused")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("refusal error = %q, want it to name the offending value %q", err.Error(), "bogus")
	}
	if credited {
		t.Fatal("an undeclared outcome must never report credited=true")
	}
}

func TestRecruitmentCreditRecordsAHarmfulOutcomeDirectlyOnTheFile(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "harmful")

	record, credited, err := recordRecruitmentCredit("contrib-harmful", recruitmentContributionRecruitmentResult, decisionID, "evidence-harmful", recruitmentCreditOutcomeHarmful, "")
	if err != nil {
		t.Fatalf("record harmful credit: %v", err)
	}
	if !credited {
		t.Fatal("a fully-facted harmful contribution must be credited (recorded), not omitted")
	}
	if record.Outcome != recruitmentCreditOutcomeHarmful {
		t.Fatalf("stored outcome = %q, want harmful", record.Outcome)
	}

	raw, err := os.ReadFile(filepath.Join(store.BasePath(), recruitmentCreditPath))
	if err != nil {
		t.Fatalf("read credit store file: %v", err)
	}
	if !strings.Contains(string(raw), "harmful") {
		t.Fatalf("credit/records.json does not contain the harmful outcome on disk:\n%s", raw)
	}
}

func TestRecruitmentCreditReplayIsByteIdentical(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "replay")

	if _, _, err := recordRecruitmentCredit("contrib-replay", recruitmentContributionNote, decisionID, "evidence-replay", recruitmentCreditOutcomeHelpful, "2026-09-13T00:00:00Z"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	creditFilePath := filepath.Join(store.BasePath(), recruitmentCreditPath)
	before, err := os.ReadFile(creditFilePath)
	if err != nil {
		t.Fatalf("read before replay: %v", err)
	}

	record, credited, err := recordRecruitmentCredit("contrib-replay", recruitmentContributionNote, decisionID, "evidence-replay", recruitmentCreditOutcomeHelpful, "2026-09-13T00:00:00Z")
	if err != nil {
		t.Fatalf("replay write: %v", err)
	}
	if !credited {
		t.Fatal("a verified replay must still report credited=true (it returns the existing record)")
	}
	if record.Outcome != recruitmentCreditOutcomeHelpful {
		t.Fatalf("replayed record outcome = %q, want helpful", record.Outcome)
	}

	after, err := os.ReadFile(creditFilePath)
	if err != nil {
		t.Fatalf("read after replay: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("replay mutated credit/records.json:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestRecruitmentCreditTwoContributionsOneDecision(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "shared")

	if _, _, err := recordRecruitmentCredit("contrib-a", recruitmentContributionNote, decisionID, "evidence-a", recruitmentCreditOutcomeHelpful, ""); err != nil {
		t.Fatalf("record contrib-a: %v", err)
	}
	if _, _, err := recordRecruitmentCredit("contrib-b", recruitmentContributionMemoryItem, decisionID, "evidence-b", recruitmentCreditOutcomeNeutral, ""); err != nil {
		t.Fatalf("record contrib-b: %v", err)
	}

	records, err := recruitmentCreditForDecision(decisionID)
	if err != nil {
		t.Fatalf("query by decision: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("credit list for decision %q has %d entries, want 2: %+v", decisionID, len(records), records)
	}
	seen := map[string]bool{}
	for _, r := range records {
		seen[r.ContributionID] = true
	}
	if !seen["contrib-a"] || !seen["contrib-b"] {
		t.Fatalf("expected both contrib-a and contrib-b in the decision's credit list, got %+v", records)
	}
}

func TestRecruitmentCreditSortOrderIsStableAcrossRepeatedReads(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "tie")

	sharedTimestamp := "2026-09-13T12:00:00Z"
	for _, c := range []string{"tie-c", "tie-a", "tie-b"} {
		if _, _, err := recordRecruitmentCredit(c, recruitmentContributionSpecialist, decisionID, "evidence-"+c, recruitmentCreditOutcomeNeutral, sharedTimestamp); err != nil {
			t.Fatalf("seed %s: %v", c, err)
		}
	}

	var previous []byte
	for i := 0; i < 10; i++ {
		records, err := recruitmentCreditForDecision(decisionID)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		encoded, err := json.Marshal(records)
		if err != nil {
			t.Fatalf("marshal read %d: %v", i, err)
		}
		if previous != nil && string(previous) != string(encoded) {
			t.Fatalf("read %d differs from prior read:\nprior: %s\nnow:   %s", i, previous, encoded)
		}
		previous = encoded
	}

	records, _ := recruitmentCreditForDecision(decisionID)
	var ids []string
	for _, r := range records {
		ids = append(ids, r.ContributionID)
	}
	want := []string{"tie-a", "tie-b", "tie-c"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("tie-break order = %v, want record-identifier ascending %v", ids, want)
	}
}

func TestRecruitmentCreditUncreditedRendersExistingNoMeasuredEffectWording(t *testing.T) {
	rendered := renderRecruitmentCreditOutcome(nil)
	if rendered != AgencyMeasuredEffectPending {
		t.Fatalf("uncredited rendering = %q, want the existing constant %q", rendered, AgencyMeasuredEffectPending)
	}

	pendingRecord := &recruitmentCreditRecord{Outcome: recruitmentCreditOutcomePending}
	pendingRendered := renderRecruitmentCreditOutcome(pendingRecord)
	if pendingRendered == AgencyMeasuredEffectPending {
		t.Fatalf("a decision-recorded-but-unverified contribution must render distinctly from the no-record-at-all wording, got %q for both", pendingRendered)
	}
	if pendingRendered != recruitmentCreditPendingWording {
		t.Fatalf("pending rendering = %q, want %q", pendingRendered, recruitmentCreditPendingWording)
	}

	helpfulRecord := &recruitmentCreditRecord{Outcome: recruitmentCreditOutcomeHelpful}
	if got := renderRecruitmentCreditOutcome(helpfulRecord); got != "helpful" {
		t.Fatalf("helpful rendering = %q, want %q", got, "helpful")
	}
}

func TestRecruitmentCreditNeitherFactRecordsNothing(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	record, credited, err := recordRecruitmentCredit("contrib-nothing", recruitmentContributionNote, "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error for the neither-fact case, got %v", err)
	}
	if credited {
		t.Fatalf("expected credited=false for the neither-fact case, got record %+v", record)
	}

	all, err := recruitmentCreditAll()
	if err != nil {
		t.Fatalf("recruitmentCreditAll: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected zero records written for the neither-fact case, got %+v", all)
	}
}

func TestRecruitmentCreditEffectEvidenceWithoutDecisionIsRefused(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	_, credited, err := recordRecruitmentCredit("contrib-orphan-evidence", recruitmentContributionNote, "", "evidence-orphan", recruitmentCreditOutcomeHelpful, "")
	if err == nil {
		t.Fatal("expected effect evidence with no changed decision to be refused")
	}
	if credited {
		t.Fatal("a refused write must never report credited=true")
	}
}

func TestRecruitmentCreditDecisionOnlyIsPendingNotHelpful(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "pending")

	record, credited, err := recordRecruitmentCredit("contrib-pending", recruitmentContributionNote, decisionID, "", "", "")
	if err != nil {
		t.Fatalf("record decision-only credit: %v", err)
	}
	if !credited {
		t.Fatal("a decision-only record is stored (as pending), not omitted")
	}
	if record.Outcome != recruitmentCreditOutcomePending {
		t.Fatalf("decision-only outcome = %q, want pending", record.Outcome)
	}
}
