# Stack Research — v1.26 Intelligent Orchestration

**Domain:** Go CLI multi-agent orchestration runtime (subsequent milestone on a mature codebase)
**Researched:** 2026-08-08
**Confidence:** HIGH for the four capability areas below; MEDIUM on OpenCode nesting semantics (documentation is thinner than Claude Code's)

---

## Headline: The Recommended Stack Addition Is Zero New Go Dependencies

Every one of the four capabilities can be built from Go 1.26 stdlib plus modules already
in `go.mod`. This is not a stylistic preference — it is what the evidence supports:

| Capability | What people usually reach for | What this repo already has |
|------------|-------------------------------|----------------------------|
| Schema-validated user config | `xeipuuv/gojsonschema`, `go-playground/validator` | `invopop/jsonschema` v0.14.0 + `santhosh-tekuri/jsonschema/v6` v6.0.2, already wired end-to-end in `cmd/contract_schema.go` |
| Hot-reload of user files | `fsnotify/fsnotify` | Nothing — and nothing is the right answer (see *What NOT to Use*) |
| Safe reads of user-supplied paths | `securejoin`, hand-rolled `filepath.Clean` checks | `os.Root` (Go 1.24+ stdlib, symlink-escape-proof) |
| Tree-scoped concurrency + cancellation | `ants`, `conc`, `tomb` | `golang.org/x/sync` v0.20.0 (`errgroup`) + `context` + existing process-group kill in `pkg/codex/process_group_unix.go` |
| Append-only spend ledger | `sqlite`, `bbolt`, a metrics library | `pkg/storage` `AppendJSONL`/`ReadJSONL` + `pkg/events` bus, both already load-bearing |
| Token counting | `tiktoken-go`, `pkoukk/tiktoken-go` | `pkg/codex/usage.go` — providers already report exact counts; a tokenizer would replace a measurement with a guess |

The one *genuine* dependency question in this milestone is a hygiene item, not a
capability blocker: **`gopkg.in/yaml.v3` was archived by its author on 1 April 2025 and
is formally unmaintained.** Details and the recommendation are in *Version Compatibility*.

---

## Recommended Stack

### Core Technologies (all already present)

| Technology | Version in `go.mod` | Purpose in v1.26 | Why Recommended |
|------------|---------------------|------------------|-----------------|
| Go | 1.26.5 (toolchain matches) | Runtime | `os.Root` (1.24+) gives symlink-proof confinement for reading user-authored agent/skill files without a third-party path library. Nothing in v1.26 needs a newer toolchain. |
| `github.com/invopop/jsonschema` | v0.14.0 (latest) | Generate JSON Schema from the Go struct that defines an agent/skill | Already used to reflect `codexExternalBuildCompletion` into `.aether/schemas/completion-packet.schema.json`. Generating the schema *from the struct* means the schema can never drift from the loader — the committed-schema byte-comparison test in `cmd/contract_schema_test.go` is the enforcement mechanism, and it already exists. |
| `github.com/santhosh-tekuri/jsonschema/v6` | v6.0.2 (latest is v6.0.3) | Validate a user-authored agent/skill against that schema and report *every* violation at once | Already compiled and used for completion-packet validation. Its `Schema.Validate(any)` accepts a decoded `map[string]any`, so YAML frontmatter can be validated without a JSON round-trip. The existing `contractViolation` shape and `violationPrinter` in `cmd/contract_schema.go` already render per-field messages — a non-technical user needs "line 4: `model` must be one of haiku, sonnet, opus", not "invalid agent". |
| `gopkg.in/yaml.v3` | v3.0.1 | Parse `colony/agents/*.yaml` and `SKILL.md` frontmatter | Already the parser behind `cmd/policy_loader.go` and `parseSkillFrontmatter`. **Archived upstream** — see Version Compatibility. |
| `golang.org/x/sync` | v0.20.0 (`errgroup`) | Tree-scoped fan-out with automatic context cancellation on first error | A dispatch *tree* is not a flat wave; `errgroup.WithContext` gives cancellation propagation down a subtree for free. Hand-rolling `sync.WaitGroup` + a done channel is where cancellation bugs live. |
| `github.com/spf13/cobra` | v1.10.2 | New subcommands (`agent-validate`, `skill-validate`, `spend-report`, …) | Every existing subcommand is Cobra; no reason to deviate. |
| `github.com/spf13/pflag` | v1.0.9 | — | Transitive; nothing new. |

### Stdlib Packages That Replace Would-Be Dependencies

| Stdlib | Purpose | Replaces |
|--------|---------|----------|
| `os.Root` / `os.OpenRoot` (Go 1.24+) | Read user-authored agent and skill directories confined to a root, symlink escapes included | `cyphar/filepath-securejoin`, ad-hoc `strings.HasPrefix(filepath.Clean(p), root)` checks that are routinely wrong |
| `crypto/sha256` | Ancestor-chain task-signature hashing for repetition detection; skill/agent manifest checksums | Any hashing library. Already used — `jsonSHA256` in `cmd/codex_build_finalize.go`, `content_hash` on pheromones |
| `context` | Cancellation propagation through a spawn subtree | `gopkg.in/tomb.v2` |
| `encoding/json` + `bufio` (via `pkg/storage.AppendJSONL`) | Append-only spend ledger | SQLite, bbolt, Prometheus client |
| `sync.Once` / `sync.RWMutex` | Roster load caching (mirrors `loadedReviewDepthPolicyOnce`, `policyCache`) | Any cache library |
| `testing/synctest` (Go 1.25+, GA) | Deterministic tests for concurrent tree cancellation without real sleeps | `go.uber.org/goleak`, timing-sensitive `time.Sleep` tests |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `go test ./... -race` | Existing gate | Tree-scoped concurrency and the spend ledger are both concurrent writers — `-race` is mandatory, not optional, for this milestone |
| `go vet ./...` | Existing gate | — |
| `goreleaser check` | Existing gate | Unaffected |
| Committed schemas under `.aether/schemas/` | Drift detection | Directory already exists (`completion-packet.schema.json`, plus 7 XSDs). Add `agent.schema.json` and `skill.schema.json` there, byte-compared in tests exactly as the completion packet is |

### Installation

```bash
# Nothing to install. The following is the full delta to go.mod for v1.26:
#   (empty)
#
# If the yaml hygiene item is taken (optional, see Version Compatibility):
go get go.yaml.in/yaml/v3@v3.0.5
go mod tidy
```

---

## Capability 1 — User-Extensible Agent Roster

### What exists today (verified by reading the code)

| Fact | Evidence |
|------|----------|
| The dispatchable roster is a hardcoded Go slice of 27 entries | `cmd/caste_relevance.go:29` `casteRelevanceRegistry` |
| `colony/agents/*.yaml` (27 files) has **no Go reader and no TypeScript reader** | grep across `**/*.go` and `.aether/ts-host/src/` returns zero hits for `colony/agents` |
| Caste names are additionally hardcoded in at least 14 non-test files | `grep -l '"gatekeeper"' cmd/*.go` → 14 files (`caste_relevance.go`, `queen_spawn_budget.go`, `queen_judgement.go`, `codex_visuals.go`, `gate.go`, `swarm_cmd.go`, `review_ledger.go`, `seal_final_review.go`, `phase_baseline.go`, `generate_cmds.go`, `codex_build.go`, `codex_continue.go`, `codex_continue_finalize.go`, `codex_plan.go`) |
| Presentation maps are separate hardcoded maps | `casteEmojiMap`, `casteColorMap`, `casteLabelMap` in `cmd/codex_visuals.go:38+`, and they already contain castes that are *not* in the relevance registry (`colonizer`, `surveyor`, `guardian`, `dreamer`, plus 8 curation ants) — the two lists already disagree |
| A caste is only spawnable if a platform agent file exists | `codexAgentNameForCaste` → `.claude/agents/ant/aether-<name>.md`, `.opencode/agents/<name>.md`, `.codex/agents/aether-<name>.toml` |
| `colony/agents` is **not published to the hub** | `cmd/install_cmd.go:794` publishes only `colony/policies` *filtered to `oracle-phase-directives.yaml`* |
| `colony/policies/model-routing.yaml` still has **no production reader** | Only reference is `cmd/policy_schema_test.go:119`, whose own comment says a typed loader "is Phase 161 / MODEL-01's job". A user-added agent has nowhere to declare its model cheaply. |

**Plain English:** the 27 agent YAML files in `colony/` look like the roster but nothing
reads them. The real roster is a Go list, duplicated across fourteen files, and it already
disagrees with the emoji/colour list next to it. A user adding an agent today changes
nothing.

### Recommended shape

**New Go package: `pkg/roster`** (not `cmd/`, so `pkg/agent` and future TS-host bridges
can use it). Three concerns, in this order:

1. **A typed `RosterAgent` struct** that is the single source of truth. Fields must cover
   what `casteRelevanceRegistry` holds *and* what `colony/agents/*.yaml` holds, because
   the two are being merged: `id`, `role`, `tier`, `model`, `color`, `description`,
   `allowed_tools`, `prompt_file`, `keywords`, `conditions`, `base_score`,
   `flows` (which of build/continue/plan/colonize/swarm/seal it may join),
   `keyword_gated` (bool), `can_spawn` (bool — feeds capability 3).

2. **Schema generation + validation using the already-proven pair.** Reflect
   `RosterAgent` with `invopop/jsonschema` into `.aether/schemas/agent.schema.json`,
   commit it, byte-compare it in a test (identical to `contract_schema_test.go`), and
   validate user files with `santhosh-tekuri/jsonschema/v6`. YAML decodes to
   `map[string]any`, which `Schema.Validate` accepts directly — but **YAML integers
   decode to `int` and the validator expects `json.Number`/`float64`**, so a small
   normalisation pass is required before validating. That is ~30 lines of stdlib, not a
   dependency.

3. **Layered scan roots with explicit precedence**, copied verbatim in shape from
   `skillScanRoots` (`cmd/skills.go:607`) because that pattern is already proven in
   production for exactly this problem:

   | Rank | Root | Meaning |
   |------|------|---------|
   | 1 (wins) | `.aether/agents/` (repo) | User's repo-local additions and overrides |
   | 2 | `~/.aether/agents/` (hub) | User's machine-wide additions |
   | 3 | `~/.aether/system/colony/agents/` | Shipped, published by `aether publish` |
   | 4 (floor) | `casteRelevanceRegistry` in Go | Compiled-in fallback so a broken/absent file tree never leaves the colony with zero castes |

   Rank-4 is not belt-and-braces; it is the same fallback discipline
   `heavyKeywordsFallback` uses in `cmd/review_depth.go:56`, and it is why a malformed
   user file can be *rejected loudly* rather than silently degrading dispatch.

### Answers to the specific questions asked

**Schema validation:** as above. Reuse, do not add.

**Hot-reload vs restart:** **restart-scoped load, cached with `sync.Once`.** Every Aether
command is a short-lived process — `aether host build`, `aether spawn-log`,
`aether build-finalize`. There is no long-running daemon for a watcher to serve. Adding
`fsnotify` would introduce a filesystem-event dependency, platform-specific behaviour
(kqueue/inotify/ReadDirectoryChangesW), and a class of bug (roster changed mid-build,
half the wave dispatched under the old roster) in exchange for a benefit no user can
observe. Cache invalidation should follow the existing `skill-cache-rebuild` precedent:
an explicit `agent-cache-rebuild`, plus mtime+size comparison on the index, not a watcher.

**Collision handling:** deterministic precedence rank, never filesystem read order. This
is a specific correctness requirement rather than a nicety — Claude Code's own docs state
that when two subagent files in the same directory declare the same `name`, it "loads only
one of them, chosen by filesystem read order rather than a documented precedence."
Aether must not inherit that. `skillIndexEntryPriority` in `cmd/skills.go` is the working
model: key on `type + ":" + name`, keep the lowest rank, and *record the shadowed entry*
so `agent-list` can show "3 agents defined, 1 shadowed by a repo-local override."

**Malformed user agent:** three rules, all derived from this project's Definition of Done
and its own history of silent degradation:

1. **Never return `nil` silently.** `indexSkillDir` (`cmd/skills.go:647`) returns `nil` on
   read error *and* on missing name, with no diagnostic. A user's malformed skill file
   vanishes with no message. Do not repeat that shape for agents, and fix it for skills
   (capability 2).
2. **Reject the file, not the run.** A malformed `colony/agents/foo.yaml` must not abort a
   build. It must be excluded, named on stderr with the failing field and rule, and
   surfaced in `agent-list` / `patrol` as a persistent warning. Fail-closed on the *agent*,
   fail-open on the *colony*.
3. **A user-added agent that no platform file backs is a validation failure, not a runtime
   surprise.** The check belongs in `agent-validate`, run at add time, not discovered when
   a Task spawn returns "unknown subagent type". `validateOpenCodeAgentFile`
   (`cmd/platform_sync.go:886`) is the existing model for this kind of pre-publish gate.

### Integration points (specific)

| File | Change |
|------|--------|
| `cmd/caste_relevance.go` | `casteRelevanceRegistry` becomes `casteRelevanceFallback`; `findProfile` and `queenCandidateDispatches` iterate `roster.Load()` |
| `cmd/codex_visuals.go` | `casteEmojiMap`/`casteColorMap`/`casteLabelMap` read from roster with the map as fallback; a user agent without a colour gets a deterministic hash-assigned one rather than blank |
| `cmd/install_cmd.go:794` | Publish `colony/agents` and the *whole* of `colony/policies` (today the `isOraclePhaseDirectivesFile` filter drops nine of ten policies) |
| `cmd/generate_cmds.go` | Generate the `.claude`/`.opencode`/`.codex` agent files for a user-added caste |
| `.aether/schemas/agent.schema.json` | New committed artifact |
| New: `agent-list`, `agent-validate`, `agent-add`, `agent-cache-rebuild` | Cobra subcommands; `agent-validate` is the command that fails when the requirement is unmet |

---

## Capability 2 — User-Added Skill Files

### What exists today

Skills are **already 80% data-driven**, which is the most important finding here. Do not
rebuild what works.

| Already working | Evidence |
|-----------------|----------|
| Multi-root scanning with 4 roots | `skillScanRoots` (`cmd/skills.go:607`) |
| Deterministic collision resolution by rank | `buildFullIndex` (`cmd/skills.go:451`) keyed on `type:name` |
| `is_user_created` tracking with declared-source override | `skillIsUserCreatedFromSource` (`cmd/skills.go:686`) |
| Update-safety manifest with per-skill checksum | `skillManifestData` / `pruneShippedFromUserSkillsDir` |
| Two-layer cache (runtime map + on-disk index) with explicit rebuild | `skillIndexRuntimeCache`, `skill-cache-rebuild` |
| Frontmatter parsing with a line-by-line fallback when YAML fails | `parseSkillFrontmatter` (`cmd/skills.go:520`) |

### The actual gaps

| Gap | Why it matters | Fix (no new deps) |
|-----|----------------|-------------------|
| **Malformed skills disappear silently.** `indexSkillDir` returns `nil` on read error or empty `name`; `buildFullIndex` `continue`s past it | A non-technical user adds a skill, gets no error, and the skill never loads. This is the exact silent-failure class v1.25 Phase 160 was built to eliminate | Return `(entry, error)`; collect errors into the index; surface in `skill-list` and a new `skill-validate` |
| **The YAML-failure fallback is worse than an error.** When `yaml.Unmarshal` fails, `parseSkillFrontmatter` silently switches to naive `strings.HasPrefix` line scraping, producing a half-parsed skill that looks fine | A typo in a list field yields a skill with a name and no `detect` patterns — it loads, matches nothing, and reports success | Keep the fallback only for the read path (backward compat), but have `skill-validate` reject anything that needed it, naming the YAML error |
| **No schema.** Frontmatter has 15 optional fields, three of them near-synonyms (`agent_roles`/`roles`, `detect`/`detect_files`/`detect_packages`) | A user cannot know which field to use, and a wrong guess fails silently | Reflect `skillFrontmatter` with `invopop/jsonschema` → `.aether/schemas/skill.schema.json`; validate; the schema doubles as the documentation |
| **"The right repo" is undecided.** `/ant-skill-create` writes somewhere; the user's stated intent ("that sounds useful") does not say *scope* | A machine-wide skill written into one repo is invisible everywhere else, and vice versa | `skill-add --scope repo\|hub` with an explicit default and the resolved path echoed back. Decision rule to recommend to the user: repo-specific conventions → `.aether/skills/`; reusable technique → `~/.aether/skills/domain/` |
| **User skill dirs are read with bare `os.ReadFile` on joined paths** | A symlinked skill directory can read outside the intended root | `os.OpenRoot` on each scan root, then `root.ReadFile` |

**Recommendation:** treat capability 2 as *hardening plus a validate command*, not new
architecture — and then build capability 1 on the same scan-root/rank/manifest machinery
rather than inventing a second one. If the agent roster and the skill roster end up with
two different precedence models, that divergence becomes the next milestone's bug.

---

## Capability 3 — Recursive Delegation

This is the area where the research most changes the plan, in two directions at once.

### Finding A: the platforms already do recursive delegation natively

Verified against current Claude Code documentation (code.claude.com/docs/en/sub-agents,
fetched 2026-08-08):

> "By default, a subagent can spawn subagents of its own, up to three layers below the
> main conversation. At the depth limit, Claude Code withholds the `Agent` tool from every
> subagent except a fork, so a subagent at the limit does its delegated work itself and
> returns one summary."

| Control | Value | Notes |
|---------|-------|-------|
| `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH` | default 3 layers below main; `1` turns nesting off | Settable in `settings.json` `env` block — which `aether publish` already syncs (`isClaudeSettingsFile`, `cmd/install_cmd.go`) |
| `CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS` | default 20 running at once; requires Claude Code v2.1.217+ | "There's no limit on the total number of subagents Claude can spawn over a session" — total spend is *not* platform-capped |
| Tool name | `Agent`. Renamed from `Task` in v2.1.63; `Task(...)` still works as an alias | Aether's agent frontmatter uses `Task` — valid, but `Agent` is the current name |
| Per-subagent opt-out | Omit `Agent` from `tools`, or list it in `disallowedTools` | Aether grants `Task` to exactly 4 of 27 agents today: architect, keeper, route-setter, queen |
| `Agent(agent_type)` allowlist | **Ignored inside a subagent definition** — only applies to a main-thread `--agent` | So Aether cannot restrict *which* castes a nested worker spawns via frontmatter. That restriction has to come from the hook. |
| Background subagents | Run in background by default as of v2.1.198; keep `Agent` but lose many built-in tools | Aether's build wrapper already mandates "Do not set `run_in_background`" — keep that |

OpenCode has a `task` tool and `permission.task` glob patterns for restricting which
subagents an agent may invoke; its documentation does not state a nesting limit
(MEDIUM confidence — treat OpenCode depth as unbounded and enforce in Aether).
Codex has no native subagent nesting; recursion on the Codex lane must be **off**, not
emulated, and the runtime should say so rather than pretend parity.

**Implication:** do not build a spawn mechanism. Build the *governor*. The platform
enforces depth correctly and for free; set `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH` in the
Aether-managed `settings.json` and let it do that job.

### Finding B: Aether's own depth machinery is a stub

| Claim | Reality |
|-------|---------|
| `spawn-can-spawn --depth N` "Check if spawning is allowed at given depth" | `cmd/spawn.go:169-182` — the entire body is `outputOK({"can_spawn": true, "depth": depth})`. It has never returned `false` for any input. |
| `SpawnEntry.Depth` is tracked | `pkg/agent/spawn_tree.go:24` — yes, persisted as field 6 of the pipe format |
| Depth is *derived* | No. `spawn-log --depth` is caller-asserted. `.claude/commands/ant/build.md:269` hardcodes `--depth 1` for every worker. Nothing checks that a child's depth is parent's + 1, or that the named parent exists. |
| `spawn-tree-depth` enforces anything | No — it reports `max(Depth)` observed. Pure telemetry. |
| Budget is enforced across a tree | `queenSpawnBudget` (`cmd/queen_spawn_budget.go`) is a **flat, pre-dispatch** cap on how many castes the Queen selects for one wave. It has no concept of children. |
| Anything enforces recursion limits | Only `.aether/ts-host/src/spawn-orchestrator.ts` (depth ≤ 2, `totalBudget` default 20, fail-closed). It is reachable from `host.ts`/`wave-orchestrator.ts` — but **not** from the host-manifest flow that `.claude/commands/ant/build.md` actually drives, where the wrapper spawns via the platform Agent tool directly. |

This is the same pattern this project's CLAUDE.md documents four times over: the machinery
exists, is documented, and has never executed. A v1.26 requirement here must be
`spawn-can-spawn` returning `false` under a named test, not "depth support added".

### What Go actually needs

**1. A depth ledger, derived not asserted.** `spawn-log` must take `--parent` (it already
does, and already requires it) and *compute* depth as `depth(parent) + 1`, rejecting an
unknown parent. Change `RecordSpawn` to resolve depth from the tree rather than accept it.
Keep `--depth` accepted-but-ignored for one release for wrapper compatibility, and make a
mismatch a loud warning. Zero new dependencies.

**2. The `PreToolUse` hook is the only deterministic gate — use it.** This is the highest-leverage
finding of the whole research. Claude Code's `PreToolUse` hook:

- fires **before** the tool executes and can **deny** it,
- carries `tool_name`, `tool_input`, `tool_use_id`,
- carries `agent_id` and `agent_type` **when running inside a subagent**,
- denies via `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` or exit code 2.

Aether already ships `aether hook-pre-tool-use` wired to `Write|Edit` in
`.claude/settings.json`. Adding a matcher for `Agent|Task` gives the Go runtime a
deterministic, LLM-proof veto over every recursive spawn — budget, roster membership
(is the requested `subagent_type` a real caste?), `can_spawn` on the parent's roster
entry, and repetition. This is prompt-independent enforcement, which is exactly what this
project has learned to require. **No new dependency; `cmd/hook_cmds.go` already parses
this input shape.**

Caveat to design around honestly: the hook payload gives the *parent's* `agent_id`, not a
depth number, and the child's `agent_id` does not exist yet. The runtime must maintain
`agent_id → depth` itself, seeded at depth 0 when `agent_id` is absent (main conversation).
Where the mapping cannot be resolved, **fail closed** — treat the spawn as being at the
deepest known level. Do not invent a confident depth.

**3. "Cycle detection" is the wrong frame — say so in the requirement.** A spawn tree is a
tree by construction; a child cannot become its own ancestor. The real hazard is
**ancestor-chain repetition**: Watcher spawns Probe spawns Watcher on the same task,
forever. The check is: hash a normalised `(caste, task-signature)` with `crypto/sha256`,
walk the ancestor chain via `ParentName` in the existing spawn tree, deny on match.
`pkg/colony/cycle.go`'s three-colour DFS is the wrong tool (it is for the task *dependency*
graph, where cycles are genuinely possible) — but its `CycleError{Tasks []string}` shape,
which names the offending path, is exactly the right error ergonomics to copy.

**4. Budget must become tree-scoped.** `queenSpawnBudget.MaxWorkers` is currently consumed
once, at wave planning. It needs a persisted `consumed` counter for the run — the
`SpawnRun` machinery in `pkg/agent/spawn_tree.go` (`BeginRun`/`EndRun`/`EntriesForRun`,
already run-scoped and already trimmed to 12 runs) is the natural home. Count all
descendants against the run budget, not just the Queen's direct wave. The TS
`spawn-orchestrator.ts` fail-closed semantics ("over-budget spawns are skipped, never
queued") is a correct precedent to port to Go.

**5. Cancellation propagation — be honest about the two lanes.**

| Lane | Mechanism | Status |
|------|-----------|--------|
| Codex / direct subprocess | `syscall.Kill(-pid, SIGTERM/SIGKILL)` on the process group, workers spawned with `Setpgid: true` | **Already works** — `pkg/codex/process_group_unix.go`, `process_tracker.go`. Killing the parent's group takes its descendants. |
| Platform Agent tool (Claude/OpenCode) | The platform owns the subagent lifecycle; there is no PID | Go **cannot** kill these. It can mark the subtree cancelled in the ledger and deny further spawns via the hook. Claude Code's `SubagentStop` hook (`agent_id`, `agent_type`, `last_assistant_message`, and it can block) is the reconciliation point. |

Do not write a requirement that promises Go can cancel a running platform subagent. Write
one that promises the ledger reflects reality within one hook event, and that no *new*
work is admitted under a cancelled ancestor.

**6. `errgroup` for in-process tree fan-out.** `golang.org/x/sync` v0.20.0 is already a
direct dependency. `errgroup.WithContext` + `SetLimit` gives cancellation-on-first-error
and a concurrency ceiling in the Codex lane without hand-rolled channel plumbing. Pair with
`testing/synctest` (Go 1.25+, stdlib) so the cancellation tests are deterministic rather
than sleep-based.

---

## Capability 4 — Spend Accounting Across a Dispatch Tree

### What exists

`pkg/codex/usage.go` defines `WorkerUsage` (input/cached/output/total tokens, USD cost,
model, and a `Source` of `"provider"` or `"estimate"`). `AttachWorkerUsage`
(`pkg/codex/platform_dispatch.go:203`) attaches it to `WorkerResult` at the one boundary
every real dispatch crosses. Its own doc comment is the design brief:

> "Every budget in the codebase counts characters of assembled context, which is ~6k
> tokens against the ~117k a real worker spends."

**It is never persisted.** `WorkerResult` is in-memory; the value dies with the process.
`grep WorkerUsage` outside `usage.go` and tests returns exactly two files, both in the
dispatch path. There is no ledger, no aggregation, no report.

### Recommended shape

**Storage: append-only JSONL via `pkg/storage.AppendJSONL`.** Path
`.aether/data/usage/spend-ledger.jsonl`. Reasons, in order of weight:

1. The pattern is already load-bearing in this repo — `pkg/events` is a JSONL event bus
   with TTL, and `AppendJSONL`/`ReadJSONL` already exist with file locking
   (`pkg/storage/lock.go`).
2. A spend ledger is write-once, read-fold. It has no update, no join, no index need.
   `modernc.org/sqlite` v1.50.0 *is* in `go.mod` (used by hive FTS recall) — using it here
   would add a schema, a migration, and a second concurrency model for no query the fold
   cannot answer.
3. Append-only is auditable. A tree total that can be recomputed from raw rows is
   falsifiable; a mutated running total is not — and falsifiability is the stated point of
   the usage work.

**Row shape.** The aggregation shape is the whole question, so state it precisely. A row
must carry enough to reconstruct the tree without a second file:

| Field | Why it must be there |
|-------|----------------------|
| `run_id` | Ties rows to a `SpawnRun`; without it, two concurrent colonies pollute each other's totals |
| `worker_name` | Join key to `spawn-tree.txt` |
| `parent_name` | The edge. Aggregation walks these, so the ledger must not depend on `spawn-tree.txt` still being present |
| `depth` | Lets a report answer "what did nesting cost me?" without a tree walk |
| `caste`, `agent_type` | Per-caste cost attribution — the input to any future cheap-model routing decision |
| `model` | Same run, different models per caste; a total without model breakdown cannot justify a routing change |
| `phase`, `workflow` | Per-phase and per-flow rollups |
| `usage` (embedded `WorkerUsage`) | The measurement, `Source` included |
| `started_at` / `ended_at` | Wall-clock cost, and run-window filtering consistent with `filterEntriesForRun` |

**Aggregation:** a fold, not a stored total. `spend-report --run <id> --by caste|model|depth|phase`
reads the ledger once and folds. Tree rollup = group by `run_id`, then walk `parent_name`
edges to attribute a subtree total to its root worker. `EntriesForRun`
(`pkg/agent/spawn_tree.go:180`) already implements the run-window filtering semantics to
mirror.

**Three hard rules, all derived from the existing `usage.go` doc comment:**

1. **Never sum a measurement and an estimate into one undifferentiated number.** A report
   must show `measured: 412k tokens (14 workers) · estimated: 31k (2 workers)`. The
   `Source` field exists precisely so this is possible; collapsing it re-creates the
   unfalsifiable claim the field was added to kill.
2. **A missing row is worse than an estimated one.** `EstimateUsage` already exists for
   this. Write a labelled estimate row for any dispatch that reported nothing, rather than
   letting the dispatch vanish and the run look cheap.
3. **Accumulate money in integer micro-USD when folding.** `WorkerUsage.USDCost` is
   `float64` (correct at the boundary — that is what providers report). Summing hundreds of
   float64 costs accumulates drift that shows up as a total that does not match the sum of
   the displayed rows. Convert to `int64` micro-dollars inside the fold. Pure stdlib.

**Do not add a tokenizer.** `tiktoken-go` and friends re-derive numbers the provider
already reports exactly. `ParseUsage` reads Codex `token_count` NDJSON events and Claude's
terminal `{"type":"result","usage":{…},"total_cost_usd":…}` — both are ground truth. A
tokenizer would replace a measurement with an approximation and reintroduce the exact
problem `usage.go` was written to solve.

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `santhosh-tekuri/jsonschema/v6` (already present) | `google/jsonschema-go` (333 snippets, benchmark 88.67, newer and actively developed) | If Aether ever needs schema *inference* from arbitrary JSON at runtime, or if `santhosh-tekuri` stalls. Not now — switching costs a rewrite of `cmd/contract_schema.go` for no capability gain |
| Restart-scoped roster load with `sync.Once` | `fsnotify/fsnotify` | Only if Aether grows a long-running daemon or a live TUI that must reflect roster edits without restart. `/ant-watch` is not that today |
| JSONL spend ledger | `modernc.org/sqlite` (already in `go.mod`) | If spend reporting later needs cross-run analytical queries, joins against hive wisdom, or retention windows measured in months. A JSONL→SQLite import is trivial later; the reverse is not |
| `PreToolUse` hook as the recursion gate | Prompt-level instruction ("do not spawn more than N") | Never. The project's own history is a list of prompt-level rules that were not followed |
| Platform-native depth limit (`CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`) | Go-enforced depth only | Use Go-enforced depth *in addition*, for the OpenCode lane where no documented platform limit exists, and as defence in depth. Do not use it *instead* — the platform limit is free and cannot be talked around |
| `os.Root` | `cyphar/filepath-securejoin` | If Aether ever needs to support Go < 1.24. It does not; `go.mod` says 1.26.5 |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `fsnotify/fsnotify` | Solves hot-reload, a problem short-lived CLI processes do not have. Adds three platform-specific backends and a mid-build-roster-change failure mode | `sync.Once` load + mtime/size index check + explicit `agent-cache-rebuild` (mirrors the working `skill-cache-rebuild`) |
| `tiktoken-go` / any local tokenizer | Replaces exact provider-reported counts with an approximation, defeating the stated purpose of `pkg/codex/usage.go` | `ParseUsage` + `EstimateUsage` (labelled), already written |
| `go-playground/validator` struct tags | A second, incompatible validation vocabulary alongside the JSON Schema already generated from Go structs. Cannot produce a committed, drift-checkable artifact | `invopop/jsonschema` → committed schema → `santhosh-tekuri/jsonschema/v6` |
| A new DAG/graph library for the spawn tree | A spawn tree is a tree with a `parent` pointer, ≤3 deep, ≤20 wide. `pkg/agent/spawn_tree.go` already stores and parses it | Ancestor walk over `ParentName` |
| SQLite (or bbolt) for the spend ledger | Schema + migration + a second concurrency model for a write-once/read-fold workload | `store.AppendJSONL` / `store.ReadJSONL` |
| A worker-pool library (`ants`, `conc`) | `pkg/agent/pool.go` exists; `errgroup.SetLimit` covers the ceiling case | `golang.org/x/sync/errgroup` |
| Emulating subagent nesting on the Codex lane | Codex has no native subagents. Emulation would be a second, divergent orchestration path — the exact failure v1.24/v1.25 spent two milestones unwinding | Set `can_spawn: false` for the Codex lane and report the limitation honestly in `command-guide` |
| A second precedence model for agents different from skills | Two precedence models is next milestone's bug report | One `scanRoots` + rank helper shared by `pkg/roster` and `cmd/skills.go` |
| Trusting `spawn-log --depth` from the caller | It is asserted by an LLM-driven wrapper and hardcoded to `1` in `build.md` | Derive depth from `--parent` in `RecordSpawn` |

---

## Stack Patterns by Variant

**If the milestone ships recursion on Claude Code only:**
- Platform enforces depth (`CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH` in the Aether-managed `settings.json`)
- Go enforces budget + repetition + roster membership via the `PreToolUse` `Agent|Task` hook
- `SubagentStop` hook reconciles `spawn-complete` and writes the spend row
- Because this needs no per-worker wrapper instrumentation, it is also the cheapest lane to prove first

**If recursion must also work on OpenCode:**
- OpenCode's `permission.task` glob restricts *which* subagents an agent may invoke — a coarser but real control
- No documented OpenCode nesting limit → Go-side depth enforcement becomes mandatory, not defence-in-depth
- OpenCode hook parity needs verification before the requirement is written (this research did not confirm an equivalent `PreToolUse` gate)

**If the user-extensible roster ships without recursion:**
- `pkg/roster` + schema + `agent-validate` + publish path, and nothing else
- `can_spawn` still belongs in the schema from day one — adding a field later means a schema version bump and a migration

**If spend accounting ships first (lowest risk, highest immediate value):**
- Ledger + `spend-report` are independently useful and have no dependency on capabilities 1–3
- It also produces the evidence needed to justify (or refute) recursion: if a nested tree costs 4× a flat wave for the same outcome, that is a measurement the roadmap should want *before* committing to recursion

---

## Version Compatibility

| Package | Current | Latest | Notes |
|---------|---------|--------|-------|
| Go | 1.26.5 | — | `os.Root` needs 1.24+, `testing/synctest` needs 1.25+. Both satisfied |
| `santhosh-tekuri/jsonschema/v6` | v6.0.2 | v6.0.3 | Patch available. Bump is low-risk but not required by any capability |
| `invopop/jsonschema` | v0.14.0 | v0.14.0 | Current |
| `spf13/cobra` | v1.10.2 | v1.10.2 | Current |
| `golang.org/x/sync` | v0.20.0 | v0.20.0 | Current |
| `modernc.org/sqlite` | v1.50.0 | v1.50.0 | Present for hive FTS; **not** recommended for the spend ledger |
| `gopkg.in/yaml.v3` | v3.0.1 | v3.0.1 (final) | **Archived 2025-04-01, author-declared unmaintained.** No further releases, including security fixes |
| `go.yaml.in/yaml/v3` | — | v3.0.5 | Community-maintained successor under the YAML org; drop-in import-path swap for v3 |
| `go.yaml.in/yaml/v4` | v4.0.0-rc.2 (**indirect, already in the graph**) | v4.0.0-rc.6 | Pulled in transitively. Still release-candidate — do not adopt directly yet |

**Recommendation on YAML:** v1.26 expands YAML's blast radius considerably — it becomes
the format a *non-technical user hand-writes*, parsed from untrusted-ish local files,
rather than a format only Aether's own committed policies use. That is a materially
different risk profile for an unmaintained parser. But this is a hygiene item, not a
capability blocker, and this project has an explicit rule against adding surface without
evidence. Recommended: **flag it as a tracked item, do not bundle the migration into
v1.26.** If it is taken, it is a mechanical import-path swap (`gopkg.in/yaml.v3` →
`go.yaml.in/yaml/v3`) across ~15 files with no API change, and it should be its own plan
with its own verification, not a rider on the roster work.

---

## Explicit "Do Not Add" List for the Requirements Doc

Copy this into v1.26 requirements as a negative constraint:

1. No new entries in `go.mod` `require` block are needed for any of the four capabilities.
2. No filesystem watcher.
3. No local tokenizer.
4. No second validation vocabulary (struct-tag validators).
5. No SQLite table for spend.
6. No graph library for the spawn tree.
7. No emulated subagent nesting on Codex.
8. No new precedence/collision model — extend the skills one.
9. No requirement that promises Go can kill a running platform subagent.
10. No requirement satisfied by `spawn-can-spawn` "supporting depth" — it is satisfied when
    `spawn-can-spawn` returns `false` and a named test fails if it stops doing so.

---

## Sources

- `cmd/caste_relevance.go`, `cmd/queen_spawn_budget.go`, `cmd/spawn.go`, `cmd/spawn_track.go`, `cmd/skills.go`, `cmd/contract_schema.go`, `cmd/policy_loader.go`, `cmd/review_depth.go`, `cmd/install_cmd.go`, `cmd/platform_sync.go`, `cmd/hook_cmds.go`, `cmd/circuit_breaker.go`, `cmd/policy_schema_test.go` — read directly, this repo, 2026-08-08 — HIGH
- `pkg/agent/spawn_tree.go`, `pkg/codex/usage.go`, `pkg/codex/platform_dispatch.go`, `pkg/codex/process_tracker.go`, `pkg/codex/process_group_unix.go`, `pkg/codex/handoff.go`, `pkg/storage/storage.go`, `pkg/colony/cycle.go` — read directly — HIGH
- `.aether/ts-host/src/spawn-orchestrator.ts`, `.aether/ts-host/src/wave-orchestrator.ts`, `.claude/commands/ant/build.md`, `.claude/settings.json`, `colony/agents/*.yaml`, `colony/policies/*.yaml` — read directly — HIGH
- https://code.claude.com/docs/en/sub-agents — subagent nesting depth, `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`, `CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`, `Task`→`Agent` rename in v2.1.63, frontmatter field table, scope precedence, duplicate-name behaviour — fetched 2026-08-08 — HIGH
- https://code.claude.com/docs/en/hooks — `PreToolUse` deny shape, `agent_id`/`agent_type` in subagent context, `SubagentStop` payload and blocking semantics, exit-code semantics — fetched 2026-08-08 — HIGH
- https://opencode.ai/docs/agents/ — agent markdown/JSON config, `task` tool, `permission.task` globs, agent file locations; **no documented nesting limit** — fetched 2026-08-08 — MEDIUM
- https://pkg.go.dev/os#Root — `os.Root` added Go 1.24, symlink-escape prevention, method set — fetched 2026-08-08 — HIGH
- https://github.com/go-yaml/yaml — repository archived 2025-04-01, author-declared unmaintained — fetched 2026-08-08 — HIGH
- Context7 `/santhosh-tekuri/jsonschema` — `Schema.Validate(map[string]any)` accepts decoded Go values; `UnmarshalJSON` uses `json.Number` for numeric precision — 2026-08-08 — HIGH
- Context7 library resolution — `/google/jsonschema-go` exists as a current alternative (benchmark 88.67 vs 88.24) — 2026-08-08 — MEDIUM
- `go list -m -versions` against the live module proxy for every version claim in the table above — 2026-08-08 — HIGH

---
*Stack research for: Aether v1.26 Intelligent Orchestration*
*Researched: 2026-08-08*
