# SEE-03 — What "live worker visibility" means on this platform

**Decision date:** 2026-08-16
**Decided by:** owner (direct instruction: "i dont want either its just for
better visual in the one terminal you are in")
**Status:** settled — this closes the written-decision gate Phase 168
criterion 3 required before any live-panel work.

## The decision

Live visibility is **better output in the one terminal the user is already
in**. Commands stream their own story as they run. There is:

- **no tmux session** (the v5.4.0 4-pane watch is not restored),
- **no second-terminal watch emphasis** (`aether watch` stays as the modest
  snapshot/interval surface it already is, essentially untouched),
- **no repaintable panel** under wrapper invocation.

## The mechanism

All live output is **append-only, one line (or one small block) per event**,
emitted by the Go process through `emitVisualProgress` / `writeVisualOutput`
(cmd/codex_visuals.go). This is the oracle-progress pattern
(cmd/oracle_progress.go:11-22 records the rationale): telemetry and rendering
are never gated on output mode in a way that can silently kill them, and
append-only lines survive every transport.

Why not in-place redraw: wrapper invocations run the binary under a Bash pipe,
not a TTY. Anything that repaints (ANSI cursor movement, screen clears)
degrades to garbage or invisibility there. The narrator
(cmd/narrator_launcher.go + .aether/ts/src/narrator.ts), which does in-place
redraw, therefore stays confined to the genuine-TTY in-process build path and
is not expanded.

## Consequences

- The autopilot, build, continue, and lifecycle commands each narrate their
  progress as scrollback the user can read and scroll, in the terminal where
  they typed the command.
- The v5.4.0 4-pane tmux watch (status / progress / spawn tree / activity log)
  is formally retired, not deferred. Its information lands in the streamed
  command output and the sectioned displays instead.
- Any future proposal for a repaintable in-session panel must overturn this
  decision in writing first.
