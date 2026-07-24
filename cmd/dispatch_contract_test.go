package cmd

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// resetDispatchPolicyState clears the loaded policy and cache so each test
// starts fresh.
func resetDispatchPolicyState() {
	loadedDispatchPolicy = nil
	loadedDispatchPolicyOnce = sync.Once{}
	policyCacheMu.Lock()
	delete(policyCache, dispatchContractPathOverride)
	policyCacheMu.Unlock()
}

func TestDispatchContractLoad_CustomSurveyExecutionModel(t *testing.T) {
	resetDispatchPolicyState()
	dir := t.TempDir()
	path := filepath.Join(dir, "dispatch-contract.yaml")
	content := `dispatch_contract:
  execution_models:
    survey: "custom survey model"
  deadline_policies:
    survey: "custom survey deadline"
  dependency_behaviors:
    survey: "custom survey dependency"
  fallback_behaviors:
    survey: "custom survey fallback"
  fallback_visibility:
    survey:
      - custom_visibility
  result_collection_policies:
    survey: "custom survey result"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	dispatchContractPathOverride = path
	defer func() { dispatchContractPathOverride = "" }()

	p := loadDispatchContractPolicy()
	if p == nil {
		t.Fatal("expected policy to load")
	}
	if p.ExecutionModels["survey"] != "custom survey model" {
		t.Errorf("execution_models.survey = %q, want %q", p.ExecutionModels["survey"], "custom survey model")
	}
	if p.DeadlinePolicies["survey"] != "custom survey deadline" {
		t.Errorf("deadline_policies.survey = %q, want %q", p.DeadlinePolicies["survey"], "custom survey deadline")
	}
	if p.DependencyBehaviors["survey"] != "custom survey dependency" {
		t.Errorf("dependency_behaviors.survey = %q, want %q", p.DependencyBehaviors["survey"], "custom survey dependency")
	}
	if p.FallbackBehaviors["survey"] != "custom survey fallback" {
		t.Errorf("fallback_behaviors.survey = %q, want %q", p.FallbackBehaviors["survey"], "custom survey fallback")
	}
	if len(p.FallbackVisibility["survey"]) != 1 || p.FallbackVisibility["survey"][0] != "custom_visibility" {
		t.Errorf("fallback_visibility.survey = %v, want [custom_visibility]", p.FallbackVisibility["survey"])
	}
	if p.ResultCollectionPolicies["survey"] != "custom survey result" {
		t.Errorf("result_collection_policies.survey = %q, want %q", p.ResultCollectionPolicies["survey"], "custom survey result")
	}
}

func TestDispatchContractFallback_NoPolicyFile(t *testing.T) {
	resetDispatchPolicyState()
	dispatchContractPathOverride = filepath.Join(t.TempDir(), "nonexistent.yaml")
	defer func() { dispatchContractPathOverride = "" }()

	contract := surveyDispatchContractWithTimeout(0)
	if contract["execution_model"] != fallbackSurveyExecutionModel {
		t.Errorf("survey execution_model = %q, want fallback %q", contract["execution_model"], fallbackSurveyExecutionModel)
	}
	if contract["deadline_policy"] != fallbackSurveyDeadlinePolicy {
		t.Errorf("survey deadline_policy = %q, want fallback %q", contract["deadline_policy"], fallbackSurveyDeadlinePolicy)
	}
	if contract["dependency_behavior"] != fallbackSurveyDependencyBehavior {
		t.Errorf("survey dependency_behavior = %q, want fallback %q", contract["dependency_behavior"], fallbackSurveyDependencyBehavior)
	}
	if contract["fallback_behavior"] != fallbackSurveyFallbackBehavior {
		t.Errorf("survey fallback_behavior = %q, want fallback %q", contract["fallback_behavior"], fallbackSurveyFallbackBehavior)
	}

	planning := planningDispatchContractWithTimeout(0)
	if planning["execution_model"] != fallbackPlanningExecutionModel {
		t.Errorf("planning execution_model = %q, want fallback %q", planning["execution_model"], fallbackPlanningExecutionModel)
	}
	if planning["deadline_policy"] != fallbackPlanningDeadlinePolicy {
		t.Errorf("planning deadline_policy = %q, want fallback %q", planning["deadline_policy"], fallbackPlanningDeadlinePolicy)
	}
	if planning["dependency_behavior"] != fallbackPlanningDependencyBehavior {
		t.Errorf("planning dependency_behavior = %q, want fallback %q", planning["dependency_behavior"], fallbackPlanningDependencyBehavior)
	}
	if planning["fallback_behavior"] != fallbackPlanningFallbackBehavior {
		t.Errorf("planning fallback_behavior = %q, want fallback %q", planning["fallback_behavior"], fallbackPlanningFallbackBehavior)
	}
}

func TestDispatchContractFallback_PartialPolicy(t *testing.T) {
	resetDispatchPolicyState()
	dir := t.TempDir()
	path := filepath.Join(dir, "dispatch-contract.yaml")
	content := `dispatch_contract:
  execution_models:
    survey: "partial survey model"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	dispatchContractPathOverride = path
	defer func() { dispatchContractPathOverride = "" }()

	// Survey should use the custom value for execution_model but fallback for others.
	contract := surveyDispatchContractWithTimeout(0)
	if contract["execution_model"] != "partial survey model" {
		t.Errorf("survey execution_model = %q, want %q", contract["execution_model"], "partial survey model")
	}
	if contract["deadline_policy"] != fallbackSurveyDeadlinePolicy {
		t.Errorf("survey deadline_policy = %q, want fallback %q", contract["deadline_policy"], fallbackSurveyDeadlinePolicy)
	}

	// Planning should use fallback for everything since no planning keys exist.
	planning := planningDispatchContractWithTimeout(0)
	if planning["execution_model"] != fallbackPlanningExecutionModel {
		t.Errorf("planning execution_model = %q, want fallback %q", planning["execution_model"], fallbackPlanningExecutionModel)
	}
	if planning["deadline_policy"] != fallbackPlanningDeadlinePolicy {
		t.Errorf("planning deadline_policy = %q, want fallback %q", planning["deadline_policy"], fallbackPlanningDeadlinePolicy)
	}
}

func TestDispatchContractLoad_TimeoutDefaultsAccessible(t *testing.T) {
	resetDispatchPolicyState()
	dir := t.TempDir()
	path := filepath.Join(dir, "dispatch-contract.yaml")
	content := `dispatch_contract:
  max_workers_per_phase: 10
  spawn_depth_limits:
    0: 4
    1: 4
    2: 2
    3: 0
  timeout_defaults:
    build: 600
    continue: 300
    plan: 300
    verify: 180
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	dispatchContractPathOverride = path
	defer func() { dispatchContractPathOverride = "" }()

	p := loadDispatchContractPolicy()
	if p == nil {
		t.Fatal("expected policy to load")
	}
	// The struct does not model numeric fields, but loading must succeed
	// without error and the string fields should be empty (not present).
	if _, ok := p.ExecutionModels["survey"]; ok {
		t.Error("expected execution_models to be empty when not present in YAML")
	}
}

func TestDispatchContractLoad_ExtendedPlanningFields(t *testing.T) {
	resetDispatchPolicyState()
	dir := t.TempDir()
	path := filepath.Join(dir, "dispatch-contract.yaml")
	content := `dispatch_contract:
  execution_models:
    planning_extended: "%d custom extended workers"
  dependency_behaviors:
    planning_extended: "custom extended dependency"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	dispatchContractPathOverride = path
	defer func() { dispatchContractPathOverride = "" }()

	contract := planningDispatchContractForDispatches([]codexPlanningDispatch{{}, {}, {}}, 0)
	wantEM := "3 custom extended workers"
	if contract["execution_model"] != wantEM {
		t.Errorf("extended execution_model = %q, want %q", contract["execution_model"], wantEM)
	}
	if contract["dependency_behavior"] != "custom extended dependency" {
		t.Errorf("extended dependency_behavior = %q, want %q", contract["dependency_behavior"], "custom extended dependency")
	}
}
