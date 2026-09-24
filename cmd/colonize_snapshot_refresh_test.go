package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// gitColonizeSnapshotRefresh runs a git command inside root, failing the test
// on error and returning trimmed stdout. Mirrors gitSurveyFreshness199's own
// shape so this file reads as the same kind of fixture, without importing a
// _test.go helper across build units.
func gitColonizeSnapshotRefresh(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// initColonizeSnapshotRefreshRepo creates a real, empty git repository with
// one initial commit -- the exact revision the seeded saved map below is
// pinned to before this test moves HEAD further.
func initColonizeSnapshotRefreshRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitColonizeSnapshotRefresh(t, root, "init")
	gitColonizeSnapshotRefresh(t, root, "config", "user.email", "test@aether.invalid")
	gitColonizeSnapshotRefresh(t, root, "config", "user.name", "Aether Test")
	gitColonizeSnapshotRefresh(t, root, "commit", "--allow-empty", "-m", "initial")
	return root
}

// TestColonizeRefreshesTheSavedMapsRevision is the local reproduction of the
// fourth proximate cause the third real practice-project walk found
// (WINDOWS.md row 53, 208-JOURNEY-RUN.md's "Third Real Run"): a survey that
// finishes and saves successfully still leaves the saved map's own record
// file naming the revision it was seeded with, not the repository's real
// current state.
//
// It drives the runtime's own path -- a plan-only colonize manifest, a
// completion packet carrying that manifest and the surveyor results, then
// the finalize step -- never a direct call to publishTerritorySnapshot,
// which would prove nothing about the path the rehearsal actually takes.
func TestColonizeRefreshesTheSavedMapsRevision(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	root := initColonizeSnapshotRefreshRepo(t)
	dataDir := filepath.Join(root, ".aether", "data")
	bindCommandTestRepositoryAt(t, root)
	withWorkingDir(t, root)

	// A software project, so the surveyor roster has something to find.
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write fixture source file: %v", err)
	}

	goal := "Survey with a stale saved map"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	// Seed a genuinely valid saved map, pinned to the repository's revision
	// at this moment -- the same shape journeySeedStaleSurvey produces for
	// the practice project's own out-of-date-code-map trap.
	writeFreshTerritorySnapshot199(t, root, time.Now().UTC().Add(-time.Hour))

	// Make further commits so the recorded revision is genuinely older than
	// the repository's real current state -- exactly what
	// scripts/build-messy-practice-project.sh's own follow-on commits do to
	// the practice project after journeySeedStaleSurvey runs.
	for i := 0; i < 3; i++ {
		gitColonizeSnapshotRefresh(t, root, "commit", "--allow-empty", "-m", fmt.Sprintf("further commit %d", i))
	}

	// Nobody is here to answer -- the exact condition the rehearsal's own
	// automated `-p` chain runs under, and the condition that made the third
	// real walk's `aether host colonize` recover the existing-survey refusal
	// silently instead of asking (208-11's self-recovery, exercised live in
	// 208-JOURNEY-RUN.md's "Third Real Run").
	t.Setenv("AETHER_UNATTENDED", "1")

	// Drive the plan-only manifest builder -- the same function
	// `aether colonize --plan-only` (what the rehearsal's survey step
	// reaches through `aether host colonize`) calls.
	planResult, err := runCodexColonizePlanOnly(root, codexColonizeOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("runCodexColonizePlanOnly: %v", err)
	}
	manifest, ok := planResult["colonize_manifest"].(codexColonizeManifest)
	if !ok {
		t.Fatalf("colonize_manifest has unexpected type %T", planResult["colonize_manifest"])
	}
	if !manifest.ForceResurvey {
		t.Fatalf("expected the plan-only manifest to carry ForceResurvey=true from unattended self-recovery, got %#v", manifest)
	}

	dispatches := make([]codexSurveyorDispatch, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		dispatch.Status = "completed"
		dispatch.Summary = "survey complete"
		dispatches = append(dispatches, dispatch)
	}

	completionPath := filepath.Join(t.TempDir(), "colonize-completion.json")
	completionData, err := json.Marshal(map[string]interface{}{
		"colonize_manifest": manifest,
		"dispatches":        dispatches,
	})
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	// Now run the finalize step -- the same command
	// `aether colonize-finalize --completion-file <file>` the rehearsal's own
	// wrapper runs after surveyors complete.
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_UNATTENDED", "1")
	stdout = &bytes.Buffer{}
	var errBuf bytes.Buffer
	stderr = &errBuf
	rootCmd.SetArgs([]string{"colonize-finalize", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize-finalize returned error: %v; stderr=%s", err, errBuf.String())
	}
	if strings.TrimSpace(stdout.(*bytes.Buffer).String()) == "" {
		t.Fatalf("colonize-finalize wrote no stdout; stderr=%s", errBuf.String())
	}

	// The saved map's own record file, read the same way the runtime and the
	// journey check both read it -- never a value carried over from earlier
	// in this test.
	snapPath := filepath.Join(root, filepath.FromSlash(territorySnapshotRelativePath))
	snapData, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatalf("read %s: %v", snapPath, err)
	}
	var snap territorySnapshotMetadata
	if err := json.Unmarshal(snapData, &snap); err != nil {
		t.Fatalf("%s does not parse: %v", snapPath, err)
	}

	// The repository's real current revision, asked of git directly at
	// assertion time -- never typed as a literal.
	head := gitColonizeSnapshotRefresh(t, root, "rev-parse", "HEAD")

	if snap.SourceRevision != head {
		t.Fatalf("saved map's recorded revision is %s, want current HEAD %s -- a survey that finished and saved successfully should leave the saved map recording the project's real, current state", snap.SourceRevision, head)
	}
}
