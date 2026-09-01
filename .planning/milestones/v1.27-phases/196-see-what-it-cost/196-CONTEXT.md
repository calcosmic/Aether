---
phase: 196-see-what-it-cost
created: 2026-08-27
source: owner decisions + 196-SALVAGE-ASSESSMENT.md
---

# Phase 196 Context — locked decisions

These bind planning. Where a decision came from the owner it says so; where it is a
technical ruling made on the owner's behalf it says that too, so a later reader can
tell which are open to revisit.

## D-01 — What the cost line says (OWNER, 2026-08-27, AMENDED same day)

Every build and continue ends with the run total, a short per-worker breakdown, and an
explicit mark on anything not measured.

**AMENDMENT (owner, 2026-08-27, after the planner surfaced a contradiction).** The
original choice showed an estimated figure beside an `estimated` mark. That collides
with success criterion 3, which promises no figure is guessed from text length — and
prompt-character-count IS the only estimate mechanism in the tree, so a marked estimate
would have been exactly the thing criterion 3 forbids, wearing a label. There is no
honest third source.

Asked to choose, the owner ruled: **show no number at all.** A worker whose tool did
not report usage is rendered as `—  not reported`, and the run total counts only
measured workers and says so.

```
Cost: 1.4M tokens across 3 workers
  Builder Mason-67    1.2M  measured
  Watcher Keen-12     220K  measured
  Scout Roam-90          —  not reported
(1 worker's tool did not report usage)
```

This is the same principle as the original decision, applied consistently: a number
the owner cannot trust is worse than no number. The cost: no rough sense of an
unreported worker's size, and a total that is incomplete rather than approximate —
accepted deliberately.

**Superseded:** the `Scout Roam-90  80K  estimated` shape in the first version of this
decision. No estimated figure is ever rendered. Nothing may derive a token count from a
character or prompt length, anywhere — not in the ledger, not in the renderer, not in
`aether spend`.

**No dollar figure as the headline, and no price table anywhere** — Phase 174's D-02,
still binding. Verified 2026-08-27: no price table or token-to-dollar arithmetic exists
anywhere in the Go source, so the old `pkg/trace/cost.go` blocker is retired rather
than pending. The one dollar value on the salvaged ledger branch
(`spendTotals.ProviderUSD`, a relay of the provider's own reported cost, not a rate
table) is to be **deleted**: nothing asks for it and it is a standing temptation.

## D-06 — Reaching the detail view (OWNER, 2026-08-27)

`aether spend` is surfaced by a plain sentence in the build and continue wrapper
output naming it as the optional detail view. The owner declined a first-class
`/ant-spend` slash command: it would ripple into the command YAML, the help listing and
the documented command counts for a view he expects to want rarely.

This also satisfies the repo's orphan ratchet, which fails a command with no caller.

## D-02 — Model choice (OWNER, 2026-08-27)

Routine roles — the documentation writer, the knowledge-keeper, the accessibility
checker — are pinned to the cheaper model. They must not silently default to
`inherit`, i.e. to whatever model happened to run last. Any worker kept on the
expensive model carries a written reason, shown on the pre-build team card.

Rejected: showing the model choice for confirmation before every build (the owner
chose the pinning without the extra prompt); and leaving model selection untouched,
which would have left criterion 4 unmet.

This does not reopen automatic model *routing*, which the owner declined on
2026-07-28 and which stays out of scope.

## D-03 — The three salvaged branches (assessment-led, owner approved the approach)

The owner chose "review them, then merge what holds up". Verdicts from
`196-SALVAGE-ASSESSMENT.md`:

| Branch | Verdict |
|---|---|
| `worktree-agent-a47f78913caf6fde0` — `cmd/spend_ledger.go` + test | **MERGE-WITH-FIXES.** Compiles and passes green against today's tree unchanged. Remove `ProviderUSD`; add a `JobName` column, because Phase 195 made one worker own several tasks and the ledger has no column for that. Merge FIRST — nothing depends on it and it is what the others write into. |
| `worktree-agent-aa57076cd698da76f` — `cmd/wrapper_usage_opencode.go` + fixtures | **MERGE-WITH-FIXES, and only inside the plan that also wires it.** Both entry points are currently orphans with no caller, and merging unwired code is this repo's named signature failure. Must fix: delete `openCodeWorkerNamePatternCache` (an unsynchronised map read and written per worker — Go aborts the process on a concurrent map write, so this is a hard crash at the end of every build, not a silent race). Must add: a multi-message fixture, because all three existing fixtures have exactly one message and real sessions have twenty-plus, leaving the accumulation path untested. |
| `worktree-agent-a59fd3ee68644ea21` — `cmd/wrapper_usage_claude_test.go` + fixture | **DELETE**, after harvesting the test design. It tests an implementation that has never existed in any ref, so merging it breaks the whole `cmd` package build. Reviving is strictly more expensive than rewriting because there is nothing to revive. |

## D-04 — The Claude transcript must dedupe (CORRECTED 2026-08-27)

**The overcount is real. The mechanism first recorded here was wrong, and the
correction changes what the parser must key on.**

Originally recorded, from the salvage assessment: "the same worker's usage block
appears two or three times, once each on the `user`, `queue-operation` and
`attachment` line types — dedupe by `tool_use_id`."

The plan checker tested that against every one of the owner's 746 real transcripts
under `~/.claude/projects` and it does not hold. Usage-bearing lines by top-level
`type`:

```
assistant  48063
user         250
```

`queue-operation` and `attachment` carry usage **zero times**. The earlier reading
matched a nested inner `type` field on a content block, not the line type — the
"structure, not substrings" mistake this very decision exists to prevent.
`tool_use_id` is essentially absent from usage-bearing lines, so a parser keyed on it
would have deduplicated NOTHING and reproduced the full overcount, while its
fixture-integrity test could never be satisfied from a real capture.

**The corrected rule.** The duplication is real and occurs WITHIN the `assistant`
line type. The key is **`message.id`**, corroborated by `requestId`. Measured on a real
transcript: 50 usage rows, 25 distinct `message.id`; one id appears four times, every
occurrence `type=assistant` with identical `output_tokens=373`.

So: **deduplicate by `message.id` across usage-bearing lines.** Handle both surviving
line types, `assistant` and `user`.

**The fixture must still be a REAL captured transcript that keeps its duplicates**, and
its integrity test asserts that at least one `message.id` occurs more than once among
usage-bearing lines, and that usage-bearing lines of both surviving types are present.
A fixture with each block appearing once cannot fail, which is the 186x-undercount
shape this repo already shipped, in this same subsystem.

**Re-derive the naive and deduplicated totals from the actual capture.** The figures
8,930,280 and 4,237,379 came from the superseded reading and must not be carried
forward as literals.

**Why this is recorded rather than quietly edited:** a wrong mechanism stated
confidently is how this repo's audits describe its own failures. The assessment was
right that there is a duplication trap and right about its size; it was wrong about
where. Both facts belong in the record.

## D-05 — One authoritative token type (TECHNICAL RULING, not an owner call)

There are currently two accounting paths. `pkg/llm.Usage` (`pkg/llm/client.go`) carries
only input and output columns and no cache columns, and `pkg/agent/pool.go` reports
exactly those two. That is a second, cache-blind path already live in the tree — the
186x shape on a different lane.

**Ruling: the spend ledger's own cache-aware row is authoritative.** Before any cost
line is displayed, `pkg/llm.Usage` must either gain the cache columns or stop being
used for accounting; a cost line rendered over two disagreeing sources is a
confidently wrong number, which is worse than no number. Planning must resolve this
BEFORE the closeout line lands, not after.

## What Phase 196 still builds from scratch

From the assessment, roughly four days:

1. The Claude transcript parser, deduping per D-04.
2. The writer that turns a finished dispatch into ledger rows on both build and
   continue — nothing writes a single row today.
3. The one closeout line per D-01, plus `TestBuildEndsWithOneCostLine` and a test that
   fails if a dollar figure appears as the headline.
4. `aether spend` itself, plus `TestSpendDoesNotMutate` — no such command exists, and
   CLAUDE.md makes "an inspection path must not mutate state" a named rule with its
   own history.
5. Criterion 4 in full per D-02 — model reason on the team card, no silent `inherit`,
   `TestRoutineBuilderIsSonnetNeverInherit` and `TestOpusRequiresRecordedReason`.
   Untouched by all three branches.
6. Reconciling the two token types per D-05.

## Standing constraints carried in

- CLAUDE.md's **Definition of Done**: a requirement is satisfied only when a command
  exists that FAILS when it is unmet.
- Do not write a test whose assertion re-implements the same arithmetic as the code
  under test. This repo shipped a 186x undercount exactly that way, in this subsystem.
- Owner-facing strings are plain English with repo jargon translated inline.
