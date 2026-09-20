---
phase: 196-see-what-it-cost
requirement: COST-05
criterion: 5 (three abandoned branches reviewed, each merged or deleted with a written reason, none left dangling)
recorded: 2026-08-28
plan: 196-08
source: 196-SALVAGE-ASSESSMENT.md (the assessment), 196-CONTEXT.md D-03 (the approved verdicts)
---

# Phase 196 — Disposition of the Three Preserved Spend Branches

Three branches were sitting in this repository holding half-finished work on
measuring what a run costs. All three were created on 2026-08-21 by the same
rescue commit, which preserved uncommitted work before a stale worktree was
removed. Each has now been reviewed, its content either taken into the tree or
deliberately discarded, and the branch itself deleted.

**Every branch's tip commit was recorded here BEFORE its branch was deleted.**

**Correction, 2026-08-28 — how recoverable these commits actually are.** This
section first said the work "can still be recovered by `git show <commit>` long
after the name is gone. That is what makes a deletion safe rather than final."
That was true on the day it was written and is not a durable fact, so it is
corrected here rather than quietly softened.

What is true: deleting a branch name does not delete its commits, and all three
tips below still resolve today (`git cat-file -t <commit>` returns `commit` for
each).

What is also true: none of the three is reachable from any ref in this
repository, and no reflog entry mentions them either — checked on 2026-08-28,
`git for-each-ref` walked in full, zero hits in `git reflog --all`. Unreachable
objects are what `git gc` exists to delete. With this repository's settings (the
git defaults: unreachable loose objects pruned after two weeks,
`gc.reflogExpireUnreachable` at 30 days), a `git gc` — which git also runs on its
own, unprompted, during ordinary commands — will eventually remove all three.
After that `git show` on these hashes returns "bad object" and the work is gone.
The three hashes below are a record of what was disposed of, not a guaranteed
restore point.

**What it would take to keep them recoverable.** One command per branch, giving
each commit a ref so it stops being garbage:

```
git tag archive/spend-ledger-salvage        be1e160b09109c2af56f5df6c8a1adee5b002a5a
git tag archive/opencode-usage-salvage      ecd98b5cc81d0c51b82c04139df14d90e38e4619
git tag archive/claude-usage-test-discarded 1cbf3615100203be7d59d49ffc2cf224367cedaa
```

A tag is a ref, and `git gc` never prunes what a ref reaches, so tagged commits
survive indefinitely (and are pushed with `git push --tags` if they should
survive this machine too). Those tags have deliberately NOT been created: the
two salvaged branches' content is already in the tree and under test, and the
third was reviewed and discarded on purpose. The choice on record is therefore
"accept that these three commits will be pruned", made knowingly — not "they are
safe forever", which is what the earlier wording implied.

## The three

| Branch | Tip commit | What it held | Verdict |
|---|---|---|---|
| `worktree-agent-a47f78913caf6fde0` | `be1e160b09109c2af56f5df6c8a1adee5b002a5a` | `cmd/spend_ledger.go` (266 lines) + its test (530 lines) | **SALVAGED** into the tree by plan 196-01, then deleted |
| `worktree-agent-aa57076cd698da76f` | `ecd98b5cc81d0c51b82c04139df14d90e38e4619` | `cmd/wrapper_usage_opencode.go` (441 lines) | **SALVAGED** into the tree by plan 196-05, then deleted |
| `worktree-agent-a59fd3ee68644ea21` | `1cbf3615100203be7d59d49ffc2cf224367cedaa` | `cmd/wrapper_usage_claude_test.go` (232 lines) + a fixture | **DISCARDED**, its test design harvested into plan 196-03 first, then deleted |

---

## `worktree-agent-a47f78913caf6fde0` — the record of what each worker cost

**Tip commit:** `be1e160b09109c2af56f5df6c8a1adee5b002a5a`
**Disposition:** salvaged into the working tree by plan 196-01, branch deleted.
**Where its content now lives:** `cmd/spend_ledger.go`, `cmd/spend_ledger_test.go`.

**Why it was kept.** This branch held the part that writes down, and later reads
back, what each worker's tools reported a run cost. It already worked: dropped
into a copy of today's tree it compiled and every one of its own tests passed,
unchanged. Rewriting it from scratch would have taken most of a day and would
probably have got one particular decision wrong the first time — the decision to
keep the building pass and the checking pass in two separate files, so that
checking a phase cannot wipe out the record of building it. The branch's own
notes say that mistake was made once and corrected, which is exactly the kind of
knowledge that is expensive to rediscover and cheap to keep.

It was taken in with four corrections, all made in plan 196-01: a stray dollar
figure was removed (nothing asked for it and its presence was a standing
temptation to put money on a screen where the owner ruled money must not go); a
column was added recording which piece of work a worker owned, because Phase 195
made one worker able to own several tasks at once; a second arithmetic check was
added that adds the four raw columns up by hand rather than by calling the same
function it is checking; and one loose vocabulary was tied to the one the rest of
the system already uses.

## `worktree-agent-aa57076cd698da76f` — reading OpenCode's own session records

**Tip commit:** `ecd98b5cc81d0c51b82c04139df14d90e38e4619`
**Disposition:** salvaged into the working tree by plan 196-05, branch deleted.
**Where its content now lives:** `cmd/wrapper_usage_opencode.go` and its tests
and fixtures under `cmd/testdata/spend/opencode/`.

**Why it was kept.** OpenCode writes its own record of every session to disk, and
this branch had worked out how to find and read it — an undocumented layout that
took real reverse-engineering. The assessment checked its conclusions against
4,385 real session records on this machine and every one of its assumptions
held, including the arithmetic that had produced a 186-times-too-small figure
somewhere else in this system once before. Rewriting it would have meant redoing
that reverse-engineering from nothing.

It was taken in only inside the plan that also connected it to something,
deliberately: merging code that nothing calls is this repository's own signature
failure, named in its audits, and this branch's two entry points had no caller at
all. Two corrections were required first. A shared lookup table it kept was
removed outright — it was written to and read from by several workers at once,
and Go deliberately kills the whole program when that happens, so it would have
crashed at the end of every build the moment it was used. And a second message
was added to one test fixture, because every existing fixture held exactly one
message while real sessions hold twenty or more, leaving the part that adds them
up completely untested.

## `worktree-agent-a59fd3ee68644ea21` — a test with nothing to test

**Tip commit:** `1cbf3615100203be7d59d49ffc2cf224367cedaa`
**Disposition:** DISCARDED. Its test design was harvested into plan 196-03
first; the branch was then deleted.
**Where its value now lives:** the six requirements it expressed are satisfied by
named tests in `cmd/wrapper_usage_claude_test.go`, written against the real
reader in `cmd/wrapper_usage_claude.go` that plan 196-03 built.

**Why it was discarded** (carried across from `196-SALVAGE-ASSESSMENT.md`, which
is where this reason was first written):

> *Deleted because it is a test with no implementation. The code it tests has
> never existed in this repository, so merging it would break the build rather
> than fail a check — and its fixture no longer matches the shape real
> transcripts use today, and hides a duplication that would make the reported
> cost roughly twice the truth. Its six requirements have been copied into the
> Phase 196 plan, which is the part that had value.*

One correction to the assessment's own reasoning is worth recording beside it.
The assessment was right that the branch's fixture concealed a real duplication
trap, and right about roughly how big it was. It was wrong about the mechanism:
it read a nested field as though it were the line's own type, and so named the
wrong thing to group by. Measured against 1,726 real session files, the right
thing to group by is the message's own identity. The corrected rule is D-04 in
`196-CONTEXT.md`, and the real measured figures are in `196-03-SUMMARY.md`. The
figures 8,930,280 and 4,237,379 that appear in the assessment are an example of a
confidently-stated wrong number and must not be cited as anything else.

---

## What was checked before anything was deleted

1. The salvaged files are present in the working tree
   (`cmd/spend_ledger.go`, `cmd/wrapper_usage_opencode.go`), together with the
   Claude reader written in their place (`cmd/wrapper_usage_claude.go`).
2. Their tests pass:
   `go test ./cmd -run 'Test(SpendLedger|OpenCode)' -count=1`.
3. Every tip commit above was read from git and written into this file first.

Only then were the three branches deleted.

*Recorded by plan 196-08. `cmd/branch_disposition_test.go` fails if any of the
three branches, any tip commit, or any written reason is removed from this file,
and if any of the three branch names still exists in the repository.*
