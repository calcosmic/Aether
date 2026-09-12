package cmd

// Phase 202.1 plan 05 -- closing the confirmed identifier leak, and the
// sibling it has, at the writer rather than the reader.
//
// nextActionEventSentence (cmd/next_action.go) is documented, correctly, to
// treat the last field of a stored lifecycle event
// (`timestamp|event_type|source|message`) as a finished human sentence.
// pauseResumeLifecycleEvent used to put `handoff=<id> <detail>` in that
// field instead -- a raw handoff identifier followed by a raw enumeration
// token, on both the pause and the resume path. 202.1-RESEARCH.md's Pitfall
// 3 names the tempting wrong fix (special-case the `handoff=` prefix inside
// the reader) and selects the other one instead: fix the writer, so the
// reader's own documented contract becomes true rather than merely assumed.
//
// This file proves both halves of that fix: the writer now only accepts a
// lifecycleEventSentence (built by one of exactly two constructors), and a
// structural guard refuses any future bypass of that type at compile time.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// underscoreTokenPattern matches an internal, hand_typed lowercase token --
// the shape of every one of pauseSafeBoundary's seven boundary values and
// colony.RecoveryProvenance's four values. A finished English sentence never
// contains a run of lowercase words joined by underscores, so this is the
// structural signature of a leaked internal token, not a hand-picked list of
// banned words.
var underscoreTokenPattern = regexp.MustCompile(`[a-z]+_[a-z_]+`)

// assertLifecycleSentence asserts sentence is a genuine, owner-readable
// sentence rather than an echo of sourceValue (the raw boundary or
// provenance value it was derived from).
func assertLifecycleSentence(t *testing.T, sourceValue, sentence string) {
	t.Helper()
	if strings.TrimSpace(sentence) == "" {
		t.Fatalf("no sentence produced for %q", sourceValue)
	}
	if sentence == sourceValue {
		t.Fatalf("sentence for %q echoes the raw value instead of describing it", sourceValue)
	}
	if strings.Contains(sentence, "=") {
		t.Fatalf("sentence for %q contains an equals sign, suggesting a raw token leaked through: %q", sourceValue, sentence)
	}
	if underscoreTokenPattern.MatchString(sentence) {
		t.Fatalf("sentence for %q contains an underscore-joined token: %q", sourceValue, sentence)
	}
	trimmed := strings.TrimSpace(sentence)
	last := trimmed[len(trimmed)-1]
	if last != '.' && last != '!' && last != '?' {
		t.Fatalf("sentence for %q does not end as a sentence: %q", sourceValue, sentence)
	}
}

// writeBoundaryTestAttempt writes a minimal, real build-attempt record and
// its latest-attempt pointer to the current store, so
// loadRelevantBuildAttemptReadOnly (cmd/session_flow_cmds.go) -- the exact
// read path pauseSafeBoundary uses -- resolves it for phaseID. alive
// controls whether buildAttemptProcessAlive reports the attempt as still
// running: an inactive status (buildAttemptFailed) makes that false
// regardless of the process ID, which is all pauseSafeBoundary's
// interrupted_attempt_boundary branch needs.
func writeBoundaryTestAttempt(t *testing.T, phaseID int, alive bool) {
	t.Helper()
	status := buildAttemptFailed
	pid := 0
	if alive {
		status = buildAttemptDispatching
		pid = os.Getpid()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	attemptID := fmt.Sprintf("attempt-boundary-%d", phaseID)
	record := buildAttemptRecord{
		SchemaVersion: buildAttemptSchemaVersion,
		ID:            attemptID,
		Phase:         phaseID,
		Status:        status,
		StartedAt:     now,
		UpdatedAt:     now,
		ProcessID:     pid,
	}
	attemptRel := fmt.Sprintf("build/phase-%d/attempt-boundary.json", phaseID)
	if err := store.SaveJSON(attemptRel, record); err != nil {
		t.Fatalf("save boundary attempt record: %v", err)
	}
	if err := store.SaveJSON(latestBuildAttemptPointerPath(phaseID), latestBuildAttemptPointer{
		SchemaVersion: buildAttemptSchemaVersion,
		AttemptID:     attemptID,
		Path:          attemptRel,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("save boundary attempt pointer: %v", err)
	}
}

// TestEveryPauseBoundaryAndProvenanceHasASentence drives pauseSafeBoundary
// over colony states constructed to produce each of its seven declared
// return values (never a hand-typed list of the seven strings), and
// enumerates colony.RecoveryProvenance's four declared constants directly,
// asserting pauseBoundarySentence / resumeProvenanceSentence produce a
// genuine sentence for every one -- including an unrecognised value, which
// must never echo what it was given.
func TestEveryPauseBoundaryAndProvenanceHasASentence(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	cases := []struct {
		want  string
		state colony.ColonyState
		setup func()
	}{
		{
			want:  "worker_completion_boundary",
			state: colony.ColonyState{State: colony.StateEXECUTING, CurrentPhase: 1},
			setup: func() { writeBoundaryTestAttempt(t, 1, true) },
		},
		{
			want:  "interrupted_attempt_boundary",
			state: colony.ColonyState{State: colony.StateEXECUTING, CurrentPhase: 2},
			setup: func() { writeBoundaryTestAttempt(t, 2, false) },
		},
		{
			want:  "interrupted_execution_boundary",
			state: colony.ColonyState{State: colony.StateEXECUTING, CurrentPhase: 3},
		},
		{
			want:  "post_build_verification_boundary",
			state: colony.ColonyState{State: colony.StateBUILT, CurrentPhase: 4},
		},
		{
			want:  "completed_episode_boundary",
			state: colony.ColonyState{State: colony.StateCOMPLETED, CurrentPhase: 5},
		},
		{
			want:  "idle_command_boundary",
			state: colony.ColonyState{State: colony.StateIDLE, CurrentPhase: 6},
		},
		{
			want:  "between_commands_boundary",
			state: colony.ColonyState{State: colony.StateREADY, CurrentPhase: 7},
		},
	}

	seen := map[string]bool{}
	for _, tc := range cases {
		if tc.setup != nil {
			tc.setup()
		}
		boundary, _, pending := pauseSafeBoundary(tc.state)
		if boundary != tc.want {
			t.Fatalf("fixture for %q actually produced boundary %q (pending=%v) -- the fixture construction is wrong, not the sentence mapping", tc.want, boundary, pending)
		}
		seen[boundary] = true
		assertLifecycleSentence(t, boundary, string(pauseBoundarySentence(boundary)))
	}
	if len(seen) != 7 {
		t.Fatalf("drove %d distinct boundary values, want all 7 pauseSafeBoundary declares", len(seen))
	}
	// An unrecognised boundary value must still produce a sentence, never an
	// echo of the value it was given.
	assertLifecycleSentence(t, "some_future_boundary_value", string(pauseBoundarySentence("some_future_boundary_value")))

	for _, provenance := range []colony.RecoveryProvenance{
		colony.RecoveryProvenanceConfirmed,
		colony.RecoveryProvenanceReconstructed,
		colony.RecoveryProvenanceConflicting,
		colony.RecoveryProvenanceUnknown,
	} {
		assertLifecycleSentence(t, string(provenance), string(resumeProvenanceSentence(provenance)))
	}
	// An unrecognised provenance value must also never echo what it was given.
	assertLifecycleSentence(t, "some-future-provenance", string(resumeProvenanceSentence(colony.RecoveryProvenance("some-future-provenance"))))
}

// extractCardSection returns the rendered text between a "── title ──" stage
// marker and the next one (or the end of the card), failing the test if the
// section is not present at all.
func extractCardSection(t *testing.T, rendered, title string) string {
	t.Helper()
	marker := "── " + title + " ──"
	idx := strings.Index(rendered, marker)
	if idx == -1 {
		t.Fatalf("card has no %q section:\n%s", title, rendered)
	}
	rest := rendered[idx+len(marker):]
	if next := strings.Index(rest, "── "); next != -1 {
		rest = rest[:next]
	}
	return rest
}

// TestPausedCardShowsASentenceNotAStateToken drives the real pause path
// (pauseColonyAt), so the lifecycle event is written by production code
// rather than assembled by the test, then renders the what-next card from
// the resulting state and asserts the "What changed" section shows the
// sentence pauseBoundarySentence produces for this fixture's boundary, and
// contains neither the handoff identifier nor any underscore-joined token.
func TestPausedCardShowsASentenceNotAStateToken(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	preState := load199State(t)
	wantBoundary, _, pending := pauseSafeBoundary(preState)
	if pending {
		t.Fatalf("fixture produced a pending boundary %q; pause would not commit an event", wantBoundary)
	}

	if _, err := pauseColonyAt(fixture.now); err != nil {
		t.Fatalf("pause: %v", err)
	}
	state := load199State(t)
	if state.PauseHandoff == nil || strings.TrimSpace(state.PauseHandoff.ID) == "" {
		t.Fatal("pause did not record a durable handoff reference")
	}
	handoffID := state.PauseHandoff.ID

	answer := resolveNextAction(nextActionInput{State: state})
	rendered := renderNextActionCardForPlatform(answer, "claude")
	section := extractCardSection(t, rendered, "What changed")

	wantSentence := string(pauseBoundarySentence(wantBoundary))
	if !strings.Contains(section, wantSentence) {
		t.Fatalf("changed section does not contain the pause sentence %q:\n%s", wantSentence, section)
	}
	if strings.Contains(section, "handoff=") {
		t.Fatalf("changed section leaks the raw event field prefix:\n%s", section)
	}
	if strings.Contains(section, handoffID) {
		t.Fatalf("changed section leaks the handoff identifier %q:\n%s", handoffID, section)
	}
	if underscoreTokenPattern.MatchString(section) {
		t.Fatalf("changed section contains an underscore-joined token:\n%s", section)
	}
}

// TestResumedCardShowsASentenceNotAStateToken is
// TestPausedCardShowsASentenceNotAStateToken's sibling on the resume path:
// it drives pauseColonyAt then resumeColonyAt (both real production paths),
// then renders the what-next card and asserts the same three things.
func TestResumedCardShowsASentenceNotAStateToken(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	if _, err := pauseColonyAt(fixture.now); err != nil {
		t.Fatalf("pause: %v", err)
	}
	outcome, err := resumeColonyAt(fixture.now.Add(time.Minute))
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	state := load199State(t)
	answer := resolveNextAction(nextActionInput{State: state})
	rendered := renderNextActionCardForPlatform(answer, "claude")
	section := extractCardSection(t, rendered, "What changed")

	wantSentence := string(resumeProvenanceSentence(outcome.Provenance))
	if !strings.Contains(section, wantSentence) {
		t.Fatalf("changed section does not contain the resume sentence %q:\n%s", wantSentence, section)
	}
	if strings.Contains(section, "handoff=") {
		t.Fatalf("changed section leaks the raw event field prefix:\n%s", section)
	}
	if outcome.Handoff.HandoffID != "" && strings.Contains(section, outcome.Handoff.HandoffID) {
		t.Fatalf("changed section leaks the handoff identifier %q:\n%s", outcome.Handoff.HandoffID, section)
	}
	if underscoreTokenPattern.MatchString(section) {
		t.Fatalf("changed section contains an underscore-joined token:\n%s", section)
	}
}

// TestStoredEventShapeIsUnchangedForExistingReaders writes one new pause
// event and one new resume event through production code
// (pauseResumeLifecycleEvent) and asserts phaseCompletionsSince
// (cmd/signal_housekeeping.go), the health scanner's event-format
// validation (scanColonyState, cmd/medic_scanner.go), and
// planningOutcomeEvidence (cmd/codex_plan.go) all behave exactly as they do
// on a pre-existing well-formed four-field record: none of the three reads
// past the first two fields (timestamp, event code), so removing the
// identifier from the fourth field changes nothing they observe.
func TestStoredEventShapeIsUnchangedForExistingReaders(t *testing.T) {
	now := time.Date(2026, time.September, 12, 10, 0, 0, 0, time.UTC)
	pauseEvent := pauseResumeLifecycleEvent(now, "pause", pauseBoundarySentence("idle_command_boundary"))
	resumeEvent := pauseResumeLifecycleEvent(now.Add(time.Minute), "resume", resumeProvenanceSentence(colony.RecoveryProvenanceConfirmed))

	for _, event := range []string{pauseEvent, resumeEvent} {
		segments := strings.Split(event, "|")
		if len(segments) != 4 {
			t.Fatalf("event %q does not have exactly 4 pipe-delimited fields (has %d)", event, len(segments))
		}
	}

	// phaseCompletionsSince only counts "phase_advanced"/"phase_completed"
	// event codes; a pause/resume event must never be counted.
	state := &colony.ColonyState{Events: []string{pauseEvent, resumeEvent}}
	if got := phaseCompletionsSince(state, now.Add(-time.Hour).Format(time.RFC3339)); got != 0 {
		t.Fatalf("phaseCompletionsSince counted a pause/resume event as a phase completion: %d", got)
	}

	// The health scanner's event-format validation only flags an entry with
	// fewer than 2 pipe-delimited segments as malformed. A well-formed
	// 4-field pause/resume event must never trigger that warning.
	dir := t.TempDir()
	goal := "Preserve a trustworthy recovery point"
	writeJSONFile(t, dir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Events:  []string{pauseEvent, resumeEvent},
	})
	fc := newFileChecker(dir)
	issues := scanColonyState(fc)
	for _, issue := range issues {
		if strings.Contains(issue.Message, "malformed") {
			t.Fatalf("health scanner flagged a well-formed lifecycle event as malformed: %s", issue.Message)
		}
	}

	// planningOutcomeEvidence only picks up events whose lowercased text
	// contains "phase_completed", "verification", or "seal_" -- neither
	// pause nor resume events match any of those substrings.
	if got := planningOutcomeEvidence(colony.ColonyState{Events: []string{pauseEvent, resumeEvent}}); len(got) != 0 {
		t.Fatalf("planningOutcomeEvidence picked up a pause/resume event unexpectedly: %v", got)
	}
}

// ---------------------------------------------------------------------------
// Task 2 -- the class is closed structurally, not just at the one known site.
// ---------------------------------------------------------------------------

// TestLifecycleEventSentenceTypeCannotBeBypassed walks the non-test .go
// files of the cmd package and refuses any conversion into
// lifecycleEventSentence outside its two legitimate constructors.
//
// This exists as a structural guard, rather than relying on
// TestPausedCardShowsASentenceNotAStateToken /
// TestResumedCardShowsASentenceNotAStateToken alone, because a rendered
// check can only catch the one instance someone happened to render: it
// would have passed the day before this leak was found, since nobody had
// yet rendered the paused card with a handoff ID in it. 202.1-RESEARCH.md's
// Pitfall 3 names exactly this failure mode -- patching the one known leak
// while the pattern that produced it (any code being free to hand-build a
// lifecycleEventSentence) remains available for the next writer to repeat.
func TestLifecycleEventSentenceTypeCannotBeBypassed(t *testing.T) {
	sites, err := scanLifecycleSentenceConversions(".")
	if err != nil {
		t.Fatalf("scan cmd/ for lifecycleEventSentence conversions: %v", err)
	}

	legitimate := 0
	var illegitimate []string
	for _, s := range sites {
		if lifecycleSentenceLegitimateConstructors[s.Function] {
			legitimate++
			continue
		}
		illegitimate = append(illegitimate, fmt.Sprintf("%s:%s %s", s.File, s.Function, s.Expr))
	}

	// Guard the guard: a scanner that matches nothing has broken (wrong type
	// name, wrong AST shape, wrong file scope), not proven both real
	// constructors clean overnight.
	if legitimate == 0 {
		t.Fatal("scanLifecycleSentenceConversions found zero conversions inside pauseBoundarySentence/resumeProvenanceSentence -- the scanner is broken, not that both constructors vanished.")
	}

	if len(illegitimate) > 0 {
		sort.Strings(illegitimate)
		t.Errorf("%d conversion(s) into lifecycleEventSentence found outside its two legitimate constructors:\n  %s\nRoute this through pauseBoundarySentence or resumeProvenanceSentence instead.",
			len(illegitimate), strings.Join(illegitimate, "\n  "))
	}
}

// TestLifecycleEventSentenceGuardCanFail proves
// TestLifecycleEventSentenceTypeCannotBeBypassed's scanner actually fires:
// a synthetic file, written to a temp directory, containing a fresh
// function that converts a plain string into lifecycleEventSentence, is
// found. Same shape as TestNextActionHardcodeDetectsAPlantedViolation
// (cmd/next_action_hardcode_ratchet_test.go).
func TestLifecycleEventSentenceGuardCanFail(t *testing.T) {
	tmp := t.TempDir()
	planted := "func plantedLifecycleSentenceBypass(raw string) lifecycleEventSentence {\n\treturn lifecycleEventSentence(raw)\n}\n"
	if err := os.WriteFile(filepath.Join(tmp, "synthetic.go"), []byte("package cmd\n\n"+planted), 0644); err != nil {
		t.Fatalf("write synthetic fixture: %v", err)
	}

	sites, err := scanLifecycleSentenceConversions(tmp)
	if err != nil {
		t.Fatalf("scan synthetic fixture: %v", err)
	}

	found := false
	for _, s := range sites {
		if s.Function == "plantedLifecycleSentenceBypass" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("a planted bypass in a clean synthetic file was not detected; found sites: %+v", sites)
	}
}
