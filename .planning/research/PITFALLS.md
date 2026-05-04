# Domain Pitfalls: v1.13 Recovery Hardening & Hive Learning

**Domain:** Adding recovery hardening, build provenance validation, confidence-targeted Oracle loops, gate recovery flows, SQLite integration, hive learning, and worker lifecycle management to an existing Go CLI colony framework
**Researched:** 2026-05-01
**Confidence:** HIGH (based on direct codebase analysis of `cmd/`, `pkg/`, and `pkg/storage/` -- all findings verified against production code)

---

## Critical Pitfalls

### Pitfall 1: Build Provenance Validation Producing False Negatives

**What goes wrong:**
AAC-001 (build-complete rejects failed/zero-modification builds) and AAC-002 (continue validates provenance) introduce a validation gate that inspects build claims (files created/modified, tests written) against actual filesystem state. The provenance check compares claimed files in `last-build-claims.json` against git diff output and file existence. False negatives occur when:

1. **Stale claims file overrides fresh results.** The build finalize flow (`cmd/codex_build_finalize.go` line 211) writes claims to `last-build-claims.json` inside the `build/` directory, but a previous phase's claims file from a prior run may still exist. If the build completes but the claims write fails silently (the error is returned but callers may not propagate it), the next `continue` reads stale claims.

2. **Race condition between worker completion and status writes.** Workers write completion results asynchronously. The current `build-finalize` flow (`codex_build_finalize.go` line 229-236) uses `UpdateJSONAtomically` for COLONY_STATE.json, but the claims file write at line 211 is a plain `SaveJSON`. If two workers finish simultaneously and both trigger claim aggregation, the second writer overwrites the first with partial claims.

3. **Git diff --name-only misses untracked files.** The `discoverChangedFilesFromGit` function (line 464-472) uses `git diff --name-only --diff-filter=A HEAD` which only shows tracked new files. Files created by workers that are not yet `git add`-ed are invisible to provenance validation, causing false "zero modification" rejections.

4. **Worktree isolation breaks path resolution.** In worktree parallel mode, workers write to isolated worktrees. The `normalizeClaimPathsToRoot` function (line 489) resolves paths against the main root, but worktree paths don't exist there until sync-back. All worktree file claims get the "Keep original -- verification will flag it as missing" treatment, causing mass false negatives.

**Why it happens:**
The provenance system assumes a linear build flow: build writes claims, continue validates claims. But Aether has parallel workers, worktree isolation, and async completion. The validation logic was designed for the simple case and doesn't account for these complexities. The existing `discoverChangedFilesFromGit` fallback (line 405-409) partially handles missing claims but only checks git-tracked files.

**Consequences:**
- Valid builds rejected as "zero modification" when workers created files that git diff cannot see
- Continue blocks on "phantom build claims" when stale claims from a previous attempt exist
- Worktree-based builds always fail provenance validation
- Users must manually run `/ant-unblock` or edit claims files to proceed, eroding trust in the gate system

**Prevention:**
- Use `git diff --name-only --diff-filter=A --diff-filter=M` against the pre-build checkpoint, not HEAD, to capture all changes including untracked files. Alternatively, use `git ls-files --others --exclude-standard` for untracked files.
- For worktree mode, validate claims against the worktree root, not the main root, and re-validate after sync-back completes.
- Add a timestamp comparison: if claims timestamp is older than the COLONY_STATE.json `build_started_at` field, treat claims as stale and re-discover from git.
- Make claims file writes atomic (use `AtomicWrite` instead of `SaveJSON` for `last-build-claims.json`).
- Add a `--force-provenance` flag to `continue` that re-discovers claims from git when the claims file is missing or stale.

**Detection:**
- Continue rejects a build that workers report as completed with file outputs
- `aether verify-claims` shows "file not found" for files that exist in the worktree but not the main root
- Users report that `/ant-continue` blocks after a successful `/ant-build` with no clear reason

**Phase assignment:** AAC-001 and AAC-002 (early phases)

---

### Pitfall 2: Confidence-Targeted Oracle Loop Without Convergence Guarantees

**What goes wrong:**
AAC-003 adds a user-settable confidence target to the Oracle loop. The current Oracle loop (`cmd/oracle_loop.go`) already has depth levels (quick/balanced/deep/exhaustive) with max iterations and target confidence, but the confidence scoring has convergence problems:

1. **Confidence score gaming by the LLM worker.** The Oracle worker returns its own confidence score (0-100) per question. An LLM that wants to stop iterating will report inflated confidence scores to hit the target faster. The `oracleOverallConfidence` function (line 1907-1921) computes a simple average of all question confidences. A single worker can set all questions to 95% and exit after 1 iteration regardless of the actual research quality.

2. **No progress detection beyond finding count.** The `oracleProgressedSince` function (line 1869-1879) checks whether findings, answered questions, touched questions, or overall confidence increased. But it does NOT check whether findings are *meaningful* -- a worker can add trivial findings ("the file exists") to satisfy the progress check without advancing understanding.

3. **Prompt drift across iterations.** Each iteration receives a context capsule with prior findings, gaps, and contradictions. After 5-6 iterations, the capsule becomes very large. The Oracle worker context capsule (`renderOracleContextCapsule`, line 769-811) includes ALL prior findings per question. At iteration 8+, the capsule can exceed the LLM's effective context window, causing the worker to ignore earlier findings and produce redundant or contradictory results.

4. **No convergence penalty.** The scoring algorithm (`scoreQuestionImpact`, line 2085-2164) uses keyword overlap, not information gain. If a question is already at 80% confidence, the algorithm still assigns it a deficit of 0.15 (target 95 - current 80) / 100 = 0.15, which may be higher than an untouched question with 0% confidence (deficit = 0.95). This means the algorithm keeps re-investigating nearly-answered questions instead of exploring new territory.

**Why it happens:**
The existing Oracle loop was designed for 4-6 iterations maximum. The v1.13 requirement for user-settable confidence targets (up to 99%) creates pressure to run many more iterations. The convergence mechanisms (progress detection, smart question selection) were not designed for long-running loops. Self-reported confidence from an LLM is fundamentally unreliable because the LLM optimizes for task completion, not accuracy.

**Consequences:**
- Oracle reports 99% confidence after 2 iterations with shallow findings
- Deep research topics never converge because the worker adds padding findings to avoid the "no progress" stop
- Context capsule grows past effective window at iteration 7+, causing quality degradation
- Users lose trust in Oracle confidence scores

**Prevention:**
- Never use self-reported confidence as the sole convergence criterion. Add external signals: finding diversity (unique source URLs, unique file paths), finding specificity (concrete code references vs vague descriptions), contradiction resolution (contradictions must decrease).
- Implement a "confidence decay" mechanism: if a question's confidence does not increase for 2 consecutive iterations, cap it at the current level and move on. This prevents gaming by setting 95% on the first iteration and coasting.
- Cap the context capsule at a fixed character limit (e.g., 4000 chars). When exceeded, summarize prior findings rather than including them verbatim. The current `renderOraclePriorFindings` (line 2285-2312) already limits to 4 findings, but the overall capsule has no cap.
- Add a "confidence ceiling" that requires external validation: to reach 95%+, the worker must provide at least one code-level evidence item (file path + line number or runtime command output), not just documentation references.
- Emit a `CeremonyTopicLoopBreak` event when confidence is self-reported above 90% with fewer than 3 iterations, so the event bus records potential gaming.

**Detection:**
- Oracle completes in 1-2 iterations with "99% confidence" for a topic that clearly needs more research
- Findings in the synthesis report are vague ("the system uses a standard approach") without concrete evidence
- Context capsule exceeds 8000 characters at iteration 5+
- `oracleReadyForCompletion` returns true when questions have "answered" status but low finding quality

**Phase assignment:** AAC-003 (mid-phase, after provenance validation is stable)

---

### Pitfall 3: Gate Recovery Flow Creating New Loops Despite Circuit Breaker

**What goes wrong:**
AAC-006 through AAC-011 add recoverable gate failure banners and a `/ant-unblock` command. The Fixer caste (27th agent, new in v1.13) is supposed to fix gate failures. The REC-LOOP-01 constraint requires all new gate/recovery flows to inherit v1.12 loop safety. But the gate recovery flow creates a new loop cycle:

1. **Gate fails** -> `/ant-unblock` displayed -> Fixer caste spawned -> Fixer makes changes -> Gate re-checked -> Gate fails again (Fixer's fix introduced a new issue) -> `/ant-unblock` displayed -> Fixer spawned again -> infinite loop.

2. **The circuit breaker (`cmd/circuit_breaker.go`) only tracks worker dispatch failures.** It uses `RecordFailure(workerName)` which increments per-worker-name. But the Fixer caste gets a NEW deterministic name each time it's spawned (via `deterministicAntName("fixer", task)`). The circuit breaker sees each Fixer invocation as a different worker, so it never trips.

3. **Cycle detection (`pkg/colony/cycle.go`) only operates on the task dependency graph, not on the gate-check -> fix -> gate-check cycle.** The cycle detector uses a three-color DFS on task DependsOn edges. The gate recovery cycle is not a task dependency -- it's a runtime control flow cycle that exists outside the dependency graph.

4. **The gate recovery template system (`cmd/gate.go` line 669+) provides recovery instructions per gate, but the Fixer agent applies those instructions mechanically.** If the recovery template says "run tests and fix failures," the Fixer may make changes that pass one gate but fail another. This creates a whack-a-mole pattern where fixing gate A breaks gate B, fixing gate B breaks gate C, fixing gate C breaks gate A.

**Why it happens:**
The v1.12 loop safety system (circuit breaker, cycle detection, watcher auto-skip) was designed for the build/continue flow where workers implement tasks and watchers verify them. The gate recovery flow introduces a new control loop (gate fail -> fix -> gate re-check) that is structurally different from the build loop. The existing loop safety mechanisms don't cover this new pattern because they track worker names and task dependencies, not gate-check cycles.

**Consequences:**
- Fixer caste enters infinite spawn-fix-check cycle
- Circuit breaker never trips because each Fixer invocation gets a unique name
- Cycle detector doesn't fire because the cycle is not in the task dependency graph
- Users see repeated "Gate failed: [gate name]" messages with Fixer attempts that never converge
- Colony resources (API calls, tokens) are consumed by the Fixer loop

**Prevention:**
- Add a gate recovery circuit breaker that tracks gate-name-level failure counts, not worker-name-level. If gate X fails 3 consecutive Fixer attempts, stop spawning Fixers and display a human-intervention message.
- Emit `CeremonyTopicLoopBreak` events from the gate recovery flow. The existing `emitLoopBreakEvent` function supports custom loop types ("watcher_skip", "circuit_break", "cycle_detected"). Add "gate_recovery" as a new type.
- Cap Fixer attempts per gate at 2 (not 3), since gate failures are typically systemic and require human judgment.
- Require the Fixer to run ALL gates after its fix, not just the failed gate. This prevents whack-a-mole where fixing one gate breaks another.
- Add a gate recovery state to COLONY_STATE.json that tracks: `{gate_name, attempt_count, last_fixer_name, last_fix_summary}`. The `/ant-unblock` command should check this state and refuse to spawn another Fixer if attempt_count >= max.
- Inherit from the existing circuit breaker: create a `GateRecoveryBreaker` instance alongside the worker circuit breaker in the build flow.

**Detection:**
- Fixer caste spawned 3+ times for the same gate in a single continue cycle
- Gate recovery state shows attempt_count increasing without progress
- Event bus contains multiple "gate_recovery" loop break events for the same gate
- `/ant-status` shows gate failures with Fixer attempts that have identical summaries

**Phase assignment:** AAC-006 through AAC-011 (mid-phase, after REC-LOOP-01 is established)

---

### Pitfall 4: SQLite Integration Colliding with Existing File-Based Storage

**What goes wrong:**
AAC-019 through AAC-031 introduce a SQLite-based hive learning layer. The PRD specifies SQLite with FTS (full-text search) for learning recall. The current storage system (`pkg/storage/`) uses JSON files with file-level locking (`FileLocker`). Introducing SQLite alongside JSON storage creates a dual-storage consistency problem:

1. **WAL mode and gitignore conflict.** The PRD places the SQLite database in `.aether/data/`, which is gitignored. This is correct for data, but WAL mode creates auxiliary files (`database.db-wal`, `database.db-shm`) that also live in `.aether/data/`. If the user runs `git add .aether/data/` (the existing gitignore has a negation exception for `COLONY_STATE.json`), the WAL files get committed. On the next clone, the WAL file from the committed version conflicts with the main database file, causing `SQLITE_CORRUPT` or `database disk image is malformed` errors.

2. **Concurrent read/write conflicts between the Go runtime and LLM workers.** The `pkg/storage/` FileLocker uses platform file locks (`flock` on Unix, `LockFileEx` on Windows). SQLite also uses file locks internally. If the Go runtime opens the database with `database/sql` connection pooling (default behavior) while a worker process also tries to access the database, the two locking systems can deadlock. The `go-sqlite3` driver uses C-level file locks that the Go `FileLocker` cannot coordinate with.

3. **Migration safety across Aether versions.** The PRD specifies schema migrations for the hive learning tables. If a user updates Aether (`aether update`) and the new version has a different schema, the migration must run atomically. But if a colony is active (state = EXECUTING) during the update, the migration may conflict with active writes. The existing `UpdateJSONAtomically` pattern doesn't apply to SQLite -- you need `BEGIN IMMEDIATE` transactions.

4. **FTS index corruption on SIGKILL.** FTS5 virtual tables maintain an in-memory index that is flushed to disk periodically. If the Aether process is killed with SIGKILL (e.g., by the OS OOM killer, or by the user pressing Ctrl+C during a write), the FTS index can become inconsistent with the base table. The `PRAGMA integrity_check` catches base table corruption but may not catch FTS-specific corruption. Rebuilding the FTS index requires `INSERT INTO fts_table(fts_table) VALUES('rebuild')`, which is an expensive operation on large datasets.

5. **The `busy_timeout` PRAGMA must be set BEFORE any connection is used.** The `go-sqlite3` driver accepts PRAGMA settings in the DSN (`file:db.sqlite?_busy_timeout=5000&_journal_mode=WAL`). If these are set after the first connection, WAL mode may not activate properly. The current `pkg/storage/` has no concept of connection strings or PRAGMAs -- everything is file paths.

**Why it happens:**
The existing storage system was designed for JSON files with atomic writes and file-level locking. SQLite introduces a fundamentally different storage paradigm with its own locking, journaling, and connection management. The two systems cannot share state without explicit coordination. The PRD specifies the database location as `.aether/data/` without considering the WAL file implications or the gitignore negation exception.

**Consequences:**
- SQLite WAL files committed to git, causing corruption on clone
- Deadlocks when Go runtime and worker processes access the database simultaneously
- FTS search returns incomplete results after an unclean shutdown
- Schema migration fails during an active colony, leaving the database in an inconsistent state
- `database is locked` errors under concurrent access because `busy_timeout` was not set

**Prevention:**
- Store the SQLite database in a dedicated subdirectory (`.aether/data/hive/`) with its own `.gitignore` that excludes ALL files (not just specific ones). Do NOT rely on the parent `.aether/data/` gitignore.
- Use a single connection pool with `SetMaxOpenConns(1)` for writes and separate read connections. The write connection should use `BEGIN IMMEDIATE` for all transactions. Read connections can use standard `BEGIN`.
- Set PRAGMAs in the DSN: `file:.aether/data/hive/learning.db?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=1&_synchronous=NORMAL`. Verify WAL mode is active by querying `PRAGMA journal_mode` after opening.
- For migration safety, check COLONY_STATE.json state before running migrations. If state is EXECUTING, defer migration to the next init or continue cycle. Write a migration lock file (`.aether/data/hive/.migrating`) to prevent concurrent migration attempts.
- Add an FTS health check to `aether patrol` that runs `PRAGMA integrity_check` and verifies FTS row counts match base table row counts. If mismatched, auto-rebuild the FTS index.
- Do NOT use the existing `FileLocker` for SQLite files. SQLite manages its own locking. Using `FileLocker` on SQLite files causes double-locking deadlocks.
- On unclean shutdown detection (check for WAL file size > 0 on startup), run `PRAGMA wal_checkpoint(TRUNCATE)` before opening for business.

**Detection:**
- `git status` shows `.aether/data/` files that should be gitignored (WAL, SHM files)
- `database is locked` errors in colony logs during concurrent access
- FTS search returns zero results for terms that should match
- `PRAGMA integrity_check` returns anything other than "ok"
- Colony hangs when both the runtime and a worker process try to write to the database

**Phase assignment:** AAC-019 through AAC-031 (later phases, after provenance and gate recovery are stable)

---

### Pitfall 5: Learning from Verified Work Producing False Confidence

**What goes wrong:**
The hive learning system (AAC-019+) turns verified colony work into reusable procedural memory. The "verified" label comes from gate checks passing. But passing gates does not mean the work is correct -- it means the tests pass and no critical flags exist. This creates a false confidence problem:

1. **Lucky passes inflate learning confidence.** A worker implements a feature with a subtle bug. Tests pass because the bug is in an untested code path. Gates pass. The learning system records this as a "verified successful approach" with high confidence. Future colonies that follow this learned pattern reproduce the bug.

2. **Skill deduplication failures create conflicting learned behaviors.** The skill system (`skill-match`) scores skills against workers using role, pheromone signals, and codebase patterns. If two learned skills have similar detect patterns but different content (e.g., one says "always use interface{} for flexibility" and another says "never use interface{}, use generics"), both can be injected into the same worker, creating contradictory instructions.

3. **Privacy gate bypasses through learned context.** The learning system injects learned patterns into worker prompts. If a learned pattern includes file paths, function names, or code snippets from a previous colony (different repo), these get injected into workers in the current colony. The existing pheromone sanitization (`cmd/signal_housekeeping.go`) blocks XML tags and prompt injection patterns, but does NOT check for cross-repo information leakage.

4. **Token budget overflow from injected learned context.** The current colony-prime context budget is 8,000 characters. The existing skill injection has its own 8K budget. Adding learned context from the hive creates a THIRD injection point. If learned context is not budgeted, it can push total prompt size past the LLM's effective context window, causing workers to ignore critical instructions.

5. **The existing hive deduplication is text-exact only.** The `hive-store` command (line 101-119) checks `e.Text == text && e.Domain == domain` for deduplication. But "always initialize structs with zero values" and "structs should always be zero-initialized" express the same wisdom with different text. Without semantic deduplication, the hive fills with near-duplicates that each consume the 200-entry LRU budget.

**Why it happens:**
The hive learning system treats gate passage as a proxy for correctness, but gates only verify surface properties (tests pass, no flags). The learning system has no way to evaluate the quality or generality of what it learns. The deduplication is syntactic, not semantic. The token budget system was designed for two injection points (colony-prime + skills) and doesn't account for a third.

**Consequences:**
- Future colonies learn buggy patterns from lucky passes
- Workers receive contradictory instructions from conflicting learned skills
- Cross-repo information leaks into worker prompts via learned context
- Workers ignore critical instructions because the total prompt exceeds the effective context window
- Hive fills with near-duplicate wisdom entries, evicting genuinely unique entries via LRU

**Prevention:**
- Never set learning confidence above 0.7 for a single-occurrence pattern. Require at least 2 successful verifications across different colonies (different repos, not different phases in the same repo) before confidence reaches 0.8. This is already the multi-repo confidence boost pattern (2 repos = 0.70, 3 repos = 0.85, 4+ = 0.95) but it should be the DEFAULT, not a boost.
- Add a semantic deduplication step before hive-store: normalize the text (lowercase, remove articles, stem words) and check for overlap > 80% with existing entries. If overlap is high, merge into the existing entry rather than creating a new one.
- Add a privacy gate to the learning injection: strip any file paths, function names, and code snippets from learned context before injecting into workers in a different repo. Only inject the generalized wisdom text, not the evidence.
- Budget learned context injection at 2,000 characters max (separate from colony-prime's 8K and skills' 8K). Trim by confidence score (lowest confidence entries trimmed first).
- Add a "learning provenance" field to hive entries that records: `{source_repo, source_phase, gate_results, timestamp}`. This allows auditing whether a learned pattern came from a genuinely verified build.

**Detection:**
- Hive contains entries with confidence 0.95 that were learned from a single colony
- Workers report contradictory instructions from learned skills
- Learned context injection pushes total prompt past 20,000 characters
- Hive wisdom entries contain file paths from other repos
- Hive is full of entries that differ only in word order

**Phase assignment:** AAC-019 through AAC-031 (later phases, learning triggers should come after storage is stable)

---

### Pitfall 6: Process Lifecycle Management Across Platforms

**What goes wrong:**
AAC-014 through AAC-017 add worker heartbeats, process groups, PID tracking, and stale worker cleanup. The current process management (`cmd/oracle_process_unix.go`) is Unix-only and has several gaps:

1. **Orphaned workers on SIGKILL.** The existing `terminateOracleProcessTree` (line 17-75) sends SIGTERM, waits 2 seconds, then SIGKILL. But if the Aether process itself receives SIGKILL (e.g., OOM killer), there is no cleanup. Child worker processes become orphans, reparented to PID 1. On the next run, the stale PID file (`.aether/data/locks/`) may still exist, causing the FileLocker to block indefinitely because the lock-holding process no longer exists but the lock file remains.

2. **PID recycling.** PIDs are recycled on Unix systems. If a worker process with PID 12345 dies and the OS reuses PID 12345 for an unrelated process, `oracleProcessExists` (line 123-138) returns true (the PID exists but is a different process). The stale cleanup logic may kill the wrong process. The current code only checks `ps -o stat=` and rejects zombies, but does not verify that the process at that PID is actually an Aether worker.

3. **Process group signaling differences across platforms.** On Linux, process groups are session-level (SID). On macOS, process groups are different from sessions. On Windows (via `oracle_process_windows.go`), there are no process groups in the Unix sense -- `cmd.Process.Kill()` is the only option. The `terminateOracleProcessTree` function uses `ps -axo pid=,ppid=` to build a process tree, which works on macOS and Linux but has different column widths on different systems.

4. **Heartbeat files as stale detection.** The PRD proposes heartbeat files for worker liveness detection. But heartbeat files have a fundamental race condition: if the worker process is alive but the filesystem write for the heartbeat fails (disk full, NFS timeout, I/O error), the heartbeat appears stale and the cleanup logic kills a healthy worker. Conversely, if the worker process dies but the heartbeat file was written 1 second before death, the heartbeat appears fresh for up to the heartbeat interval, leaving a dead worker running until the next check.

5. **The existing FileLocker has no stale lock detection.** The `FileLocker` (`pkg/storage/lock.go`) acquires locks via `platformLockFile`. If the process that held the lock dies without releasing it, the lock file remains. On Unix, `flock` locks are automatically released when the process dies (because the file descriptor is closed). But if the lock file itself is corrupted or the filesystem doesn't support `flock` (e.g., NFS), the lock persists.

**Why it happens:**
Process lifecycle management is inherently platform-specific and full of edge cases. The current implementation was written for the Oracle loop's process tree and handles that specific case (terminate all descendants of a known PID). Generalizing it to all worker processes (builders, watchers, fixers) introduces cases that the Oracle-specific code doesn't cover. PID recycling is a well-known Unix problem that most developer tools eventually hit. Heartbeat files are a common pattern but have the filesystem-reliability race condition.

**Consequences:**
- Stale worker processes consume resources after the colony exits
- PID recycling causes the stale cleanup to kill unrelated processes
- Heartbeat false positives kill healthy workers
- FileLocker blocks indefinitely on corrupted lock files after unclean shutdown
- Windows users experience different behavior than macOS/Linux users

**Prevention:**
- Verify PID identity before killing: check that `/proc/<pid>/cmdline` (Linux) or `ps -o command= -p <pid>` (macOS) contains "aether" or "codex" before sending any signal. Never kill a PID based solely on its number.
- Use process groups instead of PID tracking. Start each worker in a new process group (`cmd.SysProcAttr.Setpgid = true` on Unix, `CREATE_NEW_PROCESS_GROUP` on Windows). Kill the entire group with `syscall.Kill(-pgid, signal)`. This avoids PID recycling issues entirely because the group ID is set at creation.
- Add a startup stale lock cleanup: on every `aether` invocation, scan `.aether/locks/` for lock files whose holding process no longer exists (check via `kill(pid, 0)` which returns ESRCH if the process doesn't exist). Remove orphaned lock files.
- For heartbeats, use a shared memory segment or Unix socket instead of a file. If that's too complex, write heartbeats to a known location AND check the process existence (via PID file + `kill(pid, 0)`). Only consider a worker stale if BOTH the heartbeat is old AND the process doesn't exist.
- On startup, check for stale PID files (`.aether/data/*.pid`) and verify the processes still exist before trusting them.
- Test on all three platforms (macOS, Linux, Windows) for every process lifecycle change. The Windows implementation (`oracle_process_windows.go`) should be updated alongside the Unix implementation.

**Detection:**
- `ps aux | grep aether` shows worker processes running after the colony has exited
- Stale lock files in `.aether/locks/` that reference non-existent PIDs
- `aether build` hangs on startup because a lock file from a previous run was not cleaned up
- Workers killed mid-task due to heartbeat filesystem errors on NFS-mounted home directories
- Windows users report that worker cleanup doesn't work (the Windows implementation was not updated)

**Phase assignment:** AAC-014 through AAC-017 (mid-phase, alongside gate recovery)

---

## Moderate Pitfalls

### Pitfall 7: 31 Work Packages Creating Merge Conflicts in a Single Milestone

**What goes wrong:**
The v1.13 PRD defines 31 work packages (AAC-001 through AAC-031). If these are implemented across many phases with parallel development, the merge conflict surface is enormous. The most conflict-prone files are:

- `cmd/colony_prime_context.go` (touched by any feature that injects context into workers)
- `pkg/colony/state.go` (touched by any feature that adds state fields)
- `cmd/gate.go` (touched by any feature that adds gates)
- `.claude/commands/ant/build.md` and `.claude/commands/ant/continue.md` (touched by any feature that modifies build or continue flow)
- `cmd/main.go` (touched by any feature that adds new subcommands)

**Why it happens:**
31 work packages in a single milestone is a large surface area. The existing codebase has tight coupling between context assembly, state management, and gate checking. Features that seem independent (e.g., SQLite integration and process lifecycle) both touch the build/continue flow.

**Prevention:**
- Group work packages by the files they modify, not by feature area. Sequence groups so that files modified by group A are not touched by group B.
- Add new subcommands to a separate registration file (e.g., `cmd/subcommands.go`) instead of adding them all to `cmd/main.go`.
- Add new state fields in a single batch phase, not spread across multiple phases. Use `omitempty` for all new fields to maintain backward compatibility.
- For colony-prime context, add new sections behind a feature flag so they can be developed independently and enabled together.

**Detection:**
- `git merge` produces conflicts in more than 2 files per merge
- `go test ./...` fails after merging two feature branches

---

### Pitfall 8: New Dependencies Breaking Existing Tests

**What goes wrong:**
The PRD specifies SQLite integration. The current project has a "Zero new dependencies" policy (PROJECT.md line 158). Adding `github.com/mattn/go-sqlite3` introduces CGo, which means:

1. All builds now require a C compiler (gcc/clang)
2. Cross-compilation becomes harder (need C cross-compiler)
3. `go test -race` behavior changes because CGo has different race detection semantics
4. The `goreleaser` config may need updating for CGo builds

**Why it happens:**
The "zero new dependencies" policy was established when Aether only used JSON file storage. SQLite requires a C library (either CGo or `modernc.org/sqlite` pure Go). The PRD doesn't specify which SQLite driver to use.

**Prevention:**
- Use `modernc.org/sqlite` (pure Go SQLite) instead of `mattn/go-sqlite3` (CGo). Pure Go eliminates the C compiler dependency, simplifies cross-compilation, and maintains the zero-CGo policy. The performance difference is negligible for Aether's use case (single-user, low-concurrency).
- If CGo is unavoidable, update the README build instructions, goreleaser config, and CI matrix to include CGo requirements.
- Run the full test suite with the new dependency before committing: `go test ./... -race -count=1`.

**Detection:**
- `go build` fails on systems without gcc/clang
- `goreleaser build --snapshot` fails with CGo errors
- `go test -race` produces different results with CGo enabled

---

### Pitfall 9: OpenCode Platform Divergence from New Features

**What goes wrong:**
The v1.13 features touch build, continue, and gate flows heavily. The CLAUDE.md states that Claude and OpenCode should be "aligned first." But the new features are implemented in the Go runtime, and the wrapper markdown files for OpenCode may not be updated to reflect new flags, new output formats, or new error messages.

Specifically:
- `/ant-unblock` (new command) needs OpenCode wrapper in `.opencode/commands/ant/unblock.md`
- Gate recovery banners need OpenCode formatting in the continue wrapper
- Worker heartbeat display needs OpenCode-compatible ANSI rendering

**Why it happens:**
The Go runtime is authoritative, but the wrapper markdown files are presentation-only. When new features are added to the runtime, the wrappers must be updated separately. The existing `command_parity_test.go` checks file counts but not content parity.

**Prevention:**
- For each new command, create the OpenCode wrapper simultaneously with the Claude wrapper.
- Add the new command to the parity test.
- Use the runtime's structured JSON output as the single source of truth for both wrappers.

---

### Pitfall 10: Token Budget Overflow from Triple Injection

**What goes wrong:**
The current system has two injection points into worker prompts: colony-prime context (8K chars) and skill injection (8K chars). The hive learning system adds a third: learned context from the hive. The total prompt size is colony-prime (8K) + skills (8K) + learned context (???) + task brief (variable) + system prompt (variable). For LLMs with a 128K context window this is fine, but for smaller models or longer task briefs, the total can exceed the effective context window.

**Why it happens:**
The token budget system was designed for two injection points. Adding a third without adjusting the budget creates an unbounded injection point.

**Prevention:**
- Cap learned context at 2,000 characters.
- Add a total injection budget check: colony-prime + skills + learned context + task brief must not exceed 20,000 characters.
- If the total exceeds the budget, trim the lowest-confidence learned entries first, then the lowest-priority skill entries, then the lowest-ranked colony-prime sections.

---

## Minor Pitfalls

### Pitfall 11: Fixer Caste Making Changes That Create More Gate Failures

**What goes wrong:**
The Fixer caste (27th agent) is spawned to fix gate failures. But the Fixer's changes may introduce new failures in gates that previously passed. For example, fixing a test failure by modifying production code may break a lint gate. Fixing a lint issue by removing a function may break a coverage gate.

**Prevention:**
- The Fixer must run ALL gates after its fix, not just the failed gate.
- If the Fixer's fix causes a different gate to fail, report both failures and stop (don't attempt to fix the new failure).
- Cap Fixer at 1 attempt per gate failure cycle.

---

### Pitfall 12: Oracle Loop Stop Marker Race Condition

**What goes wrong:**
The Oracle loop checks for a stop marker file (`.aether/oracle/.stop`) at the start of each iteration (line 398) and after each iteration (line 558). But between the check and the worker invocation, the user may write the stop marker. The worker invocation is async (line 664-667, goroutine with channel). The stop marker check doesn't cancel an in-flight worker invocation -- it only prevents the NEXT iteration from starting.

**Prevention:**
- Pass a cancellable context to the worker invocation and check the stop marker in a separate goroutine that calls `cancel()` when the marker appears.
- This is already partially done in `runOracleIterationAttempt` (line 653, `attemptCtx, cancel := context.WithCancel(ctx)`), but the cancel is only called when the heartbeat ticker detects a response file, not when the stop marker appears.

---

### Pitfall 13: Hive Wisdom Abstraction Losing Actionable Specificity

**What goes wrong:**
The `promoteToHive` function (`cmd/hive.go` line 254-343) abstracts repo-specific text by replacing the source repo name with `<repo>` and stripping common path prefixes. But this abstraction can make wisdom too vague to be actionable. For example, "In the Aether repo, always use `store.UpdateJSONAtomically` instead of `store.SaveJSON` for COLONY_STATE.json" becomes "In <repo>, always use `store.UpdateJSONAtomically` instead of `store.SaveJSON` for COLONY_STATE.json" -- which is only useful for repos that happen to use the same store package.

**Prevention:**
- Add a "scope" field to wisdom entries: `repo-specific` vs `domain-general`. Repo-specific wisdom is only injected into workers in the same repo. Domain-general wisdom is injected everywhere.
- The abstraction step should produce TWO entries: the repo-specific original (high confidence, repo-scoped) and the domain-general abstracted version (lower confidence, cross-repo).

---

### Pitfall 14: Session Freshness Detection Breaking After Recovery

**What goes wrong:**
The CLAUDE.md documents session freshness detection: capture `SESSION_START=$(date +%s)` before spawning agents, then verify freshness. The recovery hardening changes may reset session state (e.g., `/ant-unblock` modifies COLONY_STATE.json) without updating the session timestamp. The freshness check then thinks the session is stale and auto-clears files that the recovery flow needs.

**Prevention:**
- Any recovery command that modifies COLONY_STATE.json must also update the session timestamp.
- The freshness check should be aware of recovery commands and skip auto-clear for recovery-modified files.

---

### Pitfall 15: Gate Recovery Templates Becoming Stale

**What goes wrong:**
The gate recovery template system (`cmd/gate.go` line 669+) provides per-gate recovery instructions. But as gates are added or modified in v1.13, the templates must be updated to match. If a gate's check logic changes but its recovery template doesn't, the Fixer caste follows outdated instructions.

**Prevention:**
- Store recovery templates in the same file as gate definitions, not in a separate file.
- Add a test that verifies every gate has a corresponding recovery template.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| AAC-001/002: Build provenance | False negatives from stale claims, git diff gaps, worktree path issues | Use checkpoint-based diff, timestamp validation, worktree-aware path resolution |
| AAC-003: Oracle confidence target | LLM gaming confidence scores, prompt drift at high iterations | External validation requirements, confidence decay, context capsule cap |
| AAC-004: Init launch brief | Synthesizing from stale scouting data | Check scouting timestamp, re-scan if stale |
| AAC-005: Full-context path restoration | Context budget overflow | Measure current context size before enabling full path |
| AAC-006-011: Gate recovery | Fixer creating infinite loops, whack-a-mole gate failures | Gate-level circuit breaker, Fixer attempt cap, all-gate re-check |
| AAC-012-013: OpenCode fixes | Agent name field divergence | Update all 4 mirror locations simultaneously |
| AAC-014-017: Worker lifecycle | PID recycling killing wrong processes, orphaned workers | Process groups instead of PIDs, identity verification before kill |
| AAC-018: E2E validation | Integration tests not covering new recovery flows | Add E2E tests for every new gate/recovery path |
| AAC-019-031: Hive learning | False confidence from lucky passes, skill dedup failures, privacy leaks | Multi-repo confidence requirement, semantic dedup, privacy gate |

---

## "Looks Done But Isn't" Checklist

- [ ] **Provenance works in worktree mode:** Often provenance is tested in-repo only -- test with `--parallel-mode worktree` and verify claims resolve correctly after sync-back
- [ ] **Oracle loop converges on hard topics:** Often tested with easy topics that converge in 2 iterations -- test with ambiguous topics that should require 5+ iterations
- [ ] **Gate recovery doesn't create loops:** Often tested with a single gate failure -- test with 2 gates failing simultaneously and verify the Fixer doesn't loop
- [ ] **SQLite survives SIGKILL:** Often tested with clean shutdown -- test with `kill -9` during an active write and verify WAL recovery
- [ ] **Heartbeat doesn't false-positive on slow filesystems:** Often tested on local SSD -- test on NFS or encrypted home directories
- [ ] **Learned context doesn't leak cross-repo information:** Often tested in a single repo -- test with hive entries from repo A injected into workers in repo B
- [ ] **All 3 platforms render new commands:** Often tested on Claude only -- test `/ant-unblock`, gate recovery banners, and heartbeat display on OpenCode and Codex
- [ ] **Circuit breaker covers gate recovery:** Often the worker circuit breaker is tested -- verify the gate recovery circuit breaker fires independently
- [ ] **COLONY_STATE.json backward compatible:** Often new fields are added -- verify that old colonies (without new fields) can still be loaded
- [ ] **No new CGo dependency surprises:** If using `modernc.org/sqlite`, verify it doesn't introduce CGo transitively

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Provenance false negatives | LOW | Add `--force-provenance` flag; re-discover claims from git; no data migration |
| Oracle confidence gaming | MEDIUM | Adjust convergence criteria; re-run Oracle for affected topics; update hive entries |
| Gate recovery infinite loop | LOW | Add gate-level circuit breaker; no data migration needed |
| SQLite corruption | MEDIUM | Delete WAL/SHM files; rebuild FTS index; re-run migration; data loss possible |
| False learning confidence | MEDIUM | Audit hive entries; reset confidence scores; add privacy gate retroactively |
| PID recycling killing wrong process | HIGH | Audit killed PIDs; restore killed processes if possible; add identity verification |
| Token budget overflow | LOW | Reduce learned context cap; adjust budget allocation; no data migration |

---

## Sources

- Direct codebase analysis: `cmd/circuit_breaker.go` (circuit breaker implementation, per-worker-name tracking)
- Direct codebase analysis: `pkg/colony/cycle.go` (cycle detection, three-color DFS on task dependency graph)
- Direct codebase analysis: `cmd/oracle_loop.go` (Oracle loop, confidence scoring, progress detection, context capsule rendering)
- Direct codebase analysis: `cmd/codex_build_finalize.go` (build claims, provenance validation, git diff fallback)
- Direct codebase analysis: `cmd/hive.go` (hive wisdom store, deduplication, abstraction, LRU eviction)
- Direct codebase analysis: `pkg/storage/storage.go` (atomic writes, file locking, JSON-first storage)
- Direct codebase analysis: `pkg/storage/lock.go` (FileLocker, platform-specific locking)
- Direct codebase analysis: `cmd/oracle_process_unix.go` (process tree termination, PID detection)
- Direct codebase analysis: `cmd/gate.go` (gate checking, recovery templates)
- Direct codebase analysis: `cmd/ceremony_emitter.go` (loop break event emission)
- Direct codebase analysis: `cmd/colony_prime_context.go` (context budget system, 8K char limit)
- [SQLite WAL mode concurrency (Reddit r/sqlite)](https://www.reddit.com/r/sqlite/comments/1nfvbh1/whats_more_performant_for_concurrent_writes_a_1/)
- [mattn/go-sqlite3 concurrency issues #1179](https://github.com/mattn/go-sqlite3/issues/1179)
- [SQLite Concurrency in Go: ChatML Blog](https://chatml.com/blog/sqlite-concurrency-in-go-desktop-ai-ide)
- [SQLite Go best practices (Tessl Registry)](https://tessl.io/registry/tessl-labs/sqlite-go-best-practices/0.2.0/quality)
- [SQLite concurrent writing performance (Stack Overflow)](https://stackoverflow.com/questions/35804884/sqlite-concurrent-writing-performance)
- PROJECT.md v1.13 requirements (31 work packages: AAC-001 through AAC-031, REC-LOOP-01 constraint)

---
*Pitfalls research for: Aether v1.13 Recovery Hardening & Hive Learning*
*Researched: 2026-05-01*
