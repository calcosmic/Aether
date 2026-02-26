---
name: ant:lay-eggs
description: "🥚🐜🥚 Set up Aether in this repo — creates .aether/ with all system files"
---

You are the **Queen**. Prepare this repository for Aether colony development.

## Instructions

This command sets up the `.aether/` directory structure and copies all system files from the global hub. It does NOT start a colony — that's what `/ant:init "goal"` is for.

<failure_modes>
### Hub Not Found
If `~/.aether/system/aether-utils.sh` does not exist:
- The global hub is not installed
- Tell the user to run `npm install -g aether-colony` first
- Stop — cannot proceed without hub

### Partial Copy Failure
If some files fail to copy from hub:
- Report which files succeeded and which failed
- The user can re-run `/ant:lay-eggs` safely (idempotent)
</failure_modes>

<success_criteria>
Command is complete when:
- `.aether/` directory exists with all subdirectories
- System files (aether-utils.sh, workers.md, etc.) are present
- Templates, docs, utils, schemas are populated
- QUEEN.md is initialized
- User sees confirmation and next steps
</success_criteria>

<read_only>
Do not touch during lay-eggs:
- .aether/data/COLONY_STATE.json (colony state belongs to init)
- .aether/dreams/ contents (user notes — create dir but don't modify files)
- .aether/chambers/ contents (archived colonies — create dir but don't modify files)
- Source code files
- .env* files
- .claude/settings.json
</read_only>

### Step 1: Check Hub Availability

Check if the global hub exists by reading `~/.aether/system/aether-utils.sh` (expand `~` to the user's home directory).

**If the hub does NOT exist:**
```
Aether hub not found at ~/.aether/system/

The global hub must be installed before setting up a repo.

  npm install -g aether-colony

This installs the Aether CLI and populates the hub at ~/.aether/system/
with all the system files your repo needs.

After installing, run /ant:lay-eggs again.
```
Stop here.

### Step 2: Check Existing Setup

Check if `.aether/aether-utils.sh` already exists using the Read tool.

**If it exists:**
```
Aether is already set up in this repo.

Refreshing system files from hub...
```
Proceed to Step 3 (this makes the command safe to re-run as an update/repair).

**If it does NOT exist:**
```
Setting up Aether in this repo...
```
Proceed to Step 3.

### Step 3: Create Directory Structure

Run using the Bash tool with description "Creating Aether directory structure...":
```bash
mkdir -p \
  .aether/data \
  .aether/data/midden \
  .aether/data/backups \
  .aether/data/survey \
  .aether/dreams \
  .aether/chambers \
  .aether/locks \
  .aether/temp \
  .aether/docs \
  .aether/utils \
  .aether/templates \
  .aether/schemas \
  .aether/exchange \
  .aether/rules \
  .aether/scripts \
  .claude/rules && \
touch .aether/dreams/.gitkeep && \
touch .aether/chambers/.gitkeep && \
touch .aether/data/midden/.gitkeep
```

### Step 4: Copy System Files from Hub

Run using the Bash tool with description "Copying system files from hub...":
```bash
# Core system files
cp -f ~/.aether/system/aether-utils.sh .aether/ && \
chmod +x .aether/aether-utils.sh && \
cp -f ~/.aether/system/workers.md .aether/ 2>/dev/null || true && \
cp -f ~/.aether/system/CONTEXT.md .aether/ 2>/dev/null || true && \
cp -f ~/.aether/system/model-profiles.yaml .aether/ 2>/dev/null || true && \

# Directories
cp -Rf ~/.aether/system/docs/* .aether/docs/ 2>/dev/null || true && \
cp -Rf ~/.aether/system/utils/* .aether/utils/ 2>/dev/null || true && \
cp -Rf ~/.aether/system/templates/* .aether/templates/ 2>/dev/null || true && \
cp -Rf ~/.aether/system/schemas/* .aether/schemas/ 2>/dev/null || true && \
cp -Rf ~/.aether/system/exchange/* .aether/exchange/ 2>/dev/null || true && \
cp -Rf ~/.aether/system/rules/* .claude/rules/ 2>/dev/null || true && \

# Version tracking
cp -f ~/.aether/version.json .aether/version.json 2>/dev/null || true

echo "System files copied."
```

### Step 5: Initialize QUEEN.md

Run using the Bash tool with description "Initializing QUEEN.md...":
```bash
bash .aether/aether-utils.sh queen-init
```

Parse the JSON result:
- If `created` is true: note `QUEEN.md initialized`
- If `created` is false: note `QUEEN.md already exists (preserved)`

### Step 6: Register Repo (Silent)

Attempt to register this repo in the global hub. Silent on failure — registry is optional.

Run using the Bash tool with description "Registering repo..." (ignore errors):
```bash
bash .aether/aether-utils.sh registry-add "$(pwd)" "$(jq -r '.version // "unknown"' ~/.aether/version.json 2>/dev/null || echo 'unknown')" 2>/dev/null || true
```

### Step 7: Verify Setup

Run using the Bash tool with description "Verifying setup...":
```bash
# Count what was set up
dirs=0
files=0
for d in .aether/data .aether/docs .aether/utils .aether/templates .aether/schemas .aether/exchange .aether/dreams .aether/chambers; do
  [ -d "$d" ] && dirs=$((dirs + 1))
done
[ -f .aether/aether-utils.sh ] && files=$((files + 1))
[ -f .aether/workers.md ] && files=$((files + 1))
[ -f .aether/QUEEN.md ] && files=$((files + 1))
[ -f .aether/CONTEXT.md ] && files=$((files + 1))
[ -d .aether/templates ] && templates=$(ls .aether/templates/*.template.* 2>/dev/null | wc -l | tr -d ' ') || templates=0
[ -d .aether/utils ] && utils=$(ls .aether/utils/*.sh 2>/dev/null | wc -l | tr -d ' ') || utils=0

echo "{\"dirs\": $dirs, \"core_files\": $files, \"templates\": $templates, \"utils\": $utils}"
```

Parse the JSON output for the display step.

### Step 8: Display Result

```
🥚 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   A E T H E R   R E A D Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 🥚

   {dirs} directories created
   {core_files} core system files
   {templates} templates
   {utils} utility scripts
   QUEEN.md: {status}

   .aether/ is set up and ready for colony work.

──────────────────────────────────────────────────
🐜 Next Up
──────────────────────────────────────────────────
   /ant:init "your goal"   🌱 Start a colony
   /ant:colonize            🗺️  Analyze existing code first
   /ant:help                📖 See all commands
```
