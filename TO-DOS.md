# TO-DOS

## ✅ 📜🐜🏛️🐜📜 ant:council - COMPLETE

Implemented in commit `ae57031` (v2.3.0)

---

## ✅ 🔥🐜🗡️🐜🔥 ant:swarm - COMPLETE

Implemented in v2.4.0

Features:
- 4 parallel scouts (Git Archaeologist, Pattern Hunter, Error Analyst, Web Researcher)
- Git checkpoint before fix, rollback on failure
- Cross-compare findings, rank by confidence
- Auto-apply best fix
- Learning injection (REDIRECT failed patterns, FOCUS working patterns)
- 3-fix architectural escalation

New utilities in aether-utils.sh:
- autofix-checkpoint / autofix-rollback
- spawn-can-spawn-swarm
- swarm-findings-init / swarm-findings-add / swarm-findings-read
- swarm-solution-set / swarm-cleanup

---

## 📊🐜🗺️🐜📊 ant:plan + 🔨🐜🏗️🐜🔨 ant:build Emoji Styling - 2026-02-09 08:04

- **Update ant:plan and ant:build command descriptions** - Add emoji styling to match the intensity and format of ant:swarm (🔥🐜🗡️🐜🔥). **Problem:** ant:plan and ant:build currently have plain text descriptions while ant:swarm uses the intense 🔥🐜🗡️🐜🔥 emoji style, creating inconsistency in the visual presentation of commands. **Files:** `.claude/commands/ant/plan.md:3`, `.claude/commands/ant/build.md:3`. **Solution:** Add emoji prefixes to the description field in the YAML frontmatter - suggested styles: ant:plan → `📊🐜🗺️🐜📊`, ant:build → `🔨🐜🏗️🐜🔨`. This is a cosmetic-only update with no functional changes to system behavior.
