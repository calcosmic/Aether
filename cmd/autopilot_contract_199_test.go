package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"gopkg.in/yaml.v3"
)

func autopilotContract199Facts(goal string, phases []colony.Phase) LifecycleFacts {
	state := colony.ColonyState{State: colony.StateREADY, CurrentPhase: 1, Plan: colony.Plan{Phases: phases}}
	if goal != "" {
		state.Goal = &goal
	}
	confirmed := LifecycleFactSource{Domain: "state", Path: "COLONY_STATE.json", Provenance: LifecycleFactConfirmed}
	return LifecycleFacts{
		CapturedAt: time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC),
		State:      LifecycleFact[colony.ColonyState]{Value: state, Source: confirmed},
		Identity: LifecycleFact[LifecycleIdentityFacts]{
			Value:  LifecycleIdentityFacts{Goal: goal, Standing: string(state.State)},
			Source: lifecycleDerivedSource("identity", confirmed),
		},
		Progress: LifecycleFact[LifecycleProgressFacts]{
			Value:  LifecycleProgressFacts{CurrentPhase: 1, Phases: phases},
			Source: lifecycleDerivedSource("progress", confirmed),
		},
		Signals: LifecycleFact[[]colony.PheromoneSignal]{
			Value: []colony.PheromoneSignal{
				{ID: "focus-1", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"preserve compatibility"}`)},
				{ID: "redirect-1", Type: "REDIRECT", Active: true, Content: json.RawMessage(`{"text":"do not auto-seal"}`)},
			},
			Source: LifecycleFactSource{Domain: "signals", Path: "pheromones.json", Provenance: LifecycleFactConfirmed},
		},
	}
}

func TestAutopilotContract199InvalidZeroWrite(t *testing.T) {
	saveGlobals(t)

	t.Run("missing initialized colony", func(t *testing.T) {
		root := t.TempDir()
		store = nil
		before := hashDirContents(t, root)
		result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{})
		if err != nil {
			t.Fatalf("early run returned a plumbing error instead of the typed refusal: %v", err)
		}
		after := hashDirContents(t, root)
		if before != after {
			t.Fatalf("missing-colony preflight mutated the workspace: %s -> %s", before, after)
		}
		visual := renderRunCompatibilityVisual(result)
		for _, want := range []string{"Autopilot did not start", "Missing: an initialized colony.", "State: unchanged.", `Next: /ant-init "goal"`} {
			if !strings.Contains(visual, want) {
				t.Errorf("missing-colony refusal lacks %q:\n%s", want, visual)
			}
		}
	})

	t.Run("missing accepted plan", func(t *testing.T) {
		root := t.TempDir()
		dataDir := filepath.Join(root, ".aether", "data")
		if err := os.MkdirAll(dataDir, 0o755); err != nil {
			t.Fatal(err)
		}
		var err error
		store, err = storage.NewStore(dataDir)
		if err != nil {
			t.Fatal(err)
		}
		goal := "Accepted goal without a plan"
		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{Goal: &goal, State: colony.StateIDLE}); err != nil {
			t.Fatal(err)
		}
		before := hashDirContents(t, root)
		result, runErr := runCompatibilityAutopilot(root, runCompatibilityOptions{})
		if runErr != nil {
			t.Fatalf("early run returned a plumbing error instead of the typed refusal: %v", runErr)
		}
		after := hashDirContents(t, root)
		if before != after {
			t.Fatalf("missing-plan preflight mutated the workspace: %s -> %s", before, after)
		}
		visual := renderRunCompatibilityVisual(result)
		for _, want := range []string{"Autopilot did not start", "Missing: an accepted plan.", "State: unchanged.", "Next: /ant-plan"} {
			if !strings.Contains(visual, want) {
				t.Errorf("missing-plan refusal lacks %q:\n%s", want, visual)
			}
		}
	})
}

func TestAutopilotContract199OperatingCard(t *testing.T) {
	facts := autopilotContract199Facts("Ship the accepted contract", []colony.Phase{
		{ID: 1, Name: "Already done", Status: colony.PhaseCompleted},
		{ID: 2, Name: "First remaining", Status: colony.PhaseReady},
		{ID: 4, Name: "Last remaining", Status: colony.PhasePending},
	})
	preflight := buildAutopilotPreflight(facts)
	if !preflight.Valid || preflight.FirstPhase != 2 || preflight.LastPhase != 4 {
		t.Fatalf("preflight = %+v, want accepted remaining range 2..4", preflight)
	}
	visual := renderAutopilotOperatingContract(preflight)
	wants := []string{
		"━━ ⚡ A U T O P I L O T ━━",
		"Goal: Ship the accepted contract",
		"Range: Phase 2 through Phase 4",
		"Active pheromones: FOCUS: preserve compatibility; REDIRECT: do not auto-seal",
		"May revise: tasks, dependencies, sequencing, and implementation details when evidence requires it.",
		"Will pause before changing: goal, promised behavior, scope, risk authority, or acceptance criteria.",
		"Also pauses for: safety failure, corrupt state, missing authority, a material owner decision, or an invalidating failed dependency.",
		"Starting now.",
	}
	last := -1
	for _, want := range wants {
		at := strings.Index(visual, want)
		if at < 0 {
			t.Fatalf("operating card lacks %q:\n%s", want, visual)
		}
		if at <= last {
			t.Fatalf("operating card field %q is out of order:\n%s", want, visual)
		}
		last = at
	}
}

func TestAutopilotContract199StartsImmediately(t *testing.T) {
	saveGlobals(t)
	_, root := seedRunFixture(t, 1)
	installAutopilotRunTestDeps(t)
	buildCalls := 0
	runAutopilotBuild = func(string, int, []string, bool, codexBuildOptions) (map[string]interface{}, error) {
		buildCalls++
		return nil, errors.New("fixture stops after proving dispatch")
	}
	result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{})
	if err != nil {
		t.Fatalf("valid invocation failed before execution: %v", err)
	}
	if buildCalls != 1 {
		t.Fatalf("build calls = %d, want immediate first dispatch", buildCalls)
	}
	card := renderAutopilotOperatingContract(result["preflight"].(AutopilotPreflight))
	if strings.Contains(strings.ToLower(card), "continue?") || strings.Contains(strings.ToLower(card), "confirm") {
		t.Fatalf("valid invocation introduced a second consent prompt:\n%s", card)
	}
}

func TestAutopilotContract199AuthorityFence(t *testing.T) {
	for _, target := range []autopilotAuthorityTarget{
		autopilotAuthorityGoal,
		autopilotAuthorityPromisedBehavior,
		autopilotAuthorityScope,
		autopilotAuthorityRisk,
		autopilotAuthorityAcceptance,
	} {
		decision := evaluateAutopilotAuthorityProposal(autopilotAuthorityProposal{Target: target, Before: "accepted", After: "proposed"})
		if decision.Allowed || decision.Disposition != autopilotDispositionPause || decision.Code != autopilotTriggerMissingAuthority {
			t.Errorf("owner target %q decision = %+v, want a pre-mutation authority pause", target, decision)
		}
	}
	for _, target := range []autopilotAuthorityTarget{autopilotAuthorityTasks, autopilotAuthorityDependencies, autopilotAuthoritySequencing, autopilotAuthorityImplementation} {
		if decision := evaluateAutopilotAuthorityProposal(autopilotAuthorityProposal{Target: target}); !decision.Allowed {
			t.Errorf("implementation target %q unexpectedly requires owner authority: %+v", target, decision)
		}
	}
}

func autopilotContract199Failure() autopilotRepairFailure {
	return autopilotRepairFailure{
		Phase:            2,
		Attempt:          "attempt-2",
		Check:            "go test ./cmd",
		Evidence:         []string{"tests failed"},
		PlannedAction:    "repair the implicated command package files",
		Baseline:         "sha256:before",
		PermittedScope:   []string{"cmd/compatibility_cmds.go"},
		ScopeSafe:        true,
		SafetySafe:       true,
		AuthoritySafe:    true,
		AffectedPaths:    []string{"phase-2", "phase-3"},
		IndependentPaths: []string{"phase-4"},
	}
}

func TestAutopilotContract199RepairEligibility(t *testing.T) {
	failure := autopilotContract199Failure()
	if evaluation := classifyAutopilotRepairFailure(failure, 1); !evaluation.Eligible || evaluation.Pause {
		t.Fatalf("bounded in-scope repair classified incorrectly: %+v", evaluation)
	}
	failure.AuthoritySafe = false
	if evaluation := classifyAutopilotRepairFailure(failure, 1); evaluation.Eligible || !evaluation.Pause || evaluation.Reason != "missing authority" {
		t.Fatalf("owner-authority repair classified incorrectly: %+v", evaluation)
	}
}

func TestAutopilotContract199RepairReceipt(t *testing.T) {
	ledger := newAutopilotRepairLedger("run-199-11", 1)
	receipt, err := beginAutopilotRepair(&ledger, autopilotContract199Failure(), time.Date(2026, 9, 4, 9, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ID != "repair-run-199-11-001" || receipt.Status != autopilotRepairPlanned {
		t.Fatalf("planned receipt = %+v", receipt)
	}
	if receipt.Phase != 2 || receipt.Attempt != "attempt-2" || receipt.Check == "" || receipt.PlannedAction == "" || receipt.Baseline == "" {
		t.Fatalf("receipt lost pre-mutation facts: %+v", receipt)
	}
	if receipt.BudgetBefore != 1 || receipt.BudgetRemaining != 1 || ledger.Remaining != 1 {
		t.Fatalf("planning consumed repair budget before verification: receipt=%+v ledger=%+v", receipt, ledger)
	}
}

func TestAutopilotContract199RepairVerification(t *testing.T) {
	ledger := newAutopilotRepairLedger("run-199-11", 1)
	receipt, err := beginAutopilotRepair(&ledger, autopilotContract199Failure(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := completeAutopilotRepair(&ledger, receipt.ID, []string{"go test ./cmd: FAIL"}, false, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got := ledger.Receipts[0]
	if got.Status != autopilotRepairFailed || got.Verification.Check != "go test ./cmd" || got.Verification.Passed || len(got.Verification.Evidence) == 0 {
		t.Fatalf("failed named verification was not attached: %+v", got)
	}
	if ledger.Remaining != 0 || got.BudgetRemaining != 0 {
		t.Fatalf("failed verification did not decrement exactly once: %+v", ledger)
	}
	if err := completeAutopilotRepair(&ledger, receipt.ID, []string{"duplicate"}, false, time.Now().UTC()); err == nil || ledger.Remaining != 0 {
		t.Fatalf("duplicate completion consumed or rewrote budget: err=%v ledger=%+v", err, ledger)
	}
}

func TestAutopilotContract199BudgetExhaustion(t *testing.T) {
	failure := autopilotContract199Failure()
	evaluation := classifyAutopilotRepairFailure(failure, 0)
	if evaluation.Eligible || evaluation.Pause || evaluation.Reason != "repair budget exhausted" {
		t.Fatalf("exhausted repair classification = %+v", evaluation)
	}
}

func TestAutopilotContract199DebtReport(t *testing.T) {
	ledger := newAutopilotRepairLedger("run-199-11", 0)
	failure := autopilotContract199Failure()
	recordAutopilotRepairDebt(&ledger, failure, "repair budget exhausted", false)
	report := autopilotRepairReportFields(ledger)
	if len(report.Debt) != 1 || report.RemainingBudget != 0 || !report.BudgetExhausted {
		t.Fatalf("repair debt report = %+v", report)
	}
	if len(report.Receipts) != 0 || report.Debt[0].Summary == "" {
		t.Fatalf("repair report hid its exhausted failure: %+v", report)
	}
}

func TestAutopilotContract199IndependentContinuation(t *testing.T) {
	continued, skipped := autopilotDependencyRouting(autopilotContract199Failure())
	if strings.Join(continued, ",") != "phase-4" || strings.Join(skipped, ",") != "phase-2,phase-3" {
		t.Fatalf("dependency routing continued=%v skipped=%v", continued, skipped)
	}
}

func TestAutopilotContract199CompletionDoesNotSeal(t *testing.T) {
	visual := renderAutopilotComplete(3)
	for _, want := range []string{"3 phases built and verified", "Sealing remains an explicit owner action", "`aether seal`"} {
		if !strings.Contains(visual, want) {
			t.Errorf("completion lacks %q:\n%s", want, visual)
		}
	}
	for _, forbidden := range []string{"auto-seal", "--force", "sealed automatically"} {
		if strings.Contains(strings.ToLower(visual), forbidden) {
			t.Errorf("completion contains autonomous seal language %q:\n%s", forbidden, visual)
		}
	}
}

func TestAutopilotWrapperContract199(t *testing.T) {
	const (
		description = "Autopilot the remaining accepted phases within the displayed safety contract."
		runtimeCall = "AETHER_OUTPUT_MODE=visual aether run $ARGUMENTS"
		source      = ".aether/commands/run.yaml"
	)
	repoRoot := filepath.Clean(filepath.Join(".."))
	yamlPath := filepath.Join(repoRoot, ".aether", "commands", "run.yaml")
	rawYAML, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Runtime     struct {
			Command string `yaml:"command"`
		} `yaml:"runtime"`
		CeremonyContract struct {
			Authority   string `yaml:"authority"`
			WrapperRole string `yaml:"wrapper_role"`
		} `yaml:"ceremony_contract"`
		Guardrails []string `yaml:"guardrails"`
	}
	if err := yaml.Unmarshal(rawYAML, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Name != "ant-run" || spec.Description != description || spec.Runtime.Command != runtimeCall {
		t.Fatalf("canonical run wrapper contract drifted: %+v", spec)
	}
	canonicalContract := strings.ToLower(strings.Join(append([]string{spec.CeremonyContract.Authority, spec.CeremonyContract.WrapperRole}, spec.Guardrails...), "\n"))
	for _, required := range []string{"invocation is consent", "typed", "owner authority", "runtime result", "exactly once", "do not write", "do not parse", "do not fabricate", "do not auto-seal"} {
		if !strings.Contains(canonicalContract, required) {
			t.Errorf("canonical wrapper contract lacks %q:\n%s", required, string(rawYAML))
		}
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant-run.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "run.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "run.md"),
	}
	var canonicalBody string
	for _, path := range wrapperPaths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		text := string(raw)
		lines := strings.Split(text, "\n")
		wantHeader := "<!-- Aether-managed: runtime spec at " + source + ". Synced by aether update. -->"
		if len(lines) == 0 || lines[0] != wantHeader {
			t.Errorf("%s lacks canonical source linkage", path)
		}
		if !strings.Contains(text, `description: "`+description+`"`) {
			t.Errorf("%s description does not match the canonical sentence", path)
		}
		if strings.Count(text, runtimeCall) != 1 {
			t.Errorf("%s has %d runtime calls, want exactly one", path, strings.Count(text, runtimeCall))
		}
		lower := strings.ToLower(text)
		for _, required := range []string{"invoking /ant-run is consent", "typed runtime result", "owner-authority pause", "do not write", "do not parse", "do not fabricate", "do not auto-seal"} {
			if !strings.Contains(lower, required) {
				t.Errorf("%s lacks %q", path, required)
			}
		}
		for _, forbidden := range []string{"continue?", "--force", "aether seal", "colony_state.json", "savejson(", "writefile("} {
			if strings.Contains(lower, forbidden) {
				t.Errorf("%s contains forbidden host behavior %q", path, forbidden)
			}
		}
		body := strings.Join(lines[6:], "\n")
		if canonicalBody == "" {
			canonicalBody = body
		} else if body != canonicalBody {
			t.Errorf("generated wrapper %s is not semantically identical to the other run wrappers", path)
		}
	}
}
