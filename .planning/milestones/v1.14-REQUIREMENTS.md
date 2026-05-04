# Requirements: Aether v1.14 Queen Authority

**Defined:** 2026-05-03
**Core Value:** Aether should feel alive and truthful at runtime — the queen makes it feel alive by coordinating autonomously, and truthful by logging every decision.

## v1.14 Requirements

### Auto-Recovery

- [x] **RECV-01**: Queen classifies worker failures as recoverable, requires-attempt, or blocking using deterministic rules (not LLM inference)
- [x] **RECV-02**: Failed workers are automatically retried up to a configurable per-phase budget (default: 3 retries per phase) before escalating to user
- [x] **RECV-03**: On worker failure, queen redistributes the failed task to a peer worker with available capacity before creating a new worker
- [x] **RECV-04**: On gate failure during continue, queen dispatches the Fixer agent automatically to attempt repair before blocking advancement
- [x] **RECV-05**: Queen distinguishes transient failures (timeout, context overflow) from systemic failures (bad task spec, missing dependency) — transient failures retry, systemic failures escalate immediately
- [x] **RECV-06**: All auto-recovery actions are logged to a phase-scoped recovery log with original error, recovery action taken, and outcome

### Smart Gates

- [ ] **GATE-01**: All 11 existing gates are classified as hard_block, soft_block, or advisory — hard_block gates always require user intervention, soft_block gates auto-resolve when safe, advisory gates log but never block
- [ ] **GATE-02**: Security gates (gatekeeper CVE findings) and watcher veto are classified as hard_block and are NEVER auto-resolved, regardless of severity or configuration
- [x] **GATE-03**: Soft_block gates (auditor score below threshold, complexity gate, TDD evidence) auto-resolve after queen verifies the finding is non-critical and logs the decision
- [ ] **GATE-04**: Gate severity thresholds (watcher veto score, auditor minimum score) are configurable via colony config, with documented safe defaults
- [ ] **GATE-05**: Every gate auto-resolution preserves the original finding in an audit trail — original detail, fix hint, and recovery options are never deleted, only annotated with queen's decision

### Clean Output

- [ ] **OUT-01**: At phase end, queen produces a concise summary (not raw worker output) showing: what was attempted, what succeeded, what failed and how it was recovered, what needs human attention
- [ ] **OUT-02**: Every queen decision during a phase is logged to a queen activity audit file (JSON) with timestamp, decision type, input finding, action taken, and rationale
- [ ] **OUT-03**: Build output defaults to filtered mode (summary only); `--verbose` flag shows full worker output for debugging or trust calibration

### Queen Coordination

- [ ] **COORD-01**: Queen manages the wave lifecycle end-to-end — she dispatches waves, monitors worker progress, handles failures within waves, and advances to next wave when ready
- [ ] **COORD-02**: Continue command splits into plan-only phase (queen evaluates gates and decides actions) and finalize phase (queen executes approved actions), with the queen making the transition decision
- [ ] **COORD-03**: Queen operates as single-invocation (not long-running daemon) — she runs, makes decisions, persists state, and returns control to the user between phases
- [ ] **COORD-04**: Queen recovery decisions respect the existing circuit breaker — she never overrides or resets breaker state, and escalates when the breaker trips

## v2 Requirements

### Deferred

- **QUEEN-01**: Queen autonomy levels (full/advisory/manual) — needs user testing to define thresholds
- **QUEEN-02**: Cross-phase queen continuity — queen remembers patterns across phases within a milestone
- **QUEEN-03**: Queen context budget is configurable per-colony — currently proposed at 12K characters
- **QUEEN-04**: Queen coordinates across phase boundaries (automatic build→continue→build transitions)

## Out of Scope

| Feature | Reason |
|---------|--------|
| LLM-based failure classification | Must be deterministic — LLM classification is unreliable and contradicts Aether's "runtime truth" core value |
| Auto-resolving security gates | Security findings require human judgment — auto-resolving them would undermine the colony's trust model |
| Auto-resolving watcher veto | Watcher has final say by design — overriding it turns Watcher into advisory role, contradicting colony principles |
| Queen as long-running daemon | Workers are short-lived subprocess invocations, not services — daemon mode adds complexity without value |
| New external dependencies | All infrastructure exists in Go runtime — adding libraries creates maintenance burden for a wiring problem |
| Override circuit breaker thresholds | Breaker exists to prevent cascading failures — queen must respect it, not bypass it |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| RECV-01 | Phase 94 | Complete |
| RECV-02 | Phase 96 | Pending |
| RECV-03 | Phase 96 | Pending |
| RECV-04 | Phase 96 | Pending |
| RECV-05 | Phase 94 | Complete |
| RECV-06 | Phase 94 | Complete |
| GATE-01 | Phase 93 | Pending |
| GATE-02 | Phase 93 | Pending |
| GATE-03 | Phase 95 | Complete |
| GATE-04 | Phase 95 | Pending |
| GATE-05 | Phase 93 | Pending |
| OUT-01 | Phase 99 | Pending |
| OUT-02 | Phase 99 | Pending |
| OUT-03 | Phase 99 | Pending |
| COORD-01 | Phase 98 | Pending |
| COORD-02 | Phase 97 | Pending |
| COORD-03 | Phase 97 | Pending |
| COORD-04 | Phase 97 | Pending |

**Coverage:**
- v1.14 requirements: 18 total
- Mapped to phases: 18
- Unmapped: 0

---
*Requirements defined: 2026-05-03*
*Last updated: 2026-05-03 after roadmap creation*
