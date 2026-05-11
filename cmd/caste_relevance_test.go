package cmd

import (
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
