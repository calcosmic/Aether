---
phase: 197-one-answer-to-what-next
created: 2026-08-28
source: roadmap criteria (already specific) + standing repo decisions
---

# Phase 197 Context

No discussion round was held. The roadmap's six success criteria are unusually
complete — they name the shared logic, the surfaces, the baseline ratchet, and
four of the tests. Adding a discussion round to re-derive decisions the roadmap
already made is the ceremony CLAUDE.md's new proportionality clause exists to
stop. This file records only the standing constraints that bind planning.

## What this phase is, in one line

One piece of logic decides "what next", and every command's closing message —
plus a new-session greeting — is generated from it, instead of ~dozens of places
hand-typing command names that drift and sometimes name commands that no longer
exist.

## Standing constraints that bind this phase

**S-01 — The command-naming chokepoint is real and this phase widens it.**
`/ant-*` translation lives in `translateHintCommandsForPlatform` /
`platformCommandName` (`cmd/codex_visuals.go`) and applies only to verbs in
`wrapperCommandNames`; Codex gets the raw `aether <verb>` form. JSON envelopes
stay raw — wrappers and the TS host EXECUTE those strings, so translating them
would hand `/ant-continue` to exec. The shared "what next" logic must produce a
platform-neutral value and let the visual writer translate, never bake a
platform's spelling into state or JSON.

**S-02 — Criterion 6 is the load-bearing one.** "Never recommend a command that
is not actually available" is the anti-orphan rule pointed at owner-facing
advice. The repo already has the machinery: `TestNoRegisteredSubcommandIsUnreferenced`
and the Cobra command tree. Resolve recommendations against the live tree, not a
list.

**S-03 — Criterion 3's baseline may only shrink.** Same discipline as
`orphan_allowlist.json` and `TestOrphanAllowlistOnlyShrinks`. Record today's real
count, and make the test fail on growth. A ratchet whose baseline can be raised
is not a ratchet.

**S-04 — Wrapper copies are hand-maintained byte-identical triplets.** Any
closing-message change reaching `.claude/commands/ant/*.md`,
`.claude/commands/ant-*.md` and `.opencode/commands/ant/*.md` goes to every copy
identically, in one plan, or a parity test fails.

**S-05 — Owner-facing text is plain English with repo jargon translated inline.**
Every string this phase produces is read by a non-technical owner; CLAUDE.md's
"READ THIS BEFORE YOU WRITE ANYTHING TO THE OWNER" section is binding. A
next-step card that says "seal the colony" without saying what that means is a
defect, not a style preference.

**S-06 — Proportionality (CLAUDE.md, "How much proof a change needs").** The
shared decision logic, the ratchet, and criterion 6's availability check are
full-rigour: they decide what the owner is told to do next. The wording of an
individual card is a passing test. The absolute stands either way — a test must
be able to fail, and a fixture must be in a shape the runtime really produces.

## Explicitly out of scope

- Changing what any command DOES. This phase changes what commands SAY at the end.
- Phase 198's territory (restoring detail the older version showed).
- Automatic model routing (declined 2026-07-28).
