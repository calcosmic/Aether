package cmd

// Decision D-01 for Phase 160 (Fail Loudly): when a documented CLI call cannot
// execute, what should happen is not uniform.
//
//   - A SAFETY GATE that cannot run must HALT the run. A build must not report
//     "passed" with its security scan, verification, or claims check silently
//     unexecuted. That is precisely how `check-antipattern` sat broken and
//     unnoticed: it failed into a `/dev/null` redirect and the build carried on.
//   - An ENRICHMENT call (survey, skill matching, progress rendering) that
//     cannot run must emit a loud, visible warning and let the run continue in a
//     degraded state. Halting a build because a progress bar failed to render
//     would be its own kind of broken.
//
// This classification lives in non-test code on purpose. D-01 requires it to be
// "encoded where the audit can test it" — a comment in a plan or a table in a
// markdown file cannot fail a build. `TestGateClassifiedCallsHaveGateWiring` in
// command_call_audit_test.go asserts against this map.

type commandCallSeverity string

const (
	// severityGate: inability to execute must halt the run.
	severityGate commandCallSeverity = "gate"
	// severityEnrichment: inability to execute warns loudly, run continues degraded.
	severityEnrichment commandCallSeverity = "enrichment"
)

// gateClassifiedCommands lists the subcommands whose failure-to-execute is a
// halting condition. Everything not named here is enrichment by default —
// deliberately, so that adding a new non-safety subcommand does not require
// touching this file, while promoting something TO a gate is an explicit,
// reviewable act.
//
// Each entry records why it halts, because "which of these is really a safety
// gate" is exactly the judgement a future reader will need to re-make.
var gateClassifiedCommands = map[string]string{
	"check-antipattern": "security scan for exposed secrets and debug artifacts; a phase must not pass with it unexecuted (LOUD-02, ROADMAP SC#2)",
	"verify-claims":     "verifies the builder's claims about what it changed; unexecuted, nothing checks that the reported work happened",
	"gate-check":        "the gate evaluation itself; if it cannot run there is no pass/fail to honour",
}

// commandCallSeverityFor returns the D-01 severity for a subcommand name.
func commandCallSeverityFor(name string) commandCallSeverity {
	if _, ok := gateClassifiedCommands[name]; ok {
		return severityGate
	}
	return severityEnrichment
}
