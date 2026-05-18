# Phase 137: Hive Wisdom Injection - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning

<domain>
## Phase Boundary

The TS host calls Go's `aether hive-read` command during prompt assembly to pull cross-colony wisdom into worker prompts. Workers in repo B can benefit from patterns learned in repo A. This is a read-only injection: the TS host reads wisdom and formats it for workers, Go owns the wisdom storage and retrieval.

The Go `hive-read` command already exists (cmd/hive.go) with domain filtering and confidence thresholds. The TS host needs: a new `hive-injector.ts` module, integration with the dispatch pipeline, and tests proving graceful degradation.

</domain>

<decisions>
## Implementation Decisions

### Wisdom Placement in Worker Prompts
- **D-01:** Hive wisdom appears as its own section in the worker prompt, separate from skills and colony context. The section header is "HIVE WISDOM (Cross-Colony Patterns)" placed after the skills section. Workers see four distinct layers: colony context > pheromones > skills > hive wisdom.

### Read Timing
- **D-02:** The `hive-read` call happens once per build, before the first wave dispatches. Results are cached and injected into every worker in the build. This is one subprocess call, keeping latency low. Workers in later waves see the same wisdom as wave 1.

### Presentation Format
- **D-03:** Wisdom entries are shown with structured metadata: domain tag and confidence score alongside each entry text. Workers see entries like: `(go, 0.95) Prefer table-driven tests over testify assertions`. This lets workers judge relevance to their specific task.

### Graceful Degradation
- **D-04:** First-time colonies without hive wisdom work identically to colonies with it -- no error, no empty section, no behavioral difference (HIVE-03)
- **D-05:** If `hive-read` fails (Go command error, missing hive directory, corrupted wisdom.json), dispatch proceeds normally with a warning logged to stderr. Workers never blocked by hive failures (HIVE-04)

### Claude's Discretion
- How `hive-injector.ts` is structured (standalone module vs. integrated into prompt-assembler)
- How domain tags are sourced at dispatch time (from colony registry, from manifest, or hardcoded)
- Token budget for the wisdom section (how many entries to include before truncating)
- How to handle the HIVE-05 integration test (colony B benefits from colony A's wisdom)
- Relevance discounting for partial domain matches (HIVE-07)
- Where in the dispatch pipeline the hive-read call is made (in the "dispatched" runner before dispatchWorkers, or in prompt-assembler during per-worker prompt construction)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Go Runtime (Wisdom Authority)
- `cmd/hive.go` -- hive-read, hive-store, hive-init, hive-abstract, hive-promote commands. hive-read returns JSON with entries array, accepts --domain and --min-confidence flags
- `cmd/colony_prime_context.go` -- Colony-prime context builder (prompt assembly, domain tag resolution)
- `cmd/registry.go` -- Colony registry with domain tags per repo
- `cmd/queen.go` -- Queen wisdom and user preferences

### TypeScript Host (Injection Target)
- `.aether/ts-host/src/prompt-assembler.ts` -- Prompt assembly with section injection pattern (compactSection for skills, same pattern for wisdom)
- `.aether/ts-host/src/worker-dispatch.ts` -- Worker dispatch with DispatchOptions, dispatchSingleWorker, simulateWorkers control
- `.aether/ts-host/src/host.ts` -- Host command routing, dispatched runner, callGoJSON bridge
- `.aether/ts-host/src/command-registry.ts` -- Command definitions and HostCommandRunner types
- `.aether/ts-host/src/go-bridge.ts` -- callGoJSON for subprocess communication with Go runtime
- `.aether/ts-host/src/types.ts` -- BuildDispatch, DispatchResult, and shared type definitions

### Wrapper Contracts
- `.aether/docs/wrapper-runtime-ux-contract.md` -- Wrapper/runtime boundary rules
- `.aether/docs/host-command-reference.md` -- Host command specifications

### Requirements
- `.planning/REQUIREMENTS.md` -- HIVE-01 through HIVE-07
- `.planning/ROADMAP.md` -- Phase 137 goal and success criteria

### Prior Phase Context
- `.planning/phases/136-production-foundation/136-CONTEXT.md` -- Production foundation decisions (D-01 through D-07)
- `.planning/phases/136-production-foundation/136-RESEARCH.md` -- Dispatch architecture, patterns, pitfalls

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `prompt-assembler.ts` section injection pattern: `compactSection()` handles undefined/non-string gracefully, `if (section) parts.push(section)` only adds non-empty sections. The same pattern works for wisdom injection (HIVE-03: no wisdom = no section)
- `callGoJSON()` in go-bridge.ts: already calls Go commands with typed JSON output. Used for manifests, finalizers, and skill commands. Can be used for `hive-read`
- Phase 136 dispatched runner: the `runDispatchedBuildCommand` function already has a pre-dispatch step where it checks platforms and logs skill injection. Hive injection fits naturally into this pre-dispatch phase

### Established Patterns
- Go subprocess call pattern: `callGoJSON<ReturnType>(opts, ["hive-read", "--domain", domain, "--min-confidence", "0.5"])` returns typed JSON
- Section injection pattern: new section string built, passed to prompt-assembler, only added when non-empty
- Graceful degradation pattern: try/catch around Go call, log warning, continue with empty result (already used for platform detection and ceremony rendering)

### Integration Points
- The "dispatched" runner in host.ts is where hive-read should be called (once per build, D-02)
- Domain tags come from the colony registry (cmd/registry.go) -- the TS host needs to read these or receive them in the manifest
- The prompt-assembler's `assemblePrompt` function receives a config object where `hiveSection` would be a new field alongside `skillSection`

</code_context>

<specifics>
## Specific Ideas

- User chose structured metadata format for wisdom entries -- workers see domain and confidence alongside each wisdom text, letting them judge relevance
- User chose separate section for wisdom (not folded into skills or colony-prime) -- distinct knowledge layer
- User chose read-once-per-build caching for performance -- one hive-read call, same wisdom for all workers

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope.

</deferred>

---

*Phase: 137-hive-wisdom-injection*
*Context gathered: 2026-05-18*
