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
