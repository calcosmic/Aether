# Phase 206 Plan 02, Task 3 — Owner Verdict

**Date:** 2026-09-22

## The question as it was put

> Aether can now put its own screens in front of you two different ways. The new way hands the
> screen to Claude Code the moment the command finishes, and Claude Code shows it to you itself —
> no waiting for the chat to repeat it, and no cost. The catch is that Claude Code labels every
> line it shows this way with a fixed prefix of its own, which nobody can change or style. How
> that actually reads on your screen is the only thing that matters here, and it is not something
> a test can judge.
>
> The old way still exists and is untouched: the chat is asked to show the screen itself, and if
> it finishes without doing so it is sent back once to show it. That route already works today.
>
> Before answering, open an ordinary chat in this project, run one command that draws a screen —
> for example `/ant-status` — and look at what appears. Then look at the line at the bottom of the
> window showing which phase you are on and what to run next. Say what you think of both.
>
> Options: `keep-on` (keep the direct route switched on by default) or `fall-back` (turn it off
> by default and keep the existing route that asks the chat to show the screen).

## The owner's answer, verbatim

**`keep-on`**

No accompanying sentence was given.

## What was done about it

Nothing in the runtime changes: the direct screen-delivery route (`aether hook-post-tool-use`,
built in plan 01) stays registered in `.claude/settings.json` exactly as it already was, and the
status line (`aether status-line`, built in this plan's Task 1) stays registered too. Per the
plan's own instruction for a `keep-on` answer, no code or settings changed for this task — only
this record, plus one added sentence in `CLAUDE.md`'s "The screen reaches the owner directly"
section stating the owner looked at it on his own screen on 2026-09-22 and kept it.

**Which route is on by default:** the direct route (systemMessage delivery, no model turn spent)
is on by default. The older Stop-hook backstop route (asking the chat to paste the screen back in
if it never showed it) remains available underneath, unchanged, and still catches anything the
direct route could not deliver — it was never removed and is not being removed by this answer.
