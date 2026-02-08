# TO-DOS

## ✅ 📜🐜🏛️🐜📜 ant:council - COMPLETE

Implemented in commit `ae57031` (v2.3.0)

---

## 🔥🐜🗡️🐜🔥 Add ant:swarm Command - 2026-02-08 22:50

- **Create ant:swarm command** - Stubborn bug destroyer that deploys parallel scouts to research deeply and fix persistent issues. **Problem:** When AI keeps failing to fix something, users need a "nuclear option" that investigates from multiple angles instead of retrying the same approach. **Files:** `.claude/commands/ant/swarm.md` (new), `.aether/data/COLONY_STATE.json`, `.aether/data/constraints.json`.

### Invocation
```
/ant:swarm "<describe the problem>"
```

### Scout Roles (4 parallel)
| Scout | Emoji | Mission |
|-------|-------|---------|
| Git Archaeologist | 🏛️ | `git log -p`, `git blame`, find when it worked, what changed |
| Pattern Hunter | 🔍 | Find similar working code in codebase |
| Error Analyst | 💥 | Parse stack traces, identify root cause patterns |
| Web Researcher | 🌐 | Docs, GitHub issues, Stack Overflow for this error |

### Flow
```
/ant:swarm "<problem>"
      │
      ▼
 Deploy 4 scouts (parallel)
      │
      ▼
 Cross-compare findings
      │
      ▼
 Rank fix options by confidence
      │
      ▼
 Present evidence (nice formatting)
      │
      ▼
 Apply best fix automatically
      │
      ▼
 Auto-inject learnings:
   • REDIRECT: patterns that failed
   • FOCUS: approaches that worked
```

### Requirements
- **Standalone command** - user calls manually when frustrated
- **Parallel scouts** - all 4 investigate simultaneously via Task tool
- **Evidence-based** - show what each scout found before applying fix
- **Auto-apply** - execute the best fix, don't just suggest
- **Auto-learn** - inject pheromones from findings (failed patterns → REDIRECT, working patterns → FOCUS)
- **Source tracking** - tag signals with `source: "swarm:*"` for audit

### Verification Failure Integration
When verification fails during build, offer swarm as an option:
```
🚫 Verification failed. Blocker created: "Tests failing in auth module"

Options:
1. Fix manually
2. Retry (light attempt)
3. 🔥🐜🗡️🐜🔥 Swarm (deep investigation - uses more tokens)
4. Something else?
```

### Safety Review (Complete)
| Aspect | Status |
|--------|--------|
| Spawn system | Safe - separate swarm cap of 6 |
| Flag integration | Safe - read-only during investigation, respects iron law |
| Git safety | Safe - checkpoint before fix, rollback on failure |
| Learning conflicts | Safe - dedup + conflict detection + confidence ranking |

### Implementation Checklist
- [ ] Add utility functions to aether-utils.sh (autofix-checkpoint, autofix-rollback, spawn-can-spawn-swarm)
- [ ] Create `.claude/commands/ant/swarm.md`
- [ ] Create `.opencode/commands/ant/swarm.md`
- [ ] Update README.md
- [ ] Update QUEEN_ANT_ARCHITECTURE.md
- [ ] Update package.json version
- [ ] Test full workflow
