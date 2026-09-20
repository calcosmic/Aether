# Phase commit drops verified files named only in task receipts — Field Report

**Date:** 2026-09-13 · **Aether:** v1.0.75
**Repo:** `CosmicDashboard` (downstream consumer, reported by a peer session)
**Outcome:** ❌ **Silent partial commit.** Seven verified files were left uncommitted in the
working tree by the phase commit, with no warning from either the build closeout or
`aether continue`.

**Status:** recorded, not yet reproduced locally, not yet fixed. Received while the
Phase 203 wave-3 verification suite was mid-run; deliberately not actioned to avoid
editing `cmd/` source under a suite whose ratchet tests parse that source.

---

## 1. Issue A — the phase commit ignores receipt-level files (PRIMARY)

Phase 2 passed `aether continue`; the phase commit `bce429e` saved **20** files. Seven
verified files stayed dirty and unmentioned:

- `dashboard/lib/google-oauth.ts`
- the integrations status route
- the Connections table component
- four test files

**Cause (reporter states this is confirmed):** the phase commit is built from each
builder's **top-level** `files_created` / `files_modified` only. One builder resent a
corrected report whose top-level lists named only its follow-up fix; its earlier work
was named only in `task_receipts[].files_modified`, so those paths were never in the
commit set.

**Repro:**
1. A builder's report lists files X and Y in `task_receipts[].files_modified` but omits
   them from top-level `files_modified`.
2. Run `build-completion-stage`, then `build-finalize`, then `continue` until it passes.
3. `phase_commit` saves the top-level files only; X and Y remain dirty.

**Workaround downstream:** committed the seven files by hand (`558c523`); their
completion checker now refuses a report whose receipts name files the top-level omits.

**Candidate fixes (reporter's):**
- Build the phase commit from the **union** of top-level and receipt files; or
- have `build-completion-stage` refuse such a report.
- Either way: **warn when claimed files are still dirty after the phase commit.**

## 2. Issue B — the anti-pattern gate reports one hit per file per run

`continue`'s `anti_pattern` gate (exposed-secret scan, `cmd/security_cmds.go`) stops at
the first hit in a file. Fix that line, re-run `continue`, and the next one surfaces —
one per run. A phase-2 test file held 22 lookalike fake values
(e.g. `TODOIST_API_TOKEN: 'todoist-token'`), making that 22 sequential runs. Should
report every hit in a file.

## 3. Why Issue A matters more than one repo

This is the SAME SHAPE as the bug fixed earlier today in v1.0.75
(`6b493e2d`, see `2026-09-13-continue-credits-only-latest-attempt.md`): a worker's
**receipts** carry the truth, a **summary/top-level field** carries a subset, and a
reader consults only the summary. There the forgotten evidence was an earlier attempt's
dispatch record; here it is a task receipt's file list. Two independent reports, two
different readers, one recurring mistake.

CLAUDE.md already states the doctrine this violates: *"A job's task list says what a
worker was asked to do. It never says what got done"* — finishing credit comes from the
two-stage receipt boundary, checked against files actually present. The phase commit is
a third consumer of that same evidence that was never wired to the receipts.

**Worth considering when fixing:** a check that no reader of worker evidence may consult
a top-level summary field where a receipt-level list exists — an invariant, not a patch
to one call site. Otherwise a fourth consumer will repeat this.

## 4. The test this needs

Full rigour by CLAUDE.md's own table — it decides what gets saved, it is receipt-derived,
and being wrong is invisible until work is lost. A builder report whose receipts name
files its top-level omits must produce a phase commit containing those files, proved by
breaking the union and watching it fail. Plus: a dirty-after-commit warning that fires.
