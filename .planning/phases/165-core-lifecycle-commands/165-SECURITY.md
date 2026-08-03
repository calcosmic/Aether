# Phase 165 — Core Lifecycle Commands — Security Audit

**Audit date:** 2026-08-03
**Auditor:** gsd-security-auditor
**Scope:** All 10 plans (165-01 through 165-10) of phase `165-core-lifecycle-commands`
**Threat register source:** `<threat_model>` blocks in `165-01-PLAN.md` through `165-10-PLAN.md` (union, 60 threats: T-165-01-01 .. T-165-10-06)
**ASVS Level:** default (not specified in phase config)
**block_on:** not configured

**Result: 60/60 CLOSED, 0/60 OPEN** *(59 verified by audit; T-165-03-05 closed by post-audit fix — see Resolved Finding below)*

Verification method: every `mitigate` threat was checked by grepping the cited
files for the literal evidence claimed in the plan's Mitigation Plan column, or
by running the named Go test and confirming PASS. `go build ./cmd/aether`,
`go vet ./...`, and `go test ./cmd/... ./pkg/...` were all re-run against the
current working tree and are green (full run: `cmd` 281.254s, all `pkg/*`
green). Every `accept` threat's rationale was independently checked against the
current code, not merely re-read from the plan. No implementation file was
modified by this audit.

---

## Threat Verification

### 165-01 — Wave 1 shared foundation

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-01-01 | Tampering | mitigate | CLOSED | `.aether/docs/wrapper-host-contract.md` has 9 `## ` sections (`grep -c '^## '` → 9); all 7 pre-existing sections (Context, Decision, Boundary Rules, Provider/Auth Boundary, Rationale, Migration Notes, Consequences) survive alongside the 2 new ones |
| T-165-01-02 | Spoofing | mitigate | CLOSED | `TestLifecycleFlatMirrorsMatchCanonical` PASS for all 4 verbs; `diff` confirms byte-identity of all 4 flat mirrors against canonical sources |
| T-165-01-03 | Repudiation | mitigate | CLOSED | `TestWrapperHostContractDocumentsManifestShapes` PASS; `.aether/docs/wrapper-host-contract.md:42` (`## Manifest and Completion Packet Shapes`) and `:67` (`## Terminal Worker Result Belongs to the Wrapper`) both present |
| T-165-01-04 | Elevation of Privilege | mitigate | CLOSED | Contract table obligation column states "preserve, never broaden `repository_read_only`"; wrapper-side instruction retained at `.claude/commands/ant/build.md:168` |
| T-165-01-05 | Information Disclosure | accept | CLOSED | Accepted-risk rationale re-verified: `.aether/docs/wrapper-host-contract.md:83` `## Provider/Auth Boundary` section is untouched and still states wrappers "must not paste raw provider output, tokens, or auth probe details" |

### 165-02 — build.md rewrite

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-02-01 | Tampering | mitigate | CLOSED | No verbatim user-transcript worked example in `.claude/commands/ant/build.md` (`grep -c 'Context Confirmation Rule\|user said:'` → 0) |
| T-165-02-02 | Elevation of Privilege | mitigate | CLOSED | `.claude/commands/ant/build.md:168` retains the `permission_profile` preserve/reject instruction inline; `TestLifecycleCommandDocsPreferRuntimeCLI` PASS |
| T-165-02-03 | Tampering | mitigate | CLOSED | `TestBuildWrapperCeremonyContract` PASS (forbidden Read-loader phrasing absent) |
| T-165-02-04 | Tampering | mitigate | CLOSED | `grep -c 'Write COLONY_STATE.json'` → 0 in `build.md` |
| T-165-02-05 | Information Disclosure | mitigate | CLOSED | `.claude/commands/ant/build.md:46,117,258` all carry the sanctioned sanitized-availability sentence and the "Do NOT expose raw provider stdout/stderr, tokens, or auth probe output" guardrail |
| T-165-02-06 | Spoofing | mitigate | CLOSED | `grep -cE '━━━|────' .claude/commands/ant/build.md` → 0 |
| T-165-02-07 | Repudiation | mitigate | CLOSED | `TestBuildMdOwnershipHandshake` PASS; `.claude/commands/ant/build.md:7` `PHASE-160:` comment precedes `:262` `<!-- PHASE-168: ... -->` (last non-empty line) |
| T-165-02-08 | Tampering | mitigate | CLOSED | `grep -cE 'git stash|git add -A|git commit'` → 0 across `.claude`/`.opencode` build.md and continue.md; forbidden-string entries present in `cmd/build_wrapper_ceremony_test.go` and `cmd/continue_wrapper_ceremony_test.go` |

### 165-03 — continue.md rewrite

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-03-01 | Tampering | mitigate | CLOSED | `grep -c "safe to clear your context"` and `grep -c 'ant-resume'` both → 0 across `.claude`, `.opencode`, and flat-mirror continue.md; `TestContinueWrapperStageSkeletonAndParity/context_clear_stays_runtime_owned` PASS |
| T-165-03-02 | Spoofing | mitigate | CLOSED | `.claude/commands/ant/continue.md:187` ("never fabricate a verification result") and `:204` ("Do NOT replay verification loops or reimplement runtime gate logic") both present |
| T-165-03-03 | Tampering | mitigate | CLOSED | `grep -cE 'continue-verify.md|continue-gates.md'` → 0 in continue.md |
| T-165-03-04 | Tampering | mitigate | CLOSED | Same forbidden strings as above; `TestContinueWrapperCeremonyContract` PASS |
| T-165-03-05 | Information Disclosure | mitigate | CLOSED | Guardrail added post-audit (2026-08-03): `.claude/commands/ant/continue.md` Guardrails now carries "Do NOT expose raw provider stdout/stderr, tokens, or auth probe output; use the Go availability category and sanitized next action" — matching build.md's line — propagated byte-identically to `.opencode/commands/ant/continue.md` and `.claude/commands/ant-continue.md`; `TestLifecycleFlatMirrorsMatchCanonical` and hygiene tests PASS. See Resolved Finding below. |
| T-165-03-06 | Elevation of Privilege | mitigate | CLOSED | `.claude/commands/ant/continue.md:191` `<read_only>` block names the never-hand-written files; `:205` "Do NOT read or write colony state files by hand" guardrail preserved |
| T-165-03-07 | Tampering | mitigate | CLOSED | `grep -cE 'git stash|git add -A|git commit'` → 0 across `.claude`, `.opencode`, and flat-mirror continue.md |

### 165-04 — plan.md rewrite

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-04-01 | Spoofing | mitigate | CLOSED | `.claude/commands/ant/plan.md:211` "Do NOT compose depth recommendations, reasons, or research recommendations in this wrapper; print the runtime-emitted card verbatim" present; `TestPlanWrapperCardsParity/both_wrappers_reference_all_four_runtime_keys` PASS |
| T-165-04-02 | Tampering | mitigate | CLOSED | `grep -c 'Update watch files for tmux visibility'` → 0 in plan.md (confirmed via `TestLifecycleCommandDocsPreferRuntimeCLI` PASS) |
| T-165-04-03 | Tampering | mitigate | CLOSED | `grep -c 'Write COLONY_STATE.json'` → 0 in plan.md |
| T-165-04-04 | Elevation of Privilege | mitigate | CLOSED | `.claude/commands/ant/plan.md:108` "Scout's `permission_profile` must be passed through verbatim from the manifest, never substituted or broadened" |
| T-165-04-05 | Repudiation | mitigate | CLOSED | `.claude/commands/ant/plan.md:151` names `--accept` explicitly; confirmed live via `aether plan --help` → `--accept  Accept the current best plan even if confidence is below target` |
| T-165-04-06 | Tampering | mitigate | CLOSED | `TestPlanWrapperStageSkeleton/termination_conditions_documented` PASS (concept markers only, no numeric thresholds in prose) |

### 165-05 — init.md ceremony restoration

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-05-01 | Tampering | mitigate | CLOSED | `grep -cE 'Write COLONY_STATE.json|queen-init|aether-utils.sh|aether state-write'` → 0 in init.md; `<read_only>` block present |
| T-165-05-02 | Tampering | mitigate | CLOSED | `.claude/commands/ant/init.md:287` passes charter as `--charter-json '<synthesized charter JSON>'` CLI argument, not executed instruction |
| T-165-05-03 | Spoofing | mitigate | CLOSED | `.claude/commands/ant/init.md:21-23` Approval stop conditions: "A cancel or a failed `aether init` both end the command with nothing persisted" |
| T-165-05-04 | Elevation of Privilege | mitigate | CLOSED | `grep -c 'aether pheromone-write' .claude/commands/ant/init.md` → 2 |
| T-165-05-05 | Information Disclosure | mitigate | CLOSED | `.claude/commands/ant/init.md:84` (README summary first 200 chars) and `:112` (up to 3 prior-colony entries, goal truncated to ~120 chars) both present |
| T-165-05-06 | Repudiation | mitigate | CLOSED | `TestInitWrapperStageSkeletonAndParity/platform_difference_is_the_sanctioned_one` PASS |

### 165-06 — cross-wrapper invariants, CMD-04 fence, CMD-03 sign-off

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-06-01 | Repudiation | mitigate | CLOSED | `165-VALIDATION.md` frontmatter: `status: complete`, `nyquist_compliant: true`, `wave_0_complete: true`; full `go test`/`go vet`/`go build` re-run green independently by this audit |
| T-165-06-02 | Tampering | mitigate | CLOSED | `TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob/a_wrapper_dominated_by_envelope_prose_would_fail` PASS (negative control present and green) |
| T-165-06-03 | Tampering | mitigate | CLOSED | `TestSpecialistCommandSurfacesUnchanged` PASS across all 17 surfaces; `len(specialistCommandSurfaces) != 17` explicit count guard at `cmd/lifecycle_wrapper_contract_test.go:538` |
| T-165-06-04 | Spoofing | mitigate | CLOSED | `165-VALIDATION.md:68` records a dated (2026-08-03), stage-by-stage CMD-03 verdict: "**Approved** — all nine stages ... were describable in one sentence each" |
| T-165-06-05 | Information Disclosure | accept | CLOSED | Rationale re-verified: `cmd/lifecycle_wrapper_contract_test.go` comment above `specialistCommandSurfaceHashes` explicitly states "This is a change-detection fence, not an integrity control against an adversary" |

### 165-07 — shelf-to-todo runtime wiring, init.md gap closure round 1

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-07-01 | Tampering | mitigate | CLOSED | `grep -cE 'active_todos|Write COLONY_STATE.json'` → 0 in init.md; forbidden entries present in `cmd/init_wrapper_ceremony_test.go`, proven RED-before-fix per SUMMARY |
| T-165-07-02 | Tampering | mitigate | CLOSED | `promotedShelfTodos` filters `Status == promoted && PromotedTo == trimmed(colonyGoal)` (`cmd/shelf_init.go`) |
| T-165-07-03 | Repudiation | mitigate | CLOSED | `TestInitSeedsSessionTodosFromPromotedShelf` and `TestSessionRefreshPreservesShelfTodos` both PASS |
| T-165-07-04 | Elevation of Privilege | mitigate | CLOSED | `TestInitWrapperCeremonyContract/approval_writes_pheromones_only_after_init_succeeds` PASS |
| T-165-07-05 | Denial of Service | mitigate | CLOSED | `promotedShelfTodos` returns empty non-nil `[]string{}` on any read error or nil store (`cmd/shelf_init.go`) |
| T-165-07-06 | Repudiation | mitigate | CLOSED | `TestLifecycleFlatMirrorsMatchCanonical` PASS; `.aether/commands/init.yaml` updated in lockstep |

### 165-08 — read/write-boundary contradiction, clarification-gate ordering

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-08-01 | Tampering | mitigate | CLOSED | `TestLifecycleWrapperReadOnlyBlocksAreConsistent` PASS (all 3 subtests, 12 surfaces); `grep -rn 'may read but never write' .claude/ .opencode/ .aether/` → no matches |
| T-165-08-02 | Repudiation | mitigate | CLOSED | `.claude/commands/ant/plan.md` line order confirmed: `## Planning Manifest` (33) → `## Clarification Gate` (50) → `## Decision Moment 2` (63) |
| T-165-08-03 | Tampering | mitigate | CLOSED | All 3 surfaces per verb (build, continue, plan) byte-identical via `diff` |
| T-165-08-04 | Denial of Service | mitigate | CLOSED | `plan.md` contains exactly 1 occurrence each of the third-decision-moment guardrail, `--approve-all`, and `--auto` |
| T-165-08-05 | Tampering | mitigate | CLOSED | `TestSpecialistCommandSurfacesUnchanged` and `TestBuildMdOwnershipHandshake` both PASS after this plan's edits |

### 165-09 — atomic shelf promotion, init-ceremony parity, recovery docs

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-09-01 | Tampering | mitigate | CLOSED | `cmd/init_cmd.go:203` `store.SaveJSON("COLONY_STATE.json", ...)` precedes `:214` `applyInitShelfSelections(store, ...)`; `TestFailedInitLeavesShelfEntriesShelved` PASS |
| T-165-09-02 | Repudiation | mitigate | CLOSED | `TestInitPromotesUnderRevisedGoal` PASS; promotion and colony goal share one `goal` variable |
| T-165-09-03 | Denial of Service | mitigate | CLOSED | `applyInitShelfSelections` (`cmd/shelf_init.go:144`) has no error return; bad IDs collected into `failed` |
| T-165-09-04 | Information Disclosure | accept | CLOSED | Rationale re-verified: shelf text flows through `session.ActiveTodos` (`cmd/recovery_snapshot.go:134,421,584`) as a rendered data line, never as an executed instruction; user explicitly promoted it in this init run |
| T-165-09-05 | Repudiation | mitigate | CLOSED | `TestShelfPromoteBatchFailsWhenAllIDsFail`/`TestShelfDismissBatchFailsWhenAllIDsFail` PASS; `outputError` calls present at `cmd/shelf_init.go:57,103` |
| T-165-09-06 | Tampering | mitigate | CLOSED | `TestInitCeremonySeedsSessionTodosFromPromotedShelf` PASS; `grep -c 'ActiveTodos: []string{}' cmd/init_ceremony.go` → 0 |

### 165-10 — wrapper half of shelf-promotion ordering fix

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-165-10-01 | Tampering | mitigate | CLOSED | `grep -rn 'aether shelf-promote-batch\|aether shelf-dismiss-batch' .claude/commands/ant/init.md .opencode/commands/ant/init.md .claude/commands/ant-init.md` → no matches; `TestInitWrapperCeremonyContract/shelf_ids_are_spent_only_inside_the_approval_init_call` PASS |
| T-165-10-02 | Repudiation | mitigate | CLOSED | `.claude/commands/ant/init.md:21-23` failure-mode claim now literally true; ordering subtest PASS |
| T-165-10-03 | Tampering | accept | CLOSED | Rationale re-verified: `applyInitShelfSelections` never fails colony creation on a bad ID; `.claude/commands/ant/init.md:289` requires reporting `shelf_failed` back to the user |
| T-165-10-04 | Repudiation | mitigate | CLOSED | Ordering subtest iterates `canonicalWrapperPaths` plus `flatMirrorPath(repoRoot, "init")`; all 3 surfaces PASS |
| T-165-10-05 | Repudiation | mitigate | CLOSED | `.aether/commands/init.yaml:48` and `cmd/command_guide.go:163` both updated in lockstep |
| T-165-10-06 | Repudiation | mitigate | CLOSED | `grep -c 'currently RED for build and init' cmd/lifecycle_wrapper_contract_test.go` → 0; doc comment rewritten to describe the standing invariant only |

---

## Resolved Finding

### T-165-03-05 — Information Disclosure — continue.md (RESOLVED 2026-08-03)

**Declared mitigation:** "The sanctioned sanitized-availability idiom is
retained; the existing guardrail forbidding raw provider output is preserved
verbatim."

**What was checked:**
- `.claude/commands/ant/continue.md` — zero occurrences of `provider`,
  `stdout`, `stderr`, `sanitized`, or `auth probe` (verified by full read and
  by `grep`)
- `.opencode/commands/ant/continue.md`, `.claude/commands/ant-continue.md` —
  same, zero occurrences (all three surfaces are byte-identical per Task 2's
  own acceptance criteria, so this was expected)
- `.aether/commands/continue.yaml` — same, zero occurrences
- Full git history of `.claude/commands/ant/continue.md` from its creation
  commit (`f891a20c feat(05-07): create /ant:continue command`) forward — no
  commit ever introduced a raw-provider-output guardrail to this file

**Why this is a blocker, not a nitpick:** the plan's mitigation claims a
guardrail is "preserved verbatim," which asserts it existed before and exists
now. Neither is true — the guardrail was never present in this file at any
point in its history. This is exactly the class of paper claim this project's
own Definition of Done was written to catch (`CLAUDE.md`: "A documentation
claim about runtime behaviour must be testable or removed"). Whether or not
the underlying risk is realistic on continue's current heavy-review path
(reviewers are spawned as platform subagents via the Task tool, not as
external CLI provider processes with their own stdout/stderr the way build.md
dispatches workers) is a scope question for the phase owner, not something
this audit resolves by assumption. Two honest outcomes exist: either the
guardrail sentence build.md already carries (`.claude/commands/ant/build.md:258`
— "Do NOT expose raw provider stdout/stderr, tokens, or auth probe output")
needs a matching line in continue.md's Guardrails section, or the threat
should be re-disposed to `accept` with a stated reason (heavy-review reviewers
never touch a provider CLI process directly) and logged in this file's
Accepted Risks Log below. Neither has happened yet.

**Resolution (2026-08-03, post-audit, user-approved):** The guardrail sentence
build.md carries was added to continue.md's Guardrails section — "Do NOT expose
raw provider stdout/stderr, tokens, or auth probe output; use the Go
availability category and sanitized next action" — and propagated to all three
surfaces (`.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`,
`.claude/commands/ant-continue.md`), which remain byte-identical.
`TestLifecycleFlatMirrorsMatchCanonical`, `TestBuildWrapperCeremonyContract`,
`TestSpecialistCommandSurfacesUnchanged`, `TestBuildMdOwnershipHandshake`, and
the platform doc hygiene tests all PASS after the change. The plan's claim is
now true in the working tree.

---

## Accepted Risks Log

The following four threats carry an `accept` disposition in their originating
PLAN.md. Each rationale below was independently re-checked against the
current working tree (not merely copied from the plan) as part of this audit,
and each still holds.

| Threat ID | Plan | Rationale | Re-verified against |
|-----------|------|-----------|----------------------|
| T-165-01-05 | 165-01 | No provider output is introduced by the new contract-doc sections; the doc's existing Provider/Auth Boundary section already forbids raw provider output and is untouched by this phase. | `.aether/docs/wrapper-host-contract.md:83-91` — section content unchanged, still forbids raw provider/token/auth-probe output |
| T-165-06-05 | 165-06 | The SHA-256 specialist-surface ledger is a change-detection fence, not an integrity control against an adversary; it does not create a false sense of tamper-proofing because the comment says so explicitly. | `cmd/lifecycle_wrapper_contract_test.go` comment directly above `specialistCommandSurfaceHashes`: "This is a change-detection fence, not an integrity control against an adversary" |
| T-165-09-04 | 165-09 | Prior-colony free text surfacing in CONTEXT.md/HANDOFF.md is acceptable because the user explicitly promoted these entries in this init run, and the text is rendered as a data line, never executed as instruction; it already rendered in session.json and `aether resume` before this phase. | `cmd/recovery_snapshot.go:134,421,584` — `session.ActiveTodos` values are string-interpolated into rendered markdown, never passed to an execution path |
| T-165-10-03 | 165-10 | LLM-interpolated shelf ID lists reaching the runtime are acceptable because IDs are opaque shelf keys; `applyInitShelfSelections` collects unknown IDs into `shelf_failed` and never fails colony creation, and the Approval bullet requires reporting failed IDs back to the user. | `cmd/init_cmd.go:214-217` (stderr warning on `shelfFailed`), `cmd/init_cmd.go:275` (`shelf_failed` in JSON result), `.claude/commands/ant/init.md:289` (user-facing report requirement) |

---

## Unregistered Flags

None. All 10 SUMMARY.md files (165-01 through 165-10) were checked for a
`## Threat Flags` section; none exists in any of them. No new attack surface
was flagged by an executor during implementation that lacks a threat-model
mapping.

---

## Verification Commands Run

```
go build ./cmd/aether                     # exit 0
go vet ./...                              # clean
go test ./cmd/... ./pkg/...               # full pass: cmd 281.254s, all pkg/* green
go test ./cmd/... -run <60+ named tests>  # all PASS (see per-threat evidence above)
```

No implementation file was created, modified, or deleted by the audit itself.
The post-audit fix for T-165-03-05 (guardrail line in continue.md, three
surfaces) was applied by the orchestrator with user approval.

---

## Security Audit 2026-08-03

| Metric | Count |
|--------|-------|
| Threats found | 60 |
| Closed | 60 |
| Open | 0 |

threats_open: 0 — all threats have dispositions.
