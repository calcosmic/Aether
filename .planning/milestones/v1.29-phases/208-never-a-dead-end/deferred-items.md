# Deferred items — Phase 208

Out-of-scope discoveries surfaced while executing this phase's plans (per
deviation-rule scope boundary: only fixed when directly caused by the
current task's changes).

## 208-01

- **Status:** acknowledged
- `TestNoRegisteredSubcommandIsUnreferenced` fails on `aether codex-native-worker context-ack`
  ("registered but nothing calls it: searched wrappers, menu specs, hooks, scripts").
  Verified via a disposable git worktree pinned to commit `381fe03c` (the
  state immediately before plan 208-01's Task 3) that this failure
  pre-dates this plan's changes entirely — unrelated to `aether report` or
  any refusal-table work. Not fixed here (out of scope for this plan); left
  for a future phase/plan to either wire a real caller or, if genuinely
  dead, remove the command.

## 208-16 (found during 208-13, written down here per this closing plan's own instruction)

- **Status:** acknowledged
Three things noticed while widening the two safety checks around Aether's
self-recovery mechanism (the code that lets the program quietly finish a
stalled step on its own when nobody is there to ask). None of the three is a
live bug today; each is a risk or a proof gap left for a later phase to pick
up rather than fixed in the moment it was noticed.

- **The self-recovery decision writes down that it succeeded before checking
  that it actually did.** The function that decides "go ahead and recover
  this on your own" (`attemptRefusalSelfRecovery`, `cmd/refusal_self_recovery.go`)
  records the recovery as done the moment it decides to attempt it, not after
  confirming the attempt worked. If a future caller stopped early or hit an
  error partway through, the record would say the recovery happened when it
  hadn't — telling whoever reads it something untrue. Not fixed now because
  the real fix is a design change: the function would need to carry out the
  recovery step itself and only record success afterward, not just decide
  that recovery should happen. **Design change, not a quick fix.**
- **The part of the program that stays silent when talking to a script
  instead of a person only covers half of what it is meant to guard.** When
  nobody is watching, the code that prints progress to the screen
  (`emitVisualProgress`) correctly says nothing. But if a future command is
  added that the program treats as "quiet" (one that doesn't show progress
  at all) and that command later also gains the ability to recover work on
  its own, nothing would print anywhere — even though a person might be
  sitting there watching. Today's only command with this ability (`colonize`,
  the code-survey command) always shows its progress, so this risk doesn't
  happen in practice yet. Not fixed now because there is nothing to
  reproduce — no command triggers it today. **Cheap to close when the first
  "quiet" command gains this ability: add one check covering that command's
  screen output at the same time.**
- **A connection this round relied on is proven only at its first step.**
  This round's fix assumed that turning on "go ahead and rebuild the map
  instead of asking" correctly carries through to stop a second, related
  check from firing later in the same run. Reading the code confirms this
  holds, but no automated check drives it the whole way through — only the
  first step is actually tested. This is the exact area of the program the
  rehearsal keeps stopping near. Not fixed now because it is a proof gap,
  not a known-wrong behaviour. **Cheap to close: add one test that drives a
  recovered survey all the way through the finishing step, rather than
  stopping at the first link in the chain.**
