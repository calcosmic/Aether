package cmd

import (
	"os"
	"strings"
)

// Per-caste model routing (Phase 201 plan 14, D-15c).
//
// This is a POLICY layer that sits above resolveCasteModel's own
// display-name resolution (cmd/codex_visuals.go) and above the existing
// spend ledger -- it never adds a new ledger field: the ledger already
// accounts whichever model actually ran per worker
// (cmd/spend_ledger.go's spendRow carries no Model field at all, and needs
// none). SYN-201-14 (.planning/phases/201-queen-led-work-cycle/
// 201-CLASSIC-SYNTHESIS.md) resolved this exact question before this file
// was written: a policy layer, not new ledger fields.
//
// A caste is named here ONLY where its own work is mechanical -- following a
// fixed procedure with no synthesis or judgment step to lose -- and every
// entry carries a plain-English reason, in the same voice
// cmd/caste_model_reason.go already uses for the opposite claim (why a role
// needs the expensive model). Routing is a stated policy decision, never a
// function of timing data (D-16): nothing in this file reads a
// jobTelemetryRecord, and the report-only guard
// (cmd/job_telemetry_test.go's TestTelemetryIsReportOnly) is unaffected by
// this file's existence.
//
// A caste present in casteModelReasons (cmd/caste_model_reason.go) -- the
// existing, recorded evidence that a role's work needs the expensive
// model's judgment -- is refused a route structurally
// (resolveCasteModelRoute checks that table directly), never by trusting
// this table to stay disjoint from it by hand.
var casteModelRoutes = map[string]struct {
	slot   string
	reason string
}{
	"porter": {
		slot:   "haiku",
		reason: "runs the fixed publish and deploy commands for work that has already been reviewed and approved -- there is no judgment call left to make, so a faster model produces the identical result",
	},
}

// resolveCasteModelRoute reports the routing policy's answer for caste: the
// resolved model (after the same ANTHROPIC_DEFAULT_<SLOT>_MODEL env
// indirection resolveCasteModel already applies, so a caller can swap this
// value in directly), the stated reason, and whether a route exists at all.
//
// A caste with no entry in casteModelRoutes returns routed=false with an
// empty model and reason -- callers must fall back to resolveCasteModel
// exactly as before this plan (TestUnroutedCasteModelResolutionIsUnchanged).
//
// A caste casteModelReasons names is refused here even if casteModelRoutes
// were ever mistakenly populated for it: the quality-sensitivity table is
// checked FIRST, unconditionally, before the routing table is ever consulted
// (TestQualitySensitiveCasteIsNeverRouted).
func resolveCasteModelRoute(caste string) (model string, reason string, routed bool) {
	key := normalizeCasteKey(caste)
	if casteModelReason(key) != "" {
		return "", "", false
	}
	route, ok := casteModelRoutes[key]
	if !ok {
		return "", "", false
	}
	model = route.slot
	if override := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_" + strings.ToUpper(route.slot) + "_MODEL")); override != "" {
		model = override
	}
	return model, route.reason, true
}
