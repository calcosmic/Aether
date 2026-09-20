# Phase 193: Free Checks Are the Floor - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-22
**Phase:** 193-free-checks-are-the-floor
**Areas discussed:** Projects with nothing to check, When a free check fails, Proving each success criterion, How much to run each time

Questions were grouped into two rounds (3 + 2) at the owner's standing preference for fewer prompts.

---

## Projects with nothing to check

| Option | Description | Selected |
|--------|-------------|----------|
| Pass on what can be checked, say so | Claimed files exist + criterion evidence; card says "no tests to run in this project"; no checker worker sent for lack of tools | ✓ |
| Ask me once per project for a check | Aether asks for a way to verify and remembers it | |
| Send a checker worker, as today | A Watcher reads the work when nothing mechanical can run | |

**User's choice:** Pass on what can be checked, say so (recommended)

---

## When a free check fails

| Option | Description | Selected |
|--------|-------------|----------|
| Stop and tell me the exact next step | Show the failed check and one command; nothing spawns on its own | |
| Auto-send one builder to fix, then stop | One fix attempt without asking; if it still fails, stop and tell | ✓ |
| Send a Watcher to diagnose first | Today's behaviour: a checker reads the failure before any fix | |

**User's choice:** Auto-send one builder to fix, then stop
**Notes:** Recorded as distinct from the 2026-08-21 "recovery decides but never retries" ruling (that one is about recovery re-running failed/interrupted workers; this is one bounded fix at the floor).

---

## How much to run each time

| Option | Description | Selected |
|--------|-------------|----------|
| Targeted per phase, full at the end | Checks scoped to what the phase touched; whole suite on the final phase and at seal | ✓ |
| Full suite every phase | Certain but slow | |
| Full suite, but with a time cap | Run everything; past a limit switch to targeted and say so | |

**User's choice:** Targeted per phase, full at the end (recommended)

---

## Proving each success criterion

| Option | Description | Selected |
|--------|-------------|----------|
| Bind a check at plan time, fall back to builder evidence | Plan-time check per criterion where possible; otherwise re-run the builder's reported command and confirm claimed files | ✓ |
| Builder evidence only, re-run by the program | No plan-time binding | |
| A check per criterion, or the plan is refused | Planning stops for unbindable criteria | |

**User's choice:** Bind a check at plan time, fall back to builder evidence (recommended)

**Follow-up — a criterion no machine can prove:**

| Option | Description | Selected |
|--------|-------------|----------|
| Advance, but flag it as "needs your eyes" | Phase moves on; card lists it; seal blocks until confirmed | (Claude's pick) |
| Block until a reviewer checks that one criterion | A reviewer worker for just that criterion | |
| Pass on the builder's say-so | Trust the report | |

**User's choice:** "you decide" → Claude chose "advance, flag as needs your eyes, seal blocks until confirmed" (no worker spent; owner stays the judge).

---

## Claude's Discretion

- Unprovable criteria (above).
- Scoping heuristics for targeted runs; "skipped: no command" vs failed; card data fields.
- `--skip-watchers` keeps its name with corrected help text.

## Deferred Ideas

- Team composition and forced reviewers → Phase 194.
- Live progress display → Phase 198.
- General large-output externalisation → later stage.
