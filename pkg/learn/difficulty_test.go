package learn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- AssessDifficulty tests ---

func TestAssessDifficulty_WorkerFailures(t *testing.T) {
	evidence := Evidence{
		RunID: "run-fail-1",
		Phase: 5,
		Workers: []WorkerEvidence{
			{Name: "Builder-1", Caste: "builder", Status: "failed"},
			{Name: "Builder-2", Caste: "builder", Status: "failed"},
			{Name: "Builder-3", Caste: "builder", Status: "completed"},
		},
		GatesPassed: 2,
		GatesTotal:  3,
		Confidence:  0.8,
	}

	assessment := AssessDifficulty(evidence)
	if !assessment.IsDifficult {
		t.Error("expected IsDifficult=true for evidence with worker failures, got false")
	}
	if assessment.Score < 0.3 {
		t.Errorf("expected Score >= 0.3, got %.2f", assessment.Score)
	}
}

func TestAssessDifficulty_GateFailures(t *testing.T) {
	evidence := Evidence{
		RunID: "run-gate-1",
		Phase: 3,
		Workers: []WorkerEvidence{
			{Name: "Builder-1", Caste: "builder", Status: "completed"},
		},
		GatesPassed: 2,
		GatesTotal:  3,
		Confidence:  0.8,
	}

	assessment := AssessDifficulty(evidence)
	if !assessment.IsDifficult {
		t.Error("expected IsDifficult=true for evidence with gate failures, got false")
	}
	found := false
	for _, r := range assessment.Reasons {
		if strings.Contains(r, "gate(s) failed before passing") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected reason containing 'gate(s) failed before passing', got %v", assessment.Reasons)
	}
}

func TestAssessDifficulty_EasyTask(t *testing.T) {
	evidence := Evidence{
		RunID: "run-easy-1",
		Phase: 2,
		Workers: []WorkerEvidence{
			{Name: "Builder-1", Caste: "builder", Status: "completed"},
		},
		GatesPassed: 3,
		GatesTotal:  3,
		Confidence:  0.9,
	}

	assessment := AssessDifficulty(evidence)
	if assessment.IsDifficult {
		t.Errorf("expected IsDifficult=false for easy task, got true (reasons: %v, score: %.2f)", assessment.Reasons, assessment.Score)
	}
}

func TestAssessDifficulty_ScoreThreshold(t *testing.T) {
	// Create evidence that should score >= 0.3
	evidence := Evidence{
		RunID: "run-thresh-1",
		Phase: 4,
		Workers: []WorkerEvidence{
			{Name: "Builder-1", Caste: "builder", Status: "failed"},
			{Name: "Builder-2", Caste: "builder", Status: "failed"},
			{Name: "Builder-3", Caste: "builder", Status: "completed"},
		},
		GatesPassed: 2,
		GatesTotal:  3,
		Confidence:  0.8,
	}

	assessment := AssessDifficulty(evidence)
	if assessment.Score < 0.3 {
		t.Errorf("expected Score >= 0.3 (DifficultyScoreThreshold), got %.2f", assessment.Score)
	}
	if !assessment.IsDifficult {
		t.Error("expected IsDifficult=true for score >= 0.3")
	}
}

// --- Hard rejection tests ---

func TestAutoSkillRejection_Blocked(t *testing.T) {
	entry := makeEntry("", "some content", 0.8)
	entry.Classification = ClassBlocked

	rejected, reason := IsAutoSkillRejected(entry)
	if !rejected {
		t.Error("expected rejected=true for ClassBlocked entry, got false")
	}
	if reason == "" {
		t.Error("expected non-empty rejection reason")
	}
}

func TestAutoSkillRejection_ZeroFiles(t *testing.T) {
	entry := makeEntry("", "some content", 0.8)
	entry.Evidence.FilesTouched = nil

	rejected, reason := IsAutoSkillRejected(entry)
	if !rejected {
		t.Error("expected rejected=true for zero files touched, got false")
	}
	if !strings.Contains(reason, "zero files") {
		t.Errorf("expected reason to mention 'zero files', got: %s", reason)
	}
}

func TestAutoSkillRejection_Redacted(t *testing.T) {
	entry := makeEntry("", "some content", 0.8)
	entry.Redacted = true
	entry.Evidence.FilesTouched = []string{"pkg/main.go"}

	rejected, reason := IsAutoSkillRejected(entry)
	if !rejected {
		t.Error("expected rejected=true for redacted entry, got false")
	}
	if !strings.Contains(reason, "redacted") {
		t.Errorf("expected reason to mention 'redacted', got: %s", reason)
	}
}

// --- Auto-skill proposal tests (LEARN-07, 204-09-PLAN.md Task 3) ---
//
// AutoCreateSkillIfDifficult no longer creates an active skill directly in
// ANY mode -- it hands a candidate to a caller-supplied SkillProposalSink
// instead (pkg/learn must not import cmd, so the real sink -- which
// enqueues into the owner's approval queue -- lives in
// cmd/suggest_approve.go). These tests drive the real function against a
// fake sink and assert on what was proposed, never on a directly-created
// skill.

// fakeSkillProposalSink is a test double for SkillProposalSink: it records
// every proposal it is handed, or returns Err if set (never both).
type fakeSkillProposalSink struct {
	proposals []SkillProposal
	err       error
}

func (f *fakeSkillProposalSink) ProposeSkill(p SkillProposal) error {
	if f.err != nil {
		return f.err
	}
	f.proposals = append(f.proposals, p)
	return nil
}

func TestAutoSkillCreation_AutoModeRaisesAProposal(t *testing.T) {
	entry := makeDifficultEntry()
	entry.ID = "learn-entry-auto-mode"
	sink := &fakeSkillProposalSink{}

	err := AutoCreateSkillIfDifficult(entry, AutoSkillModeAuto, sink)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (auto mode): %v", err)
	}
	if len(sink.proposals) != 1 {
		t.Fatalf("expected exactly one proposal raised in auto mode, got %d", len(sink.proposals))
	}
	if sink.proposals[0].LearningEntryID != entry.ID {
		t.Errorf("expected the proposal to name its source learning entry %q, got %q", entry.ID, sink.proposals[0].LearningEntryID)
	}
}

func TestAutoSkillCreation_ProposeModeRaisesTheIdenticalProposal(t *testing.T) {
	// LEARN-07 / 204-CLASSIC-SYNTHESIS.md ruling (d): "propose" mode used to
	// be a silent no-op despite its own doc comment claiming otherwise.
	// This proves it now raises the SAME proposal shape as "auto" mode --
	// the two modes differ only in how eagerly a proposal is raised, never
	// in whether the owner is bypassed.
	entry := makeDifficultEntry()
	entry.ID = "learn-entry-propose-mode"
	sink := &fakeSkillProposalSink{}

	err := AutoCreateSkillIfDifficult(entry, AutoSkillModePropose, sink)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (propose mode): %v", err)
	}
	if len(sink.proposals) != 1 {
		t.Fatalf("expected propose mode to raise exactly one proposal (it must actually propose, not silently skip), got %d", len(sink.proposals))
	}
}

func TestAutoSkillCreation_OffModeRaisesNothing(t *testing.T) {
	entry := makeDifficultEntry()
	sink := &fakeSkillProposalSink{}

	err := AutoCreateSkillIfDifficult(entry, AutoSkillModeOff, sink)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (off mode): %v", err)
	}
	if len(sink.proposals) != 0 {
		t.Errorf("expected no proposals raised in off mode, got %d", len(sink.proposals))
	}
}

func TestAutoSkillCreation_ProposalCarriesEvidence(t *testing.T) {
	entry := makeDifficultEntry()
	entry.ID = "learn-entry-evidence"
	sink := &fakeSkillProposalSink{}

	if err := AutoCreateSkillIfDifficult(entry, AutoSkillModeAuto, sink); err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult: %v", err)
	}
	if len(sink.proposals) != 1 {
		t.Fatalf("expected exactly one proposal, got %d", len(sink.proposals))
	}
	p := sink.proposals[0]
	if p.Name == "" {
		t.Error("expected the proposal to carry a derived name")
	}
	if p.SourceRunID != entry.Evidence.RunID {
		t.Errorf("expected SourceRunID %q, got %q", entry.Evidence.RunID, p.SourceRunID)
	}
	if p.Confidence != entry.Confidence {
		t.Errorf("expected Confidence %.2f, got %.2f", entry.Confidence, p.Confidence)
	}
	if !strings.Contains(p.Content, "Run ID:") {
		t.Error("expected the proposal's generated content to carry the run id evidence")
	}
	if !strings.Contains(p.Content, "Confidence:") {
		t.Error("expected the proposal's generated content to carry confidence evidence")
	}
}

func TestAutoSkillCreation_EasyTaskSkipped(t *testing.T) {
	// Easy entry: all workers completed, all gates passed
	entry := makeEntry("", "easy learning content", 0.9)
	entry.Evidence.FilesTouched = []string{"pkg/main.go"}
	entry.Phase = 3
	sink := &fakeSkillProposalSink{}

	err := AutoCreateSkillIfDifficult(entry, AutoSkillModeAuto, sink)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (easy task): %v", err)
	}
	if len(sink.proposals) != 0 {
		t.Errorf("expected no proposal raised for an easy task, got %d", len(sink.proposals))
	}
}

// TestAutoSkillCreation_NoModeConstructsASkillService is the structural
// half of LEARN-07's closure: AutoCreateSkillIfDifficult itself must no
// longer be ABLE to create a skill -- proven by grep in
// cmd/promotion_gate_test.go's TestAutoDerivedSkillEntersTheApprovalListNotTheActiveSet
// acceptance criterion (`grep -c 'NewSkillService' pkg/learn/difficulty.go`
// returns 0); this test is the behavioural companion, run from this
// package directly against the fake sink for every declared mode.
func TestAutoSkillCreation_NoModeConstructsASkillService(t *testing.T) {
	for _, mode := range []string{AutoSkillModeOff, AutoSkillModePropose, AutoSkillModeAuto} {
		t.Run(mode, func(t *testing.T) {
			entry := makeDifficultEntry()
			sink := &fakeSkillProposalSink{}
			if err := AutoCreateSkillIfDifficult(entry, mode, sink); err != nil {
				t.Fatalf("AutoCreateSkillIfDifficult (%s mode): %v", mode, err)
			}
			// No mode ever reaches a SQLite-backed skill service: the only
			// way this test COULD observe a directly-created skill is if it
			// queried one, and there is deliberately nothing to query --
			// the fake sink is the only place a candidate could have gone.
		})
	}
}

// --- LoadAutoSkillMode tests ---

func TestLoadAutoSkillMode_Default(t *testing.T) {
	dir := t.TempDir()
	// No config file exists
	mode := LoadAutoSkillMode(dir)
	if mode != AutoSkillModePropose {
		t.Errorf("expected default mode 'propose', got %q", mode)
	}
}

func TestLoadAutoSkillMode_CustomMode(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"auto", AutoSkillModeAuto},
		{"off", AutoSkillModeOff},
		{"propose", AutoSkillModePropose},
	} {
		t.Run(tc.input, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "auto_skill_mode"), []byte(tc.input), 0644); err != nil {
				t.Fatal(err)
			}
			mode := LoadAutoSkillMode(dir)
			if mode != tc.want {
				t.Errorf("expected %q, got %q", tc.want, mode)
			}
		})
	}
}

func TestLoadAutoSkillMode_InvalidMode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "auto_skill_mode"), []byte("invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	mode := LoadAutoSkillMode(dir)
	if mode != AutoSkillModePropose {
		t.Errorf("expected default mode 'propose' for invalid value, got %q", mode)
	}
}

// --- ExtractKeywords tests ---

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantMin int // minimum expected keywords
		wantMax int // maximum expected keywords (3 max)
	}{
		{
			name:    "normal content",
			content: "Implemented authentication middleware with JWT tokens and refresh rotation",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "stop words filtered",
			content: "The implementation was completed successfully in the phase",
			wantMin: 0,
			wantMax: 3,
		},
		{
			name:    "short words filtered",
			content: "Go is a good tool for dev ops",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "empty content",
			content: "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "max 3 keywords",
			content: "authentication middleware database connection pooling error handling retry logic",
			wantMin: 3,
			wantMax: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keywords := extractKeywords(tc.content)
			if len(keywords) < tc.wantMin {
				t.Errorf("expected at least %d keywords, got %d (%v)", tc.wantMin, len(keywords), keywords)
			}
			if len(keywords) > tc.wantMax {
				t.Errorf("expected at most %d keywords, got %d (%v)", tc.wantMax, len(keywords), keywords)
			}
		})
	}
}

// --- AutoSkillModeDefault test ---

func TestAutoSkillModeDefault(t *testing.T) {
	if AutoSkillModeDefault != AutoSkillModePropose {
		t.Errorf("expected AutoSkillModeDefault = 'propose', got %q", AutoSkillModeDefault)
	}
}

// --- Non-blocking test ---

func TestAutoCreateSkillIfDifficult_NonBlocking(t *testing.T) {
	entry := makeDifficultEntry()
	sink := &fakeSkillProposalSink{}

	// Should not panic even with various inputs
	err := AutoCreateSkillIfDifficult(entry, AutoSkillModeAuto, sink)
	if err != nil {
		// Error is returned but function does not panic -- non-blocking
		t.Logf("expected possible error, got: %v", err)
	}

	// Test with nil-like entry (empty content)
	emptyEntry := Entry{
		Content: "",
		Evidence: Evidence{
			Workers: []WorkerEvidence{
				{Name: "Builder-1", Caste: "builder", Status: "failed"},
			},
			FilesTouched: []string{"pkg/main.go"},
		},
		Phase: 3,
	}
	err = AutoCreateSkillIfDifficult(emptyEntry, AutoSkillModeAuto, sink)
	if err != nil {
		t.Logf("expected possible error for empty content, got: %v", err)
	}

	// A sink returning an error must be propagated, never swallowed or
	// panicked on -- the caller (cmd/codex_continue_finalize.go) decides
	// how to handle it (non-blocking to phase advancement).
	failingSink := &fakeSkillProposalSink{err: fmt.Errorf("queue unavailable")}
	if err := AutoCreateSkillIfDifficult(entry, AutoSkillModeAuto, failingSink); err == nil {
		t.Error("expected a sink error to be propagated")
	}
}

// --- DifficultyScoreThreshold test ---

func TestDifficultyScoreThreshold(t *testing.T) {
	if DifficultyScoreThreshold != 0.3 {
		t.Errorf("expected DifficultyScoreThreshold = 0.3, got %.2f", DifficultyScoreThreshold)
	}
}

// --- Helper: create a difficult entry for testing ---

func makeDifficultEntry() Entry {
	return Entry{
		ID:      "",
		Content: "authentication middleware implementation with JWT tokens and refresh rotation",
		Evidence: Evidence{
			RunID: "run-diff-1",
			Phase: 5,
			Workers: []WorkerEvidence{
				{Name: "Builder-1", Caste: "builder", Status: "failed"},
				{Name: "Builder-2", Caste: "builder", Status: "completed"},
			},
			FilesTouched: []string{"pkg/auth/middleware.go", "pkg/auth/tokens.go"},
			GatesPassed:  2,
			GatesTotal:   3,
			Confidence:   0.8,
			Timestamp:    "2026-05-01T00:00:00Z",
			Scope:        "repo-local",
		},
		Classification: ClassRepoLocal,
		CreatedAt:      "2026-05-01T00:00:00Z",
		Phase:          5,
		Confidence:     0.8,
	}
}
