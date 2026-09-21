# Deferred items — Phase 206 (Screens Reach the Owner)

Out-of-scope discoveries logged per the executor's scope-boundary rule (fix only what the
current task's changes touch; log, don't fix, everything else).

## 206-02 Task 1 — pre-existing orphan unrelated to the status line

`go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced` reports one real orphan that
predates this plan and is untouched by it:

```
aether codex-native-worker context-ack is registered but nothing calls it (searched: wrappers, menu specs, hooks, scripts)
```

- **Origin:** commit `1f905181` ("feat: require source-bound native context reads and
  acknowledgements"), part of the still-in-progress Phase 204.2 (Codex Native Worker
  Lifecycle) work recorded in `.planning/STATE.md` ("Phase204.2 Plan25 executing").
- **Why out of scope here:** `git status --porcelain` before this plan's own changes shows no
  file touching `codex-native-worker` or `context-ack`; the reachability scanner's own logic
  for Go-source/script callers is unchanged by this plan (only the JSON-settings decoder was
  extended, to recognize the new `statusLine` top-level settings key — see the fix below).
  Fixing a different phase's in-flight orphan is out of this plan's scope boundary.
- **Verification it is genuinely pre-existing:** the scanner change made in this plan
  (`hookSettingsCommandArgs` in `cmd/subcommand_reachability_ratchet_test.go`) only adds
  recognition of the `statusLine.command` settings key; it cannot affect resolution of a
  Go-registered command with no settings/wrapper/script caller at all. Re-running the same
  test before this plan's `statusLine` recognition fix (i.e. with only the `status-line`
  orphan present) showed exactly two failures, one of which (`status-line`) this plan's own
  fix resolves; the other (`codex-native-worker context-ack`) is untouched either way.

**Disposition:** left red, not allowlisted (the orphan allowlist is shrink-only per
`TestOrphanAllowlistOnlyShrinks`, and widening it to hide a true finding is exactly what that
ratchet exists to prevent). Belongs to whichever plan finishes wiring Phase 204.2's native
context-acknowledgement path.

## Fix made in scope: the reachability scanner did not recognize the new `statusLine` key

`hookSettingsCommandArgs` (the JSON hook-settings decoder inside the reachability ratchet)
only parsed the `hooks` top-level key. This plan adds a second top-level settings key,
`statusLine`, which the platform executes exactly the same way it executes a hook's command.
Without teaching the scanner about it, `aether status-line` would show up as a false orphan
forever — not a defect in the status line, but a gap in the scanner's own settings-shape
knowledge. Fixed in the same commit as Task 1 (see `cmd/subcommand_reachability_ratchet_test.go`).
