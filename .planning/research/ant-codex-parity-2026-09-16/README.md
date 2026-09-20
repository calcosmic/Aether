# `$ant-*` skills and Codex behavior parity

Research date: 2026-09-16. Source snapshot: `404731ccffda7bbca64ce801b74b0752a7161315`.

Status: research imported into GSD. On 2026-09-16 the owner requested GSD integration before remaining Phase 205 acceptance; [Phases 204.1-204.5 record that scope and order](GSD-INTEGRATION.md). Phase 204.1 now has seven independently checked plans across five waves, ready to execute. Later phases still require detailed planning. No implementation, installation, publication, or new acceptance run is recorded here.

## Plain-English conclusion

The requested names are feasible: `$ant-init`, `$ant-plan`, `$ant-build`, `$ant-continue`, and the other `ant` actions. These should become the canonical public Codex skills. The executable underneath can remain `aether`.

The naming change is a few days of work when installation and upgrades are included. Matching the main Claude Code workflow is approximately three weeks of engineering effort. Broad, tested parity across all 64 public actions, including working helper recruitment and interrupted-run recovery, is approximately six to eight weeks total for one engineer familiar with Aether. These are planning judgments, not measured delivery promises or forecasts of AI session elapsed time.

Most of the shared engine already exists. The main work is making Codex's instructions use that engine correctly, connecting real native helpers to its saved task records, and checking that interrupted work can resume safely.

## Scope and method

The owner explicitly requested three parallel research areas. Native Codex researchers inspected naming/install coverage, workflow behavior, and validation evidence. A parent review checked current official Codex capabilities and reconciled overlapping estimates.

“Parity” here means equivalent accepted intent, owner decisions, useful helper work, verification, saved results, recovery, and completion rules. It does not promise identical host interfaces or identical model-written answers. Missing native usage information must be reported as unavailable; no estimate assumes a provider can supply fields it does not expose.

The request is interpreted as parity with Claude Code, with OpenCode command coverage as a second reference. The public naming requirement is `$ant-*`, without a default duplicate `$aether-*` command menu. Existing internal helper skills can become private supporting instructions.

Supporting reports:

- [Naming, coverage, installation and upgrades](skill-surface.md)
- [Workflow behavior and runtime integration](workflow-parity.md)
- [Existing evidence and acceptance plan](parity-evidence.md)
- [Current Codex capabilities and assumptions](host-capabilities.md)

## Findings that drive the estimate

### 1. There are nine public Codex skills, compared with 64 commands elsewhere

Claude Code, OpenCode, and the YAML inventory each contain the same 64 command names. Codex currently generates nine public workflow skills and five internal helpers. The separate 86 worker skills are a different inventory and do not provide the missing user entrypoints.

Most additional entries can share a generator. Six need explicit mappings to differently named runtime operations; five contain prompt workflows without a same-named executable command. Generating `aether <name>` blindly would produce invalid behavior. A single tested inventory should identify each entry's actual route.

### 2. Installation needs more than changing a prefix

Publishing/installing generates the existing skills, but the normal transactional update path does not currently regenerate them. A complete change must cover fresh install, existing installs, repeated updates, failed-update rollback, custom-name collisions, generated project instructions, and user-facing hints.

Remove only legacy files that can be identified as Aether-owned. Preserve custom skills and edited files. Current official discovery documentation uses `.agents/skills`, while this session does discover the installed legacy `.codex/skills/aether` location. Test supported fresh clients before deciding whether a discovery-root migration is needed.

### 3. Codex instructions lag behind the shared runtime

The Go runtime already supports accepted plans, worker manifests, finalization, ordinary verification, pause/resume, Oracle limits, and retained sealed state. Specific Codex instructions still need alignment:

- Follow the current init → discuss/spec → plan approval sequence.
- Pass the full context capsule to workers, including owner steering and prior handoffs.
- Handle worker questions and owner decisions through the current runtime routes.
- Honor the approved Oracle brief and pending seal-confirmation branches.
- Remove stale playbook instructions that contradict current runtime assignments.

These are concrete gaps, rather than reasons to rewrite the engine. The detailed workflow report contains file and line references pinned to the research snapshot.

### 4. Native helper coordination is the largest integration risk

Codex itself can start native research helpers; this investigation used three. Aether still needs a complete, proven connection between a native helper and the runtime's assignment, attempt identity, permissions, result record, and finalization. Results must survive interruption without duplicated work or completion credit. Exactly one component must own launching each helper.

Three additional issues affect shared native wrapper paths, including Claude/OpenCode, and must be labelled as shared repairs:

- Recruitment does not yet establish a complete production-provider launch and useful result handback through the inspected path.
- The direct Go path has workspace allocation, but the native wrapper path needs equivalent allocation and working-directory binding for isolated worktrees.
- The external swarm finalizer lacks the checkpoint/rollback behavior of the direct execution path.

Merely copying existing wrapper instructions would inherit these issues. They are included in the stronger end-to-end estimate, not presented as exclusively Codex defects.

### 5. Current tests do not establish installed skill parity

There is substantial lower-level coverage, but some lifecycle tests use synthetic plans, fake providers, or skipped reviewers. The recorded live Codex receipt proves an explicitly requested `aether status` invocation in an empty project; it does not prove skill discovery or real native build/check/recovery behavior.

The previous full normal/race gate took about 47 minutes and retained 17 known failures, with no new failures or races reported in its saved comparison. Some failures overlap the proposed work. A future candidate must distinguish unchanged baseline failures from defects it introduces or claims to fix. Research did not rerun the gate.

## Reconciled effort estimate

All rows are cumulative alternatives, not amounts to add together. One engineering day means approximately six focused hours; one week means five such days. Calendar time depends on provider availability, review, and other work.

| Deliverable | Central planning estimate | Scope |
|---|---|---|
| Existing skills become `$ant-*` | A few working days; likely about 4 | Nine public names, private helper cleanup, safe install/update migration, instructions, focused checks |
| Main workflow parity | About 3 weeks | Naming work plus current lifecycle decisions, full worker context, native result/recovery integration, representative live proof |
| All 64 names and their workflow behavior | About 4–6 weeks | Core work plus complete routes, prompt workflow adaptations, and distinct command-family checks; shared native limitations remain explicitly recorded |
| Broad dependable parity | About 6–8 weeks | Previous scope plus recruitment handback, native worktree coordination, swarm recovery, and broad real-host validation |

The naming-only estimate discussed before this investigation concerned entrypoints. It should not be used as an estimate for full behavioral parity.

### Arithmetic and uncertainty

The researchers' totals overlap. The workflow estimate already includes integration and proof. The evidence report's 2–4 core proof days or 4–7 broad proof days must **not** be added again wholesale. Installation checks are already in the surface estimate. Prompt-body adaptation overlaps the extra command-family behavior estimate.

The following low/central/high scenarios expose the calculation; they are not statistical confidence intervals:

| Cumulative scope | Focused hours, low / central / high |
|---|---:|
| Nine-name migration plus private helper cleanup | 13 / 24 / 42 |
| Above plus core behavior | 49 / 96 / 162 |
| Full 64-command surface plus core and additional command-family behavior | 77 / 150 / 252 |
| Above plus shared native-path repairs | 113 / 210 / 354 |

The all-command calculation uses surface totals of 26/48/80 hours, helper cleanup of 3/6/12 hours, core behavior of 36/72/120 hours, extra behavior of 18/36/60 hours, and provisionally deducts 6/12/20 hours of overlapping prompt adaptation. Shared repairs add 36/60/102 hours. A detailed implementation plan should validate that overlap before scheduling.

At the high end, broad parity can reach approximately 12 engineering weeks. The first real native build/recovery experiment should narrow that uncertainty early. A discovery-root migration may require another 4–16 hours if fresh-client testing shows it is necessary; it partly overlaps the installation contingency and is not automatically additive.

Deep auditing of every optional command, all delivery backends, and complete host-hook equivalence were outside this research. An unqualified promise that every platform feature behaves identically would require a wider inventory and a new estimate. Native cost attribution also remains conditional on information the host actually provides.

## Recommended implementation sequence

1. **Ship the naming foundation.** Create canonical `$ant-*` skills from a route inventory; make installs and updates preserve that result; refresh generated instructions. Verify discovery in a fresh session.
2. **Prove one real build and check early.** In a disposable project, start with `$ant-build`, launch an actual Builder and independent Watcher, save both results, finalize, and advance. Use a transparently prepared accepted plan to isolate this bridge first.
3. **Align the main lifecycle and recovery.** Add current intent/approval behavior, worker context and answers, Oracle/seal branches, failure handling, and interruption/resume proof. Then test planning without a prepared plan.
4. **Expand to the full command inventory.** Adapt prompt workflows and explicit aliases. Share the runtime and instructions rather than maintaining another independent implementation.
5. **Close shared integration gaps and qualify the candidate.** Exercise recruitment handback, both workspace modes, swarm rollback, upgrade/rollback, and a bounded Claude comparison. Run the applicable final regression gates against the actual candidate.

The first live experiments should be small and record real tool events, edits, test results, task identities, and retained state. A generated file count or a helper emoji does not establish behavior parity. The full [acceptance matrix](parity-evidence.md#3-practical-acceptance-matrix) gives the completion bar for each family.

Most implementation can proceed in Codex. Live comparison against Claude Code requires its availability. Keep development and evidence in disposable projects; the prepared Phase 205 owner walkthrough and its original before-inventory remain separate and unchanged.

## Official Codex references

Explicit dollar-prefixed skills and native subagents are supported mechanisms for this design. See [Build skills](https://learn.chatgpt.com/docs/build-skills) and [Subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents). The repository's blanket statement that Codex has no slash commands is stale; see [Slash commands](https://learn.chatgpt.com/docs/reference/slash-commands). The owner's requested surface remains `$ant-*`.
