# Queen Commands Reference

The queen-* commands manage Aether's QUEEN.md wisdom files.

For dummies: Aether has a global brain and a project brain. The global file in
the hub stores user preferences and cross-colony wisdom. The repo-local file
stores project-specific charter and lessons for the current repo. Colony-prime
reads both before workers run.

For the QUEEN.md file format and wisdom feedback loop, see [QUEEN-SYSTEM.md](./QUEEN-SYSTEM.md).

## Commands

### queen-init

**Purpose:** Initialize the global and repo-local QUEEN.md files from the
standard template when they do not already exist.

**Usage:**
```bash
aether queen-init
```

**Returns:** JSON with creation status, file path, and template source.

**Example output:**
```json
{"ok":true,"result":{"created":true,"path":"~/.aether/QUEEN.md","local_created":true,"local_path":".aether/QUEEN.md"}}
```

**Behavior:**
- If the global QUEEN.md already exists, returns `{"created":false}` without
  overwriting it.
- Also creates repo-local `.aether/QUEEN.md` if the current repo has Aether
  state and no local Queen file yet.
- Creates the needed local `.aether/` directories through the runtime setup
  path, not by wrapper hand-edits.

---

### queen-read

**Purpose:** Read the hub-global QUEEN.md content.

**Usage:**
```bash
aether queen-read
```

**Returns:** JSON with the global file path, size, and content.

**Example output:**
```json
{"ok":true,"result":{"path":"~/.aether/QUEEN.md","size":1234,"content":"# QUEEN.md ..."}}
```

**Behavior:**
- Reads the hub file only.
- Worker priming uses colony-prime, which loads global wisdom first and then
  repo-local wisdom from `.aether/QUEEN.md`.
- User preferences are collected from both global and local Queen files.

---

### queen-promote

**Purpose:** Promote a validated learning or preference to hub-global QUEEN.md.

**Usage:**
```bash
aether queen-promote <type> <content> <colony_name>
```

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| type | Yes | Wisdom category: `philosophy`, `pattern`, `redirect`, `stack`, `decree` |
| content | Yes | The wisdom text to add |
| colony_name | Yes | Name of the colony contributing the wisdom |

**Returns:** JSON confirming the promotion with details.

**Behavior:**
- Appends the wisdom entry to the appropriate section in the hub-global file.
- Includes attribution (colony name) and timestamp
- Use `queen-write-learnings` for phase learnings that should remain
  repo-local.

---

### queen-write-learnings

**Purpose:** Write phase learning entries to the repo-local QUEEN.md.

**Usage:**
```bash
aether queen-write-learnings --learnings '[{"claim":"..."}]'
```

**Behavior:**
- Loads `.aether/QUEEN.md` from the current repo.
- Appends learning claims to the local `Wisdom` section.
- Does not overwrite hub-global user preferences.

---

## For Contributors

The queen commands are part of the colony lifecycle:

1. **Colony startup:** `/ant-init` calls `queen-init` to ensure global and
   local Queen files exist.
2. **Worker priming:** `/ant-build` loads `colony-prime --compact`, which
   includes global Queen wisdom, local Queen wisdom, preferences, context
   capsule, and top signals.
3. **Learning writes:** phase learnings go to repo-local QUEEN.md; explicit
   preferences and cross-colony promotions go to the hub-global QUEEN.md.

### Adding a New Queen Command

1. Add the function implementation in `cmd/queen.go` (domain module)
2. Add the dispatch case in `aether CLI` (alongside existing `queen-*` blocks)
3. Add it to the flat `commands` array in the `help)` case block
4. Add it to the "Queen Commands" section in help's `sections` JSON
5. Update this file with usage documentation
6. Add tests in `tests/bash/test-aether CLI`
