<!-- Generated from .aether/commands/lay-eggs.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-lay-eggs
description: "🥚 Set up Aether in this repo — creates repo-local Aether state"
---

You are the **Queen**. Prepare this repository for Aether colony development.

## Instructions

This command sets up the repo-local `.aether/` state area. It does **not**
start a colony — that is what `/ant-init "goal"` is for.

For beginners: this is the step that makes the current project Aether-aware.
The reusable Aether machinery stays installed globally; this repo only gets its
own nest state, notes, locks, and project wisdom.

<failure_modes>
### Runtime Or Hub Unavailable
If the `aether lay-eggs` command reports that the binary or hub is unavailable:
- Show the CLI error output
- Tell the user to install the Aether Go binary and run `aether install` first
- Stop after the runtime failure

### Partial Setup Failure
If the CLI reports setup errors:
- Report which setup step failed
- Do not hand-copy global assets into the repo to compensate
- Tell the user the command is safe to re-run after the hub/runtime issue is fixed

### Existing Repo State
If `.aether/` already exists:
- Treat the command as an idempotent repair/setup pass
- Preserve existing local state and project wisdom
- Do not overwrite colony state, dreams, oracle notes, or custom skills
</failure_modes>

<success_criteria>
Command is complete when:
- `.aether/` exists
- repo-local state scaffolding exists as needed (`data/`, `locks/`, and related runtime directories)
- `.aether/QUEEN.md` exists or is preserved
- `.aether/.gitignore` protects local state from version control
- reusable assets are **not** copied into repo-local `.aether/`
- user sees confirmation and next steps
</success_criteria>

<read_only>
Do not touch during lay-eggs:
- .aether/data/COLONY_STATE.json (colony state belongs to init)
- .aether/data/pheromones.json and other colony data files
- .aether/dreams/ contents (user notes)
- .aether/oracle/ contents (research artifacts)
- .aether/locks/ active lock files
- .aether/skills/ custom repo-specific skills
- Source code files
- .env* files
- User-owned platform settings
</read_only>

<global_assets>
These assets must stay global after setup:
- agents
- commands
- shipped skills
- templates
- docs
- utils
- workers.md
- exchange modules
- references

Do not manually copy them into `.aether/`, `.claude/`, `.opencode/`, or
`.codex/` inside this repo. `aether install` owns global installation, and
`aether lay-eggs` owns only repo-local setup.
</global_assets>

### Step 1: Check Existing Setup

Check whether `.aether/` already exists.

**If it exists:**
```
Aether is already present in this repo.

Refreshing repo-local setup and preserving colony state...
```
Proceed to Step 2. This makes the command safe to re-run as a repair pass.

**If it does NOT exist:**
```
Setting up Aether local state in this repo...
```
Proceed to Step 2.

### Step 2: Run Runtime Setup

Run using the Bash tool with description "Preparing Aether local state...":

```bash
AETHER_OUTPUT_MODE=visual aether lay-eggs $ARGUMENTS
```

The Go runtime is the source of truth. Do not recreate the setup logic in the
wrapper unless the runtime command is unavailable.

### Step 3: Verify Local-Only Layout

Run using the Bash tool with description "Verifying local-only Aether layout...":

```bash
dirs=0
files=0
global_leaks=0

for d in .aether .aether/data .aether/locks; do
  [ -d "$d" ] && dirs=$((dirs + 1))
done

[ -f .aether/QUEEN.md ] && files=$((files + 1))
[ -f .aether/.gitignore ] && files=$((files + 1))

if [ -d .aether ]; then
  for p in .aether/*; do
    case "$p" in
      .aether/data|.aether/locks|.aether/dreams|.aether/oracle|.aether/QUEEN.md|.aether/skills)
        ;;
      *)
        [ -e "$p" ] && global_leaks=$((global_leaks + 1))
        ;;
    esac
  done
fi

echo "{\"dirs\": $dirs, \"core_files\": $files, \"global_asset_leaks\": $global_leaks}"
```

Parse the JSON output for the display step.

If `global_asset_leaks` is greater than zero, explain that old repo-local global
assets are still present and should be cleaned by the runtime during the
global-hub migration. Do not delete user-owned custom skills or local state by
hand from this wrapper.

### Step 4: Display Result

```
🥚 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   A E T H E R   R E A D Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 🥚

   {dirs} local state directories ready
   {core_files} local setup files ready
   {global_asset_leaks} old global-asset paths detected

Shared Aether assets now live globally:
  ~/.aether/system/

This repo keeps only project-local state:
  .aether/data/
  .aether/dreams/
  .aether/oracle/
  .aether/locks/
  .aether/QUEEN.md
  .aether/skills/   (custom repo skills only)

To start a colony:
  /ant-init "your goal here"

To verify setup:
  /ant-status
```
