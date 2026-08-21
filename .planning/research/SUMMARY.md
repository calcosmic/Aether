# Project Research Summary

**Project:** Aether — milestone v1.26 "Intelligent Orchestration"
**Domain:** Go CLI multi-agent orchestration runtime, subsequent milestone on a mature codebase
**Researched:** 2026-08-08
**Confidence:** HIGH on what exists in this repo (verified by reading and executing code); HIGH on Claude Code platform behaviour (official docs); MEDIUM on OpenCode nesting and cross-framework failure data; LOW-MEDIUM on roster-size guidance

Detailed reports: [STACK.md](STACK.md) · [FEATURES.md](FEATURES.md) · [ARCHITECTURE.md](ARCHITECTURE.md) · [PITFALLS.md](PITFALLS.md)

---

## Executive Summary

**Most of v1.26 is not new construction. It is reclaiming machinery that was already built, already documented, and never wired to a caller.** All four researchers reached this independently, from four different angles, and each verified it against the source rather than inferring it. A complete recursive-delegation policy engine exists in `.aether/ts-host/src/spawn-orchestrator.ts` — depth cap, budget, fail-closed rejection — sitting on a path the interactive build wrapper is explicitly forbidden to call. The depth guard `spawn-can-spawn` accepts a `--depth` flag, ignores it, and has returned `can_spawn: true` for every input it has ever received. `.aether/workers.md` tells every worker to pass `--enforce`, a flag that was never registered, so the documented invocation exits 1. The same file hardcodes `--depth 0` for every child, which means `spawn-tree-depth` is structurally incapable of returning anything but 0. `colony/agents/*.yaml` holds 27 caste definitions with zero Go and zero TypeScript readers. Eight of nine skill-lifecycle commands have no reference anywhere outside their own definition file. `WorkerUsage` is parsed at every dispatch and read by nothing, and the wrapper path — the one users actually run — has no usage field at all. The caste-selection rationale strings are computed on every build, carried all the way into the JSON dispatch manifest, and never shown to a human.

This is the project's documented signature failure, in progress, inside the exact feature area v1.26 proposes to build on top of. The correct framing for the roadmap is therefore **"switch on and prove," not "design and build."** Two consequences follow. First, the ordering is load-bearing: a wiring ratchet must land as the *first* phase, because a ratchet written after the capabilities will be shaped to whatever shipped. Second, every requirement must name a command that fails when the requirement is unmet — the project's own Definition of Done — because the alternative has been tried here repeatedly and produces green milestones over dead code.

Three findings change the shape of the work rather than just its sequencing. **Recursive delegation is now a native platform feature**: Claude Code defaults to three layers below the main conversation, configurable via `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`, and withholds the `Agent` tool at the limit so the agent degrades gracefully instead of erroring. Aether should build the *governor*, not the mechanism. **Depth is not a budget**: a tree 2 deep and 8 wide is 72 workers, and Claude Code's own default went 5 → 1 → 3 across three releases — a public record of a team retreating from a deep default twice. Every serious system in the survey pairs depth with a second, independent bound; Aether has a per-wave width cap and no run-total bound at all. **The `PreToolUse` hook is the only deterministic gate**: it fires before the tool executes, can deny it, carries `agent_id` and `agent_type` inside a subagent, and Aether already ships `aether hook-pre-tool-use` wired to `Write|Edit`. Adding an `Agent|Task` matcher gives Go an LLM-proof veto over every recursive spawn. Prompt-level rules have been tried here and are in the audit record as the thing that did not work.

**A worked example of why measurement requirements must specify arithmetic, not intent:** during this research a token-accounting bug was found and fixed. `usageFromFields` computed `TotalTokens = InputTokens + OutputTokens`, never parsed `cache_creation_input_tokens` at all, and treated Anthropic's three disjoint input counts as if one of them were the total. On a realistic cache-heavy run it reported 550 tokens against 102,550 actually processed — a 186x undercount — while `USDCost` was read correctly, so the first spend dashboard would have shown an accurate price beside a token count two orders of magnitude too small. The existing unit test asserted `input+output` — it restated the parser instead of checking it, so the bug shipped green. The fix and its regression tests are in `pkg/codex/usage.go` and `pkg/codex/usage_test.go` (working tree at time of writing, not yet committed). A requirement reading "measure token spend" would have been satisfied by the broken version.

---

## Key Findings

### The organising insight: what already exists and has never run

This table is the milestone. Everything else is downstream of it.

| Machinery | State | Evidence |
|-----------|-------|----------|
| `spawn-can-spawn` depth guard | Takes `--depth`, ignores it, returns `can_spawn: true` unconditionally. Its only two call sites are playbooks the runtime stopped loading in v1.25 | `cmd/spawn.go:169-182`; executed: `aether spawn-can-spawn --depth 99` → `{"can_spawn":true}` |
| `--enforce` flag | Documented in `.aether/workers.md:292` as the way workers check before spawning. Never registered | Executed: `aether spawn-can-spawn 5 --enforce` → `Error: unknown flag: --enforce`, exit 1 |
| Depth recording | `.aether/workers.md:328` hardcodes `--depth 0` for every child; `.claude/commands/ant/build.md:269` hardcodes `--depth 1` for every worker. Depth is caller-asserted, never derived | `spawn-tree-depth` computes `max(entry.Depth)` — structurally always 0 |
| Recursive-delegation policy engine | Complete: `MAX_SPAWN_DEPTH = 2`, `totalBudget`, `consumedBudget`, fail-closed rejection. Unreachable from the interactive build path | `.aether/ts-host/src/spawn-orchestrator.ts`; `build.md` forbids `aether host build` on this path |
| Agent roster | 27 caste YAMLs. Zero Go readers, zero TS readers. Real roster is a Go slice duplicated across 14 files, already disagreeing with the emoji/colour maps beside it | `colony/agents/*.yaml`; `cmd/caste_relevance.go:29`; grep for `colony/agents` across `*.go`/`*.ts` returns nothing |
| Model routing policy | Maps every caste to a model. Only reference is a test whose comment says a typed loader "is Phase 161 / MODEL-01's job" | `colony/policies/model-routing.yaml`; `cmd/policy_schema_test.go:119` |
| Skill lifecycle subsystem | 8 of 9 commands with zero references outside their own definition: `skill-patch`, `skill-promote`, `skill-archive`, `skill-pin`, `skill-view`, `skill-recover`, `skill-curator-run`, `skill-list-lifecycle` | `cmd/skill_lifecycle.go`, `cmd/skill_curator.go` |
| Caste-selection rationale | Composed on every build, carried into `codex_dispatch_contract.go:550` and into the JSON manifest, never rendered. The render pattern exists two files away in `gate.go:1425` | `cmd/caste_relevance.go:168`; `cmd/queen_spawn_budget.go` |
| Token usage | `WorkerUsage` parsed and attached at every dispatch; zero readers in `cmd/`. `codexExternalBuildWorkerResult` — the wrapper path — has no usage field at all | `pkg/codex/usage.go`, `pkg/codex/platform_dispatch.go:215`, `cmd/codex_build_finalize.go:62-81` |
| Skill authoring path | `.aether/commands/skill-create.yaml:5` declares `aether skill-create`. The Claude wrapper never calls it — the model hand-writes `SKILL.md`. Codex users get the runtime path; Claude and OpenCode users bypass it | Validation added to `skill-create` would protect one third of users while docs describe it as universal |

*For dummies: the colony already has most of these parts fitted. What is missing is the wire from the part to the thing that uses it — and in several cases a note on the wall saying the wire is already there.*

### Recommended Stack

**Zero new Go dependencies.** Every one of the capabilities can be built from Go 1.26 stdlib plus modules already in `go.mod`, and in most cases from patterns already load-bearing in this repo. This is not minimalism for its own sake — each would-be dependency has an in-repo replacement that is already proven and already tested.

**Core technologies (all already present):**
- **`invopop/jsonschema` + `santhosh-tekuri/jsonschema/v6`** — generate a JSON Schema from the Go struct, commit it, byte-compare it in a test, validate user files against it. Already wired end-to-end for the completion packet in `cmd/contract_schema.go`. Generating from the struct means the schema cannot drift from the loader.
- **`os.Root` (Go 1.24+ stdlib)** — symlink-escape-proof confinement for reading user-authored agent and skill directories. Replaces `securejoin` and the hand-rolled `filepath.Clean` prefix checks that are routinely wrong.
- **`pkg/storage.AppendJSONL` / `pkg/trace`** — append-only spend rows. A ledger is write-once, read-fold; SQLite (already in `go.mod` for hive FTS) would add a schema, a migration, and a second concurrency model for no query the fold cannot answer.
- **`golang.org/x/sync/errgroup` + `context` + `testing/synctest`** — tree-scoped fan-out with cancellation propagation, tested deterministically instead of with sleeps.
- **`PreToolUse` hook (`Agent|Task` matcher)** — the deterministic recursion gate. `cmd/hook_cmds.go` already parses this payload shape.

**Explicitly do not add:** filesystem watcher (no long-running process for it to serve; adds a mid-build-roster-change failure mode), local tokenizer (replaces exact provider counts with a guess — the precise problem `usage.go` exists to solve), struct-tag validators (a second validation vocabulary that cannot produce a committed drift-checkable artifact), a graph library for the spawn tree, SQLite for spend, emulated subagent nesting on Codex, or a second precedence model for agents that differs from the skills one.

**One tracked hygiene item, not for this milestone:** `gopkg.in/yaml.v3` was archived by its author on 2025-04-01 and is formally unmaintained. v1.26 materially expands its blast radius — YAML becomes a format a non-technical user hand-writes rather than one only Aether's committed policies use. The migration to `go.yaml.in/yaml/v3` is a mechanical import-path swap across ~15 files with no API change. It should be its own plan with its own verification, not a rider on the roster work.

### Expected Features

**Must have (table stakes — every comparable system has these; missing means unsafe, not just incomplete):**
- **Enforced spawn depth, derived not asserted** — every system bounds delegation. CrewAI's documented failure without it is infinite recursion.
- **A run-total worker counter with a hard stop** — depth alone does not bound cost. Devin bounds *only* this and ignores depth entirely.
- **Graceful degradation at the limit** — Claude Code withholds the `Agent` tool so the agent quietly does the work itself. LangGraph raises `GraphRecursionError` and the community writes parent-side wrappers to catch it. One of those is a feature.
- **Live delegation tree with parent attribution, indented by depth** — universal across Claude Code `/tasks`, Devin's TUI, OpenHands' GUI. Aether's caste identity (`🔨 Builder Mason-67`) is already richer than any of them; `SpawnEntry` already carries `ParentName` and `Depth`.
- **Only `name` + `description` required to author an agent** — Claude Code's floor. The description is not documentation, it is the routing key.
- **User agents and skills survive `aether update`** — manifest discipline already exists and is proven for skills. One clobbering incident destroys trust permanently.
- **Validation that refuses to launch a broken agent** — Claude Code shipped the unvalidated version, launched zero-tool agents that returned confusing results, and fixed it in v2.1.208.
- **Add a skill from a description / by file drop** — **already shipped.** `/ant-skill-create` plus `~/.aether/skills/domain/`. The gap is post-creation validation, not creation.

**Should have (differentiators — where Aether can be genuinely ahead):**
- **A deterministic one-clause "why" per selected caste, plus the not-called list.** Nobody does this well: Claude Code shows nothing, CrewAI dumps raw unreadable chain-of-thought, the observability platforms make you leave the tool for a web UI. The strings already exist and are already discarded. **Highest value-to-cost item in the milestone.**
- **A depleting budget counter in the default output** — "14 of 20 workers used this run". Devin's core cost-control UX brought into a terminal-native agent, and the single most legible safety widget for a non-technical owner.
- **Delegation gated by Queen judgement, not user config** — every comparable system exposes depth as a knob; this project's owner will not type `--max-spawn-depth`. Follow the verification-depth precedent exactly: Queen decides, flag is an advanced override, one env var as an escape hatch that fails safe on an unrecognised value (`AETHER_HIVE_POLICY`'s shape).
- **Post-creation reality check** — "this skill's detect patterns match 0 files in this repo". Converts silent failure into honest feedback; it is the Definition of Done applied to a user-facing feature.

**Defer or refuse:**
- **An agent/skill marketplace** — third-party marketplaces now advertise automated security scanning as their headline feature; the market has priced in that unvetted skill content is a prompt-injection vector executing with the user's tools. That is a permanent moderation obligation for a single-maintainer project.
- **A per-agent cost dashboard** — LangSmith/Langfuse/Arize territory, a separate product class. PROJECT.md already scopes out the analogous ledger web UI.
- **An interactive agent-creation wizard** — Claude Code built `/agents` as a wizard and removed it in v2.1.198; Cursor removed `/Generate Cursor Rules`. Both replaced it with "ask the model to write the file, then the file is the truth." Aether already does exactly this for skills.
- **User castes joining `queenBuildSafetyRequiredCastes`** — never. That set bypasses the worker cap by design; admitting user castes would silently break the light/standard/heavy guarantees `TestBuildWorkerCapHonoursVerificationDepth` protects.
- **Depth 4+, user-tunable delegation policy, cross-platform user agents** — v2+.

### Architecture Approach

**Every capability has a pre-existing seam. None requires a new architectural layer, and no capability needs a new store** — every write lands in a file the runtime already owns and already locks. The roster follows the three working "YAML file overlays a hardcoded default" loaders (`policy_loader.go`, `visuals_config.go`, `prompt_template_loader.go`). The spend ledger joins on `buildAttemptRecord.RunID`, which `trace-summary --run-id` already aggregates by. The survey digest, if in scope, copies `codegraph_context.go` exactly.

**The single hardest constraint is not a feature — it is the manifest binding.** `ExecutionBinding.ManifestSHA256` is a SHA-256 over the whole dispatch manifest, validated on the way back in. A nested spawn is by definition a worker that was not in the manifest and whose brief did not exist when the hash was taken. Amending the manifest and recomputing the digest must be rejected outright: a digest recomputed on demand authenticates nothing, and the digest exists precisely so a result from a stale checkout or superseded attempt cannot be accepted. The two coherent answers are **(B)** the child rides inside the parent's result via a new `child_results` field, so the binding never moves, and **(C)** the runtime issues a *new bound attempt* for a follow-on wave that the top-level wrapper spawns. Recommend B as the default and C as the guaranteed fallback — they share one adjudication point and one storage shape, so this is one feature with two delivery channels.

**Major components:**
1. **Spawn adjudicator** (`cmd/spawn_request.go`, new) — `aether spawn-request` validates depth, budget and caste and returns a runtime-composed child brief or a refusal with a reason. Go adjudicates and persists; Go never spawns on the wrapper path. The parent spawns via the platform Agent tool when the platform allows, and the top-level wrapper spawns via a follow-on attempt when it does not.
2. **Spawn budget ledger** (`pkg/codex/spawn_budget.go`, new) — a live worker counter for a run, a Go port of the TS orchestrator's policy. Critically, delegation must draw only from the *optional* remainder (`MaxWorkers - len(RequiredCastes)`, floored at zero), or a chatty Builder can spend the Watcher's slot and produce a build nobody checked.
3. **Agent roster loader** (`cmd/agent_roster.go`, new) — four-step resolution (cwd → repo root → hub → compiled-in fallback) with `sync.Once` caching, plus `aether roster-validate` that exits non-zero. **Overlay the Go slice; do not delete it in the same milestone.**
4. **Spend recorder** (`cmd/build_spend.go` + `pkg/trace` `LogWorkerSpend`, new) — worker usage → `trace.jsonl` keyed on the attempt run id, carrying caste, worker name, depth, parent, and measurement source.

**Three name collisions already exist and will bite a naive roster migration:** `route_setter` vs `route-setter.yaml`; `queen` is in `colony/agents/` and the visuals maps but deliberately *not* in the dispatchable registry (a loader that treats `colony/agents/` as the roster makes the Queen dispatchable); and the visuals maps are a 35-entry *superset* with different semantics that must not be unified with the 26-entry roster.

### Critical Pitfalls

1. **Shipping a depth limit that is documented but not enforceable.** This has already failed here twice — once via an unregistered flag, once via a structurally-always-zero depth field. **Avoid:** enforce at the only chokepoint an LLM cannot route around. Make `spawn-log` itself the guard: derive the child's depth from the parent's recorded entry, never accept `--depth` from the caller, refuse with a non-zero exit past the cap, refuse when the parent is not already in the tree. A worker that cannot log a spawn cannot legitimately spawn one. **Failable test:** `TestSpawnCanSpawnRefusesAtCap` — fails against today's code, which is the correct starting state.

2. **Guards that fail open.** `spawn-can-spawn-swarm` already returns `{"can_spawn": true}` when `COLONY_STATE.json` cannot be read, commented "No state = no spawns, budget available." Under recursion the case where state is unreadable correlates with the case where many workers are concurrently writing it — so the guard is most permissive exactly when the tree is largest. **Avoid:** every delegation guard fails closed, with a message naming the reason and an explicit operator recovery command.

3. **Budgeting the wave instead of the tree.** The existing cap governs workers per build wave. Eight top-level workers each spawning three children, each spawning three, is 8 → 32 → 104 agents under a cap that reads "8". Anthropic report early versions of their research system "spawning 50 subagents for simple queries" and multi-agent systems using ~15x the tokens of a chat. At 15x, a mis-shaped tree is not a slow build, it is a bill. **Avoid:** a tree-total budget distinct from the wave cap, decremented atomically by `spawn-log`, surfaced beside the wave cap so their difference is visible.

4. **Unbounded feedback paths a depth cap does not cover.** The 2026 IAL study found 68 infinite-agentic-loop failures across 6,549 repositories and 8 frameworks; all eight frameworks *already shipped* `max_iterations`/`max_turns`/`recursion_limit`, and developers "omit them, misuse them, configure them with ineffective bounds, or place them outside the actual feedback path." A→B→A at depth 2 never exceeds depth 3 and never terminates. **Avoid:** bound three things separately and place each bound on the repeated path — depth, tree total, and ancestor-chain task-identity (hash `(caste, normalised task)`, walk `ParentName`, deny on match).

5. **Orphaned children.** `spawn-complete` is worker-driven, so a worker that dies leaves its entry `active` forever — and active entries count as consumed budget. Orphans therefore both waste money and progressively lock the colony out of spawning. OpenAI's Codex has this as a live production issue. **Avoid:** runtime-owned reaping on every `build`/`continue` entry, an `orphaned` status distinct from `failed`, budget released, and an operator command (a reaper with no command violates the Definition of Done).

6. **User-authored content executing before any model can refuse it.** Datadog Security Labs demonstrated that Claude Code's `` !` `` dynamic-context syntax executes during preprocessing — their proof-of-concept exfiltrates a GitHub token, and in their test Claude stated "I'm not going to execute this skill" *having already executed it*. Model recognition is version-unstable: Opus 4.6 caught a malicious skill Opus 4.7 ran without flagging. Aether's exposure compounds because the hub is machine-global across colonies. **Avoid:** static refusal at index time, not run time; reuse the existing pheromone sanitiser for skill and agent bodies; provenance display on every injected skill; project-scoped skills inert until explicitly approved.

7. **Silent skill displacement.** Selection caps at top-3 colony + top-3 domain, `role_match` is worth 3 points and fires if the frontmatter merely lists the role with no limit on how many roles a skill may declare, and ties break *alphabetically by name*. A well-meaning non-expert writing one broad skill named `aaa-my-notes` can evict three shipped skills from every worker prompt in the colony, with no warning and no log line. **Avoid:** cap or decay score by declared breadth; replace the alphabetical tie-break with provenance-then-specificity; log displacement by name.

---

## Implications for Roadmap

Seven suggested phases. The **ordering constraints are load-bearing**; the names are not.

### Phase 1: Wiring Proof

**Rationale:** Must be first. A ratchet written after the capabilities will be shaped to whatever shipped — that is the mechanism by which 18 of 25 milestones ended up restoring something previously marked done. This phase costs little and converts the project's dominant failure mode from invisible to blocking.
**Delivers:** `TestNoRegisteredSubcommandIsUnreferenced`, seeded with today's known orphans (the 8 `skill-*` lifecycle commands) as an allowlist that may only shrink, running in CI. Plus an executability test asserting every command string in `.aether/workers.md` and the wrappers exits 0 — which fails today on `spawn-can-spawn … --enforce`.
**Avoids:** Pitfall 18 (the capability ships, the caller never lands, the docs say it works).
**Research flag:** none. `cmd/command_call_audit_test.go` and `TestUnblockWrapperIsWiredAndAtParity` are the existing idioms to extend.

### Phase 2: Delegation Guard

**Rationale:** Enforcement must precede capability. Everything after this phase either depends on the parent/depth linkage it records or is made unsafe by its absence — and linkage cannot be retrofitted to past runs.
**Delivers:** `spawn-log` derives depth from `--parent` and refuses an unknown parent or a depth past the cap; `spawn-can-spawn` returns `false`; `--enforce` either registered or `workers.md` corrected; all guards fail closed; a tree-total budget distinct from the wave cap, drawing only from the optional remainder; ancestor task-hash cycle refusal; orphan reaping plus `aether spawn-reap` (or an 8th stuck-state class in `aether recover`); `parent_name` and `spawn_id` on every recorded row; a `PreToolUse` `Agent|Task` matcher giving Go a deterministic veto; `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH` set in the Aether-managed `settings.json`. **Nothing spawns recursively yet.**
**Avoids:** Pitfalls 1–5, 7, and the linkage half of 11.
**Research flag:** none for the Go work. The TS `spawn-orchestrator.ts` is a complete, correct specification to port (~150 lines).

### Phase 3: Spend Ledger

**Rationale:** Ships before the extensibility phases because extensibility changes what workers cost, and without a working ledger those changes are unmeasurable. Depends on Phase 2 only for the parent linkage the subtree roll-up walks.
**Delivers:** `Usage` on `codexExternalBuildWorkerResult` (the wrapper path currently has no usage field at all) with the committed schema regenerated; `LogWorkerSpend` carrying caste/worker/depth/parent/source; both wrapper and hosted paths wired; `self` and `subtree` as two explicit columns, never one "total"; measured and estimated as separate subtotals with derived metrics refusing mixed sets; the four disjoint token counts displayed separately so cache savings are visible; the cache-arithmetic fix and its regression tests committed; char budgets renamed so no reader mistakes them for token budgets, and CLAUDE.md's "Token Budget" table header corrected in the same phase.
**Avoids:** Pitfalls 8–11.
**Research flag:** none. Provider usage semantics are documented and now verified by execution.

### Phase 4: Orchestration Visibility

**Rationale:** The cheapest high-value item in the milestone and the one that makes everything before it legible to a non-technical owner. **Movable** — the "why" half depends on nothing and could be pulled ahead of Phase 3 for an early visible win; the depleting counter needs Phases 2 and 3.
**Delivers:** one deterministic clause per selected caste, rendered in the Dispatch stage; the not-called list ("why didn't it use the security one?" is the question actually asked); the delegation tree indented by depth with parent attribution; "14 of 20 workers used this run"; `/ant-why` as a Tier-3 drill-down behind a command rather than more default output. Reason strings phrased in plain English at the render layer with the numeric form kept under `--why`.
**Addresses:** the milestone's headline differentiator. Claude Code shows nothing here, CrewAI dumps unreadable non-deterministic reasoning, and the observability platforms make you leave the tool.
**Avoids:** the anti-feature of raw LLM reasoning as explanation — untestable, and a claim you cannot assert is a claim that silently rots.
**Research flag:** none. `gate.go:1425` already renders `Rationale` in a human table; copy it.

### Phase 5: Roster Reader

**Rationale:** Reader first, against the *existing* 27 shipped YAMLs, proving it changes observable dispatch behaviour — before a single user-extension path is opened. Building the schema and directory before the reader reproduces the `colony/agents` outcome at larger scale. Must precede Phase 6 because a child dispatch needs to look up a caste's agent file, model and tools from somewhere.
**Delivers:** `cmd/agent_roster.go` with four-step resolution and compiled-in fallback; scoring and capability fields added to the existing YAMLs; `queenCasteRoster()` and caste scoring repointed through the loader; `aether roster-validate` exiting non-zero; `colony/agents` added to the hub sync pairs (today it is not published, so the roster would work only in this repo); a drift test asserting file and slice agree. **`casteRelevanceRegistry` stays as fallback.**
**Avoids:** Pitfalls 12 and 13.
**Research flag:** none — three proven YAML-overlay loaders exist in-repo. But it carries **one decision the roadmap must make explicitly, not discover**: `TestCanonicalAgentSourcesRemainAligned` asserts identical base names across `.claude`, `.opencode` and `.codex`. A user adding one agent to one platform either breaks CI or gets an agent that silently vanishes on the other two. Neither is acceptable; the answer is a distinct user namespace the parity test excludes and `agent-list` reports per-platform availability for.

### Phase 6: Delegation Delivery

**Rationale:** Largest and last of the delegation work. Depends on Phase 2 for adjudication, Phase 3 for budget data, Phase 5 for child caste lookup. Split into independently provable steps.
**Delivers:** **(a)** result plumbing — a flat, non-recursive `codexChildWorkerResult` (making "a child cannot declare children" a type-level fact rather than a validation rule), schema regenerated, child file claims merged into the parent's task claim, handoffs stamped with depth and parent and ranked below same-depth peers so children cannot flood the top-5 relay; **(b)** the follow-on-wave channel as the path that always works, using a new bound attempt rather than an amended manifest; **(c)** parent-direct spawn as an opportunistic optimisation gated on runtime capability detection.
**Avoids:** the manifest-digest anti-pattern; Pitfall 6 (child context composed, never inherited; child permission profile is the *intersection* of parent and caste, never a union or a default-to-parent).
**Research flag: YES.** Platform nested-spawn behaviour is MEDIUM confidence and changes fast — Claude Code has open bugs stripping the Agent tool from some subagent types, OpenCode has a permission rework that broke frontmatter `task:` overrides *and* an open "subagents can infinitely recurse, no max depth" defect. Any requirement depending on parent-direct spawning must carry a runtime capability check rather than a version assumption. Re-verify at planning time.

### Phase 7: Skill Authoring Hardening

**Rationale:** Last because skills are the one area that is largely already built — two of the three delivery models plus the entire matching pipeline ship today. The remaining work is hardening and validation, and it is genuinely narrow. Security refusals must land *in* this phase, never a phase later, or there is a published window in which malicious skills load.
**Delivers:** one writer — `skill-create` is the only path that may create a skill file, wrappers gather input and invoke it, enforced by a wrapper-contract test; a real validator with named enumerable rules invoked at create time *and* index time (unknown caste names, uncompilable detect patterns, name collisions, empty body, role-count cap, description-specificity floor); malformed skills reported rather than silently skipped; static refusal of `` !` `` dynamic-context syntax and unrestricted tool grants at index time; breadth-decayed role scoring, provenance-then-specificity tie-break, and displacement logged by name; the 8 orphaned lifecycle commands reclaimed rather than rebuilt beside.
**Avoids:** Pitfalls 14–17.
**Research flag: YES, narrow.** The skill supply-chain threat surface is moving; re-check current dynamic-context syntax variants and loading paths at planning time.

### Phase Ordering Rationale

- **Wiring Proof precedes everything** — the ratchet must exist before the capabilities it constrains.
- **Delegation Guard precedes Spend Ledger.** *This resolves an explicit disagreement between two reports.* ARCHITECTURE argues spend should ship first because delegation budgets set without spend data are guesses. PITFALLS argues the guard must ship first because the ledger's subtree roll-up depends on parent linkage recorded at spawn time, and linkage cannot be retrofitted. Both are right about different things. The synthesis: the guard *enforces* with a deliberately conservative default (depth 2, tree total ~20, the TS host's already-chosen numbers) without needing spend data, and the ledger then supplies the evidence to *retune* those numbers with the measurement recorded — the way `spawnThreshold`'s comment records why 30 stayed 30.
- **Spend Ledger precedes the extensibility phases** — extensibility changes what workers cost; without a ledger those changes are unmeasurable and every efficiency claim is unfalsifiable.
- **Roster reader precedes user agents and precedes delegation delivery** — prove the reader against the shipped 27 first; a child dispatch needs one caste-lookup path, not two.
- **Skill security lands with skill authoring, never after.**

### Research Flags

**Needs deeper research during planning:**
- **Phase 6 (Delegation Delivery)** — platform nested-spawn behaviour is MEDIUM confidence and both vendors have broken it within the last quarter. Vendor docs plus open issues must be re-fetched at planning time.
- **Phase 7 (Skill Authoring)** — supply-chain threat surface for user-authored agent content is actively evolving.

**Standard patterns — skip phase research:**
- **Phase 1 (Wiring Proof)** — the test idioms already exist in this repo.
- **Phase 2 (Delegation Guard)** — `spawn-orchestrator.ts` is a complete specification to port.
- **Phase 3 (Spend Ledger)** — provider usage semantics documented and now execution-verified.
- **Phase 4 (Orchestration Visibility)** — pure rendering of data that already exists, with the render pattern two files away.
- **Phase 5 (Roster Reader)** — three proven YAML-overlay loaders in-repo.

---

## Where the Reports Disagree

Recorded rather than smoothed over. Each needs an explicit decision in requirements.

| Question | Positions | Recommendation |
|----------|-----------|----------------|
| **Which capabilities are in this milestone?** | The four reports researched overlapping but non-identical sets. STACK: roster, skills, recursion, spend. ARCHITECTURE: recursion, roster, **survey digest**, spend. FEATURES: recursion, agents, skills, **orchestration explainability**. PITFALLS: delegation, roster, skills, spend | Survey digest and explainability each appear in only one report. Both need an explicit in/out ruling. Explainability is the highest value-to-cost item found by any researcher and should be in. Survey digest is genuinely additive scaffolding under a budget that is already tight — see below |
| **Where does the spend ledger live?** | STACK: a new `.aether/data/usage/spend-ledger.jsonl` via `pkg/storage.AppendJSONL`. ARCHITECTURE: do *not* invent a new store — write to `trace.jsonl` keyed on the build attempt's `run_id` | **ARCHITECTURE.** They agree on the format (append-only JSONL, not SQLite) and disagree only on the file. `trace-summary --run-id` already exists as the consumer and `buildAttemptRecord.RunID` already exists as the join key. Adopt STACK's row-shape requirements wholesale as the payload |
| **Does Claude Code cap total subagents per session?** | ARCHITECTURE states "session cap 200". FEATURES and STACK cite official docs stating explicitly *"There's no limit on the total number of subagents Claude can spawn over a session"*, and FEATURES traces the 200 figure to a secondary blog | **No session cap.** Official docs win. This makes a run-total counter a *gap in the reference implementation* Aether can fill, not a feature to copy |
| **What is depth 0?** | `build.md` hardcodes `--depth 1` for manifest workers; `workers.md` hardcodes `--depth 0` for their children. ARCHITECTURE proposes manifest worker = 1, child = 2. FEATURES proposes default 2 with ceiling 3 | The repo is internally inconsistent *today*, in a way that makes any cap ambiguous. Requirements must fix the origin explicitly and a test must assert it. The number matters less than the fact that one convention exists |
| **Which castes may delegate?** | ARCHITECTURE: start with Builder and Tracker only. FEATURES: build flows only — Watcher/Gatekeeper/Auditor/Probe never delegate, citing Claude Code's guidance to omit `Agent` from a reviewer's tools | Convergent, ARCHITECTURE is narrower. Start with the narrow allowlist; it is easy to widen and impossible to narrow after users rely on it |
| **Should the roster own model routing?** | `colony/agents/*.yaml` carries a `model:` field; `colony/policies/model-routing.yaml` maps every caste to a model. Both have zero readers | Decide in requirements, not implementation. Two unread sources of the same truth is how this situation arose |

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | **HIGH** | Every version claim checked against the live module proxy; every in-repo claim read from source. The only MEDIUM element is OpenCode's nesting semantics, whose documentation is thinner than Claude Code's |
| Features | **MEDIUM-HIGH** | Claude Code findings HIGH from official docs including version history. CrewAI/LangGraph/AutoGen/OpenHands MEDIUM from multiple agreeing secondary sources. Roster-size guidance LOW-MEDIUM — practitioner blogs only |
| Architecture | **HIGH** for internal integration points (read from source, with line numbers), **MEDIUM** for platform nested-spawn behaviour (vendor docs plus open issues; changes fast) |
| Pitfalls | **HIGH** for repo-specific findings (three verified by *executing* the binary), HIGH for Anthropic cache semantics (official docs), MEDIUM-HIGH for cross-framework failure data (peer-reviewed plus vendor engineering posts), MEDIUM for community-reported routing behaviour |

**Overall confidence: HIGH** — unusually so, because the load-bearing findings are about this repository and were verified by reading and running its code rather than inferred from documentation. The weaker material is all external and all peripheral to the ordering decisions.

### Gaps to Address

- **The "4–8 agents" roster-size claim is LOW-MEDIUM and should not become a hard requirement unexamined.** It is practitioner blog consensus with no official source and no controlled study — but also no dissenting voice found, and a plausible mechanism (description-based routing is nearest-neighbour matching over prose; near-duplicate neighbours degrade it). Aether's exposure is unusually high because it already ships 27 castes, so a user cap of 3–5 is being proposed *on top of* a namespace already four times the recommended size. FEATURES recommends making the cap a stated product opinion. **Handle:** if the cap ships, ship it as an opinion with a stated rationale and an override, not as a claimed empirical limit. Consider validating against real routing behaviour before hard-coding a number.
- **The survey-digest budget conflict has no free answer.** Per-section budgets already sum to ~17,700 chars against a 24,000 global cap, and `TestBuildWorkerBriefIsMostlyTask` explicitly classifies survey pointers as scaffolding. A digest is *more* scaffolding than a pointer list, not less. The only version that does not require raising a cap two invariant tests were written to defend is one where the digest **displaces** the pointer list rather than joining it. **Handle:** if in scope, require displacement; if either test fails, shrink the digest rather than raise the floor, and record the measurement if the floor ever does move.
- **OpenCode hook parity is unverified.** This research did not confirm an equivalent to Claude Code's `PreToolUse` deny gate. If recursion must work on OpenCode, Go-side depth enforcement becomes mandatory rather than defence-in-depth. **Handle:** verify before writing an OpenCode delegation requirement.
- **Child timeout semantics are undefined.** `WorkerDispatch.Timeout` currently means "one worker" and silently becomes "one subtree" under the recommended design. **Handle:** pick one and name it in the dispatch contract.
- **The empty-child-handoff rule.** A trivially-scoped child legitimately produces a near-empty handoff, which the current rule rejects. Recommendation is to require the parent to absorb the child's findings rather than exempt children — one rule, and it forces the parent to read what it delegated. **Handle:** decide in requirements.
- **Codex has no native subagent nesting.** Recursion on the Codex lane must be *off* and reported honestly, not emulated. Emulation would be a second divergent orchestration path — the exact failure v1.24 and v1.25 spent two milestones unwinding.
- **`gopkg.in/yaml.v3` is archived and unmaintained.** Tracked, not bundled into v1.26.

---

## Sources

### Primary (HIGH confidence)

**This repository, read directly and in several cases executed, 2026-08-08:**
- `cmd/spawn.go`, `cmd/internal_cmds.go`, `pkg/agent/spawn_tree.go` — the depth-guard stub, the fail-open swarm guard, the lineage store
- `cmd/caste_relevance.go`, `cmd/queen_judgement.go`, `cmd/queen_spawn_budget.go`, `cmd/codex_dispatch_contract.go` — caste scoring, rationale strings, safety-required castes, the manifest that carries the rationale and discards it
- `cmd/codex_build.go`, `cmd/codex_build_finalize.go`, `cmd/build_attempt.go`, `cmd/contract_schema.go` — manifest composition, the SHA-256 execution binding, the reflected packet schema
- `pkg/codex/usage.go`, `pkg/codex/platform_dispatch.go`, `pkg/trace/{trace,cost}.go`, `cmd/trace_cmds.go` — spend measurement and its absent consumer
- `cmd/skills.go`, `cmd/skill_lifecycle.go`, `cmd/skill_curator.go` — scan roots, top-3 cap, role scoring, alphabetical tie-break, the 8 unreferenced lifecycle commands
- `cmd/policy_loader.go`, `cmd/visuals_config.go`, `cmd/prompt_template_loader.go` — the three proven YAML-overlay loaders
- `.aether/ts-host/src/spawn-orchestrator.ts` — the complete, unreachable recursion policy engine
- `.aether/workers.md`, `.claude/commands/ant/build.md`, `.aether/commands/skill-create.yaml` vs `.claude/commands/ant/skill-create.md` — the documented invocations that do not execute, and the two divergent authoring paths
- `cmd/codex_build_test.go`, `cmd/context_budget_test.go`, `cmd/agent_mirror_test.go`, `cmd/command_call_audit_test.go` — the invariants this milestone must not retune, and the wiring-test idioms to extend

**Official documentation:**
- [Create custom subagents — Claude Code](https://code.claude.com/docs/en/sub-agents) — nesting depth default 3, `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`, `CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`, tool withheld at limit, `Task`→`Agent` rename, frontmatter minimum, duplicate-name behaviour, no session total limit
- [Hooks — Claude Code](https://code.claude.com/docs/en/hooks) — `PreToolUse` deny shape, `agent_id`/`agent_type` in subagent context, `SubagentStop` payload
- [Anthropic — prompt caching and usage semantics](https://platform.claude.com/docs/en/build-with-claude/prompt-caching) — disjointness of `input_tokens` / `cache_read_input_tokens` / `cache_creation_input_tokens`
- [Anthropic Engineering — multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) — 50-subagents-for-simple-queries, 4x/15x token multipliers, agent scaling rules, the duplicated-work example
- [Datadog Security Labs — malicious coding agent skills](https://securitylabs.datadoghq.com/articles/malicious-skills-supply-chain-risks-in-coding-agents-with-dynamic-context/) — `!` pre-execution bypass with credential exfiltration, model-refusal-after-execution, Opus 4.6 vs 4.7 detection instability
- [pkg.go.dev — `os.Root`](https://pkg.go.dev/os#Root); [go-yaml/yaml](https://github.com/go-yaml/yaml) (archived 2025-04-01); live module proxy for every version claim

### Secondary (MEDIUM confidence)

- [*When Agents Do Not Stop: Uncovering Infinite Agentic Loops in LLM Agents* (2026)](https://arxiv.org/html/2607.01641v1) — 68 confirmed failures across 6,549 repos and 8 frameworks; cause taxonomy; the finding that shipped bounds are misplaced rather than absent
- [*When Child Inherits: Modeling and Exploiting Subagent Spawn*](https://arxiv.org/pdf/2605.08460) — inherited prompts, tools and credentials; escalation via grandchildren; capability isolation as mitigation
- CrewAI [hierarchical process docs](https://docs.crewai.com/en/learn/hierarchical-process) and issues [#4783](https://github.com/crewAIInc/crewAI/issues/4783), [PR #2068](https://github.com/crewAIInc/crewAI/pull/2068) — delegation off by default, unbounded `max_iter` recursion
- LangGraph [`recursion_limit`](https://reference.langchain.com/python/langgraph-supervisor) and [community degradation thread](https://forum.langchain.com/t/gracefully-handling-graphrecursionerror-from-subgraphs-so-the-parent-agent-can-retry-or-degrade/2018) — crash-as-safety-mechanism
- Open platform defects: [claude-code#4182](https://github.com/anthropics/claude-code/issues/4182), [#80036](https://github.com/anthropics/claude-code/issues/80036), [opencode#8114](https://github.com/anomalyco/opencode/issues/8114), [#14308](https://github.com/anomalyco/opencode/issues/14308), [#18100](https://github.com/anomalyco/opencode/issues/18100) (infinite recursion, no max depth), [codex#19197](https://github.com/openai/codex/issues/19197) (orphaned subagents, missing lifecycle controls)
- [Langfuse #12306](https://github.com/langfuse/langfuse/issues/12306) — Anthropic exclusive vs OTel inclusive `input_tokens`, and the double-count that results from applying one convention to both
- [Arize — swarm management](https://arize.com/blog/swarm-management-of-agent-harnesses/) — "spawning is the beginning of the problem"
- [OpenHands (ICLR 2025)](https://arxiv.org/pdf/2407.16741); [Devin billing docs](https://docs.devin.ai/admin/billing) — ACU session caps as the only bound
- [OpenCode agents docs](https://opencode.ai/docs/agents/) — `task` tool, `permission.task` globs, **no documented nesting limit**

### Tertiary (LOW-MEDIUM confidence — flagged, needs validation)

- Practitioner consensus on roster sizing and routing degradation: [100 subagents → 12 that earn their context](https://dev.to/suraj_khaitan_f893c243958/i-built-100-claude-code-subagents-these-are-the-12-that-actually-earn-their-context-10nn), [Subagents Done Right](https://hackernoon.com/navigating-claude-code-subagents-done-right), [orchestration guide](https://hidekazu-konishi.com/entry/claude_code_subagents_and_orchestration_guide.html) — "4–8 specialised agents", "ten sharp descriptions beat a hundred vague ones". No official source, no controlled study, no dissenting source found either
- A secondary source claiming Claude Code enforces a 200-subagent session cap — **contradicted by official docs and not relied upon**
- [ReDel (EMNLP 2024)](https://arxiv.org/abs/2408.02248) — recursive delegation toolkit; no depth-limit failure analysis found in accessible summaries, so no depth-specific claim was drawn from it

---
*Research completed: 2026-08-08*
*Ready for roadmap: yes*
