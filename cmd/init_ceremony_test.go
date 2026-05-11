package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestInitCeremonyRegistered verifies the init-ceremony command is registered on rootCmd.
func TestInitCeremonyRegistered(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"init-ceremony", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected init-ceremony --help to succeed, got error: %v", err)
	}
}

// TestInitCeremonyProceed verifies that choosing "Proceed" creates COLONY_STATE.json
// with a Charter sub-object and session.json.
func TestInitCeremonyProceed(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	// Mock stdin: choose "1" (Proceed)
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Reset cached reader and set stdinReader for test mode
	resetCachedStdinReader()
	origStdinReader := stdinReader
	stdinReader = func() *bufio.Reader { return bufio.NewReader(r) }
	defer func() { stdinReader = origStdinReader; resetCachedStdinReader() }()

	go func() {
		w.WriteString("1\n")
		w.Close()
	}()

	rootCmd.SetArgs([]string{"init-ceremony", "Build something", "--target", projectRoot})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	// Verify COLONY_STATE.json was created with Charter
	statePath := filepath.Join(s.BasePath(), "COLONY_STATE.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("COLONY_STATE.json not created: %v", err)
	}

	var state colony.ColonyState
	data, _ := os.ReadFile(statePath)
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("failed to parse COLONY_STATE.json: %v", err)
	}

	if state.Charter == nil {
		t.Fatal("Charter is nil, want non-nil Charter sub-object")
	}
	if state.Charter.Intent != "Build something" {
		t.Errorf("Charter.Intent = %q, want 'Build something'", state.Charter.Intent)
	}
	if state.Charter.TechStack == "" {
		t.Error("Charter.TechStack is empty, want non-empty")
	}
	if state.ColonyMode != colony.ColonyModeColony {
		t.Errorf("ColonyMode = %q, want %q", state.ColonyMode, colony.ColonyModeColony)
	}

	// Verify session.json was created
	sessionPath := filepath.Join(s.BasePath(), "session.json")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("session.json not created: %v", err)
	}
}

func TestInitCeremonyOrchestratorSelection(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	resetCachedStdinReader()
	origStdinReader := stdinReader
	stdinReader = func() *bufio.Reader { return bufio.NewReader(r) }
	defer func() { stdinReader = origStdinReader; resetCachedStdinReader() }()

	go func() {
		w.WriteString("1\n")
		w.Close()
	}()

	rootCmd.SetArgs([]string{"init-ceremony", "--colony-mode", "orchestrator", "Build orchestrator mode", "--target", projectRoot})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	statePath := filepath.Join(s.BasePath(), "COLONY_STATE.json")
	var state colony.ColonyState
	data, _ := os.ReadFile(statePath)
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("failed to parse COLONY_STATE.json: %v", err)
	}
	if state.ColonyMode != colony.ColonyModeOrchestrator {
		t.Fatalf("ColonyMode = %q, want %q", state.ColonyMode, colony.ColonyModeOrchestrator)
	}
}

// TestInitCeremonyCancel verifies that choosing "Cancel" creates no artifacts.
func TestInitCeremonyCancel(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	// Mock stdin: choose "3" (Cancel)
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Set stdinReader so the ceremony skips the TTY check in test mode
	resetCachedStdinReader()
	origStdinReader := stdinReader
	stdinReader = func() *bufio.Reader { return bufio.NewReader(r) }
	defer func() { stdinReader = origStdinReader; resetCachedStdinReader() }()

	go func() {
		w.WriteString("3\n")
		w.Close()
	}()

	rootCmd.SetArgs([]string{"init-ceremony", "Build something", "--target", projectRoot})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	// Verify NO artifacts were created
	statePath := filepath.Join(s.BasePath(), "COLONY_STATE.json")
	if _, err := os.Stat(statePath); err == nil {
		t.Fatal("COLONY_STATE.json should NOT exist after cancel")
	}

	sessionPath := filepath.Join(s.BasePath(), "session.json")
	if _, err := os.Stat(sessionPath); err == nil {
		t.Fatal("session.json should NOT exist after cancel")
	}

	pheromonesPath := filepath.Join(s.BasePath(), "pheromones.json")
	if _, err := os.Stat(pheromonesPath); err == nil {
		t.Fatal("pheromones.json should NOT exist after cancel")
	}
}

// TestInitCeremonyRejectThenApprove verifies that rejecting the brief
// prevents colony creation (the old "Revise" flow is replaced by the
// brief approval flow where "Reject" returns to goal prompt).
func TestInitCeremonyRejectThenApprove(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	// Mock stdin: choose "3" (Reject) -- no more input, ceremony exits
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	resetCachedStdinReader()
	origStdinReader := stdinReader
	stdinReader = func() *bufio.Reader { return bufio.NewReader(r) }
	defer func() { stdinReader = origStdinReader; resetCachedStdinReader() }()

	go func() {
		w.WriteString("3\n") // Reject brief
		w.Close()
	}()

	rootCmd.SetArgs([]string{"init-ceremony", "Original goal", "--target", projectRoot})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	// Verify NO colony was created (reject prevents creation)
	statePath := filepath.Join(s.BasePath(), "COLONY_STATE.json")
	if _, err := os.Stat(statePath); err == nil {
		t.Fatal("COLONY_STATE.json should NOT exist after brief rejection")
	}
}

// TestSynthesizeLaunchBrief verifies that synthesizeLaunchBrief produces
// markdown with all 6 required sections: Goal, Scope, Risks, Tech Stack, Dependencies, Success Criteria.
func TestSynthesizeLaunchBrief(t *testing.T) {
	goal := "Build a REST API"
	charter := &colony.Charter{
		Intent:      "Build a REST API",
		Vision:      "A clean API server",
		Governance:  "golangci-lint",
		Goals:       "Ship v1.0 with auth endpoints",
		TechStack:   "Go, PostgreSQL",
		KeyRisks:    "No tests yet",
		Constraints: "Follow lint rules",
	}
	researchData := ceremonyResearchData{
		TechStackDetail: []techStackDetail{
			{Language: "go", SourceFile: "go.mod", Deps: []depEntry{{Name: "cobra"}}},
		},
		DirClassification: dirClassification{Type: "standard"},
		ColonyContextSummary: colonyContextSummary{
			DetectedType: "go-project",
			Languages:    []string{"go"},
		},
	}

	brief := synthesizeLaunchBrief(goal, charter, researchData)

	requiredSections := []string{
		"# Colony Launch Brief",
		"## Goal",
		"## Scope",
		"## Risks",
		"## Tech Stack",
		"## Dependencies",
		"## Success Criteria",
	}
	for _, section := range requiredSections {
		if !strings.Contains(brief, section) {
			t.Errorf("synthesizeLaunchBrief missing section %q", section)
		}
	}

	// Goal section should contain the colony goal
	if !strings.Contains(brief, "Build a REST API") {
		t.Error("synthesizeLaunchBrief missing goal content")
	}

	// Tech Stack section should include detected tech stack
	if !strings.Contains(brief, "Go") {
		t.Error("synthesizeLaunchBrief missing detected language Go")
	}
	if !strings.Contains(brief, "cobra") {
		t.Error("synthesizeLaunchBrief missing detected dependency cobra")
	}
}

// TestSynthesizeLaunchBriefIncludesTechStackFromCharter verifies tech stack
// detected from research data appears in the brief.
func TestSynthesizeLaunchBriefIncludesTechStackFromCharter(t *testing.T) {
	charter := &colony.Charter{
		Intent:    "Build something",
		TechStack: "Go, Cobra, PostgreSQL",
	}
	researchData := ceremonyResearchData{
		TechStackDetail: []techStackDetail{
			{Language: "go", SourceFile: "go.mod", Deps: []depEntry{{Name: "github.com/lib/pq"}}},
		},
	}

	brief := synthesizeLaunchBrief("Build something", charter, researchData)

	if !strings.Contains(brief, "PostgreSQL") {
		t.Error("synthesizeLaunchBrief should include PostgreSQL from charter tech stack")
	}
	if !strings.Contains(brief, "github.com/lib/pq") {
		t.Error("synthesizeLaunchBrief should include dependency from research data")
	}
}

// TestSynthesizeLaunchBriefEmptyData verifies that sections with no data
// show "To be determined" rather than being empty.
func TestSynthesizeLaunchBriefEmptyData(t *testing.T) {
	charter := &colony.Charter{
		Intent: "Some goal",
	}
	researchData := ceremonyResearchData{}

	brief := synthesizeLaunchBrief("Some goal", charter, researchData)

	// Empty sections should show TBD
	if !strings.Contains(brief, "To be determined") {
		t.Error("synthesizeLaunchBrief should show 'To be determined' for empty sections")
	}
}

// TestSynthesizeLaunchBriefWithRisks verifies that KeyRisks from the charter
// appear in the Risks section.
func TestSynthesizeLaunchBriefWithRisks(t *testing.T) {
	charter := &colony.Charter{
		Intent:      "Risky project",
		KeyRisks:    "No test coverage, tight deadline",
		Constraints: "Must ship in 2 weeks",
	}
	researchData := ceremonyResearchData{}

	brief := synthesizeLaunchBrief("Risky project", charter, researchData)

	if !strings.Contains(brief, "No test coverage") {
		t.Error("synthesizeLaunchBrief missing KeyRisks content in Risks section")
	}
	if !strings.Contains(brief, "tight deadline") {
		t.Error("synthesizeLaunchBrief missing KeyRisks content in Risks section")
	}
}

// TestInitCeremonyApproveBrief verifies that approving the brief creates the colony.
func TestInitCeremonyApproveBrief(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	// Mock stdin: choose "1" (Approve brief)
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	resetCachedStdinReader()
	origStdinReader := stdinReader
	stdinReader = func() *bufio.Reader { return bufio.NewReader(r) }
	defer func() { stdinReader = origStdinReader; resetCachedStdinReader() }()

	go func() {
		w.WriteString("1\n") // Approve brief
		w.Close()
	}()

	rootCmd.SetArgs([]string{"init-ceremony", "Build something", "--target", projectRoot})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	// Verify COLONY_STATE.json was created
	statePath := filepath.Join(s.BasePath(), "COLONY_STATE.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("COLONY_STATE.json not created after brief approval: %v", err)
	}
}

// TestInitCeremonyRejectBrief verifies that rejecting the brief does NOT create the colony.
func TestInitCeremonyRejectBrief(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	// Mock stdin: choose "3" (Reject brief)
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	resetCachedStdinReader()
	origStdinReader := stdinReader
	stdinReader = func() *bufio.Reader { return bufio.NewReader(r) }
	defer func() { stdinReader = origStdinReader; resetCachedStdinReader() }()

	go func() {
		w.WriteString("3\n") // Reject brief
		w.Close()
	}()

	rootCmd.SetArgs([]string{"init-ceremony", "Build something", "--target", projectRoot})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	// Verify NO colony was created
	statePath := filepath.Join(s.BasePath(), "COLONY_STATE.json")
	if _, err := os.Stat(statePath); err == nil {
		t.Fatal("COLONY_STATE.json should NOT exist after brief rejection")
	}
}

// TestRenderCharterDisplay verifies that renderCharterDisplay produces output
// containing all 7 charter section names.
func TestRenderCharterDisplay(t *testing.T) {
	ch := colony.Charter{
		Intent:      "Build great software",
		Vision:      "A world-class go project",
		Governance:  "Linting: golangci-lint. CI: GitHub Actions",
		Goals:       "Goal: Build great software. Focus on quality.",
		TechStack:   "Languages: go. Frameworks/Tools: cobra",
		KeyRisks:    "No test framework detected -- regression risk",
		Constraints: "Follow golangci-lint rules",
	}

	output := renderCharterDisplay(ch)

	labels := []string{"Intent:", "Vision:", "Governance:", "Goals:", "Tech Stack:", "Key Risks:", "Constraints:"}
	for _, label := range labels {
		if !strings.Contains(output, label) {
			t.Errorf("renderCharterDisplay output missing label %q", label)
		}
	}

	// Verify content appears
	if !strings.Contains(output, "Build great software") {
		t.Error("renderCharterDisplay output missing Intent content")
	}
	if !strings.Contains(output, "A world-class go project") {
		t.Error("renderCharterDisplay output missing Vision content")
	}
}
