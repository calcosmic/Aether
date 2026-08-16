# SEE-07 — Why caste identity felt absent (diagnosis before fix)

**Decision date:** 2026-08-16
**Status:** diagnosed, and the fix shipped in the same restoration pass.

The roadmap required a written diagnosis before any further identity work,
because colours already rendered (`AETHER_FORCE_COLOR=1` worked) — so "add
colours" was not the missing piece. The real causes, verified against the
render sites:

1. **Identity only appeared inside build waves.** `casteIdentity` (emoji +
   coloured label) rendered in the build spawn plan and wave lines — and
   almost nowhere else a user looks. The continue worker flow printed a bare
   `Name [caste]` tag (cmd/codex_visuals.go, `renderContinueWorkerFlowLine`);
   history had no identity at all; the autopilot printed nothing between
   start and finish. A user whose main loop is `run`/`continue`/`status`
   could go a whole session without seeing a single ant.
2. **The ant itself was gone.** The v5.4.0 house style was glyph + ant
   (`🔨🐜 Hammer-42`, canonical in caste-system.md). The Go port kept only
   the glyph, so even where identity rendered it read as generic tooling
   icons, not a colony.
3. **The identity moments were missing.** Classic had spawn announcements
   (`──── 🔨🐜 Spawning 3 Builders in parallel ────`) at the moment of
   dispatch. The Go runtime showed the plan before and results after — the
   "alive right now" beat in between did not exist.

**Fixes shipped against this diagnosis:** double-emoji house style restored in
`casteIdentity` (`TestCasteIdentityUsesHouseStyle`); continue worker lines
carry full identity (`TestContinueWorkerFlowLineCarriesCasteIdentity`); spawn
announcements at dispatch (`renderSpawnAnnouncement`); autopilot narrates the
whole run (`TestRunAutopilotStreamsAdvancement`); history renders the classic
feed (`TestHistoryRendersClassicActivityLines`).
