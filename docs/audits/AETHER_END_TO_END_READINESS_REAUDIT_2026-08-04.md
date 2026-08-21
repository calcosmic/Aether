# Aether End-to-End Readiness Re-Audit

**Audit date:** 2026-08-04
**Prior audit:** [AETHER_END_TO_END_READINESS_AUDIT_2026-08-01.md](AETHER_END_TO_END_READINESS_AUDIT_2026-08-01.md)
**Prior revision:** `ea1587d6460783dcf3360fd1176e7003a624bb42`
**Re-audited revision:** `e5d290b92733a0be488bb679bfcb3fb3dc3c7a53`
**Purpose:** Determine which findings from the first audit were actually resolved and whether Aether is now ready to become the user's default AI-development framework.
**Change boundary:** Report only. No Aether product code, release, or persistent source-repository state was changed by this audit.

## Executive verdict

**The verdict remains no-go for default daily use and no-go for a public release.**

The user's impression that work has been addressed is partly correct. Aether has made real progress since 2026-08-01:

- the hardcoded, repeated TypeScript preflight was replaced with shared configuration, caching, invalidation, and explicit skip notices;
- the complete uncached Go suite passes at the current runtime revision;
- the complete race-enabled Go suite also passes;
- TypeScript-host coverage grew from 494 to 555 passing tests;
- research now feeds the planning path through a confidence loop, permission-aware dispatch, and Oracle escalation;
- the four core lifecycle wrappers contain more engineering method and stronger cross-platform contract tests;
- Phase 160 improved fail-loud behavior and fixed a real security-gate false positive found during its own UAT.

However, those improvements did not target most findings in the first audit. All original P0 adoption blockers remain open:

1. destructive swarm-cleanup path traversal;
2. unsafe tar/ZIP extraction;
3. fractured release identity and stale public channels;
4. fresh Claude/OpenCode hosted planning fails after a successful install;
5. forced update overwrites GSD's Claude settings/hooks;
6. real daily-driver acceptance remains incomplete.

Four foundational high-severity defects are also unchanged: same-process state locking, phase ID/index corruption, poisoned pre-init colonize state, and wrapper reconciliation that can pass its criteria but remain blocked.

In plain English: Aether has improved the dashboard, route planning, and engine tests, but the unsafe doors and broken delivery process identified three days ago are still there. The vehicle runs more reliably in the workshop; it is still not ready to be the only vehicle the user relies on.

## Comparative readiness score

| Area | 2026-08-01 | 2026-08-04 | Change |
|---|---:|---:|---|
| Core architecture | 4/5 | 4/5 | Strong, unchanged |
| Worker completion evidence | 4/5 | 4/5 | Strong, unchanged |
| Provider preflight | 2/5 | 4/5 | Materially improved |
| Automated verification | 3/5 | 4/5 | Normal and race suites pass |
| Research/planning integration | 2/5 | 4/5 | Substantial implementation, real UAT pending |
| Filesystem/update safety | 1/5 | 1/5 | Stop-ship defects unchanged |
| State integrity/concurrency | 2/5 | 2/5 | Foundational defects unchanged |
| Clean installation | 1/5 | 1/5 | Primary hosted path still reproduces failure |
| GSD/Claude coexistence | 1/5 | 1/5 | Forced update still deletes the GSD hook |
| Release/distribution truth | 1/5 | 1/5 | Public and local products still diverge |
| Documentation/operator truth | 2/5 | 1/5 | GSD state and release surfaces remain contradictory |
| Real-user acceptance | 1/5 | 1/5 | Phase 163 is 0/3; Phase 164 is 0/7 |

**Overall readiness: 52/100, up from 48/100.**

This is a small increase despite 216 new commits because readiness is capped by unresolved P0 safety and onboarding failures. Feature breadth and test volume cannot compensate for a destructive path boundary or an install that cannot execute its primary workflow.

## What changed since the first audit

From the prior revision to the re-audited revision:

- 216 commits landed: 104 documentation, 37 feature, 20 fix, 29 test, and 26 chore commits;
- 168 files changed, adding 31,142 lines and deleting 482;
- the branch grew from 154 to 370 commits ahead of public `main`;
- most changes belong to Phase 163.2 preflight, Phase 164 research-driven planning, Phase 165 lifecycle-wrapper method/ceremony, Phase 160 verification/security closure, and the newly contextualized Phase 162 learning work.

The implementation files behind the original critical findings were compared directly. These remain unchanged since the first audit:

- `cmd/swarm.go`
- `cmd/chamber.go`
- `pkg/downloader/extract.go`
- `pkg/storage/lock.go`
- `pkg/storage/storage.go`
- `cmd/state_extra.go`
- `cmd/codex_build.go`
- `cmd/compatibility_cmds.go`
- `cmd/codex_colonize.go`
- `cmd/state_load.go`
- `cmd/codex_continue.go`
- `cmd/codex_continue_finalize.go`
- `cmd/codex_continue_plan.go`
- `cmd/codex_workflow_cmds.go`
- `cmd/platform_sync.go`
- `cmd/setup_cmd.go`
- `cmd/update_cmd.go`
- `cmd/integrity_cmd.go`

`cmd/init_cmd.go` changed for shelf promotion, but its non-transactional ordering remains and now includes more side effects between state creation and session completion.

## Prior-finding resolution matrix

| ID | Prior finding | Status now | Re-audit evidence |
|---|---|---|---|
| P0-01 | Swarm cleanup path traversal | **OPEN** | Implementation unchanged; raw `--id` still reaches `filepath.Join` and `os.RemoveAll` at `cmd/swarm.go:185-205` |
| P0-02 | Tar/ZIP extraction traversal | **OPEN** | Implementation unchanged; archive names still reach staging joins without containment checks at `pkg/downloader/extract.go:42-70` and `:112-143` |
| P0-03 | Release identity/distribution fractured | **OPEN** | public main/release remain 1.0.43, npm remains 1.0.22, source remains 1.0.46, and installed 1.0.46 binary is 254 commits behind audited HEAD |
| P0-04 | Fresh hosted planning broken | **OPEN — REPRODUCED** | isolated source-built install/init succeeds; `aether host plan --dry-run` fails with `ERR_MODULE_NOT_FOUND: js-yaml` |
| P0-05 | Forced update breaks GSD hooks | **OPEN — REPRODUCED** | isolated `update --force` removes the planted GSD hook, permissions, and custom marker and replaces the entire settings file |
| P0-06 | Daily-driver acceptance absent | **OPEN** | Phase 163 UAT remains 0/3; Phase 164 UAT is 0/7; Phase 171 proof is not started |
| H-01 | Same-process locks lose mutual exclusion | **OPEN** | `pkg/storage/lock.go:54-67` still treats any same-process caller as re-entrant; no ownership/upgrade fix or lost-update test exists |
| H-02 | Phase insertion breaks ID/index invariant | **OPEN** | insert still creates order such as `[1,3,2]`; build and compatibility paths still use `phaseNum-1` and `phase.ID-1` |
| H-03 | Colonize-before-init poisons state | **OPEN** | `cmd/codex_colonize.go:1014-1043` still creates READY state without a goal; `loadActiveColonyState` still rejects it |
| H-04 | Wrapper reconciliation remains blocked | **OPEN, explicitly deferred** | runtime files unchanged; Phase 162 context records the issue as reviewed but deferred again |
| H-05 | TS preflight hardcoded and repeated | **RESOLVED technically** | 45-second cross-host configuration, one-hour per-platform success cache, invalidation, skip notice, temp-directory probe, and tests now exist |
| H-06 | Go 1.26.1 vulnerabilities | **OPEN** | current `govulncheck` again finds the same ten called standard-library vulnerabilities |
| H-07 | TS-host dependency advisories | **OPEN** | npm audit still reports high direct `js-yaml` and low transitive `esbuild` findings |
| M-01 | Init is non-transactional | **OPEN / expanded** | state saves first; shelf mutations now occur next; session/recovery/activity writes remain later and can fail without rollback |
| M-02 | Seal is non-transactional | **OPEN** | promotion/signal expiry still precede completed state; final summary is still written after completion |
| M-03 | Continue packet parsing weaker than build | **OPEN** | `cmd/codex_continue_finalize.go` remains unchanged and still uses permissive plain unmarshalling |
| M-04 | Plan-only mutation contradicts docs | **OPEN** | `cmd/codex_continue_plan.go` remains unchanged |
| M-05 | Vital signs overstate health | **OPEN — REPRODUCED** | reconcile shows stale/misaligned state; vital signs still report Healthy/75 with zero age, velocity, and error rate |
| M-06 | Full normal/race evidence missing | **RESOLVED for local HEAD** | both complete suites passed in this audit |
| M-07 | Command catalog/format gate failures | **OPEN / worse** | unformatted Go files remain two; strict catalog gaps grew from one command to three |
| M-08 | Version and documentation drift | **OPEN** | README/AGENTS/CLAUDE/changelog remain stale and inconsistent |
| M-09 | Learning pipeline not wired | **OPEN, now planned** | docs and tests still enforce zero lifecycle callers; Phase 162 has context but no implementation |
| M-10 | Worktree/resume/downstream proof | **OPEN** | Phase 171 is still pending; no new exact-candidate downstream proof exists |

**Summary:** two prior findings are resolved or materially improved: preflight behavior and local full-suite stability. All six original P0 blockers remain open.

## Stop-ship findings still present

### 1. Destructive swarm cleanup still escapes its base directory

`cmd/swarm.go:185-196` reads an unvalidated identifier and constructs:

```go
swarmDirAbs := filepath.Join(store.BasePath(), "swarms", id)
```

It then passes that result to `os.RemoveAll` at line 204. An absolute or parent-relative identifier can resolve outside the intended swarm directory. `cmd/chamber.go:26-59` retains the related uncontained user-supplied name used for directory creation and manifest writes.

No adversarial traversal tests were added. This remains the most immediately destructive defect in the audit.

### 2. Release archives still permit staging-directory escape

Tar and ZIP extraction still strip one leading component and join the remaining archive-controlled name beneath the staging directory without `filepath.Rel` containment proof. There are still no malicious archive fixtures covering parent paths, absolute paths, link entries, or platform-specific path forms.

This combines badly with the unchanged Go 1.26.1 `archive/tar` vulnerability. The unsafe custom path logic and vulnerable standard library are independent problems; fixing only one is insufficient.

### 3. A fresh primary-platform install still cannot plan

The current source was built into a disposable binary and installed into an isolated home/repository using supported flags. Results:

| Step | Result |
|---|---|
| `aether install --package-dir <source> --home-dir <isolated> --skip-build-binary` | Pass |
| `aether lay-eggs --home-dir <isolated> --repo-dir <isolated-repo>` | Pass |
| `aether init "Audit fresh hosted plan"` | Pass |
| `aether host plan --dry-run --depth fast --planning-depth light` | **Fail** |

Failure:

```text
Error [ERR_MODULE_NOT_FOUND]: Cannot find package 'js-yaml'
imported from .../.aether/system/ts-host/dist/caste-config.js
```

`cmd/install_cmd.go` publishes the TypeScript host but does not make its runtime dependencies available to the installed host. The repair remains `aether update --force`, which is undocumented as an immediate post-install requirement and has the destructive coexistence behavior below.

### 4. Forced update still deletes GSD's Claude configuration

The isolated repository's `.claude/settings.json` was replaced with a test configuration containing:

- a GSD `PreToolUse` hook;
- an unrelated permissions entry;
- a custom marker.

Current `aether update --force`:

- installed 63 packages;
- created a 60 MB repo-local `.aether/ts-host/node_modules` tree;
- emitted npm install-script warnings;
- removed all three planted settings;
- replaced the file with Aether-only hooks.

The cause remains `preserveLocalChanges: !force && pair.preserveLocalChanges` in `cmd/update_cmd.go:341-349`. Force changes “preserve” into whole-file replacement; there is still no structural JSON merge.

This directly contradicts the intended Aether + GSD + Claude Code workflow. It is not safe to recommend `update --force` in a GSD repository without a manual settings backup and merge.

### 5. Release integrity remains a false-positive signal

The installed stable binary reports 1.0.46 and the hub reports 1.0.46, so `aether version --check` and `aether integrity --source --channel stable` pass 5/5.

The installed binary's embedded revision is still `e0799c93f338...`, marked dirty, compiled with Go 1.26.1, and now **254 commits behind** the audited source. The source is also 370 commits ahead of public main.

`cmd/integrity_cmd.go:225-359` and `cmd/update_cmd.go:460-529` still compare displayed versions and minimum companion-file counts. They do not bind source, binary, hub, npm package, or installed host dependencies to a commit/content manifest and do not launch a hosted smoke plan.

In plain English: the label says 1.0.46 on two different products, so Aether says they match.

## State and lifecycle correctness

### Same-process state locking remains unsound

`pkg/storage/lock.go:58-67` still returns success for any existing path entry, increments a counter, and treats the acquisition as re-entrant. It does not know which goroutine owns a write lock and cannot upgrade the underlying operating-system shared lock. The full race suite does not detect logical lost updates because this defect does not require a memory data race.

Passing `-race` is therefore good evidence but not evidence that this issue is resolved.

### Phase identity remains internally contradictory

`insert-phase` still assigns `maxID+1` and inserts the phase at an arbitrary slice position (`cmd/state_extra.go:152-177`). At least twenty production references still index phases by ordinal, including:

- `cmd/codex_build.go:431`
- `cmd/compatibility_cmds.go:502`
- `cmd/recovery_snapshot.go:297`
- `cmd/phase.go:45`
- `cmd/codex_continue_plan.go:88`

Aether still mixes immutable-looking IDs with one-based slice positions. No migration or invariant decision landed.

### Init's transaction window has grown

Phase 165 correctly moved shelf promotion below successful state creation and behind wrapper consent. That fixes cancellation/consent ordering. It does not make the whole command atomic.

Current ordering is:

1. save `COLONY_STATE.json`;
2. promote/dismiss shelf selections;
3. save `session.json`;
4. write recovery artifacts;
5. append activity.

A failure in steps 3-5 can now leave active colony state plus partially consumed shelf items. The first audit's transaction finding remains and should be retested across every new side effect.

### Continue reconciliation is knowingly still open

The `continue-finalize --reconcile-task` defect is unchanged. The newly created Phase 162 context explicitly lists it under “Reviewed Todos (not folded)” and says it was deferred by Phases 163.2, 164, and 165.

Deferral can be a valid scope choice, but it means the primary wrapper path still cannot be described as complete.

## What genuinely improved

### Preflight cost/configuration architecture

Phase 163.2 is a substantive repair, not paperwork:

- `.aether/ts-host/src/preflight-config.ts` parses shared Go-style durations;
- the default is aligned at 45 seconds across hosts;
- `cmd/preflight_cache.go` adds a per-platform, one-hour success trust window;
- bad or future-schema cache files are handled deliberately;
- provider/auth failures invalidate cached trust;
- `AETHER_SKIP_PREFLIGHT` is explicit and never silent;
- probes run from neutral temporary directories;
- both dispatch chokepoints use the same gate;
- generated TypeScript distribution files were rebuilt;
- cross-host drift and probe-count tests exist.

The earlier downstream $0.59-per-dispatch estimate was not independently remeasured, so cost improvement is technically credible but not yet quantified in real use.

### Automated test stability

The exact commands run for this audit produced:

| Check | Result |
|---|---|
| `go test ./... -count=1 -timeout 20m` | **Pass**, `cmd` 285.950s |
| `go test ./... -race -count=1 -timeout 30m` | **Pass**, `cmd` 334.493s |
| `go vet ./...` | **Pass** |
| narrator typecheck/tests | **Pass**, 13/13 |
| TS-host typecheck/tests | **Pass**, 555/555 |
| npm bootstrap tests | **Pass**, 11/11 |
| `goreleaser check` | **Pass** |
| `aether source-check` | **Pass**, 16 canonical / 5 retired / 122 wrappers |

This closes the prior evidence gap around a final local normal/race run. It does not provide CI evidence for the local 370-commit stack: GitHub's public main still points to `6577f51c`, and no CI run exists for this local HEAD.

### Fail-loud verification

Phase 160's re-verification reports 13/13 requirements and its security review reports 28/28 threats closed. The phase added useful invariants around documented CLI calls, stderr suppression, scanner execution, worker failure artifacts, and retired package boundaries.

Its UAT found an absent-file security scan that incorrectly passed, fixed it, and added a regression test. This is good review behavior.

The same UAT also disclosed an open CLI issue: `gate-results-write --name ...` silently defaults omitted `--passed` to false and mutates colony state. `cmd/gate.go:1329-1355` confirms that behavior remains. It is not a release blocker by itself, but a state-mutating evidence command should require an explicit pass/fail value.

### Research and lifecycle method

Phase 164 added real research recommendation, approval, dispatch, confidence scoring, planner-context injection, and Oracle escalation behavior. Phase 165 rewrote the primary wrappers with method, purpose, stop conditions, parity tests, and clearer ownership.

These are meaningful improvements to Aether's distinctive value. Their production acceptance remains limited:

- Phase 164 UAT records 0 passed, 7 pending;
- Phase 165's prose-quality sign-off was performed by the implementing/verifying agent rather than independent user use;
- fresh hosted planning cannot reach the new path without the undocumented update repair.

## Security and quality gates

### Go vulnerabilities remain unchanged

Running `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` under Go 1.26.1 again found ten called standard-library vulnerabilities:

- `GO-2026-5856`
- `GO-2026-5039`
- `GO-2026-5037`
- `GO-2026-4971`
- `GO-2026-4947`
- `GO-2026-4946`
- `GO-2026-4918`
- `GO-2026-4870`
- `GO-2026-4869`
- `GO-2026-4866`

Fixes are available in Go 1.26.2 through 1.26.5. CI still does not enforce `govulncheck`.

### JavaScript advisories remain unchanged

- narrator workspace: zero known vulnerabilities;
- TypeScript host: direct high-severity `js-yaml` CPU denial of service and low transitive `esbuild` issue;
- fixes are available;
- the TypeScript-host audit is still absent from the main CI gate.

### Formatting and catalog gates are still red

- `gofmt -l` still reports `cmd/codex_continue.go` and `cmd/contract_schema.go`;
- strict catalog verification now reports three unclassified commands: `contract-schema`, `plan-research-approve`, and `plan-research-escalate`;
- all three also lack `since_version` and `historical_presence` metadata;
- the catalog surface grew from 399 to 401 commands;
- `git diff --check` finds generated TypeScript trailing whitespace in `dist/platform-dispatcher.js` in addition to intentional Markdown hard breaks in the prior audit report.

The new research commands were shipped without the catalog metadata expected by the repository's strict verifier.

## Release and distribution truth

As of 2026-08-04:

| Surface | Current value | Status |
|---|---|---|
| Audited source | 1.0.46 / `e5d290b9` | Local only |
| Public GitHub main | `6577f51c` | 370 commits behind local source |
| Latest GitHub release | [v1.0.43](https://github.com/calcosmic/Aether/releases/tag/v1.0.43) | Unchanged since first audit |
| Public npm | [aether-colony 1.0.22](https://www.npmjs.com/package/aether-colony) | Unchanged since 2026-04-24 |
| Installed stable binary | displays 1.0.46 / embeds `e0799c93` dirty | 254 commits behind audited source |
| Installed development binary | 1.0.41 | Older separate channel |
| README public version | 1.0.41 | Stale |
| AGENTS current version | 1.0.41 | Stale |
| CLAUDE current version | 1.0.42 header / 1.0.41 table | Contradictory |
| CHANGELOG newest release | 1.0.40 | Stale |
| GSD product version | 1.0.45 | Stale |

README still recommends the invalid module-root command `go install github.com/calcosmic/Aether@latest`, says Go 1.22+, describes Node 18+ as optional, and says 1.0.41 is current. The runtime actually requires Go 1.26.1; hosted execution requires Node 20+ and npm dependencies.

The last public CI for main remains tied to the v1.0.43 revision. Local test success is not public release evidence.

## GSD process assessment

### The good

GSD is continuing to find and correct real defects:

- Phase 163.2 went through review fixes for corrupt/future cache files, duration parsing, kill budgets, drift guards, and invalidation;
- Phase 164 required two gap-closure rounds before verification;
- Phase 165 required two gap-closure rounds around shelf consent, ordering, and durable todo display;
- Phase 160's later verification/security/UAT found issues beyond the original implementation and fixed the in-scope security defect.

### The dangerous mismatch

GSD phase completion is still not global product readiness. The completed phases have internally scoped goals, so a green phase can coexist with unchanged foundational blockers elsewhere.

The current `.planning/STATE.md` is itself contradictory:

- frontmatter says `stopped_at: Phase 162 context gathered`;
- “Current focus” says Phase 165;
- “Current Position” says Phase 166;
- progress says 6 of 29 phases complete and also `percent: 100`;
- the pending list still says to plan Phases 160, 164, and 165 even though they executed;
- it says the roadmap is awaiting approval and “do not start Phase 160,” while Phases 160, 163, 163.2, 164, and 165 have been worked or completed;
- ROADMAP summary checkboxes leave Phases 164/165 unticked while the completion table marks them complete.

This drift worsened during the audit when a concurrent GSD context session advanced HEAD by two documentation commits but updated only some state fields. A fresh agent cannot safely determine where work actually stands from the primary GSD state file.

### Forward-looking Phase 162 risk

Phase 162 has just been contextualized, not implemented. It plans to:

- make Hive read and promotion default-on;
- remove the per-colony consent file;
- invoke learning consolidation at phase end and seal;
- declare `pkg/memory` authoritative with `pkg/learn` subordinate.

This can deliver the user's “everything networked together” goal. It also expands cross-repository data flow and prompt-input trust. Before default-on promotion ships, its acceptance must explicitly cover source provenance, sanitization, project isolation, revocation, contradiction quarantine, secrets/privacy, offline behavior, and a user-visible opt-out. The existing foundational state lock should be fixed first because the new cache/consolidation paths depend on atomic storage.

## Current repository health signal

`aether reconcile --json` reports warning status with:

- version alignment false: source/npm/resolved 1.0.46, docs 1.0.45, tag 1.0.43;
- active READY colony with no phases;
- session/colony goal mismatch;
- 16 unresolved flags;
- one expired active pheromone;
- 12 backup candidates;
- stale worker process, spawn run, and build claim files.

`aether colony-vital-signs` still reports:

```json
{
  "health_label": "Healthy",
  "overall_health": 75,
  "colony_age_hours": 0,
  "build_velocity": {"phases_per_day": 0},
  "error_rate": {"errors_per_day": 0}
}
```

The richer `status` output now exposes more flags, review findings, spawn activity, and guided actions, which is useful. The headline health computation remains misleading and still cannot reconcile its own diagnostics.

## Updated adoption gate

The prior gate remains valid. Based on this re-audit, the minimum blocking order is:

### Gate 1 — stop destructive and supply-chain paths

- contain every destructive/user-derived filesystem path;
- make tar/ZIP extraction traversal- and link-safe;
- upgrade to a patched Go toolchain and rebuild all binaries;
- update vulnerable TS-host dependencies and enforce both vulnerability scans in CI.

### Gate 2 — restore state invariants

- implement real same-process per-path synchronization;
- choose one phase ID/ordinal model and migrate all consumers;
- prevent colonize-before-init poisoned state;
- make init and seal staged, recoverable, and idempotent;
- require explicit values for state-mutating evidence flags.

### Gate 3 — make the advertised install usable and coexist safely

- deliver TypeScript-host runtime dependencies during install or ship a self-contained host;
- structurally merge `.claude/settings.json` without deleting GSD or user hooks;
- document actual Go/Node/npm/network/disk requirements;
- smoke-test fresh Claude and OpenCode hosted planning before declaring integrity.

### Gate 4 — establish one release identity

- bind source commit, binary revision, hub manifest, checksums, npm bootstrap, docs, and tag;
- make a same-version stale binary fail integrity;
- make every advertised channel publish the same candidate or fail the release;
- run current CI on the exact tagged candidate.

### Gate 5 — close primary lifecycle defects

- fix `continue-finalize --reconcile-task` advancement;
- align continue packet validation with build;
- resolve plan-only mutation versus contract;
- derive vital signs from real reconcile data;
- clear strict catalog, format, and generated-file gates.

### Gate 6 — prove the product in real use

- finish Phase 163's 3 pending UAT items;
- finish Phase 164's 7 pending UAT items;
- run Phase 171's three real downstream tasks on the named inexpensive model;
- record cost, latency, retries, operator intervention, and failure thresholds;
- prove interrupted resume and worktree merge-back;
- execute install → init → colonize → plan → build → continue → seal in a separate repository on the exact release candidate;
- obtain a human usability sign-off independent of the agent that implemented the work.

## Recommended immediate course

The current sequence should pause feature expansion before implementing default-on cross-repository learning. The highest-value next work is not another orchestration feature; it is the unresolved audit foundation:

1. filesystem containment and safe extraction;
2. state locking and phase identity;
3. fresh-install TS-host delivery and GSD settings merge;
4. patched dependencies/toolchain;
5. release provenance and truthful integrity;
6. wrapper reconciliation and real downstream UAT.

Phase 162 can then be implemented on trustworthy storage and distribution boundaries rather than adding a new cross-repository writer onto the known lock and release problems.

## Final conclusion

Aether is better than it was on 2026-08-01. The preflight repair is solid, the complete normal and race suites now pass, research/planning gained real machinery, and the wrapper method is more legible and better fenced. Those improvements deserve to be retained.

They do not change the release decision. The first audit's highest-risk findings were mostly outside the scope of the phases that ran, and their code is still present unchanged. Clean-install planning and GSD settings destruction were reproduced again on the current source build. Public distribution and integrity still cannot identify one product. Real user acceptance remains empty.

The correct current description is:

> Aether is a stronger, well-tested internal development candidate with improving orchestration, but it is still unsafe and operationally inconsistent at the filesystem, state, installation, and release boundaries required of a go-to framework.

Do not promote it to default-framework status yet. Close the foundational gates above, publish one traceable candidate, and prove that exact candidate in real downstream use.
