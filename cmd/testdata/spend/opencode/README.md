# OpenCode spend fixture

A small, fully redacted stand-in for OpenCode's local session store. It contains
no real session ids, no real paths and no user content: every id, title and
number here was written by hand.

The layout it encodes is **not** guesswork any more. On 2026-08-27 the shape
below was checked, read-only, against the real store on this machine (4,385
session records) as part of the Phase 196 salvage assessment. What was checked
and what was found:

| Assumption this fixture encodes | Checked against the real store |
|---|---|
| Layout is `project/*.json`, `session/<projectID>/*.json`, `message/<sessionID>/*.json` | Confirmed — exact match |
| A project record carries a `worktree` string naming the repository | Confirmed — the record's keys are `id, sandboxes, time, vcs, worktree` |
| A child session carries `parentID` | Confirmed — 3,766 of 4,385 real sessions have one |
| A child session's title carries the deterministic worker name | Confirmed — 1,199 real titles match this fixture's shape, e.g. `🔨 Builder Anvil-20: … (@explore subagent)` |
| `tokens.total` equals `input + output + cache.read + cache.write` | Confirmed on a real 23-message session: the sum of `total` and the sum of the four columns were both 1,496,525, exactly equal |

That last row is why `tokens.reasoning` is present in the fixture but never read:
including it would have broken the equality above, so the reader deliberately has
no column for it.

## The two-message session

`message/ses_child_a/` holds **two** assistant messages on purpose. Every session
in the first version of this fixture held exactly one, which left the reader's
accumulation across messages proven by nothing — while the real session measured
above had twenty-three. If OpenCode ever reported a running cumulative total per
message instead of a per-message figure, a single-message fixture would stay
green while the reported cost multiplied by the message count.

Hand-summed, the session totals:

| Column | msg_1 | msg_2 | Sum |
|---|---|---|---|
| input | 543 | 1,201 | 1,744 |
| output | 123 | 806 | 929 |
| cache.read | 19,770 | 30,500 | 50,270 |
| cache.write | 0 | 4,096 | 4,096 |
| **total** | **20,436** | **36,603** | **57,039** |

1,744 + 929 + 50,270 + 4,096 = 57,039 — the same disjoint-column equality the
real store was checked against.

## The session whose second message states no total

`message/ses_child_c/` holds **two** assistant messages, and the second one
carries every token column and **no `tokens.total` at all**. An aborted or
errored assistant turn is the obvious real candidate for that shape.

It exists because the reader used to accumulate `tokens.total` alongside the
four disjoint columns, and `BilledTotalTokens()` prefers a positive stated total
over the columns — so one record missing `total` shrank the worker's whole
reported spend while still labelling it measured. The fixture had no such
record, so no test could catch it. Measured on this session: 3,300 reported
against a true 10,000.

Hand-summed:

| Column | msg_1 | msg_2 | Sum |
|---|---|---|---|
| input | 200 | 400 | 600 |
| output | 100 | 300 | 400 |
| cache.read | 3,000 | 5,000 | 8,000 |
| cache.write | 0 | 1,000 | 1,000 |
| **stated total** | **3,300** | **absent** | — |

600 + 400 + 8,000 + 1,000 = 10,000, which is what the worker must report.

Do not "fix" msg_2 by adding a total to it. The absence **is** the fixture, and
`TestOpenCodeReaderPrefersTheDisjointColumns` is what it protects.

## Other fixture facts

- `__REPO_ROOT__` in `project/prj_fixture.json` is rewritten by the test to its
  temporary repo root at run time, so worktree discovery is exercised for real
  rather than stubbed.
- `ses_child_c` belongs to worker `Ledge-33`, which no other test requests, so
  adding it left every hand-written sum above untouched.
- `ses_stale` carries the same worker name as `ses_child_a` but sits well outside
  the run window, and holds an impossible 999,999 figure so a leak is obvious.
- Every test that reads this fixture points `HOME` at a `t.TempDir()` first; no
  test in this package ever opens the developer's real OpenCode store.

Owned by Phase 196 plan 05. Separate from `cmd/testdata/spend/claude/`, which is
the Claude Code transcript fixture and is owned independently.
