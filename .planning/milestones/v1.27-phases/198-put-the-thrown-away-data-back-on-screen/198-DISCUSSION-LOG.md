# Phase 198: Put the Thrown-Away Data Back on Screen - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-29
**Phase:** 198-put-the-thrown-away-data-back-on-screen
**Areas discussed:** Live progress while checking, Finishing a project, Heads-up before a build, How much detail by default

---

## Live progress while checking

| Option | Description | Selected |
|--------|-------------|----------|
| Start line, then result line | "Running tests…" then "Tests ✓ 12/12 (4s)"; two lines per check | ✓ |
| Result line only | One line per check when it finishes | |
| Single updating line | In-place redraw; conflicts with one-terminal ruling | |

| Option | Description | Selected |
|--------|-------------|----------|
| Short reason inline | Failure line carries one-line plain-English reason | ✓ |
| Just the ✗ mark | Explanation waits for summary | |
| Full detail immediately | Everything printed live | |

| Option | Description | Selected |
|--------|-------------|----------|
| Reviewers get same treatment | Live start/finish with duration and tool count | ✓ |
| Only the program's own checks | Reviewer timings only in summary | |

**User's choice:** all recommended options.

---

## Finishing a project

| Option | Description | Selected |
|--------|-------------|----------|
| Short state-of-play card | Phases done/total, failing checks, open warnings, what finishing does | ✓ |
| Just the question | Bare y/n | |
| Full closing summary first | Whole final report before the question | |

| Option | Description | Selected |
|--------|-------------|----------|
| Wisdom review before confirm | Lessons shown first, recorded even on no | ✓ |
| After confirm | Review runs as part of finishing | |

| Option | Description | Selected |
|--------|-------------|----------|
| Warn and ask again explicitly | Second explicit question; yes recorded with reason | ✓ |
| Refuse to finish | Must fix or dismiss first | |
| Warn only | Single yes covers it | |

| Option | Description | Selected |
|--------|-------------|----------|
| Autopilot never seals | Stops and hands owner the finish command | ✓ |
| Autopilot may seal if all green | Finishes on its own | |

---

## Heads-up before a build

| Option | Description | Selected |
|--------|-------------|----------|
| Warn and ask | One question: carry on or stop | ✓ |
| Warn and carry on | Printed, build starts anyway | |
| Refuse to start | Must clear blocker first | |

| Option | Description | Selected |
|--------|-------------|----------|
| Hard stops only | Last continue failed, unanswered forced-reviewer waiver, open owner question | ✓ |
| Anything open | Any flag or warning triggers it | |

---

## How much detail by default

| Option | Description | Selected |
|--------|-------------|----------|
| All of it, compact | Every item every time in house style with (+N more) caps | ✓ |
| Headline + a 'more' command | Separate detail command | |

| Option | Description | Selected |
|--------|-------------|----------|
| Requirement + what proved it | "✓ Login works — proved by: …" | ✓ |
| Requirement + tick only | Bare tick | |

---

## Old notes reviewed

Owner ticked all three keyword matches. Folded: "worker turnaround is too slow" (as a no-added-waiting constraint). Recorded as deferred, not folded: spec-builder command (parked v1.28+ by the milestone brief), ts-host preflight timeout (unrelated to display).

## Claude's Discretion

Wording/ordering of headings; number of recent decisions on the resume card; drift-note phrasing; where the progress hook sits in the continue loop (both lanes must emit); design of the carried-field invariant test and its shrink-only allowlist.

## Deferred Ideas

Spec builder command; ts-host preflight timeout — see CONTEXT.md `<deferred>`.
