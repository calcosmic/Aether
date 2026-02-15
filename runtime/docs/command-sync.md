# Command Sync Strategy

This document describes how Aether slash commands are distributed to Claude Code and OpenCode, and the bulletproof sync mechanisms that ensure consistency.

---

## Distribution Model

### Claude Code: Global Sync

Aether commands are synced to the global Claude Code config directory:

```
~/.claude/commands/ant/
```

This is done via `aether install` which copies commands from the npm package to the global location. The sync uses **hash-based comparison** to only copy files that have changed.

### OpenCode: Repo-Local Only

OpenCode **does not** have a global discovery mechanism for slash commands. Commands must exist in the repo-local directory:

```
.opencode/commands/ant/
```

This is why Aether maintains parallel command directories:
- `.claude/commands/ant/` — Claude Code commands
- `.opencode/commands/ant/` — OpenCode commands (repo-local)

---

## Why the Strategies Differ

| Feature | Claude Code | OpenCode |
|---------|-------------|----------|
| Global command discovery | Yes (`~/.claude/commands/`) | No |
| Repo-local commands | Supported | Required |
| Namespace isolation | `/ant:` prefix | `/ant:` prefix |

**Key insight:** OpenCode's architecture requires repo-local commands. There is no equivalent to `~/.claude/commands/` that provides automatic slash command discovery. This is a fundamental platform difference.

---

## Hash-Based Idempotent Sync

The sync system in `bin/cli.js` uses **hash comparison before copying**:

```javascript
// Hash comparison: only copy if file doesn't exist or hash differs
let shouldCopy = true;
if (fs.existsSync(destPath)) {
  const srcHash = hashFileSync(srcPath);
  const destHash = hashFileSync(destPath);
  if (srcHash === destHash) {
    shouldCopy = false;
    skipped++;
  }
}

if (shouldCopy) {
  fs.copyFileSync(srcPath, destPath);
}
```

**Why this matters:**
- **Idempotent:** Running `aether install` multiple times produces the same result
- **Efficient:** Unchanged files are skipped (visible in logs as "skipped")
- **Bulletproof:** No unnecessary writes reduce the risk of corruption

---

## Environment Variable Validation

Before constructing any paths that use the user's home directory, the code validates the HOME environment variable:

```javascript
const HOME = process.env.HOME || process.env.USERPROFILE;
if (!HOME) {
  console.error('Error: HOME environment variable is not set');
  console.error('Please ensure HOME or USERPROFILE is defined');
  process.exit(1);
}

// Now safe to use
const COMMANDS_DEST = path.join(HOME, '.claude', 'commands', 'ant');
```

**Why this matters:**
- Prevents crashes on systems where HOME is not set
- Provides clear error message instead of cryptic path errors
- Supports both Unix-like systems (HOME) and Windows (USERPROFILE)

---

## Verification Commands

### Check Sync Status

To verify commands are in sync between Claude Code and OpenCode within the repo:

```bash
./bin/generate-commands.sh check
```

This performs two passes:
1. **Pass 1:** File count and name comparison
2. **Pass 2:** SHA-1 hash comparison for content-level drift detection

### Verify Global Installation

To verify global Claude Code commands are installed:

```bash
ls ~/.claude/commands/ant/
```

### Verify Global OpenCode Commands

To verify global OpenCode commands (if previously synced):

```bash
ls ~/.config/opencode/commands/ant/
```

---

## Sync Workflow

```
┌─────────────────┐
│  Write commands │
│  in .claude/    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  aether install                     │
│  (syncs to ~/.claude/commands/ant/)│
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  Hash-based copy                    │
│  (only copies if hash differs)      │
└─────────────────────────────────────┘
```

---

## Anti-Patterns to Avoid

1. **Unconditional copy:** Never copy files without checking if they changed
2. **Skip hash comparison:** Always use hash comparison for idempotency
3. **Ignore HOME validation:** Always validate HOME before path construction
4. **Assume OpenCode global:** OpenCode does not support global command discovery

---

## See Also

- `bin/cli.js` — Implementation of hash-based sync
- `bin/generate-commands.sh` — Repo-level sync verification
- `.aether/docs/namespace.md` — Namespace isolation strategy
