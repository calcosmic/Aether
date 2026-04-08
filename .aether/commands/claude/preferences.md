<!-- Generated from .aether/commands/preferences.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:preferences
description: "Add or list user preferences in hub QUEEN.md"
---

You are the **Queen**. Manage user preferences in the hub `~/.aether/QUEEN.md`.

## Instructions

The input is: `$ARGUMENTS`

### Step 1: Validate

If `$ARGUMENTS` empty -> show usage: `/ant:preferences "preference text"` or `/ant:preferences --list`, stop.

### Step 2: Route

- `$ARGUMENTS` is `--list` -> **List mode**
- Otherwise -> **Add mode**

### List Mode

Read `~/.aether/QUEEN.md`. If missing -> "No hub QUEEN.md found. Run /ant:init to create one.", stop.

Find the `## User Preferences` section (with or without emoji). Extract bullet lines (`- ...`) between that heading and the next `---`.

If user-added bullets exist, output each. Otherwise:
```
No user preferences set yet.
Add one with: /ant:preferences "your preference"
```
Stop.

### Add Mode

Set `PREF` to `$ARGUMENTS` (strip surrounding quotes).
If length > 500 chars -> "Preference too long (max 500 chars)", stop.

Read `~/.aether/QUEEN.md`. If missing -> "No hub QUEEN.md found. Run /ant:init to create one.", stop.

**If `## User Preferences` section exists** (with or without emoji):
1. Remove `*No user preferences recorded yet.*` placeholder if present
2. Append `- $PREF` at end of section (before next `---`)

**If section missing** (migration case):
1. Insert before `## Evolution Log` (with or without emoji):
```
## User Preferences

- $PREF

---

```

Update METADATA comment: increment `stats.total_user_prefs` by 1.

### Confirm

Output (2 lines, no banners):
```
Preference saved to hub QUEEN.md
  "$PREF"
```
