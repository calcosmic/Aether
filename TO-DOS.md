# TO-DOS

## 📜🐜🏛️🐜📜 Add ant:council Command - 2026-02-08 22:34

- **Create ant:council command** - Emergency injection for intent clarification that auto-injects pheromones. **Problem:** User needs way to clarify intent mid-workflow via multi-choice questions, with answers translated into colony constraints. **Files:** `.claude/commands/ant/council.md` (new), `.aether/data/COLONY_STATE.json`, `.aether/data/constraints.json`.

### Requirements
- **Invocable anytime** - works in any state including mid-build (EXECUTING)
- **Best-effort during build** - if called during EXECUTING, inject pheromones immediately; current workers continue with old rules, future work uses new constraints
- **Auto-inject pheromones** - translate answers to FOCUS/REDIRECT/FEEDBACK signals
- **User-driven scope** - present category menu (project type, quality priorities, domain constraints, etc.) where user picks topics OR enters custom, then drill into multi-choice questions
- **Resume capability** - return to prior workflow after council completes
- **Source tracking** - tag injected signals with `source: "council:*"` for audit

### Safety (Multi-Agent Reviewed)
| Concern | Status |
|---------|--------|
| State corruption | Safe - use existing `file-lock.sh` + `atomic-write.sh` |
| Pheromone conflicts | Safe - add deduplication + conflict detection |
| Iron law verification | Preserved - council decisions can become flags |
| Worker interruption | N/A - workers can't be paused, best-effort injection instead |

### Integration Pattern
```
Any state (including EXECUTING)
      │
      ▼
/ant:council
      ├─ Category menu (AskUserQuestion)
      ├─ Drill-down multi-choice questions
      ├─ Auto-inject:
      │    • FOCUS → constraints.json (max 5, dedup)
      │    • REDIRECT → constraints.json (max 10, dedup)
      │    • FEEDBACK → COLONY_STATE.json signals + instincts
      └─ Return to prior workflow
```

### Key Utilities to Reuse
- `.aether/utils/file-lock.sh` - acquire_lock/release_lock
- `.aether/utils/atomic-write.sh` - atomic_write with JSON validation
- `.aether/aether-utils.sh` - activity-log, flag-add functions
- Existing command patterns in `focus.md`, `redirect.md`, `feedback.md`

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
