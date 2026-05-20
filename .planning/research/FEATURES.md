# Feature Landscape: Grounded Planning + Ceremony Restore

**Domain:** AI colony framework (Aether v1.22) -- grounded planning and ceremony restoration
**Researched:** 2026-05-18
**Confidence:** HIGH

## Table Stakes

Features users expect. Missing = plans feel generic, ceremony feels disconnected.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Survey excludes `.venv`, `__pycache__`, caches | Python projects are a primary target; current survey trusts dependency paths, producing noisy plans | Low | Both `dirsToSkip` and `shouldSkipSurveyDir` need extending; a shared filter function prevents divergence |
| Plans reference concrete repo files | "Add authentication to src/auth/middleware.ts" is actionable; "Implement authentication" is not | Medium | Requires source anchor extraction from survey + grounding validation in plan finalize |
| Discuss decisions bind into plans | Users answer `/ant-discuss` questions expecting those answers to shape the plan, not vanish | Low | `HardConstraint: true` decisions should auto-emit REDIRECT pheromones |
| Build ceremony loads from playbooks | Playbooks exist (13 files) but the TS host bypasses them; ceremony is hardcoded in lifecycle.ts | Medium | TS host needs to read playbooks and follow their step sequences |
| One execution path per workflow | Currently YAML, TS, and Go all try to be the conductor; ceremony comes from three places | Medium | Clarify ownership: playbooks = ceremony script, TS host = reader, Go = visual renderer |

## Differentiators

Features that set Aether apart from other AI coding tools.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Grounding gate warns on generic plans | Most AI planning tools just accept whatever the LLM produces; Aether validates that plans are specific to the repo | Low | Soft gate: warning, not rejection. Research/architecture phases may legitimately have no file targets |
| Shared noise filter across codegraph + colonize | Prevents the common tool problem where two subsystems have different exclusion lists | Low | Single source of truth in `pkg/codegraph/scan_filter.go` |
| Playbook-driven ceremony is user-editable | Most AI tools have hardcoded ceremony; Aether's ceremony lives in markdown that anyone can read and modify | Medium | Requires TS host to consume playbook files as execution scripts |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Hard plan rejection on missing file refs | Research phases, architecture phases, and infrastructure phases legitimately have no concrete file targets. Hard rejection would break these. | Soft warning in plan output; user decides |
| `.gitignore` parsing for noise filtering | Adds external dependency (`go-gitignore`); target repos may lack `.gitignore`; parsing is complex (negation, globs, directory-only). | Hardcoded noise directory map (25 entries covers 99%) + pheromone REDIRECT for custom exclusions |
| New constraint file for discuss decisions | The pheromone system already handles priority, TTL, dedup, and injection into worker context. A parallel mechanism creates confusion. | Auto-emit REDIRECT pheromones from `resolveDiscussQuestion` when `HardConstraint: true` |
| AST-based file classification | Overkill for distinguishing source files from dependencies. The existing codegraph regex approach handles the 80% case. | Directory-based classification: source directories have >50% source files |
| LLM-based grounding validation | Grounding is a structural check (does the plan mention concrete file paths?), not a semantic judgment. An LLM adds latency and cost for a deterministic problem. | Regex pattern matching for file references (`src/...`, `cmd/...`, `./file.ts`) |
| Configurable noise list in COLONY_STATE.json | YAGNI -- the known noise directories are stable across language ecosystems. A config file adds maintenance burden for a problem already solved. | Hardcoded defaults; pheromone REDIRECT for edge cases |
| Playbook validation schema | Playbooks are markdown with step headers, not structured data needing schema enforcement. A schema would make playbooks harder to edit. | Simple step header parsing (`### Step N:`) with fallback to full file if headers missing |

## Feature Dependencies

```
Survey noise filter (shared ScanFilter) -> Source anchor extraction
Source anchor extraction -> Plan grounding gate
Discuss decision binding (REDIRECT pheromone) -> Colony-prime injection -> Plan generation
Playbook loading in TS host -> Go ceremony adapter usage -> Ceremony restore
```

The grounding gate depends on source anchors but not on the noise filter per se (anchors work with any survey output). However, without the noise filter, anchors would be polluted with dependency files on Python projects, making the grounding gate noisy and unhelpful. So the effective dependency chain is:

```
Noise filter -> Clean anchors -> Useful grounding gate
```

## MVP Recommendation

Prioritize (Phase 1):
1. **Shared noise filter** -- immediate bug fix for Python projects; low complexity
2. **Discuss decision binding** -- low complexity, closes a user-expectation gap
3. **Plan grounding gate** -- low complexity, high value (prevents generic plans)

Then (Phase 2):
4. **Playbook-driven TS orchestration** -- medium complexity, restores ceremony
5. **Source anchor extraction** -- medium complexity, makes plans specific

Defer:
- **File-level exclusion patterns** (`.min.js`, `.d.ts`): current dirs-only filter covers the `.venv` bug; file-level patterns can be added later if users report noise from generated artifacts

## Sources

- Direct codebase analysis of `cmd/codex_colonize.go` (survey system), `pkg/codegraph/codegraph.go` (scanning), `cmd/codex_plan.go` (planning), `cmd/discuss.go` (decision capture)
- M4L fixture context from `.aether/registry.json` (Python project with `.venv` noise)
- Playbook inventory: `.aether/docs/command-playbooks/` (13 files, build + continue split)
- TS host ceremony adapter: `.aether/ts-host/src/ceremony-adapter.ts` (already functional)
