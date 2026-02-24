---
name: ant:update
description: "🔄🐜📦🐜🔄 Update Aether safely from the global hub (transactional)"
---

You are the **Queen Ant Colony**. Update this repo's Aether system files from the global distribution hub.

## Safety Rules

1. Use the CLI transactional updater (`aether update`) instead of manual `cp` chains.
2. Never overwrite colony runtime data (`.aether/data/`) or user wisdom (`.aether/QUEEN.md`).
3. Do **not** assume version numbers are monotonic. Labels may reset; avoid "downgrade" wording.
4. If update reports dirty managed files, stop and show recovery options unless user requested force.

## Instructions

### Step 1: Check Hub Availability

Run using the Bash tool with description "Checking Aether hub...":

```bash
test -f ~/.aether/version.json && cat ~/.aether/version.json || echo "__NO_HUB__"
```

If output is `__NO_HUB__`, display:

```
No Aether distribution hub found at ~/.aether/

To set up the hub, run:
  npx aether-colony install
  - or -
  aether install
```

Stop here.

Parse `version` from the JSON as `available_version`.

### Step 1.5: Verify CLI Availability

Run using the Bash tool with description "Checking aether CLI...":

```bash
command -v aether >/dev/null 2>&1 && echo "__CLI_OK__" || echo "__CLI_MISSING__"
```

If output is `__CLI_MISSING__`, display:

```
The transactional updater is not available because the `aether` CLI is missing.

Install/update it, then retry:
  npx aether-colony install
  - or -
  npm i -g aether-colony
```

Stop here.

### Step 2: Parse Force Flag

Treat either of these as force:
- `--force`
- `--force-update`

Set:
- `update_flags="--force"` when force requested
- `update_flags=""` otherwise

### Step 3: Dry-Run Preview

Run using the Bash tool with description "Previewing update...":

```bash
aether update --dry-run $update_flags
```

If this fails, show the error output and stop.

### Step 4: Execute Transactional Update

Run using the Bash tool with description "Applying update...":

```bash
aether update $update_flags
```

This command handles:
- checkpoint creation
- safe sync
- integrity verification
- automatic rollback on failure

### Step 5: Clear Version Cache

Run using the Bash tool:

```bash
rm -f .aether/data/.version-check-cache
```

### Step 6: Display Summary

Display a concise summary:

```
🔄🐜📦🐜🔄 AETHER UPDATE COMPLETE

Hub version label: {available_version}
Update mode: {normal|force}
Colony data (.aether/data/) untouched.

Note: version labels are treated as identifiers, not strict upgrade/downgrade ordering.
```

### Next Up

Generate the state-based Next Up block by running:

```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```
