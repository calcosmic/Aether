# Queen Commands Reference

The queen-* commands manage the QUEEN.md wisdom file — a persistent cross-colony knowledge base that accumulates philosophies, patterns, redirects, stack wisdom, and decrees across sessions.

For the QUEEN.md file format and wisdom feedback loop, see [QUEEN-SYSTEM.md](./QUEEN-SYSTEM.md).

## Commands

### queen-init

**Purpose:** Initialize a new QUEEN.md from the system template.

**Usage:**
```bash
bash .aether/aether-utils.sh queen-init
```

**Returns:** JSON with creation status, file path, and template source.

**Example output:**
```json
{"ok":true,"result":{"created":true,"path":".aether/QUEEN.md","source":"~/.aether/system/templates/QUEEN.md.template"}}
```

**Behavior:**
- If QUEEN.md already exists, returns `{"created":false}` without overwriting
- Searches for template in: hub system path, then local .aether/templates/
- Creates `.aether/` directory if it doesn't exist

---

### queen-read

**Purpose:** Read QUEEN.md wisdom as structured JSON for worker priming.

**Usage:**
```bash
bash .aether/aether-utils.sh queen-read
```

**Returns:** JSON with metadata, wisdom sections, and priming flags.

**Example output:**
```json
{"ok":true,"result":{"metadata":{...},"wisdom":{"philosophies":"...","patterns":"...","redirects":"...","stack_wisdom":"...","decrees":"..."},"priming":{"has_philosophies":true,...}}}
```

**Behavior:**
- Extracts METADATA JSON block from `<!-- METADATA ... -->` comment
- Parses each wisdom section (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees)
- Returns priming flags indicating which sections have content
- Returns E_JSON_INVALID if METADATA block contains malformed JSON
- Returns E_FILE_NOT_FOUND if QUEEN.md doesn't exist

---

### queen-promote

**Purpose:** Promote a validated learning to QUEEN.md wisdom.

**Usage:**
```bash
bash .aether/aether-utils.sh queen-promote <type> <content> <colony_name>
```

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| type | Yes | Wisdom category: `philosophy`, `pattern`, `redirect`, `stack`, `decree` |
| content | Yes | The wisdom text to add |
| colony_name | Yes | Name of the colony contributing the wisdom |

**Returns:** JSON confirming the promotion with details.

**Behavior:**
- Appends the wisdom entry to the appropriate section in QUEEN.md
- Includes attribution (colony name) and timestamp
- Updates the METADATA block's stats

---

## For Contributors

The queen commands are part of the colony lifecycle:

1. **Colony startup:** `/ant:init` calls `queen-init` to ensure QUEEN.md exists
2. **Worker priming:** `/ant:build` loads `colony-prime --compact`, which includes QUEEN wisdom + compact context capsule + top signals
3. **Colony end:** `/ant:seal` can call `queen-promote` to persist learnings

### Adding a New Queen Command

1. Add the case block in `aether-utils.sh` (alongside existing `queen-*` blocks)
2. Add it to the flat `commands` array in the `help)` case block
3. Add it to the "Queen Commands" section in help's `sections` JSON
4. Update this file with usage documentation
5. Add tests in `tests/bash/test-aether-utils.sh`
