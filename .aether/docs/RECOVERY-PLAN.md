# Aether v3.1 Recovery Plan

**Created:** 2026-02-15
**Context:** Post-diagnosis of destructive update loop
**Status:** **RESOLVED (v4.0)** — The `runtime/` staging directory was eliminated in v4.0. The destructive update loop described below is no longer possible. This document is retained for historical context.

---

## ⚠️ THE PROBLEM: Destructive Update Loop

> **As of v4.0, this problem is resolved.** The flow below describes the pre-v4.0 architecture. The `runtime/` staging directory has been eliminated. `.aether/` is now packaged directly with private dirs excluded by `.aether/.npmignore`.

**You have been working in a broken development cycle (pre-v4.0):**

```
runtime/ (source) ──npm install──→ ~/.aether/ (hub) ──aether update──→ ./.aether/ (working)
     │                                                                    │
     │          ←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←     │
     │                                                                    │
  756 lines                                                        768 lines
  NO emoji section                                                 HAS emoji section
  (stale)                                                          (your work)
```

**What happened (pre-v4.0):**
1. You edit `./.aether/` (working copy) - emojis work, features work
2. You run `npm install` - copies stale `runtime/` to `~/.aether/` (hub)
3. You run `aether update` - copies stale hub to `./.aether/` (DESTROYS your work)
4. Emojis disappear, features break

**Root cause:** Previously `runtime/` was the source of truth, but then `.aether/` became the source of truth. The sync script (bin/sync-to-runtime.sh) auto-populated runtime/ from .aether/ during npm install, but if runtime/ was stale, it overwrote good work.

**Resolution (v4.0):** `runtime/` eliminated entirely. `.aether/` is now packaged directly. The destructive loop cannot occur.

---

## ✅ WHAT WAS BUILT TODAY (v3.1 Open Chambers)

**Milestone Status: COMPLETE** (27/27 requirements satisfied)

### Phase 9: Caste Model Assignment
- `bin/lib/model-profiles.js` - Model configuration library
- `aether caste-models list/set/reset` CLI commands
- `/ant:verify-castes` slash command
- Proxy health verification
- Spawn logging with model tracking

### Phase 10: Entombment & Egg Laying
- `/ant:entomb` - Archive colony to `.aether/chambers/`
- `/ant:lay-eggs` - Fresh colony start
- `/ant:tunnels` - Browse archived chambers
- Chamber utilities: `chamber-create`, `chamber-verify`, `chamber-list`
- Milestone auto-detection (First Mound → Open Chambers → Brood Stable...)

### Phase 11: Foraging Specialization
- Task-based routing (design → glm-5, code → kimi-k2.5)
- Model performance telemetry
- Per-command `--model` override
- Enhanced telemetry with routing decisions

### Phase 12: Colony Visualization
- **Real-time swarm display** (`swarm-display.sh`)
  - Caste colors and emojis
  - Tool usage stats (Read/Grep/Edit/Bash counters)
  - Trophallaxis metrics (token consumption)
  - Progress bars with excavation animation
- **ASCII art anthill** (`/ant:maturity`)
  - 6 milestone stages with unique art
  - Firefly effects
- **Tunnel view** (`watch-spawn-tree.sh`)
  - Collapsible nested spawn tree
  - View state persistence
  - Depth-based auto-collapse
- **Chamber comparison** (`chamber-compare.sh`)
- **Caste colors** (`bin/lib/caste-colors.js`)

### Files Created Today (v3.1)
```
bin/lib/model-profiles.js
bin/lib/caste-colors.js
bin/lib/telemetry.js (enhanced)
.aether/utils/chamber-*.sh (4 files)
.aether/utils/swarm-display.sh
.aether/utils/watch-spawn-tree.sh (enhanced)
.aether/utils/spawn-tree.sh
.aether/utils/spawn-with-model.sh
.aether/utils/state-loader.sh
.aether/utils/error-handler.sh
.aether/visualizations/ (ASCII art files)
.claude/commands/ant/verify-castes.md
.claude/commands/ant/maturity.md
.claude/commands/ant/tunnels.md (enhanced)
.claude/commands/ant/swarm.md (enhanced)
```

---

## 🔍 CURRENT STATE (Pre-Recovery, 2026-02-15)

> **Historical record.** This described the state at the time of the recovery incident. The recovery steps below no longer apply.

### What's in Git (Committed)
- All planning documents (`.planning/phases/09-12/`)
- v3.1 milestone audit
- Emoji section in `runtime/workers.md` (commit `5b1410a`)
- Working runtime/aether-utils.sh

### What's Modified (Uncommitted)
- `.aether/workers.md` - Has `general-purpose` fix + emoji section
- `.aether/aether-utils.sh` - Additional swarm commands
- Multiple command files with `subagent_type` corrections
- `runtime/` files need sync from `.aether/`

### What's Missing from runtime/
1. **workers.md** - Missing 12-line emoji mapping section
2. **utils/** - Missing 4 files:
   - `error-handler.sh`
   - `spawn-tree.sh`
   - `spawn-with-model.sh`
   - `state-loader.sh`
3. **utils drift** - `chamber-utils.sh` and `watch-spawn-tree.sh` outdated

---

## 🛠️ RECOVERY STEPS

> **These recovery steps applied to the pre-v4.0 pipeline and are no longer needed.** The `runtime/` directory has been removed. For reference only.

### Step 1: Commit Current Work
```bash
# Stage the recovered planning files
git add .planning/
git add .aether/docs/biological-reference.md .aether/docs/command-sync.md \
        .aether/docs/namespace.md .aether/docs/pathogen-schema-example.json \
        .aether/docs/pathogen-schema.md .aether/docs/pheromones.md \
        .aether/docs/progressive-disclosure.md
git add .opencode/commands/ant/ant:build.md .opencode/commands/ant/ant:verify-castes.md

# Commit the recovery
git commit -m "recovery: restore deleted planning and docs files"

# Now commit the actual improvements
git add .aether/workers.md .aether/aether-utils.sh
git add .claude/commands/ant/build.md .claude/commands/ant/organize.md \
        .claude/commands/ant/plan.md
# ... add other modified command files
git commit -m "fix: subagent_type correction and emoji mapping"
```

### Step 2: Sync runtime/ from .aether/ (CRITICAL)
```bash
# Copy working .aether/ back to runtime/ (reverse the broken flow)

# Core files
cp .aether/workers.md runtime/workers.md
cp .aether/aether-utils.sh runtime/aether-utils.sh
cp .aether/verification-loop.md runtime/verification-loop.md

# Utils - copy new files
cp .aether/utils/error-handler.sh runtime/utils/ 2>/dev/null || echo "error-handler.sh not in .aether"
cp .aether/utils/spawn-tree.sh runtime/utils/ 2>/dev/null || echo "spawn-tree.sh not in .aether"
cp .aether/utils/spawn-with-model.sh runtime/utils/ 2>/dev/null || echo "spawn-with-model.sh not in .aether"
cp .aether/utils/state-loader.sh runtime/utils/ 2>/dev/null || echo "state-loader.sh not in .aether"

# Utils - sync existing files
cp .aether/utils/chamber-utils.sh runtime/utils/
cp .aether/utils/watch-spawn-tree.sh runtime/utils/
cp .aether/utils/chamber-compare.sh runtime/utils/ 2>/dev/null || true
cp .aether/utils/swarm-display.sh runtime/utils/ 2>/dev/null || true

# Docs - sync all reference docs
mkdir -p runtime/docs
cp .aether/docs/constraints.md runtime/docs/ 2>/dev/null || true
cp .aether/docs/pheromones.md runtime/docs/ 2>/dev/null || true
cp .aether/docs/pathogen-schema.md runtime/docs/ 2>/dev/null || true
cp .aether/docs/pathogen-schema-example.json runtime/docs/ 2>/dev/null || true
cp .aether/docs/progressive-disclosure.md runtime/docs/ 2>/dev/null || true
```

### Step 3: Verify the Sync
```bash
# Check that emoji section is now in runtime/
grep -A 10 "Caste Emoji Mapping:" runtime/workers.md

# Check that get_caste_emoji function exists in runtime/
grep -A 5 "get_caste_emoji()" runtime/aether-utils.sh

# Check utils count
echo "runtime/utils count: $(ls runtime/utils/*.sh 2>/dev/null | wc -l)"
echo ".aether/utils count: $(ls .aether/utils/*.sh 2>/dev/null | wc -l)"
```

### Step 4: Commit the Sync
```bash
git add runtime/
git commit -m "sync: runtime/ updated from working .aether/ changes

- Added emoji mapping section to workers.md
- Added missing utils: error-handler.sh, spawn-tree.sh,
  spawn-with-model.sh, state-loader.sh
- Synced chamber-utils.sh and watch-spawn-tree.sh
- Added missing docs to runtime/docs/"
```

### Step 5: Test the Fix
```bash
# Reinstall from local source
npm install -g .

# Verify hub was updated
ls ~/.aether/system/workers.md
grep "Caste Emoji Mapping:" ~/.aether/system/workers.md

# Test in a fresh repo (or this one)
cd /tmp && mkdir test-colony && cd test-colony
git init
/ant:init "Test recovery"
# Check that emojis appear in spawn output
```

---

## 📋 VERIFICATION CHECKLIST

> **As of v4.0, these checks are replaced by `npm pack --dry-run` and `bash bin/validate-package.sh`.**

After recovery (pre-v4.0), verify:

- [ ] `runtime/workers.md` has emoji mapping section (12 lines)
- [ ] `runtime/aether-utils.sh` has `get_caste_emoji()` function
- [ ] `runtime/utils/` has 11+ files (including 4 new ones)
- [ ] `~/.aether/system/` reflects runtime/ after `npm install`
- [ ] `./.aether/` retains changes after `aether update`
- [ ] `/ant:build` shows emojis in spawn output
- [ ] `/ant:swarm` displays real-time visualization
- [ ] `/ant:maturity` shows ASCII art anthill

---

## 🚨 ANTI-PATTERNS TO AVOID

> **As of v4.0, `.aether/` is published directly. There is no `runtime/` to sync, so the anti-patterns below no longer apply.**

**Pre-v4.0 anti-patterns (historical):**

**DO NOT:**
1. Edit `.aether/` directly without syncing to `runtime/`
2. Run `npm install` without committing changes to `runtime/`
3. Run `aether update` without syncing `runtime/` first
4. Treat `.aether/` as source of truth (it's the working copy)

**DO:**
1. Edit `.aether/` as the source of truth
2. Commit `.aether/` changes before `npm install`
3. The sync script auto-populates `runtime/` from `.aether/`
4. Test changes with `npm install -g .` from the repo

---

## 📊 CDS ARCHIVE REFERENCE

All work is documented in the CDS (Cosmic Dev System) format:

| Phase | Location | Plans | Status |
|-------|----------|-------|--------|
| 9 | `.planning/phases/09-caste-model-assignment/` | 5 | Complete |
| 10 | `.planning/phases/10-entombment-egg-laying/` | 4 | Complete |
| 11 | `.planning/phases/11-foraging-specialization/` | 4 | Complete |
| 12 | `.planning/phases/12-colony-visualization/` | 5 | Complete |

**Key Documents:**
- `.planning/v3.1-MILESTONE-AUDIT.md` - Full audit report
- `.planning/ROADMAP.md` - v3.1 roadmap overview
- `.planning/PROJECT.md` - Requirements and context
- `.planning/STATE.md` - Current state tracking

---

## 🎯 NEXT STEPS AFTER RECOVERY

> **As of v4.0, step 5 is complete.** The `runtime/` staging directory has been removed and `.aether/` is now the direct package source.

1. **Publish v3.1** - npm version patch && npm publish
2. **Update CHANGELOG** - Add v3.1.6 entry with emoji fix
3. **Update README** - Fix version number (currently says v1.1.0)
4. **Archive model routing** - Already done (tag: model-routing-v1-archived)
5. ~~**Consider future architecture** - Remove runtime/ and use .aether/ directly?~~ **DONE in v4.0**

---

## 🔗 COMMANDS REFERENCE

**To execute this recovery after /clear:**

```bash
# 1. Read this document
cat .aether/RECOVERY-PLAN.md

# 2. Follow the recovery steps above

# 3. After recovery, verify with:
ant:verify-castes  # Should show caste emojis
ant:status         # Should show colony status
ant:swarm          # Should show real-time display
```

---

**Document Status:** Resolved (v4.0 — runtime/ eliminated)
**Last Updated:** 2026-02-19
**Author:** Claude (post-diagnosis → v4.0 resolution)
