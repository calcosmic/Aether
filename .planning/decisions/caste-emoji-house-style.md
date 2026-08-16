# Caste emoji house style — the ant returns

**Decision date:** 2026-08-16
**Status:** settled — double emoji restored.

## The decision

Worker identity renders in the classic v5.4.0 house style: **caste glyph
followed by the ant** — `🔨🐜 Builder Mason-67`, `👁️🐜 Watcher Vigil-12`,
`👑🐜 Queen`. The v5.4.0 caste-system.md declared this canonical ("Workers are
displayed as `{caste_emoji} {worker_name}`, e.g. `🔨🐜 Hammer-42`"); the Go
port dropped the trailing ant. Restored in `casteIdentity`
(cmd/codex_visuals.go), which propagates to every render site.

Exception: when the glyph IS the generic ant (unknown caste fallback, and the
handful of command emoji that are plain `🐜`), it stays single — `🐜🐜` was
never the style.

## Churn assessment (honest)

Measured before the change: exactly one literal test assertion
(cmd/ceremony_cmd_test.go "👑 Queen Orchestration", updated with the change),
one hard-coded renderer literal (cmd/ceremony_cmd.go:386, aligned), and three
golden visual fixtures (golden_build/continue/plan.txt, refreshed with
-update-golden because this change caused the mismatch). Well under the
fallback threshold set in the plan (>~10 files → keep single emoji), so the
restore proceeded.

## Lock

`TestCasteIdentityUsesHouseStyle` (cmd/caste_house_style_test.go) fails if the
ant is dropped again, and fails if the fallback ever doubles.
