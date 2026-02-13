---
name: ant:update
description: "🔄🐜📦🐜🔄 Update system files from the global Aether hub"
---

You are the **Queen Ant Colony**. Update this repo's Aether system files from the global distribution hub.

## Instructions

### Step 1: Check Hub Availability

Use the Read tool to read `~/.aether/version.json` (expand `~` to the user's home directory).

If the file does not exist, output:

```
No Aether distribution hub found at ~/.aether/

To set up the hub, run:
  npx aether-colony install
  — or —
  aether install

The hub provides system file updates across all your Aether repos.
```

Stop here. Do not proceed.

Read the `version` field — this is the **available version**.

### Step 2: Check Current Version

Use the Read tool to read `.aether/version.json`.

If the file does not exist, set current version to "unknown".
Otherwise, read the `version` field — this is the **current version**.

If current version equals available version, output:

```
Already up-to-date (v{version}).

System files and commands match the global hub.
Colony data (.aether/data/) is always untouched by updates.
```

Stop here. Do not proceed.

### Step 3: Bootstrap System Files

Run using the Bash tool:
```
bash .aether/aether-utils.sh bootstrap-system
```

This copies system files (docs, utils, aether-utils.sh) from `~/.aether/system/` into `.aether/` using an explicit allowlist. Colony data is never touched.

Parse the JSON output to get the count of copied files.

### Step 4: Update Commands

Copy command files from the hub to this repo. Run using the Bash tool:

```
cp -R ~/.aether/commands/claude/* .claude/commands/ant/ 2>/dev/null; echo "claude: done"
cp -R ~/.aether/commands/opencode/* .opencode/commands/ant/ 2>/dev/null; echo "opencode: done"
cp -R ~/.aether/agents/* .opencode/agents/ 2>/dev/null; echo "agents: done"
```

### Step 5: Register and Version Stamp

Run using the Bash tool:
```
bash .aether/aether-utils.sh registry-add "$(pwd)" "{available_version}"
```

Substitute `{available_version}` with the version from Step 1.

Then use the Write tool to write `.aether/version.json`:
```json
{
  "version": "{available_version}",
  "updated_at": "{ISO-8601 timestamp}"
}
```

### Step 6: Display Summary

Output:

```
🔄🐜📦🐜🔄 ═══════════════════════════════════════════════════
   A E T H E R   U P D A T E
═══════════════════════════════════════════════════ 🔄🐜📦🐜🔄

Updated: v{current_version} -> v{available_version}

  System files: {N} updated
  Commands: synced from hub
  Agents: synced from hub

Colony data (.aether/data/) untouched.
Repo registered in ~/.aether/registry.json.
```
