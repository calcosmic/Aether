# Wrapper-Runtime UX Contract

Updated: 2026-05-11

This contract defines the ownership boundary between the Go runtime and the
platform wrapper layer (Claude Code, OpenCode). It ensures wrappers enhance
presentation and execute platform-only Task-tool spawns without duplicating
runtime planning logic or drifting from runtime truth.

Use the Go `aether` CLI as the source of truth for runtime behavior. Wrappers
add presentation, pacing, and platform-owned Task-tool execution on top of that
runtime contract.

## Runtime Surface (Go — Authoritative)

The Go runtime owns ALL of the following. Wrappers must never replicate them:

### State Management
- Colony state transitions (READY → EXECUTING → BUILT → COMPLETED)
- Phase advancement and gating
- Session tracking and freshness detection
- File locking and concurrent access

### Build/Continue/Seal Workflow
- Build manifest generation (`cmd/codex_build.go`)
- Worker dispatch planning and wave allocation
- External worker finalization:
  - Build: `aether build-finalize <phase> --completion-file <file>`
    (`cmd/codex_build_finalize.go`)
  - Continue: `aether continue-finalize --completion-file <file>`
    (`cmd/codex_continue_finalize.go`)
  - Seal: `aether seal-finalize --completion-file <file>`
    (`cmd/seal_final_review.go`)
- Verification step execution
- Gate evaluation (security, quality, coverage, performance)
- Claims collection and persistence
- Continue verification, signal housekeeping, and advancement
- Seal final review, finding persistence, and Crowned Anthill completion

### Visual Rendering
- All banner, progress bar, and status formatting (`cmd/codex_visuals.go`)
- ANSI color handling and terminal capability detection
- Caste emoji/color maps and identity rendering
- Spawn plan visualization
- Phase/task status display

### Structured Output
The runtime exposes two output modes:

| Mode | Env Var | Consumer | Content |
|------|---------|----------|---------|
| JSON | `AETHER_OUTPUT_MODE=json` | Machines, tests | Structured data envelopes |
| Visual | `AETHER_OUTPUT_MODE=visual` | Terminals, humans | Formatted ANSI output |

JSON output includes: state, phases, tasks, dispatches, verification results,
gates, claims, housekeeping, blockers, and next-step suggestions.

Visual output is the JSON data rendered through `codex_visuals.go` functions.

### Provider Availability Diagnostics

The runtime owns provider availability preflight before real worker dispatch.
The preflight reports whether the selected Codex, Claude, or OpenCode-compatible
CLI exists and appears authenticated. It does not guarantee that the launched
worker request will later pass provider API/auth checks.

Wrappers and Codex skills may surface only the sanitized provider, cause, and
next action returned by the runtime. They must not expose raw provider
stdout/stderr, tokens, or auth probe output. If a worker process starts and then
returns provider API/auth output instead of worker claims JSON, describe it as a
post-launch provider/API/auth failure and keep final state handling in the
runtime/finalizer path.

### Orchestrator Boundary Guidance

In Orchestrator Mode, plan-only lifecycle commands for `plan`, `build`, heavy
external-review `continue`, and `seal` may return
`orchestrator_boundary_guidance` in the top-level result and matching manifest.
Wrappers must inspect that object before rendering spawn ceremonies, spawning
workers, or preparing finalizer packets.

If `orchestrator_boundary_guidance.active` is true, or its `next` value is
`aether discuss`, wrappers must stop the lifecycle flow, show the runtime
summary, route to `aether discuss`, and tell the user to rerun
`after_discuss_next` after answers are resolved. After the answer is resolved,
wrappers must request a fresh plan-only manifest and must not reuse the
pre-discuss manifest.

## Wrapper Additions (Markdown — Enhancement)

Wrappers MAY add the following on top of runtime output:

### Pre-Build Context
- Colony atmosphere (Queen persona, ant metaphor)
- Current phase context (what we're building and why)
- Pheromone signal summary (what guidance is active)
- Historical context (what previous phases accomplished)

### During-Build Narration
- Explaining what workers are doing in plain language
- Status updates with colony framing
- Noting when workers encounter issues

### Task-Tool Execution Bridge
- Requesting the lifecycle dispatch manifest:
  - Colonize: `aether host colonize`
  - Plan: `aether host plan --depth <choice> --planning-depth <choice>`
  - Build: `aether host build <phase>`
  - Continue (default): `AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard`
  - Continue (classic/heavy review): `aether host continue --classic-ceremony`
  - Seal: `aether host seal`
- Honoring `orchestrator_boundary_guidance` before any lifecycle worker spawn
- Spawning Claude/OpenCode agents from `result.dispatch_manifest`
- Recording live visibility with `aether spawn-log` and `aether spawn-complete`
- Sending terminal worker results back through the matching finalizer:
  - Build: `aether build-finalize <phase> --completion-file <file>`
  - Continue: `aether continue-finalize --completion-file <file>`
  - Seal: `aether seal-finalize --completion-file <file>`

`colonize` and `seal` are TS-host orchestration targets. Their wrappers still
return worker completion packets to Go finalizers; TypeScript only fetches the
manifest and forwards supported flags.

### Post-Build Summary
- What was accomplished in colony terms
- Key decisions made during the phase
- What the verification found
- What the next phase will address

### Follow-Up Guidance
- Suggesting pheromone signals for steering
- Recommending focus areas for upcoming phases
- Highlighting risks or blockers the user should know about

## Wrapper Anti-Patterns (Prohibited)

Wrappers MUST NOT:

1. **Mutate state files** — Never write to COLONY_STATE.json, session.json,
   pheromones.json, or any file in `.aether/data/` directly.

2. **Replay runtime logic** — Never duplicate build dispatch planning,
   verification sequencing, gate evaluation, or phase advancement logic.
   Wrappers may spawn the exact workers from `dispatch_manifest`; they must not
   invent the worker mix.

3. **Parse visual text as truth** — Never scrape ANSI-formatted output to
   extract state information. Use JSON mode if programmatic data is needed.

4. **Duplicate verification** — Never re-implement test running, security
   scanning, coverage analysis, or quality gating.

5. **Override runtime routing** — Never contradict the runtime's next-step
   suggestions. Wrappers may suggest alternatives but must present the
   runtime's recommendation first.

6. **Add unsanctioned menus** — Never create option menus or recovery paths
   that don't come from the runtime itself.

7. **Own boundary questions** — Never ask, answer, or store Orchestrator
   boundary questions in wrapper markdown or chat-only state. Route through
   `aether discuss`, then request a fresh manifest before continuing.

8. **Expose provider secrets or raw output** — Never paste provider stdout,
   stderr, tokens, or auth probe output into wrapper summaries or generated
   context. Use the runtime's sanitized provider/cause/next-action wording.

## Codex Platform

In Codex, pick an ant skill to enter a workflow. The skill asks the Go program
what is allowed and follows its answer; it does not decide that work passed.

Exactly nine public skills are supported: `$ant-init`, `$ant-discuss`,
`$ant-oracle`, `$ant-colonize`, `$ant-plan`, `$ant-build`, `$ant-continue`,
`$ant-swarm`, and `$ant-seal`. The remaining 55 action skills and native-worker
parity belong to later phases. Runtime IDs, `aether` executable routes, owner
answers, and automatic-learning limits keep their existing meaning.

- `aether command-guide <command> --platform codex` remains authoritative.
- The public `ant-<command>/SKILL.md` reads `../support/<helper>.md` relative to
  its installed location. The three private bodies are
  `support/aether-colony-creation.md`, `support/aether-colony-research.md`, and
  `support/aether-colony-build-cycle.md`; none is a public helper skill.
- Runtime commands own state mutation, dispatch manifests, verification,
  persistence and finalizers. Codex skills coordinate only the accepted work.
- Literal passthrough commands (including `aether status` and `aether update`)
  execute exactly first. Raw, exact, no-interview and no-orchestration requests
  bypass the coordination layer. Preserve executable arguments and machine IDs.
- Visual UX remains in `cmd/codex_visuals.go`. Help and Next Up display dollar
  skills only for the nine supported actions; unsupported actions and
  environment-prefixed shell commands retain CLI spelling.
- Agents keep their internal IDs and `.codex/agents/*.toml` definitions. Worker
  skill injection remains in-process and separate from the public menu.

Install/publish/update share the versioned `system/codex-skills` payload contract,
with discovery at the qualified `~/.codex/skills/aether/` root. Copied or removed
skills need a fresh session; unchanged assets do not. Custom skills and project
documents retain their ownership boundaries, including under `--force`.
See the [publish/update runbook](publish-update-runbook.md#codex-skill-upgrade-and-recovery)
for payload validation, dev isolation, failed/interrupted recovery limits, and
named proof commands. Combined update claims require the Plan 03/06 merge checks;
Plan 07 owns the all-nine fresh-client receipt. Naming does not establish full
lifecycle or native-worker capability parity.

## Source Chain

```
.aether/commands/*.yaml          ← Source definitions (name, runtime command, guardrails)
    ↓ (generation)
.claude/commands/ant/*.md        ← Claude Code wrappers
.opencode/commands/ant/*.md      ← OpenCode wrappers
    ↓ (delegation)
.aether/skills/colony/aether-colony-*/SKILL.md ← private support sources
cmd/codex_skill_surface.go       ← nine public skills + versioned payload
hub/system/codex-skills/         ← public SKILL.md files + private support/*.md
aether command-guide              ← Runtime-readable orchestration contract
cmd/codex_*.go                   ← Go runtime (authoritative execution)
cmd/codex_visuals.go             ← Visual renderer (authoritative presentation)
```

The current repo does not check in a wrapper generator. The maintained contract
is YAML-backed manual sync plus automated parity and provenance tests.

## Delivering the screen to the owner

The runtime draws the screen (banners, spawn plans, closeouts, status
dashboards) through `cmd/codex_visuals.go` and `cmd/ceremony_cmd.go`, but a
screen the runtime drew inside a tool call reaches the owner only if the
wrapper relays it. Prior releases left this implicit, and the owner reported
seeing nothing where a screen should have appeared
(`.planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md` §3).
Release 1.0.88 makes the relay a declared, tested requirement rather than an
assumption:

- Every wrapper step that runs `AETHER_OUTPUT_MODE=visual` carries two
  sentences nearby, byte-identical across the wrapper and its owning YAML's
  `guardrails:` list (see `relayCardSentence` and `screenFormatSentence` in
  `cmd/seal_wrapper_accuracy_test.go`): relay the runtime's own output
  unchanged, and show it in a fenced text block from the first banner line
  (`━━`) to the end, leaving out running commentary above it and adding at
  most two short sentences of the assistant's own after.
- This is the program's own rendered screen, and it must be shown. It is a
  different thing from raw provider stdout/stderr (anti-pattern 8 above),
  which must never be pasted to the owner regardless of this rule.
- `TestEveryWrapperThatDrawsAScreenRelaysIt` (`cmd/seal_wrapper_accuracy_test.go`)
  is the structural guard: it globs every wrapper on both platforms, fails on
  any `AETHER_OUTPUT_MODE=visual` step lacking the relay pair nearby, and
  fails if any wrapper still carries an instruction that told the assistant
  to compress or paraphrase the screen instead of showing it.

## Enforcement

- Tests in `cmd/codex_visuals_test.go` verify visual output correctness
- `TestCodexAntSkillDisplayRoutes`, `TestCodexAntSkillGuideSupport` and
  `TestCodexAntSkillGuidesPreserveOtherPlatforms` bind public spelling and private
  support to unchanged executable identities and other-platform behavior
- YAML source files in `.aether/commands/` define wrapper boundaries
- `source-of-truth-map.md` documents the ownership hierarchy
- CLAUDE.md and CODEX.md reference this contract

## References

- `.aether/docs/source-of-truth-map.md` — Authority hierarchy
- `cmd/codex_visuals.go` — Visual rendering implementation
- `cmd/codex_build.go` — Build workflow implementation
- `cmd/codex_build_finalize.go` — Build finalizer implementation
- `cmd/codex_continue.go` — Continue workflow implementation
- `cmd/codex_continue_finalize.go` — Continue finalizer implementation
- `cmd/seal_final_review.go` — Seal final review and finalizer implementation
- `.aether/commands/build.yaml` — Build wrapper source definition
- `.aether/commands/continue.yaml` — Continue wrapper source definition
- `.aether/commands/seal.yaml` — Seal wrapper source definition
