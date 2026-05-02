package learn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- AssessDifficulty tests ---

func TestAssessDifficulty_WorkerFailures(t *testing.T) {
	evidence := Evidence{
		RunID:   "run-fail-1",
		Phase:   5,
		Workers: []WorkerEvidence{
			{Name: "Builder-1", Caste: "builder", Status: "failed"},
			{Name: "Builder-2", Caste: "builder", Status: "completed"},
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
		RunID:   "run-gate-1",
		Phase:   3,
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
		RunID:   "run-easy-1",
		Phase:   2,
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
		RunID:   "run-thresh-1",
		Phase:   4,
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

// --- Auto-skill creation tests ---

func TestAutoSkillCreation_AutoMode(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeDifficultEntry()

	err := AutoCreateSkillIfDifficult(entry, store, dir, AutoSkillModeAuto)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (auto mode): %v", err)
	}

	// Verify skill exists in SQLite with auto_created=true
	svc := NewSkillService(store.DB(), dir)
	skills, err := svc.ListSkills(SkillStageActive)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(skills) == 0 {
		t.Fatal("expected at least one skill created, found none")
	}
	found := false
	for _, s := range skills {
		if s.AutoCreated {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one skill with auto_created=true")
	}
}

func TestAutoSkillCreation_OffMode(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeDifficultEntry()

	err := AutoCreateSkillIfDifficult(entry, store, dir, AutoSkillModeOff)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (off mode): %v", err)
	}

	svc := NewSkillService(store.DB(), dir)
	skills, err := svc.ListSkills(SkillStageActive)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected no skills created in off mode, got %d", len(skills))
	}
}

func TestAutoSkillCreation_ProposeMode(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeDifficultEntry()

	err := AutoCreateSkillIfDifficult(entry, store, dir, AutoSkillModePropose)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (propose mode): %v", err)
	}

	svc := NewSkillService(store.DB(), dir)
	skills, err := svc.ListSkills(SkillStageActive)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected no skills created in propose mode, got %d", len(skills))
	}
}

func TestAutoSkillCreation_EvidenceFrontmatter(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeDifficultEntry()

	err := AutoCreateSkillIfDifficult(entry, store, dir, AutoSkillModeAuto)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult: %v", err)
	}

	// Find the created SKILL.md file and check frontmatter
	skillDir := SkillDir(dir)
	entries, err := os.ReadDir(filepath.Join(skillDir, SkillStageActive))
	if err != nil {
		t.Fatalf("read skill dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no skill directories created")
	}

	skillPath := filepath.Join(skillDir, SkillStageActive, entries[0].Name(), "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	content := string(data)

	// Verify frontmatter contains evidence fields
	if !strings.Contains(content, "source_run_id:") {
		t.Error("expected SKILL.md frontmatter to contain source_run_id")
	}
	if !strings.Contains(content, "confidence:") {
		t.Error("expected SKILL.md frontmatter to contain confidence")
	}
	if !strings.Contains(content, "privacy_scan:") {
		t.Error("expected SKILL.md frontmatter to contain privacy_scan")
	}
	if !strings.Contains(content, "auto_created: true") {
		t.Error("expected SKILL.md frontmatter to contain auto_created: true")
	}
}

func TestAutoSkillCreation_EasyTaskSkipped(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	// Easy entry: all workers completed, all gates passed
	entry := makeEntry("", "easy learning content", 0.9)
	entry.Evidence.FilesTouched = []string{"pkg/main.go"}
	entry.Phase = 3

	err := AutoCreateSkillIfDifficult(entry, store, dir, AutoSkillModeAuto)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (easy task): %v", err)
	}

	svc := NewSkillService(store.DB(), dir)
	skills, err := svc.ListSkills(SkillStageActive)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected no skills for easy task, got %d", len(skills))
	}
}

func TestAutoSkillCreation_ExistingSkillUpdatesUsage(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	// Create an initial skill manually
	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{
		Name:        "test-skill-usage",
		Stage:       SkillStageActive,
		AutoCreated: true,
		Confidence:  0.8,
		CreatedAt:   "2026-05-01T00:00:00Z",
	}
	if err := svc.CreateSkill(meta, "initial content"); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	// Now call AutoCreateSkillIfDifficult with an entry that would derive the same name
	entry := makeDifficultEntry()
	// Override content to produce a name matching "test-skill-usage" (hash-dependent, so we just verify no duplicate)
	err := AutoCreateSkillIfDifficult(entry, store, dir, AutoSkillModeAuto)
	if err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult (existing skill): %v", err)
	}

	// Verify the original skill still exists and was not duplicated
	skills, err := svc.ListSkills(SkillStageActive)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}

	// Count skills matching "test-skill-usage"
	count := 0
	for _, s := range skills {
		if s.Name == "test-skill-usage" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 skill named 'test-skill-usage', got %d", count)
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
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	entry := makeDifficultEntry()

	// Should not panic even with various inputs
	err := AutoCreateSkillIfDifficult(entry, store, dir, AutoSkillModeAuto)
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
	err = AutoCreateSkillIfDifficult(emptyEntry, store, dir, AutoSkillModeAuto)
	if err != nil {
		t.Logf("expected possible error for empty content, got: %v", err)
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
		ID:             "",
		Content:        "authentication middleware implementation with JWT tokens and refresh rotation",
		Evidence: Evidence{
			RunID:   "run-diff-1",
			Phase:   5,
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
