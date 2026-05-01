# Technology Stack

**Project:** Aether v1.13 -- Recovery Hardening & Hive Learning
**Researched:** 2026-05-01
**Confidence:** HIGH (all findings from source code inspection, Context7 docs, and verified web sources)

## Executive Summary

v1.13 requires exactly **one new external dependency**: `modernc.org/sqlite` for the hive learning layer. Everything else -- process groups, confidence scoring, privacy scanning, skill lifecycle, trajectory compression, and the learning policy engine -- builds on existing Go stdlib patterns already proven in the codebase. The process group infrastructure already exists in `pkg/codex/process_group_unix.go` and `cmd/verification_process_group_unix.go`. The confidence scoring engine already exists in `pkg/memory/trust.go` with 40/35/25 weighted rubrics, 7 trust tiers, and half-life decay. The skill system already has parse, diff, index, detect, match, and inject -- missing only create/patch/archive/promote lifecycle operations. The privacy gate can extend the existing `check-antipattern` command in `cmd/security_cmds.go` rather than importing gitleaks or trufflehog.

## New Dependencies

| Dependency | Version | Purpose | Why |
|------------|---------|---------|-----|
| `modernc.org/sqlite` | v1.42.1 | Colony memory store with FTS5 recall | Pure Go (no CGO), supports FTS5 out of the box, standard `database/sql` interface. Only new dependency needed for v1.13. |
| `modernc.org/libc` | (matched to sqlite) | Transitive dependency of modernc.org/sqlite | Must match the exact version in modernc.org/sqlite's own go.mod. |

**Why modernc.org/sqlite over alternatives:**

| Criterion | modernc.org/sqlite | mattn/go-sqlite3 | zombiezen/go-sqlite |
|-----------|-------------------|-------------------|---------------------|
| CGO required | No | Yes | No |
| FTS5 support | Built-in | Build tag needed | No (low-level wrapper) |
| database/sql | Yes | Yes | No (custom API) |
| Cross-compile | Trivial | Hard | Trivial |
| Maintenance | Active (v1.42.1, Feb 2026) | Active | Active |
| Aether fit | Single binary, no C toolchain | Requires gcc/clang | Different API surface |

**Confidence: HIGH** -- FTS5 support confirmed via Context7 documentation and multiple web sources. The driver imports as `_ "modernc.org/sqlite"` and uses `sql.Open("sqlite", path)`.

## No-Change Stack (Existing)

### Core Framework

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.26.1 | Runtime language | Existing go.mod target. No version change needed. |
| `github.com/spf13/cobra` | v1.10.2 | CLI subcommand registration | All new hive/learning commands register via `rootCmd.AddCommand()`. |
| `pkg/storage.Store` | existing | File-locked JSON persistence | Colony state, pheromones, and session data remain on JSON. SQLite is only for the hive learning store, not a replacement. |
| `pkg/events.Bus` | existing | Typed event bus with JSONL persistence | Learning events publish through existing bus. New topics (`hive.learn`, `hive.promote`) extend the existing pattern. |
| `pkg/memory` | existing | Trust scoring, promotion pipeline, instinct storage | Confidence engine already has 40/35/25 weighted rubrics, 7 tiers, half-life decay. Extended, not replaced. |

### Existing Pattern Reuse by Feature

| New Feature | Existing Code to Extend | What's New |
|-------------|------------------------|------------|
| Process groups | `pkg/codex/process_group_unix.go`, `cmd/verification_process_group_unix.go` | Heartbeat tracking, PID registry, stale worker detection. Infrastructure already has `Setpgid: true`, SIGTERM/SIGKILL escalation, process existence checks. |
| Confidence scoring | `pkg/memory/trust.go` (Calculate, Decay, Tier), `pkg/memory/promote.go` | Oracle confidence target loop. The 40/35/25 weighted rubric (source/evidence/activity) already produces scores 0.2-1.0 with 7 named tiers. The Oracle loop just iterates until target threshold is met. |
| Privacy/secret scanning | `cmd/security_cmds.go` (check-antipattern, 6 patterns) | Extend existing regex scanner with ~15 additional patterns (API key prefixes, tokens, credentials). No new library needed -- the existing scanner already does per-line regex matching with critical/warning classification. |
| Skill lifecycle | `cmd/skills.go` (parse, index, detect, match, inject, diff) | Add create (write SKILL.md with frontmatter), patch (update existing), archive (move to `.aether/skills/archive/`), promote (user domain to shipped). All file operations, no new deps. |
| Trajectory recording | `pkg/events/event.go` (Event struct, JSONL persistence), `cmd/recovery_snapshot.go` | Trajectory is an ordered sequence of phase snapshots stored in SQLite. Compression is time-based: merge consecutive identical-state snapshots, keep state-change boundaries. Uses stdlib only. |
| Learning policy engine | `pkg/memory/promote.go` (PromoteService), `pkg/memory/consolidate.go` | Evidence validation rules are Go structs with thresholds, not a separate rule engine library. The policy is: observation must have trust score >= threshold, evidence type must be test_verified or multi_phase, and content must pass dedup. All implementable in ~100 lines of Go. |

## Feature 1: SQLite Colony Memory Store with FTS5

### Schema Design

```sql
-- Colony-scoped memory entries
CREATE TABLE IF NOT EXISTS memories (
    id          TEXT PRIMARY KEY,         -- mem_{unix}_{6hex}
    colony_id   TEXT NOT NULL,            -- repo-scoped
    category    TEXT NOT NULL,            -- learning | instinct | pheromone_skill | trajectory
    content     TEXT NOT NULL,
    metadata    TEXT DEFAULT '{}',        -- JSON blob for domain, confidence, provenance
    trust_score REAL DEFAULT 0.0,
    confidence  REAL DEFAULT 0.0,
    source_type TEXT DEFAULT '',
    evidence    TEXT DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    expires_at  TEXT                     -- NULL = never expires
);

CREATE INDEX IF NOT EXISTS idx_memories_colony ON memories(colony_id);
CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(colony_id, category);
CREATE INDEX IF NOT EXISTS idx_memories_confidence ON memories(colony_id, confidence);
CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(colony_id, created_at);

-- FTS5 for content recall
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    content,
    category,
    content=memories,
    content_rowid=rowid
);

-- Triggers to keep FTS in sync
CREATE TRIGGER memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, content, category) VALUES (new.rowid, new.content, new.category);
END;
CREATE TRIGGER memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, content, category) VALUES('delete', old.rowid, old.content, old.category);
END;
CREATE TRIGGER memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, content, category) VALUES('delete', old.rowid, old.content, old.category);
    INSERT INTO memories_fts(rowid, content, category) VALUES (new.rowid, new.content, new.category);
END;
```

### Integration Pattern

```go
// pkg/hive/store.go
package hive

import (
    "database/sql"
    _ "modernc.org/sqlite"
)

type Store struct {
    db *sql.DB
}

func Open(dbPath string) (*Store, error) {
    db, err := sql.Open("sqlite",
        dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
    // ...
}
```

**Key design decisions:**
- **WAL mode** for concurrent read access (build workers read, continue writes)
- **busy_timeout(5000)** to handle concurrent write contention gracefully
- **External content FTS5** (`content=memories`) to avoid data duplication -- the content table is authoritative, FTS is an index
- **Repo-scoped** via `colony_id` column -- each colony gets isolated recall, no cross-colony leakage
- **DB location**: `~/.aether/hive/colony-memory.db` (hub-level, shared across colonies on same machine)

### Query Patterns

```sql
-- Full-text recall
SELECT m.* FROM memories m
JOIN memories_fts f ON m.rowid = f.rowid
WHERE f.memories_fts MATCH ? AND m.colony_id = ?
ORDER BY bm25(memories_fts) LIMIT 20;

-- High-confidence instincts for a colony
SELECT * FROM memories
WHERE colony_id = ? AND category = 'instinct' AND confidence >= 0.8
ORDER BY confidence DESC, updated_at DESC;

-- Recent learning observations
SELECT * FROM memories
WHERE colony_id = ? AND category = 'learning'
ORDER BY created_at DESC LIMIT 50;
```

**Confidence: HIGH** -- FTS5 syntax verified via Context7 docs and SQLite official documentation. External content pattern verified.

## Feature 2: Process Group Management

### Existing Infrastructure (No Changes Needed)

The codebase already has complete process group management:

| File | What It Does |
|------|-------------|
| `pkg/codex/process_group_unix.go` | `workerSysProcAttr()` returns `Setpgid: true`, `terminateWorkerProcess()` sends SIGTERM to process group, `killWorkerProcess()` sends SIGKILL, `workerProcessExists()` checks via `ps`, `workerProcessCommandLine()` reads command line |
| `pkg/codex/process_group_windows.go` | Windows stubs (no-op, correct for Windows) |
| `cmd/verification_process_group_unix.go` | `configureVerificationCommandProcessGroup()` and `terminateVerificationCommandProcessGroup()` for verification subprocess isolation |
| `pkg/codex/process_group_unix_test.go` | Test verifying `Setpgid: true` is set |

### What to Add

| Component | Location | Pattern |
|-----------|----------|---------|
| PID registry | `pkg/codex/pid_registry.go` (new) | Map of worker name to PID, persisted to `worker-pids.json` via existing `pkg/storage.Store` |
| Heartbeat checker | `pkg/codex/heartbeat.go` (new) | Goroutine that polls `workerProcessExists()` every N seconds, marks workers as stale if not responding |
| Stale cleanup | `cmd/` (extend existing recover or add new subcommand) | On colony start, check PID registry against running processes, clean orphaned entries |
| Graceful shutdown | Extend existing `terminateWorkerProcess()` | Add timeout: SIGTERM, wait 5s, then SIGKILL. Already has the two-signal pattern conceptually. |

**Confidence: HIGH** -- All primitives exist. This is assembly work, not new infrastructure.

## Feature 3: Confidence Scoring with Weighted Rubrics

### Existing Engine (No Changes to Core)

`pkg/memory/trust.go` already implements the full scoring system:

```
raw_score = 0.4 * source_score + 0.35 * evidence_score + 0.25 * activity_score
score = max(0.2, raw_score)

Source weights:  user_feedback=1.0, error_resolution=0.9, success_pattern=0.8, observation=0.6, heuristic=0.4
Evidence weights: test_verified=1.0, multi_phase=0.9, single_phase=0.7, anecdotal=0.4
Activity: 0.5^(days/60)  -- 60-day half-life

Tiers: canonical(>=0.9), trusted(>=0.8), established(>=0.7), emerging(>=0.6), provisional(>=0.45), suspect(>=0.3), dormant(<0.3)
```

### Oracle Confidence Target Loop

The Oracle loop (AAC-003) iterates research until confidence meets a user-settable target:

```go
// cmd/oracle_loop.go -- extend existing
func runOracleWithTarget(ctx context.Context, query string, target float64, maxIterations int) (OracleResult, error) {
    for i := 0; i < maxIterations; i++ {
        result := runSingleOraclePass(ctx, query)
        if result.Confidence >= target {
            return result, nil
        }
        // Refine query based on gaps
        query = refineQuery(query, result.Gaps)
    }
    return result, fmt.Errorf("confidence target %.2f not met after %d iterations", target, maxIterations)
}
```

**Confidence: HIGH** -- Core scoring engine exists and is battle-tested (2900+ tests). Oracle loop is new orchestration on top of existing primitives.

## Feature 4: Learning Policy Engine

### Approach: Struct-Based Rules, Not a Rule Engine Library

No external rule engine library needed. The learning policy is a set of Go structs with threshold checks:

```go
// pkg/hive/policy.go
type LearningPolicy struct {
    rules []LearningRule
}

type LearningRule struct {
    Name        string
    Category    string  // "instinct" | "pheromone_skill" | "trajectory"
    MinTrust    float64 // Minimum trust score to promote
    MinEvidence string  // Required evidence type (test_verified, multi_phase, etc.)
    MinObsCount int     // Minimum observation count
    AutoPromote bool    // Whether promotion happens automatically
}

// Default policies matching the existing system
var DefaultPolicies = []LearningRule{
    {Name: "instinct-promote", Category: "instinct", MinTrust: 0.6, MinEvidence: "single_phase", MinObsCount: 2, AutoPromote: true},
    {Name: "hive-promote", Category: "instinct", MinTrust: 0.8, MinEvidence: "multi_phase", MinObsCount: 3, AutoPromote: true},
    {Name: "skill-auto-create", Category: "pheromone_skill", MinTrust: 0.7, MinEvidence: "test_verified", MinObsCount: 4, AutoPromote: false},
}
```

**Why not a rule engine library (Cedar, govaluate, etc.):**
- Only 3-5 rules, all structurally identical (threshold checks)
- Rules change infrequently (milestone boundaries, not runtime)
- A rule engine adds complexity, binary size, and a new abstraction to learn
- Go if-statements with struct-based rules are debuggable and testable

**Confidence: HIGH** -- Pattern already used in `pkg/memory/promote.go` (capacity check, dedup check, trust threshold).

## Feature 5: Privacy/Secret Scanning

### Extend Existing Scanner

The existing `cmd/security_cmds.go` has `check-antipattern` with 6 patterns across Swift, TypeScript, and shell. The hive learning store needs a privacy gate that prevents secrets from being stored as learning content.

### Additional Patterns to Add

| Pattern | Purpose | Severity |
|---------|---------|----------|
| `AKIA[0-9A-Z]{16}` | AWS access key | critical |
| `ghp_[a-zA-Z0-9]{36}` | GitHub personal access token | critical |
| `gho_[a-zA-Z0-9]{36}` | GitHub OAuth token | critical |
| `ghs_[a-zA-Z0-9]{36}` | GitHub app token | critical |
| `ghu_[a-zA-Z0-9]{36}` | GitHub user token | critical |
| `sk-[a-zA-Z0-9]{48}` | OpenAI API key | critical |
| `sk-ant-[a-zA-Z0-9-]{80}` | Anthropic API key | critical |
| `xox[bposa]-[0-9]{10,13}-[a-zA-Z0-9]{24,}` | Slack token | critical |
| `-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----` | Private key block | critical |
| `password\s*[:=]\s*["'][^"']{8,}["']` | Hardcoded password | critical |
| `(token|secret|api_key|apikey)\s*[:=]\s*["'][a-zA-Z0-9_-]{20,}["']` | Generic credential | warning |

**Why not import gitleaks/trufflehog:**
- Adding gitleaks as a dependency brings in 150+ transitive dependencies
- The hive learning privacy gate only scans content being stored, not entire repos
- A simple regex pass over learning content (a few KB per observation) is sufficient
- False positives on learning content are acceptable (conservative: reject on match)

**Confidence: HIGH** -- Pattern established in existing code. Extensions are well-known regex patterns from the secret scanning community.

## Feature 6: Skill Lifecycle Management

### Existing Operations (No Changes)

| Operation | Command | Location |
|-----------|---------|----------|
| Parse frontmatter | `skill-parse-frontmatter` | `cmd/skills.go` |
| Build index | `skill-index` | `cmd/skills.go` |
| Detect matches | `skill-detect` | `cmd/skills.go` |
| Match to worker | `skill-match` | `cmd/skills.go` |
| Inject into prompt | `skill-inject` | `cmd/skills.go` |
| Diff user vs shipped | `skill-diff` | `cmd/skills.go` |
| Check if user-created | `skill-is-user-created` | `cmd/skills.go` |
| Cache rebuild | `skill-cache-rebuild` | `cmd/skills.go` |

### New Operations

| Operation | Command | Implementation |
|-----------|---------|----------------|
| Create skill | `skill-create` | Write SKILL.md with YAML frontmatter to `~/.aether/skills/domain/{name}/SKILL.md`. Follow existing `skillFrontmatter` struct format. ~50 lines. |
| Patch skill | `skill-patch` | Read existing SKILL.md, update specific frontmatter fields, preserve body content. ~40 lines. |
| Archive skill | `skill-archive` | Move SKILL.md from active location to `.aether/skills/archive/{name}/SKILL.md`. Remove from index. ~30 lines. |
| Promote skill | `skill-promote` | Copy from `~/.aether/skills/domain/` to `.aether/skills/domain/` (shipped). Update index and manifest. ~40 lines. |
| List versions | `skill-versions` | Show shipped vs user vs hub versions with timestamps. Extend existing index. ~30 lines. |

All use `os.ReadFile`/`os.WriteFile`/`os.MkdirAll` from stdlib. The `skillManifestEntry` struct (name, version, checksum) already exists for tracking.

**Confidence: HIGH** -- All primitives exist. Assembly work on existing patterns.

## Feature 7: Trajectory Recording and Compression

### Approach: SQLite Table + Time-Based Compression

Trajectories are phase-level state snapshots stored in the hive SQLite database:

```sql
CREATE TABLE IF NOT EXISTS trajectories (
    id          TEXT PRIMARY KEY,
    colony_id   TEXT NOT NULL,
    phase       INTEGER NOT NULL,
    snapshot    TEXT NOT NULL,       -- JSON blob of colony state at phase boundary
    metrics     TEXT DEFAULT '{}',   -- JSON blob of metrics (tests, coverage, etc.)
    created_at  TEXT NOT NULL,
    UNIQUE(colony_id, phase)
);
```

### Compression Strategy

No external compression library needed. Two approaches, both using stdlib:

**1. Delta encoding** (preferred): Store only the diff between consecutive snapshots.

```go
// pkg/hive/trajectory.go
func compressTrajectory(prev, curr []byte) ([]byte, error) {
    // If previous snapshot exists, store only changed fields
    var prevMap, currMap map[string]interface{}
    json.Unmarshal(prev, &prevMap)
    json.Unmarshal(curr, &currMap)

    delta := make(map[string]interface{})
    for k, v := range currMap {
        if prevVal, ok := prevMap[k]; !ok || !reflect.DeepEqual(prevVal, v) {
            delta[k] = v
        }
    }
    return json.Marshal(delta)
}
```

**2. TTL pruning**: Old trajectory entries (older than colony lifetime + 30 days) are deleted on colony seal. This is a simple `DELETE FROM trajectories WHERE created_at < ?` query.

**3. Aggregation**: On colony seal, compress the full trajectory into a summary: phase count, average confidence trend, total learning count. Stored as a single summary row.

**Confidence: HIGH** -- SQLite handles this naturally. Delta encoding is ~30 lines of Go with `encoding/json`. No external dependencies.

## Package Structure

### New Package: `pkg/hive/`

All hive learning functionality goes in a single new package to keep concerns isolated:

```
pkg/hive/
├── store.go          # SQLite connection, schema migration, CRUD
├── fts.go            # FTS5 query helpers (search, rank, snippet)
├── policy.go         # Learning rules and evidence validation
├── trajectory.go     # Trajectory recording and delta compression
├── privacy.go        # Secret scanning for content before storage
├── skill_lifecycle.go # Create/patch/archive/promote operations
├── store_test.go
├── fts_test.go
├── policy_test.go
├── trajectory_test.go
└── privacy_test.go
```

### Extended Packages

| Package | Extension |
|---------|-----------|
| `pkg/codex/` | `pid_registry.go`, `heartbeat.go` (process lifecycle) |
| `pkg/memory/` | No changes -- called from `pkg/hive/policy.go` |
| `cmd/` | New cobra commands for hive operations, heartbeats, skill lifecycle |

## Installation

```bash
# Core (one new dependency)
go get modernc.org/sqlite@latest

# The libc dependency is pulled automatically by Go module resolution.
# Verify it matches: check modernc.org/sqlite's go.mod for the required version.
```

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| SQLite driver | modernc.org/sqlite | mattn/go-sqlite3 | Requires CGO, breaks cross-compilation and single-binary distribution |
| SQLite driver | modernc.org/sqlite | zombiezen/go-sqlite | Not a database/sql driver, custom API, would need to rewrite all query code |
| Secret scanning | Extend existing regex | gitleaks (library) | 150+ transitive deps, overkill for content-level scanning |
| Rule engine | Go structs | Cedar (AWS) | Authorization-focused, not learning policies; adds AWS SDK dependency |
| Rule engine | Go structs | govaluate | Expression parser adds complexity for simple threshold checks |
| Compression | Delta encoding + TTL | gzip/zlib | Trajectory data is structured JSON; delta encoding is more semantically useful than binary compression |
| FTS | SQLite FTS5 | Meilisearch/Typesense | Separate service, overkill for single-user colony recall |
| FTS | SQLite FTS5 | Bleve (Go native) | Good but adds another dependency; SQLite FTS5 already covers the use case |

## Sources

- modernc.org/sqlite FTS5 support: Context7 `/modernc-org/sqlite` docs, SQLite official FTS5 documentation (https://www.sqlite.org/fts5.html), pkg.go.dev/modernc.org/sqlite
- modernc.org/sqlite version: pkg.go.dev (v1.42.1), Reddit r/golang discussion Feb 2026
- Process group management: Go syscall package docs (https://pkg.go.dev/syscall), HackerNoon process management guide (https://hackernoon.com/everything-you-need-to-know-about-managing-go-processes), Felix Geisendorfer child process article (https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773)
- Secret scanning patterns: Gitleaks (https://github.com/gitleaks/gitleaks), TruffleHog (https://github.com/trufflesecurity/trufflehog)
- Existing codebase inspection: `pkg/memory/trust.go`, `pkg/memory/promote.go`, `pkg/codex/process_group_unix.go`, `cmd/security_cmds.go`, `cmd/skills.go`, `pkg/events/event.go`, `cmd/recovery_snapshot.go`
