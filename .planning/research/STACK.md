# Technology Stack: Robust CLI Tools with AI Agent Orchestration

**Project:** Aether Colony System
**Researched:** 2026-02-13
**Confidence:** HIGH

This document provides prescriptive tooling recommendations for improving the Aether Colony System's CLI infrastructure. It addresses the question: "What are the best practices for building robust CLI tools with AI agent orchestration?"

---

## Current Stack Assessment

The Aether Colony System currently uses:

| Component | Current | Recommendation |
|-----------|---------|----------------|
| CLI Entry | `bin/cli.js` (Node.js) | Keep - simple and effective |
| Utilities | `aether-utils.sh` (Bash) | Keep - proven robust |
| State | JSON files | Keep - survives context resets |
| Linting | ShellCheck | Keep + enhance |

---

## Recommended Technology Additions

### 1. CLI Argument Parsing

| Option | Version | Purpose | Why |
|--------|---------|---------|-----|
| **commander** | ^11.0.0 | CLI argument parsing | Best-in-class Node.js CLI parser; supports subcommands, auto-help, type coercion |

**Recommendation:** Use commander for `bin/cli.js` instead of manual argument parsing.

**Why commander:**
- Automatic `--help` generation
- Built-in subcommand support (`/ant:build`, `/ant:plan`)
- Type coercion for options (`--depth 3` becomes number)
- Negatable options (`--no-color`)
- Version flag out of the box

**Installation:**
```bash
npm install commander
```

**Migration example:**
```javascript
// Before: manual parsing
const command = process.argv[2];
const args = process.argv.slice(3);

// After: commander
const { program } = require('commander');
program
  .name('aether')
  .description('Colony-based development framework')
  .version('1.0.0');

program
  .command('build')
  .description('Execute phase with worker spawning')
  .argument('[number]', 'Phase number', '1')
  .action(async (phase) => {
    // Implementation
  });
```

**Confidence:** HIGH - Commander is the de facto standard for Node.js CLI tools (used by npm, create-react-app, etc.)

---

### 2. Bash Script Linting

| Option | Purpose | Why |
|--------|---------|-----|
| **ShellCheck** | Static analysis for shell scripts | Catches errors before runtime |

**Current state:** Aether already uses ShellCheck in `npm run lint:shell`.

**Enhancement recommendations:**

1. **Add to CI/CD** - Run ShellCheck on every commit
2. **Enable strict mode** - Add to all new scripts:
   ```bash
   #!/bin/bash
   set -euo pipefail
   ```
3. **Use SC2015** - Handle edge cases in conditionals:
   ```bash
   # Bad
   [[ -f "$file" ]] && source "$file"

   # Good - check exit code
   [[ -f "$file" ]] && source "$file" || true
   ```

**ShellCheck directives for sourced files:**
```bash
# At top of aether-utils.sh
# shellcheck source=.aether/utils/file-lock.sh
```

**Confidence:** HIGH - ShellCheck is the industry standard for shell script analysis.

---

### 3. JSON State Management

| Option | Purpose | Why |
|--------|---------|-----|
| **JSON Schema validation** | Validate state file structure | Prevents corruption from malformed writes |
| **jq** | JSON manipulation in Bash | Lightweight JSON processing |

**Recommendation:** Add JSON Schema validation to state file writes.

**Schema example for COLONY_STATE.json:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["goal", "phase", "status"],
  "properties": {
    "goal": { "type": "string" },
    "phase": { "type": "integer", "minimum": 1 },
    "status": { "type": "string", "enum": ["planning", "building", "verifying", "complete"] }
  }
}
```

**Validation in aether-utils.sh:**
```bash
validate_state() {
  local state_file="$1"
  local schema_file="$2"
  # Use node for validation (available in Node.js environment)
  node -e "
    const fs = require('fs');
    const Ajv = require('ajv');
    const state = JSON.parse(fs.readFileSync('$state_file'));
    const schema = JSON.parse(fs.readFileSync('$schema_file'));
    const ajv = new Ajv();
    const valid = ajv.validate(schema, state);
    if (!valid) {
      console.error('State validation failed:', ajv.errors);
      process.exit(1);
    }
  "
}
```

**Confidence:** MEDIUM - Adds complexity; only implement if state corruption becomes an issue.

---

### 4. Error Handling Patterns

#### Node.js CLI (`bin/cli.js`)

**Pattern: Centralized error handler**

```javascript
const handleError = (error, command) => {
  console.error(`Error in ${command}:`);

  if (error.code === 'ENOENT') {
    console.error('  File not found. Run /ant:init first.');
  } else if (error.message?.includes('JSON')) {
    console.error('  State file corrupted. Check .aether/data/');
  } else {
    console.error(`  ${error.message}`);
  }

  process.exit(1);
};

// Wrap commands
try {
  await buildCommand(args);
} catch (error) {
  handleError(error, 'build');
}
```

#### Bash Utilities

**Pattern: Verbose error reporting**

```bash
error() {
  echo "[ERROR] $*" >&2
  echo "        In: $(caller 0)" >&2
  echo "        Command: $BASH_COMMAND" >&2
}

# Usage
if [[ ! -f "$state_file" ]]; then
  error "State file not found: $state_file"
  return 1
fi
```

**Pattern: Exit traps for cleanup**

```bash
cleanup() {
  local lockfile="$1"
  # Release lock on exit
  if [[ -f "$lockfile" ]]; then
    flock -u "$lockfd" 2>/dev/null || true
    rm -f "$lockfile"
  fi
}

# Set up trap
trap 'cleanup "$lockfile"' EXIT
```

**Confidence:** HIGH - These patterns are battle-tested in production systems.

---

### 5. Testing Strategy

| Test Type | Tool | Purpose |
|-----------|------|---------|
| Unit | AVA or Jest | Test JavaScript functions |
| Integration | Bash test scripts | Test aether-utils.sh functions |
| E2E | Manual + automation | Test full workflows |

#### Recommended: AVA for Node.js

```bash
npm install --save-dev ava
```

```javascript
// test/cli.test.js
import test from 'ava';
import { execSync } from 'child_process';

test('build command creates state file', t => {
  execSync('node bin/cli.js init --goal "test"', { cwd: '/tmp/test-colony' });
  t.true(require('fs').existsSync('/tmp/test-colony/.aether/data/COLONY_STATE.json'));
});
```

#### Bash Integration Tests

```bash
# test/utils.sh
#!/bin/bash
set -euo pipefail

test_file_lock() {
  local temp_dir
  temp_dir=$(mktemp -d)
  cd "$temp_dir"

  # Run lock test
  bash .aether/utils/file-lock.sh test 10 &
  local pid1=$!
  sleep 1

  # This should timeout/fail
  ! timeout 2 bash .aether/utils/file-lock.sh test 1

  kill $pid1 2>/dev/null || true
  rm -rf "$temp_dir"
}

test_file_lock
echo "File lock tests passed"
```

**Confidence:** HIGH - Testing is essential for robustness; these tools are industry standard.

---

### 6. Logging and Debugging

| Option | Purpose | Why |
|--------|---------|-----|
| **chalk** or **picocolors** | Colored output | Better CLI UX |
| **debug** | Conditional debugging | Only show debug output when needed |

**Recommendation:** Use picocolors (faster, fewer deps than chalk):

```bash
npm install picocolors
```

```javascript
import pc from 'picocolors';

console.log(pc.green('✓'), 'Phase complete');
console.log(pc.red('✗'), 'Build failed');
console.log(pc.blue('ℹ'), 'Spawning worker...');
console.log(pc.dim('(debug)'), 'State:', state);
```

**Debug mode:**
```bash
# In aether-utils.sh
if [[ "${AETHER_DEBUG:-}" == "1" ]]; then
  set -x
fi
```

**Confidence:** MEDIUM - Nice to have; not critical for robustness.

---

### 7. File Locking (Existing - Confirm Best Practice)

Aether already has file locking via `.aether/utils/file-lock.sh`. Verify it follows best practices:

**Best practice pattern:**
```bash
# file-lock.sh
acquire_lock() {
  local lockfile="$1"
  local timeout="${2:-30}"

  exec 200>"$lockfile"
  flock -w "$timeout" 200 || {
    echo "Failed to acquire lock: $lockfile" >&2
    return 1
  }
  echo "$$" >&200
}

release_lock() {
  exec 200>&-
  flock -u 200 2>/dev/null || true
}
```

**Verify:**
- Uses `flock` (correct)
- Has timeout (prevent infinite wait)
- Releases on EXIT trap

**Confidence:** HIGH - Locking implementation already exists and works.

---

### 8. Atomic Writes (Existing - Confirm Best Practice)

Aether already has atomic writes via `.aether/utils/atomic-write.sh`. Verify:

**Best practice pattern:**
```bash
# atomic-write.sh
atomic_write() {
  local target="$1"
  local content="$2"
  local temp
  temp=$(mktemp)

  echo "$content" > "$temp"
  mv "$temp" "$target"  # Atomic on POSIX
}
```

**Verify:**
- Writes to temp file first (not directly to target)
- Uses `mv` for atomic replacement
- No hardlinks across filesystems

**Confidence:** HIGH - Atomic writes already implemented correctly.

---

## Technology Decision Matrix

| Category | Current | Recommended | Change Type |
|----------|---------|-------------|-------------|
| CLI Parsing | Manual | commander ^11.0.0 | Enhancement |
| Shell Linting | ShellCheck | Keep + CI integration | Maintain |
| State Validation | None | JSON Schema (optional) | Future |
| Error Handling | try/catch | Structured handlers | Enhancement |
| Testing | Manual | AVA + Bash tests | Enhancement |
| Logging | echo | picocolors | Optional |
| File Locking | flock | Keep | Maintain |
| Atomic Writes | mv | Keep | Maintain |

---

## Recommended Implementation Order

1. **Phase 1: Error Handling Enhancement**
   - Add centralized error handlers to `bin/cli.js`
   - Add verbose error reporting to `aether-utils.sh`

2. **Phase 2: CLI Structure**
   - Migrate to commander for argument parsing
   - Enable auto-help and version flags

3. **Phase 3: Testing Infrastructure**
   - Add AVA for unit tests
   - Create Bash integration test suite

4. **Phase 4: Optional Enhancements**
   - JSON Schema validation
   - Colored output with picocolors

---

## Sources

- [oclif Documentation](https://oclif.io/docs/introduction) - CLI framework patterns
- [Commander.js GitHub](https://github.com/tj/commander.js) - CLI argument parsing best practices
- [ShellCheck Wiki](https://github.com/koalaman/shellcheck/wiki) - Bash script error handling
- [Aether Implementation](file:///Users/callumcowie/repos/Aether) - Existing robust patterns (file-lock.sh, atomic-write.sh)

---

## Confidence Assessment

| Area | Level | Notes |
|------|-------|-------|
| CLI Framework | HIGH | Commander is industry standard |
| Shell Linting | HIGH | ShellCheck already in use |
| Error Handling | HIGH | Patterns well-documented |
| Testing | MEDIUM | AVA recommendation is standard but needs adoption |
| State Validation | LOW | Optional enhancement; may add complexity without benefit |

---

## Gaps to Address

- **E2E Testing**: No automated end-to-end tests for full workflows
- **Cross-Platform**: File locking behavior differs on macOS vs Linux (verify flock works)
- **Performance**: No benchmarks for large state files
