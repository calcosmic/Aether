package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningStageResume is the stage an already-started planning run is still
// waiting on, together with the exact stage manifest its worker must be bound
// to.
//
// Why this exists: the staged planner alternates Scout -> Route-Setter ->
// Scout. Each finalize issues the NEXT stage's authorization and stage manifest
// (the Scout finalize returns route_stage_manifest; the Route-Setter finalize
// returns scout_stage_manifest), but nothing ever wrapped that stage manifest
// back into the codexPlanManifest the finalizer requires. runCodexPlanPlanOnly
// called preparePlanningScoutStage unconditionally, so a run parked at
// route_running could not be advanced at all: re-running started a SECOND run
// at the Scout stage while the first sat parked forever, and the Route-Setter
// stage manifest it had already issued had nowhere to be submitted.
//
// resolvePlanningStageResume closes that loop. It is read-only -- it resolves
// what the run is waiting on and rebuilds the dispatch envelope for it; it
// never advances the stage machine, which stays owned by the finalizers.
type planningStageResume struct {
	RunID    string
	Pass     int
	Stage    planningStage
	Caste    planningStageWorkerCaste
	Manifest planningStageManifest
}

// nextCommandHint is the plain sentence the plan result shows for this resume.
func (r planningStageResume) nextCommandHint() string {
	switch r.Caste {
	case planningStageCasteRouteSetter:
		return "dispatch Route-Setter with stage_manifest, then `aether plan-finalize --completion-file <file>`"
	case planningStageCasteScout:
		return "dispatch Scout with stage_manifest, then `aether plan-finalize --completion-file <file>`"
	default:
		return "aether plan"
	}
}

// resolvePlanningStageResume reports the in-flight stage a new plan-only call
// must dispatch instead of starting a fresh run, or nil when there is no such
// run.
//
// It takes the CALLER'S session rather than a root, and every loader below is
// the InSession variant, because the plan-only path already holds the planning
// lock when it asks. The root-taking loaders each open a session of their own;
// calling one from inside an existing session made the process wait on a lock
// it was already holding, and `aether plan` hung forever against a real colony
// with no CPU burn and no error -- caught only by running the wired path, since
// calling this helper directly in a test never holds the lock first.
//
// A missing iteration state, a missing stage file, or a stage that is not
// waiting on a worker are all "nothing to resume" -- they return (nil, nil) so
// the caller falls through to starting a run normally. Only a genuinely
// corrupt in-flight run returns an error: a run that says it is waiting on a
// worker whose authorization cannot be loaded must not be silently replaced by
// a second run, because that is the stray-run bug this function exists to fix.
func resolvePlanningStageResume(session *planningMutationSession) (*planningStageResume, error) {
	runID := ""
	if seed, ok := loadPlanningIterationState(); ok {
		runID = strings.TrimSpace(seed.PlanningRunID)
	}
	if runID == "" {
		// The iteration state is a convenience pointer, not the durable
		// truth, and a colony can sit parked with no pointer at all -- the
		// real CosmicDashboard colony that reported this defect had a run
		// stopped at route_running and no iteration-state.json anywhere.
		// Keying the resume solely off the pointer would leave exactly the
		// stuck colonies this exists to rescue still stuck, so fall back to
		// the per-run stage files, which ARE durable.
		discovered, err := discoverParkedPlanningRun(session)
		if err != nil {
			return nil, err
		}
		if discovered == "" {
			return nil, nil
		}
		runID = discovered
	}
	stageState, err := loadPlanningStageStateInSession(session, runID)
	if err != nil {
		// No stage file for this run: nothing is in flight.
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(stageState.RunID) == "" {
		return nil, nil
	}

	switch stageState.Stage {
	case planningStageRouteRunning:
		dispatch, err := loadPlanningScoutRouteDispatchInSession(session, stageState.RunID, stageState.Pass)
		if err != nil {
			return nil, fmt.Errorf("planning run %s is waiting on Route-Setter but its authorization could not be read: %w", stageState.RunID, err)
		}
		return &planningStageResume{
			RunID:    stageState.RunID,
			Pass:     dispatch.Manifest.Pass,
			Stage:    stageState.Stage,
			Caste:    planningStageCasteRouteSetter,
			Manifest: dispatch.Manifest,
		}, nil
	case planningStageScoutRunning:
		// Pass 1's Scout stage is created by preparePlanningScoutStage and has
		// no separate authorization file; only a Scout re-authorized BY a
		// Route-Setter pass does. A pass-1 Scout that is still running is
		// therefore left to the normal path, which reissues it.
		dispatch, err := loadPlanningRouteScoutDispatchInSession(session, stageState.RunID, stageState.Pass)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			return nil, fmt.Errorf("planning run %s is waiting on Scout but its authorization could not be read: %w", stageState.RunID, err)
		}
		return &planningStageResume{
			RunID:    stageState.RunID,
			Pass:     dispatch.Manifest.Pass,
			Stage:    stageState.Stage,
			Caste:    planningStageCasteScout,
			Manifest: dispatch.Manifest,
		}, nil
	default:
		// preset_required, owner_decision, spec_approval_required,
		// candidate_ready, failed: none of these is waiting on a worker
		// dispatch, so none of them is resumed here.
		return nil, nil
	}
}

// buildPlanningStageResumeManifest rebuilds the codexPlanManifest that binds a
// worker to an already-issued stage manifest.
//
// Every field the stage finalizers cross-check is copied FROM the stage
// manifest rather than recomputed, because the finalizer refuses any manifest
// whose run, pass, preset, or base plan revision disagrees with the stage it
// claims to serve. Deriving them a second time from live state is exactly how
// those fields drift apart.
func buildPlanningStageResumeManifest(root string, state colony.ColonyState, resume planningStageResume, granularity colony.PlanGranularity, planDepth, planningDepth, verificationDepth string, survey codexSurveyContext, contextCapsule string, opts codexPlanOptions, generatedAt time.Time) (codexPlanManifest, error) {
	spec, ok := planningStageResumeWorkerSpec(string(resume.Caste))
	if !ok {
		return codexPlanManifest{}, fmt.Errorf("no planning worker definition for caste %q", resume.Caste)
	}
	stageManifest := resume.Manifest

	dispatch := codexPlanningDispatch{
		Stage:         string(resume.Stage),
		Wave:          1,
		Caste:         spec.Caste,
		AgentName:     strings.TrimSuffix(spec.AgentFile, ".toml"),
		Name:          deterministicAntName(spec.Caste, root+"|plan|"+spec.Caste),
		Task:          spec.Task,
		TaskID:        stageManifest.AuthorizationID,
		Outputs:       append([]string{}, spec.Outputs...),
		Status:        "planned",
		StageManifest: &stageManifest,
	}
	dispatches := []codexPlanningDispatch{dispatch}
	attachPlanningDispatchSkillAssignments(dispatches)
	dispatches[0].Brief = renderPlanningWorkerBrief(root, survey, spec)

	manifest := codexPlanManifest{
		Goal:               strings.TrimSpace(*state.Goal),
		Root:               root,
		GeneratedAt:        generatedAt.Format(time.RFC3339),
		BaseRevisionID:     stageManifest.BasePlanRevisionID,
		BasePlanStateHash:  stageManifest.BasePlanRevisionHash,
		ColonyMode:         string(state.EffectiveColonyMode()),
		ExistingPlan:       len(state.Plan.Phases) > 0,
		ExistingPhaseCount: len(state.Plan.Phases),
		Synthetic:          opts.Synthetic,
		SyntheticWarning:   planningSyntheticWarningForMode(opts.Synthetic),
		PlanningRunID:      stageManifest.RunID,
		Iteration:          stageManifest.Pass,
		SelectedPreset:     stageManifest.Preset,
		Depth:              planDepth,
		Granularity:        string(granularity),
		GranularityMin:     granularityMin(granularity),
		GranularityMax:     granularityMax(granularity),
		PlanningDepth:      planningDepth,
		VerificationDepth:  verificationDepth,
		Survey:             survey,
		Dispatches:         dispatches,
		ExpectedWorkers:    append([]codexPlanningDispatch{}, dispatches...),
		DispatchMode:       "plan-only",
		DispatchContract:   planningScoutStageDispatchContract(dispatches, opts.WorkerTimeout),
		FinalizeSurface:    "pending",
		RequiresFinalizer:  true,
		StageManifest:      &stageManifest,
		ContextCapsule:     contextCapsule,
	}
	return manifest, nil
}

// planningStageResumeEnvelope is the plan-only envelope for a resumed stage.
//
// It carries the same plan_manifest/stage_manifest keys a fresh Scout run
// emits, so a wrapper dispatches a resumed Route-Setter through exactly the
// path it already uses for a Scout -- no second wrapper branch, no second
// manifest shape. `planned` is false because a stage dispatch is not a finished
// plan; `resumed_run` is what tells the wrapper this advanced an existing run
// rather than starting one.
func planningStageResumeEnvelope(state colony.ColonyState, resume planningStageResume, manifest codexPlanManifest, granularity colony.PlanGranularity, planDepth, planningDepth, verificationDepth string) map[string]interface{} {
	return map[string]interface{}{
		"plan_only":          true,
		"planned":            false,
		"resumed_run":        true,
		"existing_plan":      len(state.Plan.Phases) > 0,
		"colony_mode":        string(state.EffectiveColonyMode()),
		"goal":               strings.TrimSpace(*state.Goal),
		"depth":              planDepth,
		"planning_depth":     planningDepth,
		"verification_depth": verificationDepth,
		"granularity":        string(granularity),
		"planning_run_id":    resume.RunID,
		"iteration":          resume.Pass,
		"stage":              string(resume.Stage),
		"next_boundary":      string(resume.Stage),
		"expected_caste":     string(resume.Caste),
		"selected_preset":    string(resume.Manifest.Preset),
		"plan_manifest":      manifest,
		"planning_manifest":  manifest,
		"stage_manifest":     resume.Manifest,
		"dispatches":         manifest.Dispatches,
		"requires_finalizer": true,
		"next":               resume.nextCommandHint(),
	}
}

// planningStageResumeWorkerSpec resolves a stage caste to its worker
// definition. The two core planning castes (scout, route_setter) live in the
// canonical planningWorkerSpecs list, while the optional advisory castes live
// in planningWorkerSpecForCaste's switch; a resume can be for either, so this
// checks the canonical list first and falls back to the switch.
func planningStageResumeWorkerSpec(caste string) (planningWorkerSpec, bool) {
	for _, spec := range planningWorkerSpecsForGoal("") {
		if strings.EqualFold(spec.Caste, caste) {
			return spec, true
		}
	}
	return planningWorkerSpecForCaste(caste)
}

// discoverParkedPlanningRun finds the one planning run waiting on a worker when
// no iteration-state pointer names it.
//
// It never guesses: a run only counts when its own durable stage file says it
// is waiting on a worker, and finding more than one such run returns nothing
// rather than picking a favourite. Starting the wrong run is precisely the
// stray-run damage this whole path exists to prevent, so ambiguity falls back
// to the ordinary "start a run" behaviour the caller already has.
func discoverParkedPlanningRun(session *planningMutationSession) (string, error) {
	// The session's own data root, not the global store: the session IS the
	// authority here, and depending on a package-level store made discovery
	// silently do nothing whenever that global was not bound.
	entries, err := os.ReadDir(filepath.Join(session.DataRoot(), "planning"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	parked := make([]string, 0, 2)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := entry.Name()
		state, err := loadPlanningStageStateInSession(session, candidate)
		if err != nil {
			// A directory without a readable stage file is not a parked run.
			continue
		}
		if strings.TrimSpace(state.RunID) == "" {
			continue
		}
		if planningStageAwaitsAWorker(session, state) {
			parked = append(parked, candidate)
		}
	}
	if len(parked) != 1 {
		return "", nil
	}
	return parked[0], nil
}

// planningStageAwaitsAWorker reports whether a run's own durable stage state
// says it is genuinely waiting on a worker that was authorized for it.
//
// route_running always qualifies: reaching it means a Scout pass committed and
// authorized Route-Setter.
//
// scout_running qualifies ONLY when a Route-Setter pass authorized that Scout,
// which is exactly when a scout-authorizations/pass-NNNN.json exists for it.
// That distinction is structural rather than a heuristic, and it is what keeps
// abandoned first-pass runs out: a run whose Scout stage was merely opened by
// `aether plan` has no such authorization on disk, so it is never mistaken for
// work in progress. The real colony that reported this had one genuine run at
// scout_running pass 2 carrying scout-authorizations/pass-0002.json, beside two
// abandoned pass-1 runs holding nothing but a manifest and a run header;
// preferring "pass > 1" would have worked on that colony by luck, while this
// asks the question that actually matters.
func planningStageAwaitsAWorker(session *planningMutationSession, state planningStageState) bool {
	switch state.Stage {
	case planningStageRouteRunning:
		return true
	case planningStageScoutRunning:
		_, err := loadPlanningRouteScoutDispatchInSession(session, state.RunID, state.Pass)
		return err == nil
	default:
		return false
	}
}
