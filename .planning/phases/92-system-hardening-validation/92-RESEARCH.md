# Phase 92: System Hardening & Validation - Research

**Researched:** 2026-05-02
**Domain:** Worker lifecycle management, context assembly audit, E2E system validation
**Confidence:** HIGH

## Summary

Phase 92 is the capstone for v1.13. It has three distinct work streams: (1) adding heartbeat-based worker liveness detection to the existing process lifecycle management, (2) auditing and ensuring colony-prime context completeness and freshness per the AAC-005 specification, and (3) writing comprehensive E2E validation tests that exercise the full v1.13 system end-to-end.

The existing infrastructure is substantial. Process group management already exists in `pkg/codex/process_group_unix.go` (Setpgid/SIGTERM/SIGKILL), PID tracking already works via `pkg/codex/process_tracker.go` with file-based persistence in `.aether/data/worker-processes.json`, and stale worker cleanup already runs before dispatch via `cmd/codex_worker_cleanup.go`. The colony-prime context assembly in `cmd/colony_prime_context.go` already assembles 15+ sections with budget-aware ranking. The gaps are narrow and well-scoped.

**Primary recommendation:** Extend existing patterns rather than build new ones. The heartbeat system adds a file-based liveness check on top of the existing process tracker. The context audit is a comparison exercise against AAC-005. The E2E tests follow the established pattern in `cmd/e2e_recovery_test.go`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** File-based heartbeat mechanism. Workers write `.aether/data/heartbeat-{worker-id}.json` at intervals (first immediately, then ~30s throttled). Driven by prompt instruction.
- **D-02:** Heartbeat writes are driven by prompt instruction, not by the Go runtime. Workers write the heartbeat file as part of their task execution.
- **D-03:** A background goroutine in the Go runtime periodically scans heartbeat files and emits warnings or auto-cleans stuck workers.
- **D-04:** Audit `buildColonyPrimeOutput()` against AAC-005 requirements to identify any missing sections. If sections are missing, add them.
- **D-05:** Colony-prime context must be assembled fresh immediately before each worker spawn -- not cached from session start.
- **D-06:** Write a single Go integration test that exercises the full v1.13 flow: init -> build -> gate failure -> unblock -> fixer -> continue -> learning capture -> hive search -> skill lifecycle -> seal cleanup.
- **D-07:** The E2E test covers the full v1.13 system, not just Phase 92 scope.
- **D-08:** Write an update round-trip test: create known agent/command files, run update flow, verify all files still exist with correct content.
- **D-09:** Round-trip test covers both agent definitions AND command files.
- **D-10:** Every new command and file format from v1.13 gets validation with actionable error messages. This includes: gate-results.json format, learning entry format, skill SKILL.md format, colony.db schema, heartbeat file format.

### Claude's Discretion
- Heartbeat file format and exact goroutine monitoring interval
- Specific missing sections in the AAC-005 audit (determined by reading code)
- E2E test structure and which specific v1.13 features to exercise in sequence
- Heartbeat cleanup behavior on session exit (part of existing worker cleanup flow)
- How heartbeat staleness threshold maps to worker timeout behavior

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SAFE-05 | Worker prompts include all v5.4 context sections: colony-prime, prompt_section, survey context, phase research, matched skills, midden/graveyard cautions (AAC-005) | Context audit section below maps all 15 current sections to AAC-005 requirements, identifies gaps |
| SAFE-06 | Context is refreshed immediately before worker spawn, not cached from session start (AAC-005) | Existing `resolveCodexWorkerContext()` already calls `buildColonyPrimeOutput()` per-dispatch; need to verify no session-level caching |
| PLAT-03 | Workers emit periodic heartbeats (first immediately, then throttled to ~30s intervals) (AAC-014) | File-based heartbeat design below, driven by prompt instruction |
| PLAT-04 | Workers spawn in managed process groups (Setpgid on Unix, stub on Windows) (AAC-015) | Already implemented in `pkg/codex/process_group_unix.go` -- verification only |
| PLAT-05 | Worker PIDs are tracked and killed on exit (SIGTERM then SIGKILL after ~2s) (AAC-016) | Already implemented in `pkg/codex/process_tracker.go` -- verification only |
| PLAT-06 | Stale workers from previous sessions are detected and cleaned before new dispatch (AAC-017) | Already implemented in `cmd/codex_worker_cleanup.go` -- heartbeat adds liveness detection |
| VAL-01 | Full smoke test from init/oracle through phase advancement with gate failure, unblock, fixer, continue, and process cleanup (AAC-018) | E2E test design below, follows `e2e_recovery_test.go` pattern |
| VAL-02 | All generated/mirrored files (agents, commands) survive aether update (AAC-018) | Update round-trip test design below |
| VAL-03 | Every new command/file format has validation and actionable errors (AAC-018) | Validation patterns section below with format schemas |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Heartbeat file writes | LLM Worker (prompt-driven) | -- | Workers are Claude/OpenCode/Codex agents that can write files per prompt instruction |
| Heartbeat monitoring goroutine | Go Runtime (cmd/) | -- | Background goroutine in Go scans heartbeat files, runs alongside dispatch |
| Process group management | Go Runtime (pkg/codex/) | -- | Already implemented: Setpgid on Unix, stub on Windows |
| PID tracking & cleanup | Go Runtime (pkg/codex/) | -- | Already implemented: ProcessTracker with file-based persistence |
| Context assembly (colony-prime) | Go Runtime (cmd/) | -- | `buildColonyPrimeOutput()` assembles 15+ sections with budget-aware ranking |
| Context freshness enforcement | Go Runtime (cmd/) | -- | `resolveCodexWorkerContext()` must call assembly per-dispatch, not cache |
| E2E smoke test | Go test (cmd/) | -- | Integration test following `e2e_recovery_test.go` pattern |
| Update round-trip test | Go test (cmd/) | -- | Install/update test following `setup_cmd_test.go` pattern |
| File format validation | Go Runtime (cmd/) | -- | Validation functions for gate-results, learning entries, skills, colony.db, heartbeat |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.26.1 | Process groups (syscall), file I/O, time, encoding/json | No external dependencies needed for heartbeat/monitoring |
| modernc.org/sqlite | (already in go.mod) | colony.db SQLite | Already used for FTS5 search and colony store |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/storage | (internal) | Atomic file writes for heartbeat files | Use existing Store.SaveJSON and Store.UpdateFile patterns |
| pkg/cache | (internal) | Session cache for data reads | 24h TTL fine for data reads; NOT for final assembly |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| File-based heartbeat | HTTP callback from worker | HTTP requires network infrastructure; file-based works across all platforms with no extra config |
| Prompt-driven heartbeat | Go-managed subprocess heartbeat | Workers are LLM agents, not traditional processes -- they can only write files, not respond to signals |
| Background goroutine monitor | On-demand stale check | Goroutine catches stuck workers faster; on-demand only catches at dispatch boundaries |

## Architecture Patterns

### System Architecture Diagram

```
                     BUILD DISPATCH
                          |
                    resolveCodexWorkerContext()
                    (fresh per-dispatch)
                          |
                    buildColonyPrimeOutput()
                    (15+ sections, ranked by budget)
                          |
         +----------------+----------------+
         |                |                |
    colony-prime     skill-inject     pheromone
    context          (matched         section
    (main budget)    skills)          (signals)
         |                |                |
         +------- Assembled into WorkerDispatch -------+
                          |
                   Worker Invoker
                   (RealInvoker / FakeInvoker)
                          |
              +-----------+-----------+
              |                       |
         Worker Process          Heartbeat Monitor
         (LLM agent)            (background goroutine)
              |                       |
     writes heartbeat file     scans heartbeat files
     per prompt instruction    detects staleness
              |                       |
              +--- .aether/data/ -----+
                    heartbeat-{id}.json
                    worker-processes.json
                          |
              ProcessTracker + CleanupStaleWorkers
              (PID tracking, SIGTERM/SIGKILL)
```

### Recommended Project Structure

```
cmd/
  heartbeat_monitor.go        # NEW: heartbeat file scan + goroutine monitor
  heartbeat_monitor_test.go   # NEW: unit tests for heartbeat detection
  colony_prime_context.go     # EXISTING: audit for AAC-005 completeness
  colony_prime_audit_test.go  # NEW: test verifying all AAC-005 sections present
  e2e_v113_test.go            # NEW: full v1.13 E2E smoke test
  update_roundtrip_test.go    # NEW: update integrity test
  validation_v113_test.go     # NEW: file format validation tests

pkg/codex/
  worker.go                   # EXISTING: already has heartbeat interval constants
  process_tracker.go          # EXISTING: already tracks PIDs and cleans up
  process_group_unix.go       # EXISTING: already has Setpgid/SIGTERM/SIGKILL
```

### Pattern 1: File-Based Heartbeat

**What:** Workers write a timestamped JSON file periodically. The Go runtime scans these files to detect stuck workers.

**When to use:** PLAT-03 -- worker liveness detection for LLM-driven agents.

**Example:**

```go
// Heartbeat file format: .aether/data/heartbeat-{worker-id}.json
// Written by the worker via prompt instruction, read by Go runtime

type HeartbeatFile struct {
    WorkerID  string `json:"worker_id"`
    Caste     string `json:"caste"`
    Timestamp string `json:"timestamp"` // RFC3339
    Phase     int    `json:"phase"`
}
```

The worker prompt includes an instruction like:

```
PERIODIC HEARTBEAT: Every ~30 seconds, write a JSON file to
.aether/data/heartbeat-{your-worker-id}.json with:
{"worker_id": "<your-id>", "caste": "<your-caste>",
 "timestamp": "<current ISO 8601>", "phase": <current phase>}
Write the first heartbeat immediately upon starting.
```

### Pattern 2: Background Goroutine Monitor

**What:** A goroutine started at build dispatch time, scans heartbeat files at intervals, and reports staleness.

**When to use:** PLAT-03 -- detecting stuck workers between dispatch checkpoints.

**Example:**

```go
// Start heartbeat monitor alongside dispatch
func startHeartbeatMonitor(ctx context.Context, dataDir string, staleThreshold time.Duration) {
    ticker := time.NewTicker(15 * time.Second) // scan interval
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            scanHeartbeatFiles(dataDir, staleThreshold)
        }
    }
}
```

### Pattern 3: E2E Integration Test

**What:** A Go test that exercises the full v1.13 flow by creating a temp directory, initializing state, and running commands in sequence.

**When to use:** VAL-01 -- comprehensive system validation.

**Example:**

```go
// Follows e2e_recovery_test.go pattern
func TestE2EV113FullFlow(t *testing.T) {
    // 1. Setup temp dir + store
    tmpDir := t.TempDir()
    dataDir := filepath.Join(tmpDir, ".aether", "data")
    // 2. Init colony state
    // 3. Build phase 1 (fake invoker)
    // 4. Trigger gate failure
    // 5. Run unblock
    // 6. Dispatch fixer
    // 7. Continue with verification
    // 8. Capture learning
    // 9. Hive search
    // 10. Skill lifecycle
    // 11. Seal + cleanup
}
```

### Anti-Patterns to Avoid

- **Heartbeat via network callback:** Workers are LLM agents on different platforms. They cannot reliably make HTTP calls. Use file-based writes that every platform supports.
- **Session-level context caching:** The entire point of SAFE-06 is fresh context per spawn. Never cache `buildColonyPrimeOutput()` result at the session level.
- **Testing only the happy path:** The E2E test must exercise failure modes (gate failure, unblock, fixer). Happy-path-only tests miss the recovery validation that is core to v1.13.
- **Over-engineering heartbeat goroutine:** A simple ticker-based scan is sufficient. Do not add priority queues, backpressure, or complex scheduling.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Process group management | Custom Setpgid/SIGTERM logic | `pkg/codex/process_group_unix.go` already implements this | Proven, tested, platform-tagged |
| PID tracking | Custom PID registry | `pkg/codex/process_tracker.go` with file persistence | Already handles TrackProcess/UntrackProcess/KillAll |
| Stale worker cleanup | New cleanup mechanism | `CleanupStaleWorkers()` already detects and kills stale processes | Already integrated into dispatch flow |
| File-based state persistence | Custom file locking | `pkg/storage.Store` with SaveJSON/LoadJSON/UpdateFile | Atomic writes, file locking, concurrent-safe |
| Context budget management | Custom trimming logic | `colony.RankContextCandidates()` with priority-based ranking | Already handles budget overflow gracefully |

**Key insight:** This phase is primarily about extending and auditing existing infrastructure, not building new systems. The process tracker, context assembler, and test patterns all exist and work. The work is adding heartbeat liveness on top of process tracking, verifying context completeness, and writing validation tests.

## Runtime State Inventory

> This is not a rename/refactor phase. Including for completeness.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | N/A (new heartbeat files in `.aether/data/`) | Create new files, no migration |
| Live service config | N/A | None |
| OS-registered state | N/A | None |
| Secrets/env vars | N/A | None |
| Build artifacts | N/A | None |

## Common Pitfalls

### Pitfall 1: Heartbeat Files Not Cleaned Up

**What goes wrong:** Heartbeat files accumulate in `.aether/data/` after workers complete. Over many builds, hundreds of stale heartbeat files remain.
**Why it happens:** The goroutine monitor detects staleness but may not clean up files from completed workers.
**How to avoid:** Clean up heartbeat files at build completion (in the build-complete flow) AND during stale worker cleanup. Add heartbeat file removal to `UntrackProcess()`.
**Warning signs:** `.aether/data/` contains many `heartbeat-*.json` files from old sessions.

### Pitfall 2: Context Budget Overflow from Missing Sections

**What goes wrong:** Adding missing AAC-005 sections (survey context, phase research, midden cautions) pushes the total context over the 8,000 char budget, causing higher-priority sections to be trimmed.
**Why it happens:** The budget is fixed and new sections compete with existing ones for space.
**How to avoid:** New sections should have appropriate priority values (lower priority means more likely to be trimmed). The ranking system already handles this correctly -- just set priorities thoughtfully. Survey context and midden cautions are informational and should have lower priority than blockers or pheromones.
**Warning signs:** `colony-prime loaded N signal(s), M instinct(s), ... used 8000/8000 chars` in logs with important sections trimmed.

### Pitfall 3: E2E Test Depends on External State

**What goes wrong:** E2E tests pass locally but fail in CI because they depend on files in `~/.aether/` or on specific platform tools being installed.
**Why it happens:** Hub directory setup, QUEEN.md, and hive wisdom are cross-repo resources that may not exist in test environments.
**How to avoid:** Follow the `setupIntegrationStore()` pattern -- create ALL needed state in a temp directory. Mock the hub path. Use FakeInvoker.
**Warning signs:** Tests fail with "hub directory not found" or "QUEEN.md not found" in CI.

### Pitfall 4: Heartbeat Prompt Instruction Ignored by Worker

**What goes wrong:** The LLM worker ignores the heartbeat instruction and never writes the file. The monitor immediately flags the worker as stuck.
**Why it happens:** LLM agents may prioritize task execution over heartbeat writes, especially for short tasks.
**How to avoid:** (1) Make the heartbeat instruction prominent in the worker prompt. (2) Set the staleness threshold generously (e.g., 2-3x the expected interval). (3) The monitor should warn first, not kill immediately. (4) Do not gate dispatch success on heartbeat presence -- it is a liveness signal, not a correctness requirement.
**Warning signs:** Heartbeat monitor reports all workers as stuck immediately after dispatch.

### Pitfall 5: AAC-005 Audit Confuses Context Capsule with Colony-Prime

**What goes wrong:** The audit finds sections missing from colony-prime that actually exist in the separate context capsule, or vice versa.
**Why it happens:** Worker context is assembled from TWO separate calls: `buildColonyPrimeOutput()` (colony-prime section) and `buildContextCapsuleOutput()` (context capsule section). Some AAC-005 sections may be in the capsule, not colony-prime.
**How to avoid:** Audit both functions. The full worker context is the combination of colony-prime + skill section + pheromone section + task brief. AAC-005 refers to the complete context, not just colony-prime.
**Warning signs:** Audit reports "midden cautions missing" when they actually exist in the context capsule.

## Code Examples

### Heartbeat File Format (NEW)

```json
// .aether/data/heartbeat-{worker-id}.json
{
  "worker_id": "Hammer-23",
  "caste": "builder",
  "timestamp": "2026-05-02T14:30:00Z",
  "phase": 1
}
```

### Heartbeat Monitor Goroutine Pattern

```go
// Source: based on existing pkg/codex/worker.go heartbeat interval pattern
func startHeartbeatMonitor(ctx context.Context, dataDir string) {
    ticker := time.NewTicker(15 * time.Second)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                entries, _ := os.ReadDir(dataDir)
                for _, e := range entries {
                    if !strings.HasPrefix(e.Name(), "heartbeat-") {
                        continue
                    }
                    // Read and check staleness
                    var hb HeartbeatFile
                    path := filepath.Join(dataDir, e.Name())
                    data, err := os.ReadFile(path)
                    if err != nil {
                        continue
                    }
                    if err := json.Unmarshal(data, &hb); err != nil {
                        continue
                    }
                    ts, _ := time.Parse(time.RFC3339, hb.Timestamp)
                    if time.Since(ts) > 90*time.Second { // 3x the ~30s interval
                        emitVisualProgress(fmt.Sprintf(
                            "Heartbeat stale for worker %s (last seen %v ago)",
                            hb.WorkerID, time.Since(ts).Round(time.Second)))
                    }
                }
            }
        }
    }()
}
```

### Heartbeat Prompt Instruction

```markdown
## Heartbeat Protocol

You MUST write a heartbeat file every ~30 seconds while working:
1. Write to `.aether/data/heartbeat-{worker-id}.json` immediately upon starting
2. Continue writing every ~30 seconds during your work
3. The file should contain: `{"worker_id":"<your-id>","caste":"<your-caste>","timestamp":"<ISO 8601>","phase":<phase>}`

Example: `echo '{"worker_id":"Hammer-23","caste":"builder","timestamp":"2026-05-02T14:30:00Z","phase":1}' > .aether/data/heartbeat-Hammer-23.json`
```

### Context Freshness Verification

```go
// Source: existing cmd/colony_prime_context.go
// resolveCodexWorkerContext() already calls buildColonyPrimeOutput() each time.
// Verify it is not cached at session level:

func resolveCodexWorkerContext() string {
    context := strings.TrimSpace(buildColonyPrimeOutput(true).PromptSection)
    // buildColonyPrimeOutput is called fresh each time -- no caching.
    // The session cache (sc) inside is for DATA READS, not for the assembly result.
    if context == "" {
        context = buildContextCapsuleOutput(true, 8, 3, 2, 220).PromptSection
    }
    // ...
}
```

### Existing Colony-Prime Sections (AUDIT REFERENCE)

```go
// Source: cmd/colony_prime_context.go buildColonyPrimeOutput()
// Current sections assembled (15 total):
// 1.  "state"              - Colony State (priority 5)
// 2.  "review_depth"       - Review Depth (priority 6)
// 3.  "pheromones"         - Pheromone Signals (priority 9)
// 4.  "instincts"          - Active Instincts (priority 6)
// 5.  "decisions"          - Key Decisions (priority 3)
// 6.  "learnings"          - Phase Learnings (priority 2)
// 7.  "hive_wisdom"        - HIVE WISDOM (priority 4)
// 8.  "learned_memory"     - Learned Memory (priority 5)
// 9.  "global_queen_md"    - Global Queen Wisdom (priority 5)
// 10. "user_preferences"   - User Preferences (priority 7)
// 11. "prior_reviews"      - Prior Reviews (priority 8)
// 12. "local_queen_wisdom" - Local Queen Wisdom (priority 5)
// 13. "clarified_intent"   - Clarified Intent (priority 8)
// 14. "blockers"           - Active Blockers (priority 10)
// 15. "medic_health"       - Colony Health Issues (priority 9)
//
// AAC-005 requires: colony-prime, prompt_section, survey context,
//                   phase research, matched skills, midden/graveyard cautions
//
// MAPPING:
// - colony-prime:     -> "state" + all ranked sections
// - prompt_section:   -> result.PromptSection (same as Context)
// - survey context:   -> NOT in colony-prime; in buildContextCapsuleOutput() or codex_dispatch_contract.go
// - phase research:   -> NOT in colony-prime; read from .aether/data/phase-research/phase-N-research.md
// - matched skills:   -> NOT in colony-prime; separate SkillSection in WorkerDispatch
// - midden cautions:  -> NOT in colony-prime; midden data exists in context capsule only (context.go line 817)
// - graveyard cautions: -> NOT in colony-prime; graveyard data in context capsule only
//
// GAPS:
// survey context, phase research, midden/graveyard cautions are handled OUTSIDE
// colony-prime (in separate dispatch paths). The audit must verify that these
// reach the worker prompt through the combined assembly, not just colony-prime.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Worker cleanup at dispatch only | Process group + PID tracking + stale detection | v1.12+ | Workers are reliably cleaned up; heartbeat adds liveness detection |
| Session-cached context | Fresh context per dispatch | v1.13 (this phase) | Workers always get current colony state, not stale data |
| Fragmented validation | Per-command error messages | v1.13 (this phase) | Every file format has clear, actionable validation errors |

**Deprecated/outdated:**
- `verification_process_group_unix.go` in cmd/ was the original process group management. Now superseded by `pkg/codex/process_group_unix.go` which is used by the worker invoker.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Survey context, phase research, matched skills, and midden/graveyard cautions already reach workers through the combined assembly (colony-prime + context capsule + skill section + task brief), not through colony-prime alone | Context Audit | May need to add these sections to colony-prime, increasing budget pressure |
| A2 | The existing `resolveCodexWorkerContext()` is called per-dispatch and not cached at session level | SAFE-06 | May need to refactor to ensure freshness |
| A3 | PLAT-04 and PLAT-05 are already fully implemented and need only verification tests, not new code | PLAT-04/05 | May discover gaps requiring implementation |
| A4 | Heartbeat prompt instructions will be followed by Claude/OpenCode/Codex agents at roughly the requested interval | PLAT-03 | LLM agents may ignore or delay heartbeat writes |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions (RESOLVED)

1. **AAC-005 section delivery path** — RESOLVED: Audit test in 92-02 Task 1 verifies the combined assembly path (colony-prime + context capsule + skill-inject + task brief) includes all AAC-005 required sections.
   - What we know: Colony-prime assembles 15 sections. Survey context, phase research, matched skills, and midden/graveyard cautions are delivered through separate paths (context capsule, skill-inject, build brief).
   - Resolution: Verify combined prompt, not just colony-prime. Plan 92-02 Task 1 tests this explicitly.

2. **Heartbeat staleness threshold** — RESOLVED: 90s warning threshold (3x interval), configurable. Implemented in 92-01 Task 1.
   - What we know: Workers should write every ~30s. LLM agents may not follow precisely.
   - Resolution: 90s warning, auto-cleanup integrated into existing worker cleanup flow. Both configurable via constants.

3. **E2E test scope vs. runtime** — RESOLVED: Single test with FakeInvoker per 92-03 Task 1.
   - What we know: Full v1.13 flow involves init, build, gate-fail, unblock, fixer, continue, learn, hive-search, skill, seal. Some of these require external tooling.
   - Resolution: Single `TestE2EV113FullFlow` with FakeInvoker mocking all external dependencies.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- all changes are internal Go code and tests)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none -- see Wave 0 |
| Quick run command | `go test ./cmd/... -run TestE2EV113 -v -count=1 -timeout 120s` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SAFE-05 | Colony-prime includes all AAC-005 sections | unit | `go test ./cmd/... -run TestColonyPrimeAAC005Audit -v` | Wave 0 |
| SAFE-06 | Context is fresh per-dispatch | unit | `go test ./cmd/... -run TestContextFreshness -v` | Wave 0 |
| PLAT-03 | Heartbeat files written and monitored | unit | `go test ./cmd/... -run TestHeartbeatMonitor -v` | Wave 0 |
| PLAT-04 | Workers spawn in process groups (Unix) | unit | `go test ./pkg/codex/... -run TestProcessGroup -v` | EXISTS |
| PLAT-05 | Worker PIDs tracked and cleaned up | unit | `go test ./pkg/codex/... -run TestProcessTracker -v` | EXISTS |
| PLAT-06 | Stale workers detected and cleaned | unit | `go test ./cmd/... -run TestStaleWorkerCleanup -v` | EXISTS |
| VAL-01 | Full v1.13 E2E smoke test | integration | `go test ./cmd/... -run TestE2EV113FullFlow -v` | Wave 0 |
| VAL-02 | Update round-trip preserves files | integration | `go test ./cmd/... -run TestUpdateRoundTrip -v` | Wave 0 |
| VAL-03 | File format validation | unit | `go test ./cmd/... -run TestV113Validation -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run TestHeartbeat -v -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/heartbeat_monitor_test.go` -- covers PLAT-03
- [ ] `cmd/colony_prime_audit_test.go` -- covers SAFE-05
- [ ] `cmd/context_freshness_test.go` -- covers SAFE-06
- [ ] `cmd/e2e_v113_test.go` -- covers VAL-01
- [ ] `cmd/update_roundtrip_test.go` -- covers VAL-02
- [ ] `cmd/validation_v113_test.go` -- covers VAL-03

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Workers are dispatched by trusted runtime, not user-authenticated |
| V3 Session Management | no | No user sessions involved |
| V4 Access Control | no | Workers operate within the repo boundary |
| V5 Input Validation | yes | Heartbeat file parsing, gate-results validation, learning entry validation |
| V6 Cryptography | no | No encryption needed for local heartbeat files |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious heartbeat file injection | Tampering | Validate JSON structure and timestamp before processing; ignore malformed files |
| Heartbeat file path traversal | Tampering | Use filepath.Base() on worker ID; never trust user input in file paths |
| Stale process killing wrong PID | Denial of Service | Already mitigated by `isKnownWorkerProcess()` which checks command name before killing |

## Sources

### Primary (HIGH confidence)
- `cmd/colony_prime_context.go` -- verified 15 sections assembled with budget-aware ranking [VERIFIED: codebase grep]
- `pkg/codex/worker.go` -- verified WorkerConfig, FakeInvoker, RealInvoker, heartbeat interval constants [VERIFIED: codebase read]
- `pkg/codex/process_tracker.go` -- verified TrackProcess, UntrackProcess, KillAll, CleanupStaleWorkers [VERIFIED: codebase read]
- `pkg/codex/process_group_unix.go` -- verified Setpgid, SIGTERM, SIGKILL [VERIFIED: codebase read]
- `cmd/codex_worker_cleanup.go` -- verified cleanupStaleWorkersBeforeDispatch [VERIFIED: codebase read]
- `cmd/e2e_recovery_test.go` -- verified E2E test pattern with temp dir setup [VERIFIED: codebase read]
- `cmd/verification_process_group_unix.go` -- verified configureVerificationCommandProcessGroup [VERIFIED: codebase read]

### Secondary (MEDIUM confidence)
- `.planning/research/PITFALLS.md` -- AAC-005 budget overflow risk, AAC-014-017 PID recycling risk [CITED: project research]
- `.planning/research/ARCHITECTURE.md` -- Phase group assignments for worker lifecycle and hardening [CITED: project architecture]

### Tertiary (LOW confidence)
- None -- all claims verified against codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no external dependencies; all internal packages verified in codebase
- Architecture: HIGH - existing infrastructure fully mapped; gaps are narrow and well-scoped
- Pitfalls: HIGH - based on verified code patterns and existing test infrastructure

**Research date:** 2026-05-02
**Valid until:** 2026-06-02 (stable -- no fast-moving external dependencies)
