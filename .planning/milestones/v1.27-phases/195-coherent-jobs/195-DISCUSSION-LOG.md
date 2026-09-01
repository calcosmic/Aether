# Phase 195: Coherent Jobs - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-27
**Phase:** 195-coherent-jobs
**Areas discussed:** Grouping boundaries, invalid grouping recovery, partial completion, one-worker fast path

---

## Grouping Boundaries

### Crossing worker-caste boundaries

| Option | Description | Selected |
|--------|-------------|----------|
| Never cross castes | Different worker types always mean separate jobs. | |
| Always follow task relationships | Related tasks become one job even when originally mapped to different castes. | |
| Preserve by default, allow an explicit merge | Automatic grouping keeps caste boundaries; the Queen may assign cross-caste work to one suitable worker with a stated reason. | ✓ |

**User's choice:** Preserve by default, allow an explicit merge.

### Incidental shared files

| Option | Description | Selected |
|--------|-------------|----------|
| Any shared file | One matching path always forms a job. | |
| Meaningful shared files only | Common bookkeeping files alone do not combine unrelated work. | ✓ |
| Shared directory is enough | Tasks in the same folder group without an exact meaningful file match. | |

**User's choice:** Meaningful shared files only. The user typed `BV`; after explicit confirmation, `B` was locked.

### Job size

| Option | Description | Selected |
|--------|-------------|----------|
| Coherence is the only limit | Keep the whole connected cluster regardless of size. | |
| Soft limit with explained exception | Split an unusually large cluster unless the Queen explains why one worker should retain it. | ✓ |
| Hard task-count limit | Never permit a job beyond a fixed number of tasks. | |

**User's choice:** Soft limit with an explained exception.

### Grouping reason

| Option | Description | Selected |
|--------|-------------|----------|
| Connection only | State only that tasks share files or dependencies. | |
| Connection and benefit | Name the relationship and why one worker is better. | ✓ |
| Per-task justification | Give a separate explanation for every task. | |

**User's choice:** Connection and benefit.

---

## Invalid Grouping Recovery

| Decision | Alternatives considered | Selected |
|----------|-------------------------|----------|
| Dependency-order violation | Run with a warning; refuse the group; stop the whole phase | Refuse before dispatch and name the task plus unmet dependency. |
| Scope of repair | Discard the proposal; repair only the affected group; split every task | Keep safe jobs and minimally split the affected group. |
| Reordering | Silently reorder; visibly substitute safe jobs; require a fresh owner decision | Refuse visibly and show the safe replacement jobs. |
| Dependency cycle | Break heuristically; ignore one edge; block for plan repair | Block and name the cycle because no safe order exists. |

**User's choice:** Delegated to Codex after selecting all discussion areas.
**Notes:** The selected behavior preserves useful work without allowing an invalid Queen proposal to pass silently.

---

## Partial Completion

| Decision | Alternatives considered | Selected |
|----------|-------------------------|----------|
| Credit model | Whole job atomic; evidence-backed partial credit; trust named tasks | Credit only tasks with explicit task-level proof. |
| Evidence | Infer from changed files; require task-specific evidence; require owner confirmation | Require task-specific requirements, files, and verification evidence. |
| Retry scope | Retry the whole group; retry unfinished tasks; stop manually | Retry unfinished tasks only after dependency validation. |
| History | Replace original; append linked attempt; keep only final result | Append a new linked attempt and retain the original receipt. |

**User's choice:** Delegated to Codex.
**Notes:** This avoids both wasted rework and fabricated completion. Without a trustworthy per-task receipt, partial credit is refused.

---

## One-Worker Fast Path

| Decision | Alternatives considered | Selected |
|----------|-------------------------|----------|
| Skip condition | Every one-worker manifest; decision-free one-worker manifest only; never skip | Skip only when one implementation worker is present and no owner decision is pending. |
| Visibility | Show nothing; compact summary; full card without pause | Show a compact, non-blocking summary. |
| Owner override | No override; explicit `--checkin`; project preference only | Add `--checkin`; conflict with `--no-checkin` is an error. |
| Group size | Pause above a task threshold; keep fast path; Queen decides each time | Keep the fast path regardless of covered-task count. |

**User's choice:** The owner had already said, "one worker build should go straight through"; detailed safeguards were delegated to Codex.
**Notes:** A forced-reviewer waiver or other unresolved owner decision keeps the blocking check-in, preserving Phase 194's safety control.

---

## Codex's Discretion

- The user explicitly delegated all decisions after the grouping-boundary questions.
- Concrete housekeeping-path policy and soft-size signals remain implementation discretion, subject to CONTEXT.md's observable constraints.
- Manifest/CLI field names and exact plain-English rendering remain implementation discretion.

## Deferred Ideas

- Spec-builder command — separate v1.28+ capability, not part of job grouping.
- Cost/token rendering for grouped jobs — Phase 196.
- Universal closing-card and richer live-display changes — Phases 197–198.
