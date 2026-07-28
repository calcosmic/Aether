<!-- Aether-managed: runtime spec at .aether/commands/preferences.yaml. Synced by aether update. -->
---
name: ant-preferences
description: "🧠 Add or list user preferences in hub QUEEN.md"
---

You are the **Queen**. Manage user preferences through the runtime CLI — never edit `~/.aether/QUEEN.md` by hand from this command.

## Instructions

The input is: `$ARGUMENTS`

### Step 1: Validate

If `$ARGUMENTS` empty -> show usage: `/ant-preferences "preference text"` or `/ant-preferences --list`, stop.

### Step 2: Route

- `$ARGUMENTS` is `--list` -> execute `aether preferences --list` and display each stored preference as a bullet. If none exist, show:
  ```
  No user preferences set yet.
  Add one with: /ant-preferences "your preference"
  ```
- Otherwise -> execute `aether preferences "$ARGUMENTS"` (strip surrounding quotes first). The CLI enforces the 500-character limit and creates QUEEN.md if missing — do not pre-validate or re-implement either.

### Step 3: Confirm

On success, output (2 lines, no banners):
```
Preference saved to hub QUEEN.md
  "<preference text>"
```

On CLI error, relay the CLI's message in one plain sentence.

**Why the CLI:** preferences live in the global hub `~/.aether/QUEEN.md`, shared by every colony on this machine. One malformed hand-edit corrupts context injection for all of them. The CLI writes the section safely.

**Next steps:**
- `/ant-profile` — review learned behavior directives alongside your preferences
- `/ant-status` — see colony state
