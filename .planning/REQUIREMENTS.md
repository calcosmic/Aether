# Requirements: Aether v1.9 Review Persistence

**Defined:** 2026-04-26
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.9 Requirements

### Continue-Review Reports

- [x] **CONT-01**: Continue-review workers produce per-worker `.md` outcome reports at `build/phase-N/worker-reports/{name}.md`, mirroring the existing build worker report pattern
- [x] **CONT-02**: `codexContinueWorkerFlowStep` struct includes `Blockers []string`, `Duration float64`, and `Report string` fields for carrying detailed data through the continue flow
- [x] **CONT-03**: `codexContinueExternalDispatch` struct includes `Report string` field so the wrapper can pass each worker's full markdown findings through the completion packet
- [x] **CONT-04**: `mergeExternalContinueResults()` preserves all three new fields (`Blockers`, `Duration`, `Report`) when merging external worker results into the flow
- [x] **CONT-05**: Wrapper completion packet instructions in `.claude/commands/ant/continue.md` and `.opencode/commands/ant/continue.md` document the `report` field with guidance to include full structured findings as markdown
- [x] **CONT-06**: Old completion packets without `report`/`blockers`/`duration` fields still work (backward compatible via `omitempty`)

### Domain Ledger

- [x] **LEDG-01**: `review-ledger-write --domain <d> --phase <N> --findings <json>` creates domain ledger if missing, assigns deterministic IDs, appends entries, recomputes summary
- [x] **LEDG-02**: `review-ledger-read --domain <d> [--phase <N>] [--status open]` reads ledger entries with optional phase and status filters
- [x] **LEDG-03**: `review-ledger-summary` returns one-line summary per domain showing total, open, and by-severity counts (for colony-prime injection)
- [x] **LEDG-04**: `review-ledger-resolve --domain <d> --id <id>` marks an entry as resolved with timestamp
- [x] **LEDG-05**: Seven domain ledgers exist under `.aether/data/reviews/`: security, quality, performance, resilience, testing, history, bugs
- [x] **LEDG-06**: Ledger entries include: id, phase, phase_name, agent, agent_name, generated_at, status, severity, file, line, category, description, suggestion
- [x] **LEDG-07**: Deterministic IDs use format `{domain-prefix}-{phase}-{index}` (e.g., `sec-2-001`, `qlt-5-003`)
- [x] **LEDG-08**: Each ledger includes a computed summary with total count, open/resolved counts, and by-severity breakdown
- [x] **LEDG-09**: All ledger writes use file-locking atomic writes via `pkg/storage/` (follow pheromone pattern, not hive pattern)
- [x] **LEDG-10**: Agent-to-domain mapping is enforced: Gatekeeper→security, Auditor→quality/security/performance, Chaos→resilience, Watcher→testing/quality, Archaeologist→history, Measurer→performance, Tracker→bugs

### Colony-Prime Integration

- [x] **PRIME-01**: Colony-prime assembles a `prior-reviews` section at priority 8 (between user_preferences at 7 and pheromones at 9)
- [x] **PRIME-02**: Prior-reviews section shows open findings per domain with severity and file/location summary (e.g., "Security (5 open): HIGH — bcrypt... ")
- [x] **PRIME-03**: Prior-reviews section is capped at 800 chars (normal mode) / 400 chars (compact mode) to prevent token budget blowout
- [x] **PRIME-04**: Prior-reviews section gracefully degrades when no review ledgers exist (omitted entirely, not empty)
- [x] **PRIME-05**: Section reads from cached summary file for performance (not 7 direct ledger reads on every colony-prime call)

### Agent Definitions

- [x] **AGENT-01**: Gatekeeper agent definition includes Write tool in `tools:` frontmatter, findings write instructions for security domain, and write-scope guardrails
- [x] **AGENT-02**: Auditor agent definition includes Write tool, findings write instructions for quality/security/performance domains, and write-scope guardrails
- [x] **AGENT-03**: Chaos agent definition includes Write tool, findings write instructions for resilience domain, and write-scope guardrails
- [x] **AGENT-04**: Watcher agent definition includes Write tool, findings write instructions for testing/quality domains, and write-scope guardrails
- [x] **AGENT-05**: Archaeologist agent definition includes Write tool, findings write instructions for history domain, and write-scope guardrails
- [x] **AGENT-06**: Measurer agent definition includes Write tool, findings write instructions for performance domain, and write-scope guardrails
- [x] **AGENT-07**: Tracker agent definition includes Write tool, findings write instructions for bugs domain, and write-scope guardrails
- [x] **AGENT-08**: All 7 agent definitions are synced across 4 surfaces: `.claude/agents/ant/`, `.aether/agents-claude/`, `.opencode/agents/`, `.codex/agents/` (28 file edits total)
- [x] **AGENT-09**: Write-scope guardrails explicitly restrict agents to ONLY write to their designated review ledger files, never source code, tests, or colony state
- [x] **AGENT-10**: Build and continue dispatch flows inject findings-path instructions into review agent task prompts (e.g., "Write findings to `.aether/data/reviews/security/` using `review-ledger-write`")

### Lifecycle

- [ ] **LIFE-01**: `/ant-seal` archives `.aether/data/reviews/` directory alongside existing survey and build archives
- [ ] **LIFE-02**: `/ant-seal` flags high-severity unresolved findings in the seal report
- [ ] **LIFE-03**: `/ant-entomb` includes reviews directory in the chamber archive
- [ ] **LIFE-04**: `/ant-status` displays review ledger counts per domain showing total and open entries
- [ ] **LIFE-05**: Colony init clears stale reviews from prior colony (prevent cross-colony contamination)

## v1.10 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Cross-Colony

- **CROSS-01**: Review findings shared across colonies via Hive Brain (generalized patterns only, not code-specific paths)
- **CROSS-02**: Auto-promotion of high-confidence finding patterns to Hive Brain instincts

### Automation

- **AUTO-01**: Auto-block on critical findings during continue flow
- **AUTO-02**: Automatic finding-to-pheromone promotion
- **AUTO-03**: Bulk resolve by domain or phase in `review-ledger-resolve`

## Out of Scope

| Feature | Reason |
|---------|--------|
| Cross-colony ledger sharing | Findings contain code-specific file paths and line numbers that go stale across repos |
| Auto-block on critical findings | Would create conflicting signals with existing continue-review blocking |
| Auto finding-to-pheromone promotion | Mapping between "finding" and "action" requires judgment, not automation |
| Real-time ledger sync across agents | YAGNI — agents write during build/continue, not concurrently |
| Separate phase-level ledger snapshots | Single ledger with phase field per entry is sufficient (YAGNI) |
| Ledger web UI | CLI-only for now; web dashboard is a future consideration |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CONT-01 | 52 | Pending |
| CONT-02 | 52 | Pending |
| CONT-03 | 52 | Pending |
| CONT-04 | 52 | Pending |
| CONT-05 | 52 | Pending |
| CONT-06 | 52 | Pending |
| LEDG-01 | 53 | Done |
| LEDG-02 | 53 | Done |
| LEDG-03 | 53 | Done |
| LEDG-04 | 53 | Done |
| LEDG-05 | 53 | Done |
| LEDG-06 | 53 | Complete |
| LEDG-07 | 53 | Complete |
| LEDG-08 | 53 | Complete |
| LEDG-09 | 53 | Done |
| LEDG-10 | 53 | Done |
| PRIME-01 | 54 | Pending |
| PRIME-02 | 54 | Pending |
| PRIME-03 | 54 | Pending |
| PRIME-04 | 54 | Pending |
| PRIME-05 | 54 | Pending |
| AGENT-01 | 55 | Done |
| AGENT-02 | 55 | Done |
| AGENT-03 | 55 | Done |
| AGENT-04 | 55 | Done |
| AGENT-05 | 55 | Done |
| AGENT-06 | 55 | Done |
| AGENT-07 | 55 | Done |
| AGENT-08 | 55 | Done |
| AGENT-09 | 55 | Done |
| AGENT-10 | 55 | Complete |
| LIFE-01 | 56 | Pending |
| LIFE-02 | 56 | Pending |
| LIFE-03 | 56 | Pending |
| LIFE-04 | 56 | Pending |
| LIFE-05 | 56 | Pending |

**Coverage:**
- v1.9 requirements: 36 total
- Mapped to phases: 36
- Unmapped: 0

---
*Requirements defined: 2026-04-26*
*Last updated: 2026-04-26 after roadmap creation*
