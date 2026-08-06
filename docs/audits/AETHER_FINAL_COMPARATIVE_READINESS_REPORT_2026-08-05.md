# Aether Final Comparative Readiness Report

**Audit date:** 2026-08-05  
**Purpose:** Final report-only handoff for Claude Code after the 2026-08-04 readiness and root-cause reports  
**Primary tested revision:** `233c0ebef0aafc708295050badff001a3b108f01`  
**Post-cut revision reviewed:** `411ac122611f931c0771ef3fd3ef8df86895c71a`  
**Change boundary:** Report only. No Aether product code, release, install, hub, or source state was intentionally changed by this audit.  
**Method:** Independent git-history, runtime-safety, and real-workflow reviews; clean detached snapshots; normal and race Go suites; Go vet and vulnerability scanning; TypeScript/npm tests and audits; daily-driver smoke; source and public-channel comparison.

## Executive verdict

**Aether is materially better than it was on 2026-08-04, but it is still a no-go as the user's dependable, default AI-development framework and a no-go for public release.**

The previous reports were not overtaken by events. Claude fixed several of their most urgent findings correctly:

- malicious tar and ZIP entries are now contained;
- destructive `swarm-cleanup` traversal is blocked;
- `update --force` structurally preserves GSD and unrelated Claude settings;
- the TypeScript host can provision its own dependencies;
- colonize-before-init no longer fabricates an unusable colony;
- inserted phase IDs are renumbered sequentially;
- verified reconciled work can advance instead of deadlocking;
- the Go toolchain was upgraded and current `govulncheck` finds no called vulnerabilities;
- the normal build wrapper now obtains its manifest directly from Go;
- skill matching and skill injection are real runtime paths, not documentation-only claims.

Those are important repairs. They move Aether from **obviously unsafe and locally broken** to **a credible recovery candidate**.

They do not yet make it the product the user asked for. Five adoption-level facts remain decisive:

1. The exact candidate has not completed a real Claude Code or OpenCode install-to-seal journey in a clean downstream repository.
2. The cheap-model policy is still an unread YAML file; Phase 161 is not started and Phase 171's three cheap-model tasks are not started.
3. State locking, continue-manifest binding, lifecycle transactions, and phase-evidence migration still have correctness holes.
4. Source, installed binary, public GitHub release, npm package, and development channel still identify different products.
5. The full Go release gate is environment-sensitive and red in a clean Codex/no-remote snapshot; the TypeScript host also retains a directly used high-severity dependency advisory.

### For dummies

Aether's engine now starts, several dangerous holes have been covered, and its skill system really does hand relevant instructions to workers. But the gearbox is still connected through several different control panels, the odometer shows four different versions, and nobody has completed the full road test that matters. It is ready for controlled recovery work. It is not ready to be the one tool the user trusts without babysitting.

## Adoption score

| Area | 2026-08-04 | 2026-08-05 | Current assessment |
|---|---:|---:|---|
| Core architecture and concept | 4/5 | 4/5 | The colony concept remains coherent |
| Worker completion evidence | 4/5 | 3/5 | Strong build binding; continue binding remains weaker |
| Provider preflight | 4/5 | 4/5 | Configuration and provisioning improved |
| Automated verification | 4/5 | 3/5 | Broad coverage, but exact clean-snapshot suites are not hermetic |
| Research/context/skill integration | 4/5 | 4/5 | Real wiring exists; human proof remains incomplete |
| Filesystem and update safety | 1/5 | 3/5 | Original P0s fixed; sibling identifier paths remain open |
| State integrity and concurrency | 2/5 | 2/5 | Foundational lock/transaction defects remain |
| Clean installation | 1/5 | 4/5 | Fresh-host dependency failure fixed; full lifecycle not proved |
| GSD/Claude coexistence | 1/5 | 5/5 | Structural settings merge is a genuine fix |
| Release/distribution truth | 1/5 | 1/5 | Still fractured and content-unbound |
| Documentation/operator truth | 1/5 | 2/5 | User docs improved; GSD control records contradict each other |
| Real-user and cheap-model acceptance | 1/5 | 1/5 | Still pending |

**Overall readiness: approximately 60/100, up from 52/100.**

The increase is real. Readiness is still capped by unresolved state correctness, release identity, dependency health, and absent real-user proof. Another thousand unit tests cannot lift those caps.

## Audit boundary and moving worktree

All repeatable gates were pinned to commit `233c0eb`. During the audit Claude continued working in the shared checkout. Four later commits were reviewed:

- `5e52dbfb` — adds a status/update-availability nudge;
- `4453d237` — documents that nudge in the owner checklist;
- `3d81adae` — reports the repository's before/after version during update;
- `411ac122` — documents that transition output.

Together, the post-cut commits add 481 lines and delete 3 lines across seven files. They do not resolve any blocker in this report. The exact execution evidence remains pinned to `233c0eb`; the later update-notification feature received source review only.

From the publication cutoff of the 2026-08-04 reports (`2b86e191`) through `411ac122`, Aether added:

- 84 commits;
- changes to 183 files;
- 13,233 inserted lines and 2,590 deleted lines;
- 13 feature, 21 fix, 12 test, 30 documentation, and 8 other commits.

That is a very large one-day change window for a release candidate. It is also direct evidence that Aether has not yet entered stabilization.

## Direct answer: why did it become so difficult?

The framework did not become bad because the original idea was bad. It became difficult because the Go migration changed the shape of authority before the old behavior had been captured as an end-to-end contract.

The sequence was:

1. Classic Aether relied on editable Queen/wrapper behavior and the host platform's native agent execution.
2. A broad Go conversion moved state and orchestration into strict commands over a short period.
3. The cutover happened while the migration audit still contained gaps and before a golden user journey was mandatory.
4. Behavior missing from Go was restored through a TypeScript host, so Aether acquired another critical control layer rather than one replacement layer.
5. YAML sources, generated Claude wrappers, OpenCode wrappers, Codex skills, command-guide output, TypeScript, and Go then evolved together but not atomically.
6. GSD phases closed on plan artifacts, targeted tests, and verification documents while real Claude/OpenCode use was deferred to Phase 171 or the owner.
7. New capabilities continued landing before the safety kernel and acceptance loop were complete.

This is the “more made it worse” effect, but the problem is not feature count by itself. The problem is **authority multiplication**:

```text
user intent
   -> platform wrapper
      -> YAML/generated mirror
         -> Go command or TypeScript host
            -> Go plan/finalizer
               -> persistent state
                  -> renderer and recovery hints
```

Every extra boundary creates another place where names, flags, schemas, paths, versions, and failure meanings can drift. A cheap model is especially sensitive to this because it has less capacity to infer which layer is actually authoritative.

### What the Go migration got right

Go is a good fit for:

- atomic and locked state writes;
- path and archive containment;
- schema and manifest validation;
- deterministic state transitions;
- version and content verification;
- a small, dependable CLI kernel.

### What went wrong in the migration

Go grew into the orchestration product instead of remaining the safety kernel. Production Go is now roughly 126,900 lines across 352 files and exposes 399 catalogued commands. The TypeScript host remains roughly 11,600 lines, while Claude and OpenCode wrapper surfaces add thousands more. The result is not one Go replacement for the old framework; it is the old wrapper model plus a Go control plane plus a TypeScript restoration layer.

The recovery work since the prior report improved individual seams. It did not yet collapse the system back to one primary vertical slice.

## What is genuinely working now

### 1. Archive extraction containment

`pkg/downloader/extract.go` now applies lexical containment to tar and ZIP paths and rejects/skips link-like entries. Adversarial fixtures cover traversal, absolute paths, absence of escaped files, and valid archives. The original P0-02 is resolved.

### 2. GSD and Claude settings coexistence

`cmd/settings_merge.go` structurally merges `.claude/settings.json`. It preserves custom top-level keys, permissions, GSD hooks, and other foreign hooks; refreshes Aether-owned hooks; rejects invalid JSON; and is idempotent. The exact smoke gate passed both preservation and repeat-update checks. The original P0-05 is resolved.

### 3. Fresh-host dependency provisioning

The runtime now provisions `.aether/ts-host` dependencies with `npm ci` and a lockfile-hash stamp. Missing, stale, and legacy states are tested. The old immediate `ERR_MODULE_NOT_FOUND: js-yaml` failure is resolved.

This is a technical repair, not an architecture simplification. First use still depends on Node, npm, network/package availability, and a sizeable `node_modules` installation for lifecycle paths that use the host.

### 4. Several lifecycle correctness defects

- Colonize-before-init no longer creates poisoned READY state.
- Inserted phases receive sequential IDs.
- Verified reconciliation can satisfy completion and advance.
- The normal Claude/OpenCode build wrapper uses a direct Go plan-only manifest instead of hopping through the TypeScript host.

These are real improvements. Phase insertion remains only partially repaired because durable artifacts keyed by the old phase number are not migrated.

### 5. Skills are a real system

This part of the user's intended concept exists:

- `skill-index` discovers shipped, user, and repository skills;
- `skill-match` scores workflow, role, task, and codebase evidence;
- `skill-inject` loads matched content under a bounded budget;
- build, colonize, plan, continue, quick, swarm, and seal paths call the resolver;
- worker manifests expose matched skill names and injected skill content.

In plain English: placing a valid `SKILL.md` in a supported custom-skill folder can influence a worker without hard-coding the skill into the Queen. The Queen chooses the task and caste; the runtime matches the recipe cards. This is one of the healthier parts of the current design.

The mechanism works; current selection quality does not yet meet the user's goal. In an isolated journey:

- a task explicitly requesting a Go CLI matched the Rust skill;
- a Probe matched React, Svelte, and Vue because substring matching found `ui` inside `quality`;
- four skills were reported as matched, but only one reached the injected section because the first skill consumed the entire 8,000-character budget.

A tiny hello-world discovery phase also selected six workers, including three Oracles, with no model field in the manifest. The worker context capsule itself was present and measured at 1,590 characters. The failure is therefore not “skills do nothing”; it is that coarse matching, unbounded individual skill size, and over-selection waste the very context and model budget Aether is supposed to protect.

### 6. Go vulnerability remediation

The module now declares Go 1.26.5. On the exact audited snapshot:

```text
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
```

The scan found two vulnerabilities in required modules that Aether does not appear to call. The current Go toolchain remediation is therefore verified, not merely documented. Go's official release history confirms that 1.26.5 includes security fixes: <https://go.dev/doc/devel/release>.

### 7. Useful deterministic smoke coverage

`make smoke` passed and proved:

- forced update preserves foreign settings;
- repeated forced update is idempotent;
- host dependencies are present enough to execute the host;
- direct Go build planning emits a dispatch manifest;
- OpenCode commands, agents, router, and hint syntax install.

This is useful plumbing coverage. It is not a real colony acceptance test.

## Verification results at the audited revision

| Gate | Result | Interpretation |
|---|---|---|
| `git diff --check` | PASS | No whitespace errors in the pinned snapshot |
| `go vet ./...` | PASS | Static Go checks clean |
| `govulncheck ./...` | PASS | Zero called Go vulnerabilities |
| `gofmt -l cmd pkg internal` | FAIL | `cmd/contract_schema.go`, `cmd/settings_merge_test.go` |
| Strict command catalog | FAIL | 3/399 commands lack classification/version/history metadata |
| `make smoke` | PASS | Deterministic install/update/build-plan surfaces pass |
| Ceremony narrator typecheck/tests | PASS | 13/13 tests |
| TypeScript host typecheck/tests | PASS | 555/555 tests |
| npm bootstrap tests | PASS | 11/11 tests |
| Ceremony narrator `npm audit` | PASS | 0 advisories |
| TypeScript host `npm audit` | FAIL | 1 high direct `js-yaml`; 1 low transitive `esbuild` |
| Full Go suite in clean Codex/no-remote snapshot | FAIL | 17 packages pass; `cmd` fails 16 tests |
| Full race suite in the same snapshot | FAIL | 17 packages pass; same 16 failures; no data-race warning observed |
| Clean source install/init | PASS | 1,213 companion files installed at 1.0.48; `lay-eggs` and `init` completed without hand-editing state |
| Hosted-plan first use | PASS with cost | Automatically installed 63 packages and ran instead of failing on missing `js-yaml` |
| Representative pause/resume | PASS | Goal, phases, BUILT state, tasks, attempt, failure evidence, and recovery command survived |
| Synthetic/offline continue | FAIL | `--synthetic` launched a real Watcher, contacted a provider, received HTTP 401, and blocked |
| Real Claude lifecycle | NOT RUN/NOT RECORDED | Owner checklist is instructions, not result evidence |
| Real OpenCode lifecycle | NOT RUN/NOT RECORDED | Smoke checks installed files, not live orchestration |
| Cheap-model benchmark | NOT RUN/NOT RECORDED | Phase 161 and Phase 171 proof remain open |

### Why the 16 Go failures matter

Fourteen failures are visual/golden tests that assume Claude slash-command hints. The clean audit ran inside Codex, so inherited `CODEX_CI`/`CODEX_THREAD_ID` made `DetectActivePlatform()` correctly select Codex and render `aether ...` commands. The tests did not pin their intended platform. A focused subset passes with `AETHER_PLATFORM=claude`.

The affected tests are:

- `TestCeremonyCloseoutBlockedPathRendersBlockedNotCompletion`
- `TestPlanVisualOutput`
- `TestBuildVisualOutputShowsSpawnPlan`
- `TestColonizeVisualOutputShowsDispatchPreview`
- `TestContinueBlockedVisualOutputShowsWorkerFlow`
- `TestContinueVisualOutputShowsColonyCompleteStageMarker`
- `TestPrintNextUpVisualOutput`
- `TestRenderBinaryActionVisualPublishGuidanceSeparatesRepoSetupFromUpdate`
- `TestRenderUpdateVisualShowsRemovedAssets`
- `TestSetupVisualOutput`
- `TestPauseResumePatrolPhaseAndHistoryVisualOutput`
- `TestGoldenPlanVisualOutput`
- `TestGoldenBuildVisualOutput`
- `TestGoldenContinueVisualOutput`

This is primarily a test-isolation defect, not proof that Claude users see the wrong hints. It still means the documented `go test ./...` gate is not environment-independent and is red in a supported Codex development environment.

Two failures reveal a real no-remote repository edge case in seal/Hive learning. Without an `origin`, seal falls back to the current directory name as `sourceRepo`. In the `cmd` package, abstraction replaces every occurrence of `cmd` in an instinct with `<repo>`. The sanitizer then rejects `<repo>` as an XML structural tag, so eligible instincts are not promoted. The seal continues, but reports promotion failures. The focused tests pass in the canonical checkout because its `origin` yields `Aether`, so the collision does not occur.

Those tests are `TestSealHiveEligibleLog` and `TestSealHivePromotedCount`. With `AETHER_PLATFORM=claude` in the no-origin snapshot, the fourteen visual tests pass and only these two fail. In the canonical checkout with an `origin`, these two pass as well. That isolates both causes precisely; it does not excuse the environment dependence.

Plain English: the learning code creates its own placeholder and then treats that placeholder as an attack. It only happens for certain repository names, which is exactly why clean downstream testing matters.

### Smoke-test limitation

The host smoke gate records the host exit code but passes any nonzero exit unless output looks like a missing-dependency error (`scripts/smoke-daily-driver.sh:93-107`). In this audit it printed:

```text
host executed (exit 1, no dependency errors)
```

The OpenCode lane validates installed files, counts, router presence, and hint text. Even when the OpenCode binary is present, it does not execute an OpenCode lifecycle (`scripts/smoke-daily-driver.sh:148-173`). The script also hand-writes a `COLONY_STATE.json` fixture before testing build planning rather than reaching that state through init and plan. A green smoke result must not be described as daily-driver acceptance.

## Prior-finding resolution matrix

| ID | 2026-08-04 finding | Status now | Evidence and residual |
|---|---|---|---|
| P0-01 | Destructive swarm cleanup traversal | **Original defect resolved; family partially open** | Cleanup uses `safeIdentifierSegment`; other swarm storage commands still interpolate raw IDs |
| P0-02 | Tar/ZIP traversal | **RESOLVED** | Containment and adversarial tests in `pkg/downloader` |
| P0-03 | Release identity fractured | **OPEN** | Source/stable 1.0.48, public GitHub 1.0.43, npm 1.0.22, dev 1.0.41 |
| P0-04 | Fresh host cannot plan | **RESOLVED technically** | Self-provisioning works; host remains a first-use dependency |
| P0-05 | Update destroys GSD settings | **RESOLVED** | Structural merge and smoke proof |
| P0-06 | Daily-driver acceptance absent | **OPEN** | Phase 163 UAT 0/3; Phase 171 not started |
| H-01 | Same-process locks lose mutual exclusion | **OPEN** | Reference count is not goroutine ownership; shared-to-exclusive is not upgraded |
| H-02 | Phase insertion breaks identity | **PARTIAL** | IDs renumber; phase-keyed evidence is not migrated |
| H-03 | Colonize-before-init poisons state | **RESOLVED** | Colonize now refuses to fabricate active colony state |
| H-04 | Reconciled work cannot advance | **RESOLVED** | Verified reconciled claims now count for advancement |
| H-05 | Hardcoded repeated TS preflight | **RESOLVED** | Shared timeout/configuration/cache remains present |
| H-06 | Called Go vulnerabilities | **RESOLVED** | Go 1.26.5 plus exact `govulncheck` pass |
| H-07 | TypeScript host advisories | **OPEN** | Direct `js-yaml` 4.1.1 high advisory; transitive `esbuild` low advisory |
| M-01 | Init non-transactional | **OPEN** | Multiple writes/consumption steps can leave partial active state |
| M-02 | Seal non-transactional | **OPEN** | Learning/Hive/signal side effects occur outside state commit |
| M-03 | Continue packet weaker than build | **OPEN / worse understood** | Review-control fields are unbound and mutable |
| M-04 | Plan-only can write | **OPEN** | Continue plan-only persists evidence/decisions in some modes |
| M-05 | Vital signs misleading/disconnected | **OPEN — REPRODUCED** | Blocked provider/reconcile state was reported Healthy with zero errors |
| M-06 | Full normal/race suites clean | **REGRESSED / ENVIRONMENT-SENSITIVE** | Clean Codex/no-remote snapshot fails 16 tests |
| M-07 | Formatting/catalog gates red | **OPEN** | Two unformatted files; three unclassified commands |
| M-08 | Version/documentation drift | **PARTIAL** | primary docs moved to 1.0.48; GSD and public truth remain inconsistent |
| M-09 | Learning pipeline not wired | **WIRED, SAFETY/PROOF OPEN** | lifecycle callers exist; default-on Hive outran locks, transactions, and UAT |
| M-10 | Worktree/resume/downstream proof absent | **PARTIAL** | Representative pause/resume passed; worktree merge-back and Phase 171 downstream proof remain absent |

Of the six original P0s, two are fully resolved, one is technically resolved with architectural cost, one is only locally narrowed, and two adoption gates remain open. The exact count matters less than the remaining type: the open items are the ones that decide whether a user can trust the product.

## Current release blockers

### 1. Same-process state locking is not mutual exclusion

`pkg/storage/lock.go:54-67` treats another caller in the same process as re-entrant and increments a count without tracking ownership. Two goroutines using one `Store` can therefore enter the same read-modify-write section. Shared-to-exclusive acquisition changes bookkeeping without upgrading the operating-system lock.

Impact: workers can silently lose colony state updates even when the race detector is green.

Required closure: an ownership-correct in-process lock plus a test that concurrently updates one shared store and proves no lost update.

### 2. Continue completion packets are not bound to the emitted manifest

Build finalization has substantially stronger binding. Continue finalization still accepts a permissively unmarshalled packet whose review depth, skip-watcher option, and dispatch list can differ from the plan emitted by `continue --plan-only`. A light/skip/empty-dispatch combination can advance without review.

Impact: returned worker data can weaken the verification policy that the runtime originally requested.

Required closure: persist a canonical continue manifest, bind it with a nonce/digest, reject mutations to control fields, and validate current state inside the atomic update.

### 3. Phase insertion can orphan durable evidence

The phase list is renumbered, but paths such as phase manifests, gate results, attempts, and checkpoints remain keyed to old phase numbers.

Impact: a formerly active phase can move from 2 to 3 while its proof stays under `phase-2`, making recovery or continue treat valid work as absent.

Required closure: either make immutable phase UUIDs authoritative or transactionally migrate every phase-keyed artifact and reference.

### 4. Identifier containment is incomplete

The dangerous `swarm-cleanup` call is fixed. Other commands still build paths such as `swarms/%s/findings.json` and `swarms/%s/display.json` from raw IDs. `Store.resolvePath()` accepts traversal and absolute paths. In an isolated proof, `swarm-findings-init --id ../../escape` reported success and created `.aether/escape/findings.json` outside `.aether/data/swarms`.

Chamber creation is guarded, but verify/compare and tunnel import paths still accept raw path components. Tunnel import can read an out-of-tree archive and import signals.

Required closure: enforce containment centrally in the storage API and validate every identifier at ingress. Patching only destructive call sites is not enough.

### 5. Lifecycle transitions are not transactions

- Init can save active state before later session/recovery/activity writes succeed.
- Direct continue commits phase advancement before fallible reports and housekeeping.
- External continue replaces freshly loaded atomic-update state with an older snapshot.
- Seal promotes learning, mutates Hive/Queen, and expires signals before committing sealed state; it writes the final summary afterward.

Impact: the command can fail after partly succeeding, leaving the user in a state that the next command does not understand.

Required closure: journal or stage lifecycle changes and commit once; fault-injection tests must observe either the complete old state or complete new state.

### 6. TypeScript host dependency security is red

`.aether/ts-host` directly depends on `js-yaml` 4.1.1. Current `npm audit` reports a high-severity quadratic CPU denial-of-service advisory and recommends 4.3.1. A low transitive `esbuild` advisory is also present. The host loads repository YAML, so the YAML advisory is relevant, not merely theoretical. See <https://github.com/advisories/GHSA-52cp-r559-cp3m>.

Required closure: upgrade the lockfile, run the complete host suite, and add host `npm audit` to CI/release gates.

### 7. Release integrity proves labels, not bytes

At audit time:

| Surface | Version/revision |
|---|---|
| Local source and `npm/package.json` | 1.0.48 |
| Installed stable binary/hub | 1.0.48 |
| Public GitHub main/latest release | 1.0.43 at `6577f51` |
| npm `aether-colony@latest` | 1.0.22 |
| Development binary/hub | 1.0.41 |
| Local branch | 456 commits ahead of public main at post-cut review |

The public GitHub latest release is still [v1.0.43](https://github.com/calcosmic/Aether/releases/tag/v1.0.43). `aether integrity` can report freshness from version labels and file counts without proving source commit, binary hash, wrapper hashes, dependency locks, or hub content identity.

Required closure: one signed content manifest must bind version, commit, toolchain, binary, assets, wrappers, and lockfiles. Test the exact artifact through the public installation channel before publication.

### 8. Real user acceptance remains absent

`.planning/phases/163-context-reaches-workers/163-HUMAN-UAT.md` remains 0/3, Phase 164 UAT remains 0/7, and Phase 171 “Prove It” remains not started. `docs/OWNER-TEST-CHECKLIST.md` is a good checklist, but it is not a completed result artifact.

Required closure: run and record the journey rather than handing it to the user as the integration test.

### 9. Cheap-model behavior is still a promise

`colony/policies/model-routing.yaml` has schema tests but no production reader. Phase 161 “Cheap Models By Design” is `0/TBD`. Phase 171 explicitly requires three real tasks on an inexpensive model with operator interventions counted; none are recorded.

This is central to the user's goal. Aether cannot claim its old cheap-model advantage until this proof exists.

The isolated workflow probe reinforces the risk: a tiny discovery phase over-selected six workers, including three Oracles, and exposed no model field; skill matching selected irrelevant domains and allowed one large skill to crowd out three other reported matches.

Required closure: do not begin with elaborate automatic routing. Pick the user's preferred inexpensive model, tighten matching and per-skill budgets, run the three fixed benchmark tasks repeatedly, and make the worker packets simple enough that the model succeeds.

### 10. Primary-platform parity is still contradictory

The OpenCode Scout definition denies write, edit, and shell access and explicitly says not to create documents. Phase 163 requires Scout to create phase research/planning artifacts. Claude has a sanctioned-write hook path; OpenCode does not have the same enforceable contract.

Required closure: either change the OpenCode task contract to return structured findings for a different writer, or give Scout a narrowly enforceable artifact channel. Do not claim parity while the agent definition makes the required task impossible.

### 11. Synthetic continue is not synthetic

In the isolated downstream journey, `aether continue --synthetic` launched a real Watcher, reached the configured provider, received HTTP 401, and blocked. That contradicts the help/contract that synthetic mode skips real workers.

Impact: an offline or deterministic test path can unexpectedly spend provider capacity, depend on credentials, and fail for external reasons.

Required closure: make synthetic mode mechanically incapable of provider dispatch and add a network/provider canary test.

### 12. Health reporting can contradict recovery truth

In the same blocked continue/provider scenario, reconcile reported a warning while vital signs called the colony Healthy and showed zero errors. The ordinary status and wrapper surfaces do not use `colony-vital-signs` consistently.

Impact: Aether can tell the user the colony is healthy at the exact moment its recovery path knows it is blocked.

Required closure: derive health from authoritative blockers, failed attempts, stale activity, gate results, and recovery state; expose the same calculation on every user surface.

## GSD process assessment

GSD helped Aether produce focused plans, tests, reviews, and traceable artifacts. It also allowed the project to confuse phase completion with product completion.

Current control records disagree:

- `STATE.md` says Phase 162 is the focus, Phase 163 is not started, and the milestone is 24% complete;
- `ROADMAP.md` declares Phases 162–165 complete while Phase 161 is not started;
- top-level roadmap checkboxes and the detailed table disagree;
- `REQUIREMENTS.md` leaves requirements pending that phase verification calls complete;
- product-version declarations span old values while source declares 1.0.48;
- Phase 163 is called complete while mandatory human UAT remains 0/3.

This is not cosmetic. The planning files are inputs to both humans and agents. Contradictory inputs cause cheaper models to follow the wrong phase, repeat work, or believe proof exists when it does not.

### The largest strategic miss since the recovery report

The root-cause report explicitly recommended freezing feature expansion and default-on cross-repository learning until locks, transactions, provenance, and downstream proof were sound. Phase 162 instead:

- changed Hive policy to promotion by default;
- removed the per-colony retrieval-consent gate;
- wired additional lifecycle-side learning writes;
- added large ceremony and documentation surfaces.

The learning implementation is thoughtful and well tested in many focused cases. The order is wrong. It places automatic cross-project writes on top of the still-unsound storage and lifecycle foundation.

## The architecture Aether should finish toward

The user's concept is simpler than the current implementation:

```text
User goal
   -> Queen chooses tasks, castes, and waves
      -> native Claude/OpenCode agents do the work
         -> Go kernel validates and commits state
            -> event stream renders ceremony

Skills folder
   -> deterministic match
      -> bounded skill context in each worker packet
```

The ownership rules should be:

| Concern | One owner |
|---|---|
| Intent interview and orchestration | Native Queen/wrapper |
| Worker execution and model choice | Native platform |
| State, locks, paths, manifests, transitions | Small Go kernel |
| Skill discovery/matching/injection | Go service called by the Queen path |
| Ceremony | Read-only projection of committed events |
| TypeScript | Optional narrator/dashboard only, never required for the core lifecycle |

This preserves Aether's identity: rich ceremony, castes, skills, pheromones, Queen orchestration, memory, and networked agents. It removes the need for the user or a cheap model to understand internal host hops, finalizer schemas, and mirrored command authorities.

## Exact finish-line programme for Claude Code

### Work package 0 — Freeze the candidate

Do not add commands, castes, ceremony, update UX, Hive behavior, or new GSD phases until the adoption gate is green.

Deliverables:

- choose one candidate commit;
- reconcile `STATE.md`, `ROADMAP.md`, `REQUIREMENTS.md`, and product version;
- mark mandatory UAT as blocking rather than “carried forward”;
- place every item below on one short blocker board.

Exit condition: agents and the user see one truthful current state.

### Work package 1 — Make the release gate hermetic and green

Fix the small red gates first:

1. pin platform expectations in visual tests;
2. fix no-origin Hive abstraction so it never generates a sanitizer-forbidden placeholder;
3. format the two Go files;
4. classify the three catalog commands;
5. upgrade `js-yaml` and `esbuild` where applicable;
6. add `govulncheck` and both npm audits to CI;
7. make the smoke host gate require the expected result, not merely absence of dependency errors.

Exit condition: normal, race, vet, formatting, catalog, Go vulnerability, TypeScript, npm audit, and smoke gates pass in a clean checkout with and without an `origin`, and from Claude, OpenCode, and Codex environments.

### Work package 2 — Finish the safety kernel

In this order:

1. central storage path containment;
2. all swarm/chamber/tunnel identifier validation;
3. ownership-correct process locking and upgrade semantics;
4. immutable phase identity or full evidence migration;
5. continue manifest binding and stale-state protection;
6. transactional init, continue, and seal with fault injection.

Exit condition: adversarial and concurrent tests prove no escaped path, lost update, orphaned evidence, weakened review, or partial lifecycle state.

### Work package 3 — Collapse the primary vertical slice

Make this the only supported daily-driver path on Claude Code:

```text
install -> init -> plan -> build -> continue -> resume -> seal
```

For every action:

- the wrapper owns interaction;
- Go exposes one deterministic plan/finalize boundary when agents are needed;
- native platform agents execute workers;
- Go is the only persistent writer;
- one failure produces one truthful recovery action;
- the TypeScript host is absent from the mandatory path;
- ceremony reads events after truth is committed.

Exit condition: a user never needs to know about hub paths, host dependencies, completion files, state JSON, or finalizer internals.

### Work package 4 — Prove skills and cheap models together

Use three fixed downstream tasks:

1. a small bug fix in an existing repository;
2. a multi-file feature requiring plan, Builder, Watcher, and tests;
3. an interrupted build resumed in a fresh session.

Run each at least twice with the chosen inexpensive model. For every run record:

- model and cost;
- Queen questions and decisions;
- workers/castes spawned and why;
- skills matched and injected;
- context size;
- files and tests actually produced;
- independent verification result;
- interruption/resume behavior;
- number of user interventions;
- any undocumented retry, manual state edit, or environment repair.

Any repair outside normal user intent is a failed run.

Exit condition: six green runs, with user involvement limited to intent approval and genuine product decisions.

### Work package 5 — Prove both primary platforms

From a clean source-independent environment:

- install into a repo that already contains GSD and unrelated Claude settings;
- complete the full Claude Code lifecycle;
- complete the full OpenCode lifecycle;
- repeat update and verify idempotence;
- exercise worktree mode and recovery;
- prove no source-repository mutation;
- archive transcripts and artifact hashes.

Exit condition: Phase 163 UAT and Phase 171 proof have real PASS evidence, not instructions for the owner.

### Work package 6 — Publish one exact product

Only the candidate that passed Work packages 1–5 may be released.

Bind and verify:

- semantic version;
- Git commit;
- Go toolchain;
- binary hashes;
- companion-asset manifest hash;
- Claude/OpenCode/Codex surface hashes;
- npm package hash/version;
- dependency lock hashes.

Publish GitHub, npm, hub, and binaries from that candidate, then reinstall through the public channel and rerun the golden workflow.

Exit condition: public source, release, npm, stable hub, dev channel, and installed binary no longer tell different stories.

## What Claude Code should not do next

- Do not add another GSD milestone to solve the current milestone.
- Do not reconnect more of the 399 commands because they exist.
- Do not add another host, adapter, schema layer, or mirror.
- Do not treat the owner checklist as completed UAT.
- Do not mark Phase 171 optional or defer cheap-model proof again.
- Do not use a passing targeted package to override a failing clean full gate.
- Do not fix the current 16 tests by blindly updating goldens; first pin the intended platform and retain both Claude and Codex expectations.
- Do not keep Hive promotion default-on while storage and lifecycle transactions are unresolved.
- Do not publish 1.0.48 merely because local version labels agree.
- Do not make the user debug `.aether/data`, host dependencies, or completion packets during acceptance.

## Recommended operating decision today

| Use | Decision |
|---|---|
| Continue repairing Aether in its own repository | **YES** |
| Controlled experiments in disposable downstream repos | **YES, with backups and recorded failures** |
| Use as the only framework for important production work | **NO** |
| Recommend as a low-maintenance cheap-model framework | **NO — not yet proved** |
| Publish the current candidate | **NO** |
| Resume broad feature expansion | **NO** |

## Final conclusion

Aether has not failed because the colony metaphor, ceremony, skills, or Queen are too ambitious. Those are the product's value. The failure mode is that correctness became distributed across too many authorities while real user proof was scheduled after implementation instead of governing it.

The recent repairs show the project is recoverable. Three dangerous original defects are closed, GSD coexistence is genuinely better, skills are wired, normal worker planning is more direct, and the Go vulnerability gate is clean. That is meaningful progress.

The remaining work is not “polish.” It is the minimum trust foundation: contained paths, correct locks, bound manifests, atomic transitions, one primary lifecycle, a green hermetic release gate, real cheap-model runs, real Claude/OpenCode runs, and one release identity.

**The fastest route to “it just works” is now to stop adding, finish that narrow foundation, and make the real downstream journey the authority over every plan, test count, ceremony, and version label.**
