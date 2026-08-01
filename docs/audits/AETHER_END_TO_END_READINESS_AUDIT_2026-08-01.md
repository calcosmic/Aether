# Aether End-to-End Readiness Audit

**Audit date:** 2026-08-01  
**Repository:** `calcosmic/Aether`  
**Audited revision:** `ea1587d6460783dcf3360fd1176e7003a624bb42`  
**Purpose:** Decide whether Aether is ready to become the user's default framework for AI-assisted software development.  
**Change boundary:** Report only. No product code, state, release, or configuration was intentionally changed.

## Executive verdict

**No-go for default daily use or a new public release in the present state.**

Aether is not a failed framework. Its Go runtime, durable build evidence, worker-claim validation, recovery concepts, multi-platform assets, and recent GSD review discipline are substantial. The current code is best described as a strong but unproven release candidate built around several foundational defects.

In plain English: the house has an impressive control room, detailed logs, and good emergency procedures, but some door locks, floor plans, and delivery labels are still wrong. Most ordinary journeys may work; the failure modes are serious enough that Aether should not yet be trusted as the one tool on which all development depends.

The adoption blockers are concentrated, not diffuse:

1. A destructive command can escape its intended directory through an unvalidated identifier.
2. downloaded release archives can write outside their staging directory.
3. same-process state locking can lose concurrent updates.
4. inserting a phase breaks assumptions used by build, recovery, and dry-run code.
5. `colonize` can report success while creating state that the recommended next command rejects.
6. the source, installed binary, GitHub release, npm package, documentation, and integrity checks do not identify one trustworthy product revision.
7. the exact real-world experience the user wants—reliable, inexpensive, resumable work in downstream repositories—still has zero completed human UAT checks in the current acceptance record.
8. a fresh install cannot run the primary Claude/OpenCode hosted planning path because its TypeScript dependencies are absent.
9. the documented forced update path overwrites an existing GSD Claude hook configuration instead of merging it.

The recommended stance is therefore:

- **Development/dogfooding:** yes, with supervision, backups, and known-risk avoidance.
- **Default framework for important work:** no.
- **Public promotion as finished:** no.
- **Candidate for a focused finish-line programme:** emphatically yes.

## Readiness scorecard

The scores are judgment aids, not mathematical measurements.

| Area | Score | Assessment |
|---|---:|---|
| Core architecture | 4/5 | Clear Go authority, declarative assets, strong attempt/evidence concepts |
| Worker boundary and completion evidence | 4/5 | Strict claims, process-group cleanup, guarded finalization |
| Filesystem and update safety | 1/5 | Destructive traversal and archive extraction containment defects |
| State integrity and concurrency | 2/5 | Durable patterns exist, but same-process locking and phase identity are unsound |
| Lifecycle correctness | 2/5 | Main path is broad; important ordering, finalization, and transaction gaps remain |
| Automated verification | 3/5 | Very large suite and broad CI design; current full run is not reliably green |
| Distribution and release truth | 1/5 | Public channels and local artifacts materially disagree; integrity gives a false positive |
| Documentation and operator truth | 2/5 | Extensive documentation, but versions, GSD state, and some contracts contradict behavior |
| Learning and feedback loop | 2/5 | Pheromones are real; the advertised consolidation pipeline is not wired into lifecycle calls |
| Real-user acceptance | 1/5 | Named downstream, cheap-model, resume, cost, and intervention proofs remain pending |

**Overall readiness judgment: 48/100 — promising internal release candidate, not a dependable default.**

## Scope and method

The audit combined four independent workstreams:

- repository and release inspection by the primary reviewer;
- a 2,781-commit archaeological review;
- a security, state-integrity, concurrency, and runtime review;
- an end-to-end user/workflow review across installation, lifecycle, recovery, update, and documentation surfaces.

The review examined Go runtime code, TypeScript host code, npm bootstrap code, Claude/OpenCode/Codex platform assets, workflows, release metadata, GSD planning records, local colony state, and public GitHub/npm distribution state.

Safe automated checks were run. Potentially destructive traversal findings were established by code inspection; no exploit was executed. No authenticated paid-model lifecycle was launched because that would spend money and mutate external/provider state. This limitation is itself important: the missing real-provider proof is one of the current release gaps.

## Current product and release reality

There is no single authoritative answer to “which Aether is current?”

| Surface | Observed version/revision | Meaning |
|---|---|---|
| Local source at HEAD | 1.0.46 / `ea1587d6` | Audited candidate |
| Local branch vs cached `origin/main` | 154 commits ahead, 0 behind | 149 files, approximately +23,932/-494 lines beyond the published baseline |
| GitHub latest release | 1.0.43 / `6577f51c` | Last immutable release and last recorded fully green public baseline |
| Public npm `aether-colony` | 1.0.22 | 21 releases behind GitHub and 24 version steps behind local source |
| Installed stable binary | reports 1.0.46, compiled from `e0799c93`, dirty | Same displayed version as source, but 38 commits behind audited HEAD |
| Installed dev binary/hub | 1.0.41 | Separate older development channel |
| README | 1.0.41 in several public surfaces | Stale user guidance |
| AGENTS.md | 1.0.41 | Stale project guidance |
| CLAUDE.md | 1.0.42 header, 1.0.41 table | Internally inconsistent |
| `.planning/STATE.md` | 1.0.45 | Stale and internally contradictory |
| `.planning/PROJECT.md` | 1.0.42 | Stale |
| CHANGELOG newest entry | 1.0.40 | Does not describe the shipped or local product |

The public references checked during the audit were the [GitHub v1.0.43 release](https://github.com/calcosmic/Aether/releases/tag/v1.0.43), its [successful release workflow](https://github.com/calcosmic/Aether/actions/runs/30369488477), and the [public npm package](https://www.npmjs.com/package/aether-colony).

### False-positive integrity result

`aether integrity --source --channel stable` reports all five checks passing and says the publish is fresh. The installed binary nevertheless contains VCS revision `e0799c93`, is marked `vcs.modified=true`, and is 38 commits behind HEAD.

The explanation is visible in `cmd/integrity_cmd.go:89-107`, `:225-275`, and `:323-360`, plus `cmd/update_cmd.go:460-529`: freshness is primarily version equality plus minimum file counts. It does not bind the binary, hub assets, or source checkout to a content digest or commit identity. Two different builds labelled `1.0.46` therefore pass as identical.

This is not cosmetic. It makes Aether's own “safe to use” diagnostic incapable of detecting a same-version stale or mixed installation.

### Installation paths are not release-grade

- README lines 112-143 recommend `go install github.com/calcosmic/Aether@latest`. A clean isolated run fails because the module root is not a `main` package. The installable package is under `/cmd/aether`.
- The README says Go 1.22+ while `go.mod` and workflows require Go 1.26.1.
- `npx --yes aether-colony@latest --bootstrap-version` resolves to 1.0.22, not the GitHub release or audited source.
- The v1.0.43 release workflow succeeded even though its npm publish job was skipped when `NPM_TOKEN` was unavailable (`.github/workflows/release.yml:71-80`, `:166-168`). A green release therefore does not mean all advertised channels were published.
- Tags 1.0.44, 1.0.45, and 1.0.46 do not exist even though local source and binaries use 1.0.46.

**User effect:** a new user can follow the primary instructions, encounter a hard failure, or install a substantially older product while believing it is current.

### A fresh install cannot run the primary hosted planning path

The workflow reviewer exercised both public 1.0.43 and source-built 1.0.46 in isolated homes. In both cases, `install`, `lay-eggs`, and `init` succeeded. The first primary-platform planning step then failed:

```text
aether host plan --dry-run --depth fast --planning-depth light
ERR_MODULE_NOT_FOUND: Cannot find package 'js-yaml'
```

The installed hub had TypeScript-host distribution files but no dependencies, and the fresh repository had no local TS host. The Claude `/ant-plan` wrapper calls this exact hosted path (`.claude/commands/ant/plan.md:39-45`; `cmd/host_cmd.go:183-190`). Running `aether update --force` later installs the missing dependencies and makes the path work, but requiring an undocumented repair immediately after a successful install is not viable onboarding.

On public 1.0.43, `aether integrity --channel stable` reported 4/4 checks passed immediately before the hosted plan failed. Integrity therefore has a second false-green mode: it checks file/version presence but does not prove that the main installed execution path can load.

The update also installed 63 npm packages and created an approximately 61 MB repo-local `.aether/ts-host/node_modules` tree (`cmd/update_cmd.go:592-661`). That contradicts the message that shared machinery stays global and introduces undeclared Node, npm, network, disk, and install-script requirements.

Runtime prerequisites are consequently inaccurate: README says Go 1.22+ and treats Node 18+ as optional, while `go.mod` requires Go 1.26.1, host code rejects Node below 20 (`cmd/host_cmd.go:158-176`), and the practical repair invokes npm.

### Forced update overwrites GSD's Claude hook settings

In a disposable repository containing a GSD-like `.claude/settings.json`, GSD command, `.planning/PROJECT.md`, and `CLAUDE.md`, `aether lay-eggs` preserved the files but did not add Aether's hooks. The documented `aether update --force` remedy then replaced `.claude/settings.json` wholesale: the GSD hook disappeared and only Aether hooks remained. The GSD command, planning file, and `CLAUDE.md` were otherwise preserved.

The relevant copy behavior is in `cmd/platform_sync.go:88`, `cmd/setup_cmd.go:120-122`, and `cmd/update_cmd.go:341-343`. This is not an abstract compatibility concern; it breaks the exact Aether + GSD + Claude Code stack the user intends to operate. Aether needs a semantic JSON merge with explicit ownership/conflict behavior and an idempotent coexistence test. Until then, forced update is unsafe for that workflow.

## Stop-ship findings

### P0 — destructive path traversal in swarm cleanup

`cmd/swarm.go:176-205` accepts `swarm-cleanup --id` without validating it, joins the value beneath the swarm directory, and passes the result to `os.RemoveAll`. `pkg/storage/storage.go:315-322` also permits absolute or uncontained relative paths.

An identifier containing parent-directory components can resolve outside `.aether/data/swarms`. In plain English, a cleanup command intended to remove one Aether work record could delete unrelated repository content.

This must be fixed with a single audited containment primitive used by all destructive paths, plus adversarial tests for absolute paths, `..`, separators, symlink boundaries, and platform-specific path forms. `cmd/chamber.go:26-59` has a related uncontained `--name` write path and should be included in the sweep.

### P0 — archive extraction can escape the staging directory

`pkg/downloader/extract.go:39-70` (tar) and `:106-143` (ZIP) strip a leading archive component and then join the remaining name beneath a staging directory without proving containment. A crafted entry such as `a/../../../target` can write elsewhere.

Checksums do not solve this: a checksum proves that the downloaded bytes match a declared archive; it does not prove that the archive contains safe paths. Because this code participates in install/update delivery, the issue is a supply-chain boundary failure.

Extraction must reject absolute paths, parent-relative paths, device/special entries, and unsafe links; prove each destination remains inside the staging root; and have malicious tar and ZIP regression fixtures.

### P0 — release identity and distribution are fractured

The false integrity result, stale public npm package, invalid README Go command, stale documentation, missing tags, and same-version/multiple-revision local builds collectively make it impossible for a user to establish what they installed.

The release must be treated as a content-addressed bundle: source commit, binary build metadata, checksums, hub asset manifest, npm bootstrap, docs, and tag should agree. A missing required channel should fail the release, not create a green workflow with an old package left as `latest`.

### P0 — fresh primary-platform planning is broken

Both the public and current candidate can say installation and initialization succeeded while failing at `aether host plan` due to a missing runtime dependency. Since Claude and OpenCode are Aether's primary supported platforms, this is a central-path failure, not an optional dashboard defect.

Install/publish must deliver a self-contained runnable host or declare and install its dependencies transactionally. Release integrity must execute a hosted smoke plan; file counts alone cannot detect this failure.

### P0 — the recommended update breaks GSD coexistence

`aether update --force` replaces GSD's Claude hook settings. Because the current finish-line process explicitly uses GSD with Claude Code, the update guidance can disable the system being used to develop Aether.

The settings file must be structurally merged, with Aether-owned hook entries added or updated without deleting unrelated hooks. Setup, update, repeated update, and uninstall/cleanup behavior need disposable-repository regression tests.

### P0 acceptance gate — the desired real-world experience is not proven

`.planning/phases/163-context-reaches-workers/163-HUMAN-UAT.md:15-34` records **0 passed and 3 pending**. The missing checks are:

1. inspect the charter in a real colony within ten seconds;
2. run a real build and prove suggestion closeout is clean;
3. prove normal cheap-model behavior.

Lines 38-45 say the phase was closed because the GSD mechanism had no UAT gate. Phase 171's real-repository, interrupted-resume, downstream-install, cost, operator-intervention, and failure-threshold proofs also remain pending.

Automated contract tests are useful but cannot substitute for these claims. The product should not be called the user's go-to framework until the exact go-to journey has been observed and recorded.

## High-severity correctness and reliability findings

### Same-process locks do not provide mutual exclusion

`pkg/storage/lock.go:54-67` treats an existing lock as re-entrant for any goroutine, not merely the owner. A requested writer can update bookkeeping without upgrading an existing shared operating-system lock; unlock is reference-count based without ownership (`:89-109`).

The advertised read-modify-write safety in `pkg/storage/storage.go:57-76` and `:118-136` can therefore lose updates when parallel workers share one `Store`. Existing tests use separate `FileLocker` instances or only prove that writes returned successfully, so they miss the lost-update condition.

This is a foundational data-integrity defect. It needs per-path in-process reader/writer coordination and deterministic same-`Store` lost-update tests.

### Phase insertion corrupts the ID/index model

`cmd/state_extra.go:152-177` assigns a new maximum ID and inserts it in the middle of the phase slice. Its test accepts an order such as `[1, 3, 2]`. Other paths treat the CLI phase number or `phase.ID` as a slice ordinal: `cmd/codex_build.go:423-432`, `cmd/recovery_snapshot.go:283-297`, and `cmd/compatibility_cmds.go:487-503`.

After insertion, Aether can build, preview, recover, complete, or report the wrong phase. The system must choose one model—immutable IDs resolved by lookup, or contiguous ordinals renumbered atomically—and enforce it everywhere.

### `colonize` before `init` creates poisoned state

When no colony exists, `cmd/codex_colonize.go:1014-1043` can create READY state with no goal and then recommend `aether plan` (`:212-243`). `cmd/state_load.go:17-31` rejects that state, and plan immediately invokes the validation.

This is a simple but damaging happy-path contradiction: the command says success and tells the user what to do next, but that next command cannot run. Either `init` must be required first or pre-init survey output must remain independent of active colony state.

### Wrapper `continue-finalize` can reject valid reconciliation

The pending defect record `.planning/todos/pending/2026-08-01-finalize-reconcile-task-evidence-gate.md:8-28`, the gate at `cmd/codex_continue.go:3055-3067`, and the assertion in `cmd/continue_criterion_evidence_finalize_test.go:187-267` confirm that an external wrapper can provide `--reconcile-task`, pass criterion verification, and still receive `advanced:false`, `blocked:true`.

This affects the Claude/OpenCode wrapper path—the path most relevant to the user's GSD/Claude workflow—rather than merely an obscure diagnostic command.

### TypeScript-host preflight is costly and fragile

`.aether/ts-host/src/platform-dispatcher.ts:149-155` runs a preflight with a literal 20-second timeout on every dispatch. The generated distribution has the same behavior. The Go provider path has `AETHER_PREFLIGHT_TIMEOUT`; the primary TypeScript host path does not.

A downstream report estimates approximately $0.59 per dispatch and reports slow-auth failures. That cost was not independently benchmarked in this audit, but the hardcoded repeated preflight is confirmed. Phase 163.2 exists for this problem and has no implementation plans yet.

## Security and dependency posture

### Go toolchain vulnerabilities

`govulncheck ./...` under Go 1.26.1 found ten called standard-library vulnerabilities with fixes in Go 1.26.2 through 1.26.5. They include TLS privacy/denial-of-service issues, X.509 validation issues, an HTTP/2 loop, and unbounded tar allocation. The reported IDs were:

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

Both the audited toolchain and installed stable binary report Go 1.26.1. Given that Aether downloads over TLS and extracts archives, this is release-blocking until the toolchain is upgraded, binaries rebuilt, and the vulnerability scan is green. CI currently has no `govulncheck` gate.

### TypeScript dependencies

- narrator npm audit: zero known vulnerabilities;
- TypeScript host: two audit findings, including a direct high-severity `js-yaml` denial-of-service advisory and a low transitive `esbuild` issue;
- CI audits the narrator package but not the TypeScript host (`.github/workflows/ci.yml:70-88`).

An open dependency update exists for `js-yaml`, but its PR checks were failing at audit time. All shipped JavaScript workspaces should be audited in the release gate.

## End-to-end user journey assessment

| Journey stage | What works | What prevents confidence | Verdict |
|---|---|---|---|
| Discover and install | GitHub provides multi-platform release assets; bootstrap logic exists | README Go command fails; npm `latest` is 1.0.22; versions disagree; host dependencies are missing | **Fail** |
| Initialize | `init` records goal/session/recovery concepts | state is committed before later artifacts; a later failure can strand an active colony and block retry | **At risk** |
| Survey/colonize | Rich surveyor model and multi-platform assets | pre-init colonize can create unusable state | **Fail for supported ordering** |
| Plan | Raw/synthetic planning machinery is substantial | fresh hosted plan fails on missing `js-yaml`; poisoned state can block plan; GSD roadmap/state are stale | **Fail on primary platforms** |
| Build | Durable attempts, execution binding, strict claims, path validation in finalizer | phase ID/index conflict, shared-store lost updates, worktree merge history, preflight overhead | **At risk** |
| Continue | Verification/gate concepts are strong | wrapper reconciliation defect; continue packet parsing is weaker than build parsing; plan-only mutates despite docs | **At risk** |
| Recover/resume | Reconcile, recovery snapshots, attempt journals, process cleanup are meaningful strengths | phase identity and non-transactional steps undermine exact recovery; real interruption UAT pending | **Unproven** |
| Observe | status, watch, history, reconcile, flags, and vital-sign commands exist | current repository reports contradictory stale state; vital signs report “Healthy” from hardcoded zero rates | **Misleading** |
| Learn/steer | pheromones have deduplication, sanitization, priority, decay, and injection | consolidation pipeline has zero lifecycle callers; two learning systems coexist; Hive is intentionally off by default | **Partly implemented** |
| Seal | ceremony, summaries, promotion, and signal expiry exist | irreversible side effects precede durable completion; summary follows completed state | **At risk** |
| Update/release | publish/update tooling and checksum concepts exist | unsafe extraction, false freshness, stale npm, green partial release, stale docs, GSD settings overwrite | **Fail** |

### Transaction boundaries

Two lifecycle commands can leave partially committed state:

- `init` writes active colony state before session, recovery, and activity artifacts (`cmd/init_cmd.go:171-243`). A later failure blocks a normal retry (`:56-97`).
- `seal` promotes instincts and expires FOCUS signals before saving completed state, then writes the final summary after completion (`cmd/codex_workflow_cmds.go:409-515`).

These should become staged, idempotent transactions with resumable journals. “Idempotent” means retrying the same action safely produces the intended single result rather than duplicating or corrupting it.

### Observability currently overstates health

The audited repository's `aether reconcile --json` reports:

- source/npm/resolved 1.0.46, docs 1.0.45, tag 1.0.43, `aligned:false`;
- READY colony state with no phases and an old goal;
- goal/session mismatch and stale baseline;
- 16 unresolved flags;
- an expired active pheromone;
- 12 backup candidates;
- stale worker-process, spawn-run, and last-build-claim artifacts.

`aether status` reports READY, phase 0 of 0, stale for more than seven days. Yet `aether colony-vital-signs` and patrol call the colony healthy with a score of 75. `cmd/memory_details.go:92-155` hardcodes phases/day, errors/day, and colony age to zero and does not incorporate the stale flags/backups above.

The reconcile command is one of Aether's strongest truth surfaces; the headline health command currently hides the truth it already knows.

## Architecture assessment

### What is genuinely strong

1. **Authority is clearer than in the shell era.** Go owns core state mutation; Markdown/YAML assets hold editable behavior; TypeScript bridges hosted platform execution.
2. **Build completion is evidence-based.** `cmd/build_attempt.go:75-134`, `:193-292`, and `:302-413` persist attempts, bind manifests to executions, and validate staged completion. `cmd/codex_build.go:642-657` guards final state transition.
3. **Worker boundaries are comparatively hardened.** `pkg/codex/worker.go:884-976` uses strict claim schemas; `:1207-1227` cleans process groups; provider errors are sanitized in `pkg/codex/platform_dispatch.go:783-804`.
4. **Build finalizer path claims receive containment checks.** `cmd/codex_build_finalize.go:1292-1345` and `:1549-1717` are examples of the safety discipline that should be applied to every filesystem path.
5. **Cross-platform parity is unusually explicit.** There are 27 agent definitions on each main platform, 61 Claude wrappers, 61 OpenCode wrappers, and 86 shipped skills. Source checks pass.
6. **Tests are a first-class artifact.** The repository contains roughly 3,924 Go test functions and recent Phase 163.1 work added materially more test code than runtime code.
7. **Security review has changed outcomes.** The recent GSD review caught tamper-evidence laundering and added refusal tests instead of merely recording the risk.
8. **Basic isolated CLI setup is viable.** Both public and current binaries launched; `install`, `lay-eggs`, `init`, status, a synthetic raw plan, assumption analysis, and autopilot preview worked. The failure begins at the real hosted platform boundary, which narrows the repair target.

### Complexity and ownership risks

- The CLI exposes approximately 399 Cobra commands, of which 366 are top-level. That is a very large public and maintenance surface for a lifecycle framework.
- Core lifecycle files are very large: several range from approximately 2,900 to 3,900 lines. Build/continue behavior is spread across runtime, finalizers, command guides, playbooks, YAML, Claude wrappers, OpenCode wrappers, and Codex skills.
- `pkg/learn` is live while `pkg/memory` contains a consolidation pipeline with no lifecycle caller. `.planning/REQUIREMENTS.md` still marks the learning requirements pending.
- Model/policy assets have incomplete readership/distribution, while automatic model selection has been intentionally descoped. That user decision should be preserved: finish the explicit policy/control story without silently reintroducing automatic routing.
- History reveals practical bus factor one: three human author identities account for 2,780 of 2,781 commits and use the same email; the remaining commit is a bot.

In plain English: Aether has many individually clever rooms, but too many connecting doors. The finish line should remove or consolidate surfaces and prove the central path, not add another layer.

## Git-history findings

### Evolution

1. **February–early April 2026:** rapid shell-era expansion used v2/v3/v5 labels.
2. **April 2026:** commit `1d11ea85` reset the Go-native product to v1.0.0. The older v5.4.0 tag is therefore not newer than current 1.0.x.
3. **April–May:** releases concentrated on state truth, recovery, platform parity, packaging, ceremony restoration, and worker-loop control.
4. **May–July:** multiple product versions accumulated without tags. v1.0.43 bundled the work and became the first recorded fully green public CI baseline.
5. **July 28–August 1:** 154 local commits added 70 documentation changes, 26 features, 25 fixes, 18 chores, 14 test changes, and one refactor. The delta is test-heavy but unpublished.

### Regression pattern

The dominant historical risk is not lack of functionality. It is disconnection between layers:

- `.claude/commands/ant/build.md` has 115 commits; `continue.md` has 83.
- `cmd/codex_build.go` has 36 commits; `cmd/codex_continue.go` has 41.
- 47 commit subjects contain restore/recover/revert/rollback, and 289 begin with fix.
- worktree incidents `0ab7ff3c`, `d04fa702`, and `9f45a7d2` restored implementation/tests lost by orphaned or overwritten merges; later commits restored additional wave or summary files.
- the current local delta contains 15 executor-worktree merges; `efd9aa25` was needed for a post-wave conflict on 2026-08-01.

This does not prove every current worktree operation is broken. It does prove that “the worker completed” and “the work survived integration” must be separate, enforced facts.

### What GSD is doing well

- Phase 163.1 initially met only four of seven requirements; the process added work rather than declaring a paper success.
- its security review identified a critical evidence-laundering flaw, which was fixed with refusal tests;
- the final recorded targeted set was 22/22 with 44/44 planned threats reviewed;
- the 154-commit delta adds approximately 6,504 test lines for 2,593 runtime lines.

### Where GSD is not yet a release authority

- `.planning/STATE.md` simultaneously says planning, points to both Phase 163.1 and 163.2, reports 1.0.45, and retains obsolete pending planning tasks.
- `.planning/ROADMAP.md` lists inserted phases inconsistently.
- `.planning/PROJECT.md` says 1.0.42.
- Phase 163 closed with all three human UAT checks pending because the mechanism did not enforce them.
- Phase 171's decisive real-world proof remains unstarted.

**Conclusion on GSD:** keep using its verifier/reviewer/security loop; do not let GSD phase completion equal release acceptance. Machine-enforced release gates and explicit human UAT sign-off must sit above it.

## Verification results

| Check | Result | Interpretation |
|---|---|---|
| `go vet ./...` | Pass | Baseline Go diagnostics clean |
| `go test ./... -count=1` | Fail under full-suite load | One 3-second Codex login-status preflight test timed out; isolated rerun passed in 0.28s |
| Independent full-suite audit | Fail under contention in another preflight timing test | Isolated repeated runs, including race-enabled, passed; test boundary is non-hermetic |
| Full `go test ./... -race` at final HEAD | No clean authoritative result | Required before release |
| npm narrator tests | 11/11 pass | Clean functional result |
| TypeScript control tests | 13/13 pass | Clean functional result |
| TypeScript host tests | 494/494 pass | Clean functional result; Node deprecation warnings present |
| TypeScript typechecks | Pass | Both workspaces typecheck |
| `go list ./...` and clean binary launch | Pass | Packages resolve; source binary builds and launches |
| `goreleaser check` | Pass | Release configuration parses/validates |
| `aether source-check` | Pass | Canonical/retired/wrapper counts align |
| strict command-catalog verifier | Fail | New `contract-schema` command lacks classification/since/history metadata |
| `gofmt -l` | Two files | `cmd/codex_continue.go` and `cmd/contract_schema.go` are not formatted |
| `govulncheck ./...` | Fail | Ten called Go standard-library vulnerabilities under Go 1.26.1 |
| narrator `npm audit` | Pass | Zero known issues |
| TS-host `npm audit` | Fail | Direct high `js-yaml` plus low transitive issue |

The full Go failure appears to be a scheduling-sensitive test flaw, not a reproduced product defect. It still matters: a release gate that intermittently fails under the concurrency it is designed to test is not authoritative.

Current CI is broad—it includes build, vet, normal tests, race tests, release snapshots, and TypeScript suites—but it omits Go vulnerability scanning and does not audit the TypeScript host. The release workflow also omits the strict catalog check that current CI would reject.

## Contract and documentation discrepancies

1. **Continue packet parsing is weaker than build parsing.** `cmd/codex_continue_finalize.go:69-105` uses plain `json.Unmarshal`, stops at the first type error, and accepts unknown fields. The build loader now preserves submitted bytes and applies strict schema/batch validation.
2. **Plan-only is documented as non-mutating but writes claims.** `cmd/codex_continue_plan.go:107-119` writes read-only evidence to `last-build-claims.json`, while `.aether/docs/codex-observable-output-contract.md:92-101` promises no `.aether/data` mutation before finalizers.
3. **The old Phase 159 acceptance used stub adapters.** `docs/ACCEPTANCE_REPORT.md` correctly discloses that its end-to-end demo did not make real LLM calls. It proves pipeline mechanics, not current real-provider reliability or cost.
4. **The command catalog has drifted.** There are 61 wrappers on the primary hosted platforms while some documentation still says 60; one new command is unclassified.
5. **“Shared machinery stays global” conflicts with update behavior.** Repairing the fresh-host failure installs a large repo-local dependency tree and requires npm/network access.
6. **GSD coexistence is not a supported contract.** History records GSD as a local development tool rather than a distributed Aether integration, even though current `.planning/STATE.md` uses GSD and it is the intended daily stack.
7. **Autopilot dry-run advances its advice too early.** `aether run --dry-run --max-phases 1` lists build 1 and continue but renders `aether build 2` as the next step even though phase 1 has not run (`cmd/compatibility_cmds.go:435-502`, `:620-624`).

For a stateful orchestration framework, contract/document disagreements are product defects: agents and users act on those contracts.

## Required go-to-framework acceptance gate

The following should be one blocking checklist. Aether is ready only when every item is evidenced at the same tagged revision.

### Gate A — safety and state correctness

- [ ] Fix and regression-test every destructive/user-controlled path, including swarm cleanup and chambers.
- [ ] Make tar/ZIP extraction containment- and link-safe with malicious fixtures.
- [ ] Replace same-process lock bookkeeping with real per-path synchronization and prove no lost updates.
- [ ] Establish and migrate to one phase identity invariant; test insertion through build, dry-run, recovery, continue, status, and completion.
- [ ] Prevent or repair the colonize-before-init state.
- [ ] Make init and seal interruption-safe and idempotently resumable.

### Gate B — one truthful release

- [ ] Choose one version and tagged commit; derive all product surfaces from it.
- [ ] Bind binary, hub assets, npm bootstrap, docs, and checksums to the same commit/content manifest.
- [ ] Make integrity fail for the stale same-version binary demonstrated in this audit.
- [ ] Correct and smoke-test every documented clean-install path in disposable environments.
- [ ] Make a fresh Claude and OpenCode hosted plan work immediately after installation, without a hidden update repair.
- [ ] Declare and enforce the real Go, Node, npm, network, and disk prerequisites.
- [ ] Make npm publication required if npm remains an advertised `latest` channel.
- [ ] Upgrade the Go toolchain, rebuild all binaries, clear called vulnerabilities, and audit every JavaScript workspace.

### Gate C — lifecycle and wrapper parity

- [ ] Fix `continue-finalize --reconcile-task` advancement and match build's strict submitted-packet handling.
- [ ] Make TS-host preflight configurable/cached or remove repeated unnecessary calls; measure actual time and cost.
- [ ] Resolve plan-only mutation by changing either behavior or the contract, with tests.
- [ ] Prove worktree completion and merge-back separately, including conflict and interruption cases.
- [ ] Make status/vital signs derive health from reconcile truth, staleness, flags, backups, and live processes.
- [ ] Merge `.claude/settings.json` structurally so Aether hooks coexist with GSD and other user hooks across setup and forced update.

### Gate D — clean automated release evidence

- [ ] Clean normal and full race suites at final HEAD, without timing-dependent provider tests.
- [ ] Clean Linux, macOS, and Windows CI on the tagged commit.
- [ ] Enforce format, vet, strict command catalog, Go vulnerability, and all npm audit checks in CI and release.
- [ ] Build release artifacts from the clean tagged tree and re-run downstream installation from each channel.
- [ ] Include an integrity smoke test that launches the installed TypeScript host and loads its runtime dependencies.

### Gate E — real daily-driver acceptance

- [ ] Complete the three pending Phase 163 human UAT checks.
- [ ] Run at least three representative tasks in clean, real downstream repositories.
- [ ] Name the inexpensive model/provider being accepted; record cost, latency, retries, and operator interventions.
- [ ] Interrupt a live build, restart the host, and prove exact resume without duplicate or lost work.
- [ ] Exercise init → colonize → plan → build → continue → seal, then begin a second colony.
- [ ] Test failure recovery for provider outage, malformed worker completion, worktree conflict, and partial filesystem write.
- [ ] Define pass/fail thresholds before running the trials; retain raw evidence and a short human usability sign-off.

### Gate F — scope truth

- [ ] Either wire the advertised learning-consolidation pipeline into real lifecycle calls or label it future work.
- [ ] Reconcile `STATE`, `ROADMAP`, `PROJECT`, changelog, README, AGENTS, CLAUDE, and runtime version surfaces.
- [ ] Classify each of the planned unreachable commands/assets as connected, internal, compatibility-only, or retired.
- [ ] Freeze the central user path for one release cycle; prioritize deletion/consolidation and regression locks over new features.

## Recommended sequence for the finish line

1. **Security containment first.** Freeze release/update and destructive cleanup until path and archive boundaries are fixed and tested.
2. **State truth second.** Repair locking, phase identity, poisoned colonize state, and transaction/retry semantics.
3. **Release truth third.** Produce one traceable candidate and make every install channel deliver it.
4. **Primary Claude/GSD path fourth.** Deliver host dependencies, preserve/merge GSD hooks, close finalizer reconciliation and TS preflight behavior, then freeze the contract.
5. **Automated evidence fifth.** Make CI deterministic and add missing vulnerability/distribution gates.
6. **Real user proof last.** Run Phase 171-style downstream trials on the exact candidate without changing it during acceptance.

This ordering matters. Running expensive UAT before filesystem, state, and release identity are trustworthy would test a candidate that cannot safely be shipped.

## Temporary operating posture

Until the gates pass, Aether can be used as a supervised development system on non-critical copies of repositories. It should not be the sole recovery authority or installed from an unverified/mixed channel.

Risk-reduction guidance for dogfooding—not a substitute for fixes—is:

- keep external version-control backups;
- do not use user-controlled or path-like values with `swarm-cleanup` or chamber names;
- do not trust archives from any untrusted or compromised distribution source;
- manually preserve and merge `.claude/settings.json` before any forced update in a GSD repository;
- verify TypeScript-host dependencies are present before relying on Claude/OpenCode planning;
- avoid mid-plan phase insertion;
- run `init` before `colonize`;
- treat `reconcile` as more authoritative than the current vital-sign score;
- verify the actual binary VCS metadata rather than trusting displayed version/freshness alone;
- inspect worktree merge-back before accepting worker completion;
- keep a human in the build/continue/finalize loop.

## Final conclusion

Aether is closer to the finish line than its defect count may suggest. Its distinctive value—the colony model, explicit worker roles, durable evidence, recovery orientation, reusable skills, and multi-platform orchestration—is real. The recent GSD work has strengthened the most difficult parts and has demonstrated a willingness to reject false completion.

However, the present blockers sit below the feature layer. Unsafe paths, unsafe extraction, unreliable mutual exclusion, confused phase identity, poisoned lifecycle ordering, and unverifiable distribution identity are precisely the kinds of defects a development framework must solve before it can safely supervise other software.

The right conclusion is not “rewrite Aether” and not “ship because most tests pass.” It is:

> Freeze feature expansion, close the small set of foundational safety and truth gaps, create one traceable candidate, and make that exact candidate survive real downstream work with an inexpensive model and an interrupted resume.

When those gates are satisfied, Aether will have a credible basis for becoming the user's go-to AI development framework. Today, it does not yet meet that bar.
