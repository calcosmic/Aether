package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestOracleEscalationKeepsScopedWriteAccess covers behaviours 1-2: escalating
// a Scout to an Oracle must not silently broaden write access. Before this
// change "oracle" fell through to behavioralRestrictionsForCaste's default
// branch and returned nil -- no restriction at all.
func TestOracleEscalationKeepsScopedWriteAccess(t *testing.T) {
	profile := codex.PermissionProfileForCaste("oracle")

	if len(profile.BehavioralRestrictions) == 0 {
		t.Fatal("PermissionProfileForCaste(\"oracle\").BehavioralRestrictions is empty; escalated Oracle would carry no write scope restriction")
	}

	joined := strings.Join(profile.BehavioralRestrictions, " ")
	if !strings.Contains(joined, ".aether/data/phase-research") {
		t.Errorf("oracle restriction = %q, want it to mention .aether/data/phase-research", joined)
	}
	if !strings.Contains(joined, ".aether/oracle") {
		t.Errorf("oracle restriction = %q, want it to mention Oracle's own artifact directory (.aether/oracle)", joined)
	}
	if strings.Contains(joined, "unrestricted") {
		t.Errorf("oracle restriction = %q, must not grant unrestricted workspace writes", joined)
	}
}

func escalationTestSurvey() codexSurveyContext {
	return codexSurveyContext{}
}

// TestOracleEscalationDispatchNamesTheStall covers behaviours 3-7: the
// dispatch builder produces an identifiable Oracle dispatch overwriting the
// same artifact, distinct from the Scout dispatch it replaces, with the
// escalation announced in the brief -- and the plan-research-escalate cobra
// command surfaces that dispatch (or a clean error) over the JSON envelope.
func TestOracleEscalationDispatchNamesTheStall(t *testing.T) {
	candidate := phaseResearchCandidate{ID: 3, Name: "Wire exporter", Description: "Ship the exporter"}
	root := "/tmp/escalation-root"
	goal := "Build the exporter"

	t.Run("dispatch_shape", func(t *testing.T) {
		dispatch := phaseResearchEscalationDispatch(root, goal, candidate, escalationTestSurvey(), 82, 95)

		if dispatch.Caste != "oracle" {
			t.Errorf("Caste = %q, want oracle", dispatch.Caste)
		}
		if dispatch.AgentName != "aether-oracle" {
			t.Errorf("AgentName = %q, want aether-oracle", dispatch.AgentName)
		}
		if dispatch.Stage != phaseResearchStage {
			t.Errorf("Stage = %q, want %q", dispatch.Stage, phaseResearchStage)
		}
		if dispatch.Wave != 1 {
			t.Errorf("Wave = %d, want 1", dispatch.Wave)
		}
		if len(dispatch.Outputs) != 1 || dispatch.Outputs[0] != "phase-3-research.md" {
			t.Errorf("Outputs = %v, want [phase-3-research.md]", dispatch.Outputs)
		}
	})

	t.Run("name_differs_from_scout_dispatch", func(t *testing.T) {
		escalation := phaseResearchEscalationDispatch(root, goal, candidate, escalationTestSurvey(), 82, 95)
		scoutDispatches := plannedPhaseResearchDispatches(root, "deep", goal, []phaseResearchCandidate{candidate}, escalationTestSurvey(), false, map[int]bool{3: true})
		if len(scoutDispatches) != 1 {
			t.Fatalf("scout dispatches = %d, want 1", len(scoutDispatches))
		}
		if escalation.Name == scoutDispatches[0].Name {
			t.Errorf("escalation dispatch name %q collides with scout dispatch name for the same phase and root", escalation.Name)
		}
	})

	t.Run("brief_names_the_stall", func(t *testing.T) {
		dispatch := phaseResearchEscalationDispatch(root, goal, candidate, escalationTestSurvey(), 82, 95)
		if !strings.Contains(dispatch.Brief, "82") {
			t.Errorf("brief does not name the stalled confidence (82): %s", dispatch.Brief)
		}
		if !strings.Contains(dispatch.Brief, "95") {
			t.Errorf("brief does not name the target (95): %s", dispatch.Brief)
		}
		if !strings.Contains(dispatch.Brief, "Phase 3") && !strings.Contains(dispatch.Brief, "phase 3") {
			t.Errorf("brief does not name the phase: %s", dispatch.Brief)
		}
		if !strings.Contains(dispatch.Brief, "Phase Domain Research") {
			t.Errorf("brief is missing the original research mission: %s", dispatch.Brief)
		}
	})

	t.Run("cli_prints_dispatch_and_exits_zero", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan:    colony.Plan{Phases: []colony.Phase{{ID: 3, Name: "Wire exporter", Description: "Ship the exporter", Status: colony.PhaseReady}}},
		})
		resetRootCmd(t)
		forceJSONOutputModeForTest(t)
		var buf bytes.Buffer
		stdout = &buf
		rootCmd.SetArgs([]string{"plan-research-escalate", "--phase", "3", "--confidence", "82", "--target", "95"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("plan-research-escalate failed: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("expected ok:true, got %v", env)
		}
		result, ok := env["result"].(map[string]interface{})
		if !ok {
			t.Fatalf("result is not a map: %v", env["result"])
		}
		dispatch, ok := result["dispatch"].(map[string]interface{})
		if !ok {
			t.Fatalf("result missing dispatch: %v", result)
		}
		if dispatch["caste"] != "oracle" {
			t.Errorf("dispatch.caste = %v, want oracle", dispatch["caste"])
		}
	})

	t.Run("unknown_phase_returns_clean_error_not_panic", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan:    colony.Plan{Phases: []colony.Phase{{ID: 3, Name: "Wire exporter", Description: "Ship the exporter", Status: colony.PhaseReady}}},
		})
		resetRootCmd(t)
		forceJSONOutputModeForTest(t)
		var outBuf, errBuf bytes.Buffer
		stdout = &outBuf
		stderr = &errBuf
		rootCmd.SetArgs([]string{"plan-research-escalate", "--phase", "99", "--confidence", "82", "--target", "95"})

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("plan-research-escalate panicked on unknown phase: %v", r)
			}
		}()
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("rootCmd.Execute() returned a cobra error rather than a JSON envelope: %v", err)
		}
		env := parseEnvelope(t, errBuf.String())
		if env["ok"] != false {
			t.Fatalf("expected ok:false for unknown phase, got %v", env)
		}
	})
}
