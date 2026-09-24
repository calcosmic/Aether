package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
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

	// Every dispatch actually writes its declared output -- whatever path
	// the manifest names, live survey dir or candidate dir alike -- and
	// records terminal spawn-tree evidence, so this reproduction drives a
	// genuinely complete survey exactly like a real one, on both the
	// pre-fix (legacy) and post-fix (transactional) lane. Neither addition
	// changes what this test is checking: only the saved map's own
	// recorded revision, asserted below.
	completeColonizeSnapshotRefreshDispatches(t, root, manifest.Dispatches)

	completionPath := filepath.Join(t.TempDir(), "colonize-completion.json")
	completionData, err := json.Marshal(map[string]interface{}{
		"colonize_manifest": manifest,
		"dispatches":        manifest.Dispatches,
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
	snap := readColonizeSnapshotRefreshMetadata(t, root)

	// The repository's real current revision, asked of git directly at
	// assertion time -- never typed as a literal.
	head := gitColonizeSnapshotRefresh(t, root, "rev-parse", "HEAD")

	if snap.SourceRevision != head {
		t.Fatalf("saved map's recorded revision is %s, want current HEAD %s -- a survey that finished and saved successfully should leave the saved map recording the project's real, current state", snap.SourceRevision, head)
	}

	// The chain the whole rehearsal depends on, checked end to end rather
	// than reasoned about: the program's own freshness reader now agrees the
	// saved map is up to date, ...
	freshness := classifySurveyFreshness(root, time.Now().UTC())
	if freshness.Freshness != colony.SurveyFreshnessFresh || freshness.SnapshotID != snap.SnapshotID {
		t.Fatalf("freshness reader = %#v after a successful refresh, want Fresh with snapshot %s", freshness, snap.SnapshotID)
	}

	// ... and a second survey pass over the same, now-fresh project does not
	// fall into the sibling colonize-finalize-existing-survey-found refusal
	// 208-11 taught the program to recover from (that refusal only lives on
	// the legacy, non-transactional finalize lane -- reaching it here would
	// mean this fix's manifest binding stopped taking effect on a repeat
	// survey).
	secondPlanResult, err := runCodexColonizePlanOnly(root, codexColonizeOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("second runCodexColonizePlanOnly: %v", err)
	}
	secondManifest, ok := secondPlanResult["colonize_manifest"].(codexColonizeManifest)
	if !ok {
		t.Fatalf("second colonize_manifest has unexpected type %T", secondPlanResult["colonize_manifest"])
	}
	completeColonizeSnapshotRefreshDispatches(t, root, secondManifest.Dispatches)
	secondCompletionPath := filepath.Join(t.TempDir(), "colonize-completion-second.json")
	secondCompletionData, err := json.Marshal(map[string]interface{}{
		"colonize_manifest": secondManifest,
		"dispatches":        secondManifest.Dispatches,
	})
	if err != nil {
		t.Fatalf("marshal second completion: %v", err)
	}
	if err := os.WriteFile(secondCompletionPath, secondCompletionData, 0644); err != nil {
		t.Fatalf("write second completion: %v", err)
	}
	if _, err := runCodexColonizeFinalize(root, codexExternalColonizeCompletion{
		ColonizeManifest: &secondManifest,
		Dispatches:       secondManifest.Dispatches,
	}); err != nil {
		t.Fatalf("second colonize-finalize hit the sibling existing-survey refusal on a repeat pass: %v", err)
	}
}

// completeColonizeSnapshotRefreshDispatches marks every dispatch completed,
// records terminal spawn-tree evidence for it, and writes real content at
// each of its declared output paths -- whatever the manifest says, live
// survey dir or candidate dir alike -- claiming the write via FilesCreated.
// Mutates dispatches in place.
func completeColonizeSnapshotRefreshDispatches(t *testing.T, root string, dispatches []codexSurveyorDispatch) {
	t.Helper()
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for i := range dispatches {
		dispatch := &dispatches[i]
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			t.Fatalf("record survey spawn: %v", err)
		}
		dispatch.Status = "completed"
		dispatch.Summary = "survey complete"
		dispatch.FilesCreated = nil
		for _, rel := range dispatch.OutputPaths {
			path := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir dispatch output dir: %v", err)
			}
			if err := os.WriteFile(path, []byte("# "+filepath.Base(path)+"\n\nsurveyed for TestColonizeRefreshesTheSavedMapsRevision\n"), 0o644); err != nil {
				t.Fatalf("write dispatch output: %v", err)
			}
			dispatch.FilesCreated = append(dispatch.FilesCreated, rel)
		}
		if err := spawnTree.UpdateStatus(dispatch.Name, dispatch.Status, dispatch.Summary); err != nil {
			t.Fatalf("complete survey spawn: %v", err)
		}
	}
}

// readColonizeSnapshotRefreshMetadata reads the saved map's own record file
// the same way the runtime and the journey check both read it.
func readColonizeSnapshotRefreshMetadata(t *testing.T, root string) territorySnapshotMetadata {
	t.Helper()
	snapPath := filepath.Join(root, filepath.FromSlash(territorySnapshotRelativePath))
	snapData, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatalf("read %s: %v", snapPath, err)
	}
	var snap territorySnapshotMetadata
	if err := json.Unmarshal(snapData, &snap); err != nil {
		t.Fatalf("%s does not parse: %v", snapPath, err)
	}
	return snap
}

// TestSavedMapPublicationHasOneBuilder is the structural guard for the
// must-have that whatever arranges for the saved map to be rewritten is
// built in exactly one place: bindTransactionalTerritoryPublication
// (cmd/codex_colonize.go), called by both the plan front door
// (territoryPlanPreflight, cmd/codex_workflow_cmds.go) and colonize's own
// forced resurvey (runCodexColonizePlanOnly). It fails by name if a second,
// independent call site anywhere in the module sets the lane-selecting field
// (codexColonizeManifest.PublicationMode) directly instead of going through
// the shared builder -- exactly the kind of drift 208-14-PLAN.md exists to
// close. The walk catches both ordinary ways Go can set that field: an
// assignment statement (x.PublicationMode = ...) and a composite struct
// literal (codexColonizeManifest{PublicationMode: ...}, or the same field
// set inside a literal whose type is inferred from context) -- the second
// of which is the more idiomatic of the two, and was the exact shape
// 208-REVIEW-GAP3.md's CR-01 used to defeat the original, narrower walk.
//
// What this still cannot catch, stated honestly rather than claimed away,
// to the same standard TestSelfRecoveryHasOneDecision's doc comment sets
// (cmd/refusal_self_recovery_test.go): a write performed through Go's
// reflection package, where the field name never appears as a source
// identifier at all; the already-set field travelling into a second
// manifest by copying a whole struct value -- propagation of an existing
// decision rather than a new one, but just as invisible to a walk that only
// looks for the field name being assigned; the field arriving from decoded
// JSON, where the name appears only as a struct tag on a field declaration,
// never as an identifier being assigned; and a second builder written
// inside cmd/codex_colonize.go itself, which this guard allows by
// construction because that whole file sits on the permitted side of the
// walk. Nothing in this repository writes PublicationMode any of these four
// ways today.
func TestSavedMapPublicationHasOneBuilder(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	const soleBuilderFile = "cmd/codex_colonize.go"

	var goFiles []string
	walkErr := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "vendor", "node_modules", "testdata", ".planning", "dist", "worktrees":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		goFiles = append(goFiles, path)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", repoRoot, walkErr)
	}
	if len(goFiles) < 50 {
		t.Fatalf("only %d non-test .go files found under %s -- the walk is not reaching the module, so this guard would pass vacuously", len(goFiles), repoRoot)
	}

	var sawSoleBuilderFile bool
	var offenders []string
	fset := token.NewFileSet()
	for _, path := range goFiles {
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if rel == soleBuilderFile {
			sawSoleBuilderFile = true
		}

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", rel, parseErr)
		}

		var setsPublicationMode bool
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				for _, lhs := range node.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if ok && sel.Sel.Name == "PublicationMode" {
						setsPublicationMode = true
					}
				}
			case *ast.CompositeLit:
				// Not narrowed to a particular struct type name: a
				// composite literal can be written with its type inferred
				// from context (inside a slice, a map, or a nested field),
				// so requiring the type name to appear here would
				// reintroduce a hole of exactly the shape this case exists
				// to close (208-REVIEW-GAP3.md CR-01).
				for _, elt := range node.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if ok && key.Name == "PublicationMode" {
						setsPublicationMode = true
					}
				}
			}
			return true
		})
		if setsPublicationMode && rel != soleBuilderFile {
			offenders = append(offenders, rel)
		}
	}

	if !sawSoleBuilderFile {
		t.Fatalf("%s was never visited by the walk -- this guard cannot be passing for the right reason", soleBuilderFile)
	}
	if len(offenders) > 0 {
		t.Fatalf("only %s may set PublicationMode -- the saved map's publication binding must be built in exactly one shared place; found a second builder in: %v", soleBuilderFile, offenders)
	}
}
