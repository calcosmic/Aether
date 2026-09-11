package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/events"
)

// --- shared fixture -----------------------------------------------------

// setupOracleLiveFixtureWorkspace writes a minimal Go project plus an
// Oracle workspace with questionCount open questions, targetConfidence and
// maxIterations -- exactly the same shape
// TestOracleRunLoopModeResumesInitializedWorkspace already establishes for
// driving `oracle run-loop` in a test. Paired with oracleCompletingInvoker
// (each call answers exactly one question at confidence 99), a fixture with
// 4 questions and a target of 60 completes in exactly 3 rounds: the plan
// average is 99/4=24, 198/4=49, then 297/4=74 -- the first round at or
// above the 60 target. That arithmetic is what gives every test below a
// deterministic, non-flaky round count without needing to guess how many
// questions a topic-derived plan would generate.
func setupOracleLiveFixtureWorkspace(t *testing.T, root string, questionCount, targetConfidence, maxIterations int) oraclePaths {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-live-fixture\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure oracle workspace: %v", err)
	}

	started := "2026-01-01T00:00:00Z"
	state := oracleStateFile{
		Version:           "1.1",
		Topic:             "oracle live fixture topic",
		Scope:             "repo",
		Template:          "custom",
		Phase:             "survey",
		Iteration:         0,
		MaxIterations:     maxIterations,
		TargetConfidence:  targetConfidence,
		OverallConfidence: 0,
		StartedAt:         started,
		LastUpdated:       started,
		Status:            "active",
		Strategy:          defaultOracleStrategy,
		Platform:          "opencode",
	}
	questions := make([]oracleQuestion, 0, questionCount)
	for i := 1; i <= questionCount; i++ {
		questions = append(questions, oracleQuestion{
			ID:          fmt.Sprintf("q%d", i),
			Text:        fmt.Sprintf("Live fixture question %d: what must round %d establish?", i, i),
			Status:      "open",
			KeyFindings: []oracleFinding{},
		})
	}
	plan := oraclePlanFile{
		Version:   "1.1",
		Sources:   map[string]oracleSource{},
		Questions: questions,
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return paths
}

// oracleLiveEventsOfKind filters events to those carrying episode kind
// "oracle" and the given topic.
func oracleLiveEventsOfKind(evts []liveTestEvent, topic string) []liveTestEvent {
	out := make([]liveTestEvent, 0, len(evts))
	for _, e := range evts {
		if e.Topic == topic && e.Payload.EpisodeKind == events.EpisodeKindOracle {
			out = append(out, e)
		}
	}
	return out
}

// --- Task 1: emission at the loop's existing mutation points --------------

// TestOracleRoundsReachTheLiveStream drives a real three-round fixture and
// proves the round-started, confidence-changed and round-ended events all
// reach the persisted live stream carrying the round number, cap, phase,
// question and confidence.
func TestOracleRoundsReachTheLiveStream(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	setupOracleLiveFixtureWorkspace(t, root, 4, 60, 5)

	result, err := runOracleCompatibility(root, []string{"run-loop"}, "", "")
	if err != nil {
		t.Fatalf("oracle run-loop: %v", err)
	}
	if iterations, _ := result["iterations_run"].(int); iterations != 3 {
		t.Fatalf("fixture is broken: iterations_run = %v, want 3 (see setupOracleLiveFixtureWorkspace's doc comment)", result["iterations_run"])
	}

	allEvents := liveEventsSince(t)

	roundStarted := oracleLiveEventsOfKind(allEvents, events.LiveTopicWorkerStarted)
	if len(roundStarted) != 3 {
		t.Fatalf("got %d round-started events, want 3:\n%+v", len(roundStarted), roundStarted)
	}
	for i, e := range roundStarted {
		want := i + 1
		if e.Payload.Wave != want {
			t.Errorf("round-started[%d].Wave = %d, want %d", i, e.Payload.Wave, want)
		}
		if e.Payload.RoundCap != 5 {
			t.Errorf("round-started[%d].RoundCap = %d, want 5", i, e.Payload.RoundCap)
		}
		if e.Payload.Status == "" {
			t.Errorf("round-started[%d] carries no phase", i)
		}
		if e.Payload.Question == "" {
			t.Errorf("round-started[%d] carries no question", i)
		}
	}

	confidenceChanged := oracleLiveEventsOfKind(allEvents, events.LiveTopicConfidenceChanged)
	if len(confidenceChanged) == 0 {
		t.Fatal("no confidence-changed events reached the live stream")
	}
	for _, e := range confidenceChanged {
		if e.Payload.TargetConfidence != 60 {
			t.Errorf("confidence-changed.TargetConfidence = %v, want 60", e.Payload.TargetConfidence)
		}
		if e.Payload.Confidence == e.Payload.PreviousConfidence {
			t.Errorf("confidence-changed carries equal previous (%v) and new (%v) confidence -- should only fire on a genuine change", e.Payload.PreviousConfidence, e.Payload.Confidence)
		}
	}

	roundEnded := oracleLiveEventsOfKind(allEvents, events.LiveTopicWorkerProgress)
	if len(roundEnded) != 3 {
		t.Fatalf("got %d round-ended events, want 3:\n%+v", len(roundEnded), roundEnded)
	}
	for i, e := range roundEnded {
		if e.Payload.Confidence <= 0 {
			t.Errorf("round-ended[%d] carries no confidence", i)
		}
		if i > 0 && e.Payload.Confidence < roundEnded[i-1].Payload.Confidence {
			t.Errorf("round-ended[%d].Confidence = %v, fell below round-ended[%d].Confidence = %v", i, e.Payload.Confidence, i-1, roundEnded[i-1].Payload.Confidence)
		}
	}
}

// oracleGapContradictionInvoker returns the SAME contradiction text every
// round (proving dedup) and a genuinely NEW gap text every round (proving
// each new gap still gets its own event).
type oracleGapContradictionInvoker struct {
	calls int
}

func (i *oracleGapContradictionInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.calls++
	answer := fmt.Sprintf("Round %d gathered genuinely new evidence for its question.", i.calls)
	if err := writeOracleTestResponse(cfg, oracleWorkerResponse{
		QuestionID: oracleTestActiveQuestionID(cfg.Root),
		Status:     "answered",
		Confidence: 99,
		Summary:    answer,
		Findings: []oracleWorkerFinding{{
			Text: answer,
		}},
		Gaps:           []string{fmt.Sprintf("open gap discovered in round %d", i.calls)},
		Contradictions: []string{"Cache invalidation timing is inconsistent across workers"},
		Recommendation: answer,
	}); err != nil {
		return codex.WorkerResult{}, err
	}
	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    "Oracle iteration completed.",
	}, nil
}

func (i *oracleGapContradictionInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleGapContradictionInvoker) ValidateAgent(path string) error      { return nil }

// TestOracleContradictionAnnouncedOnceNotEveryRound proves a contradiction
// present across three rounds produces exactly one live event, while a gap
// newly added each round produces its own event every time.
func TestOracleContradictionAnnouncedOnceNotEveryRound(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleGapContradictionInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	setupOracleLiveFixtureWorkspace(t, root, 4, 60, 5)

	if _, err := runOracleCompatibility(root, []string{"run-loop"}, "", ""); err != nil {
		t.Fatalf("oracle run-loop: %v", err)
	}

	allEvents := liveEventsSince(t)

	contradictionEvents := oracleLiveEventsOfKind(allEvents, events.LiveTopicContradictionFound)
	if len(contradictionEvents) != 1 {
		t.Fatalf("got %d contradiction events for one contradiction repeated every round, want exactly 1:\n%+v", len(contradictionEvents), contradictionEvents)
	}

	gapEvents := oracleLiveEventsOfKind(allEvents, events.LiveTopicGapTargeted)
	if len(gapEvents) != 3 {
		t.Fatalf("got %d gap-targeted events for three genuinely new gaps, want 3:\n%+v", len(gapEvents), gapEvents)
	}
	seen := map[string]bool{}
	for _, e := range gapEvents {
		if len(e.Payload.Findings) != 1 || e.Payload.Findings[0] == "" {
			t.Errorf("gap-targeted event names no gap: %+v", e.Payload)
			continue
		}
		gap := e.Payload.Findings[0]
		if seen[gap] {
			t.Errorf("gap %q announced more than once", gap)
		}
		seen[gap] = true
	}
}

// oracleEmissionFixtureOutcome is the small comparable (all-value-field)
// struct TestFailedOracleEmitNeverChangesTheRun compares between a normal
// run and an emission-forced-to-fail run. StartedAt/LastUpdated/timestamps
// are deliberately excluded -- those differ between any two real-clock runs
// regardless of live-event emission and would make the comparison
// meaningless, mirroring TestFailedLiveEmitNeverChangesLaneOutcome's own
// documented exclusion.
type oracleEmissionFixtureOutcome struct {
	Status              string
	StopReason          string
	Iteration           int
	OverallConfidence   int
	OpenGapsCount       int
	ContradictionsCount int
	ProgressEvents      string
}

// runOracleEmissionFixture runs the same three-round fixture as
// TestOracleRoundsReachTheLiveStream, once normally and once with
// emitColonyLive forced to no-op at its real publish point
// (colonyLiveEmissionFailureOverride), and returns the deterministic facts
// this test compares.
func runOracleEmissionFixture(t *testing.T, forceFailure bool) oracleEmissionFixtureOutcome {
	t.Helper()
	colonyLiveEmissionFailureOverride = forceFailure
	t.Cleanup(func() { colonyLiveEmissionFailureOverride = false })

	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	paths := setupOracleLiveFixtureWorkspace(t, root, 4, 60, 5)

	if _, err := runOracleCompatibility(root, []string{"run-loop"}, "", ""); err != nil {
		t.Fatalf("oracle run-loop (forceFailure=%v): %v", forceFailure, err)
	}

	state, err := loadOracleStateFile(paths.StatePath)
	if err != nil {
		t.Fatalf("load state (forceFailure=%v): %v", forceFailure, err)
	}

	progressEvents, _, err := readOracleProgressFrom(paths.ProgressPath, 0)
	if err != nil {
		t.Fatalf("read progress (forceFailure=%v): %v", forceFailure, err)
	}
	for i := range progressEvents {
		progressEvents[i].Timestamp = ""
	}
	encodedProgress, err := json.Marshal(progressEvents)
	if err != nil {
		t.Fatalf("encode progress (forceFailure=%v): %v", forceFailure, err)
	}

	return oracleEmissionFixtureOutcome{
		Status:              state.Status,
		StopReason:          state.StopReason,
		Iteration:           state.Iteration,
		OverallConfidence:   state.OverallConfidence,
		OpenGapsCount:       len(state.OpenGaps),
		ContradictionsCount: len(state.Contradictions),
		ProgressEvents:      string(encodedProgress),
	}
}

// TestFailedOracleEmitNeverChangesTheRun proves a failed live-event publish
// never changes the research round's outcome: the persisted state and the
// existing round log are identical whether or not live emission itself
// succeeds.
func TestFailedOracleEmitNeverChangesTheRun(t *testing.T) {
	normal := runOracleEmissionFixture(t, false)
	forced := runOracleEmissionFixture(t, true)
	if normal != forced {
		t.Fatalf("oracle run outcome differs with live emission forced to fail:\n normal: %+v\n forced: %+v", normal, forced)
	}
}
