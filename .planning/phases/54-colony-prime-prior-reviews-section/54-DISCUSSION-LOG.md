# Phase 54: Colony-Prime Prior-Reviews Section - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-26
**Phase:** 54-colony-prime-prior-reviews-section
**Areas discussed:** All deferred to Claude's discretion

---

## Finding detail level

| Option | Description | Selected |
|--------|-------------|----------|
| Counts only | Just "Security (3 open)" per domain | |
| Top finding + file | One-line per domain with highest severity finding and file location | ✓ |
| Full detail | All findings with descriptions | |

**User's choice:** You decide (recommended)
**Notes:** Chose top finding + file — gives workers actionable info within the char budget.

---

## Cache refresh timing

| Option | Description | Selected |
|--------|-------------|----------|
| On ledger-write | Refresh cache after every review-ledger-write call | |
| On colony-prime call | Refresh when colony-prime runs, with mtime check | ✓ |
| TTL-based | Refresh if cache older than N minutes | |

**User's choice:** You decide (recommended)
**Notes:** On colony-prime call with mtime-based staleness check — simple and correct since colony-prime runs infrequently.

---

## Domain prioritization

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed order | Always security, quality, performance, etc. | |
| Severity-first | Domains with HIGH findings first | ✓ |
| Most findings first | Domains with most open findings first | |

**User's choice:** You decide (recommended)
**Notes:** Severity-first ordering — HIGH findings matter most to downstream workers.

---

## Claude's Discretion

- Finding detail level — top finding per domain + file location
- Cache strategy — per-call refresh with mtime check
- Domain ordering — severity-first, domainOrder tiebreaker
- Cache file format, error handling, string formatting

## Deferred Ideas

None.
