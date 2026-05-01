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

	// Verify session.json was created
	sessionPath := filepath.Join(s.BasePath(), "session.json")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("session.json not created: %v", err)
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

// TestInitCeremonyRevise verifies that choosing "Revise" re-runs research
// with a new goal and creates colony with revised charter.
func TestInitCeremonyRevise(t *testing.T) {
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

	// Mock stdin: choose "2" (Revise), type new goal, then choose "1" (Proceed)
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
		w.WriteString("2\n")           // Revise
		w.WriteString("Revised goal\n") // New goal
		w.WriteString("1\n")           // Proceed
		w.Close()
	}()

	rootCmd.SetArgs([]string{"init-ceremony", "Original goal", "--target", projectRoot})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	// Verify COLONY_STATE.json was created with the REVISED charter
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
	if state.Charter.Intent != "Revised goal" {
		t.Errorf("Charter.Intent = %q, want 'Revised goal'", state.Charter.Intent)
	}
	if ptrStr(state.Goal) != "Revised goal" {
		t.Errorf("Goal = %q, want 'Revised goal'", ptrStr(state.Goal))
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
