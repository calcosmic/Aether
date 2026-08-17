package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// WS7 — critics must bring solutions, and the classic flag triage grammar is
// restored: blocker = stops the line and cannot be parked; issue = shown,
// parkable; note = parked. Every block carries its way forward in the same
// breath — the reviewer's fix or the Fixer.

func writeTestFlags(t *testing.T, flags ...colony.FlagEntry) {
	t.Helper()
	if err := store.SaveJSON("pending-decisions.json", colony.FlagsFile{Version: "1.0", Decisions: flags}); err != nil {
		t.Fatalf("write flags: %v", err)
	}
}

func readTestFlags(t *testing.T) []colony.FlagEntry {
	t.Helper()
	var ff colony.FlagsFile
	if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("read flags: %v", err)
	}
	return ff.Decisions
}

// TestBlockerFlagBlocksContinue kills the lying gate: checkNoCriticalFlags
// claimed to run "every time for safety" but never opened
// pending-decisions.json, so a blocker raised with /ant-flag did not block
// anything. The classic Iron Law: no phase advancement with unresolved
// blockers.
func TestBlockerFlagBlocksContinue(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	writeTestFlags(t, colony.FlagEntry{ID: "f1", Type: "blocker", Description: "auth bypass found in login flow", CreatedAt: time.Now().UTC().Format(time.RFC3339)})
	check := checkUnresolvedBlockerFlags()
	if check.Passed {
		t.Fatalf("an unresolved blocker flag did not fail the flags gate — the Iron Law is still unenforced")
	}
	if !strings.Contains(check.Detail, "auth bypass") {
		t.Fatalf("gate detail does not name the blocker: %q", check.Detail)
	}

	// The wiring: the CONTINUE gates include the Iron Law check as a
	// blocking entry — advancement-scoped (the pre-build gates deliberately
	// do not run it; you may build with an open blocker, not advance past
	// it).
	gates := runCodexContinueGates(colony.Phase{ID: 1, Name: "p"}, codexContinueManifest{}, codexContinueVerificationReport{ChecksPassed: true}, codexContinueAssessment{}, time.Now().UTC(), nil)
	wired := false
	for _, c := range gates.Checks {
		if c.Name == "no_unresolved_blockers" && !c.Passed {
			wired = true
			if c.FixHint == "" || len(c.RecoveryOptions) == 0 {
				t.Fatalf("Iron Law gate blocks without a way forward: %+v", c)
			}
		}
	}
	if !wired {
		t.Fatalf("the Iron Law check is not wired into the continue gates: %+v", gates.Checks)
	}
	if gates.Passed {
		t.Fatalf("continue gates passed with an unresolved blocker flag")
	}

	// A resolved blocker and a mere issue must NOT block.
	writeTestFlags(t,
		colony.FlagEntry{ID: "f1", Type: "blocker", Description: "fixed", Resolved: true, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		colony.FlagEntry{ID: "f2", Type: "issue", Description: "smell worth a look", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	)
	if check := checkUnresolvedBlockerFlags(); !check.Passed {
		t.Fatalf("resolved blockers / open issues wrongly block: %q", check.Detail)
	}
}

func TestBlockersCannotBeAcknowledged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	writeTestFlags(t,
		colony.FlagEntry{ID: "b1", Type: "blocker", Description: "hard stop", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		colony.FlagEntry{ID: "i1", Type: "issue", Description: "parkable", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	)

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	// Classic law: blockers cannot be parked.
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"flag-acknowledge", "--id", "b1"})
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("a blocker was acknowledged — the classic lifecycle forbids parking blockers: %s", buf.String())
	}
	for _, flag := range readTestFlags(t) {
		if flag.ID == "b1" && flag.Acknowledged {
			t.Fatalf("blocker was marked acknowledged despite the refusal")
		}
	}

	// Issues park cleanly, with a timestamp.
	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"flag-acknowledge", "--id", "i1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("acknowledge issue: %v", err)
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("acknowledging an issue failed: %s", errBuf.String())
	}
	for _, flag := range readTestFlags(t) {
		if flag.ID == "i1" {
			if !flag.Acknowledged || flag.AcknowledgedAt == "" {
				t.Fatalf("issue not parked with a timestamp: %+v", flag)
			}
		}
	}
}

func TestChaosBlockersNeverAutoResolve(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC().Format(time.RFC3339)
	writeTestFlags(t,
		colony.FlagEntry{ID: "m1", Type: "blocker", Source: "verification", Description: "tests failed", CreatedAt: now},
		colony.FlagEntry{ID: "c1", Type: "blocker", Source: "chaos-standalone", Description: "state corruption on double-submit", CreatedAt: now},
		colony.FlagEntry{ID: "u1", Type: "blocker", Source: "", Description: "owner said wait", CreatedAt: now},
		colony.FlagEntry{ID: "i1", Type: "issue", Source: "verification", Description: "issue stays untouched", CreatedAt: now},
	)

	resolved := autoResolveVerificationBlockers(true, 4)
	if resolved != 1 {
		t.Fatalf("auto-resolved %d flags, want exactly 1 (the machine-raised verification blocker)", resolved)
	}
	for _, flag := range readTestFlags(t) {
		switch flag.ID {
		case "m1":
			if !flag.Resolved {
				t.Fatalf("machine-raised verification blocker did not clear on green verification")
			}
			if !strings.Contains(flag.Resolution, "verification passed") {
				t.Fatalf("resolution does not record the evidence: %q", flag.Resolution)
			}
		case "c1":
			if flag.Resolved {
				t.Fatalf("a Chaos-raised blocker auto-resolved — the v2.4.3 exemption is gone; a Chaos finding always demands a human")
			}
		case "u1":
			if flag.Resolved {
				t.Fatalf("a user-raised blocker auto-resolved — the owner's hold was waved through by a green build")
			}
		case "i1":
			if flag.Resolved {
				t.Fatalf("auto-resolve touched a non-blocker flag")
			}
		}
	}
}

func TestAutoResolveRequiresBuildPass(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC().Format(time.RFC3339)
	writeTestFlags(t, colony.FlagEntry{ID: "m1", Type: "blocker", Source: "verification", Description: "tests failed", CreatedAt: now})

	// Red verification clears nothing — the evidence is the trigger.
	if resolved := autoResolveVerificationBlockers(false, 4); resolved != 0 {
		t.Fatalf("flags cleared without passing verification: %d", resolved)
	}

	// And the CLI's age-based path never runs implicitly: no --max-days, no
	// resolution (age is not evidence).
	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"flag-auto-resolve"})
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("flag-auto-resolve ran without an explicit --max-days: %s", buf.String())
	}
	for _, flag := range readTestFlags(t) {
		if flag.Resolved {
			t.Fatalf("flag resolved by the refused invocation")
		}
	}
}

func TestStructuredBlockingFindingBlocksContinue(t *testing.T) {
	now := time.Now().UTC()
	steps := []codexContinueWorkerFlowStep{
		{
			Stage:  "review",
			Caste:  "watcher",
			Name:   "Sentinel-2",
			Status: "completed",
			Findings: []codexReviewFinding{
				{Domain: "testing", Severity: "HIGH", Blocking: true, Description: "the CSV export writes headers twice", Suggestion: "guard the header write with the firstRow flag in export.go:88"},
			},
		},
	}
	planned := []codexContinueExternalDispatch{{Stage: "review", Caste: "watcher", Name: "Sentinel-2"}}
	report := externalContinueReviewReport(4, steps, now, false, colony.VerificationDepthStandard, planned)
	if report.Passed {
		t.Fatalf("a completed review with a typed blocking finding passed — structured findings are still decorative")
	}
	found := false
	for _, blocker := range report.BlockingIssues {
		if strings.Contains(blocker, "fix: guard the header write") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the block does not carry the reviewer's fix in the same breath: %v", report.BlockingIssues)
	}

	// A completed review with only non-blocking findings must still pass.
	steps[0].Findings = []codexReviewFinding{{Domain: "testing", Severity: "LOW", Description: "cosmetic naming", Suggestion: "rename later"}}
	if report := externalContinueReviewReport(4, steps, now, false, colony.VerificationDepthStandard, planned); !report.Passed {
		t.Fatalf("non-blocking findings blocked the phase: %v", report.BlockingIssues)
	}
}

func TestFixlessBlockingFindingOffersFixer(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC()
	steps := []codexContinueWorkerFlowStep{
		{
			Stage:  "review",
			Caste:  "gatekeeper",
			Name:   "Warden-9",
			Status: "completed",
			Findings: []codexReviewFinding{
				{Domain: "security", Severity: "CRITICAL", Description: "session token logged in plaintext"},
			},
		},
	}
	planned := []codexContinueExternalDispatch{{Stage: "review", Caste: "gatekeeper", Name: "Warden-9"}}
	report := externalContinueReviewReport(4, steps, now, false, colony.VerificationDepthStandard, planned)
	if report.Passed {
		t.Fatalf("a CRITICAL finding did not block")
	}
	found := false
	for _, blocker := range report.BlockingIssues {
		if strings.Contains(blocker, "/ant-unblock") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a fix-less block does not offer the Fixer in the same breath: %v", report.BlockingIssues)
	}

	// The gate-results bridge: the Fixer's intake (aether unblock reads
	// gate-results-<N>.json) receives the findings and the unblock option.
	appendReviewFindingsGateResult(4, steps, now)
	entries, err := gateResultsReadPhase(4)
	if err != nil {
		t.Fatalf("read gate results: %v", err)
	}
	var reviewGate *GateCheckResult
	for i := range entries {
		if entries[i].Name == "review_findings" {
			reviewGate = &entries[i]
		}
	}
	if reviewGate == nil {
		t.Fatalf("no review_findings gate entry written for the Fixer's intake")
	}
	if reviewGate.FixHint == "" {
		t.Fatalf("review_findings gate entry has no fix hint")
	}
	unblockNamed := false
	for _, option := range reviewGate.RecoveryOptions {
		if strings.Contains(option, "/ant-unblock") {
			unblockNamed = true
		}
	}
	if !unblockNamed {
		t.Fatalf("recovery options do not name /ant-unblock: %v", reviewGate.RecoveryOptions)
	}
}

func TestReviewLedgerRejectsCriticalWithoutSuggestion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"review-ledger-write", "--domain", "security", "--phase", "1",
		"--findings", `[{"severity":"CRITICAL","file":"a.go","line":1,"category":"secrets","description":"api key committed"}]`,
		"--agent", "gatekeeper"})
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("a CRITICAL finding with no suggestion was accepted: %s", buf.String())
	}
	if !strings.Contains(errBuf.String(), "api key committed") {
		t.Fatalf("rejection does not name the finding: %s", errBuf.String())
	}

	// With a suggestion the same finding writes cleanly.
	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"review-ledger-write", "--domain", "security", "--phase", "1",
		"--findings", `[{"severity":"CRITICAL","file":"a.go","line":1,"category":"secrets","description":"api key committed","suggestion":"move it to an env var and rotate the key"}]`,
		"--agent", "gatekeeper"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("valid write failed: %v", err)
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("valid write refused: %s", errBuf.String())
	}
}

func TestBlockedContinueOutputNamesUnblock(t *testing.T) {
	result := map[string]interface{}{
		"blocking_issues": []interface{}{"Sentinel-2 blocking finding: headers written twice (fix: guard with firstRow flag)"},
		"gates": map[string]interface{}{
			"checks": []interface{}{
				map[string]interface{}{
					"name": "no_critical_flags", "passed": false,
					"fix_hint":         "Resolve critical flags before continuing",
					"recovery_options": []interface{}{"Fix the issue, then resolve its flag: /ant-flags --resolve <id> \"what fixed it\""},
				},
			},
		},
	}
	out := renderContinueBlockedVisual(colony.ColonyState{}, colony.Phase{ID: 2, Name: "phase"}, result, colony.VerificationDepthStandard)
	if !strings.Contains(out, "Way forward") {
		t.Fatalf("blocked output has no way-forward section:\n%s", out)
	}
	if !strings.Contains(out, "/ant-unblock") {
		t.Fatalf("blocked output does not offer the Fixer:\n%s", out)
	}
	if !strings.Contains(out, "Resolve critical flags") {
		t.Fatalf("blocked output drops the gates' fix hints:\n%s", out)
	}
}

// Contract anchors: the agent definitions demand solutions, in both wrapper
// trees (parity keeps the copies identical; these anchors keep the CONTENT).
func TestWatcherContractCarriesSuggestionAndBlocking(t *testing.T) {
	for _, path := range []string{"../.claude/agents/ant/aether-watcher.md", "../.opencode/agents/aether-watcher.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, anchor := range []string{"`suggestion`", "`blocking`", "smallest change that would make this issue pass"} {
			if !strings.Contains(text, anchor) {
				t.Fatalf("%s lost the critics-bring-solutions contract anchor %q", path, anchor)
			}
		}
	}
}

func TestChaosContractCarriesPerScenarioHardening(t *testing.T) {
	for _, path := range []string{"../.claude/agents/ant/aether-chaos.md", "../.opencode/agents/aether-chaos.md", "../.claude/commands/ant/chaos.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(raw), "suggested_hardening") {
			t.Fatalf("%s lost the per-scenario suggested_hardening contract", path)
		}
	}
}
