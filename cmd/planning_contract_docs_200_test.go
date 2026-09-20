package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlanningContractDocuments200 keeps the renderer-neutral public contract
// aligned with the Phase 200 Go state machine. The assertions are grouped by
// the document to which a contract applies: a generic union-of-three grep
// would let one exhaustive file hide a missing host or Specification boundary.
func TestPlanningContractDocuments200(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	documents := map[string]string{
		"plan": filepath.Join(root, "cmd", "contracts", "plan.md"),
		"spec": filepath.Join(root, "cmd", "contracts", "spec.md"),
		"host": filepath.Join(root, ".aether", "docs", "wrapper-host-contract.md"),
	}
	text := make(map[string]string, len(documents))
	for name, path := range documents {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s contract %s: %v", name, path, readErr)
		}
		if strings.TrimSpace(string(raw)) == "" {
			t.Fatalf("%s contract %s is empty", name, path)
		}
		text[name] = strings.ToLower(string(raw))
	}

	assertDocumentContractMarkers200(t, "plan", text["plan"], []string{
		// Staged authority and one-caste finalization.
		"preset_required", "scout_running", "route_running", "candidate_ready", "planning-stage-manifest/v1",
		"planning-scout-result/v1", "planning-route-setter-result/v1", "stage_receipt", "route_stage_receipt",
		// Route-Setter proposes; Go validates all five readiness axes and derives policy.
		"route-setter assessment proposes", "go validates", "knowledge", "requirements", "risks", "dependencies", "effort",
		// Decision/SPEC branch and causal pass history.
		"direct_resume", "successor_spec_required", "spec_approval_required", "iteration_card", "candidate_iteration_detail",
		// Evidence-bearing stop/candidate recommendation and exact acceptance.
		"evidence_that_would_change", "producer `queen`", "evidence ids", "rationale", "acceptance_command",
		"aether plan --accept-candidate <candidate-id>", "specification approval", "planning stop", "candidate acceptance",
		// Replay, recovery, machine refusal vocabulary, and platform/host authority.
		"exact retry", "divergent replay", "replayed: true", "recovery_command", "plan_authority_candidate_not_accepted",
		"wrappers may", "they may not", "aether spec", "aether plan", "/ant-spec", "/ant-plan",
	})

	assertDocumentContractMarkers200(t, "spec", text["spec"], []string{
		"spec-command/v1", "aether spec --inspect", "aether spec --add", "aether spec --modify", "aether spec --remove",
		"aether spec --approve", "aether spec --repair-projection", "revision id", "content hash", "approval token", "receipt",
		"outcomes", "included_behaviors", "exclusions", "binding_decisions", "requirements", "acceptance_checks",
		"negative_expectations", "recovery_expectations", "affected_public_paths", "owner-readable",
		"classified_delta", "affected_scope", "exact retry", "divergent replay", "state unchanged",
		"specification approval", "planning stop", "candidate acceptance", "direct_resume", "successor_spec_required",
		"wrappers may", "they may not", "aether spec", "/ant-spec",
	})

	assertDocumentContractMarkers200(t, "wrapper-host", text["host"], []string{
		"result.plan_manifest.stage_manifest", "stage_manifest.expected_caste", "stage_manifest.expected_result_type",
		"result.stage_receipt", "result.route_stage_manifest", "result.route_stage_receipt", "result.scout_stage_manifest",
		"result.decision_cards", "result.iteration_card", "result.plan_candidate", "result.acceptance_command",
		"route-setter proposes", "go validates", "direct_resume", "successor_spec_required", "evidence_that_would_change",
		"recommendation's producer", "exact replay", "divergent replay", "specification approval", "planning stop",
		"candidate acceptance", "may", "must not", "aether spec", "aether plan", "/ant-spec", "/ant-plan",
	})
}

func assertDocumentContractMarkers200(t *testing.T, name, text string, required []string) {
	t.Helper()
	for _, marker := range required {
		if !strings.Contains(text, strings.ToLower(marker)) {
			t.Errorf("%s contract is missing Phase 200 marker %q", name, marker)
		}
	}
}
