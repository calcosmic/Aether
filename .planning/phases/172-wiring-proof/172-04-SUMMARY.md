---
phase: 172-wiring-proof
plan: "04"
subsystem: testing
tags: [go, static-analysis, ci-gate, wiring-proof, documentation]

# Dependency graph
requires:
  - phase: 172-wiring-proof (plan 02)
    provides: "the orphan reachability ratchet (TestNoRegisteredSubcommandIsUnreferenced), its orphanAllowlistEntry shape, loadOrphanAllowlist, and TestOrphanAllowlistOnlyShrinks's set-membership diff pattern, which this plan mirrors rather than duplicates"
provides:
  - "cmd/cli_flag_audit_test.go: skipSubcommands promoted to a package-level []flagAuditSkipEntry, guarded by TestFlagAuditSkipListOnlyShrinks against a committed baseline"
  - "cmd/testdata/flag_audit_skiplist_baseline.json: the committed shrink-only baseline for the flag audit's exemption list"
  - ".aether/docs/orphan-allowlist-policy.md: the plain-English policy for both guarded lists, stating the real seeded numbers (278 orphans, 6 owner_phase 178)"
  - "TestAllowlistPolicyNamesEveryGuardedFile: fails if the policy doc stops naming any of the four guarded files, derived from code not prose"
affects: [178-skill-authoring-hardening (reads the policy doc's stated phase-178 count), any future phase adding an exemption to either guarded list]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shrink-only set-membership diff against a committed baseline (same message shape as TestOrphanAllowlistOnlyShrinks), replicated rather than shared because the source diff logic is inlined in a test body with no standalone comparator function to call"
    - "Policy-doc-names-every-guarded-file invariant test, deriving its expected file list from a Go slice constant rather than re-typing paths in the test or trusting the document's own prose"

key-files:
  created:
    - cmd/testdata/flag_audit_skiplist_baseline.json
    - .aether/docs/orphan-allowlist-policy.md
  modified:
    - cmd/cli_flag_audit_test.go

key-decisions:
  - "The 172-02 comparator was REPLICATED, not called. cmd/subcommand_reachability_ratchet_test.go's shrink-only diff is inlined directly in TestOrphanAllowlistOnlyShrinks's body — there is no standalone function with a signature this plan could call for a second list. This plan may not edit that file (per its own instructions), so cmd/cli_flag_audit_test.go's TestFlagAuditSkipListOnlyShrinks re-implements the identical set-membership diff and the identical failure-message shape by hand. What IS reused directly, unchanged: the orphanAllowlistEntry JSON shape (mirrored as flagAuditSkipEntry, same three fields, same tags) and the general loadOrphanAllowlist pattern (mirrored as loadFlagAuditSkipBaseline). The two diff implementations must be kept in step by hand until a later phase extracts a shared comparator — flagged here as the concrete follow-up, not left implicit."
  - "guardedAllowlistFiles (a new []string in cmd/cli_flag_audit_test.go) is the single source of truth TestAllowlistPolicyNamesEveryGuardedFile checks against, listing all four guarded files: cmd/testdata/orphan_allowlist.json, cmd/testdata/orphan_allowlist_baseline.json, cmd/cli_flag_audit_test.go itself (the file holding the flag audit's live list), and cmd/testdata/flag_audit_skiplist_baseline.json. cmd/subcommand_reachability_ratchet_test.go has no named Go constants for its own paths (they are inline string literals) and this plan may not edit that file, so the constants live in the one file this plan owns."
  - "cmd/cli_flag_audit_test.go itself — not a JSON file — is one of the four 'guarded files' the policy names, because the flag audit's live exemption list is a Go var declaration, not a checked-in data file. Its two live-list siblings are cmd/testdata/orphan_allowlist.json (the orphan ratchet's live JSON list); the flag audit has no equivalent live JSON file by design (D-09's reasoning: scripts/classify_commands.py rewrites JSON catalogs in place, so keeping the exemption list as Go source rather than a second regenerable JSON file was already the plan's own choice)."
  - "skipSubcommands' two existing entries were tagged reason: markdown-only-shorthand, owner_phase: RECLAIM (not the reason strings the plan's action text gave verbatim, which described them individually) — both are markdown-only shorthands with no direct Go subcommand, matching RECLAIM per .planning/REQUIREMENTS.md's parked wider unreachable-command sweep, exactly as the plan's <action> text specifies."

requirements-completed: [WIRE-01, WIRE-03]

# Metrics
duration: ~35min
completed: 2026-08-11
---

# Phase 172 Plan 04: Close the Second Exemption List Summary

**Promoted the flag audit's unguarded `skipSubcommands` map to a package-level, shrink-only-guarded list (mirroring the orphan ratchet's committed-baseline diff), and wrote the plain-English policy document that names all four guarded files and is itself checked against code, not trusted prose.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-11
- **Tasks:** 2 (both landed in separate commits)
- **Files modified:** 1 modified, 2 created

## Accomplishments

- `skipSubcommands` promoted from a `TestCLIFlagAudit`-local `map[string]bool` to a package-level `[]flagAuditSkipEntry` (name/reason/owner_phase), keeping its two existing entries and explanatory comments.
- `cmd/testdata/flag_audit_skiplist_baseline.json` created — 2 entries, sorted, matching `orphan_allowlist_baseline.json`'s shape exactly.
- `TestFlagAuditSkipListOnlyShrinks` added: a pure set-membership diff, proven (below) to catch both a bare third-entry addition and a one-out-one-in swap that a count comparison would miss.
- `.aether/docs/orphan-allowlist-policy.md` written for a non-technical reader: plain English first (no file path, Go identifier, or untranslated repo term in the opening paragraph), then the real seeded numbers (278 unused-command entries, 6 tagged to the future skill cleanup effort), then a Detail section naming all four guarded files and every enforcing test.
- `TestAllowlistPolicyNamesEveryGuardedFile` added: derives its expected file list from a new `guardedAllowlistFiles` Go slice (not from re-reading the document's own prose), and is proven (below) to fail the moment the document stops naming one of them.

## Task Commits

1. **Task 1: shrink-only guard on the flag audit's skip list** — `577e8578` (feat)
2. **Task 2: the written policy and its self-checking test** — `8f80ee10` (docs)

## Files Created/Modified

- `cmd/cli_flag_audit_test.go` — `skipSubcommands` promoted to a package-level `[]flagAuditSkipEntry`; added `flagAuditSkipEntry`, `skipSubcommandNames`, `guardedAllowlistFiles`, `loadFlagAuditSkipBaseline`, `TestFlagAuditSkipListOnlyShrinks`, `TestAllowlistPolicyNamesEveryGuardedFile`.
- `cmd/testdata/flag_audit_skiplist_baseline.json` (new) — the committed baseline, 2 entries.
- `.aether/docs/orphan-allowlist-policy.md` (new) — the written policy.

## Comparator Reuse vs. Replication

Per the plan's own fallback clause: `cmd/subcommand_reachability_ratchet_test.go`'s shrink-only diff has **no standalone callable comparator** — the set-membership logic lives directly inside `TestOrphanAllowlistOnlyShrinks`'s body, not in a separate function with a reusable signature. Since this plan is barred from editing that file, `TestFlagAuditSkipListOnlyShrinks` **replicates** the identical diff logic and the identical failure-message shape (`"N command(s) were added to the tolerated ... list without being added to the committed baseline: ..."` / `"The ... list may only shrink. ... do not edit the baseline to make this pass."`) rather than reusing a shared function. What genuinely is reused, unchanged: the JSON entry shape (`name`/`reason`/`owner_phase`, mirrored as `flagAuditSkipEntry`) and the load-baseline pattern (mirrored as `loadFlagAuditSkipBaseline`, matching `loadOrphanAllowlist`'s error-message and unmarshal shape). **Follow-up for a later phase:** extract a single shared `diffAllowlistNames(live, baseline []entry) []string` helper both files call, so the two implementations can't silently drift apart from each other.

## Required Red-Then-Green Transcripts

**1. Bare third-entry addition fires the shrink-only guard.** Added `{Name: "skiplist-proof-entry", Reason: "proof", OwnerPhase: "172"}` to the live `skipSubcommands` declaration:
```
=== RUN   TestFlagAuditSkipListOnlyShrinks
    cli_flag_audit_test.go:266: 1 command(s) were added to the tolerated skip list without being added to the committed baseline: skiplist-proof-entry
        The skip list may only shrink. Make the documented call correct, or delete its entry — do not edit the baseline to make this pass.
--- FAIL: TestFlagAuditSkipListOnlyShrinks (0.00s)
FAIL
```
Reverted, confirmed clean, re-ran:
```
=== RUN   TestFlagAuditSkipListOnlyShrinks
--- PASS: TestFlagAuditSkipListOnlyShrinks (0.00s)
PASS
```

**2. One-out-one-in swap fires the guard (the case a count comparison would pass).** Removed `verify-castes`, added `{Name: "skiplist-swap-proof", ...}` — the live declaration still had exactly 2 entries, matching the baseline's count:
```
=== RUN   TestFlagAuditSkipListOnlyShrinks
    cli_flag_audit_test.go:265: 1 command(s) were added to the tolerated skip list without being added to the committed baseline: skiplist-swap-proof
        The skip list may only shrink. Make the documented call correct, or delete its entry — do not edit the baseline to make this pass.
--- FAIL: TestFlagAuditSkipListOnlyShrinks (0.00s)
FAIL
```
Reverted, re-ran:
```
=== RUN   TestFlagAuditSkipListOnlyShrinks
--- PASS: TestFlagAuditSkipListOnlyShrinks (0.00s)
PASS
```

**3. Policy doc drift fires the invariant test.** Deleted the line and its wrapped continuation naming `cmd/testdata/flag_audit_skiplist_baseline.json` from the Detail section:
```
=== RUN   TestAllowlistPolicyNamesEveryGuardedFile
    cli_flag_audit_test.go:305: .aether/docs/orphan-allowlist-policy.md does not name 1 guarded file(s): cmd/testdata/flag_audit_skiplist_baseline.json
        Add each path to the document, or the policy silently drifts out of step with what the guards actually read.
--- FAIL: TestAllowlistPolicyNamesEveryGuardedFile (0.00s)
FAIL
```
Restored, re-ran:
```
=== RUN   TestAllowlistPolicyNamesEveryGuardedFile
--- PASS: TestAllowlistPolicyNamesEveryGuardedFile (0.00s)
PASS
```

## Verification Performed

- `go build ./...` — clean.
- `go vet ./cmd` — clean.
- `gofmt -l cmd/cli_flag_audit_test.go` — clean.
- `go test ./cmd -run 'TestCLIFlagAudit|TestFlagAuditSkipListOnlyShrinks|TestWiringGuardsHaveNoRuntimeEscapeHatch' -count=1 -v` — all pass; `TestCLIFlagAudit` still skips exactly `verify-castes` and `pending-decisions`, unchanged.
- `go test ./cmd -run 'TestAllowlistPolicyNamesEveryGuardedFile|TestFlagAuditSkipListOnlyShrinks|TestOrphanAllowlistOnlyShrinks' -count=1 -v` — all pass.
- JSON shape check on `flag_audit_skiplist_baseline.json` — 2 objects, keys exactly `name`/`reason`/`owner_phase`, names `{verify-castes, pending-decisions}` — passes.
- Automated count check tying the policy doc's stated numbers to the real JSON: `policy states 278 entries, 6 owned by phase 178` — matches `cmd/testdata/orphan_allowlist.json` exactly.
- `go test ./cmd/... -count=1 -timeout 900s` — full package suite, no regression (`ok github.com/calcosmic/Aether/cmd 269.891s`).
- All three required red-then-green transcripts above, captured directly from actual `go test -v` runs.

## Known Stubs

None — this plan is entirely test infrastructure and a documentation file.

## Threat Flags

None — this plan's threat model (T-172-14, T-172-15, T-172-16) is fully covered by the work above; no new surface introduced beyond what the plan's own threat register anticipated.

## Issues Encountered

None. No deviations from the plan were required.

## User Setup Required

None.

## Next Phase Readiness

- D-12 is satisfied: both exemption lists in this repo (`cmd/testdata/orphan_allowlist.json` and `cmd/cli_flag_audit_test.go`'s `skipSubcommands`) now carry the identical shrink-only discipline against a committed baseline, proven by red-then-green transcripts for a bare addition and a one-out-one-in swap on each.
- D-11's "no escape hatch" holds across both lists: `TestWiringGuardsHaveNoRuntimeEscapeHatch` (which already covers `cmd/cli_flag_audit_test.go` in its `guardFiles` list) continues to pass unchanged, confirming this plan's additions introduced no `os.Getenv`, `t.Skip`, or build constraint.
- The written policy states the real seeded numbers (278 / 6) rather than ROADMAP.md's originally expected 8, and is asserted — not merely hoped — to stay in step with the four files it names.
- Follow-up flagged for a later phase (not blocking): unify `TestOrphanAllowlistOnlyShrinks`'s inline diff and `TestFlagAuditSkipListOnlyShrinks`'s replicated diff into one shared comparator function, since both currently implement the identical logic independently.

---
*Phase: 172-wiring-proof*
*Completed: 2026-08-11*

## Self-Check: PASSED

- FOUND: `cmd/testdata/flag_audit_skiplist_baseline.json`
- FOUND: `.aether/docs/orphan-allowlist-policy.md`
- FOUND: `cmd/cli_flag_audit_test.go` modified
- FOUND commit: `577e8578` (Task 1)
- FOUND commit: `8f80ee10` (Task 2)
