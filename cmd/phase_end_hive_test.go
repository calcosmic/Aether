package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Task 1 (198.1-05): strong lessons reach the shared cross-project store at
// every check, not only at project close, under the same AETHER_HIVE_POLICY
// switch seal already honours.
// ---------------------------------------------------------------------------

// testPhaseEndHiveSentence is the exact sentence every promotion assertion
// below looks for -- named after the shared floor both check lanes run, so
// a reader recognizes it as a real fact about this repository rather than a
// throwaway fixture string.
const testPhaseEndHiveSentence = "cmd/deterministic_floor.go is the one body both check lanes run; changing it changes both"

// seedPhaseEndHiveInstinct writes a single instinct directly to
// instincts.json against the current store global. Seeding directly (rather
// than deriving the instinct through the full observe -> consolidate
// pipeline Plan 03 already proved) is deliberate: this task's own concern is
// what happens to an ALREADY-ELIGIBLE instinct, not how one comes to exist,
// and the plan's own action text says to seed it.
func seedPhaseEndHiveInstinct(t *testing.T, action string, confidence float64) {
	t.Helper()
	if err := store.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{
		{ID: "inst_phase_end_hive", Trigger: "t", Action: action, Domain: "pattern", TrustScore: 0.5, TrustTier: "medium", Confidence: confidence},
	}}); err != nil {
		t.Fatalf("seed instinct fixture: %v", err)
	}
}

// readHiveWisdomTexts reads every stored entry's Text field from the hub's
// hive/wisdom.json under hubDir. A missing file (nothing ever promoted)
// returns nil, not an error.
func readHiveWisdomTexts(t *testing.T, hubDir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(hubDir, "hive", "wisdom.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read hub wisdom.json: %v", err)
	}
	var wf hiveWisdomData
	if err := json.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("unmarshal hub wisdom.json: %v", err)
	}
	texts := make([]string, 0, len(wf.Entries))
	for _, e := range wf.Entries {
		texts = append(texts, e.Text)
	}
	return texts
}

func hiveWisdomContains(texts []string, want string) bool {
	for _, got := range texts {
		if got == want {
			return true
		}
	}
	return false
}

// setupPhaseEndHiveContinueFixture creates a two-phase colony (phase 1 ready
// to build with one task, phase 2 pending) and returns the project root, so
// a test can drive a real check lane to a durable phase advance rather than
// calling promotePhaseEndInstinctsToHive directly.
func setupPhaseEndHiveContinueFixture(t *testing.T, name string) string {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := name
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   name,
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Advance durably", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Continue forward", Status: colony.TaskPending}},
				},
			},
		},
	})

	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed observations fixture: %v", err)
	}

	seedContinueBuildPacket(t, dataDir, 1, name, goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-hive-1", Task: "Advance durably", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-hive-1", Task: "Independent verification before advancement", Status: "completed"},
	})

	return root
}

// driveFastLaneContinue drives cmd/codex_continue.go's default lane to a
// durable advance, failing the test if it does not advance.
func driveFastLaneContinue(t *testing.T, root string) map[string]interface{} {
	t.Helper()
	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}
	return result
}

// driveWrapperLaneContinue drives cmd/codex_continue_finalize.go's external
// (wrapper-driven) lane to a durable advance, failing the test if it does
// not advance.
func driveWrapperLaneContinue(t *testing.T, root string) map[string]interface{} {
	t.Helper()
	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}
	results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
	for _, dispatch := range plan.Dispatches {
		results = append(results, codexContinueExternalDispatch{
			Stage: dispatch.Stage, Wave: dispatch.Wave, Caste: dispatch.Caste,
			Name: dispatch.Name, Task: dispatch.Task, TaskID: dispatch.TaskID,
			Status:    "completed",
			Summary:   dispatch.Name + " cleared the check",
			Artifacts: validCompletedReviewerArtifacts(t, dispatch.Caste),
			Handoff: codex.WorkerHandoff{
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{dispatch.Name + " found no blocking issues"},
			},
		})
	}
	result, _, _, _, _, _, err := runCodexContinueFinalize(root, codexExternalContinueCompletion{
		ContinueManifest: &plan,
		Dispatches:       results,
	}, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true, got %v", result)
	}
	return result
}

// TestStrongInstinctReachesTheSharedStoreAtCheck is the plan's headline
// proof: a strong instinct promotes to the hub's shared store at the end of
// a real check, on both continue lanes -- not only at seal.
func TestStrongInstinctReachesTheSharedStoreAtCheck(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, root string) map[string]interface{}
	}{
		{name: "fast lane", run: driveFastLaneContinue},
		{name: "wrapper lane", run: driveWrapperLaneContinue},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			hubDir := t.TempDir()
			t.Setenv("AETHER_HUB_DIR", hubDir)
			t.Setenv(hivePolicyEnv, "promote")

			root := setupPhaseEndHiveContinueFixture(t, "Strong instinct reaches the hive at check, "+tc.name)
			seedPhaseEndHiveInstinct(t, testPhaseEndHiveSentence, 0.85)

			tc.run(t, root)

			texts := readHiveWisdomTexts(t, hubDir)
			if !hiveWisdomContains(texts, testPhaseEndHiveSentence) {
				t.Fatalf("expected %q in the hub's wisdom.json after the check, got %v", testPhaseEndHiveSentence, texts)
			}
		})
	}
}

// TestHivePromotionAtCheckHonoursThePolicySwitch proves phase-end promotion
// obeys AETHER_HIVE_POLICY exactly the same way seal's own promotion loop
// does: unset, empty and "promote" write; "read" and "off" do not; an
// unrecognized value fails safe to off AND warns to stderr naming it.
func TestHivePromotionAtCheckHonoursThePolicySwitch(t *testing.T) {
	cases := []struct {
		name      string
		unset     bool
		value     string
		wantWrite bool
		wantWarn  bool
	}{
		{name: "unset", unset: true, wantWrite: true},
		{name: "empty", value: "", wantWrite: true},
		{name: "promote", value: "promote", wantWrite: true},
		{name: "read", value: "read", wantWrite: false},
		{name: "off", value: "off", wantWrite: false},
		{name: "nonsense", value: "nonsense", wantWrite: false, wantWarn: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			resetHivePolicyWarnOnceForTest()
			s, _ := newTestStore(t)
			store = s

			hubDir := t.TempDir()
			t.Setenv("AETHER_HUB_DIR", hubDir)

			if tc.unset {
				orig, had := os.LookupEnv(hivePolicyEnv)
				os.Unsetenv(hivePolicyEnv)
				t.Cleanup(func() {
					if had {
						os.Setenv(hivePolicyEnv, orig)
					} else {
						os.Unsetenv(hivePolicyEnv)
					}
				})
			} else {
				t.Setenv(hivePolicyEnv, tc.value)
			}

			action := testPhaseEndHiveSentence + " (" + tc.name + ")"
			seedPhaseEndHiveInstinct(t, action, 0.85)

			var eligible, promoted int
			stderrOut := captureStderrForConsolidationTest(t, func() {
				eligible, promoted = promotePhaseEndInstinctsToHive(1)
			})

			if eligible != 1 {
				t.Fatalf("eligible = %d, want 1 (the switch must never affect eligibility, only writing)", eligible)
			}

			texts := readHiveWisdomTexts(t, hubDir)
			gotWrite := hiveWisdomContains(texts, action)
			if gotWrite != tc.wantWrite {
				t.Fatalf("wrote to the hub = %v, want %v (texts=%v)", gotWrite, tc.wantWrite, texts)
			}
			wantPromoted := 0
			if tc.wantWrite {
				wantPromoted = 1
			}
			if promoted != wantPromoted {
				t.Fatalf("promoted = %d, want %d", promoted, wantPromoted)
			}
			if tc.wantWarn && !strings.Contains(stderrOut, tc.value) {
				t.Fatalf("expected the unrecognized-value stderr warning naming %q, got %q", tc.value, stderrOut)
			}
		})
	}
}

// TestWeakInstinctIsNotPromotedAtCheck proves the 0.8 confidence bar is
// still honoured at check time, exactly as it is at seal.
func TestWeakInstinctIsNotPromotedAtCheck(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	t.Setenv(hivePolicyEnv, "promote")

	seedPhaseEndHiveInstinct(t, "a weak lesson nobody should share across projects yet", 0.79)

	eligible, promoted := promotePhaseEndInstinctsToHive(1)
	if eligible != 0 {
		t.Fatalf("eligible = %d, want 0 (confidence 0.79 is below the 0.8 bar)", eligible)
	}
	if promoted != 0 {
		t.Fatalf("promoted = %d, want 0", promoted)
	}
	texts := readHiveWisdomTexts(t, hubDir)
	if len(texts) != 0 {
		t.Fatalf("expected nothing written to the hub, got %v", texts)
	}
}

// TestRepeatedPhaseEndPromotionDoesNotDuplicate proves the same instinct
// promoted at the end of two consecutive phases produces one hub entry, not
// two -- the storage layer's own (text, domain) reinforcement, exercised
// from this call site.
func TestRepeatedPhaseEndPromotionDoesNotDuplicate(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	t.Setenv(hivePolicyEnv, "promote")

	seedPhaseEndHiveInstinct(t, testPhaseEndHiveSentence, 0.85)

	if _, promoted := promotePhaseEndInstinctsToHive(1); promoted != 1 {
		t.Fatalf("phase 1: promoted = %d, want 1", promoted)
	}
	if _, promoted := promotePhaseEndInstinctsToHive(2); promoted != 1 {
		t.Fatalf("phase 2: promoted = %d, want 1 (reinforcing an existing entry still counts as promoted)", promoted)
	}

	texts := readHiveWisdomTexts(t, hubDir)
	count := 0
	for _, txt := range texts {
		if txt == testPhaseEndHiveSentence {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 hub entry for the twice-promoted instinct, got %d (texts=%v)", count, texts)
	}
}

// TestHiveFailureNeverBlocksThePhase points the hub at an unwritable path
// and proves a real check lane still durably advances the phase.
func TestHiveFailureNeverBlocksThePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	hubParent := t.TempDir()
	hubFile := filepath.Join(hubParent, "hub-is-a-file")
	if err := os.WriteFile(hubFile, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("seed hub-as-file fixture: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hubFile)
	t.Setenv(hivePolicyEnv, "promote")

	root := setupPhaseEndHiveContinueFixture(t, "Hive failure never blocks the phase")
	seedPhaseEndHiveInstinct(t, testPhaseEndHiveSentence, 0.85)

	// The proof is that this still returns advanced:true despite the hub
	// being unwritable -- driveFastLaneContinue itself fails the test if
	// not. wisdom.json is never even created (hubFile is a plain file, not
	// a directory), so there is nothing further to read.
	driveFastLaneContinue(t, root)
}

// ---------------------------------------------------------------------------
// Task 2 (198.1-05): every learning claim CLAUDE.md keeps names a live test.
// ---------------------------------------------------------------------------

// claudeMDLearningSections are the top-level ("## ") CLAUDE.md sections this
// plan's own edits touch. Scoped here rather than the whole document
// because other CLAUDE.md sections cite their own tests under a different
// guard, authored for a different owner's edit -- this guard only owns the
// learning-loop claims it rewrote.
var claudeMDLearningSections = []string{
	"## The Core Insight",
	"## Midden System (Failure Tracking)",
	"## Wisdom Pipeline",
	"## Hive Brain (Cross-Colony Wisdom)",
}

// testNameShapeRe matches the Go test-function-name shape: "Test" followed
// immediately by an upper-case letter, whether backtick-wrapped in CLAUDE.md
// prose or bare.
var testNameShapeRe = regexp.MustCompile(`\bTest[A-Z]\w*`)

// extractMarkdownSection is the shared section extractor already used by
// cmd/codex_continue.go (verification-command scanning); reused here rather
// than duplicated -- it is heading-level aware, so "## Wisdom Pipeline" ends
// at the next "## " heading without being fooled by the "### Pipeline
// Stages" subheading inside it.

// collectLiveGoTestFuncNames parses every *_test.go file under cmd/ and
// pkg/ with go/parser -- never grep, so a name sitting inside a comment or a
// string literal cannot satisfy this guard -- and returns the set of
// top-level func TestXxx(t *testing.T)-shaped declarations actually
// compiled into the suite.
func collectLiveGoTestFuncNames(t *testing.T, root string) map[string]bool {
	t.Helper()
	names := make(map[string]bool)
	for _, sub := range []string{"cmd", "pkg"} {
		dir := filepath.Join(root, sub)
		fset := token.NewFileSet()
		walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || fn.Body == nil {
					continue
				}
				name := fn.Name.Name
				if len(name) <= len("Test") || !strings.HasPrefix(name, "Test") {
					continue
				}
				if !unicode.IsUpper(rune(name[len("Test")])) {
					continue
				}
				if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
					continue
				}
				names[name] = true
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", dir, walkErr)
		}
	}
	return names
}

// TestEveryLearningClaimInCLAUDEMDNamesALiveTest is the removal-proof guard
// for D-06: every learning-loop sentence this plan kept in CLAUDE.md names a
// test, and every named test must actually exist. Renaming or deleting any
// one cited test, without updating CLAUDE.md to match, makes this fail
// naming that test.
func TestEveryLearningClaimInCLAUDEMDNamesALiveTest(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	claudeMD, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	text := string(claudeMD)

	cited := make(map[string]bool)
	for _, heading := range claudeMDLearningSections {
		section := extractMarkdownSection(text, heading)
		if section == "" {
			t.Errorf("expected to find section %q in CLAUDE.md -- this guard cannot check a section it cannot locate", heading)
			continue
		}
		for _, name := range testNameShapeRe.FindAllString(section, -1) {
			cited[name] = true
		}
	}
	if t.Failed() {
		return
	}
	if len(cited) == 0 {
		t.Fatal("extracted zero cited test names from the scanned CLAUDE.md sections -- this guard is checking nothing")
	}

	live := collectLiveGoTestFuncNames(t, root)

	var missing []string
	for name := range cited {
		if !live[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("CLAUDE.md names %d test(s) that do not exist as func %s(t *testing.T) under cmd/ or pkg/: %s",
			len(missing), missing[0], strings.Join(missing, ", "))
	}
}

// ---------------------------------------------------------------------------
// Task 3 (198.1-05): one run, four stores fed -- the phase's own proof.
// ---------------------------------------------------------------------------

const (
	oneRunDoNotRepeat = "never merge cmd/phase_end_hive.go without running the four-store fixture test first"
	oneRunBlocker     = "review blocked: the Wisdom Pipeline table still described a stage nothing called"
)

// setupOneRunFeedsEveryStoreFixture is setupExternalBuildAttemptTestWithVerifiableWork
// (build_attempt_external_test.go) with a second, pending phase added, so a
// real check-lane run following the build has somewhere durable to advance
// to.
func setupOneRunFeedsEveryStoreFixture(t *testing.T) string {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "One run feeds every store"
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{
			{
				ID:          1,
				Name:        "One run feeds every store",
				Description: "Bind one wrapper dispatch to one lifecycle commit; design the boundary first",
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Write durable evidence", Status: colony.TaskPending}},
			},
			{
				ID:     2,
				Name:   "Next phase",
				Status: colony.PhasePending,
				Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Continue forward", Status: colony.TaskPending}},
			},
		}},
	})
	return root
}

// TestOneRunFeedsEveryStore is the phase's own end-to-end proof: one build
// (one worker succeeding with a do-not-repeat sentence, one worker failing
// with a blocker sentence) followed by one check to a durable advance must
// leave content in all four stores this phase (and 198.1-01 through -04)
// wired -- the observation log, the failure log, the instinct store, and the
// signal store. Each assertion names the fixture's own sentence and its own
// store, so a regression says which link broke.
func TestOneRunFeedsEveryStore(t *testing.T) {
	root := setupOneRunFeedsEveryStoreFixture(t)

	manifest, _ := prepareExternalBuildCompletionWithProposal(t, root,
		[]string{"builder", "architect"}, []string{"architect=needs a design review before this lands"})
	if len(manifest.Dispatches) < 2 {
		t.Fatalf("fixture must produce at least 2 dispatches, got %d", len(manifest.Dispatches))
	}
	if err := os.WriteFile(filepath.Join(root, "external-evidence.txt"), []byte("durable external work\n"), 0o644); err != nil {
		t.Fatalf("write external evidence: %v", err)
	}

	results := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		if dispatch.Caste == "builder" {
			results = append(results, codexExternalBuildWorkerResult{
				Stage: dispatch.Stage, Wave: dispatch.Wave, ExecutionWave: normalizedDispatchWave(dispatch),
				Caste: dispatch.Caste, Name: dispatch.Name, TaskID: dispatch.TaskID,
				Status:        "completed",
				Summary:       dispatch.Name + " completed",
				FilesModified: []string{"external-evidence.txt"},
				Handoff: codex.WorkerHandoff{
					CommandsRun:            []string{"go test ./..."},
					VerificationStatus:     "pass",
					NextWorkerInstructions: []string{dispatch.Name + " work is complete"},
					DoNotRepeat:            []string{oneRunDoNotRepeat},
				},
			})
			continue
		}
		results = append(results, codexExternalBuildWorkerResult{
			Stage: dispatch.Stage, Wave: dispatch.Wave, ExecutionWave: normalizedDispatchWave(dispatch),
			Caste: dispatch.Caste, Name: dispatch.Name, TaskID: dispatch.TaskID,
			Status:   "failed",
			Summary:  "worker hit a blocking failure",
			Blockers: []string{oneRunBlocker},
		})
	}
	completion := loadedCompletionFromManifest(t, manifest, results)

	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
		t.Fatalf("build-finalize: %v", err)
	}

	// Store 1: the observation log. The successful worker's own
	// do-not-repeat sentence must be captured, on the same boundary Plan 01
	// wired.
	var obsFile colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &obsFile); err != nil {
		t.Fatalf("learning-observations.json: load: %v", err)
	}
	foundObservation := false
	for _, obs := range obsFile.Observations {
		if obs.Content == oneRunDoNotRepeat {
			foundObservation = true
			break
		}
	}
	if !foundObservation {
		t.Errorf("learning-observations.json: expected the successful worker's own sentence %q, got %+v", oneRunDoNotRepeat, obsFile.Observations)
	}

	// Store 2: the failure log. The failed worker's own blocker sentence
	// must be recorded.
	mf, err := loadMiddenFile(store)
	if err != nil {
		t.Fatalf("midden.json: load: %v", err)
	}
	foundFailure := false
	for _, entry := range mf.Entries {
		if strings.Contains(entry.Message, oneRunBlocker) {
			foundFailure = true
			break
		}
	}
	if !foundFailure {
		t.Errorf("midden.json: expected the failed worker's own sentence %q, got %+v", oneRunBlocker, mf.Entries)
	}

	// Now drive the check to a durable advance, so consolidation (Plan 03),
	// hive promotion (this plan) and phase-end signals (Plan 04) all fire.
	seedContinueBuildPacket(t, filepath.Join(root, ".aether", "data"), 1, "One run feeds every store", "One run feeds every store", []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-one-run", Task: "Write durable evidence", Status: "completed", TaskID: "1.1"},
		{Stage: "verification", Caste: "watcher", Name: "Keen-one-run", Task: "Independent verification before advancement", Status: "completed"},
	})

	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected the check to durably advance, got %v", result)
	}

	// Store 3: the instinct store. Consolidation must have produced at
	// least one instinct whose provenance traces back to an observation this
	// run actually captured.
	instFile := loadInstinctFileOrEmpty(store)
	if len(instFile.Instincts) == 0 {
		t.Errorf("instincts.json: expected at least 1 instinct after consolidation, got none")
	} else {
		obsHashes := make(map[string]bool, len(obsFile.Observations))
		var refreshedObs colony.LearningFile
		if err := store.LoadJSON("learning-observations.json", &refreshedObs); err == nil {
			for _, obs := range refreshedObs.Observations {
				obsHashes[obs.ContentHash] = true
			}
		}
		tracedToObservation := false
		for _, inst := range instFile.Instincts {
			if obsHashes[inst.Provenance.Source] {
				tracedToObservation = true
				break
			}
		}
		if !tracedToObservation {
			t.Errorf("instincts.json: expected at least 1 instinct whose Provenance.Source matches an observation content hash, got %+v", instFile.Instincts)
		}
	}

	// Store 4: the signal store. A finished check leaves an active FEEDBACK
	// note naming the phase (Plan 04).
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("pheromones.json: load: %v", err)
	}
	foundSignal := false
	for _, sig := range pf.Signals {
		if sig.Type == "FEEDBACK" && sig.Active && strings.Contains(extractSignalText(sig.Content), "One run feeds every store") {
			foundSignal = true
			break
		}
	}
	if !foundSignal {
		t.Errorf("pheromones.json: expected an active FEEDBACK signal naming the fixture phase, got %+v", pf.Signals)
	}
}
