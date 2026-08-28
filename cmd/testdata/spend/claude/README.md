# Claude Code transcript fixture — `transcript-with-duplicates.jsonl`

This is a **real captured Claude Code transcript slice**, redacted but otherwise
untouched. It is not hand-written, and it deliberately **keeps its duplicate
usage blocks**. A fixture that lists each usage block once cannot fail, which is
the same structural failure as the 186x undercount this repository already
shipped in this same subsystem.

Plain English: Claude Code writes one line to its log for every piece of a
reply — the thinking, the text, each tool call — and every one of those lines
repeats the *same* running token count for the whole reply. Add the lines up and
you bill the same reply two or three times. This file is a real, unedited-in-
that-respect example of that, kept so a future change that starts adding the
lines up again fails a test instead of shipping a doubled bill.

---

## Source

| | |
|---|---|
| Captured from | `~/.claude/projects/-Users-callumcowie-repos-Aether/9401193d-46bb-4153-9575-6a42aaae3a99.jsonl` |
| Slice | lines 63–90 inclusive, **contiguous**, one session, nothing reordered |
| Captured on | 2026-08-28 (Phase 196 plan 03) |
| Claude Code version in the capture | 2.1.223 |

The source transcript carries **250 usage-bearing lines across 138 distinct
`message.id`** — the duplication is a property of the whole file, not of the
window chosen here.

## Where usage actually lives (re-measured on this machine, 2026-08-28)

Measured across the **whole corpus** of `~/.claude/projects` — 1,726 transcripts,
1.0 GB — counting every line that carries a usage object, grouped by the line's
**top-level** `type` field:

```
assistant   142581   usage at .message.usage
user           252   usage at .toolUseResult.usage   (a subagent's completion result)
```

**No other top-level line type carries usage anywhere in the corpus.**
`queue-operation` and `attachment` carry it zero times — an earlier reading of
this defect matched a *nested* `type` field on a content block and mistook it for
the line type. That is why the reader must read the top-level field.

The two usage-bearing types differ in shape and both are represented here:

- `assistant` — usage at `.message.usage`, carries `message.id` and `requestId`.
  **This is where the duplication happens:** one line per content block, each
  repeating the whole message's cumulative usage.
- `user` — usage at `.toolUseResult.usage`, a Task-tool subagent's completion
  record. It carries `agentType` / `agentId` but **no `message.id` at all**, so
  it is an identifier-less usage-bearing line and must be counted once on its
  own rather than dropped or folded into another row.

## What this fixture contains

| | |
|---|---|
| Lines | 28 |
| Usage-bearing occurrences | 18 (17 `assistant`, 1 `user`) |
| Distinct `message.id` among them | 9 |
| Identifier-less usage-bearing lines | 1 (the `user` / `toolUseResult` row) |
| Most-repeated `message.id` | `msg_011CdwU6bPPCyfEmc7tF5PXA` and `msg_011CdwW3P6c2VmnuTWw88aJc`, 3 occurrences each |
| Appears exactly once | `msg_011CdwU7f2gTr5bxuGPoGibV`, `msg_011CdwVdsNnmqatgPQtKhXjB`, `msg_011CdwVfLCrd3ZrTip24sWEj` |

Non-usage line types (`last-prompt`, `mode`, `attachment`, and `user` lines whose
`toolUseResult` is an ordinary tool result) are kept, so the fixture also proves
those are correctly *not* counted.

## The two totals, measured on THIS capture

Both are the sum of the four disjoint billed columns
(`input_tokens + cache_creation_input_tokens + cache_read_input_tokens +
output_tokens`) — the same columns `codex.WorkerUsage.BilledTotalTokens()` sums.

| Total | Value | How it was measured |
|---|---|---|
| **Naive** — every usage-bearing occurrence added up | **2,764,260** | `jq` over the fixture emitting one row per usage-bearing line, summed with `awk`. No Go code involved. |
| **Deduplicated** — the truth | **1,572,834** | Same `jq` rows; last occurrence wins per `message.id`; the one identifier-less `user` row counted once. |

Naive is **1.757x** the truth. A reader that stops deduplicating produces
2,764,260 and fails `TestClaudeTranscriptDeduplicatesUsageByMessageID`.

Both figures were derived with `jq`/`awk` directly from this file and
hand-checked by addition. **Neither was produced by calling the code under
test**, and neither is one of the two figures recorded before decision D-04 was
corrected — those came from a mistaken reading of the line types, are named only
in `196-CONTEXT.md`, and must never be reused.

## Redaction rules applied

Every line was rewritten through a `jq` whitelist. Nothing was renumbered,
reordered, or normalised.

**Removed / replaced:**

- `cwd` → `/redacted/project`; `worktreePath` → `/redacted/worktree`
- `gitBranch`, `worktreeBranch` → `redacted-branch`
- Every assistant `text` and `thinking` body, and every thinking `signature`
  → `[redacted]`
- Every `tool_use` `input` → `{"redacted": true}` (the block's `type`, `id` and
  `name` survive)
- Every `tool_result` `content` → `[redacted]`
- `toolUseResult.content` and `toolUseResult.prompt` → `[redacted]`
- Non-usage-bearing line types (`last-prompt`, `mode`, `attachment`) reduced to
  their `type`, `uuid`, `timestamp`, `sessionId` and a `redacted` marker
- A `toolUseResult` that carries no usage reduced to `{"redacted": true}`

**Kept byte-for-byte, because they are the data under test:**

- the top-level `type` of every line
- `message.id`, `requestId`, `uuid`, `parentUuid` (random identifiers, not
  identifying information)
- every `usage` object in full, including `cache_creation`, `iterations`,
  `service_tier` and `speed`
- `toolUseResult.agentType`, `agentId`, `status`, `resolvedModel`,
  `totalTokens`, `totalToolUseCount`, `totalDurationMs`
- `model`, `version`, `timestamp`, `sessionId`, `stop_reason`, `isSidechain`

Verified after redaction: no `/Users/`, no username, no email address, no URL
remains; and the 18 usage rows are byte-identical in type, identifier and billed
total to the same rows in the unredacted source.

## Do not tidy this file

If you find yourself "cleaning up" the repeated `message.id` values so each
appears once, stop. The repetition **is** the fixture.
`TestClaudeTranscriptFixtureContainsRepeatedMessageIDs` fails the moment it is
lost, and it is the test that protects every other test in
`cmd/wrapper_usage_claude_test.go`.
