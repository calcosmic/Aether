# Phase 194: The Queen Decides the Team - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-23
**Phase:** 194-the-queen-decides-the-team
**Areas discussed:** What counts as risky, The forced reviewer, A reason for every helper, Autopilot and your dials

---

## Saved ideas (todo cross-reference)

| Option | Description | Selected |
|--------|-------------|----------|
| Small jobs get the small machine (Recommended) | 2026-08-21 note tagged `resolves_phase: 194`: builds feel heavy for small work; pipeline should fit the task automatically | ✓ (folded) |
| A readable spec before building | 2026-08-20 idea: a command that writes a plain-English spec for sign-off; Phase 193 already parked it as a later milestone | ✓ at first — then parked (see below) |
| Slow start-up check timeout | 2026-08-01 plumbing note, hard-coded 20s wait in the TS host; matched on generic words only | |

**User's choice:** folded both of the first two.
**Notes:** Claude flagged the spec idea as a new capability (scope guardrail); see next entry.

---

## Spec idea — scope check

| Option | Description | Selected |
|--------|-------------|----------|
| Keep it parked (Recommended) | Recorded as deferred; phase stays focused on who gets sent and why | ✓ |
| Fold it in anyway | Phase grows to include a new user-facing spec command; showdown proof waits on it | |

**User's choice:** Keep it parked.

---

## Area selection

All four proposed areas selected: What counts as risky · The forced reviewer · A reason for every helper · Autopilot and your dials.

---

## What counts as risky

| Option | Description | Selected |
|--------|-------------|----------|
| The five named in your ruling (Recommended) | Passwords/logins, payments, deleting data, database structure changes, release sign-off (ruling D11) | ✓ (decided by Claude) |
| The five, plus the program's own core engine | Adds Aether's own state-machine / dispatch work (today's "wide blast radius" list) | |
| Only logins and releases (today's list) | Smallest set, nothing new to detect | |

**User's choice:** "unsureee", then — after worked examples (CSV export, password reset, refund button, delete inactive accounts, change how Aether saves progress) — "i dont know....".
**Notes:** Claude took the decision on the owner's behalf: the five from D11, reversible (one table), visible on every firing. Owner did not object.

The remaining questions in this area were decided by Claude and presented as picks (locked below): plan wording decides up front; changed files may add a reviewer at the checking step, never remove one.

---

## The forced reviewer

Decided by Claude and presented as picks (locked): one reviewer per signal (security reviewer for logins/payments/release; code reviewer for deletion/migrations); runs at the checking step after the build, announced on the pre-build card; "production work" alone adds nobody.

| Option (last phase of a plan) | Description | Selected |
|--------|-------------|----------|
| Treat it like any other phase (Recommended) | Reviewers only if risky or the Queen asks; "last" is not a risk by itself | ✓ |
| Keep the full panel on the last phase | Security + code + coverage reviewers always run before a plan is done; costs three reviewers per plan | |

**User's choice:** Treat it like any other phase.

---

## A reason for every helper

Decided by Claude and presented as picks (locked): a helper named without a reason is refused by name and the rest are sent; a generic job description is not a reason; the automatic engine must word a plain-English reason or not send the helper.

| Option (false-alarm waiver) | Description | Selected |
|--------|-------------|----------|
| Only you, on the pre-build card (Recommended) | Card asks keep-or-waive; waiver recorded with reason; assistant cannot waive | ✓ |
| The assistant can, with a reason you see | Less friction; safety net depends on the assistant's judgement that day | |
| Nobody — forced means forced | A misfire costs one reviewer's tokens | |

**User's choice:** Only the owner, on the pre-build card.

---

## Autopilot and your dials

Decided by Claude and presented as picks (locked): with no proposal the program sends the code-writer plus any forced reviewer and stops guessing from keywords; "heavy" = the full review panel at the checking step; "light" drops optional reviewers but never a forced one or a free check; the "small jobs" note is covered without a mode switch.

| Option (one-helper team card) | Description | Selected |
|--------|-------------|----------|
| Show a one-line card, don't pause (Recommended) | See who's going and why; build doesn't wait | |
| Pause and ask, same as today | Every build waits for the owner's OK, even one-helper jobs | ✓ |
| Skip the card entirely | One-helper jobs just start | |

**User's choice:** Pause and ask, same as today.

---

## Lock-in

| Option | Description | Selected |
|--------|-------------|----------|
| Lock them all and write the notes | Write the decisions file; ready to plan | ✓ |
| I want to change some first | Adjust before writing | |
| Explore more areas first | Suggest 2-3 further areas | |

**User's choice:** Lock them all and write the notes.

---

## Claude's Discretion

- The risk list itself (owner: "I don't know") — chose the five from D11.
- Signal vocabulary and file-path patterns; the per-worker reason flag shape; the one-scout discovery fallback; manifest field names; ledger wording; CLAUDE.md and wrapper rewrites; retire-or-rewrite for `TestHighRiskPhaseKeepsBothReviewers`.
- Format note: after two "I don't know" answers, Claude switched from one-question-per-turn to presenting its picks as a list with a single lock-in question, keeping only the three owner-taste questions (false-alarm waiver, last phase, one-helper card).

## Deferred Ideas

- Spec-writing command for owner sign-off before building (v1.28+).
- Per-project custom risk words with their own UI.
- Cost-line and card rendering of the recorded reasons (Phases 196–198); task grouping (195).
