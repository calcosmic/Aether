# Security Audit — Phase 163: Context Reaches Workers

**Audited:** 2026-07-30
**Auditor:** gsd-security-auditor
**ASVS Level:** 1
**block_on:** high
**Threats Closed:** 18/18 (all declared mitigations verified present in code)
**Residual findings requiring follow-up:** 3 (WARNING — unregistered/degraded, not blocking)

This audit verifies each threat declared in the six plan files' `<threat_model>`
blocks (163-01 through 163-06) against the actual implementation. Evidence is a
grep/read of the cited file at the cited location, not documentation or intent.
No new threat scanning was performed — this is a disposition-by-disposition
verification of the register the plans themselves declared, per
`gsd-secure-phase`'s "does not scan blindly" mandate. `.planning/phases/163-context-reaches-workers/163-REVIEW.md`
(the phase's independent code review) was cross-checked against every
`mitigate` threat whose surface it also reviewed; where the review found a
defect touching a declared mitigation, that finding is recorded below rather
than silently dropped.

## Threat Verification

| Threat ID | Category | Component | Disposition | Status | Evidence |
|-----------|----------|-----------|-------------|--------|----------|
| T-163-01 | Tampering | `protectedHookWriteReason` `.aether/data/` carve-out | mitigate | CLOSED | `cmd/hook_cmds.go:232-238` — `sanctionedDataWritePrefixes` is exactly 4 full-segment entries (`/.aether/data/planning/`, `/.aether/data/phase-research/`, `/.aether/data/survey/`, `/.aether/data/worker-debug/`), checked at `hook_cmds.go:247-251` before the blanket `/.aether/data/` block. `TestHookPreToolUseBlocksProtectedPath` (`hook_cmds_test.go:34`) unmodified. Substring-not-segment negative subtest present (`behavior_7_substring_not_segment_still_blocked`). Positive allow-tests present (`TestHookPreToolUseAllowsSanctionedScratchDirs`). |
| T-163-02 | Tampering / Info Disclosure | `artifacts` schema | mitigate | CLOSED | `pkg/codex/worker.go:930-937` — `additionalProperties: false` retained; `properties` carries 3 named typed fields (`research_file`, `survey_file`, `plan_file`). `TestWorkerArtifactsSchemaAcceptsNamedFields` (`pkg/codex/worker_test.go:138`) asserts `additionalProperties=false` and rejects an `unnamed_field`. |
| T-163-03 | Tampering / Spoofing | Charter entering worker prompts | mitigate | CLOSED | `cmd/colony_prime_context.go:460-481` appends the charter section into the same `sections` slice that `AssessPromptSource` (line 905) and `RankContextCandidates` (line 918) process uniformly for every section. `grep -rn 'Charter' cmd/codex_build.go` returns **no matches** — no bypass concatenation exists. |
| T-163-04 | Tampering / Spoofing | `codexBuildManifest.ContextCapsule` | mitigate | CLOSED | `cmd/codex_build.go:97` (field), `:1629` (single new manifest-level `resolveCodexWorkerContext()` call, guarded by `planOnly`). `grep -c resolveCodexWorkerContext cmd/codex_build.go` = 2 (Path A at `:1413` + the manifest site) — no third, per-dispatch occurrence. `TestBuildManifestCarriesContextCapsuleOnce` verified with an observed-and-reverted regression proof in 163-01-SUMMARY.md. |
| T-163-05 | Information Disclosure | Manifest JSON written outside `.aether/data/` by wrapper | accept | CLOSED (logged below) | Pre-existing behavior unchanged by this phase; logged in Accepted Risks. |
| T-163-06 | Tampering | Wrapper reconstructing/summarizing capsule | mitigate | CLOSED | `.claude/commands/ant/build.md:84,97` and `.opencode/commands/ant/build.md:84,97` both state "VERBATIM"/"verbatim" and "Do not summarize, reorder, or reconstruct." `TestBuildWrapperCeremonyContract` (`cmd/build_wrapper_ceremony_test.go:43`) requires `dispatch_manifest.context_capsule` in both files; regression proof observed and quoted in 163-01-SUMMARY.md. |
| T-163-07 | DoS (self-inflicted) | Charter compliance gate false-positive blocking | mitigate | CLOSED | `cmd/gate.go:635-712` — `checkCharterComplianceGate` requires all three conditions (declared + config file exists + no step references the token) before raising a violation (`:698-703`); charter-drift-only (config file absent) explicitly continues without violation (`:692-697`); empty/fallback charter returns pass (`:667-673`). |
| T-163-08 | Repudiation | Gate silently not running read as pass | mitigate | CLOSED | Two distinct `gateCheck` values (`charter_compliance`, `charter_compliance_executed`) returned from every branch of `checkCharterComplianceGate`; `charter_compliance_executed` registered in `alwaysRunGates` (`cmd/gate.go:924`) so a skip can never suppress the non-execution signal. Live call site: `cmd/codex_continue.go:2966`. |
| T-163-09 | Elevation of Privilege | Scout write-access broadening | mitigate | CLOSED* | `pkg/codex/permission_profile.go:53-55` — `repositoryReadOnlyCastes` now contains only `"includer"`; scout removed. `:93-94` — `behavioralRestrictionsForCaste("scout")` returns the phase-research-only restriction string. All three literal elements of the plan's mitigation text are present. **See Residual Finding R-1 below** — the restriction is prose-only on every platform, including the Claude Code hook, which does not scope writes to `.aether/data/phase-research` (it only blocks a curated list of protected paths). This matches the plan's own stated design (identical to the surveyor precedent, "no hard read-only lock") but is weaker than the disposition text's wording implies. |
| T-163-10 | Tampering / EoP | `surveyStalenessNotice` → git subprocess | mitigate | CLOSED | `cmd/survey_staleness.go:47-51` — `time.Parse(time.RFC3339, ...)` runs and returns `""` on failure *before* `commitsSinceSurvey` is ever called (no git invocation on malformed input). `:76-80` — `exec.CommandContext(ctx, "git", "-C", root, "rev-list", "--count", sinceArg, "HEAD")`, discrete argv, no `sh -c`. |
| T-163-11 | Information Disclosure | Survey pointer list naming files | accept | CLOSED (logged below) | `cmd/helpers.go:222-260` — `resolveSurveySection` confines the pointer list to `store.BasePath()/survey`, repo-relative. Logged in Accepted Risks. |
| T-163-12 | Repudiation | Requirement text reworded, scope quietly reduced | mitigate | CLOSED | `.planning/REQUIREMENTS.md:70,73,77,78` each carry `2026-07-29` and cite `Phase 163 D-07`/`D-08`. New Corrections-carried-forward row at line 27 cites commits `281dd34a`/`a2c8288e`. Requirement count/IDs unchanged (verified by the plan's own grep criterion, recorded green in 163-04-SUMMARY.md). |
| T-163-13 | Tampering / Spoofing | Repo-derived suggestion text becoming a steering pheromone | mitigate | CLOSED | `cmd/suggest_analyze.go:143` — `colony.SanitizeSignalContent` appears exactly once, inside `runSuggestAnalyze`, applied to every candidate before persistence. Confirmed a `PendingSuggestion` only becomes an active `PheromoneSignal` via an explicit `aether suggest-approve --approve <id>` call (`cmd/suggest_approve.go:94-170`) — no automatic activation path exists. |
| T-163-14 | DoS | Unbounded `PendingSuggestions` growth | mitigate | CLOSED | `cmd/suggest_analyze.go:19` (`changeThreshold = 5`), `:123` (`loadActivePheromoneHashes` dedup against active signal hashes), merge-with-existing preserved per 163-05-SUMMARY.md. |
| T-163-15 | Availability | suggest-analyze failure blocking a successful build | mitigate | CLOSED | `cmd/codex_build_finalize.go:461-470` — `collectPendingSuggestions` swallows the error into a `stderr` warning and returns `(false, 0)`, never propagated to the finalize caller. `TestBuildFinalizeCollectsSuggestAnalyzeResults` asserts finalize succeeds with a zero count on failure (observed-regression-proof quoted in 163-05-SUMMARY.md). |
| T-163-16 | Information Disclosure | `--full` printing whole assembled prompt | mitigate | CLOSED | `cmd/build_print_brief.go:87-95` — raw `brief` + composition table only printed when `options.Full` is true; default branch calls `renderBriefChecklist` (names/sizes only). |
| T-163-17 | Tampering | Inspection command mutating state | mitigate | CLOSED | `grep -n 'Brief =' cmd/build_print_brief.go` returns no matches — no assignment into any dispatch's `Brief` field. `printWorkerBriefs`'s own doc comment (`:11-25`) states it calls only pure readers; `attachBuildDispatchContext`/`resolveCodexWorkerContext` are read-only calls. |
| T-163-18 | Repudiation | Budget figure drifting from enforced budgets | mitigate | CLOSED* | `cmd/build_print_brief.go:256-261` — `assembledContextBudgetCeilingChars()` sums 5 named constants, no literal total. **See Residual Finding R-2 below** — the summed `colonyPrimeBudgetChars` (8000, non-compact) does not match the budget actually enforced by `resolveCodexWorkerContext()`, which calls `buildColonyPrimeOutput(true)` → `colonyPrimeCompactBudgetChars` (4000). The guard is real and grep-clean but ~4000 chars looser than the real delivery path. |

`*` = mitigation as declared in the plan is present and verified; a residual
implementation defect independent of the plan's literal wording was found by
cross-referencing `163-REVIEW.md` and is recorded as a Residual Finding, not a
BLOCKER, per the instruction to weigh review evidence honestly without
patching implementation.

## Accepted Risks Log

### T-163-05 — Manifest JSON written outside `.aether/data/` by the wrapper
**Disposition:** accept
**Rationale (from plan 163-01):** Pre-existing wrapper behavior for the whole
manifest — dispatch briefs already carry the same class of colony-state
content. The context capsule adds no new secret class beyond what the brief
already exposed, and the temp file is local to the user's own machine, not
transmitted anywhere.
**Verified pre-existing:** Confirmed the manifest-to-tempfile write path is
unchanged by this phase's commits (only `ContextCapsule` was added as a field
alongside the pre-existing `Brief`/`WorkerBriefs` fields).
**Accepted by:** Phase 163 plan authors, 2026-07-29.

### T-163-11 — Survey pointer list naming files a worker then reads
**Disposition:** accept
**Rationale (from plan 163-04):** Pre-existing behavior of `resolveSurveySection`.
The pointer list is repo-relative and confined to `.aether/data/survey/`,
which the colony's own `/ant-colonize` step generated — no path outside that
directory can appear in the list.
**Verified:** `cmd/helpers.go:225` restricts `os.ReadDir` to
`store.BasePath()/survey`; filenames are filtered to `.md`/`.json` only.
**Accepted by:** Phase 163 plan authors, 2026-07-29.

## Residual Findings (WARNING — not BLOCKER, follow-up recommended)

These were surfaced by cross-referencing `163-REVIEW.md` against the declared
threat register. None of the six plan SUMMARY.md files (163-03 through 163-06
in particular) carry a populated `## Threat Flags` section reporting them —
163-03's SUMMARY records an *adjacent* but different OpenCode finding
("scout's real *enforced* OpenCode agent-level permissions still block writes,
unlike Claude Code and Codex") as a "Known Follow-up," not as a Threat Flag,
and none of the three findings below appear anywhere in a Threat Flags
section. They are logged here as `unregistered_flag` entries.

### R-1 (relates to T-163-01 / T-163-09): Scout write confinement is prose-only on every platform; hook path resolution is lexical only
**Source:** 163-REVIEW.md WR-05
**Finding:** `PermissionProfileForCaste("scout")` now returns
`PermissionWorkspaceWrite` — a workspace-wide write grant — with only a
*behavioral restriction string* ("write phase research artifacts under
`.aether/data/phase-research` only") limiting scope. No mechanism on any
platform enforces that narrower scope:
- Claude Code's `PreToolUse` hook (`protectedHookWriteReason`) only blocks a
  curated *denylist* (`.aether/data/` minus 4 sanctioned subpaths,
  `.aether/dreams/`, `.env*`, `.codex/config.toml`) — it does not allow-list
  `.aether/data/phase-research` as scout's *only* writable location, so a
  scout can still write arbitrary project source files anywhere outside
  those denylisted paths.
- OpenCode's own doc (`.opencode/OPENCODE.md`) explicitly concedes its
  protection is "conduct, not a sandbox."
- Codex-native has no equivalent hook at all.
- Separately, `normalizeHookPath` (`cmd/hook_cmds.go:316-334`) uses only
  `filepath.Clean`/`filepath.Abs` — no `filepath.EvalSymlinks` — so a symlink
  planted inside a sanctioned scratch directory (e.g.
  `.aether/data/planning/link -> ../COLONY_STATE.json`) would resolve
  lexically to an "allowed" path while landing on protected state.

**Verified in this audit:** confirmed both claims by reading
`normalizeHookPath` (no symlink resolution) and `protectedHookWriteReason`
(denylist shape, not an allowlist scoped to phase-research).
**Assessment:** The T-163-09 mitigation exactly as the plan described it is
implemented and matches its own cited precedent (the surveyor caste has the
same "no hard read-only lock" shape). This is a pre-existing design pattern
this phase extended to a second caste, not a novel regression — but the
disposition text's implication that scout is confined to one directory
overstates what any platform actually enforces. Recommend: (a) add
`filepath.EvalSymlinks` to `normalizeHookPath` before matching, (b) record the
cross-platform enforcement asymmetry explicitly in
`.aether/references/contracts/protected-local-state-contract.md` as a named
residual risk, matching WR-05's suggested fix.

### R-2 (relates to T-163-18): Budget ceiling sums the non-compact constant while the delivered capsule is compact
**Source:** 163-REVIEW.md WR-03
**Finding:** `assembledContextBudgetCeilingChars()` (`cmd/build_print_brief.go:256-261`)
sums `colonyPrimeBudgetChars` (8000). Every capsule this phase actually
delivers — the plan-only manifest, the hosted path, and the checklist's own
display — is built via `buildColonyPrimeOutput(true)`
(`cmd/colony_prime_context.go:962-963`), which uses
`colonyPrimeCompactBudgetChars` (4000).
**Verified in this audit:** confirmed both call sites directly; `compact=true`
is hardcoded at `resolveCodexWorkerContext()`'s only call to
`buildColonyPrimeOutput`.
**Assessment:** The mitigation's mechanical requirement (derive from named
constants, no bare literal) is satisfied and grep-verifiable. But the specific
constant chosen is the wrong one for the runtime path being measured, so the
guard cannot trip until the real capsule is ~2x its actual enforced budget —
undermining the disposition's own stated purpose ("the reported number cannot
silently diverge from enforcement"). Recommend: sum
`colonyPrimeCompactBudgetChars` instead (or parameterize the ceiling on the
same `compact` flag `resolveCodexWorkerContext` uses).

### R-3 (relates to T-163-14 / T-163-15, informational): Non-atomic persistence now runs on every build's critical path
**Source:** 163-REVIEW.md WR-01
**Finding:** `runSuggestAnalyze`'s persistence (`cmd/suggest_analyze.go:194-201`)
marshals the `cs` snapshot loaded at function entry and does a whole-file
`store.AtomicWrite`, with a multi-second window (git + repo tree walks)
between load and write during which another writer's changes would be
clobbered. This pattern pre-dates this phase, but this phase gave the
function its first live, per-build caller (`collectPendingSuggestions` in
`cmd/codex_build_finalize.go:461`), moving the exposure window from
manual-invocation-only to every completed build.
**Assessment:** Not a violation of any of the three declared threats for this
plan (T-163-13/14/15 concern sanitization, growth-bounding, and
non-blocking-on-failure specifically, all of which are independently verified
CLOSED above) — this is a correctness/race-condition concern the plan did not
scope in as a threat, surfaced honestly for follow-up rather than silently
dropped. Recommend routing the merge through `store.UpdateJSONAtomically`
(the primitive `cmd/gate.go:1225` already uses for the same class of problem).

### Operational note (not a code defect)
The local `.aether/data/COLONY_STATE.json` on this machine currently still
carries ~136 junk `pending_suggestions` written by the now-fixed
`TestSuggestAnalyze_NonBlockingOnError` sandbox escape (163-REVIEW.md CR-01,
resolved in commit `ac88530b` — confirmed present, test now uses
`newTestStore(t)` for isolation). This is local, gitignored, non-shipping
data — not modified by this audit per the implementation-files-are-read-only
constraint — but will surface via the phase's new "Suggestions From This
Build" closeout block (IN-04) until cleaned with
`aether suggest-approve --dismiss-all`.

## Unregistered Flags

None beyond R-1/R-2/R-3 above (all three are recorded there rather than
re-listed here, since each maps to — and is scoped against — a specific
declared threat ID rather than constituting genuinely new, unmapped attack
surface).

## Verification Commands Run

```
go build ./cmd/aether                                          # exit 0
grep -rn 'Charter' cmd/codex_build.go                           # no matches
grep -c resolveCodexWorkerContext cmd/codex_build.go             # 2
grep -n 'Brief =' cmd/build_print_brief.go                       # no matches
```

All grep/read evidence above was captured directly against the files cited in
each plan's threat model at audit time (2026-07-30), not copied from
SUMMARY.md claims.
