# Deferred Items — Phase 190

Out-of-scope discoveries found during execution. Not fixed here per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current task's changes).

## From 190-02 (TS-host hive double-injection removal)

### Stale ceremony-adapter snapshot fixtures (pre-existing, unrelated to hive_section)

**Found during:** Task 3 verification (`npm test --prefix .aether/ts-host`)

**Symptom:** Three tests in `.aether/ts-host/test/ceremony-snapshots.test.ts` fail:
- `Go renderSpawnPlan(build) matches snapshot`
- `Go renderWaveStart(build) matches snapshot`
- `Go renderWaveStart(continue) matches snapshot`

**Cause:** The Go runtime's caste-identity rendering (`cmd/codex_visuals.go`) now emits the
house style described in `CLAUDE.md`'s "Caste Identity System" section — emoji + ant glyph
(e.g. `🔨🐜`) plus a model tag (e.g. `[sonnet]`) — but the committed snapshot fixtures under
`.aether/ts-host/test/__snapshots__/` (`spawn-frame-builder.txt`, `stage-separator-build.txt`,
`stage-separator-continue.txt`) still expect the older single-emoji, no-model-tag format
(e.g. `🔨 Builder Bolt-69` instead of `🔨🐜 Builder [sonnet] Bolt-69`).

**Proof this predates Plan 190-02:** `git log` shows the snapshot fixture
(`.aether/ts-host/test/__snapshots__/spawn-frame-builder.txt`) was last touched in commit
`9f40bf48`, while `cmd/codex_visuals.go` was last touched in `a81cf6aa` (2026-08-18, Phase 187
work) — a later commit that both predate this plan's base commit (`b60a0075`). Nothing in
Plan 190-02's diff touches ceremony rendering, caste emoji, model tags, or these snapshot
files.

**Not fixed here because:** Out of this plan's scope (hive_section removal only). The fix is
almost certainly `AETHER_UPDATE_SNAPSHOTS=1 npm test --prefix .aether/ts-host` to regenerate
the three stale fixtures, but that is a distinct, unrelated correctness gap for a future plan
(or a quick standalone fix) to pick up.

### Inert `hive-read` mock branches left in unrelated host-integration.test.ts fixtures

**Found during:** Task 3 test rewrites

**Symptom:** Several unrelated `describe` blocks in `host-integration.test.ts` (e.g. "spawn
orchestrator initialization") have mock `callGoJSON` handlers with a `if (cmd === "hive-read")`
branch returning canned data. Since the TS host no longer calls `hive-read` anywhere (Phase
190), these branches are now dead code — never reached, harmless.

**Not fixed here because:** Scrubbing every last unreachable mock branch across ~8 unrelated
tests is cosmetic cleanup, not required for correctness or for proving the hive_section removal
(the branches don't pair with any assertion about hive content). Left as a minor, low-priority
cleanup opportunity for whoever next touches those tests.
