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
	if !HasCaste(dispatches, "watcher") {
		t.Error("Settings UI: expected watcher")
	}
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

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Auth token: expected builder")
	}
	if !HasCaste(dispatches, "watcher") {
		t.Error("Auth token: expected watcher")
	}
	if !HasCaste(dispatches, "gatekeeper") {
		t.Error("Auth token: expected gatekeeper for security work")
	}
	if !HasCaste(dispatches, "probe") {
		t.Error("Auth token: expected probe for verification")
	}
	if !HasCaste(dispatches, "architect") {
		t.Error("Auth token: expected architect for design boundaries")
	}
	if !HasCaste(dispatches, "auditor") {
		t.Error("Auth token: expected auditor for production mode")
	}
}

// Test 3: "Database migration" -> Builder + Watcher + Auditor + Architect
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

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("DB migration: expected builder")
	}
	if !HasCaste(dispatches, "watcher") {
		t.Error("DB migration: expected watcher")
	}
	if !HasCaste(dispatches, "auditor") {
		t.Error("DB migration: expected auditor for production")
	}
	if !HasCaste(dispatches, "architect") {
		t.Error("DB migration: expected architect for schema design")
	}
}

// Test 4: "Performance optimization" -> Builder + Watcher + Measurer + Probe
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

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Performance: expected builder")
	}
	if !HasCaste(dispatches, "watcher") {
		t.Error("Performance: expected watcher")
	}
	if !HasCaste(dispatches, "measurer") {
		t.Error("Performance: expected measurer")
	}
	if !HasCaste(dispatches, "probe") {
		t.Error("Performance: expected probe")
	}
}

// Test 5: "Refactor legacy parser" -> Weaver + Archaeologist + Builder + Watcher
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

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

	if !HasCaste(dispatches, "builder") {
		t.Error("Refactor: expected builder")
	}
	if !HasCaste(dispatches, "watcher") {
		t.Error("Refactor: expected watcher")
	}
	if !HasCaste(dispatches, "weaver") {
		t.Error("Refactor: expected weaver for restructuring")
	}
	if !HasCaste(dispatches, "archaeologist") {
		t.Error("Refactor: expected archaeologist for legacy analysis")
	}
}

// Test 6: "Discovery spike on vector DB" -> Oracle + Scout + Architect (no Builder, no Watcher)
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

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

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

	// Gatekeeper should score high for security keywords
	score := casteRelevanceScore(phase, "gatekeeper")
	if score < 80 {
		t.Errorf("Gatekeeper score too low for security phase: got %d, want >= 80", score)
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
func TestQueenOrchestrate_ContinueFlow(t *testing.T) {
	phase := colony.Phase{
		Name:        "Auth system implementation",
		Description: "Implement OAuth2 flow with token management",
		Mode:        colony.PhaseModePrototype,
		Tasks:       []colony.Task{{Goal: "Implement OAuth flow"}},
	}

	dispatches := queenOrchestrate(phase, "continue", colony.ColonyState{})

	if !HasCaste(dispatches, "watcher") {
		t.Error("Continue: expected watcher")
	}
	if !HasCaste(dispatches, "gatekeeper") {
		t.Error("Continue: expected gatekeeper for auth phase")
	}
	if !HasCaste(dispatches, "probe") {
		t.Error("Continue: expected probe")
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

func TestQueenOrchestrate_DiscoveryBuildSuppressesImplementationCastes(t *testing.T) {
	phase := colony.Phase{
		Name:        "Discovery spike on cache strategy",
		Description: "Research options and build only a written recommendation",
		Mode:        colony.PhaseModeDiscovery,
		Tasks: []colony.Task{
			{Goal: "Build a comparison matrix before implementation"},
		},
	}

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

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

	for _, caste := range []string{"tracker", "scout", "archaeologist", "builder", "watcher"} {
		if !HasCaste(dispatches, caste) {
			t.Errorf("Swarm: expected %s", caste)
		}
	}
}

func TestQueenOrchestrate_ContinueHeavyIncludesReviewGates(t *testing.T) {
	phase := colony.Phase{
		Name: "Phase verification",
		Mode: colony.PhaseModeProduction,
	}
	state := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}

	dispatches := queenOrchestrate(phase, "continue", state)

	for _, caste := range []string{"watcher", "gatekeeper", "auditor", "probe"} {
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

func TestQueenOrchestratePreservesSafetyCastes(t *testing.T) {
	safetyCastes := []string{"builder", "watcher", "probe", "gatekeeper", "auditor"}
	tests := []struct {
		name                string
		phase               colony.Phase
		belowThresholdCaste string
	}{
		{
			name: "security",
			phase: colony.Phase{
				Name:        "Security hardening",
				Description: "Protect privileged configuration before production rollout",
				Mode:        colony.PhaseModeProduction,
				Tasks: []colony.Task{
					{Goal: "Build hardened configuration checks"},
				},
			},
		},
		{
			name: "release",
			phase: colony.Phase{
				Name:        "Release candidate packaging",
				Description: "Prepare the candidate for ship readiness",
				Mode:        colony.PhaseModeProduction,
				Tasks: []colony.Task{
					{Goal: "Build release candidate artifacts"},
				},
			},
			belowThresholdCaste: "gatekeeper",
		},
		{
			name: "final-review",
			phase: colony.Phase{
				Name:        "Final review",
				Description: "Complete final signoff before handoff",
				Mode:        colony.PhaseModeProduction,
				Tasks: []colony.Task{
					{Goal: "Build final review evidence and address blockers"},
				},
			},
			belowThresholdCaste: "gatekeeper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := colony.ColonyState{}
			if tt.belowThresholdCaste != "" {
				score := casteRelevanceScore(tt.phase, tt.belowThresholdCaste)
				threshold := spawnThreshold("build", state)
				if score >= threshold {
					t.Fatalf("%s fixture %s score = %d, want below build threshold %d so safety is not inferred from score",
						tt.name, tt.belowThresholdCaste, score, threshold)
				}
			}

			dispatches := queenOrchestrate(tt.phase, "build", state)

			for _, caste := range safetyCastes {
				if !HasCaste(dispatches, caste) {
					t.Errorf("%s: expected safety caste %s to survive Queen orchestration", tt.name, caste)
				}
			}
		})
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

	dispatches := queenOrchestrate(phase, "build", colony.ColonyState{})

	// Builder and Watcher survive pruning: something has to do the work, and
	// something has to check it.
	for _, caste := range []string{"builder", "watcher"} {
		if !HasCaste(dispatches, caste) {
			t.Errorf("Low-risk docs: expected required build caste %s to survive budget pruning", caste)
		}
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

	for _, caste := range []string{"builder", "watcher", "probe"} {
		if !HasCaste(dispatches, caste) {
			t.Fatalf("failure contract phase should keep required build caste %s", caste)
		}
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
			for _, want := range []string{"not spawned", "Queen spawn budget"} {
				if !strings.Contains(decision.Rationale, want) {
					t.Fatalf("pruned rationale missing %q: %q", want, decision.Rationale)
				}
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

func TestQueenSpawnBudgetDecisionsPreservesSafetyCastesUnderPressure(t *testing.T) {
	tests := []struct {
		name  string
		phase colony.Phase
	}{
		{
			name: "security",
			phase: colony.Phase{
				Name:        "Security hardening",
				Description: "Protect privileged configuration before rollout",
				Mode:        colony.PhaseModePrototype,
				Tasks: []colony.Task{
					{Goal: "Build hardened configuration checks"},
				},
			},
		},
		{
			name: "release",
			phase: colony.Phase{
				Name:        "Release candidate packaging",
				Description: "Prepare candidate artifacts for handoff",
				Mode:        colony.PhaseModePrototype,
				Tasks: []colony.Task{
					{Goal: "Build candidate artifacts"},
				},
			},
		},
		{
			name: "final-review",
			phase: colony.Phase{
				Name:        "Final review",
				Description: "Complete final signoff evidence before handoff",
				Mode:        colony.PhaseModePrototype,
				Tasks: []colony.Task{
					{Goal: "Build final review evidence"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := queenSpawnBudgetForPhase(tt.phase, "build", colony.ColonyState{})
			dispatches := budgetPressureDispatches()
			if len(dispatches) <= budget.MaxWorkers {
				t.Fatalf("fixture dispatch count = %d, want budget pressure beyond max workers %d", len(dispatches), budget.MaxWorkers)
			}

			decisions := queenSpawnBudgetDecisions(dispatches, budget)

			for _, caste := range []string{"builder", "watcher", "probe", "gatekeeper", "auditor"} {
				decision, ok := budgetDecisionForCaste(decisions, caste)
				if !ok {
					t.Fatalf("%s: missing decision for safety caste %s", tt.name, caste)
				}
				if !decision.Required || !decision.Selected {
					t.Fatalf("%s: %s decision = %+v, want required and selected under budget pressure", tt.name, caste, decision)
				}
			}
		})
	}
}

func budgetPressureDispatches() []CasteDispatch {
	return []CasteDispatch{
		{Caste: "ambassador", Score: 95},
		{Caste: "architect", Score: 94},
		{Caste: "auditor", Score: 10},
		{Caste: "builder", Score: 10},
		{Caste: "chaos", Score: 93},
		{Caste: "gatekeeper", Score: 10},
		{Caste: "keeper", Score: 92},
		{Caste: "measurer", Score: 91},
		{Caste: "probe", Score: 10},
		{Caste: "scout", Score: 90},
		{Caste: "watcher", Score: 10},
		{Caste: "weaver", Score: 89},
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
