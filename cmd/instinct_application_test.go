package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Shared fixtures (198.1-03)
// ---------------------------------------------------------------------------

// promoteRealInstinct creates an instinct through the real
// memory.PromoteService from a real observation, never by typing a
// colony.InstinctEntry literal into instincts.json by hand (D-04). content
// must satisfy memory.IsAdmissibleInstinctContent (20-240 chars, names a
// file, command, or error; no lifecycle narration).
func promoteRealInstinct(t *testing.T, s *storage.Store, content, wisdomType string) colony.InstinctEntry {
	t.Helper()
	bus := events.NewBus(s, events.DefaultConfig())
	now := time.Now().UTC()
	hash := sha256.Sum256([]byte(content + ":" + wisdomType))
	obs := colony.Observation{
		ContentHash:      "sha256:" + hex.EncodeToString(hash[:]),
		Content:          content,
		WisdomType:       wisdomType,
		ObservationCount: 1,
		FirstSeen:        events.FormatTimestamp(now),
		LastSeen:         events.FormatTimestamp(now),
		Colonies:         []string{"test-colony"},
		SourceType:       "success_pattern",
		EvidenceType:     "single_phase",
	}
	svc := memory.NewPromoteService(s, bus)
	result, err := svc.Promote(context.Background(), obs, "test-colony")
	if err != nil {
		t.Fatalf("promote real instinct for %q: %v", content, err)
	}
	return result.Instinct
}

// historyEntryKeys extracts the key set of a raw ApplicationHistory entry
// (a map[string]interface{} once round-tripped through JSON).
func historyEntryKeys(t *testing.T, raw interface{}) map[string]bool {
	t.Helper()
	m, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("history entry is not a map: %#v", raw)
	}
	keys := make(map[string]bool, len(m))
	for k := range m {
		keys[k] = true
	}
	return keys
}

func mapKeysEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Task 1 (198.1-03): record which instincts a worker was actually given.
// ---------------------------------------------------------------------------

func TestInstinctDeliveryIsRecordedOnlyWhenTheTextWasActuallyDelivered(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	instA := promoteRealInstinct(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")
	instB := promoteRealInstinct(t, s, "check pkg/colony/context_ranking.go before assuming score-based trim order", "pattern")

	capsule, _ := resolveCodexWorkerContextWithTrim()
	if capsule == "" {
		t.Fatal("expected a non-empty capsule from a colony holding two active instincts")
	}

	// Derive the expected delivered set from the capsule string itself
	// (content-based, not heading-based -- 198.1-CONTEXT.md decision).
	expected := map[string]bool{}
	for _, inst := range []colony.InstinctEntry{instA, instB} {
		if strings.Contains(capsule, strings.TrimSpace(inst.Action)) {
			expected[inst.ID] = true
		}
	}
	if len(expected) == 0 {
		t.Fatalf("test fixture invalid: capsule contains neither instinct's action text: %q", capsule)
	}

	recorded := recordInstinctDeliveries(198, "build", capsule)
	if recorded != len(expected) {
		t.Fatalf("recordInstinctDeliveries returned %d, want %d (expected=%v)", recorded, len(expected), expected)
	}

	var df instinctDeliveryFile
	if err := s.LoadJSON(instinctDeliveriesPath, &df); err != nil {
		t.Fatalf("load %s: %v", instinctDeliveriesPath, err)
	}
	got := map[string]bool{}
	for _, d := range df.Deliveries {
		if d.Phase != 198 {
			t.Fatalf("unexpected phase %d recorded in a delivery entry", d.Phase)
		}
		if d.Workflow != "build" {
			t.Fatalf("unexpected workflow %q recorded in a delivery entry", d.Workflow)
		}
		got[d.InstinctID] = true
	}
	if len(got) != len(expected) {
		t.Fatalf("recorded deliveries = %v, want %v", got, expected)
	}
	for id := range expected {
		if !got[id] {
			t.Errorf("expected a delivery for %s, none recorded", id)
		}
	}
	for id := range got {
		if !expected[id] {
			t.Errorf("unexpected delivery recorded for %s, whose action text is not in the capsule", id)
		}
	}
}

func TestInstinctDeliveryIsRecordedOncePerPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	inst := promoteRealInstinct(t, s, "run go build ./cmd/aether before go vet ./cmd/ to catch compile errors early", "pattern")
	capsule := inst.Action

	first := recordInstinctDeliveries(11, "continue", capsule)
	if first != 1 {
		t.Fatalf("first recordInstinctDeliveries call returned %d, want 1", first)
	}
	second := recordInstinctDeliveries(11, "continue", capsule)
	if second != 0 {
		t.Fatalf("second recordInstinctDeliveries call for the same phase returned %d, want 0", second)
	}

	var df instinctDeliveryFile
	if err := s.LoadJSON(instinctDeliveriesPath, &df); err != nil {
		t.Fatalf("load %s: %v", instinctDeliveriesPath, err)
	}
	if len(df.Deliveries) != 1 {
		t.Fatalf("expected exactly 1 delivery record after two identical calls, got %d: %+v", len(df.Deliveries), df.Deliveries)
	}
}

func TestTrimmedInstinctsSectionRecordsNoDelivery(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	inst := promoteRealInstinct(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")

	// Force the compact (4000-char) worker budget to drop the instincts
	// section: a single protected blocker (unconditionally included, no
	// budget check of its own) large enough to consume the whole budget by
	// itself outranks the instincts section (relevance 0.80, not protected),
	// which gets trimmed entirely rather than partially included.
	hugeDescription := strings.Repeat("blocking condition that must be resolved before this phase can advance. ", 80)
	if err := s.SaveJSON("pending-decisions.json", colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{
			{
				ID:          "flag_huge_1",
				Type:        "blocker",
				Description: hugeDescription,
				Source:      "test",
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
				Resolved:    false,
			},
		},
	}); err != nil {
		t.Fatalf("seed pending-decisions.json: %v", err)
	}

	capsule, trimmed := resolveCodexWorkerContextWithTrim()
	if capsule == "" {
		t.Fatal("expected a non-empty capsule (the blocker alone satisfies the minimum-length floor)")
	}
	foundTrimmed := false
	for _, name := range trimmed {
		if name == "instincts" {
			foundTrimmed = true
		}
	}
	if !foundTrimmed {
		t.Fatalf("test fixture invalid: instincts section was not trimmed (trimmed=%v)", trimmed)
	}
	if strings.Contains(capsule, strings.TrimSpace(inst.Action)) {
		t.Fatal("test fixture invalid: instinct action text is still present in the capsule despite the trim")
	}

	recorded := recordInstinctDeliveries(55, "build", capsule)
	if recorded != 0 {
		t.Fatalf("recordInstinctDeliveries recorded %d against a capsule that trimmed the instincts section, want 0", recorded)
	}

	var df instinctDeliveryFile
	if err := s.LoadJSON(instinctDeliveriesPath, &df); err == nil && len(df.Deliveries) != 0 {
		t.Fatalf("expected no deliveries recorded, got %+v", df.Deliveries)
	}
}

// ---------------------------------------------------------------------------
// Task 2 (198.1-03): a phase that passed records one use of each instinct it
// was given.
// ---------------------------------------------------------------------------

func TestDeliveredInstinctGainsOneApplicationPerPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	inst := promoteRealInstinct(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")
	if recorded := recordInstinctDeliveries(7, "continue", inst.Action); recorded != 1 {
		t.Fatalf("seed delivery: recordInstinctDeliveries returned %d, want 1", recorded)
	}

	first := recordInstinctApplicationsForPhase(7)
	if first != 1 {
		t.Fatalf("first recordInstinctApplicationsForPhase(7) = %d, want 1", first)
	}
	second := recordInstinctApplicationsForPhase(7)
	if second != 0 {
		t.Fatalf("second recordInstinctApplicationsForPhase(7) = %d, want 0", second)
	}

	file := loadInstinctFileOrEmpty(s)
	var updated *colony.InstinctEntry
	for i := range file.Instincts {
		if file.Instincts[i].ID == inst.ID {
			updated = &file.Instincts[i]
		}
	}
	if updated == nil {
		t.Fatalf("instinct %s missing after recording applications", inst.ID)
	}
	summary := memory.SummarizeInstinctApplications(*updated)
	if summary.Applications != 1 {
		t.Fatalf("Applications = %d, want 1 after two recorder runs for the same phase", summary.Applications)
	}
}

func TestUndeliveredInstinctGainsNothing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	inst := promoteRealInstinct(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")
	// No delivery is ever recorded for phase 9.

	recorded := recordInstinctApplicationsForPhase(9)
	if recorded != 0 {
		t.Fatalf("recordInstinctApplicationsForPhase(9) = %d, want 0 for an undelivered instinct", recorded)
	}

	file := loadInstinctFileOrEmpty(s)
	for _, e := range file.Instincts {
		if e.ID == inst.ID && len(e.ApplicationHistory) != 0 {
			t.Fatalf("expected no application history for an undelivered instinct, got %+v", e.ApplicationHistory)
		}
	}
}

func TestApplicationHistoryShapeMatchesInstinctApply(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	instApply := promoteRealInstinct(t, s, "run go build ./cmd/aether before go vet ./cmd/ to catch compile errors early", "pattern")
	instRecorder := promoteRealInstinct(t, s, "check pkg/colony/context_ranking.go before assuming score-based trim order", "pattern")

	rootCmd.SetArgs([]string{"instinct-apply", instApply.ID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("instinct-apply failed: %v", err)
	}

	if recorded := recordInstinctDeliveries(21, "continue", instRecorder.Action); recorded != 1 {
		t.Fatalf("seed delivery: recordInstinctDeliveries returned %d, want 1", recorded)
	}
	if n := recordInstinctApplicationsForPhase(21); n != 1 {
		t.Fatalf("recordInstinctApplicationsForPhase(21) = %d, want 1", n)
	}

	file := loadInstinctFileOrEmpty(s)
	var applyEntry, recorderEntry *colony.InstinctEntry
	for i := range file.Instincts {
		switch file.Instincts[i].ID {
		case instApply.ID:
			applyEntry = &file.Instincts[i]
		case instRecorder.ID:
			recorderEntry = &file.Instincts[i]
		}
	}
	if applyEntry == nil || recorderEntry == nil {
		t.Fatalf("both instincts must be present after both writers ran: apply=%v recorder=%v", applyEntry, recorderEntry)
	}
	if len(applyEntry.ApplicationHistory) != 1 || len(recorderEntry.ApplicationHistory) != 1 {
		t.Fatalf("expected exactly one history entry each: apply=%d recorder=%d", len(applyEntry.ApplicationHistory), len(recorderEntry.ApplicationHistory))
	}

	applyKeys := historyEntryKeys(t, applyEntry.ApplicationHistory[0])
	recorderKeys := historyEntryKeys(t, recorderEntry.ApplicationHistory[0])
	// The recorder's entry carries exactly one extra key: "phase".
	delete(recorderKeys, "phase")
	if !mapKeysEqual(applyKeys, recorderKeys) {
		t.Fatalf("history key sets differ apart from 'phase': instinct-apply=%v recordInstinctApplicationsForPhase(minus phase)=%v", applyKeys, recorderKeys)
	}

	applySummary := memory.SummarizeInstinctApplications(*applyEntry)
	recorderSummary := memory.SummarizeInstinctApplications(*recorderEntry)
	if applySummary.Applications != 1 {
		t.Fatalf("instinct-apply Applications = %d, want 1", applySummary.Applications)
	}
	if recorderSummary.Applications != 1 {
		t.Fatalf("recordInstinctApplicationsForPhase Applications = %d, want 1", recorderSummary.Applications)
	}
}

func TestPhaseEndConsolidationReportsWhatReachedTheQueenFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}

	inst := promoteRealInstinct(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")

	var summary phaseEndConsolidationSummary
	for phase := 1; phase <= 6; phase++ {
		if recorded := recordInstinctDeliveries(phase, "continue", inst.Action); recorded == 0 {
			t.Fatalf("phase %d: expected a new delivery to be recorded", phase)
		}
		summary = runPhaseEndConsolidation(phase)
		if !summary.Ran {
			t.Fatalf("phase %d: consolidation did not run: %s", phase, summary.Reason)
		}
		if len(summary.QueenPromoted) > 0 {
			break
		}
	}

	if len(summary.QueenPromoted) == 0 {
		t.Fatalf("instinct was never promoted to QUEEN.md across 6 rounds; last summary=%+v", summary)
	}
	if summary.QueenPromoted[0] != inst.ID {
		t.Fatalf("QueenPromoted = %v, want [%s]", summary.QueenPromoted, inst.ID)
	}

	queenText, err := os.ReadFile(consolidationQueenPath())
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	if !strings.Contains(string(queenText), strings.TrimSpace(inst.Action)) {
		t.Fatalf("QUEEN.md does not contain the promoted instinct's action text")
	}
}

// ---------------------------------------------------------------------------
// Task 3 (198.1-03): end to end -- a worker's lesson becomes Queen-file
// wisdom, and cannot without a recorded use.
// ---------------------------------------------------------------------------

const workerLessonSentence = "always run go vet ./cmd/ before go test ./cmd/ here; a vet failure makes the test output unreadable"

func TestWorkerLessonBecomesQueenFileWisdom(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Stage 1: the worker's own sentence, captured through the real capture
	// path (Plan 01's captureWorkerObservation), becomes an observation.
	ok, reason := captureWorkerObservation(workerLessonSentence, "pattern", observationSourceTypeForOutcome(true), "single_phase")
	if !ok {
		t.Fatalf("captureWorkerObservation rejected the fixture sentence: %s", reason)
	}
	var obsFile colony.LearningFile
	if err := s.LoadJSON("learning-observations.json", &obsFile); err != nil {
		t.Fatalf("load learning-observations.json: %v", err)
	}
	if len(obsFile.Observations) != 1 {
		t.Fatalf("expected exactly 1 observation, got %d", len(obsFile.Observations))
	}
	if !strings.Contains(obsFile.Observations[0].Content, "go vet ./cmd/") {
		t.Fatalf("captured observation lost the worker's sentence: %q", obsFile.Observations[0].Content)
	}
	contentHash := obsFile.Observations[0].ContentHash

	// Stage 2: one phase-end consolidation promotes the observation into an
	// instinct -- proving the instinct came from the pipeline, not the
	// fixture (no instinct or observation is ever seeded by hand in this
	// test).
	runPhaseEndConsolidation(1)

	file := loadInstinctFileOrEmpty(s)
	var created *colony.InstinctEntry
	for i := range file.Instincts {
		if file.Instincts[i].Provenance.Source == contentHash {
			created = &file.Instincts[i]
		}
	}
	if created == nil {
		t.Fatalf("expected an instinct promoted from observation %s, instincts=%+v", contentHash, file.Instincts)
	}

	// Stage 3: deliver + consolidate for up to 6 more phases until the
	// instinct is promoted to QUEEN.md. The round count is discovered by
	// running the real pipeline, never asserted as a constant.
	var summary phaseEndConsolidationSummary
	promotedRound := 0
	for round := 2; round <= 7; round++ {
		capsule, _ := resolveCodexWorkerContextWithTrim()
		if capsule == "" {
			t.Fatalf("round %d: empty capsule", round)
		}
		if recordInstinctDeliveries(round, "continue", capsule) == 0 {
			t.Fatalf("round %d: expected a new delivery to be recorded", round)
		}
		summary = runPhaseEndConsolidation(round)
		if len(summary.QueenPromoted) > 0 {
			promotedRound = round
			break
		}
	}

	if promotedRound == 0 {
		t.Fatalf("instinct never promoted to QUEEN.md within 6 rounds; last summary=%+v", summary)
	}
	t.Logf("promotion occurred on round %d (relative to the phase-1 instinct creation)", promotedRound)

	if len(summary.QueenPromoted) != 1 || summary.QueenPromoted[0] != created.ID {
		t.Fatalf("QueenPromoted = %v, want [%s]", summary.QueenPromoted, created.ID)
	}

	queenText, err := os.ReadFile(consolidationQueenPath())
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	if !strings.Contains(string(queenText), strings.TrimSpace(created.Action)) {
		t.Fatal("QUEEN.md does not contain the promoted instinct's action text")
	}
	if !strings.Contains(string(queenText), workerLessonSentence) {
		t.Fatal("QUEEN.md does not contain the worker's own sentence verbatim")
	}
}

func TestQueenPromotionNeverHappensWithoutRecordedUse(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ok, reason := captureWorkerObservation(workerLessonSentence, "pattern", observationSourceTypeForOutcome(true), "single_phase")
	if !ok {
		t.Fatalf("captureWorkerObservation rejected the fixture sentence: %s", reason)
	}

	// Phase 1 still creates the instinct from the observation (same as
	// Stage 2 above) -- promotion to an instinct does not require a
	// recorded delivery, only promotion to QUEEN.md does.
	runPhaseEndConsolidation(1)
	file := loadInstinctFileOrEmpty(s)
	if len(file.Instincts) != 1 {
		t.Fatalf("expected exactly 1 instinct after phase 1, got %d", len(file.Instincts))
	}
	created := file.Instincts[0]

	// Every subsequent round consolidates WITHOUT ever recording a
	// delivery -- this is the negative proof: no recorded use, no
	// promotion, however many rounds run.
	for round := 2; round <= 7; round++ {
		summary := runPhaseEndConsolidation(round)
		if len(summary.QueenPromoted) != 0 {
			t.Fatalf("round %d: QueenPromoted = %v, want empty -- no delivery was ever recorded", round, summary.QueenPromoted)
		}
	}

	queenText, err := os.ReadFile(consolidationQueenPath())
	if err == nil && strings.Contains(string(queenText), strings.TrimSpace(created.Action)) {
		t.Fatal("QUEEN.md contains the instinct's action text despite no recorded use ever occurring")
	}
}
