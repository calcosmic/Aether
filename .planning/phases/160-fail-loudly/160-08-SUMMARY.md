# Plan 160-08 — Summary

**Status:** Complete (3/3 tasks)
**Requirements:** LOUD-08 (decision D-02)
**Completed:** 2026-07-28

> **Execution note:** run inline by the orchestrator after the account spend limit stopped
> worktree executors.

## What was wrong

`cmd/unblock_cmd.go:126` told users:

> `3. Run /ant-unblock --dispatch to dispatch Fixer (propose mode by default)`

`/ant-unblock` existed on **no platform**. A user following the runtime's own recovery
guidance hit a dead end.

LOUD-08 allowed either fix — build the wrapper, or reword the message to point at the CLI.
Decision **D-02 chose to build it**, because the Go side was already complete and working:
`aether unblock` with `--phase`, `--dispatch`, and `--fixer-mode` (full/propose/advise). That
fits the milestone's theme — switch on what already exists rather than delete the promise.

## What changed

**Task 1 — the command triple** (following the `preferences` chain exactly):

| File | Role |
|---|---|
| `.aether/commands/unblock.yaml` | YAML source — head of the generation chain |
| `.claude/commands/ant/unblock.md` | Claude Code wrapper |
| `.opencode/commands/ant/unblock.md` | OpenCode wrapper (byte-identical, as the parity model requires) |

**Codex gets no wrapper markdown** — per CLAUDE.md's Platform Policy it is runtime-native,
reaching the same command through `command-guide` and the direct CLI. Verified: `aether
command-guide unblock` now returns the literal passthrough contract for the Codex platform.

The wrapper delegates entirely to the CLI and re-implements nothing. It does add one piece
of judgement the raw CLI cannot: Fixer autonomy defaults to `propose`, and the wrapper is
instructed to state which mode will run before dispatching and never to silently escalate to
`full`. Autonomous repair should be something a user chose, not something they discovered.

**Task 2 — `cmd/command_guide.go`** — `unblock` added to the guided-command set, so all three
platforms describe the same command surface.

**Task 3 — `cmd/slash_command_guidance_test.go`**, two tests:

- `TestSlashCommandGuidancePointsAtRealCommands` — generalises past the one known-bad
  string. **Any** `/ant-*` slash command that Go code tells a user to run must exist as a
  wrapper on both maintained platforms. Fixing only `unblock_cmd.go` would have left the
  next such string free to appear the same way.
- `TestUnblockWrapperIsWiredAndAtParity` — the YAML routes to `aether unblock`, the two
  platform wrappers are identical, and the CLI still has every flag the wrapper documents.
  If `--fixer-mode` were ever removed, the wrapper would start lying; this fails first.

## Deliberate-regression proof

`.claude/commands/ant/unblock.md` moved aside, test re-run:

```
--- FAIL: TestSlashCommandGuidancePointsAtRealCommands
    codex_continue.go tells the user to run /ant-unblock, but
    .claude/commands/ant/unblock.md does not exist — guidance that names a command
    available on no platform is a dead end for the user (LOUD-08)
```

Restored after. Note what the failure names: **`codex_continue.go`**, not `unblock_cmd.go`.
The guidance string appears in more than one place — the generalised test found a second
site that a targeted fix would have missed.

## Goldens

Adding a command shifts the platform catalog. `cmd/testdata/parity_snapshot.json`
regenerated with `-update-golden`; the diff was reviewed before acceptance and contained
exactly four additions, all `"unblock"`, and nothing else.

## Verification

```
go build ./cmd/aether              → OK
go test ./cmd -count=1             → ok (full package, no failures)
aether command-guide unblock       → returns the literal passthrough contract
```

## Files

- `.aether/commands/unblock.yaml` (created)
- `.claude/commands/ant/unblock.md` (created)
- `.opencode/commands/ant/unblock.md` (created)
- `cmd/command_guide.go` (modified — one line)
- `cmd/slash_command_guidance_test.go` (created — 2 tests)
- `cmd/testdata/parity_snapshot.json` (regenerated)

## Follow-up for the release

The wrapper files are hub-published assets. `aether publish` is required before downstream
repos see `/ant-unblock`; until then it exists in this repo only. Flagged rather than run,
since publishing is a release action.
