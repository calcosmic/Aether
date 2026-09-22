# Deferred items — Phase 208

Out-of-scope discoveries surfaced while executing this phase's plans (per
deviation-rule scope boundary: only fixed when directly caused by the
current task's changes).

## 208-01

- `TestNoRegisteredSubcommandIsUnreferenced` fails on `aether codex-native-worker context-ack`
  ("registered but nothing calls it: searched wrappers, menu specs, hooks, scripts").
  Verified via a disposable git worktree pinned to commit `381fe03c` (the
  state immediately before plan 208-01's Task 3) that this failure
  pre-dates this plan's changes entirely — unrelated to `aether report` or
  any refusal-table work. Not fixed here (out of scope for this plan); left
  for a future phase/plan to either wire a real caller or, if genuinely
  dead, remove the command.
