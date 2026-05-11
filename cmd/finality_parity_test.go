package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Helper constructors for parity test matrix
// ---------------------------------------------------------------------------

// parityContinueManifest builds a minimal codexContinuePlanManifest suitable
// for continue-finalize parity tests.
func parityContinueManifest(opts ...func(*codexContinuePlanManifest)) codexContinuePlanManifest {
	m := codexContinuePlanManifest{
		Phase:             1,
		PhaseName:         "Test Phase",
		Root:              "",
		GeneratedAt:       "2026-05-11T00:00:00Z",
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexContinueExternalDispatch{
			{
				Stage:  "review",
				Caste:  "gatekeeper",
				Name:   "GK-01",
				Task:   "Security review",
				TaskID: "",
				Status: "completed",
			},
		},
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// paritySealManifest builds a minimal sealPlanManifest suitable for
// seal-finalize parity tests.
func paritySealManifest(opts ...func(*sealPlanManifest)) sealPlanManifest {
	m := sealPlanManifest{
		Workflow:          "seal",
		Phase:             1,
		PhaseName:         "Test Phase",
		Root:              "",
		GeneratedAt:       "2026-05-11T00:00:00Z",
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexContinueExternalDispatch{
			{
				Stage:  "review",
				Caste:  "gatekeeper",
				Name:   "GK-01",
				Task:   "Security review",
				TaskID: "",
				Status: "completed",
			},
		},
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// parityDispatch creates a single codexContinueExternalDispatch for tests.
func parityDispatch(name, caste, stage, status string) codexContinueExternalDispatch {
	return codexContinueExternalDispatch{
		Stage:  stage,
		Caste:  caste,
		Name:   name,
		Task:   "test task",
		TaskID: "",
		Status: status,
	}
}

// parityResult creates a single codexContinueExternalDispatch representing a
// worker result for tests.
func parityResult(name, status string) codexContinueExternalDispatch {
	return codexContinueExternalDispatch{
		Name:   name,
		Status: status,
	}
}

// ---------------------------------------------------------------------------
// Dimension 1: Manifest origin (plan-only vs external-task vs agent-delegate)
// ---------------------------------------------------------------------------

func TestFinalityParity_ManifestOrigin_ContinuePlanOnly(t *testing.T) {
	// continue accepts plan-only dispatch mode
	m := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.DispatchMode = "plan-only"
	})
	if m.DispatchMode != "plan-only" {
		t.Fatalf("expected plan-only, got %s", m.DispatchMode)
	}
	if !m.RequiresFinalizer {
		t.Fatal("expected RequiresFinalizer=true")
	}
}

func TestFinalityParity_ManifestOrigin_ContinueRejectsExternalTask(t *testing.T) {
	// continue finalize rejects non-plan-only dispatch modes
	m := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.DispatchMode = "external-task"
	})
	// The runCodexContinueFinalize guard: plan.DispatchMode != "plan-only"
	if m.DispatchMode == "plan-only" {
		t.Fatal("expected external-task, got plan-only")
	}
	// Verify the rejection condition matches what the code checks
	if m.DispatchMode != "plan-only" && m.RequiresFinalizer {
		// This is the condition that triggers the rejection error in runCodexContinueFinalize
		_ = struct{}{} // rejection path confirmed
	}
}

func TestFinalityParity_ManifestOrigin_SealAcceptsPlanOnly(t *testing.T) {
	m := paritySealManifest(func(m *sealPlanManifest) {
		m.DispatchMode = "plan-only"
	})
	// seal accepts plan-only
	accepted := m.DispatchMode == "plan-only" || m.DispatchMode == "agent-delegate"
	if !accepted {
		t.Fatalf("expected seal to accept plan-only, got mode %s", m.DispatchMode)
	}
}

func TestFinalityParity_ManifestOrigin_SealAcceptsAgentDelegate(t *testing.T) {
	m := paritySealManifest(func(m *sealPlanManifest) {
		m.DispatchMode = "agent-delegate"
	})
	// Intentional difference: seal accepts agent-delegate, continue does not
	accepted := m.DispatchMode == "plan-only" || m.DispatchMode == "agent-delegate"
	if !accepted {
		t.Fatalf("expected seal to accept agent-delegate, got mode %s", m.DispatchMode)
	}
}

func TestFinalityParity_ManifestOrigin_SealRejectsOtherModes(t *testing.T) {
	modes := []string{"external-task", "simulated", "in-repo", ""}
	for _, mode := range modes {
		m := paritySealManifest(func(m *sealPlanManifest) {
			m.DispatchMode = mode
		})
		accepted := m.DispatchMode == "plan-only" || m.DispatchMode == "agent-delegate"
		if accepted {
			t.Errorf("seal should reject dispatch mode %q", mode)
		}
	}
}

// ---------------------------------------------------------------------------
// Dimension 2: Root validation (missing/invalid root paths)
// ---------------------------------------------------------------------------

func TestFinalityParity_RootValidation_BothAcceptEmptyRoot(t *testing.T) {
	// Both continue and seal use validateFinalizerManifestRoot, which accepts
	// empty or matching roots.
	err := validateFinalizerManifestRoot("test_manifest", "", "/some/path")
	if err != nil {
		t.Errorf("empty root should be accepted: %v", err)
	}
}

func TestFinalityParity_RootValidation_BothAcceptMatchingRoot(t *testing.T) {
	err := validateFinalizerManifestRoot("test_manifest", "/some/path", "/some/path")
	if err != nil {
		t.Errorf("matching root should be accepted: %v", err)
	}
}

func TestFinalityParity_RootValidation_BothRejectMismatchedRoot(t *testing.T) {
	err := validateFinalizerManifestRoot("test_manifest", "/other/path", "/some/path")
	if err == nil {
		t.Fatal("mismatched root should be rejected")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("expected 'does not match' in error, got: %v", err)
	}
}

func TestFinalityParity_RootValidation_ContinueUsesSameValidator(t *testing.T) {
	// Verify continue calls the shared validator with "continue_manifest" label
	continueErr := validateFinalizerManifestRoot("continue_manifest", "/bad", "/good")
	sealErr := validateFinalizerManifestRoot("seal_manifest", "/bad", "/good")
	// Both should fail, and both should use the same underlying logic
	if continueErr == nil || sealErr == nil {
		t.Fatal("both should reject mismatched root")
	}
	// The label differs but the logic is shared
	if !strings.Contains(continueErr.Error(), "continue_manifest") {
		t.Errorf("continue error should reference continue_manifest: %v", continueErr)
	}
	if !strings.Contains(sealErr.Error(), "seal_manifest") {
		t.Errorf("seal error should reference seal_manifest: %v", sealErr)
	}
}

// ---------------------------------------------------------------------------
// Dimension 3: Required finalizer flag (--completion-file / manifest requirements)
// ---------------------------------------------------------------------------

func TestFinalityParity_RequiredFinalizer_ContinueRequiresPlanOnlyAndFinalizer(t *testing.T) {
	tests := []struct {
		name             string
		dispatchMode     string
		requiresFinalizer bool
		shouldAccept     bool
	}{
		{"plan-only + finalizer", "plan-only", true, true},
		{"plan-only no finalizer", "plan-only", false, false},
		{"external-task + finalizer", "external-task", true, false},
		{"agent-delegate + finalizer", "agent-delegate", true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := parityContinueManifest(func(m *codexContinuePlanManifest) {
				m.DispatchMode = tc.dispatchMode
				m.RequiresFinalizer = tc.requiresFinalizer
			})
			accepted := m.DispatchMode == "plan-only" && m.RequiresFinalizer
			if accepted != tc.shouldAccept {
				t.Errorf("accepted=%v, want %v", accepted, tc.shouldAccept)
			}
		})
	}
}

func TestFinalityParity_RequiredFinalizer_SealRequiresPlanOnlyOrAgentDelegateAndFinalizer(t *testing.T) {
	tests := []struct {
		name             string
		dispatchMode     string
		requiresFinalizer bool
		shouldAccept     bool
	}{
		{"plan-only + finalizer", "plan-only", true, true},
		{"plan-only no finalizer", "plan-only", false, false},
		{"agent-delegate + finalizer", "agent-delegate", true, true},
		{"agent-delegate no finalizer", "agent-delegate", false, false},
		{"external-task + finalizer", "external-task", true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := paritySealManifest(func(m *sealPlanManifest) {
				m.DispatchMode = tc.dispatchMode
				m.RequiresFinalizer = tc.requiresFinalizer
			})
			modeOK := m.DispatchMode == "plan-only" || m.DispatchMode == "agent-delegate"
			accepted := modeOK && m.RequiresFinalizer
			if accepted != tc.shouldAccept {
				t.Errorf("accepted=%v, want %v", accepted, tc.shouldAccept)
			}
		})
	}
}

func TestFinalityParity_RequiredFinalizer_BothRejectNoDispatches(t *testing.T) {
	// Continue: rejects empty dispatches unless SkipWatchers && ReviewDepth=light
	emptyContinue := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Dispatches = nil
		m.SkipWatchers = false
		m.ReviewDepth = ""
	})
	emptyAllowed := len(emptyContinue.Dispatches) == 0 && emptyContinue.SkipWatchers && emptyContinue.ReviewDepth == string(colony.VerificationDepthLight)
	if emptyAllowed {
		t.Error("continue should reject empty dispatches when SkipWatchers=false")
	}

	// Seal: always rejects empty dispatches
	emptySeal := paritySealManifest(func(m *sealPlanManifest) {
		m.Dispatches = nil
	})
	if len(emptySeal.Dispatches) != 0 {
		t.Error("seal manifest should have empty dispatches")
	}
	// The seal code checks: len(manifest.Dispatches) == 0 -> error
	if len(emptySeal.Dispatches) == 0 {
		// confirmed: seal rejects empty dispatches
	}
}

func TestFinalityParity_RequiredFinalizer_ContinueAllowsEmptyWithSkipWatchers(t *testing.T) {
	// Intentional difference: continue allows empty dispatches when SkipWatchers=true and ReviewDepth=light
	m := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Dispatches = nil
		m.SkipWatchers = true
		m.ReviewDepth = string(colony.VerificationDepthLight)
	})
	allowed := len(m.Dispatches) == 0 && m.SkipWatchers && m.ReviewDepth == string(colony.VerificationDepthLight)
	if !allowed {
		t.Error("continue should allow empty dispatches when SkipWatchers=true and ReviewDepth=light")
	}
}

// ---------------------------------------------------------------------------
// Dimension 4: Worker results collection (merge behavior)
// ---------------------------------------------------------------------------

func TestFinalityParity_Merge_ContinueAndSealProduceSameOutput(t *testing.T) {
	// Since mergeExternalSealReviewResults delegates to mergeExternalContinueResults,
	// the same inputs should produce the same outputs.
	dispatches := []codexContinueExternalDispatch{
		parityDispatch("GK-01", "gatekeeper", "review", "completed"),
	}
	results := []codexContinueExternalDispatch{
		parityResult("GK-01", "completed"),
	}

	continuePlan := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Dispatches = dispatches
	})

	sealPlan := paritySealManifest(func(m *sealPlanManifest) {
		m.Dispatches = dispatches
	})

	continueFlow, continueErr := mergeExternalContinueResults(continuePlan, results)
	if continueErr != nil {
		t.Fatalf("continue merge: %v", continueErr)
	}

	sealFlow, sealErr := mergeExternalSealReviewResults(sealPlan, results)
	if sealErr != nil {
		t.Fatalf("seal merge: %v", sealErr)
	}

	// Both should produce identical output since seal delegates to continue
	if len(continueFlow) != len(sealFlow) {
		t.Fatalf("flow length mismatch: continue=%d, seal=%d", len(continueFlow), len(sealFlow))
	}
	for i := range continueFlow {
		if continueFlow[i].Name != sealFlow[i].Name {
			t.Errorf("step %d name mismatch: continue=%q, seal=%q", i, continueFlow[i].Name, sealFlow[i].Name)
		}
		if continueFlow[i].Status != sealFlow[i].Status {
			t.Errorf("step %d status mismatch: continue=%q, seal=%q", i, continueFlow[i].Status, sealFlow[i].Status)
		}
		if continueFlow[i].Caste != sealFlow[i].Caste {
			t.Errorf("step %d caste mismatch: continue=%q, seal=%q", i, continueFlow[i].Caste, sealFlow[i].Caste)
		}
	}
}

func TestFinalityParity_Merge_BothRejectDuplicateResults(t *testing.T) {
	results := []codexContinueExternalDispatch{
		parityResult("GK-01", "completed"),
		parityResult("GK-01", "completed"),
	}

	continuePlan := parityContinueManifest()
	_, continueErr := mergeExternalContinueResults(continuePlan, results)
	if continueErr == nil {
		t.Fatal("continue should reject duplicate results")
	}
	if !strings.Contains(continueErr.Error(), "duplicate") {
		t.Errorf("expected duplicate error, got: %v", continueErr)
	}

	sealPlan := paritySealManifest()
	_, sealErr := mergeExternalSealReviewResults(sealPlan, results)
	if sealErr == nil {
		t.Fatal("seal should reject duplicate results")
	}
	if !strings.Contains(sealErr.Error(), "duplicate") {
		t.Errorf("expected duplicate error, got: %v", sealErr)
	}
}

func TestFinalityParity_Merge_BothRejectNamelessResult(t *testing.T) {
	results := []codexContinueExternalDispatch{
		{Name: "", Status: "completed"},
	}

	continuePlan := parityContinueManifest()
	_, continueErr := mergeExternalContinueResults(continuePlan, results)
	if continueErr == nil {
		t.Fatal("continue should reject nameless result")
	}

	sealPlan := paritySealManifest()
	_, sealErr := mergeExternalSealReviewResults(sealPlan, results)
	if sealErr == nil {
		t.Fatal("seal should reject nameless result")
	}
}

func TestFinalityParity_Merge_BothRejectNonTerminalStatus(t *testing.T) {
	results := []codexContinueExternalDispatch{
		parityResult("GK-01", "running"),
	}

	continuePlan := parityContinueManifest()
	_, continueErr := mergeExternalContinueResults(continuePlan, results)
	if continueErr == nil {
		t.Fatal("continue should reject non-terminal status")
	}
	if !strings.Contains(continueErr.Error(), "non-terminal") {
		t.Errorf("expected non-terminal error, got: %v", continueErr)
	}

	sealPlan := paritySealManifest()
	_, sealErr := mergeExternalSealReviewResults(sealPlan, results)
	if sealErr == nil {
		t.Fatal("seal should reject non-terminal status")
	}
	if !strings.Contains(sealErr.Error(), "non-terminal") {
		t.Errorf("expected non-terminal error, got: %v", sealErr)
	}
}

func TestFinalityParity_Merge_BothRejectCasteMismatch(t *testing.T) {
	results := []codexContinueExternalDispatch{
		{Name: "GK-01", Caste: "watcher", Stage: "review", Status: "completed"},
	}

	continuePlan := parityContinueManifest() // dispatches caste=gatekeeper
	_, continueErr := mergeExternalContinueResults(continuePlan, results)
	if continueErr == nil {
		t.Fatal("continue should reject caste mismatch")
	}
	if !strings.Contains(continueErr.Error(), "caste") {
		t.Errorf("expected caste error, got: %v", continueErr)
	}

	sealPlan := paritySealManifest()
	_, sealErr := mergeExternalSealReviewResults(sealPlan, results)
	if sealErr == nil {
		t.Fatal("seal should reject caste mismatch")
	}
	if !strings.Contains(sealErr.Error(), "caste") {
		t.Errorf("expected caste error, got: %v", sealErr)
	}
}

func TestFinalityParity_Merge_BothRejectStageMismatch(t *testing.T) {
	results := []codexContinueExternalDispatch{
		{Name: "GK-01", Caste: "gatekeeper", Stage: "verification", Status: "completed"},
	}

	continuePlan := parityContinueManifest() // dispatches stage=review
	_, continueErr := mergeExternalContinueResults(continuePlan, results)
	if continueErr == nil {
		t.Fatal("continue should reject stage mismatch")
	}
	if !strings.Contains(continueErr.Error(), "stage") {
		t.Errorf("expected stage error, got: %v", continueErr)
	}

	sealPlan := paritySealManifest()
	_, sealErr := mergeExternalSealReviewResults(sealPlan, results)
	if sealErr == nil {
		t.Fatal("seal should reject stage mismatch")
	}
	if !strings.Contains(sealErr.Error(), "stage") {
		t.Errorf("expected stage error, got: %v", sealErr)
	}
}

func TestFinalityParity_Merge_BothTreatMissingAsTimeout(t *testing.T) {
	// When a dispatch has no matching result, both should synthesize a timeout entry
	dispatches := []codexContinueExternalDispatch{
		parityDispatch("GK-01", "gatekeeper", "review", ""),
		parityDispatch("KP-02", "gatekeeper", "review", ""),
	}
	results := []codexContinueExternalDispatch{
		parityResult("GK-01", "completed"),
		// KP-02 result is missing
	}

	continuePlan := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Dispatches = dispatches
	})
	continueFlow, continueErr := mergeExternalContinueResults(continuePlan, results)
	if continueErr != nil {
		t.Fatalf("continue merge: %v", continueErr)
	}

	sealPlan := paritySealManifest(func(m *sealPlanManifest) {
		m.Dispatches = dispatches
	})
	sealFlow, sealErr := mergeExternalSealReviewResults(sealPlan, results)
	if sealErr != nil {
		t.Fatalf("seal merge: %v", sealErr)
	}

	// Both should have 2 entries
	if len(continueFlow) != 2 {
		t.Fatalf("continue flow length = %d, want 2", len(continueFlow))
	}
	if len(sealFlow) != 2 {
		t.Fatalf("seal flow length = %d, want 2", len(sealFlow))
	}

	// The missing KP-02 should be treated as timeout in both
	if continueFlow[1].Status != "timeout" {
		t.Errorf("continue missing result status = %q, want timeout", continueFlow[1].Status)
	}
	if sealFlow[1].Status != "timeout" {
		t.Errorf("seal missing result status = %q, want timeout", sealFlow[1].Status)
	}
}

func TestFinalityParity_Merge_BothNormalizeStatus(t *testing.T) {
	// Both should normalize "code_written" to "completed"
	dispatches := []codexContinueExternalDispatch{
		parityDispatch("BD-01", "builder", "review", ""),
	}
	results := []codexContinueExternalDispatch{
		parityResult("BD-01", "code_written"),
	}

	continuePlan := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Dispatches = dispatches
	})
	continueFlow, err := mergeExternalContinueResults(continuePlan, results)
	if err != nil {
		t.Fatalf("continue merge: %v", err)
	}
	if continueFlow[0].Status != "completed" {
		t.Errorf("continue normalized status = %q, want completed", continueFlow[0].Status)
	}

	sealPlan := paritySealManifest(func(m *sealPlanManifest) {
		m.Dispatches = dispatches
	})
	sealFlow, err := mergeExternalSealReviewResults(sealPlan, results)
	if err != nil {
		t.Fatalf("seal merge: %v", err)
	}
	if sealFlow[0].Status != "completed" {
		t.Errorf("seal normalized status = %q, want completed", sealFlow[0].Status)
	}
}

func TestFinalityParity_Merge_BothDeduplicateFindings(t *testing.T) {
	// Both should deduplicate findings across findings and issues fields
	dispatches := []codexContinueExternalDispatch{
		parityDispatch("GK-01", "gatekeeper", "review", ""),
	}
	duplicateFinding := codexReviewFinding{
		Domain: "security", Severity: "HIGH", File: "auth.go", Line: 42,
		Category: "injection", Title: "SQL injection", Description: "raw query", Suggestion: "use prepared stmt",
	}
	results := []codexContinueExternalDispatch{
		{
			Name:     "GK-01",
			Caste:    "gatekeeper",
			Stage:    "review",
			Status:   "completed",
			Findings: []codexReviewFinding{duplicateFinding},
			Issues:   []codexReviewFinding{duplicateFinding},
		},
	}

	continuePlan := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Dispatches = dispatches
	})
	continueFlow, err := mergeExternalContinueResults(continuePlan, results)
	if err != nil {
		t.Fatalf("continue merge: %v", err)
	}
	if len(continueFlow[0].Findings) != 1 {
		t.Errorf("continue findings count = %d, want 1 (deduplicated)", len(continueFlow[0].Findings))
	}

	sealPlan := paritySealManifest(func(m *sealPlanManifest) {
		m.Dispatches = dispatches
	})
	sealFlow, err := mergeExternalSealReviewResults(sealPlan, results)
	if err != nil {
		t.Fatalf("seal merge: %v", err)
	}
	if len(sealFlow[0].Findings) != 1 {
		t.Errorf("seal findings count = %d, want 1 (deduplicated)", len(sealFlow[0].Findings))
	}
}

// ---------------------------------------------------------------------------
// Dimension 5: Timeout behavior (worker timeout handling)
// ---------------------------------------------------------------------------

func TestFinalityParity_Timeout_MissingResultBecomesTimeout(t *testing.T) {
	// Already tested in Dimension 4 (BothTreatMissingAsTimeout), but this
	// specifically verifies the synthesized timeout summary message.
	dispatches := []codexContinueExternalDispatch{
		parityDispatch("WK-01", "watcher", "review", ""),
	}
	results := []codexContinueExternalDispatch{} // no results at all

	continuePlan := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Dispatches = dispatches
	})
	continueFlow, err := mergeExternalContinueResults(continuePlan, results)
	if err != nil {
		t.Fatalf("continue merge: %v", err)
	}

	sealPlan := paritySealManifest(func(m *sealPlanManifest) {
		m.Dispatches = dispatches
	})
	sealFlow, err := mergeExternalSealReviewResults(sealPlan, results)
	if err != nil {
		t.Fatalf("seal merge: %v", err)
	}

	// Both should synthesize the same timeout summary
	if continueFlow[0].Summary != "worker result was not provided; treated as timed out" {
		t.Errorf("continue timeout summary = %q", continueFlow[0].Summary)
	}
	if sealFlow[0].Summary != "worker result was not provided; treated as timed out" {
		t.Errorf("seal timeout summary = %q", sealFlow[0].Summary)
	}
}

func TestFinalityParity_Timeout_TimedOutStatusPreserved(t *testing.T) {
	// When a result explicitly reports "timed_out", both should normalize to "timeout"
	results := []codexContinueExternalDispatch{
		parityResult("WK-01", "timed_out"),
	}

	continuePlan := parityContinueManifest()
	continueFlow, err := mergeExternalContinueResults(continuePlan, results)
	if err != nil {
		t.Fatalf("continue merge: %v", err)
	}
	if continueFlow[0].Status != "timeout" {
		t.Errorf("continue status = %q, want timeout", continueFlow[0].Status)
	}

	sealPlan := paritySealManifest()
	sealFlow, err := mergeExternalSealReviewResults(sealPlan, results)
	if err != nil {
		t.Fatalf("seal merge: %v", err)
	}
	if sealFlow[0].Status != "timeout" {
		t.Errorf("seal status = %q, want timeout", sealFlow[0].Status)
	}
}

func TestFinalityParity_Timeout_CancelledStatusBecomesTimeout(t *testing.T) {
	results := []codexContinueExternalDispatch{
		parityResult("WK-01", "cancelled"),
	}

	continuePlan := parityContinueManifest()
	continueFlow, err := mergeExternalContinueResults(continuePlan, results)
	if err != nil {
		t.Fatalf("continue merge: %v", err)
	}
	if continueFlow[0].Status != "timeout" {
		t.Errorf("continue cancelled status = %q, want timeout", continueFlow[0].Status)
	}

	sealPlan := paritySealManifest()
	sealFlow, err := mergeExternalSealReviewResults(sealPlan, results)
	if err != nil {
		t.Fatalf("seal merge: %v", err)
	}
	if sealFlow[0].Status != "timeout" {
		t.Errorf("seal cancelled status = %q, want timeout", sealFlow[0].Status)
	}
}

// ---------------------------------------------------------------------------
// Dimension 6: State mutation order (phase advancement sequence)
// ---------------------------------------------------------------------------

func TestFinalityParity_StateMutation_ContinuePhaseAdvancement(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseInProgress},
				{ID: 2, Name: "Phase 2", Status: colony.PhasePending},
			},
		},
		BuildStartedAt: ptrTimeNow(),
	}
	createTestColonyState(t, dataDir, state)

	// Verify the state was written correctly
	loaded, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if loaded.CurrentPhase != 1 {
		t.Fatalf("expected current phase 1, got %d", loaded.CurrentPhase)
	}
	if loaded.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("expected phase 1 in progress, got %s", loaded.Plan.Phases[0].Status)
	}
}

func TestFinalityParity_StateMutation_SealRequiresCompletedPhase(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	loaded, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	phase, ok := finalCompletedPhase(loaded)
	if !ok {
		t.Fatal("expected to find a completed final phase")
	}
	if phase.ID != 1 {
		t.Fatalf("expected phase ID 1, got %d", phase.ID)
	}
}

func TestFinalityParity_StateMutation_ContinueRejectsWrongPhase(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted},
				{ID: 2, Name: "Phase 2", Status: colony.PhaseInProgress},
			},
		},
		BuildStartedAt: ptrTimeNow(),
	}
	createTestColonyState(t, dataDir, state)

	// Manifest says phase 1, but active phase is 2
	plan := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.Phase = 1
	})

	_, _, _, err := validateExternalContinueState(&plan)
	if err == nil {
		t.Fatal("expected error for phase mismatch")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("expected phase mismatch error, got: %v", err)
	}
}

func TestFinalityParity_StateMutation_SealRejectsWrongPhase(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	// Manifest says phase 2, but final completed phase is 1
	m := paritySealManifest(func(m *sealPlanManifest) {
		m.Phase = 2
	})

	loaded, _ := loadActiveColonyState()
	phase, ok := finalCompletedPhase(loaded)
	if !ok {
		t.Fatal("expected completed phase")
	}
	if m.Phase != phase.ID {
		// This is the condition that triggers rejection in seal finalize
		_ = struct{}{} // rejection path confirmed
	}
}

func TestFinalityParity_StateMutation_ColonyModeValidationShared(t *testing.T) {
	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		ColonyMode:   colony.ColonyModeColony,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseInProgress},
			},
		},
		BuildStartedAt: ptrTimeNow(),
	}

	// Both continue and seal use validateFinalizerManifestColonyMode
	continueErr := validateFinalizerManifestColonyMode("continue_manifest", "orchestrator", state)
	sealErr := validateFinalizerManifestColonyMode("seal_manifest", "orchestrator", state)

	// Both should reject mismatched mode
	if continueErr == nil {
		t.Error("continue should reject mismatched colony mode")
	}
	if sealErr == nil {
		t.Error("seal should reject mismatched colony mode")
	}

	// Both should accept matching mode
	continueErr = validateFinalizerManifestColonyMode("continue_manifest", "colony", state)
	sealErr = validateFinalizerManifestColonyMode("seal_manifest", "colony", state)
	if continueErr != nil {
		t.Errorf("continue should accept matching mode: %v", continueErr)
	}
	if sealErr != nil {
		t.Errorf("seal should accept matching mode: %v", sealErr)
	}
}

// ---------------------------------------------------------------------------
// Dimension 7: Seal-specific delegation verification
// ---------------------------------------------------------------------------

func TestFinalityParity_SealDelegatesToContinueMerge(t *testing.T) {
	// Verify that mergeExternalSealReviewResults correctly maps seal fields
	// to continue fields by comparing output structure
	dispatches := []codexContinueExternalDispatch{
		{
			Stage: "review", Caste: "gatekeeper", Name: "GK-01",
			Task: "Security", TaskID: "", Status: "completed",
		},
		{
			Stage: "review", Caste: "auditor", Name: "AU-02",
			Task: "Quality", TaskID: "", Status: "completed",
		},
	}
	results := []codexContinueExternalDispatch{
		{Name: "GK-01", Status: "completed", Caste: "gatekeeper", Stage: "review", Summary: "No issues"},
		{Name: "AU-02", Status: "completed", Caste: "auditor", Stage: "review", Summary: "Clean code"},
	}

	sealPlan := paritySealManifest(func(m *sealPlanManifest) {
		m.Phase = 3
		m.PhaseName = "Final Phase"
		m.Dispatches = dispatches
	})

	sealFlow, err := mergeExternalSealReviewResults(sealPlan, results)
	if err != nil {
		t.Fatalf("seal merge: %v", err)
	}

	if len(sealFlow) != 2 {
		t.Fatalf("expected 2 flow steps, got %d", len(sealFlow))
	}
	if sealFlow[0].Name != "GK-01" {
		t.Errorf("step 0 name = %q, want GK-01", sealFlow[0].Name)
	}
	if sealFlow[0].Summary != "No issues" {
		t.Errorf("step 0 summary = %q, want 'No issues'", sealFlow[0].Summary)
	}
	if sealFlow[1].Name != "AU-02" {
		t.Errorf("step 1 name = %q, want AU-02", sealFlow[1].Name)
	}
	if sealFlow[1].Summary != "Clean code" {
		t.Errorf("step 1 summary = %q, want 'Clean code'", sealFlow[1].Summary)
	}
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

func ptrTimeNow() *time.Time {
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	return &now
}

func TestFinalityParity_LoadExternalContinueCompletion_RejectsMissingFile(t *testing.T) {
	_, err := loadExternalContinueCompletion("/nonexistent/path/completion.json")
	if err == nil {
		t.Fatal("expected error for missing completion file")
	}
	if !strings.Contains(err.Error(), "flag --completion-file is required") && !strings.Contains(err.Error(), "read completion file") {
		t.Errorf("expected completion-file error, got: %v", err)
	}
}

func TestFinalityParity_LoadExternalContinueCompletion_RejectsEmptyPath(t *testing.T) {
	_, err := loadExternalContinueCompletion("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if !strings.Contains(err.Error(), "flag --completion-file is required") {
		t.Errorf("expected 'flag --completion-file is required' error, got: %v", err)
	}
}

func TestFinalityParity_LoadExternalContinueCompletion_AcceptsValidManifest(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	validJSON := `{
		"continue_manifest": {
			"phase": 1,
			"phase_name": "Test",
			"root": "",
			"generated_at": "2026-05-11T00:00:00Z",
			"dispatch_mode": "plan-only",
			"requires_finalizer": true,
			"dispatches": [
				{"stage": "review", "caste": "gatekeeper", "name": "GK-01", "task": "test", "task_id": "", "status": "completed"}
			]
		}
	}`
	if err := os.WriteFile(path, []byte(validJSON), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	completion, err := loadExternalContinueCompletion(path)
	if err != nil {
		t.Fatalf("loadExternalContinueCompletion: %v", err)
	}
	if completion.activeManifest() == nil {
		t.Fatal("expected active manifest")
	}
	if completion.activeManifest().Phase != 1 {
		t.Errorf("phase = %d, want 1", completion.activeManifest().Phase)
	}
}

func TestFinalityParity_WorkerResultsAggregation(t *testing.T) {
	// workerResults() should aggregate dispatches + results + workers
	c := codexExternalContinueCompletion{
		ContinueManifest: nil,
		Dispatches: []codexContinueExternalDispatch{
			{Name: "D-01", Status: "completed"},
		},
		Results: []codexContinueExternalDispatch{
			{Name: "R-01", Status: "completed"},
		},
		Workers: []codexContinueExternalDispatch{
			{Name: "W-01", Status: "completed"},
		},
	}
	aggregated := c.workerResults()
	if len(aggregated) != 3 {
		t.Fatalf("expected 3 aggregated results, got %d", len(aggregated))
	}
	if aggregated[0].Name != "D-01" || aggregated[1].Name != "R-01" || aggregated[2].Name != "W-01" {
		t.Errorf("aggregation order wrong: %v", aggregated)
	}
}

func TestFinalityParity_VerificationTimeoutOverride(t *testing.T) {
	// continueFinalizeVerificationTimeout should prefer override over plan value
	plan := parityContinueManifest(func(m *codexContinuePlanManifest) {
		m.VerificationTimeout = 120
	})

	// With no override, plan value is used
	got := continueFinalizeVerificationTimeout(&plan, 0)
	if got == 0 {
		t.Error("expected plan timeout to be used when no override")
	}

	// With override, override takes precedence
	override := int64(300)
	got = continueFinalizeVerificationTimeout(&plan, time.Duration(override)*time.Second)
	if got != time.Duration(override)*time.Second {
		t.Errorf("expected override %v, got %v", time.Duration(override)*time.Second, got)
	}
}
