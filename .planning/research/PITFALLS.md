# Pitfalls Research: v1.9 Review Persistence

**Domain:** Adding structured review findings persistence and domain-ledger system to an existing Go CLI framework with file-based state management
**Researched:** 2026-04-26
**Confidence:** HIGH (based on direct codebase analysis of storage layer, colony-prime injection, agent definitions, and existing patterns)

## Critical Pitfalls

### Pitfall 1: Token Budget Blowout From Prior-Review Injection

**What goes wrong:**
The colony-prime context builder (`buildColonyPrimeOutput` in `cmd/colony_prime_context.go`) already assembles 9 sections within an 8,000 character budget (4,000 in compact mode). Adding a "prior-reviews" section with open findings per domain will compete with existing sections. If 7 domains each accumulate 5-10 open findings across phases, the prior-reviews section alone could consume 2,000-3,000 characters, crowding out pheromone signals (priority 9) and blockers (priority 10) that are critical for worker guidance.

**Why it happens:**
The `RankContextCandidates` function in `pkg/colony/context_ranking.go` uses a greedy inclusion strategy sorted by composite score (0.40 relevance weight, 0.20 trust/freshness/confirmation). Prior-review content has high relevance (it is about the actual codebase) but is verbose by nature. Without strict character limits per domain or a summarization step, review findings will dominate the budget.

**How to avoid:**
- Set a hard cap on the prior-reviews section: no more than 800 characters in normal mode, 400 in compact
- Summarize at the domain level, not the individual finding level: "security: 2 critical, 3 warning" not full descriptions
- Only include unresolved findings with severity >= "warning" in the injection
- Set priority to 8 (between pheromones at 9 and clarified-intent at 8) so review content trims before pheromones and blockers
- Consider making the section eligible for trimming early by giving it a moderate freshness score that decays with phase age

**Warning signs:**
- Colony-prime output shows "trimmed" list includes pheromones or blockers when prior-reviews is present
- Workers ignore pheromone signals after review findings accumulate past phase 3
- `colony-prime` log line shows used chars near 8000 with prior-reviews taking >30% of budget

**Phase to address:**
Phase that implements colony-prime injection (the Part B integration phase). The ranking parameters must be tuned with a budget test before the feature ships.

---

### Pitfall 2: Agent Write-Scope Escape Via New Write Tool Access

**What goes wrong:**
The v1.9 plan gives 7 review agents (Gatekeeper, Auditor, Chaos, Watcher, Archaeologist, Measurer, Tracker) the Write tool so they can persist findings. Currently, 6 of these 7 agents are explicitly read-only by design. The agent definitions contain strong language like "Your constraint is absolute: you are read-only. No Write. No Edit. No Bash." (Auditor, line 12). Adding Write to these agents creates a trust boundary violation: a review agent that finds a security issue could theoretically "fix" it during a review pass, bypassing the Builder TDD workflow.

**Why it happens:**
The hook system (`protectedHookWriteReason` in `cmd/hook_cmds.go`) only protects `.aether/data/`, `.aether/dreams/`, `.env*`, `.codex/config.toml`, and `.github/workflows/`. It does not restrict writes to only approved ledger paths. A well-intentioned agent with Write access could modify source files or test files during a review pass.

**How to avoid:**
- Extend the hook system (`protectedHookWriteReason` or a new function) to enforce a write-scope whitelist for review agents: they may ONLY write to `.aether/data/reviews/`
- Add explicit write-scope instructions to each agent definition: "You may only write files under `.aether/data/reviews/`. Any write attempt outside this path will be blocked."
- Keep the existing read-only language in agent definitions but scope the exception: "You are read-only for all source code, test files, and configuration. You have write access ONLY for `.aether/data/reviews/{domain}/ledger.json`."
- Test the hook protection with a dedicated test case that attempts writes outside the reviews directory

**Warning signs:**
- Review agents modify source files during a continue pass
- Medic health check detects changed files outside build artifacts
- Audit findings reference code that was "fixed" during the audit itself

**Phase to address:**
Phase that updates agent definitions (the agent changes phase). Write-scope guardrails must ship in the same phase as the Write tool addition, not as a follow-up.

---

### Pitfall 3: JSON Backward Compatibility Break in codexContinueWorkerFlowStep

**What goes wrong:**
The v1.9 plan extends `codexContinueWorkerFlowStep` with new fields (`Blockers`, `Duration`, `Report`). The `mergeExternalContinueResults` function in `cmd/codex_continue_finalize.go` constructs flow steps by copying specific fields from the dispatch struct. If new fields are added to the struct but the merge function is not updated to copy them, the data is silently dropped. Additionally, if wrappers (`.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`) submit completion files with the new fields but the Go binary is an older version, the old binary will silently ignore the unknown JSON fields (Go's default behavior with `json.Unmarshal`). This is benign for new fields but creates a confusing state where reports exist in the completion file but not in the flow step.

**Why it happens:**
Go's `json.Unmarshal` silently ignores unknown fields by default. There is no schema version field on the completion structures. The `codexContinueExternalDispatch` struct already has `Blockers` and `Duration` fields, but `codexContinueWorkerFlowStep` does not. The merge function currently only copies `Stage`, `Caste`, `Name`, `Task`, `Status`, and `Summary` -- any new fields must be explicitly added.

**How to avoid:**
- Add all new fields to both structs simultaneously: `codexContinueExternalDispatch` and `codexContinueWorkerFlowStep`
- Update `mergeExternalContinueResults` to copy every new field from the dispatch result into the flow step
- Add a test that round-trips a completion file through `mergeExternalContinueResults` and verifies all fields survive
- Do NOT rely on JSON tag `omitempty` to mask missing fields in tests -- test with and without the new fields
- Add a `Report` field to `codexContinueExternalDispatch` so the wrapper can pass the report content through the completion pipeline

**Warning signs:**
- Continue worker reports show `Blockers: null` or `Duration: 0` when the completion file contained values
- `review.json` has summary strings but no structured blocker details
- Tests pass but the `.md` outcome reports are missing report content

**Phase to address:**
Phase that implements Part A (continue-review worker outcome reports). This is the first implementation phase and must get the struct extensions right.

---

### Pitfall 4: File Locking Contention Between Review Writes and Colony-Prime Reads

**What goes wrong:**
The domain-ledger system creates 7 JSON files under `.aether/data/reviews/{domain}/ledger.json`. During a continue pass, review agents write findings to these files via the `review-ledger-write` subcommand. Simultaneously, if colony-prime is called (e.g., for the next worker dispatch), it reads ledger files to build the prior-reviews section. The storage layer uses per-file locks (one lock file per data file in `.aether/locks/`). If a review agent holds a write lock on `reviews/security/ledger.json` while colony-prime tries to read it, colony-prime will block until the write completes. On slow file systems or with large ledgers, this could cause worker timeouts.

**Why it happens:**
The `Store.LoadJSON` method acquires a shared (read) lock via `locker.RLock`, while `Store.SaveJSON` acquires an exclusive (write) lock via `locker.Lock`. The lock granularity is per-file, so concurrent access to different domain ledgers is fine. But the `review-ledger-summary` subcommand (used by colony-prime) may need to read all 7 domain files, holding 7 shared locks. If a review agent is writing to any of them at the same time, the summary call blocks.

**How to avoid:**
- Use `UpdateFile` (which holds an exclusive lock) for `review-ledger-write` to ensure atomic read-modify-write
- For `review-ledger-summary`, read each domain file independently with `LoadRawJSON` and tolerate missing files (a domain may have no findings yet)
- Never hold locks on multiple files simultaneously -- process domains sequentially in the summary reader
- Consider caching the summary in a single `reviews/summary.json` file that is updated only when a write occurs, so colony-prime reads one file instead of seven
- Add a timeout to the colony-prime prior-reviews section: if reading takes >2 seconds, skip the section and log a warning

**Warning signs:**
- Worker dispatch times increase after phase 3 when ledgers have accumulated entries
- Colony-prime log shows "failed to read reviews" errors in stderr
- Intermittent test failures in CI due to lock contention under parallel test execution

**Phase to address:**
Phase that implements the domain-ledger Go subcommands (the ledger runtime phase). The locking strategy must be decided before writing the subcommands.

---

### Pitfall 5: Stale Review Data Accumulation Without Lifecycle Cleanup

**What goes wrong:**
Review findings accumulate across phases but have no automatic cleanup. After a 15-phase colony, each domain ledger could contain 50-100 entries. When the colony is sealed and a new colony is initialized in the same repo, the old review files persist under `.aether/data/reviews/`. The new colony's workers will see findings from the previous colony that reference files and line numbers that no longer exist.

**Why it happens:**
The v1.9 design says findings are "colony-scoped" and "archived at seal," but there is no existing mechanism for seal-phase cleanup of custom data directories. The seal command (`cmd/codex_workflow_cmds.go`) and entomb command (`cmd/entomb_cmd.go`) handle `COLONY_STATE.json`, `pheromones.json`, `instincts.json`, and session files, but there is no hook for custom data directories. The `cleanupStaleContinueReports` function only cleans `review.json` from build artifacts, not the domain-ledger files.

**How to avoid:**
- Add a cleanup step to the seal flow that archives `reviews/` to the chamber (alongside other colony data)
- When a new colony is initialized (`/ant-init`), clear any existing `reviews/` directory
- Add `reviews/` to the entomb chamber contents so review history is preserved in archives
- In `review-ledger-read`, filter by colony goal or phase range to avoid cross-colony contamination
- Consider adding a `review-ledger-purge` subcommand for manual cleanup

**Warning signs:**
- After sealing and reinitializing, workers reference findings from the old colony
- `aether status` shows review counts that include findings from a completed colony
- Ledger entries contain file paths that no longer exist in the codebase

**Phase to address:**
Phase that integrates ledger lifecycle with seal/entomb/status (the lifecycle integration phase). This must not be deferred to a later milestone.

---

### Pitfall 6: Mirror Sync Drift When Agent Definitions Change

**What goes wrong:**
The v1.9 plan updates 7 agent definitions to add Write tool and findings instructions. These changes must be reflected in 4 locations per agent: `.claude/agents/ant/`, `.aether/agents-claude/`, `.opencode/agents/`, and `.codex/agents/`. The CLAUDE.md documents that `.claude/agents/ant/*.md` is canonical and `.aether/agents-claude/*.md` is a byte-identical packaging mirror. If the Write tool addition is made to the Claude agent but the Codex TOML translation is forgotten, Codex workers will run without Write access and silently fail to persist findings. The Medic health check (`cmd/medic_wrapper.go`) validates mirror counts but not content parity.

**Why it happens:**
There are 25 agents x 4 mirror locations = 100 files to keep in sync. The Codex TOML format is structurally different from the Markdown format, requiring manual translation. The Medic check counts files but does not diff content. No CI check enforces mirror parity.

**How to avoid:**
- Update all 4 locations for each agent in a single commit
- Add a test that verifies the Write tool is present in all 4 mirrors for review agents
- Extend the Medic wrapper check to flag agents with different tool lists across mirrors
- Include mirror sync verification in the phase acceptance criteria

**Warning signs:**
- Codex continue passes show no review findings
- Medic reports mirror count mismatch (currently checks 25 Claude + 25 Codex)
- `aether update` does not propagate Write tool changes to downstream repos

**Phase to address:**
Phase that updates agent definitions. The sync must be verified as part of that phase's acceptance criteria, not deferred to a later integration phase.

---

### Pitfall 7: Race Condition Between Review Writes and Continue-Finalize Reads

**What goes wrong:**
In the external continue flow, the wrapper spawns review agents that run in parallel. Each agent is expected to write findings to the domain ledger via `review-ledger-write`. The `runCodexContinueFinalize` function then reads the worker results from the completion file. But the ledger writes happen inside the agent execution, not inside the finalize function. If an agent completes its task but the ledger write fails (disk full, permission error, lock timeout), the finalize function has no way to know. The `review.json` will show the worker "completed" but the ledger will be missing those findings.

**Why it happens:**
The current continue flow treats agent execution and data persistence as separate concerns. The `codexContinueWorkerFlowStep` captures status and summary from the completion file, but has no field to indicate whether findings were persisted. The ledger write is a side effect of agent execution, not a tracked outcome.

**How to avoid:**
- Make ledger writing the responsibility of the finalize function, not the agents. Agents return structured findings in their completion payload; finalize persists them to the ledger.
- Alternatively, if agents write directly, add a verification step in finalize that reads the ledger after all agents complete and confirms the expected number of new entries.
- Add a `findings_persisted` boolean to the worker flow step so the review report can flag incomplete persistence.
- Test the scenario where an agent completes but its ledger write fails, and verify the system degrades gracefully.

**Warning signs:**
- Ledger entry counts do not match the number of completed review workers
- Workers show "completed" status but prior-reviews section shows fewer findings than expected
- Intermittent missing findings in CI that correlate with parallel test execution

**Phase to address:**
Phase that implements the ledger write flow. The decision about who writes (agents vs finalize) must be made before implementation begins.

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Letting agents write directly to ledger files instead of routing through finalize | Simpler implementation, fewer code changes to finalize pipeline | Race conditions, untracked persistence failures, no single source of truth for "what was written" | Never -- the finalize function must own persistence |
| Using `json.Unmarshal` with silent unknown field skipping for ledger entries | No version field needed, backward compatible | Cannot detect schema mismatches, silent data loss when fields are renamed | Only if a schema version field is added simultaneously |
| Reusing the `codexContinueWorkerFlowStep` struct for report content instead of a dedicated struct | Fewer type definitions | Struct becomes a grab-bag of optional fields, unclear which fields are populated in which context | Acceptable if fields are well-documented with comments indicating when they are populated |
| Skipping mirror sync for Codex TOML in the initial phase | Faster initial delivery | Codex users get silent failures, discover the gap in production | Only if Codex is explicitly marked as "best-effort" and the phase plan documents the deferral |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Colony-prime prior-reviews injection | Reading all 7 domain ledger files synchronously, blocking the context builder | Cache a summary file (`reviews/summary.json`) updated on write, read one file in colony-prime |
| Hook system write-scope enforcement | Only protecting `.aether/data/` as a whole, not scoping review agents to `reviews/` subdirectory | Add a new write-scope check that validates the target path matches the agent's permitted write directory |
| Seal lifecycle | Forgetting to archive `reviews/` directory, leaving stale findings for the next colony | Add `reviews/` to the seal archive list alongside `pheromones.json` and `instincts.json` |
| Status command | Not showing review counts in `aether status`, making accumulation invisible to users | Add a "Reviews" row to the status output showing open findings per domain |
| `review-ledger-write` concurrency | Using `SaveJSON` (which does a full file rewrite) when multiple agents may write to the same domain | Use `UpdateFile` or `UpdateJSONAtomically` to ensure atomic read-modify-write under exclusive lock |
| Wrapper completion file | Adding `Report` field to wrapper output but not validating it in `loadExternalContinueCompletion` | Parse the `report` field in the completion loader and propagate it through `mergeExternalContinueResults` |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Unbounded ledger growth per domain | Colony-prime reads slow down after phase 5; `review-ledger-summary` takes >500ms | Cap entries per domain (e.g., 100), prune resolved entries older than the current colony, compact on resolve | Phase 5+ in a long colony (10+ phases) |
| Colony-prime reading 7 domain files on every call | Worker dispatch latency increases by 200-500ms per worker | Cache summary in a single file; invalidate cache on write | When 4+ domains have entries |
| Report content in completion file bloats the completion JSON | `loadExternalContinueCompletion` parses a large JSON blob; memory usage spikes | Strip report content to essentials in the completion file; write full report to a separate file referenced by path | When reports exceed 10KB per worker |
| Ledger ID computation (e.g., `sec-2-001`) using sequential scanning | `review-ledger-write` reads the full ledger to find the next ID, O(n) per write | Track the next ID in a metadata field at the top of the ledger JSON, increment atomically | When a domain has 50+ entries |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Review agent findings contain file paths that could leak sensitive directory structures | LOW -- findings are colony-scoped, not shared cross-colony, but could appear in chamber archives | Sanitize absolute paths to relative paths before storing in ledger entries |
| Prompt injection via malicious review finding descriptions injected into colony-prime | MEDIUM -- if an agent produces a finding with "ignore previous instructions" content, it gets injected into subsequent worker prompts | Apply the existing `SanitizeSignalContent` or `DetectPromptIntegrityFindings` to review finding descriptions before storage |
| Write-scope bypass through path traversal in agent write targets | HIGH -- an agent could write `../../COLONY_STATE.json` if write targets are not canonicalized | Canonicalize all write targets using `filepath.Abs` and validate they are under `.aether/data/reviews/` before writing |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Workers produce different review persistence behavior across Claude, OpenCode, and Codex | User sees findings on Claude but not Codex, confusion about whether reviews are working | Test all 3 platforms in the same phase; use the runtime-ledger approach (Go binary writes) instead of relying on agent-side writes |
| `aether status` shows stale review counts from a previous colony | User thinks there are unresolved issues when the colony is fresh | Clear reviews on `/ant-init`; show colony-scoped review counts only |
| Review findings reference line numbers that shift between phases | Worker tries to fix an issue at line 42 but the code has moved to line 67 | Include a "file hash" or "context snippet" in findings instead of relying on exact line numbers; consider line numbers advisory, not authoritative |

## "Looks Done But Isn't" Checklist

- [ ] **Agent Write tool added:** Often missing write-scope guardrails in agent definition body -- verify each agent has explicit "only write to reviews/" instructions AND hook enforcement
- [ ] **Ledger subcommands working:** Often missing the `UpdateJSONAtomically` call for concurrent write safety -- verify write path uses exclusive lock
- [ ] **Colony-prime injection:** Often missing budget test -- verify colony-prime output with populated reviews still fits within 8000 chars and does not trim pheromones
- [ ] **Mirror sync:** Often missing Codex TOML mirrors -- verify all 7 agents have updated TOML files with Write tool listed
- [ ] **Seal lifecycle:** Often missing reviews cleanup on seal -- verify `aether seal` archives the reviews directory
- [ ] **Report content propagation:** Often missing the `Report` field in `mergeExternalContinueResults` -- verify the merge function copies the new field
- [ ] **Deterministic IDs:** Often missing ID collision handling -- verify `sec-2-001` style IDs handle the case where the entry already exists
- [ ] **Summary caching:** Often missing cache invalidation -- verify summary.json is regenerated when any ledger changes

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Token budget blowout | LOW | Adjust priority score or add character cap in colony-prime section builder; no data migration needed |
| Agent write-scope escape | MEDIUM | Revert Write tool from affected agents; add hook enforcement; audit any files modified outside reviews/ |
| JSON backward compatibility break | MEDIUM | Add missing fields to merge function; no data migration needed since Go ignores unknown fields |
| File locking contention | LOW | Add summary caching; no data format change needed |
| Stale review data accumulation | MEDIUM | Add seal cleanup step; run `review-ledger-purge` or manually delete `reviews/` directory |
| Mirror sync drift | LOW | Update missing mirror files; run Medic check to verify counts |
| Race condition (write vs finalize) | HIGH | Refactor to make finalize own persistence; may require re-running continue passes for affected phases |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Token budget blowout from prior-reviews | Colony-prime injection phase | Budget test: colony-prime with 50 findings across 7 domains stays under 8000 chars |
| Agent write-scope escape | Agent definition update phase | Hook test: review agent attempting to write outside reviews/ is blocked |
| JSON backward compatibility break | Part A implementation (continue-review reports) | Round-trip test: completion file with all new fields survives merge |
| File locking contention | Ledger runtime subcommands phase | Concurrent write test: 3 goroutines writing to same domain simultaneously |
| Stale review data accumulation | Lifecycle integration (seal/entomb) phase | Seal test: reviews/ directory is archived and cleared on new init |
| Mirror sync drift | Agent definition update phase | Mirror parity test: all 4 locations have matching tool lists |
| Race condition (review writes vs finalize reads) | Ledger write flow design phase | Failure injection test: agent completes but ledger write fails, system degrades gracefully |

## Sources

- Direct codebase analysis of `cmd/colony_prime_context.go` (sections, priority values, budget handling)
- Direct codebase analysis of `pkg/colony/context_ranking.go` (ranking algorithm, budget enforcement)
- Direct codebase analysis of `cmd/codex_continue_finalize.go` (merge function, review report generation)
- Direct codebase analysis of `cmd/codex_continue_plan.go` (dispatch struct definitions)
- Direct codebase analysis of `pkg/storage/storage.go` (locking strategy, atomic write, SaveJSON)
- Direct codebase analysis of `pkg/storage/lock.go` (per-file locking, shared vs exclusive)
- Direct codebase analysis of `cmd/hook_cmds.go` (write protection, path normalization)
- Direct codebase analysis of `pkg/colony/sanitize.go` (content sanitization patterns)
- Direct codebase analysis of `.claude/agents/ant/aether-auditor.md`, `aether-gatekeeper.md`, `aether-watcher.md` (current read-only constraints)
- Direct codebase analysis of `cmd/codex_build.go` (existing worker report writing pattern)

---
*Pitfalls research for: Aether v1.9 Review Persistence*
*Researched: 2026-04-26*
