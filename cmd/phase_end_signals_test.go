package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Task 1 (198.1-04): a finished phase and an answered question each leave a
// note.
// ---------------------------------------------------------------------------

func TestFinishedPhaseLeavesANoteNamingWhatItProduced(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Feed the Memory"}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed COLONY_STATE.json: %v", err)
	}
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("consolidation did not run: %s", summary.Reason)
	}

	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones.json: %v", err)
	}
	var found *colony.PheromoneSignal
	for i := range pf.Signals {
		if pf.Signals[i].Active && pf.Signals[i].Type == "FEEDBACK" {
			found = &pf.Signals[i]
		}
	}
	if found == nil {
		t.Fatalf("expected an active FEEDBACK signal, signals=%+v", pf.Signals)
	}
	text := extractText(found.Content)

	if !strings.Contains(text, "Feed the Memory") {
		t.Errorf("note does not name the phase: %q", text)
	}
	// Every number asserted below is read off the returned summary, never
	// typed as a literal (D-03/D-04).
	if !strings.Contains(text, fmt.Sprintf("%d lesson", summary.PromotionCandidates)) {
		t.Errorf("note does not carry the lessons-captured count (%d) from the returned summary: %q", summary.PromotionCandidates, text)
	}
	if !strings.Contains(text, fmt.Sprintf("%d failure", summary.FailuresRecorded)) {
		t.Errorf("note does not carry the failures-recorded count (%d) from the returned summary: %q", summary.FailuresRecorded, text)
	}
	if !strings.Contains(text, fmt.Sprintf("%d instinct", len(summary.QueenPromoted))) {
		t.Errorf("note does not carry the instincts-promoted count from the returned summary: %q", text)
	}
}

func TestAnsweredQuestionLeavesANoteCarryingTheAnswer(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	question := "should the drill-down show deferred lessons separately?"
	answer := "yes, in their own block under the pending list"
	if _, err := recordDecisionAnswer(question, answer, 1, "worker-handoff"); err != nil {
		t.Fatalf("recordDecisionAnswer failed: %v", err)
	}

	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones.json: %v", err)
	}
	var text string
	found := false
	for _, sig := range pf.Signals {
		if !sig.Active || sig.Type != "FEEDBACK" {
			continue
		}
		candidate := extractText(sig.Content)
		if strings.Contains(candidate, answer) {
			found = true
			text = candidate
		}
	}
	if !found {
		t.Fatalf("expected an active FEEDBACK signal carrying the exact answer %q, signals=%+v", answer, pf.Signals)
	}
	if !strings.Contains(text, question) {
		t.Errorf("note does not carry the question it answers: %q", text)
	}
}

func TestRepeatedNoteReinforcesRatherThanDuplicates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	emitDecisionFeedback("should X happen?", "yes, do X", 1)
	emitDecisionFeedback("should X happen?", "yes, do X", 1)

	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones.json: %v", err)
	}
	active := 0
	var sig colony.PheromoneSignal
	for _, s2 := range pf.Signals {
		if s2.Active && s2.Type == "FEEDBACK" {
			active++
			sig = s2
		}
	}
	if active != 1 {
		t.Fatalf("expected exactly 1 active FEEDBACK signal after two identical emissions, got %d", active)
	}
	if sig.ReinforcementCount == nil || *sig.ReinforcementCount != 1 {
		t.Errorf("expected ReinforcementCount=1 after the second identical emission, got %v", sig.ReinforcementCount)
	}
}

func TestSignalWriteFailureNeverBlocksThePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state := colony.ColonyState{
		Version:      "3.0",
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Feed the Memory"}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed COLONY_STATE.json: %v", err)
	}
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}

	// Make pheromones.json unwritable by pre-creating it as a directory --
	// narrower than chmod'ing the whole data dir (which would also break the
	// consolidation pipeline's own instincts/observations writes and hide
	// what this test claims: that a SIGNAL write failure specifically never
	// blocks the phase). AtomicWrite's rename-into-place fails reliably and
	// portably against an existing directory, with no root/permission
	// caveats.
	if err := os.MkdirAll(filepath.Join(s.BasePath(), "pheromones.json"), 0755); err != nil {
		t.Fatalf("seed pheromones.json as a directory (unwritable-as-file fixture): %v", err)
	}

	summary := runPhaseEndConsolidation(1)
	if !summary.Ran {
		t.Fatalf("expected Ran: true when only the pheromone signal write fails, got Ran: false, reason=%s", summary.Reason)
	}
}

// ---------------------------------------------------------------------------
// Task 2 (198.1-04): a run of the same failure produces an automatic
// don't-do-this note.
// ---------------------------------------------------------------------------

func seedMiddenEntries(t *testing.T, s interface {
	SaveJSON(path string, data interface{}) error
}, entries []colony.MiddenEntry) {
	t.Helper()
	mf := colony.MiddenFile{Version: "1.0.0", Entries: entries}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}
}

func TestThreeFailuresOfOneKindProduceOneRedirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	newestMessage := "go test ./cmd/ -run TestExpireSignals failed: nil pointer dereference in expireSignalsByType"
	seedMiddenEntries(t, s, []colony.MiddenEntry{
		{ID: "m1", Timestamp: "2026-08-01T00:00:00Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "go test ./cmd/ -run TestA failed: assertion mismatch"},
		{ID: "m2", Timestamp: "2026-08-01T00:00:01Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "go test ./cmd/ -run TestB failed: timeout"},
		{ID: "m3", Timestamp: "2026-08-01T00:00:02Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: newestMessage},
	})

	crossed := emitMiddenThresholdRedirect()
	if crossed != 1 {
		t.Fatalf("emitMiddenThresholdRedirect returned %d, want 1", crossed)
	}

	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones.json: %v", err)
	}
	active := 0
	var text string
	for _, sig := range pf.Signals {
		if sig.Active && sig.Type == "REDIRECT" {
			active++
			text = extractText(sig.Content)
		}
	}
	if active != 1 {
		t.Fatalf("expected exactly 1 active REDIRECT, got %d", active)
	}
	if !strings.Contains(text, newestMessage) {
		t.Errorf("REDIRECT text does not contain the newest failure's own message: %q", text)
	}
}

func TestTwoFailuresProduceNoRedirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	seedMiddenEntries(t, s, []colony.MiddenEntry{
		{ID: "m1", Timestamp: "2026-08-01T00:00:00Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "failure one"},
		{ID: "m2", Timestamp: "2026-08-01T00:00:01Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "failure two"},
	})

	crossed := emitMiddenThresholdRedirect()
	if crossed != 0 {
		t.Fatalf("emitMiddenThresholdRedirect returned %d, want 0", crossed)
	}

	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err == nil {
		for _, sig := range pf.Signals {
			if sig.Active && sig.Type == "REDIRECT" {
				t.Fatalf("expected zero REDIRECT signals for two unacknowledged failures, found one: %+v", sig)
			}
		}
	}
}

func TestFourthFailureReinforcesTheSameRedirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	repeatedMessage := "go build ./cmd/aether failed: undefined symbol"
	seedMiddenEntries(t, s, []colony.MiddenEntry{
		{ID: "m1", Timestamp: "2026-08-01T00:00:00Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "unrelated first failure"},
		{ID: "m2", Timestamp: "2026-08-01T00:00:01Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "unrelated second failure"},
		{ID: "m3", Timestamp: "2026-08-01T00:00:02Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: repeatedMessage},
	})

	if crossed := emitMiddenThresholdRedirect(); crossed != 1 {
		t.Fatalf("first emitMiddenThresholdRedirect returned %d, want 1", crossed)
	}

	// A fourth, genuinely repeated occurrence of the SAME newest failure --
	// the newest entry's own message is unchanged, so the emitted text (and
	// its content hash) stays stable and the writer reinforces instead of
	// creating a second signal.
	seedMiddenEntries(t, s, []colony.MiddenEntry{
		{ID: "m1", Timestamp: "2026-08-01T00:00:00Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "unrelated first failure"},
		{ID: "m2", Timestamp: "2026-08-01T00:00:01Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "unrelated second failure"},
		{ID: "m3", Timestamp: "2026-08-01T00:00:02Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: repeatedMessage},
		{ID: "m4", Timestamp: "2026-08-01T00:00:03Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: repeatedMessage},
	})

	if crossed := emitMiddenThresholdRedirect(); crossed != 1 {
		t.Fatalf("second emitMiddenThresholdRedirect returned %d, want 1", crossed)
	}

	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones.json: %v", err)
	}
	active := 0
	var sig colony.PheromoneSignal
	for _, s2 := range pf.Signals {
		if s2.Active && s2.Type == "REDIRECT" {
			active++
			sig = s2
		}
	}
	if active != 1 {
		t.Fatalf("expected the signal count to stay at 1 across the fourth failure, got %d", active)
	}
	if sig.ReinforcementCount == nil || *sig.ReinforcementCount < 1 {
		t.Errorf("expected the reinforcement count to rise after the fourth failure, got %v", sig.ReinforcementCount)
	}
}

func TestAcknowledgedFailuresDoNotCountTowardTheThreshold(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ack := true
	seedMiddenEntries(t, s, []colony.MiddenEntry{
		{ID: "m1", Timestamp: "2026-08-01T00:00:00Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "failure one", Acknowledged: &ack},
		{ID: "m2", Timestamp: "2026-08-01T00:00:01Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "failure two", Acknowledged: &ack},
		{ID: "m3", Timestamp: "2026-08-01T00:00:02Z", Category: middenCategoryCheckFailed, Source: "aether continue", Message: "failure three", Acknowledged: &ack},
	})

	crossed := emitMiddenThresholdRedirect()
	if crossed != 0 {
		t.Fatalf("emitMiddenThresholdRedirect returned %d, want 0 for three ACKNOWLEDGED failures", crossed)
	}

	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err == nil {
		for _, sig := range pf.Signals {
			if sig.Active && sig.Type == "REDIRECT" {
				t.Fatalf("expected zero REDIRECT signals for three acknowledged failures, found one: %+v", sig)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Task 3 (198.1-04): a valuable note that expires is kept in long-term
// memory.
// ---------------------------------------------------------------------------

func TestExpiringAValuableNoteKeepsItInLongTermMemory(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	text := "never edit cmd/pheromone_write.go expiry without running go test ./cmd/ -run TestPheromoneExpire"
	if _, _, err := writePheromoneSignal("REDIRECT", text, "", "test", "", "", 0, nil); err != nil {
		t.Fatalf("seed REDIRECT signal: %v", err)
	}

	expired, promoted := expireSignalsByType(s, "REDIRECT")
	if expired != 1 {
		t.Fatalf("expireSignalsByType expired = %d, want 1", expired)
	}
	if promoted != 1 {
		t.Fatalf("expireSignalsByType promoted = %d, want 1", promoted)
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "eternal", "memory.json"))
	if err != nil {
		t.Fatalf("read eternal memory.json: %v", err)
	}
	var ed struct {
		Entries []struct {
			Content string `json:"content"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &ed); err != nil {
		t.Fatalf("unmarshal eternal memory.json: %v", err)
	}
	found := false
	for _, e := range ed.Entries {
		if e.Content == text {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an eternal memory entry with content %q, got %+v", text, ed.Entries)
	}
}

func TestExpiringAThrowawayNoteKeepsNothing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	if _, _, err := writePheromoneSignal("FEEDBACK", "low-value note never reinforced", "", "test", "", "", 0, nil); err != nil {
		t.Fatalf("seed FEEDBACK signal: %v", err)
	}

	expired, promoted := expireSignalsByType(s, "FEEDBACK")
	if expired != 1 {
		t.Fatalf("expireSignalsByType expired = %d, want 1", expired)
	}
	if promoted != 0 {
		t.Fatalf("expireSignalsByType promoted = %d, want 0", promoted)
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "eternal", "memory.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read eternal memory.json: %v", err)
	}
	var ed struct {
		Entries []interface{} `json:"entries"`
	}
	if err := json.Unmarshal(data, &ed); err != nil {
		t.Fatalf("unmarshal eternal memory.json: %v", err)
	}
	if len(ed.Entries) != 0 {
		t.Fatalf("expected zero eternal memory entries for a throwaway note, got %d", len(ed.Entries))
	}
}

func TestExpiringTheSameNoteTwiceKeepsOneCopy(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	appended1, err := appendEternalMemoryEntry("repeated valuable note", "promoted_on_expire", 0.9, "promoted_on_expire")
	if err != nil {
		t.Fatalf("first append: %v", err)
	}
	if !appended1 {
		t.Fatalf("expected the first append to succeed")
	}
	appended2, err := appendEternalMemoryEntry("repeated valuable note", "promoted_on_expire", 0.9, "promoted_on_expire")
	if err != nil {
		t.Fatalf("second append: %v", err)
	}
	if appended2 {
		t.Fatalf("expected the second identical append to be skipped as a duplicate")
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "eternal", "memory.json"))
	if err != nil {
		t.Fatalf("read eternal memory.json: %v", err)
	}
	var ed struct {
		Entries []interface{} `json:"entries"`
	}
	if err := json.Unmarshal(data, &ed); err != nil {
		t.Fatalf("unmarshal eternal memory.json: %v", err)
	}
	if len(ed.Entries) != 1 {
		t.Fatalf("expected exactly 1 entry after two identical appends, got %d", len(ed.Entries))
	}
}

func TestEternalWriterShapeMatchesTheEternalStoreCommand(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	rootCmd.SetArgs([]string{"eternal-store", "--content", "cli-written entry", "--category", "general", "--confidence", "0.9"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("eternal-store failed: %v", err)
	}

	if _, err := appendEternalMemoryEntry("writer-written entry", "general", 0.9, "test"); err != nil {
		t.Fatalf("appendEternalMemoryEntry failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "eternal", "memory.json"))
	if err != nil {
		t.Fatalf("read eternal memory.json: %v", err)
	}
	var ed struct {
		Entries []map[string]interface{} `json:"entries"`
	}
	if err := json.Unmarshal(data, &ed); err != nil {
		t.Fatalf("unmarshal eternal memory.json: %v", err)
	}
	if len(ed.Entries) != 2 {
		t.Fatalf("expected 2 entries (one per writer), got %d: %+v", len(ed.Entries), ed.Entries)
	}

	keySet := func(m map[string]interface{}) map[string]bool {
		keys := map[string]bool{}
		for k := range m {
			keys[k] = true
		}
		return keys
	}
	a := keySet(ed.Entries[0])
	b := keySet(ed.Entries[1])
	if len(a) != len(b) {
		t.Fatalf("key sets differ in size: %v vs %v", a, b)
	}
	for k := range a {
		if !b[k] {
			t.Errorf("key %q present in the eternal-store command's entry but not the writer's", k)
		}
	}
	for k := range b {
		if !a[k] {
			t.Errorf("key %q present in the writer's entry but not the eternal-store command's", k)
		}
	}
}

func TestExpiryStillSucceedsWhenTheHubIsUnwritable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Point the hub at a path that is a FILE, not a directory, so
	// os.MkdirAll(hub/"eternal", ...) fails reliably and portably --
	// promotion must warn to stderr and never fail the expiry itself.
	hubParent := t.TempDir()
	hubFile := filepath.Join(hubParent, "hub-is-a-file")
	if err := os.WriteFile(hubFile, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("seed hub-as-file fixture: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hubFile)

	text := "avoid this pattern, it caused an outage"
	if _, _, err := writePheromoneSignal("REDIRECT", text, "", "test", "", "", 0, nil); err != nil {
		t.Fatalf("seed REDIRECT signal: %v", err)
	}

	expired, promoted := expireSignalsByType(s, "REDIRECT")
	if expired != 1 {
		t.Fatalf("expireSignalsByType expired = %d, want 1 (expiry itself must succeed despite the hub being unwritable)", expired)
	}
	if promoted != 0 {
		t.Fatalf("expireSignalsByType promoted = %d, want 0 (the hub write should have failed silently)", promoted)
	}
}
