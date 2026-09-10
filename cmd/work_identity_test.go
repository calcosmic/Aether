package cmd

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 08 -- one job identity and one attempt identity carried all
// the way through the work cycle (SYN-201-02, WORK-01, WORK-03), and claims/
// verification artifacts moved onto exact attempt-bound paths without
// weakening legacy-compatibility validation (CAP-071). See
// cmd/coherent_jobs.go, cmd/codex_verify_advance.go, cmd/codex_build.go and
// cmd/attempt_artifacts.go for the mechanism these tests prove.

// --- Task 1: waves and job ownership are read from the one planning result ---

// dispatchesFromJobPlanForTest mirrors exactly what
// plannedBuildDispatchesWithJobProposals (cmd/codex_build.go) writes onto
// each job-owning dispatch -- Wave, JobName, JobReason, JobSource,
// CoveredTaskIDs, DependsOn -- so a test can prove the read path
// (attemptCoherentJobWaves/attemptCoherentJobDependencies) reproduces
// planCoherentJobs's own Waves/Jobs verbatim, without depending on the
// caste-detection keyword engine those production dispatches also run
// through.
func dispatchesFromJobPlanForTest(plan *coherentJobPlan) []codexBuildDispatch {
	dispatches := make([]codexBuildDispatch, 0, len(plan.Jobs))
	for _, job := range plan.Jobs {
		primary := ""
		if len(job.Tasks) > 0 {
			primary = job.Tasks[0].ID
		}
		dispatches = append(dispatches, codexBuildDispatch{
			Stage:          "wave",
			Wave:           job.Wave,
			Caste:          job.OwnerCaste,
			Name:           "worker-" + job.Name,
			Task:           "fixture task",
			Status:         "spawned",
			TaskID:         primary,
			JobName:        job.Name,
			JobReason:      job.JobReason,
			JobSource:      job.Source,
			CoveredTaskIDs: append([]string{}, job.TaskIDs...),
			DependsOn:      append([]string{}, job.DependsOn...),
		})
	}
	return dispatches
}

// threeWaveCoherentJobPlanForTest builds three tasks of three different
// castes, chained A -> B -> C. Differing castes keep planAutomaticCoherentJobs
// from merging them into one job (its union rule requires the SAME caste),
// while the dependency chain still forces three sequential job-level waves
// via populateCoherentJobDAG -- a fixture that exercises real wave
// derivation, not a hand-authored Waves value.
func threeWaveCoherentJobPlanForTest(t *testing.T) *coherentJobPlan {
	t.Helper()
	taskA, seedA := coherentJobTestTask("wave-a", "builder")
	taskB, seedB := coherentJobTestTask("wave-b", "scout", "wave-a")
	seedB.TaskIndex = 1
	taskC, seedC := coherentJobTestTask("wave-c", "watcher", "wave-b")
	seedC.TaskIndex = 2

	plan, err := planCoherentJobs(
		coherentJobTestPhase(301, taskA, taskB, taskC),
		[]coherentJobTask{seedA, seedB, seedC},
		nil,
	)
	if err != nil {
		t.Fatalf("planCoherentJobs returned error: %v", err)
	}
	if len(plan.Waves) != 3 {
		t.Fatalf("fixture produced %d wave(s) (%#v), want exactly 3 -- fixture is broken, not the code under test", len(plan.Waves), plan.Waves)
	}
	return plan
}

// TestCoherentJobPlanningPerformsNoWrites proves planCoherentJobs stays pure:
// no attempt, no manifest, no workspace touch -- a before/after hash of the
// entire fixture data directory must be byte-identical across the call.
func TestCoherentJobPlanningPerformsNoWrites(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	before := hashDirForTest(t, store.BasePath())
	_ = threeWaveCoherentJobPlanForTest(t)
	after := hashDirForTest(t, store.BasePath())
	if before != after {
		t.Fatalf("planCoherentJobs wrote to the fixture data directory: before=%s after=%s", before, after)
	}
}

// TestWavesAreReadNotRederived proves a three-wave planning result reports
// three waves, in the recorded order, downstream -- through
// attemptCoherentJobWaves (the read path attemptCoherentJobWaves feeds) and
// through runContinueAcceptVerifyAdvance, which must read the SAME recorded
// order rather than re-grouping phase.Tasks.
func TestWavesAreReadNotRederived(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	plan := threeWaveCoherentJobPlanForTest(t)
	dispatches := dispatchesFromJobPlanForTest(plan)

	waves := attemptCoherentJobWaves(dispatches)
	if len(waves) != 3 {
		t.Fatalf("attemptCoherentJobWaves reported %d wave(s), want 3: %#v", len(waves), waves)
	}
	for i, wave := range plan.Waves {
		if !reflect.DeepEqual(waves[i].Jobs, wave) {
			t.Fatalf("wave %d job order = %v, want the recorded order %v -- this is a re-derivation, not a read", i, waves[i].Jobs, wave)
		}
	}

	// Now prove the SAME recorded order survives all the way through the
	// unified continue decision body: seed a build attempt whose Dispatches
	// are exactly these, and confirm runContinueAcceptVerifyAdvance's
	// Waves field matches, with no second grouping pass in between.
	phase := colony.Phase{ID: 301, Name: "Three-wave fixture"}
	_, record := seedBuildAttemptForTest(t, 301, "attempt-three-wave", func(r *buildAttemptRecord) {
		r.Dispatches = dispatches
	})
	assessment := codexContinueAssessment{}
	gates := codexContinueGateReport{Passed: true}
	decision := runContinueAcceptVerifyAdvance(phase, assessment, gates, nil, colony.ColonyState{})
	if len(decision.Waves) != 3 {
		t.Fatalf("runContinueAcceptVerifyAdvance reported %d wave(s), want 3: %#v", len(decision.Waves), decision.Waves)
	}
	for i, wave := range plan.Waves {
		if !reflect.DeepEqual(decision.Waves[i].Jobs, wave) {
			t.Fatalf("decision wave %d job order = %v, want the recorded order %v", i, decision.Waves[i].Jobs, wave)
		}
	}
	if record.ID == "" {
		t.Fatalf("seeded attempt has no ID")
	}
}

// TestJobRenderOrderIsDeterministic proves two renders of the same recorded
// plan are byte-identical: attemptCoherentJobWaves and
// attemptCoherentJobDependencies do no map-order-dependent work that isn't
// resorted into a stable result.
func TestJobRenderOrderIsDeterministic(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	plan := threeWaveCoherentJobPlanForTest(t)
	dispatches := dispatchesFromJobPlanForTest(plan)

	first := attemptCoherentJobWaves(dispatches)
	second := attemptCoherentJobWaves(dispatches)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("attemptCoherentJobWaves is not deterministic:\n first=%#v\nsecond=%#v", first, second)
	}

	firstDeps := attemptCoherentJobDependencies(dispatches)
	secondDeps := attemptCoherentJobDependencies(dispatches)
	if !reflect.DeepEqual(firstDeps, secondDeps) {
		t.Fatalf("attemptCoherentJobDependencies is not deterministic:\n first=%#v\nsecond=%#v", firstDeps, secondDeps)
	}
}

// TestSameWaveFileConflictIsRefusedBeforeDispatch is a regression proof that
// the existing pre-dispatch ownership guard (validateDeclaredWorktreeOwnership,
// cmd/codex_build_worktree.go, predates this plan and is unmodified by it)
// still refuses two distinct same-wave jobs declaring the same file, naming
// the file and both jobs, before any worker starts.
func TestSameWaveFileConflictIsRefusedBeforeDispatch(t *testing.T) {
	jobOne := codex.WorkerDispatch{TaskID: "job-one-task", WorkerName: "worker-one", Wave: 1, JobName: "job-one", DeclaredPaths: []string{"shared/README.md"}}
	jobTwo := codex.WorkerDispatch{TaskID: "job-two-task", WorkerName: "worker-two", Wave: 1, JobName: "job-two", DeclaredPaths: []string{"shared/README.md"}}

	err := validateDeclaredWorktreeOwnership([]codex.WorkerDispatch{jobOne, jobTwo})
	if err == nil {
		t.Fatal("two distinct same-wave jobs declaring one file were accepted; want a pre-dispatch refusal")
	}
	if !strings.Contains(err.Error(), "shared/README.md") {
		t.Fatalf("refusal does not name the contested file: %v", err)
	}
	if !strings.Contains(err.Error(), "job-one") || !strings.Contains(err.Error(), "job-two") {
		t.Fatalf("refusal does not name both jobs: %v", err)
	}
}

// --- Task 2: one job and attempt identity on every job-scoped surface ---

// workIdentitySuccessInvoker is a minimal always-succeeds worker invoker: it
// writes the one file it claims (so root-backed receipt evidence validates)
// and reports a passing, completed result for whatever single task it was
// dispatched for.
type workIdentitySuccessInvoker struct{}

func (workIdentitySuccessInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	file := "work-identity-" + strings.ReplaceAll(config.TaskID, ".", "-") + ".go"
	_ = os.WriteFile(filepath.Join(config.Root, file), []byte("package fixture\n"), 0644)
	return codex.WorkerResult{
		WorkerName:    config.WorkerName,
		Caste:         config.Caste,
		TaskID:        config.TaskID,
		Status:        "completed",
		Summary:       "identity fixture finished " + config.TaskID,
		FilesModified: []string{file},
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "pass",
			CommandsRun:        []string{"go test ./..."},
		},
	}, nil
}

func (workIdentitySuccessInvoker) IsAvailable(context.Context) bool { return true }
func (workIdentitySuccessInvoker) ValidateAgent(string) error       { return nil }

// missingIdentitySurfaces walks dispatches -- the live runtime structure
// every job-scoped surface (Queen card rows, workspace lease/dispatch
// records, task receipts, reviewer findings, fan-in) is built from -- and
// names, by dispatch Name, any entry that is missing the attempt identifier
// it should carry once attemptID is non-empty, or missing a job name on a
// dispatch that plainly owns tasks (CoveredTaskIDs or a non-empty TaskID on
// a "wave" stage dispatch). It is not a hand-maintained list of surfaces: a
// new dispatch field added later is still walked by the SAME loop, because
// it is the same struct.
func missingIdentitySurfaces(attemptID string, dispatches []codexBuildDispatch) []string {
	var missing []string
	for _, dispatch := range dispatches {
		if attemptID != "" && strings.TrimSpace(dispatch.AttemptID) == "" {
			missing = append(missing, dispatch.Name+": missing attempt identifier")
			continue
		}
		jobScoped := dispatch.Stage == "wave" && (len(dispatch.CoveredTaskIDs) > 0 || strings.TrimSpace(dispatch.TaskID) != "")
		if jobScoped && strings.TrimSpace(dispatch.JobName) == "" {
			missing = append(missing, dispatch.Name+": missing job name")
		}
	}
	return missing
}

// TestOneJobIdentityReachesEverySurface builds a real phase through the
// direct/native build lane with two independent, same-caste, non-dependent
// tasks (so planCoherentJobs plans them as two DISTINCT jobs in the same
// wave, per planAutomaticCoherentJobs's own union rule -- no shared path, no
// dependency edge), runs it to a real completed attempt, and asserts every
// dispatch on the resulting attempt record -- the Queen card's own rows,
// each one's task receipts, and (for reviewer castes) findings -- carries
// both the job name and this exact attempt's ID.
func TestOneJobIdentityReachesEverySurface(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	taskOneID := "1.1"
	taskTwoID := "1.2"
	goal := "Two independent jobs both name the same attempt"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "light", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Two independent jobs", Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskOneID, Goal: "Add the alpha fixture helper", Hints: []string{"cmd/identity_alpha.go"}, Status: colony.TaskPending},
				{ID: &taskTwoID, Goal: "Add the beta fixture helper", Hints: []string{"cmd/identity_beta.go"}, Status: colony.TaskPending},
			},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return workIdentitySuccessInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}

	attemptRel, record, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatalf("no build attempt was recorded for phase 1")
	}
	if record.ID == "" {
		t.Fatalf("recorded attempt has no ID: %+v", record)
	}
	if len(record.Dispatches) == 0 {
		t.Fatalf("recorded attempt has no dispatches: %+v", record)
	}

	jobNames := map[string]bool{}
	for _, dispatch := range record.Dispatches {
		if dispatch.JobName != "" {
			jobNames[dispatch.JobName] = true
		}
	}
	if len(jobNames) < 2 {
		t.Fatalf("fixture did not produce two distinct jobs (got %v) -- fixture is broken, not the code under test: %#v", jobNames, record.Dispatches)
	}

	if missing := missingIdentitySurfaces(record.ID, record.Dispatches); len(missing) > 0 {
		t.Fatalf("surfaces missing identity on attempt %s: %v", record.ID, missing)
	}
	for _, dispatch := range record.Dispatches {
		if dispatch.AttemptID != "" && dispatch.AttemptID != record.ID {
			t.Fatalf("dispatch %q carries a foreign attempt id %q, want %q", dispatch.Name, dispatch.AttemptID, record.ID)
		}
	}
	_ = attemptRel
}

// TestMissingIdentitySurfaceFailsByName proves missingIdentitySurfaces names
// the exact surface (by dispatch Name) that omits an identifier, so a
// regression in stampDispatchAttemptIdentity or the job-metadata assignment
// is caught by name rather than by a silent aggregate count.
func TestMissingIdentitySurfaceFailsByName(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Name: "worker-alpha", TaskID: "alpha", JobName: "job-alpha", AttemptID: "attempt-x"},
		{Stage: "wave", Name: "worker-beta", TaskID: "beta", JobName: "job-beta" /* AttemptID intentionally omitted */},
	}
	missing := missingIdentitySurfaces("attempt-x", dispatches)
	if len(missing) != 1 {
		t.Fatalf("missingIdentitySurfaces = %v, want exactly one entry", missing)
	}
	if !strings.Contains(missing[0], "worker-beta") {
		t.Fatalf("missing-identity report does not name the offending surface: %v", missing)
	}
}

// --- Task 3: attempt-bound claims/verification artifact paths ---

// TestArtifactsAreAttemptBound proves a new attempt writes its claims and
// verification artifacts under a path containing its attempt identifier,
// and that two attempts produce two distinct paths whose bytes never
// change when the other attempt writes.
func TestArtifactsAreAttemptBound(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	claimsOne := &codexBuildClaims{FilesCreated: []string{"one.go"}, BuildPhase: 1}
	if err := writeAttemptBoundArtifact("attempt-one", attemptArtifactKindClaims, claimsOne); err != nil {
		t.Fatalf("write attempt-one claims: %v", err)
	}
	relOne, err := attemptBoundArtifactPath("attempt-one", attemptArtifactKindClaims)
	if err != nil {
		t.Fatalf("attemptBoundArtifactPath: %v", err)
	}
	if !strings.Contains(relOne, "attempt-one") {
		t.Fatalf("attempt-bound path %q does not contain the attempt identifier", relOne)
	}

	var loadedOne codexBuildClaims
	if err := readAttemptBoundArtifact("attempt-one", attemptArtifactKindClaims, &loadedOne); err != nil {
		t.Fatalf("read attempt-one claims: %v", err)
	}
	if !reflect.DeepEqual(&loadedOne, claimsOne) {
		t.Fatalf("loadedOne = %+v, want %+v", loadedOne, claimsOne)
	}

	verificationOne := map[string]any{"phase": 1, "verified": true}
	if err := writeAttemptBoundArtifact("attempt-one", attemptArtifactKindVerification, verificationOne); err != nil {
		t.Fatalf("write attempt-one verification: %v", err)
	}

	claimsTwo := &codexBuildClaims{FilesCreated: []string{"two.go"}, BuildPhase: 1}
	if err := writeAttemptBoundArtifact("attempt-two", attemptArtifactKindClaims, claimsTwo); err != nil {
		t.Fatalf("write attempt-two claims: %v", err)
	}
	relTwo, err := attemptBoundArtifactPath("attempt-two", attemptArtifactKindClaims)
	if err != nil {
		t.Fatalf("attemptBoundArtifactPath: %v", err)
	}
	if relOne == relTwo {
		t.Fatalf("two distinct attempts produced the same artifact path %q", relOne)
	}

	// Writing the second attempt's claims must not change the first's.
	var reloadedOne codexBuildClaims
	if err := readAttemptBoundArtifact("attempt-one", attemptArtifactKindClaims, &reloadedOne); err != nil {
		t.Fatalf("reload attempt-one claims: %v", err)
	}
	if !reflect.DeepEqual(&reloadedOne, claimsOne) {
		t.Fatalf("attempt-one claims changed after writing attempt-two: got %+v, want %+v", reloadedOne, claimsOne)
	}
}

// TestLegacyArtifactReadIsValidatedIdentically proves a legacy root-level
// artifact (written before attempt-bound paths existed) is still readable
// through the identical validation path, and that a malformed legacy file
// is refused with the same class of error a malformed attempt-bound file
// produces -- never a silent empty-value fallback.
func TestLegacyArtifactReadIsValidatedIdentically(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	legacy := &codexBuildClaims{FilesCreated: []string{"legacy.go"}, BuildPhase: 3}
	legacyName := legacyAttemptArtifactNames[attemptArtifactKindClaims]
	if err := store.SaveJSON(legacyName, legacy); err != nil {
		t.Fatalf("seed legacy claims fixture: %v", err)
	}

	var loaded codexBuildClaims
	if err := readAttemptBoundArtifact("attempt-never-written", attemptArtifactKindClaims, &loaded); err != nil {
		t.Fatalf("legacy compatibility read failed: %v", err)
	}
	if !reflect.DeepEqual(&loaded, legacy) {
		t.Fatalf("loaded = %+v, want the legacy fixture %+v", loaded, legacy)
	}

	// A malformed legacy file must be refused, not silently treated as
	// empty. store.SaveJSON refuses invalid JSON outright, so bypass it via
	// a direct filesystem write to the resolved legacy path.
	malformedLegacyName := legacyAttemptArtifactNames[attemptArtifactKindVerification]
	if err := os.WriteFile(filepath.Join(store.BasePath(), malformedLegacyName), []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("write malformed legacy fixture: %v", err)
	}
	var out map[string]any
	legacyErr := readAttemptBoundArtifact("attempt-legacy-malformed", attemptArtifactKindVerification, &out)
	if legacyErr == nil {
		t.Fatal("malformed legacy artifact was accepted")
	}
	if !strings.Contains(legacyErr.Error(), "unmarshal") {
		t.Fatalf("malformed legacy artifact error does not name a validation failure: %v", legacyErr)
	}

	// The SAME validation path must refuse an equally malformed
	// attempt-bound file, with an error of the same shape.
	attemptRel, err := attemptBoundArtifactPath("attempt-bound-malformed", attemptArtifactKindVerification)
	if err != nil {
		t.Fatalf("attemptBoundArtifactPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(store.BasePath(), attemptRel)), 0755); err != nil {
		t.Fatalf("mkdir attempt directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.BasePath(), attemptRel), []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("write malformed attempt-bound fixture: %v", err)
	}
	attemptErr := readAttemptBoundArtifact("attempt-bound-malformed", attemptArtifactKindVerification, &out)
	if attemptErr == nil {
		t.Fatal("malformed attempt-bound artifact was accepted")
	}
	if !strings.Contains(attemptErr.Error(), "unmarshal") {
		t.Fatalf("malformed attempt-bound artifact error does not name a validation failure: %v", attemptErr)
	}
}

// TestArtifactPathRefusesEscape proves a traversal-shaped attempt identifier
// and a symlink escape are both refused before any read or write.
func TestArtifactPathRefusesEscape(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	traversalCases := []string{"../escape", "a/../../etc", "a/b", "", ".", "..", "a\x00b"}
	for _, id := range traversalCases {
		if _, err := attemptBoundArtifactPath(id, attemptArtifactKindClaims); err == nil {
			t.Fatalf("attempt id %q was accepted, want refusal", id)
		}
	}

	attemptsDir := filepath.Join(store.BasePath(), "attempts")
	if err := os.MkdirAll(attemptsDir, 0755); err != nil {
		t.Fatalf("mkdir attempts dir: %v", err)
	}
	outsideDir := t.TempDir()
	symlinkPath := filepath.Join(attemptsDir, "escape-link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("symlinks unsupported in this environment: %v", err)
	}
	if _, err := attemptBoundArtifactPath("escape-link", attemptArtifactKindClaims); err == nil {
		t.Fatal("a symlinked attempt directory escaping the colony data root was accepted")
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "claims.json")); !os.IsNotExist(err) {
		t.Fatalf("escape refusal did not prevent a write outside the data root (stat err=%v)", err)
	}
}
