# Phase 125: Wrapper Cutover — Plan/Build/Continue - Research

**Researched:** 2026-05-14
**Domain:** Markdown/YAML wrapper simplification + TS host delegation
**Confidence:** HIGH

## Summary

The Claude Code and OpenCode wrappers for `plan`, `build`, and `continue` are ~1,100 lines of manually orchestrated Go CLI commands, manifest parsing, worker spawning, and finalizer calls. The TypeScript host (`aether host <workflow>`) now exists (Phase 124) and can delegate to the Go CLI, but its individual `plan`/`build`/`continue` commands are thin JSON proxies — they do NOT handle finalizers, worker dispatch, or ceremony rendering. Only the `lifecycle` command handles full orchestration.

This phase must cut the wrappers over to delegate to the TS host while preserving the wrapper-runtime contract: wrappers own platform-specific presentation and agent spawning; the runtime (Go + TS host) owns state mutation and manifest generation. The wrappers must become materially shorter (WCO-07) by removing inline CLI command chains and replacing them with `aether host <workflow>` delegation.

**Primary recommendation:** Update all six wrapper files (Claude + OpenCode plan/build/continue), three YAML command sources, `cmd/command_guide.go`, and the Codex build-cycle skill to reference `aether host <workflow>` as the primary execution path. Keep direct Go CLI commands as documented fallback paths (WCO-06). Maintain the cross-platform drift guard by updating all four artifacts (wrapper, YAML, command_guide.go, skill) in lockstep.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Wrapper markdown generation | YAML source + generator | — | YAML is source of truth; wrappers are generated |
| Platform agent spawning | Claude/OpenCode wrapper | TS host (lifecycle) | Wrappers know platform Task/subagent APIs |
| Manifest fetching | TS host `plan`/`build`/`continue` | Go CLI `--plan-only` | TS host calls Go, returns JSON to wrapper |
| State mutation (finalize) | Go CLI finalizers | — | Go remains sole state mutator per safety invariant |
| Ceremony rendering | Go CLI `ceremony` subcommands | — | Visual output owned by Go runtime |
| Cross-platform guidance | `cmd/command_guide.go` | Codex skill | Single catalog drives all platforms |
| Fallback when TS host missing | Go CLI direct | — | WCO-06 requires direct commands still work |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| TypeScript host | 1.0 (in-repo) | Orchestration layer for plan/build/continue | Already built at `.aether/ts-host/` [VERIFIED: src/host.ts] |
| Go CLI `host` cmd | v1.0.38+ | Entry point to TS host | Added in Phase 124 [VERIFIED: cmd/host_cmd.go] |
| Node.js | >= 20 | TS host runtime | Specified in package.json engines [VERIFIED: package.json] |
| Cobra | v1.8.0 | CLI framework | Already used for all `aether` subcommands [VERIFIED: go.mod] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `aether host plan` | — | Thin proxy to `aether plan --plan-only` | Wrapper needs JSON manifest |
| `aether host build` | — | Thin proxy to `aether build --plan-only` | Wrapper needs JSON manifest |
| `aether host continue` | — | Thin proxy to `aether continue --plan-only` | Wrapper needs JSON manifest |
| `aether host lifecycle` | — | Full orchestration with finalizers | When wrapper wants full delegation (future) |

### Installation
No new dependencies. The TS host and Go CLI host command already exist.

**Version verification:**
- Node: `v25.9.0` installed locally [VERIFIED: `node --version`]
- Go CLI host command: exists and has subcommands `plan`, `build`, `continue`, `lifecycle`, `oracle` [VERIFIED: cmd/host_cmd.go]

## Architecture Patterns

### System Architecture Diagram

```
User runs: /ant-plan
                |
                v
        +-------------------+
        | Claude/OpenCode   |
        | wrapper markdown  |
        +-------------------+
                |
                v
        +-------------------+
        | aether host plan  |  <-- NEW: wrapper delegates here
        | (Go -> Node)      |
        +-------------------+
                |
        +-------+-------+
        |               |
        v               v
+---------------+  +------------------+
| TS host calls |  | Fallback: direct |
| aether plan   |  | aether plan      |
| --plan-only   |  | --plan-only      |
+---------------+  +------------------+
        |
        v
+---------------+
| JSON manifest |
| returned to   |
| wrapper       |
+---------------+
        |
        v
+---------------------------------------+
| Wrapper parses manifest, spawns       |
| platform agents (Task/subagent)       |
+---------------------------------------+
        |
        v
+---------------------------------------+
| Wrapper builds completion JSON,       |
| calls aether plan-finalize            |
| (direct Go CLI — not through host)    |
+---------------------------------------+
```

### Recommended File Changes

```
.claude/commands/ant/
├── plan.md          # MODIFY: delegate manifest fetch to `aether host plan`
├── build.md         # MODIFY: delegate manifest fetch to `aether host build`
└── continue.md      # MODIFY: delegate manifest fetch to `aether host continue`

.opencode/commands/ant/
├── plan.md          # MODIFY: same as Claude
├── build.md         # MODIFY: same as Claude
└── continue.md      # MODIFY: same as Claude

.aether/commands/
├── plan.yaml        # MODIFY: runtime.command references host path
├── build.yaml       # MODIFY: runtime.command references host path
└── continue.yaml    # MODIFY: runtime.command references host path

cmd/
└── command_guide.go # MODIFY: PreSteps/RunCommand reference host delegation

.aether/skills/colony/aether-colony-build-cycle/
└── SKILL.md         # MODIFY: plan/build/continue flows reference host
```

### Pattern 1: Wrapper Delegates Manifest Fetch to TS Host
**What:** Replace inline `AETHER_OUTPUT_MODE=json aether plan --plan-only ...` with `aether host plan --depth ...` in the wrapper.
**When to use:** All wrapper orchestration sections that need a JSON manifest.
**Example:**
```markdown
<!-- Before (current wrapper) -->
```
AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice2> $ARGUMENTS
```

<!-- After (cutover wrapper) -->
```
aether host plan --depth <choice> --planning-depth <choice2> $ARGUMENTS
```
```

### Pattern 2: Keep Direct Go CLI for Finalizers
**What:** Finalizer commands (`plan-finalize`, `build-finalize`, `continue-finalize`) remain direct Go CLI calls, not routed through the TS host.
**When to use:** All completion/finalization steps.
**Why:** The TS host's individual commands do not handle finalizers — they only proxy `--plan-only` calls. Finalizers must remain direct Go CLI calls to satisfy the safety invariant (SAFE-01: Go is sole state mutator).
**Example:**
```markdown
<!-- Finalizer stays direct -->
```
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <completion_file>
```
```

### Pattern 3: Fallback Path Documentation
**What:** Document the direct Go CLI command as a fallback when the TS host is unavailable (Node missing, assets not built).
**When to use:** In wrapper guardrails or a dedicated fallback section.
**Example:**
```markdown
## Fallback (when TS host unavailable)

If `aether host plan` fails with "node not available" or "TS host assets not found",
fall back to the direct Go CLI:

```
AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice2> $ARGUMENTS
```
```

### Pattern 4: Cross-Platform Drift Guard Update
**What:** When updating wrappers, update YAML sources, command_guide.go, and Codex skill together.
**When to use:** Every wrapper change in this phase.
**Example from current wrappers:**
```markdown
## Cross-Platform Drift Guard

If you change planning depth selection, clarification handling, worker spawning,
finalization, or closeout behavior here, update `.aether/commands/plan.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the
same change.
```

### Anti-Patterns to Avoid
- **Delegating finalizers through the TS host:** The TS host `plan`/`build`/`continue` commands do not call finalizers. Only `lifecycle` does. Delegating finalizers through the host would silently fail or require host changes outside this phase's scope.
- **Removing fallback documentation:** WCO-06 requires direct Go CLI commands still work. The wrappers must document the fallback path.
- **Updating only one platform:** Claude and OpenCode wrappers must stay in sync. Updating only Claude creates drift.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TS host full orchestration in plan/build/continue | Add finalizer logic to TS host individual commands | Keep finalizers direct Go CLI; use TS host only for manifest proxy | The `lifecycle` command already has full orchestration; individual commands are intentionally thin |
| Custom wrapper markdown generator | Hand-edit 6 wrapper files | Update `.aether/commands/*.yaml` and regenerate | YAML is the source of truth; wrappers have "Generated from ... - DO NOT EDIT DIRECTLY" headers |
| Platform-specific manifest parsing | Reimplement JSON parsing per platform | Use the same manifest schema across all platforms | `result.plan_manifest`, `result.dispatch_manifest`, `result.continue_manifest` are runtime-contracted |

**Key insight:** The wrappers' job is to shrink, not grow. The TS host handles the Go CLI invocation complexity; the wrapper handles platform-specific agent spawning. Don't reintroduce orchestration logic into the TS host's thin commands — that's what `lifecycle` is for.

## Runtime State Inventory

This phase is a wrapper refactor, not a data migration. No stored data changes.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — wrapper markdown only | None |
| Live service config | None | None |
| OS-registered state | None | None |
| Secrets/env vars | None | None |
| Build artifacts | `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md` are generated from YAML | Regenerate from updated YAML after edit |

**Nothing found in category:** All categories explicitly checked — no runtime state impact.

## Common Pitfalls

### Pitfall 1: TS Host Individual Commands Don't Handle Finalizers
**What goes wrong:** A wrapper calls `aether host plan`, gets the manifest, dispatches workers, builds a completion file, then mistakenly calls `aether host plan-finalize` — which doesn't exist. The finalizer is never called and state is not mutated.
**Why it happens:** The TS host `host.ts` only has `plan`, `build`, `continue`, `lifecycle`, and `oracle` commands. There is no `plan-finalize` subcommand in the TS host.
**How to avoid:** Keep finalizer calls as direct Go CLI commands (`aether plan-finalize`, `aether build-finalize`, `aether continue-finalize`). Only the manifest-fetching step delegates to the TS host.
**Warning signs:** Wrapper mentions `aether host plan-finalize` or similar.

### Pitfall 2: OpenCode Wrappers Drift from Claude Wrappers
**What goes wrong:** Claude wrappers are updated but OpenCode wrappers are forgotten, causing inconsistent behavior between platforms.
**Why it happens:** OpenCode files are in a separate directory and easy to miss.
**How to avoid:** Update all six wrapper files (Claude plan/build/continue + OpenCode plan/build/continue) in the same commit. Use the drift guard checklist.
**Warning signs:** File diff shows only `.claude/` changes without matching `.opencode/` changes.

### Pitfall 3: YAML Sources Not Updated
**What goes wrong:** Wrappers are updated but `.aether/commands/*.yaml` are not, so the next wrapper regeneration reverts the changes.
**Why it happens:** Wrappers have "Generated from .aether/commands/*.yaml - DO NOT EDIT DIRECTLY" headers.
**How to avoid:** Edit YAML first, then regenerate wrappers. The YAML is the source of truth.
**Warning signs:** Wrapper diff shows manual edits without corresponding YAML changes.

### Pitfall 4: Command Guide Becomes Stale
**What goes wrong:** `cmd/command_guide.go` still references the old direct CLI commands, so Codex users get outdated guidance.
**Why it happens:** command_guide.go is a separate file from wrappers and YAML.
**How to avoid:** Update `command_guide.go` `PreSteps` and `RunCommand` for `plan`, `build`, and `continue` to reference the TS host delegation pattern.
**Warning signs:** `aether command-guide plan --platform codex` returns old command strings.

### Pitfall 5: Codex Skill Drift
**What goes wrong:** The Codex skill `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` still documents the old direct CLI flows.
**Why it happens:** Skills are loaded by Codex agents and may not be visibly broken until a Codex user tries to follow them.
**How to avoid:** Update the Plan Flow, Build Flow, and Continue Flow sections in the skill to reference `aether host <workflow>` for manifest fetching.
**Warning signs:** Codex agent executes `aether plan --plan-only` instead of `aether host plan`.

## Code Examples

### Updated Wrapper Manifest Fetch (Plan)
```markdown
## Planning Manifest

Ask the Go runtime for the authoritative planning manifest via the TypeScript host:

```
aether host plan --depth <choice> --planning-depth <choice2> $ARGUMENTS
```

If the TS host is unavailable (Node missing or assets not built), fall back to:

```
AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice2> $ARGUMENTS
```

Parse `result.plan_manifest` or `result.planning_manifest`...
```

### Updated Wrapper Manifest Fetch (Build)
```markdown
## Dispatch Manifest

Ask the Go runtime for the authoritative worker plan via the TypeScript host:

```
aether host build $ARGUMENTS
```

If the TS host is unavailable, fall back to:

```
AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only
```

Parse `result.dispatch_manifest`...
```

### Updated Wrapper Manifest Fetch (Continue Heavy Review)
```markdown
## Heavy External Review

Only use this path when the user explicitly requests `--verification-depth heavy`...

1. Run via the TypeScript host:
   `aether host continue --verification-depth heavy $ARGUMENTS`

   Or fall back to direct Go CLI:
   `AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy $ARGUMENTS`
```

### Updated YAML Source (plan.yaml)
```yaml
runtime:
  command: "aether host plan --depth <fast|balanced|deep|exhaustive> --planning-depth <light|standard|deep>"
  fallback: "AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice2>"
```

### Updated command_guide.go (Plan)
```go
catalog["plan"] = commandGuideDefinition{
    // ...
    PreSteps: []string{
        "Load the aether-colony-build-cycle Codex skill.",
        "Select planning depth and decomposition depth with the user unless arguments already specify them.",
        "Run `AETHER_OUTPUT_MODE=visual aether status` for current colony context.",
        "Run `aether host plan --depth <choice> --planning-depth <choice>` to fetch the manifest via the TS host. If the TS host is unavailable, fall back to `AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice>`.",
        "Parse `result.plan_manifest` or `result.planning_manifest`...",
        // ... rest unchanged
    },
    // ...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct `aether plan --plan-only` in wrappers | `aether host plan` delegation | Phase 125 (this phase) | Wrappers become shorter, TS host owns CLI invocation |
| Direct `aether build --plan-only` in wrappers | `aether host build` delegation | Phase 125 (this phase) | Same as above |
| Direct `aether continue --plan-only` in wrappers | `aether host continue` delegation | Phase 125 (this phase) | Same as above |
| ~1,100 lines across 6 wrapper files | Target: materially shorter (WCO-07) | Phase 125 (this phase) | Reduced maintenance burden |

**Deprecated/outdated:**
- Inline CLI command chains in wrapper markdown: being replaced by TS host delegation per v1.19 roadmap
- Wrapper-owned manifest fetching logic: moving to TS host proxy

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The TS host `plan`/`build`/`continue` commands output the same JSON structure as the direct Go CLI `--plan-only` commands | Code Examples | If JSON structure differs, wrapper parsing breaks. The TS host uses `callGoJSON` which preserves Go output structure, so this is safe. |
| A2 | Wrappers should remain responsible for finalizer calls (`plan-finalize`, `build-finalize`, `continue-finalize`) | Architecture Patterns | If we later want finalizers through the host, that requires Phase 127+ work on the `lifecycle` command or new TS host subcommands |
| A3 | "Materially shorter" (WCO-07) means removing the inline CLI command strings and ceremony command strings from wrappers, not removing worker spawning logic | Summary | The wrapper still needs to spawn platform agents; the shortening comes from delegating manifest fetch to the host |
| A4 | OpenCode wrappers are identical to Claude wrappers in content | Standard Stack | Verified by reading both sets — they are byte-for-byte identical except for the directory path |

## Open Questions

1. **Should the TS host's individual commands grow to handle finalizers?**
   - What we know: Only `lifecycle` handles finalizers today. The individual commands are thin proxies.
   - What's unclear: Whether future phases (127+) will add finalizer support to individual commands.
   - Recommendation: Keep finalizers direct in this phase. Revisit when `lifecycle` matures.

2. **How much shorter should wrappers be to satisfy WCO-07?**
   - What we know: Current wrappers are ~550 lines (Claude) + ~550 lines (OpenCode) = ~1,100 total.
   - What's unclear: The quantitative threshold for "materially shorter."
   - Recommendation: Target 30-50% reduction by removing inline CLI strings and replacing multi-step orchestration with single `aether host` calls. Document line count before/after.

3. **Should the `aether host` commands accept `--plan-only` explicitly?**
   - What we know: The TS host `plan` command currently calls `aether plan --plan-only` internally.
   - What's unclear: Whether wrappers should pass `--plan-only` to `aether host plan` for clarity.
   - Recommendation: The TS host commands are plan-only by definition (they return JSON). No need to expose `--plan-only` in the wrapper call — it would be redundant.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| TS host dist | Wrapper delegation | Yes | built | `npm --prefix .aether/ts-host run build` |
| Node.js | TS host runtime | Yes | v25.9.0 | None — fallback to direct Go CLI |
| Go CLI host cmd | Wrapper delegation | Yes | v1.0.38+ | None — required |
| aether binary | All commands | Yes | v1.0.38 | `go run ./cmd/aether` |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:**
- Node.js missing on user machine: Wrapper falls back to direct Go CLI commands (documented in wrapper).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — standard `go test` |
| Quick run command | `go test ./cmd -run TestCommandGuide -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WCO-01 | Claude plan wrapper calls TS host | manual review | diff `.claude/commands/ant/plan.md` | Yes |
| WCO-02 | Claude build wrapper calls TS host | manual review | diff `.claude/commands/ant/build.md` | Yes |
| WCO-03 | Claude continue wrapper calls TS host | manual review | diff `.claude/commands/ant/continue.md` | Yes |
| WCO-04 | OpenCode wrappers call TS host | manual review | diff `.opencode/commands/ant/*.md` | Yes |
| WCO-05 | YAML sources reference host path | manual review | diff `.aether/commands/*.yaml` | Yes |
| WCO-06 | Direct Go CLI still documented | manual review | grep "fallback" in wrappers | Yes |
| WCO-07 | Wrapper files shorter after cutover | line count | `wc -l` before/after | Yes |
| REL-03 | `go test ./...` passes | integration | `go test ./...` | Yes |
| REL-05 | Cross-platform consistency | manual review | compare Claude/OpenCode/Codex skill | Yes |

### Sampling Rate
- **Per task commit:** `go test ./cmd -run TestCommandGuide -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `.aether/commands/plan.yaml` — update runtime.command to reference host
- [ ] `.aether/commands/build.yaml` — update runtime.command to reference host
- [ ] `.aether/commands/continue.yaml` — update runtime.command to reference host
- [ ] `cmd/command_guide.go` — update plan/build/continue PreSteps/RunCommand
- [ ] `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` — update plan/build/continue flows
- [ ] Wrapper regeneration script/tool — verify wrappers can be regenerated from YAML

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | — |
| V3 Session Management | No | — |
| V4 Access Control | No | — |
| V5 Input Validation | Yes | Wrapper must not pass unsanitized user args to shell; use `exec.Command` slice args |
| V6 Cryptography | No | — |

### Known Threat Patterns for Wrapper/Host Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command injection via `$ARGUMENTS` | Tampering | `$ARGUMENTS` is passed as positional args to `aether host`, which passes them as slice args to Go CLI — no shell interpolation |
| Wrapper markdown injection | Tampering | Wrappers are generated from YAML, not user-editable at runtime |
| TS host path traversal | Tampering | `resolveTsHostPath` validates candidate paths exist and are files, not directories [VERIFIED: cmd/host_cmd.go] |

## Sources

### Primary (HIGH confidence)
- `.claude/commands/ant/plan.md` — Current wrapper content (241 lines) [VERIFIED: direct read]
- `.claude/commands/ant/build.md` — Current wrapper content (182 lines) [VERIFIED: direct read]
- `.claude/commands/ant/continue.md` — Current wrapper content (127 lines) [VERIFIED: direct read]
- `.opencode/commands/ant/plan.md` — Identical to Claude wrapper [VERIFIED: direct read]
- `.opencode/commands/ant/build.md` — Identical to Claude wrapper [VERIFIED: direct read]
- `.opencode/commands/ant/continue.md` — Identical to Claude wrapper [VERIFIED: direct read]
- `.aether/commands/plan.yaml` — YAML source (59 lines) [VERIFIED: direct read]
- `.aether/commands/build.yaml` — YAML source (52 lines) [VERIFIED: direct read]
- `.aether/commands/continue.yaml` — YAML source (66 lines) [VERIFIED: direct read]
- `cmd/command_guide.go` — Codex orchestration guidance (446 lines) [VERIFIED: direct read]
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` — Codex skill (311 lines) [VERIFIED: direct read]
- `cmd/host_cmd.go` — Go CLI host command (152 lines) [VERIFIED: direct read]
- `.aether/ts-host/src/host.ts` — TS host entry point (189 lines) [VERIFIED: direct read]
- `.aether/ts-host/src/lifecycle.ts` — Full lifecycle orchestrator (487 lines) [VERIFIED: direct read]
- `cmd/safety_invariant_test.go` — Safety invariant tests (703 lines) [VERIFIED: direct read]

### Secondary (MEDIUM confidence)
- `.aether/docs/wrapper-runtime-ux-contract.md` — Wrapper-runtime contract (referenced in CLAUDE.md, not directly read)
- `.aether/docs/publish-update-runbook.md` — Publish workflow documentation (referenced in CLAUDE.md)

### Tertiary (LOW confidence)
- None — all critical claims verified by direct file reads.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all components exist and were verified by direct file read
- Architecture: HIGH — wrapper-runtime contract is well-documented; TS host capabilities are clear
- Pitfalls: HIGH — based on direct analysis of TS host source code showing individual commands are thin proxies

**Research date:** 2026-05-14
**Valid until:** 2026-06-14 (stable stack, low churn expected)
