package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// --- Unit tests for confidence targeting (Task 1) ---

func TestDefaultOracleTargetConfidenceIs95(t *testing.T) {
	if defaultOracleTargetConfidence != 95 {
		t.Errorf("defaultOracleTargetConfidence = %d, want 95", defaultOracleTargetConfidence)
	}
}

func TestResolveOracleDepthRestoresMarathonCaps(t *testing.T) {
	tests := []struct {
		depth string
		want  int
	}{
		{"quick", 5},
		{"balanced", 15},
		{"standard", 15},
		{"deep", 30},
		{"exhaustive", 50},
		{"marathon", 50},
	}
	for _, tt := range tests {
		t.Run(tt.depth, func(t *testing.T) {
			got := resolveOracleDepth(tt.depth)
			if got.MaxIterations != tt.want {
				t.Fatalf("resolveOracleDepth(%q).MaxIterations = %d, want %d", tt.depth, got.MaxIterations, tt.want)
			}
		})
	}
}

func TestParseOracleMaxIterations(t *testing.T) {
	got, err := parseOracleMaxIterations("50")
	if err != nil {
		t.Fatalf("parseOracleMaxIterations returned error: %v", err)
	}
	if got != 50 {
		t.Fatalf("parseOracleMaxIterations = %d, want 50", got)
	}
	for _, value := range []string{"0", "51", "abc"} {
		if _, err := parseOracleMaxIterations(value); err == nil {
			t.Fatalf("parseOracleMaxIterations(%q) returned nil error", value)
		}
	}
}

func TestDefaultOracleInvokerAvoidsOpenCodeInsideOpenCodeAgent(t *testing.T) {
	t.Setenv("OPENCODE_AGENT", "1")
	t.Setenv("AETHER_CODEX_REAL_DISPATCH", "1")
	t.Setenv("AETHER_CODEX_PATH", "go")
	t.Setenv("AETHER_CLAUDE_PATH", "missing-claude-binary-12345")
	t.Setenv("AETHER_ORACLE_ALLOW_OPENCODE", "")
	t.Setenv("AETHER_WORKER_PLATFORM", "")

	invoker := newDefaultOracleWorkerInvoker()
	if got := codex.PlatformFromInvoker(invoker); got != codex.PlatformCodex {
		t.Fatalf("PlatformFromInvoker() = %s, want %s", got, codex.PlatformCodex)
	}
}

func TestOracleBackgroundEnvMarksDetachedController(t *testing.T) {
	env := oracleBackgroundEnv([]string{
		"OPENCODE_AGENT=1",
		"CLAUDE_CODE_SIMPLE=1",
		"AETHER_AGENT_DELEGATE=1",
		"AETHER_OUTPUT_MODE=visual",
		"KEEP_ME=1",
	})
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{"OPENCODE_AGENT=1", "CLAUDE_CODE_SIMPLE=1", "AETHER_AGENT_DELEGATE=1", "AETHER_OUTPUT_MODE=visual"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("oracleBackgroundEnv leaked %q in:\n%s", forbidden, joined)
		}
	}
	for _, want := range []string{"AETHER_OUTPUT_MODE=json", "AETHER_ORACLE_AVOID_OPENCODE=1", "KEEP_ME=1"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("oracleBackgroundEnv missing %q in:\n%s", want, joined)
		}
	}
}

func TestOracleRunLoopModeResumesInitializedWorkspace(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-run-loop\n\ngo 1.24\n"), 0644); err != nil {
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
	state := oracleStateFile{
		Version:          "1.1",
		Topic:            "run-loop topic",
		Scope:            "repo",
		Template:         "custom",
		Phase:            "survey",
		Iteration:        0,
		MaxIterations:    1,
		TargetConfidence: 60,
		Status:           "active",
		Strategy:         defaultOracleStrategy,
		Platform:         "opencode",
	}
	plan := oraclePlanFile{
		Version: "1.1",
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{{
			ID:          "q1",
			Text:        "What should run-loop resume investigate?",
			Status:      "open",
			KeyFindings: []oracleFinding{},
		}},
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	result, err := runOracleCompatibility(root, []string{"run-loop"}, "", "")
	if err != nil {
		t.Fatalf("run-loop returned error: %v", err)
	}
	if result["mode"] != "run" {
		t.Fatalf("mode = %v, want run", result["mode"])
	}
	if result["status"] != "complete" {
		t.Fatalf("status = %v, want complete", result["status"])
	}
	if result["iterations_run"] != 1 {
		t.Fatalf("iterations_run = %v, want 1", result["iterations_run"])
	}
}

func TestResolveOracleTemplate(t *testing.T) {
	tests := []struct {
		name      string
		topic     string
		requested string
		want      string
	}{
		{"explicit prd", "scope a new colony", "prd", "prd"},
		{"requirements alias", "write product requirements", "requirements", "prd"},
		{"auto prd", "turn this into user stories and acceptance criteria", "auto", "prd"},
		{"auto bug", "root cause the failing update command", "", "bug-investigation"},
		{"auto tech eval", "compare sqlite vs postgres", "", "tech-eval"},
		{"auto architecture", "design the reference injection architecture", "", "architecture-review"},
		{"auto fallback", "research a general topic", "", "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveOracleTemplate(tt.topic, tt.requested)
			if err != nil {
				t.Fatalf("resolveOracleTemplate returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveOracleTemplate(%q, %q) = %q, want %q", tt.topic, tt.requested, got, tt.want)
			}
		})
	}
}

func TestOracleConfidenceTargetFlagDefaultUsesDepthPreset(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/conf-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	// Run with balanced depth (no --confidence-target) -- should use depth preset of 85
	rootCmd.SetArgs([]string{"oracle", "--depth", "balanced", "test topic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle with balanced depth returned error: %v", err)
	}

	statePath := filepath.Join(root, ".aether", "oracle", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read oracle state: %v", err)
	}
	var state oracleStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse oracle state: %v", err)
	}
	if state.TargetConfidence != 85 {
		t.Errorf("TargetConfidence = %d, want 85 (balanced depth preset)", state.TargetConfidence)
	}
}

func TestOracleConfidenceTargetFlagOverride(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/conf-override\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	// Run with --confidence-target 80 overriding depth preset
	rootCmd.SetArgs([]string{"oracle", "--depth", "deep", "--confidence-target", "80", "test topic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle with --confidence-target 80 returned error: %v", err)
	}

	statePath := filepath.Join(root, ".aether", "oracle", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read oracle state: %v", err)
	}
	var state oracleStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse oracle state: %v", err)
	}
	// Deep preset is 95, but explicit --confidence-target 80 should override it
	if state.TargetConfidence != 80 {
		t.Errorf("TargetConfidence = %d, want 80 (explicit override of deep preset)", state.TargetConfidence)
	}
}

func TestOracleConfidenceTargetRejectsOutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"zero", "0", false},
		{"negative", "-5", false},
		{"one", "1", true},
		{"fifty", "50", true},
		{"hundred", "100", true},
		{"over100", "101", false},
		{"twohundred", "200", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateOracleConfidenceTarget(tt.value)
			if tt.valid && got != "" {
				t.Errorf("validateOracleConfidenceTarget(%q) = %q, want empty string", tt.value, got)
			}
			if !tt.valid && got == "" {
				t.Errorf("validateOracleConfidenceTarget(%q) = %q, want error message", tt.value, got)
			}
		})
	}
}

func TestOracleCommandInvalidConfidenceReturnsRenderedError(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	rootCmd.SetArgs([]string{"oracle", "--confidence-target", "0", "test topic"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected oracle command to return rendered error for invalid confidence target")
	}
	var rendered renderedCommandError
	if !errors.As(err, &rendered) {
		t.Fatalf("expected renderedCommandError, got %T: %v", err, err)
	}
	if rendered.code != 1 {
		t.Fatalf("rendered error code = %d, want 1", rendered.code)
	}
}

func TestOracleConfidenceTargetQuickDepthPreset(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/conf-quick\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	// Run with quick depth (no --confidence-target) -- should use depth preset of 60
	rootCmd.SetArgs([]string{"oracle", "--depth", "quick", "test topic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle with quick depth returned error: %v", err)
	}

	statePath := filepath.Join(root, ".aether", "oracle", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read oracle state: %v", err)
	}
	var state oracleStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse oracle state: %v", err)
	}
	if state.TargetConfidence != 60 {
		t.Errorf("TargetConfidence = %d, want 60 (quick depth preset)", state.TargetConfidence)
	}
}

func TestOracleConfidenceTargetNoDepthUsesBalancedPreset(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/conf-nodepth\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	// Run with no depth and no --confidence-target -- should use balanced preset (85)
	// because balanced is the default depth
	rootCmd.SetArgs([]string{"oracle", "test topic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle with no flags returned error: %v", err)
	}

	statePath := filepath.Join(root, ".aether", "oracle", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read oracle state: %v", err)
	}
	var state oracleStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse oracle state: %v", err)
	}
	// No explicit depth => balanced => TargetConfidence = 85
	if state.TargetConfidence != 85 {
		t.Errorf("TargetConfidence = %d, want 85 (balanced default depth)", state.TargetConfidence)
	}
}

// --- Unit tests for rubric output and non-finalization (Task 2) ---

func TestMapApprovalStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"complete", "approved"},
		{"blocked", "blocked"},
		{"max_iterations_reached", "max_iterations"},
		{"below_target", "below_target"},
		{"stopped", "below_target"},
		{"unknown_status", "below_target"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapApprovalStatus(tt.input)
			if got != tt.expected {
				t.Errorf("mapApprovalStatus(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestHasHardBlocker(t *testing.T) {
	plan := oraclePlanFile{
		Questions: []oracleQuestion{
			{
				ID:     "q1",
				Status: "answered",
				KeyFindings: []oracleFinding{
					{Text: "normal finding"},
				},
			},
			{
				ID:     "q2",
				Status: "open",
				KeyFindings: []oracleFinding{
					{Text: "blocked finding", Blocker: true},
				},
			},
			{
				ID:          "q3",
				Status:      "partial",
				KeyFindings: nil,
			},
		},
	}
	state := oracleStateFile{Iteration: 2}

	if !hasHardBlocker(plan, state) {
		t.Error("hasHardBlocker should return true when a finding has Blocker=true")
	}

	// No blockers
	planClean := oraclePlanFile{
		Questions: []oracleQuestion{
			{ID: "q1", Status: "answered", KeyFindings: []oracleFinding{{Text: "normal"}}},
			{ID: "q2", Status: "partial"},
		},
	}
	if hasHardBlocker(planClean, state) {
		t.Error("hasHardBlocker should return false when no findings have Blocker=true")
	}
}

func TestBuildOracleRubric(t *testing.T) {
	plan := oraclePlanFile{
		Sources: map[string]oracleSource{
			"S1": {URL: "cmd/test.go", Title: "test file", Type: "codebase"},
		},
		Questions: []oracleQuestion{
			{
				ID:         "q1",
				Text:       "What is X?",
				Status:     "answered",
				Confidence: 90,
				KeyFindings: []oracleFinding{
					{Text: "Finding about X", SourceIDs: []string{"S1"}, Iteration: 1},
				},
			},
			{
				ID:         "q2",
				Text:       "What is Y?",
				Status:     "partial",
				Confidence: 40,
				KeyFindings: []oracleFinding{
					{Text: "Partial finding about Y", SourceIDs: []string{"S1"}, Iteration: 2},
				},
			},
			{
				ID:          "q3",
				Text:        "What is Z?",
				Status:      "open",
				Confidence:  0,
				KeyFindings: nil,
			},
		},
	}
	state := oracleStateFile{Iteration: 2}

	rubric := buildOracleRubric(plan, state)
	if len(rubric) != 3 {
		t.Fatalf("buildOracleRubric returned %d entries, want 3", len(rubric))
	}

	// Check q1 entry
	q1 := rubric[0]
	if q1["question_id"] != "q1" {
		t.Errorf("rubric[0] question_id = %v, want q1", q1["question_id"])
	}
	if q1["confidence"] != 90 {
		t.Errorf("rubric[0] confidence = %v, want 90", q1["confidence"])
	}
}

func TestIdentifyGaps(t *testing.T) {
	plan := oraclePlanFile{
		Questions: []oracleQuestion{
			{ID: "q1", Text: "What is X?", Status: "answered", Confidence: 90},
			{ID: "q2", Text: "What is Y?", Status: "partial", Confidence: 40},
			{ID: "q3", Text: "What is Z?", Status: "open", Confidence: 0},
		},
	}
	state := oracleStateFile{
		TargetConfidence: 80,
		OpenGaps:         []string{"unresolved area A"},
	}

	gaps := identifyGaps(plan, state)
	if len(gaps) == 0 {
		t.Fatal("identifyGaps returned empty, expected gaps for low-confidence questions")
	}

	// Should include gaps for q2 (partial, below target) and q3 (open)
	hasQ2Gap := false
	hasQ3Gap := false
	hasOpenGap := false
	for _, g := range gaps {
		if g.QuestionID == "q2" {
			hasQ2Gap = true
		}
		if g.QuestionID == "q3" {
			hasQ3Gap = true
		}
		if g.QuestionID == "open_gap" {
			hasOpenGap = true
		}
	}
	if !hasQ2Gap {
		t.Error("identifyGaps should include q2 (partial, below target)")
	}
	if !hasQ3Gap {
		t.Error("identifyGaps should include q3 (open)")
	}
	if !hasOpenGap {
		t.Error("identifyGaps should include open gaps from state")
	}
}

func TestCollectEvidence(t *testing.T) {
	plan := oraclePlanFile{
		Sources: map[string]oracleSource{
			"S1": {URL: "cmd/test.go", Title: "test file", Type: "codebase"},
			"S2": {URL: "https://example.com", Title: "example", Type: "official"},
		},
		Questions: []oracleQuestion{
			{
				ID:     "q1",
				Status: "answered",
				KeyFindings: []oracleFinding{
					{Text: "Finding A", SourceIDs: []string{"S1"}, Iteration: 1},
					{Text: "Finding B", SourceIDs: []string{"S2"}, Iteration: 2},
				},
			},
			{
				ID:          "q2",
				Status:      "open",
				KeyFindings: nil,
			},
		},
	}
	state := oracleStateFile{}

	evidence := collectEvidence(plan, state)
	if len(evidence) != 1 {
		t.Fatalf("collectEvidence returned %d entries, want 1 (only q1 has findings)", len(evidence))
	}

	q1Evidence := evidence[0]
	if q1Evidence.QuestionID != "q1" {
		t.Errorf("evidence[0] question_id = %v, want q1", q1Evidence.QuestionID)
	}
	if q1Evidence.SummaryCount != 2 {
		t.Errorf("evidence[0] summary_count = %v, want 2", q1Evidence.SummaryCount)
	}
}

func TestFinalizeOracleLoopRubricOutput(t *testing.T) {
	// Set up oracle workspace
	tmpDir := t.TempDir()
	paths := oracleWorkspacePaths(tmpDir)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensureOracleWorkspace: %v", err)
	}

	plan := oraclePlanFile{
		Version: "1.1",
		Sources: map[string]oracleSource{
			"S1": {URL: "cmd/test.go", Title: "test file", Type: "codebase"},
		},
		Questions: []oracleQuestion{
			{
				ID:         "q1",
				Text:       "What is X?",
				Status:     "answered",
				Confidence: 90,
				KeyFindings: []oracleFinding{
					{Text: "Finding about X", SourceIDs: []string{"S1"}, Iteration: 1},
				},
				IterationsTouched: []int{1},
			},
		},
	}
	state := oracleStateFile{
		Version:           "1.1",
		Topic:             "Test research",
		TargetConfidence:  80,
		OverallConfidence: 90,
		Iteration:         1,
		MaxIterations:     4,
		Status:            "active",
		Platform:          "codex",
	}

	result, err := finalizeOracleLoop(paths, state, plan, "go", []string{"go"}, []string{}, 1, "complete", "", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop returned error: %v", err)
	}

	// Verify rubric fields in output
	if _, ok := result["rubric"]; !ok {
		t.Error("finalizeOracleLoop output missing 'rubric' field")
	}
	if _, ok := result["gaps"]; !ok {
		t.Error("finalizeOracleLoop output missing 'gaps' field")
	}
	if _, ok := result["evidence"]; !ok {
		t.Error("finalizeOracleLoop output missing 'evidence' field")
	}
	if _, ok := result["approval_status"]; !ok {
		t.Error("finalizeOracleLoop output missing 'approval_status' field")
	}
	if _, ok := result["target_confidence"]; !ok {
		t.Error("finalizeOracleLoop output missing 'target_confidence' field")
	}
	if _, ok := result["final_confidence"]; !ok {
		t.Error("finalizeOracleLoop output missing 'final_confidence' field")
	}

	// Verify approval_status is "approved" for complete status
	if result["approval_status"] != "approved" {
		t.Errorf("approval_status = %v, want 'approved' for complete status", result["approval_status"])
	}

	// Verify target and final confidence
	if result["target_confidence"] != 80 {
		t.Errorf("target_confidence = %v, want 80", result["target_confidence"])
	}
	if result["final_confidence"] != 90 {
		t.Errorf("final_confidence = %v, want 90", result["final_confidence"])
	}
}

func TestFinalizeOracleLoopBelowTarget(t *testing.T) {
	tmpDir := t.TempDir()
	paths := oracleWorkspacePaths(tmpDir)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensureOracleWorkspace: %v", err)
	}

	plan := oraclePlanFile{
		Version: "1.1",
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{ID: "q1", Text: "Q1", Status: "partial", Confidence: 50, IterationsTouched: []int{1}},
		},
	}
	state := oracleStateFile{
		Version:           "1.1",
		Topic:             "Test research",
		TargetConfidence:  90,
		OverallConfidence: 50,
		Iteration:         4,
		MaxIterations:     4,
		Status:            "active",
		Platform:          "codex",
	}

	// max_iterations_reached but confidence below target
	result, err := finalizeOracleLoop(paths, state, plan, "go", []string{"go"}, []string{}, 4, "max_iterations_reached", "max_iterations_reached", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop returned error: %v", err)
	}

	if result["approval_status"] != "max_iterations" {
		t.Errorf("approval_status = %v, want 'max_iterations'", result["approval_status"])
	}
}

func TestFinalizeOracleLoopBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	paths := oracleWorkspacePaths(tmpDir)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensureOracleWorkspace: %v", err)
	}

	plan := oraclePlanFile{
		Version: "1.1",
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{
				ID:     "q1",
				Text:   "Q1",
				Status: "open",
				KeyFindings: []oracleFinding{
					{Text: "Hard blocker finding", Blocker: true},
				},
			},
		},
	}
	state := oracleStateFile{
		Version:           "1.1",
		Topic:             "Test research",
		TargetConfidence:  90,
		OverallConfidence: 30,
		Iteration:         2,
		MaxIterations:     4,
		Status:            "active",
		Platform:          "codex",
	}

	result, err := finalizeOracleLoop(paths, state, plan, "go", []string{"go"}, []string{}, 2, "blocked", "worker_blocked", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop returned error: %v", err)
	}

	if result["approval_status"] != "blocked" {
		t.Errorf("approval_status = %v, want 'blocked'", result["approval_status"])
	}
}

func TestOracleReadyForCompletion(t *testing.T) {
	tests := []struct {
		name      string
		overall   int
		target    int
		questions []oracleQuestion
		expected  bool
	}{
		{"meets target", 95, 95, nil, true},
		{"exceeds target", 98, 95, nil, true},
		{"below target", 94, 95, nil, false},
		{"below target at half with all answered", 47, 95, []oracleQuestion{
			{Status: "answered"},
			{Status: "answered"},
		}, false},
		{"below target by one", 94, 95, nil, false},
		{"zero confidence", 0, 95, nil, false},
		{"zero target", 50, 0, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := oraclePlanFile{Questions: tt.questions}
			state := oracleStateFile{OverallConfidence: tt.overall, TargetConfidence: tt.target}
			got := oracleReadyForCompletion(plan, state)
			if got != tt.expected {
				t.Errorf("oracleReadyForCompletion() = %v, want %v", got, tt.expected)
			}
		})
	}
}
