package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Test 1: "Settings UI panel" -> Builder + Watcher only
func TestQueenOrchestrate_SettingsUI(t *testing.T) {
	phase := colony.Phase{
		ID:          1,
		Name:        "Settings UI panel",
		Description: "Build a settings panel for user preferences",
		Mode:        colony.PhaseModePrototype,
		Tasks: []colony.Task{
			{Goal: "Implement SettingsPanel component with form controls"},
		},
	}

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Settings UI: expected builder")
	}
	// Plan 194-02 (D-07): watcher is no longer required at build -- Phase 193
	// (D-08) already stopped dispatching it there, and this floor shrink
	// removes the requirement that used to restore it regardless.
	if HasCaste(dispatches, "gatekeeper") {
		t.Error("Settings UI: should NOT spawn gatekeeper for UI work")
	}
	if HasCaste(dispatches, "oracle") {
		t.Error("Settings UI: should NOT spawn oracle for implementation")
	}
	if HasCaste(dispatches, "chaos") {
		t.Error("Settings UI: should NOT spawn chaos for UI work")
	}
	if HasCaste(dispatches, "measurer") {
		t.Error("Settings UI: should NOT spawn measurer for UI work")
	}
}

// Test 2: "Auth token rotation" -> Builder + Watcher + Gatekeeper + Probe + Architect
//
// Plan 194-05 (D-11): queenOrchestrate's no-proposal path on build no longer
// goes through the relevance/keyword scoring engine at all -- it answers
// with the required-caste floor only (queenFallbackTeam). This test's real
// subject is the scoring registry (which caste a phase's own wording earns),
// so it now calls queenCandidateDispatches directly -- the scoring function
// is unchanged by this plan; only queenOrchestrate's use of it for build was
// gated. See TestKeywordEngineNoLongerSelectsOnBuildOrContinue
// (cmd/queen_fallback_team_test.go) for the entry-point behaviour itself.
func TestQueenOrchestrate_AuthToken(t *testing.T) {
	phase := colony.Phase{
		ID:          2,
		Name:        "Auth token rotation",
		Description: "Implement secure token refresh and rotation",
		Mode:        colony.PhaseModeProduction,
		Tasks: []colony.Task{
			{Goal: "Implement token rotation endpoint with crypto"},
		},
	}

	dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Auth token: expected builder")
	}
	// Plan 194-02 (D-07): watcher and probe are no longer required at build;
	// gatekeeper here is a genuine relevance-score hit on this phase's own
	// security wording, not the deleted always-required floor.
	if !HasCaste(dispatches, "gatekeeper") {
		t.Error("Auth token: expected gatekeeper for security work")
	}
	if !HasCaste(dispatches, "architect") {
		t.Error("Auth token: expected architect for design boundaries")
	}
	// The "production mode ⇒ auditor" assertion that used to sit here is
	// gone (D-06/Ruling D11): this phase's own wording names no auditor
	// keyword (no "compliance", "audit", "release", "standards"), so an
	// auditor appearing here would be the exact implicit floor this plan
	// removes, restated as a test expectation. See
	// TestQueenOrchestrate_DBMigration for the matching fix.
	if HasCaste(dispatches, "auditor") {
		t.Error("Auth token: auditor must not be summoned by production mode alone with no auditor-relevant wording")
	}
}

// Test 3: "Database migration" -> Builder + Watcher + Auditor + Architect
//
// See TestQueenOrchestrate_AuthToken's comment: this exercises the scoring
// registry directly (queenCandidateDispatches), since queenOrchestrate's
// no-proposal build path no longer runs it (plan 194-05, D-11).
func TestQueenOrchestrate_DBMigration(t *testing.T) {
	phase := colony.Phase{
		ID:          3,
		Name:        "Database migration",
		Description: "Migrate user data to new schema with integrity checks",
		Mode:        colony.PhaseModeProduction,
		Tasks: []colony.Task{
			{Goal: "Write migration scripts and schema updates"},
		},
	}

	dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("DB migration: expected builder")
	}
	if !HasCaste(dispatches, "watcher") {
		t.Error("DB migration: expected watcher")
	}
	// Same fix as TestQueenOrchestrate_AuthToken: this phase's wording names
	// no auditor keyword either, so "production mode ⇒ auditor" (D-06,
	// Ruling D11) must not summon one here.
	if HasCaste(dispatches, "auditor") {
		t.Error("DB migration: auditor must not be summoned by production mode alone with no auditor-relevant wording")
	}
	if !HasCaste(dispatches, "architect") {
		t.Error("DB migration: expected architect for schema design")
	}
}

// Test 4: "Performance optimization" -> Builder + Watcher + Measurer + Probe
//
// See TestQueenOrchestrate_AuthToken's comment: this exercises the scoring
// registry directly (queenCandidateDispatches), since queenOrchestrate's
// no-proposal build path no longer runs it (plan 194-05, D-11).
func TestQueenOrchestrate_Performance(t *testing.T) {
	phase := colony.Phase{
		ID:          4,
		Name:        "Performance optimization",
		Description: "Optimize query latency and reduce memory usage",
		Mode:        colony.PhaseModePrototype,
		Tasks: []colony.Task{
			{Goal: "Benchmark and optimize slow queries"},
		},
	}

	dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Performance: expected builder")
	}
	// Plan 194-02 (D-07): watcher and probe are no longer required at build.
	if !HasCaste(dispatches, "measurer") {
		t.Error("Performance: expected measurer")
	}
}

// Test 5: "Refactor legacy parser" -> Weaver + Archaeologist + Builder + Watcher
//
// See TestQueenOrchestrate_AuthToken's comment: this exercises the scoring
// registry directly (queenCandidateDispatches), since queenOrchestrate's
// no-proposal build path no longer runs it (plan 194-05, D-11).
func TestQueenOrchestrate_RefactorLegacy(t *testing.T) {
	phase := colony.Phase{
		ID:          5,
		Name:        "Refactor legacy parser",
		Description: "Modernize the old parser and remove deprecated patterns",
		Mode:        colony.PhaseModeMaintenance,
		Tasks: []colony.Task{
			{Goal: "Refactor legacy parser to use modern patterns"},
		},
	}

	dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Refactor: expected builder")
	}
	// Plan 194-02 (D-07): watcher is no longer required at build.
	if !HasCaste(dispatches, "weaver") {
		t.Error("Refactor: expected weaver for restructuring")
	}
	if !HasCaste(dispatches, "archaeologist") {
		t.Error("Refactor: expected archaeologist for legacy analysis")
	}
}

// Test 6: "Discovery spike on vector DB" -> Oracle + Scout + Architect (no Builder, no Watcher)
//
// See TestQueenOrchestrate_AuthToken's comment: this exercises the scoring
// registry directly (queenCandidateDispatches), since queenOrchestrate's
// no-proposal build path no longer runs it (plan 194-05, D-11). The
// no-proposal ENTRY POINT's own discovery behaviour (one scout, D-12) is
// covered separately by TestDiscoveryFallbackSendsOneResearcher
// (cmd/queen_fallback_team_test.go).
func TestQueenOrchestrate_DiscoverySpike(t *testing.T) {
	phase := colony.Phase{
		ID:          6,
		Name:        "Discovery spike on vector DB",
		Description: "Research vector database options and evaluate fit",
		Mode:        colony.PhaseModeDiscovery,
		Tasks: []colony.Task{
			{Goal: "Research and evaluate vector database solutions"},
		},
	}

	dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "oracle") {
		t.Error("Discovery: expected oracle")
	}
	if !HasCaste(dispatches, "scout") {
		t.Error("Discovery: expected scout for research")
	}
	if !HasCaste(dispatches, "architect") {
		t.Error("Discovery: expected architect for design evaluation")
	}
	if HasCaste(dispatches, "builder") {
		t.Error("Discovery: should NOT spawn builder for research phase")
	}
	if HasCaste(dispatches, "chaos") {
		t.Error("Discovery: should NOT spawn chaos for discovery")
	}
}

// Test 7: Verify score thresholds work correctly
func TestCasteRelevanceScore_Thresholds(t *testing.T) {
	phase := colony.Phase{
		Name:        "Security audit and compliance review",
		Description: "Audit auth system for compliance requirements",
		Mode:        colony.PhaseModeProduction,
		Tasks:       []colony.Task{{Goal: "Audit security compliance"}},
	}

	// Gatekeeper should score high for security keywords. The >= 80 floor
	// this test used to assert came from the deleted "risk high ⇒ 100"
	// special rule (D-06); the genuine keyword-driven score for this
	// wording (auth, security, compliance, audit all hit) is 60, and that is
	// now the ceiling this fixture can honestly reach without the deleted
	// implicit floor.
	score := casteRelevanceScore(phase, "gatekeeper")
	if score < 50 {
		t.Errorf("Gatekeeper score too low for security phase: got %d, want >= 50", score)
	}

	// Dreamer should score low for security phase
	score = casteRelevanceScore(phase, "dreamer")
	if score > 30 {
		t.Errorf("Dreamer score too high for security phase: got %d, want <= 30", score)
	}
}

// Test 8: Verify chaos is excluded for discovery mode
func TestChaos_ExcludedForDiscovery(t *testing.T) {
	phase := colony.Phase{
		Name:  "Discovery spike",
		Mode:  colony.PhaseModeDiscovery,
		Tasks: []colony.Task{{Goal: "Research new technology"}},
	}

	score := casteRelevanceScore(phase, "chaos")
	if score != 0 {
		t.Errorf("Chaos should be 0 for discovery mode, got %d", score)
	}
}

// Test 9: Verify oracle is auto-included for discovery mode
func TestOracle_AutoIncludedForDiscovery(t *testing.T) {
	phase := colony.Phase{
		Name:  "Discovery spike",
		Mode:  colony.PhaseModeDiscovery,
		Tasks: []colony.Task{{Goal: "Research new technology"}},
	}

	score := casteRelevanceScore(phase, "oracle")
	if score != 100 {
		t.Errorf("Oracle should be 100 for discovery mode, got %d", score)
	}
}

// Test 10: Verify builder is always included for implementation tasks
func TestBuilder_AlwaysForImplementation(t *testing.T) {
	phase := colony.Phase{
		Name:  "Simple task",
		Mode:  colony.PhaseModePrototype,
		Tasks: []colony.Task{{Goal: "Implement the feature"}},
	}

	score := casteRelevanceScore(phase, "builder")
	if score != 100 {
		t.Errorf("Builder should be 100 for implementation tasks, got %d", score)
	}
}

// Test 11: Verify continue flow includes gatekeeper for security
//
// Plan 194-05 (D-13) removed Watcher's and Probe's unconditional continue
// membership at light/standard depth -- neither is forced any more, and
// neither's own keyword list matches this fixture's wording, so those two
// assertions are gone. Gatekeeper still appears: "token" is one of its own
// keywords, a genuine relevance-score hit, not the deleted floor. This test
// calls queenCandidateDispatches directly (the scoring engine, unaffected by
// D-11's gate on queenOrchestrate's no-proposal entry point).
func TestQueenOrchestrate_ContinueFlow(t *testing.T) {
	phase := colony.Phase{
		Name:        "Auth system implementation",
		Description: "Implement OAuth2 flow with token management",
		Mode:        colony.PhaseModePrototype,
		Tasks:       []colony.Task{{Goal: "Implement OAuth flow"}},
	}

	dispatches := queenCandidateDispatches(phase, "continue", colony.ColonyState{})

	if !HasCaste(dispatches, "gatekeeper") {
		t.Error("Continue: expected gatekeeper for auth phase")
	}
}

// Test 12: Verify plan flow includes scout and route_setter
func TestQueenOrchestrate_PlanFlow(t *testing.T) {
	phase := colony.Phase{
		Name:  "New feature planning",
		Mode:  colony.PhaseModePrototype,
		Tasks: []colony.Task{{Goal: "Plan the implementation"}},
	}

	dispatches := queenOrchestrate(phase, "plan", colony.ColonyState{})

	if !HasCaste(dispatches, "scout") {
		t.Error("Plan: expected scout")
	}
	if !HasCaste(dispatches, "route_setter") {
		t.Error("Plan: expected route_setter")
	}
}

// TestQueenOrchestrate_DiscoveryBuildSuppressesImplementationCastes exercises
// the scoring registry directly (queenCandidateDispatches), since
// queenOrchestrate's no-proposal build path no longer runs it (plan 194-05,
// D-11). The no-proposal ENTRY POINT's own discovery behaviour (one scout)
// is covered separately by TestDiscoveryFallbackSendsOneResearcher
// (cmd/queen_fallback_team_test.go).
func TestQueenOrchestrate_DiscoveryBuildSuppressesImplementationCastes(t *testing.T) {
	phase := colony.Phase{
		Name:        "Discovery spike on cache strategy",
		Description: "Research options and build only a written recommendation",
		Mode:        colony.PhaseModeDiscovery,
		Tasks: []colony.Task{
			{Goal: "Build a comparison matrix before implementation"},
		},
	}

	dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})

	if HasCaste(dispatches, "builder") {
		t.Error("Discovery build: should suppress builder even when implementation keywords appear")
	}
	if !HasCaste(dispatches, "oracle") {
		t.Error("Discovery build: expected oracle")
	}
	if !HasCaste(dispatches, "scout") {
		t.Error("Discovery build: expected scout")
	}
}

func TestQueenOrchestrate_ColonizeUsesConcreteSurveyors(t *testing.T) {
	phase := colony.Phase{
		Name:        "Colonize repository",
		Description: "Survey architecture, provisions, disciplines, and pathogens",
		Mode:        colony.PhaseModeDiscovery,
	}

	dispatches := queenOrchestrate(phase, "colonize", colony.ColonyState{})

	for _, caste := range []string{"surveyor-provisions", "surveyor-nest", "surveyor-disciplines", "surveyor-pathogens"} {
		if !HasCaste(dispatches, caste) {
			t.Errorf("Colonize: expected %s", caste)
		}
	}
	if HasCaste(dispatches, "surveyor") {
		t.Error("Colonize: should not dispatch the virtual surveyor caste")
	}
}

func TestQueenOrchestrate_SwarmUsesInvestigationAndFixCastes(t *testing.T) {
	phase := colony.Phase{
		Name:        "Swarm parser regression",
		Description: "Investigate a failing parser bug and fix the regression",
		Mode:        colony.PhaseModeMaintenance,
		Tasks: []colony.Task{
			{Goal: "Fix the parser bug and add regression tests"},
		},
	}

	dispatches := queenOrchestrate(phase, "swarm", colony.ColonyState{})

	for _, caste := range []string{"tracker", "builder", "watcher"} {
		if !HasCaste(dispatches, caste) {
			t.Errorf("Swarm: expected %s", caste)
		}
	}
	// "Investigate a failing parser bug" is investigation wording, so Scout is
	// selected on relevance — the mandatory floor is only the
	// investigate/fix/verify trio (see TestSwarmTrivialBugSkipsHistoryAndResearch).
	if !HasCaste(dispatches, "scout") {
		t.Error("Swarm: investigation wording should select scout via relevance")
	}
	// Nothing in this phase names legacy code or git history, so the
	// Archaeologist stays home.
	if HasCaste(dispatches, "archaeologist") {
		t.Error("Swarm: archaeologist selected with no history signal in the phase")
	}
}

// TestQueenOrchestrate_ContinueHeavyIncludesReviewGates exercises the
// scoring registry directly (queenCandidateDispatches). Heavy continue's
// required set (gatekeeper, auditor, probe-if-testable, plan 194-05 D-13) is
// unaffected by D-11's gate -- isAlwaysRequired is consulted by
// queenCandidateDispatches exactly as before, only queenOrchestrate's
// no-proposal ENTRY POINT changed. Watcher is no longer part of heavy's
// unconditional panel (194-05, D-13) and this fixture's wording does not
// clear its own keyword threshold, so it is not asserted here.
func TestQueenOrchestrate_ContinueHeavyIncludesReviewGates(t *testing.T) {
	phase := colony.Phase{
		Name: "Phase verification",
		Mode: colony.PhaseModeProduction,
	}
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}

	dispatches := queenCandidateDispatches(phase, "continue", state)

	for _, caste := range []string{"gatekeeper", "auditor", "probe"} {
		if !HasCaste(dispatches, caste) {
			t.Errorf("Heavy continue: expected %s", caste)
		}
	}
	if HasCaste(dispatches, "builder") {
		t.Error("Heavy continue: should not dispatch builder")
	}
}

func TestQueenOrchestrate_SealLightSkipsReviewGates(t *testing.T) {
	phase := colony.Phase{
		Name:        "Production release",
		Description: "Release and ship final artifacts",
		Mode:        colony.PhaseModeProduction,
	}
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}

	dispatches := queenOrchestrate(phase, "seal", state)

	for _, caste := range []string{"gatekeeper", "auditor", "probe"} {
		if HasCaste(dispatches, caste) {
			t.Errorf("Light seal: should not dispatch %s", caste)
		}
	}
}

func TestQueenOrchestrate_EmptyFlowDefaultsToBuild(t *testing.T) {
	phase := colony.Phase{
		Name: "Implementation phase",
		Mode: colony.PhaseModePrototype,
		Tasks: []colony.Task{
			{Goal: "Implement the feature"},
		},
	}

	dispatches := queenOrchestrate(phase, "", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Default flow: expected builder")
	}
	for _, dispatch := range dispatches {
		if dispatch.FlowType != "build" {
			t.Errorf("Default flow: dispatch flow type = %q, want build", dispatch.FlowType)
		}
	}
}

// TestQueenOrchestrateAppliesAdaptiveSpawnBudget tests the SCORING-PLUS-
// BUDGET path (applyQueenSpawnBudget over queenCandidateDispatches), not the
// no-proposal ENTRY POINT: queenOrchestrate's build path (queenFallbackTeam,
// plan 194-05, D-11) no longer scores anything, so it cannot exercise a
// budget-pruning claim any more -- there is nothing left to prune once the
// only candidate is the required builder. The claim this test protects
// (budget trims optional picks while a documentation phase still never
// summons a Probe) still lives in the scoring-plus-budget path, called here
// explicitly.
func TestQueenOrchestrateAppliesAdaptiveSpawnBudget(t *testing.T) {
	phase := colony.Phase{
		ID:          13,
		Name:        "Documentation maintenance",
		Description: "Update the architecture guide, README, changelog, manual, project standards, and quality review notes.",
		Mode:        colony.PhaseModeMaintenance,
		Tasks: []colony.Task{
			{Goal: "Write documentation, organize structure, preserve knowledge patterns, and validate examples."},
		},
	}

	dispatches := applyQueenSpawnBudget(queenCandidateDispatches(phase, "build", colony.ColonyState{}), phase, "build", colony.ColonyState{})

	// Builder survives pruning: something has to do the work. Watcher is no
	// longer a required build caste (194-02, D-06/D-07) -- it may or may not
	// appear on relevance score alone; this test does not assert it either
	// way.
	if !HasCaste(dispatches, "builder") {
		t.Error("Low-risk docs: expected required build caste builder to survive budget pruning")
	}

	// Probe is deliberately absent. This phase writes documentation — there is
	// no new code for a test-coverage specialist to cover, so spawning one
	// spends a worker run to report nothing. Probe used to be unconditionally
	// required here, and required castes bypass the budget, which is why a
	// documentation phase summoned a coverage specialist no matter what depth
	// the operator asked for. See TestProbeIsRequiredOnlyWhereItCanFindSomething.
	if HasCaste(dispatches, "probe") {
		t.Errorf("Low-risk docs: Probe should not spawn on a documentation-only phase: %+v", dispatches)
	}

	const queenBudget = 4
	if len(dispatches) > queenBudget {
		t.Fatalf("Low-risk docs: selected %d castes, want <= Queen budget %d: %+v", len(dispatches), queenBudget, dispatches)
	}
}

func TestQueenSpawnBudgetAcceptanceContractIsCasteLevel(t *testing.T) {
	phase := colony.Phase{
		ID:          14,
		Name:        "Failure Contract Reproduction",
		Description: "Pin stale-boundary and missed worker-result contracts before changing runtime behavior.",
		Mode:        colony.PhaseModePrototype,
		Tasks: []colony.Task{
			{Goal: "Map pending-decision source and session fields"},
			{Goal: "Reproduce stale clarification masking"},
			{Goal: "Reproduce missed worker result collection"},
			{Goal: "Define lean-spawn recovery acceptance"},
		},
	}

	budget := queenSpawnBudgetForPhase(phase, "build", colony.ColonyState{})
	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

	// Plan 194-02 (D-07): the required-caste floor shrank to the builder
	// alone (watcher and probe are no longer forced), so the acceptance
	// contract this test guards -- caste-level budget enforcement, not
	// worker-count enforcement -- is now pinned on builder alone.
	if !HasCaste(dispatches, "builder") {
		t.Fatalf("failure contract phase should keep required build caste builder")
	}
	if len(dispatches) > budget.MaxWorkers {
		t.Fatalf("selected caste count = %d, want <= Queen budget %d: %+v", len(dispatches), budget.MaxWorkers, dispatches)
	}
}

func TestQueenSpawnBudgetDecisionsPrunesDeterministically(t *testing.T) {
	tests := []struct {
		name         string
		dispatches   []CasteDispatch
		budget       queenSpawnBudget
		wantOrder    []string
		wantSelected []string
	}{
		{
			name: "required before higher-scoring optional castes and optional ties by caste",
			dispatches: []CasteDispatch{
				{Caste: "chaos", Score: 90},
				{Caste: "watcher", Score: 10},
				{Caste: "auditor", Score: 90},
				{Caste: "builder", Score: 5},
				{Caste: "ambassador", Score: 90},
				{Caste: "measurer", Score: 90},
			},
			budget: queenSpawnBudget{
				FlowType:       "build",
				MaxWorkers:     4,
				RequiredCastes: []string{"builder", "watcher"},
				Reason:         "test budget pressure",
			},
			wantOrder:    []string{"watcher", "builder", "ambassador", "auditor", "chaos", "measurer"},
			wantSelected: []string{"watcher", "builder", "ambassador", "auditor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisions := queenSpawnBudgetDecisions(tt.dispatches, tt.budget)

			if got := budgetDecisionCastes(decisions); !reflect.DeepEqual(got, tt.wantOrder) {
				t.Fatalf("decision order = %v, want %v", got, tt.wantOrder)
			}
			if got := selectedBudgetDecisionCastes(decisions); !reflect.DeepEqual(got, tt.wantSelected) {
				t.Fatalf("selected castes = %v, want %v", got, tt.wantSelected)
			}
			decision, ok := budgetDecisionForCaste(decisions, "chaos")
			if !ok {
				t.Fatal("missing pruned chaos decision")
			}
			if decision.Selected {
				t.Fatalf("chaos decision should be pruned: %+v", decision)
			}
			// D-09: the pruned rationale is plain English -- no internal
			// "Queen spawn budget" jargon, and it must still say the pick was
			// not sent and name the phase's worker cap.
			for _, want := range []string{"not sent", "capped at"} {
				if !strings.Contains(decision.Rationale, want) {
					t.Fatalf("pruned rationale missing %q: %q", want, decision.Rationale)
				}
			}
			if strings.Contains(decision.Rationale, "Queen spawn budget") {
				t.Fatalf("pruned rationale still names the internal 'Queen spawn budget' identifier: %q", decision.Rationale)
			}
		})
	}
}

func TestQueenSpawnBudgetDecisionsKeepsRequiredOverflow(t *testing.T) {
	budget := queenSpawnBudget{
		FlowType:       "build",
		MaxWorkers:     2,
		RequiredCastes: []string{"auditor", "gatekeeper", "probe"},
		Reason:         "required overflow",
	}
	dispatches := []CasteDispatch{
		{Caste: "measurer", Score: 100},
		{Caste: "auditor", Score: 20},
		{Caste: "gatekeeper", Score: 20},
		{Caste: "probe", Score: 20},
		{Caste: "keeper", Score: 90},
	}

	decisions := queenSpawnBudgetDecisions(dispatches, budget)

	wantSelected := []string{"auditor", "gatekeeper", "probe"}
	if got := selectedBudgetDecisionCastes(decisions); !reflect.DeepEqual(got, wantSelected) {
		t.Fatalf("selected castes = %v, want required castes despite max workers %d", got, budget.MaxWorkers)
	}
	for _, caste := range wantSelected {
		decision, ok := budgetDecisionForCaste(decisions, caste)
		if !ok {
			t.Fatalf("missing decision for required caste %s", caste)
		}
		if !decision.Required || !decision.Selected {
			t.Fatalf("%s decision = %+v, want required and selected", caste, decision)
		}
	}
}

func budgetDecisionCastes(decisions []queenSpawnBudgetDecision) []string {
	castes := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		castes = append(castes, decision.Caste)
	}
	return castes
}

func selectedBudgetDecisionCastes(decisions []queenSpawnBudgetDecision) []string {
	castes := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		if decision.Selected {
			castes = append(castes, decision.Caste)
		}
	}
	return castes
}

func budgetDecisionForCaste(decisions []queenSpawnBudgetDecision, caste string) (queenSpawnBudgetDecision, bool) {
	for _, decision := range decisions {
		if decision.Caste == caste {
			return decision, true
		}
	}
	return queenSpawnBudgetDecision{}, false
}

func TestFilterCastesByMinScore(t *testing.T) {
	dispatches := []CasteDispatch{
		{Caste: "builder", Score: 100},
		{Caste: "keeper", Score: 25},
		{Caste: "watcher", Score: 70},
	}

	filtered := FilterCastesByMinScore(dispatches, 70)

	if len(filtered) != 2 {
		t.Fatalf("filtered len = %d, want 2", len(filtered))
	}
	if filtered[0].Caste != "builder" || filtered[1].Caste != "watcher" {
		t.Fatalf("filtered castes = %+v, want builder and watcher in original order", filtered)
	}
}

// TestChroniclerOwnsTheWordDocument: the keeper trim moved "document" away
// from keeper on the stated grounds that it "belongs to chronicler", but
// chronicler's keyword was "documentation" and keyword matching anchors only
// on the left boundary — so "documentation" never matched the word
// "document" and neither caste owned it.
func TestChroniclerOwnsTheWordDocument(t *testing.T) {
	phase := colony.Phase{Name: "Document the export module"}
	if !containsKeyword(collectPhaseText(phase), "document") {
		t.Fatal("control failed: the phase text does not contain the word document")
	}
	chronicler := casteRelevanceScore(phase, "chronicler")
	base := casteRelevanceScore(colony.Phase{Name: "Add a CSV export"}, "chronicler")
	if chronicler <= base {
		t.Fatalf("a phase asking to document something must score chronicler above its base (%d), got %d", base, chronicler)
	}
	if keeper := casteRelevanceScore(phase, "keeper"); keeper > casteRelevanceScore(colony.Phase{Name: "Add a CSV export"}, "keeper") {
		t.Fatal("keeper must not score on the word document -- it keys on preservation intent")
	}
}
