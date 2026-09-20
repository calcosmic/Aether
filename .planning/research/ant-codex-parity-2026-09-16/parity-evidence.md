# Codex `$ant-*` parity: proof, acceptance and validation effort

Research snapshot: 2026-09-16, Aether HEAD `404731ccffda7bbca64ce801b74b0752a7161315`. Read-only repository/receipt review. No tests, model runs, owner journeys, installation, publishing, state changes or worktree changes were performed for this research. This note is the only written artifact.

## Plain-English conclusion

There is substantial reusable testing of Aether's machinery. What is missing is the proof that a person can start that machinery through an installed `$ant-*` skill in a fresh Codex session and get the intended helpers, checked work, recovery and explanations all the way through.

Renaming the skills can be proven cheaply. Claiming the Claude Code behavior works equivalently in Codex needs real Codex runs as well as the existing tests. A test that supplies a pretend helper's answer can prove the program handles that answer correctly; it cannot prove the installed skill actually launches the right helper or that the helper returns the required answer.

Native Codex subagents exist: the parent research has official-documentation and current-session evidence for that. Their existence is separate from whether Aether's installed skills correctly select custom castes, dispatch them, wait for them, collect results, and preserve the results through interruption. None of the reviewed Phase 205 platform receipts establishes that integration.

## 1. What existing proof actually establishes

### Real platform runs are narrow, and older than the current release

All three Phase 205 captures used a disposable copy at `aa67c905`, not the current `404731cc` source or an end-to-end 1.0.79 feature journey.

| Receipt | Observed fact | What it does not establish |
|---|---|---|
| `.planning/phases/205-owner-acceptance-and-restoration-seal/evidence/platform-runs/codex-run.txt:1` | Codex CLI 0.154.0, gpt-6-astra, exit 0; the exact instructed shell command `AETHER_OUTPUT_MODE=visual aether status` ran, reporting no initialized colony. Command/prompt at lines 3 and 21, real shell execution at 26–34. | Skill discovery, `$ant-*` invocation, autonomous command selection, planning, builders, reviewers, native helper tools, completion packets, recovery, or an in-progress project. The card's “chose on its own” wording is stronger than the raw prompt: the prompt already specified the exact shell command. |
| `…/evidence/platform-runs/claude-run.txt:1` | Claude Code 2.1.272 ran `/ant-status`, exit 0; empty-project status and a plain-English next action at lines 8–18. | A paired build/check comparison or a complete Claude lifecycle to use as an empirical parity reference. |
| `…/evidence/platform-runs/opencode-run.txt:1` | OpenCode 1.1.63 launched its configured model, then printed a connection error at line 11. The captured process exit was 0. | It never reached Aether status. This is also an example of why an exit code alone is insufficient acceptance evidence. |

The corresponding cards explicitly limit their claims: `205-PLATFORM-CARD-CODEX.md:17`, `205-PLATFORM-CARD-CLAUDE.md:17`, and `205-PLATFORM-CARD-OPENCODE.md:15`. `205-10-SUMMARY.md:117` is especially clear: these were user-facing platform invocations, not Aether launching platforms as build workers; only the command surface was exercised. Its historical no-menu-command wording is not a current test of native Codex skill or helper capability.

### Existing automated tests are useful but work at different levels

| Existing test / evidence | What it really proves | Limit for the proposed claim |
|---|---|---|
| `cmd/parity_test.go:123` (`TestPlatformParityGolden`) and `:257` (`TestAllYamlHaveWrappersAndGuide`) | YAML, Claude/OpenCode wrappers, guide and runtime command inventories are consistent with the checked snapshot. | The snapshot has no installed Codex skill inventory or live invocation. `TestCodexCoverageByDesign` at `:608` only requires a non-empty guide catalog. |
| `cmd/command_guide_test.go:1027` | Nine generated `aether-*` command skills exist; trigger metadata, guide command, raw bypass, source skill and runtime-command strings are present; frontmatter parses. | It inspects generated text. No fresh Codex session discovers or invokes a skill. Its required nine-name list needs explicit expansion for the new public surface. |
| `cmd/command_guide_test.go:1156` (`TestCommandGuideBuildSmoke`) | Build guide has nonempty run command, pre-steps and post-steps. | Despite “Smoke” in the name, nothing builds or spawns. The coherent-job contract checks at `:1474` likewise inspect required/forbidden strings. |
| `cmd/codex_e2e_test.go:334` (`TestCodexE2EFullLifecycle`) | Real Aether install → setup → update commands run against temporary homes/repos and place or omit agent files as intended. | “Full lifecycle” here means installation of agent files, not init → plan → build → verify using Codex. |
| `cmd/codex_e2e_test.go:596` | Install prunes a retired full-skill mirror and creates each generated shim on disk (`:651–662`). | It asserts file presence, not Codex discovery or behavior. It does not prove transactional update migrates renamed command skills. |
| `pkg/codex/worker_test.go:411` | The test executable intentionally defaults to `FakeInvoker`. | Passing most runtime workflow tests is not a live provider test. Even “RealWhenEnvSet” at `:420` points `AETHER_CODEX_PATH` at `go`; it proves selection/probing, not inference. |
| `pkg/codex/worker_test.go:615` (`TestRealInvoker_Invoke_UsesAgentPromptAndFinalMessageFile`) | The real subprocess adapter sends a composed prompt/schema, handles a final-message file and parses a structured result. | `:633–654` writes `fake-codex.sh`, which emits a fixed answer. This is good adapter-contract coverage, not an actual Codex helper run or built-in `spawn_agent` integration. |
| `cmd/e2e_lifecycle_test.go:21` (`TestFullLifecycleInDownstreamRepo`) | A real temporary Git project passes through Aether's deterministic lifecycle plumbing. | Comment at `:25` states FakeInvoker; accepted plan is seeded at `:250`; build uses `--synthetic` at `:269`; continue uses `--skip-watchers --light` at `:300`. It does not exercise the live skill-led conversation or independent real Watcher. |
| `cmd/codex_build_native_partial_test.go:19`, `:111`, `:136` | A substituted invoker writes actual fixture files and partial receipts; real runtime code journals partial completion and exposes recovery. | “Native” means Aether's direct runtime lane in this test. It is not Codex's built-in native subagent API. |
| `cmd/codex_build_worktree_test.go:19`, `:146` | Actual temporary Git worktrees and sync behavior are tested with a substituted worker that writes a fixed file (`:43–48`). | Native skill-driven helper cwd, allocation, result return and interruption are still unproven. |
| `cmd/codex_build_finalize_test.go:1581` | A rejected completion packet leaves attempt status/digest and colony completion unchanged. Many surrounding tests check stale/malformed/duplicate identity and receipts. | Strong reusable safety coverage; still needs a correctly produced packet from a real native helper to prove the bridge. |
| `cmd/recruitment_test.go:24` (`TestRecruitmentTracerEndToEnd`) | The actual recruit command admits a child, launches a real subprocess, waits and binds a result in a temp store. | It launches the Go test executable (`:66–67`); its child only prints `recruited-child-ok` and returns (`:35–43`). No real Codex launch, useful recruited output, or native parent/child handoff is proven. |
| `cmd/recruitment_probe_test.go:270` (`TestEveryDeclaredAdapterCanDispatch`) | AST inspection finds switch branches for the declared adapter kinds. | It does not execute those adapters. The separate `:343` dispatch test again substitutes the test executable (`:349–350`). |
| `cmd/wrapper_path_parity_test.go:322` | Direct and wrapper closeout renderers produce the same strings from a shared typed fixture. | Does not prove a live assistant relays that output, narrates actual helper progress or stops at the correct checkpoint. |
| `.aether/ts-host/test/cross-platform-parity.test.ts:115`, `:138` | Codex agent files contain fields; source text declares the Go adapter as provider-launch owner. | Regex/source inspection, not a Codex app or native helper test. |
| `cmd/platform_honesty_card_test.go:202` | Every claim's quoted text exists on its cited transcript line; required cards/runs exist (`:291`). | It does not rerun a platform or determine whether a sentence's inference is warranted by the quoted line. |
| `cmd/classic_parity_record_test.go:324`, `:379` | Every signed restoration slice has the required dimensions, a public path or an explicit lack-of-evidence reason. | A public path resolves only against Cobra or a Claude slash-command file (`:366–376`); `$ant-*` is not supported by this validator. It does not verify evidence semantics. |

A targeted text search of `cmd`, `pkg`, and `.aether/ts-host` test files found no `spawn_agent`, `close_agent`, `wait_agent`, or `collaboration.spawn` reference. This is a bounded observation, not a claim that no tests exist anywhere. It agrees with the explicit fake/provider boundaries above.

### The restoration ledger is not a blanket Codex-parity certificate

`205-PARITY.md:13–19` expressly says a structural test/count does not prove the truth of any evidence. `205-09-SUMMARY.md:83–88` assigns semantic evidence quality to human/independent judgment.

The current ledger also carries real limits:

- Phase 200 has no owner walkthrough (`205-PARITY.md:121–129`); planning legibility remains unevidenced (`:149`).
- Phase 201's five owner-confirmed checks are identified (`:154–161`), but several surfaces remain unevidenced; the repair announcement on the chat-driven checking path was specifically outside scope (`:189`). These are not identified as native Codex skill journeys.
- Phase 202's three owner walkthroughs were skipped; no owner watched the live Swarm/Oracle/cleanup shapes (`:214–222`).
- Phase 204 names zero production callers for derived changelog and outcome views (`:360–372`).

The practical implication is to create new, scoped Codex skill evidence and preserve the historical receipts. Do not relabel the old captures or aggregate restoration ledger as proof of new native behavior.

## 2. Baseline failures that matter to a new acceptance claim

The complete gate at `8391fef9cfc7ba122d8834d83009c4006ed5af3e` is reusable baseline evidence, not all-green proof. `205-12-SUMMARY.md:155–190` records build/vet passing, normal/race raw exit **1**, each cmd run discovering/executing **5,573 tests across 59 groups**, all **18 other tested packages passing**, no unexpected top-level failures, no race warnings or timeouts. Exactly **17** known top-level names failed. `TestGateProbe` is an intentional failing child inside a passing enclosing test, not an eighteenth baseline failure.

`205-13-SUMMARY.md:138–142` records identical objects/modes for 1,517 runtime/dependency files between that tested revision and the metadata/docs release. It records 1.0.79 installed, with 14 existing shims unchanged and twenty user-level Codex agent variants preserved (`:160`, `:191–195`). That preservation is useful release evidence, but also means a real-run receipt must identify which installed agent definition was actually used.

The 17 baseline names are in `205-12-SUMMARY.md:170–188`. Most relevant intersections with this new work:

| Baseline group | Observed failure in the existing raw log | Consequence for acceptance |
|---|---|---|
| `TestCompletionPacketSchemaMatchesStructs` | Committed completion schema differs from current Go structs. `…/repaired-gate/full.log:15916–15918`; test intent at `cmd/contract_schema_test.go:15`. | New skill-to-finalizer packets need a current, tested schema. Existing red schema synchronization cannot certify the new bridge. |
| `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount` | Fixture expected concrete worker count above caste ceiling; both were 8. Log `:21518–21521`; assertion setup `cmd/codex_build_test.go:1338`. | Does not establish that production spawning is broken, but this red fixture cannot be claimed as proof of native parallel-helper budgeting. |
| Four fix-attempt tests: `TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestFixAttemptIsCountedSeparately`, `TestFixAttemptNeverOverwritesTheFirstResult`, `TestNoSecondAutomaticFixAttempt` | Missing/separate attempt records or unexpected repeat repair. Log `:696–735`. | Real failed-check → bounded repair → persisted result needs direct proof. Do not call this harmless naming drift without diagnosis. |
| `TestResolveTestCommand_GoProject` | Test-command extraction includes prose after `go test ./...`. Log `:852–854`. | Verify the new journey actually executes the intended tests, not just a malformed command or an empty check. |
| `TestGoSourceHintsMatchCobraContracts` | User hint names nonexistent `aether interpret`. Log `:14779–14783`. | New `$ant-*` next actions must resolve to actual skills/runtime mappings, including recovery paths. |
| Display/audit/vocabulary goldens and `TestHumanFacingOutputGoesThroughWriteVisualOutput` | Changed explanation/icon text, inventory drift or output-writer violations. Examples log `:159–169`, `:8078–8081`, `:12284–12287`. | Review intended `$ant-*` wording changes; update expected evidence deliberately. Blind golden refreshes can hide behavioral drift. |
| `TestPlanningPublicPaths200`, `TestPlanningAdversarial200` | These aggregate failures include nested vocabulary inventory drift; raw log `:10165–10169`, `:12530–12533`. | A failing aggregate is not proof all planning safety assertions fail. Record exact relevant subcase outcomes, and establish the native planning/acceptance path independently. |
| `TestBuildStartLegacyHelpersRetired200`, `TestPhase199GateReceipt` | Source guard finds test helper writes; protected ownership fingerprint/receipt stale. Log `:4181–4184`, `:180–182`. | Distinguish verifier maintenance from a product defect. Neither should be used as a green assurance. |

Here `…/repaired-gate/full.log` means `/Users/callumcowie/.aether-backups/phase205-execution-evidence-20260915/repaired-gate/full.log`.

No need to repair every old defect merely to expose a skill name. Tests directly affected by the promised behavior should become meaningful passing tests. Any retained unrelated baseline failures must remain named and unchanged; the new work should not enlarge an allowed-failure set to make a parity claim.

Some current tests intentionally forbid the requested product surface: `cmd/maintenance_wrapper_contract_199_test.go:104`, `:161`, `:283` rejects a Codex `$ant-*` surface; `cmd/command_guide_test.go:1249` forbids `$ant-` in the plan guide. These are stale scope contracts to revise alongside the new feature, preserving their underlying authority/safety assertions. Do not simply delete the whole tests.

## 3. Practical acceptance matrix

Compare outcome, mechanism, visible explanation and safety for each family. Exact model prose or identical model reasoning is not required. A real helper ID, actual file diff and durable result are better evidence than a helper emoji, text saying “spawned”, or a total pass count.

| Acceptance area | Cheapest meaningful proof | Completion bar / negative control |
|---|---|---|
| Canonical public names and mappings | Deterministic matrix for all agreed public Claude names (surface research found 64), generated `ant-*` frontmatter and explicit runtime/host mappings. | Each public name has one intended entry; nontrivial aliases and prompt-only commands are mapped explicitly. Inject a missing/incorrect mapping and show the check fails. |
| Installation, upgrade and rollback | Temporary package/home/consumer fixtures: clean install, existing aether-prefixed installation, current transaction-based `update --force`, second idempotent run, failed update rollback. | `$ant-*` appears through both fresh install and the supported upgrade path; retired managed names are handled predictably; custom skills and customized agents remain intact; rollback restores exact prior bytes. The surface researcher established current update does not regenerate skills, so install-only coverage is insufficient. |
| Fresh-session skill discovery and dispatch | Start a fresh actual Codex session against the candidate install, inspect its skill list/picker where available, invoke `$ant-status` and `$ant-help` by name. | Trace shows intended installed SKILL loaded and real command executed without an exact shell command embedded in the prompt. Record source/version and installation paths; test a second fresh session after upgrade. |
| Parameters and read-only behavior | Real `$ant-status`, then one `$ant-focus` containing spaces/quotes on a disposable colony; deterministic argument tests cover the complete catalog. | Status changes no scoped state; the focus content stored once matches the user input. No accidental argument loss or shell interpretation. |
| Init/discuss/spec/plan | Small fresh fixture, one actual material choice with predeclared test responses, real Scout/Route-Setter stages, real accepted plan. Compare equivalent Claude and Codex prompts. | No fabricated owner decision; exact accepted candidate identity is stored; a refused/stale candidate does not activate work. Reuse existing Go acceptance tests, and capture the live skill orchestration. |
| Build and independent verification | Tiny fixture with one real Builder and an actual independent Watcher. Add a two-disjoint-task case for parallelism. Start with `$ant-build`, not a manually scripted finalizer call. | Correct custom castes, assigned roots and briefs, actual native helper IDs/tool events, real edits, useful handoffs, correctly issued/completed task identities, runtime-granted completion credit. No fake provider, `--synthetic`, skipped watcher or parent doing the promised helper's work in the acceptance run. |
| Continue, failed check and bounded repair | Run `$ant-continue` on passing work, then clone the fixture and intentionally make one check fail. | Actual test command/result recorded; failure blocks advancement; repair gets the proper separate attempt, verified result and limit; another invocation cannot silently repeat an already-spent automatic repair. This intersects known-red tests. |
| Recruitment and helper feedback | One actual Aether-mediated helper request that produces a small useful answer/file; one admission refusal. | Child uses the chosen Codex mechanism; parent receives usable result/handoff, linked identity and terminal state, not just process exit. Refusal is visible and parent can proceed. Test the production launch path, not `AETHER_RECRUIT_BINARY` substitution. |
| Signals and learned context | One distinctive FOCUS/REDIRECT and one prior handoff/lesson in the tiny fixture; trace the selected helper's assembled brief and resulting work. | Relevant source context arrives once, revoked/expired context is absent, and outcome records distinguish delivered context from proven useful effect. Reuse resolver/context/credit unit tests. |
| Pause, interruption and resume | Stop after one helper has returned but before overall finalization; resume in a fresh Codex session. | Finished work/result survives; unfinished work is accurately identified; no duplicate completion, task credit or replayed writes. Need evidence that per-helper results are durable before the single final completion submission. |
| Worktrees, if parity includes the advertised mode | Two helpers in actual disposable worktrees, then one conflict or interrupted helper. | Each native helper works in its assigned cwd; accepted edits reconcile once; failed/overlapping edits are retained/refused honestly. Existing direct-lane worktree tests are reusable but do not establish native skill cwd/allocation. |
| Seal/retained review/archive | Completed small fixture and separate incomplete fixture. | Correct successful seal; unresolved work refuses normal seal; status shows retained sealed state. Explicit archive, if exercised, verifies before clearing. New Codex evidence does not complete the unrelated Phase 205 owner acceptance. |
| Research, bug-hunt and other public families | At least one bounded real `$ant-oracle` and `$ant-swarm` case, plus deterministic path checks for every public name and individual live cases for distinct orchestration families. | Correct real research/diagnostic helper paths, bounded stop, evidence/next action and no unsupported shell spelling. One build tracer cannot certify all 64 behaviors. |
| Progress, costs, unavailable capability | Capture live tool events and emitted Aether status alongside failure/timeout run; preserve only actually available usage/cost fields. | Visible activity corresponds to real running/finished helpers, unavailable service is a truthful failure/refusal, no silent simulation/fallback. Missing native usage is recorded as missing, never a fabricated cost or zero. |

For noninteractive acceptance, test prompts can preauthorize routine fixture choices. A genuinely unanswered material product choice should remain pending; a harness reply must be identified as a test response. This need not involve the owner or the prepared acceptance project.

## 4. Cheapest useful end-to-end tracer

1. Create a disposable Git project containing a dependency-free tiny library, two independent change requests and fast real tests. Use a source-pinned candidate installation with isolated Aether state. Record Codex/Claude versions, model, custom agent definitions and skill hashes. Do not use the prepared owner project.
2. In a fresh Codex session, invoke the installed `$ant-status` skill by name. This catches wrong frontmatter/discovery paths quickly without paying for a full plan.
3. For the first helper-bridge test, use a transparently fixture-prepared accepted plan, created through the real runtime acceptance path. Invoke only `$ant-build 1` with the agreed goal/authorization and let the skill do the orchestration. Require one real Builder result and an actual Watcher/check result. Save native events, before/after files and runtime receipt identities. This isolates the uncertain native bridge from the cost of model-driven planning; it does **not** count as live planning proof.
4. Invoke `$ant-continue`, inspect the actual test exit and durable advancement, then read status. The useful end-to-end assertion is that a requested change exists, its checks ran, and the program credited the correct work once.
5. Repeat on a cloned fixture with a deliberately failing test and verify the failure/repair boundary. Stop/reopen one run between helper completion and finalization to test loss/duplication.
6. Run the equivalent minimal Claude journey from the same fixture baseline. Compare semantic results, prompt/approval boundaries, helper topology and durable records. Capture any platform-specific difference explicitly.
7. Only after the bridge works, add the unseeded init → discuss/spec → plan journey and the remaining distinct command families. Reuse one generated mapping test for all public command names; do not spend one model call per alias merely to verify spelling.

A successful initial tracer justifies “the installed Codex skill can complete this native build/check path.” It does not justify “all of Aether now behaves identically to Claude Code.”

## 5. Verification stages and effort

These are **total proof-work estimates**, not extra days to add blindly to another researcher's already test-inclusive implementation estimate. “Proof implementation” below means writing/updating test fixtures, harnesses, assertions and evidence capture. Product changes to skill generation, native orchestration, update migration, recruitment, recovery or cost collection are separately owned by the implementation estimate.

| Stage | Work and exit condition | Proof implementation / triage | Execution and review |
|---|---|---:|---:|
| A. Contract and regression inventory | Canonical skill/runtime mapping, stale no-`$ant` assertions revised, exact old failure baseline retained. | 0.25–0.5 day | 0.1–0.25 day |
| B. Deterministic packaging + runtime contracts | Install/update/rollback/custom preservation, mappings, packet/identity/context regressions; relevant unit and integration checks pass. | 0.5–1 day | 0.25–0.5 day |
| C. Real core comparison | Fresh discovery, real native build/check, Claude reference, failure and resume tracers with readable receipts. | 0.25–0.5 day | 0.5–1 day |
| D. Candidate release rehearsal | Candidate install/update in disposable consumers; rerun fresh discovery after upgrade; hashes/rollback; appropriate final gates. | 0.1–0.25 day | 0.25–0.5 day |
| E. Full advertised behavior breadth | Remaining independent families, recruitment result, both execution modes, repeated interrupted/recovery scenarios, authority and usage checks. | Additional 0.5–1 day | Additional 1–2 days |

Rounded estimates, allowing normal overlap: **2–4 working days total proof effort for the core skill/native lifecycle claim**; **4–7 working days for a defensible broad 64-command behavior-equivalence claim**, excluding product defect repair. Roughly 1–2 days of the core total is test/harness implementation and 1–2 days is running/reviewing/releasing the evidence. This is not a statistical estimate; confidence is moderate for deterministic checks, lower for live native recovery/recruitment because there is no recorded end-to-end baseline.

If another estimate already includes test/harness coding, count only the execution/review portion additionally. If it includes packaging tests, live tests and release rehearsal, do not add these ranges at all; use them to check that those activities were budgeted. A naming-only milestone needs a much smaller subset (mapping/install/update/fresh discovery) but earns only a naming/discovery claim.

### Measured machine time and model cost

- The saved gate took **1,373.3 s normal (22.9 min)** and **1,467.2 s race (24.5 min)**, approximately **47.3 min combined**, plus 6.7 s build and 3.4 s vet. Source: `/Users/callumcowie/.aether-backups/phase205-execution-evidence-20260915/repaired-gate/results.json:43–54` and `:88–100`. Both test gates remained **baseline-relative failures**, raw exit 1. This does not include analysis of failures or new product fixes.
- No need to rerun that gate during this research. After runtime/test changes, run focused changed-area checks first and the appropriate full normal/race gate once for the actual candidate. Compare discovered/executed counts and exact failure names; zero unexpected failures is not the same phrase as all-green.
- The Phase 205 platform-card work took 19 minutes total (`205-10-SUMMARY.md:103`), but it only ran three status entry points. The prior local release/backup/update process took 27 minutes (`205-13-SUMMARY.md:88`, `:100`). These are lower-bound reference tasks, not timings for full behavior parity.
- The real Codex status capture alone reports **31,976 tokens** (`codex-run.txt:47–48`) at high reasoning/context settings. One status sample is not a trustworthy lifecycle token forecast or dollar estimate. Budget a bounded number of real model journeys and record actual elapsed time/tokens/cost when present. A low-single-digit handful of runs proves the initial bridge; wider acceptance is likely roughly 8–12 bounded scenario sessions across both platforms plus targeted retries, with the exact count set by distinct behavior families, not 64 names.

Main schedule uncertainty is not test-suite execution time. It is whether the first real skill-to-native-helper run exposes mismatches in completion schema, authority, worktree routing, helper result return, or interrupted recovery. Those are concrete unproven boundaries identified by this review and the parallel runtime research, not reasons to defer the requested naming or demand a different surface.

## 6. Preserve the current acceptance boundary

`205-14-CHECKPOINT.md:7–13` says only preflight task 1 of 3 is complete and the owner session has not begun. `:46–60` reserves the ten journeys and personal verdicts to the owner and forbids declaring that plan complete in advance. The new Codex work can be developed and proven in disposable fixtures without touching this project or treating its pending session as a prerequisite for every implementation task. Its new engineering receipts cannot replace that separate owner acceptance.

Recommended wording after a future scoped pass: “Canonical `$ant-*` skills are installed and upgrade correctly; the named Codex journeys passed with real native helpers on the recorded versions. The listed untested families and pre-existing baseline failures remain.” Expand that statement only as additional actual evidence arrives.
