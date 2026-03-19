# Codebase Concerns

**Analysis Date:** 2026-03-19

## Tech Debt

**Single Monolithic Utility Script:**
- Issue: `.aether/aether-utils.sh` is 10,249 lines — largest single file in the system
- Files: `.aether/aether-utils.sh`
- Impact: Difficult to navigate, test, and maintain; all 150 subcommands in one file creates change risk
- Fix approach: Split into domain-specific modules (state-management.sh, pheromone-management.sh, wisdom-management.sh, etc.) with clear boundaries and single responsibility

**Inconsistent Error Code Usage:**
- Issue: Some `json_err` calls use hardcoded strings instead of error constants (e.g., "Failed to add flag" vs. `$E_JSON_INVALID`)
- Files: `.aether/aether-utils.sh` (lines 74, 814, 856, 899, 930, 933, 1758+, 2947)
- Impact: Inconsistent error handling; callers cannot programmatically detect error types
- Fix approach: Standardize all error calls to use `json_err "$E_*" "message"` pattern; audit all 150 subcommands for compliance

**Incomplete Error Code Documentation:**
- Issue: Error code constants exist (E_JSON_INVALID, E_LOCK_FAILED, etc.) but aren't documented
- Files: `.aether/utils/error-handler.sh`, `.aether/aether-utils.sh` (lines 33-47)
- Impact: Developers don't know which error codes to use for new commands
- Fix approach: Create `.aether/docs/error-codes.md` documenting all E_* constants, when to use each, and their exit codes

**Missing Entrypoint for JSON Processing Failure:**
- Issue: Fallback `json_err` in `.aether/aether-utils.sh` (lines 69-79) is used if error-handler.sh fails to load
- Files: `.aether/aether-utils.sh:69-79`
- Impact: If error-handler.sh is corrupted or missing, error codes are lost and callers cannot parse responses
- Fix approach: Ensure error-handler.sh is always available; add validation on load; consider pre-parsing validation

**Activity Logging Not Atomic:**
- Issue: Activity log appends happen outside atomic write boundary; log could be truncated if process dies
- Files: `.aether/aether-utils.sh:1353-1401`
- Impact: Activity log entries lost on crash; difficult to debug multi-session issues
- Fix approach: Use atomic write pattern for log appends; rotate logs on size threshold

---

## Known Bugs

**Hardcoded Error Code in flag-acknowledge:**
- Severity: MEDIUM
- Symptom: Uses hardcoded string instead of `$E_VALIDATION_FAILED`
- Files: `.aether/aether-utils.sh:930`
- Fix: Change to `json_err "$E_VALIDATION_FAILED" "Usage: ..."`

**Missing Fallback for Lock Release on JSON Validation:**
- Severity: MEDIUM
- Symptom: If JSON validation fails in atomic_write, temp file cleaned but lock not released
- Files: `.aether/utils/atomic-write.sh:66`
- Impact: Lock remains held if caller had acquired it; subsequent operations hang
- Fix: Document lock ownership contract clearly; add force-unlock as safety valve (IMPLEMENTED in Phase 16)

---

## Security Considerations

**Git Operations with execSync:**
- Risk: Git commands use execSync with file arguments; shell injection possible if filenames contain shell metacharacters
- Files: `.bin/lib/update-transaction.js` (lines with execSync and git commands)
- Current mitigation: File arguments quoted (`"${file}"`)
- Recommendations: Use array arguments instead of string concatenation; pass array to execSync with shell: false

**Checkpoint Stashing Allows User Data Loss:**
- Risk: Build checkpoint could stash user work (TO-DOs.md, dreams, Oracle specs) before allowlist was implemented
- Files: `.aether/data/checkpoint-allowlist.json`, `build.md` (Claude and OpenCode)
- Current mitigation: Explicit allowlist system implemented in Phase 16; user data never touched
- Recommendations: Document allowlist in user guide; add warning if user files present during checkpoint

**XML Processing Without Validation:**
- Risk: XML files processed via pheromone-xml.sh, wisdom-xml.sh without schema validation
- Files: `.aether/exchange/pheromone-xml.sh`, `.aether/exchange/wisdom-xml.sh`
- Current mitigation: xsd-validate subcommand available
- Recommendations: Always validate before processing; reject invalid XML early in pipeline

**No Rate Limiting on Lock Retries:**
- Risk: acquire_lock retries 100 times with 500ms interval (50 second total); no backoff or max CPU safeguard
- Files: `.aether/utils/file-lock.sh:22-23`
- Impact: Aggressive polling could waste CPU on slow filesystems
- Recommendations: Implement exponential backoff; add jitter to prevent thundering herd

---

## Performance Bottlenecks

**Synchronous JSON Parsing in Hot Path:**
- Problem: Every context-update operation parses COLONY_STATE.json via jq; no caching
- Files: `.aether/aether-utils.sh:1105-1145` (context-update), `.aether/aether-utils.sh:251` (acquire_lock)
- Cause: jq invoked for every field access; no in-memory state cache
- Improvement path: Load state once at command start; cache in-memory for duration of operation; write back atomically at end

**Spawn Tree Not Rotated:**
- Problem: `spawn-tree.txt` grows unbounded; tail -f becomes slow after many sessions
- Files: `.aether/data/spawn-tree.txt`, `.aether/aether-utils.sh:402-448`
- Cause: Entries appended every spawn; no rotation policy
- Improvement path: Implement spawn-tree rotation with timestamp archives (PARTIALLY FIXED in Phase 18-01: `_rotate_spawn_tree` added but may need tuning)

**Activity Log Unbounded Growth:**
- Problem: `activity.log` (134KB observed) grows with every command; no size limit or rotation
- Files: `.aether/data/activity.log` (134KB), `.aether/aether-utils.sh:1353-1401`
- Cause: Append-only log with no cleanup policy
- Improvement path: Rotate logs on size threshold (e.g., 5MB); compress old logs; implement retention policy (e.g., keep 10 rotated files)

**No Connection Pooling for External Services:**
- Problem: Each pheromone-display, wisdom-read call may make HTTP requests; no pooling or caching
- Files: `.aether/exchange/pheromone-xml.sh`, `.aether/exchange/wisdom-xml.sh`
- Impact: Slow for repeated calls; no resilience to transient failures
- Improvement path: Implement caching layer; add retry logic with exponential backoff

**State File Corruption Risk on Disk Full:**
- Problem: If disk fills during atomic write, temp file may not be cleaned; lock may not be released
- Files: `.aether/utils/atomic-write.sh:61-86`
- Impact: Prevents future operations until disk is freed and lock manually cleared
- Improvement path: Check disk space before write; implement pre-write validation; improve error messages

---

## Fragile Areas

**File Lock System:**
- Files: `.aether/utils/file-lock.sh`, `.aether/aether-utils.sh:99-110` (trap handling)
- Why fragile: Multiple exit paths (normal, error, signal); trap pattern replaces previous traps; PID-based staleness detection races with process creation
- Safe modification: Add comprehensive lock lifecycle tests; use explicit cleanup with EXIT trap; validate PID before using
- Test coverage: `tests/bash/test-lock-lifecycle.sh` covers basic cases but not edge cases (SIGKILL, rapid retries, clock skew)

**Pheromone Decay Calculation:**
- Files: `.aether/aether-utils.sh` (pheromone-display, pheromone-decay subcommands)
- Why fragile: Epoch-based decay; timezone-aware calculation complex; off-by-one errors in decay scoring
- Safe modification: Add date utility wrapper; document epoch expectations; test with known decay scenarios
- Test coverage: No dedicated tests for decay math; audited in Phase 33 with `fix: separate lay-eggs (bootstrap) from init (colony start)` but validation tests needed

**JSON Schema Validation on State Migration:**
- Files: `.aether/aether-utils.sh:1105-1145` (_migrate_colony_state)
- Why fragile: Pre-v3.0 state files assumed to have compatible structure; additive-only migration doesn't validate required fields
- Safe modification: Add strict schema validation before and after migration; test with malformed state files
- Test coverage: No tests for migration with corrupt state; Phase 18-04 added backup-on-error but validation is loose

**XML XInclude Processing:**
- Files: `.aether/exchange/pheromone-xml.sh` (xinclude handling), `.aether/utils/xml-compose.sh`
- Why fragile: Recursive includes possible; circular references not detected; external entity expansion risk
- Safe modification: Track include depth; reject cycles; validate DTD before processing
- Test coverage: `tests/bash/test-xinclude-composition.sh` exists but doesn't test circular includes or XXE attacks

**Spawn Tree Entry Cleanup:**
- Files: `.aether/data/spawn-tree.txt`, `.aether/utils/spawn-tree.sh:222-263`
- Why fragile: Entries never deleted; stale entries can confuse parent chain traversal; infinite loop possible with circular parents
- Safe modification: Add entry expiration logic; validate parent chain before using; add loop detection
- Test coverage: Safety limit of 5 exists but circular reference tests missing

---

## Scaling Limits

**State File Size:**
- Current capacity: COLONY_STATE.json ~1.3KB (observed); can grow with event history, pheromones, learnings
- Limit: At 10,000 events + 100 pheromones + 100 learnings, file could reach 50KB; jq parsing time becomes noticeable
- Scaling path: Implement state archival; split into COLONY_STATE.json + events.jsonl + learnings.jsonl; use streaming JSON parser

**Concurrent Access Serialization:**
- Current capacity: Single lock file serializes all operations; 50-second max lock wait time
- Limit: If colony has 10+ concurrent commands, some will timeout; no queuing mechanism
- Scaling path: Implement fine-grained locking (per-resource locks); add work queue with timeout; implement optimistic concurrency

**Backup Directory Growth:**
- Current capacity: MAX_BACKUPS=3 in atomic-write.sh keeps 3 versions
- Limit: Each backup ~1.3KB; no cleanup for old backups beyond limit; could accumulate if backup rotation fails
- Scaling path: Implement timestamp-based cleanup; test backup rotation under load

---

## Clustering/Distribution Issues

**No Multi-Repo Support:**
- Issue: Session state in `.aether/data/` tied to single repo; no support for multi-project colonies
- Files: `.aether/aether-utils.sh:18-19` (AETHER_ROOT detection)
- Impact: Cannot coordinate work across multiple repos; requires separate session per repo
- Improvement path: Add multi-workspace state management; implement session pooling

**No Network Distribution:**
- Issue: All state stored locally; no sync mechanism for shared repos (e.g., monorepo with multiple branches)
- Files: `.aether/data/COLONY_STATE.json` (local only)
- Impact: Two developers on different branches cannot coordinate pheromones or learnings
- Improvement path: Add git-based state sync; implement CRDT for conflict-free merges

---

## Dependencies at Risk

**jq Version Compatibility:**
- Risk: aether-utils.sh uses jq extensively; jq 1.6 has subtle parsing differences from 1.7+
- Impact: Commands that work in CI (jq 1.7) fail locally (jq 1.6)
- Migration plan: Add jq version detection; conditionally use compatibility patterns; test against both versions

**Bash Version Compatibility:**
- Risk: Uses bash 4+ features (associative arrays, declare -A); macOS ships bash 3.2 by default
- Impact: Commands fail on systems with bash 3.x
- Migration plan: Add bash version check on startup; provide homebrew formula for updated bash; add compatibility shims

**Node.js Minimum Version:**
- Risk: package.json requires Node >=16.0.0; some systems may have older versions
- Impact: Installation fails silently; user confused
- Migration plan: Add preinstall check; provide clear error message with upgrade instructions

**Git Version Requirements:**
- Risk: Some commands use git features from 2.x; no version check
- Impact: Fail on systems with older git (e.g., CentOS 7 with git 1.8)
- Migration plan: Add git version detection; feature-flag based on version; document minimum requirement

---

## Missing Critical Features

**No Audit Trail for State Changes:**
- Problem: COLONY_STATE.json modified by 150+ subcommands; no tracking of who/what changed it
- Blocks: Cannot investigate state corruption; difficult to debug multi-session issues
- Fix: Add audit logging to atomic_write; record caller, timestamp, before/after state

**No State Rollback Mechanism:**
- Problem: If state corruption occurs, no way to recover to known-good version
- Blocks: Cannot recover from data corruption without manual intervention
- Fix: Keep versioned state backups (not just latest); implement rollback command; expose via /ant:rollback

**No Batch Operations:**
- Problem: Each operation acquires lock separately; no atomic multi-operation transactions
- Blocks: Cannot safely update multiple state files together (e.g., pheromones + constraints)
- Fix: Implement transaction wrapper; allow multiple operations in single lock acquisition

**No Observability Dashboard:**
- Problem: No way to see current state without reading JSON files
- Blocks: Cannot debug live issues; /ant:status shows limited info
- Fix: Create web dashboard (or TUI) showing live pheromones, constraints, events, learnings

**No Performance Monitoring:**
- Problem: No metrics for command execution time, lock wait time, jq parse time
- Blocks: Cannot identify performance bottlenecks; cannot track improvements
- Fix: Add timing instrumentation to key operations; emit metrics to file; create performance dashboard

**No Dead Letter Queue:**
- Problem: Failed spawns lose context; no way to retry or inspect failure
- Blocks: Transient spawn failures cause build failures; no recovery path
- Fix: Implement DLQ for failed tasks; expose via /ant:dlq; allow manual retry

---

## Test Coverage Gaps

**Untested Spawn Tree Circular References:**
- What's not tested: Spawn tree with circular parent chains (A→B→C→A)
- Files: `.aether/utils/spawn-tree.sh:222-263`, `tests/bash/test-spawn-tree.sh`
- Risk: If circular reference occurs, spawn-parent traversal loops infinitely (loop limit of 5 may not trigger)
- Priority: HIGH — spawn tree is critical for swarm operations

**Untested Lock Acquisition Under Disk Full:**
- What's not tested: acquire_lock behavior when /aether/locks/ directory runs out of space
- Files: `.aether/utils/file-lock.sh:38-129`
- Risk: Lock creation fails; no cleanup; subsequent operations see stale lock
- Priority: MEDIUM — disk full is rare but should degrade gracefully

**Untested JSON Parsing with Malformed State:**
- What's not tested: jq processing when COLONY_STATE.json is corrupt (truncated, extra commas, missing braces)
- Files: `.aether/aether-utils.sh:1105-1145`
- Risk: jq error not caught; command fails without proper error message
- Priority: HIGH — state corruption is possible during crashes

**Untested Pheromone Decay Edge Cases:**
- What's not tested: Decay calculations with leap seconds, DST transitions, clocks set backwards
- Files: `.aether/aether-utils.sh` (pheromone decay), `tests/bash/test-pheromone*.sh`
- Risk: Decay calculations off by hours or days in edge cases; pheromones decay too fast/slow
- Priority: MEDIUM — edge cases rare but scoring will be wrong

**Untested XML with External Entities:**
- What's not tested: pheromone-xml.sh processing with XXE payloads (e.g., <!DOCTYPE root [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>)
- Files: `.aether/exchange/pheromone-xml.sh`, `tests/bash/test-xml-security.sh`
- Risk: Could leak sensitive files; security vulnerability
- Priority: HIGH — XXE is a known attack vector

**Untested Context Update with Concurrent Modification:**
- What's not tested: Two context-update calls racing on COLONY_STATE.json
- Files: `.aether/aether-utils.sh:251-290` (context-update with lock)
- Risk: Lock implementation assumed correct but not stress-tested
- Priority: MEDIUM — race condition unlikely but catastrophic if it happens

**Untested Graceful Degradation:**
- What's not tested: Commands when feature_disable() has been called (activity_log, git_integration, json_processing, file_locking all disabled)
- Files: `.aether/aether-utils.sh:85-97`, `tests/bash/test-aether-utils.sh`
- Risk: Feature detection works but degraded mode never tested; hidden failures
- Priority: LOW — feature detection is defensive code, rarely triggered

---

## Integration Gaps

**GSD Integration Not Fully Hardened:**
- Issue: `/gsd:map-codebase` spawned from Aether but documents aren't fed back into pheromone system
- Files: `.claude/commands/gsd/map-codebase.md`, `.aether/aether-utils.sh` (no GSD signal handling)
- Impact: GSD findings (tech debt, concerns) aren't incorporated into colony decisions
- Fix: Add GSD signal handler; convert GSD CONCERNS.md findings into pheromones

**OpenCode Agent Definitions May Drift:**
- Issue: `.claude/agents/ant/*.md` (canonical) and `.opencode/agents/*.md` (OpenCode mirror) must stay in sync
- Files: All agent definitions in both directories
- Impact: If OpenCode definitions diverge, behavior differs between Claude Code and OpenCode
- Fix: Implement `lint:sync` check for agent count parity; add automated sync on publish

**Model Routing Not Integrated with Pheromones:**
- Issue: Model profiles exist in `.aether/archive/model-routing/model-profiles.js` but not referenced by pheromone system
- Files: `.aether/archive/model-routing/model-profiles.js`, `.aether/aether-utils.sh` (no model routing)
- Impact: Worker assignment doesn't respect model profiles; workers may get routed to wrong models
- Fix: Integrate model routing into spawn logic; add pheromone for model preference

**Survey Data Not Persisted Across Sessions:**
- Issue: Survey results in `.aether/data/survey/` not loaded on session resume
- Files: `.aether/data/survey/`, `.aether/aether-utils.sh` (no survey loading)
- Impact: Codebase reanalyzed each session; slow; no learning from previous survey
- Fix: Load survey results on session init; cache in COLONY_STATE.json; invalidate on code change

---

## Architecture Concerns

**Error Handler Fallback Chain Too Complex:**
- Issue: Multiple error handling layers (error-handler.sh, fallback json_err, bash trap) could mask root causes
- Files: `.aether/utils/error-handler.sh`, `.aether/aether-utils.sh:12-79`
- Impact: Errors from missing dependencies (jq, git) reported as generic "E_UNKNOWN"
- Fix: Simplify to single error path; add dependency check at startup; fail fast on missing dependencies

**Feature Detection Brittle:**
- Issue: feature_disable() called but no way to check feature status at runtime; commands don't verify features before use
- Files: `.aether/aether-utils.sh:85-97`, individual commands (no feature checks)
- Impact: If feature disabled, command fails mysteriously instead of with clear error
- Fix: Add feature_enabled() check; each command verifies required features before use

**No Clear Separation of Concerns in aether-utils.sh:**
- Issue: Single file contains state management, pheromone logic, wisdom logic, spawn management, XML processing
- Files: `.aether/aether-utils.sh` (10,249 lines)
- Impact: Hard to understand data flow; difficult to modify without side effects; testing is global
- Fix: Refactor into modules: state-mgmt.sh (150 lines), pheromone-mgmt.sh (200 lines), wisdom-mgmt.sh (150 lines), etc.

---

## Documentation Gaps

**No Operational Runbook:**
- Missing: How to diagnose lock deadlocks, state corruption, stale pheromones
- Impact: Operators resort to manual file inspection; no systematic troubleshooting steps
- Fix: Create `.aether/docs/runbook.md` with diagnostic tools and recovery procedures

**No Performance Tuning Guide:**
- Missing: How to optimize for large codebases, high concurrency, slow filesystems
- Impact: Users have no way to improve performance
- Fix: Create `.aether/docs/performance-tuning.md` with configuration options and benchmarks

**No Security Hardening Guide:**
- Missing: Best practices for securing state files, logs, credentials in multi-user environments
- Impact: Users may leak sensitive data in activity logs or dreams
- Fix: Create `.aether/docs/security.md` with permissions model and secrets handling

---

## Summary by Severity

**Critical (Fix Now):**
- Monolithic 10K-line aether-utils.sh (refactor)
- Hardcoded error codes (standardize to E_* constants)
- No audit trail for state changes (add logging)
- XXE vulnerability in XML processing (add validation)

**High (Fix Soon):**
- Missing error code documentation (create guide)
- Unbounded activity.log growth (implement rotation)
- Untested circular spawn tree references (add tests)
- Untested malformed state handling (add tests)
- No rollback mechanism (implement recovery)

**Medium (Fix This Quarter):**
- Lock acquisition could be aggressive (add backoff)
- Pheromone decay math untested (add validation)
- Spawn tree rotation partial (complete tuning)
- Activity logging not atomic (refactor to atomic writes)
- GSD integration not feeding back (add signal handler)

**Low (Backlog):**
- Model routing not integrated with pheromones
- Survey data not persisted across sessions
- No observability dashboard
- No dead letter queue for failed spawns

---

*Concerns audit: 2026-03-19*
