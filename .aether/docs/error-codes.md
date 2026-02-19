# Aether Error Code Reference

This document is the complete reference for all `E_*` error constants used in Aether's bash utilities and Node.js CLI. When a command fails, the error output includes a `code` field that maps to one of the entries below.

**How to read this document:** Find the code you see in the error output, then check the meaning, when it happens, and what to do.

---

## File Errors

### E_FILE_NOT_FOUND

- **Meaning:** A required file or directory doesn't exist at the expected path.
- **When it happens:**
  - A command is invoked but the data file it needs (e.g., `COLONY_STATE.json`, `flags.json`, `CONTEXT.md`) hasn't been created yet.
  - A directory path passed as an argument doesn't exist.
- **Suggested fix:** Check that `aether init` (or the relevant setup command) has been run. Verify the path exists and is spelled correctly.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_FILE_NOT_FOUND","message":"Couldn't find CONTEXT.md. Try: run context-update init first.","details":null,"recovery":"Check file path and permissions","timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## Lock Errors

### E_LOCK_FAILED

- **Meaning:** Couldn't acquire a file lock — another process is currently holding it.
- **When it happens:**
  - Two commands try to write to the same file (e.g., `flags.json`) at the same time.
  - A previous command crashed and didn't release its lock. Use `force-unlock` to clear stale locks.
- **Suggested fix:** Wait for any running operations to finish, then retry. If you're sure nothing else is running, use `aether force-unlock` to clear the lock manually.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_LOCK_FAILED","message":"Failed to acquire lock on flags.json","details":null,"recovery":"Wait for other operations to complete","timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## Tool / Dependency Errors

### E_FEATURE_UNAVAILABLE

- **Meaning:** A feature or optional tool required for this operation isn't installed or enabled.
- **When it happens:**
  - XML operations are attempted but `xmllint` isn't installed.
  - A feature has been disabled due to a missing dependency or configuration.
- **Suggested fix:** Install the required tool (e.g., `brew install libxml2` for xmllint on macOS) or check the feature's documentation for setup steps.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_FEATURE_UNAVAILABLE","message":"xmllint not available. Try: brew install libxml2","details":null,"recovery":null,"timestamp":"2026-02-19T13:00:00Z"}}
  ```

### E_DEPENDENCY_MISSING

- **Meaning:** A required utility script or binary is missing from the expected location.
- **When it happens:**
  - A utility script in `.aether/utils/` (e.g., `error-handler.sh`, `file-lock.sh`) can't be found.
  - A required binary (e.g., `jq`, `git`) isn't installed or isn't on `$PATH`.
- **Suggested fix:** Run `aether install` to restore missing system files, or install the missing binary via your system package manager.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_DEPENDENCY_MISSING","message":"Couldn't load error-handler.sh. Try: run aether install to restore system files.","details":null,"recovery":"Install the required dependency","timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## JSON / Data Errors

### E_JSON_INVALID

- **Meaning:** A JSON file is malformed or is missing required fields.
- **When it happens:**
  - A data file (`COLONY_STATE.json`, `flags.json`, etc.) has been manually edited and contains a syntax error.
  - A write operation was interrupted, leaving a partially-written file.
- **Suggested fix:** Open the file and fix the JSON syntax, or delete it and re-run the initialization command to regenerate a fresh copy.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_JSON_INVALID","message":"Failed to parse COLONY_STATE.json. Try: validate with jq '.'' .aether/data/COLONY_STATE.json","details":null,"recovery":"Validate JSON syntax","timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## Validation Errors

### E_VALIDATION_FAILED

- **Meaning:** The command was called with missing or invalid arguments.
- **When it happens:**
  - A required argument (e.g., flag type, flag ID, spawn summary) is missing.
  - An argument value is outside the allowed range or format.
- **Suggested fix:** Check the usage message in the error output. The error message typically includes the correct usage syntax.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_VALIDATION_FAILED","message":"Usage: flag-add <type> <title> <description> [source] [phase]","details":null,"recovery":null,"timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## System Errors

### E_BASH_ERROR

- **Meaning:** An unexpected system command failure occurred — a bash command returned a non-zero exit code where one wasn't expected.
- **When it happens:**
  - A system command (e.g., `cp`, `mkdir`, `git`) fails unexpectedly.
  - A script runs under `set -e` and a command exits non-zero.
- **Suggested fix:** Check the `details` field in the error output for the exact command and line number. Fix the underlying system issue (permissions, disk space, etc.) and retry.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_BASH_ERROR","message":"Bash command failed","details":{"line":42,"command":"cp src dst","exit_code":1},"recovery":null,"timestamp":"2026-02-19T13:00:00Z"}}
  ```

### E_UNKNOWN

- **Meaning:** An unclassified error occurred — something went wrong that doesn't fit a more specific category.
- **When it happens:**
  - An error path that hasn't been given a specific code yet.
  - A catch-all for unexpected failure conditions.
- **Suggested fix:** Check the error message for context. If you see this frequently, please report it so a specific code can be added.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_UNKNOWN","message":"An unknown error occurred","details":null,"recovery":null,"timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## Hub / Repository Errors

### E_HUB_NOT_FOUND

- **Meaning:** The Aether hub (`~/.aether/`) doesn't exist on this machine.
- **When it happens:**
  - Aether hasn't been installed yet.
  - The hub directory was accidentally deleted.
- **Suggested fix:** Run `npm install -g aether` to install Aether. This creates the hub at `~/.aether/`.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_HUB_NOT_FOUND","message":"Couldn't find the Aether hub. Try: npm install -g aether","details":null,"recovery":"Run: aether install","timestamp":"2026-02-19T13:00:00Z"}}
  ```

### E_REPO_NOT_INITIALIZED

- **Meaning:** The current repository hasn't been initialized with Aether.
- **When it happens:**
  - Running an Aether command in a repo before running `/ant:init`.
  - The `.aether/` directory or `COLONY_STATE.json` is missing from the current repo.
- **Suggested fix:** Run `/ant:init` in Claude Code to initialize Aether in this repository.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_REPO_NOT_INITIALIZED","message":"Couldn't find Aether initialization in this repo. Try: run /ant:init first.","details":null,"recovery":"Run /ant:init in this repo first","timestamp":"2026-02-19T13:00:00Z"}}
  ```

### E_GIT_ERROR

- **Meaning:** A git operation failed.
- **When it happens:**
  - `git stash`, `git commit`, `git log`, or another git command returns an error.
  - The directory isn't a git repository.
  - There are merge conflicts or a detached HEAD state.
- **Suggested fix:** Run `git status` to see the current state of your repository. Resolve any conflicts or uncommitted changes, then retry the command.
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_GIT_ERROR","message":"Git operation failed. Try: run git status to check repository state.","details":null,"recovery":"Check git status and resolve conflicts","timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## Resource Errors

### E_RESOURCE_NOT_FOUND

- **Meaning:** A runtime resource (e.g., an active session, a worker, a swarm) doesn't exist.
- **When it happens:**
  - A command refers to a session ID, worker name, or swarm ID that doesn't exist or has already completed.
  - Attempting to read or update a resource that hasn't been created yet.
- **Suggested fix:** Check that the resource was created first. List available resources with the relevant `list` command (e.g., `flag-list`, `swarm-findings-read`).
- **Example output:**
  ```json
  {"ok":false,"error":{"code":"E_RESOURCE_NOT_FOUND","message":"Couldn't find the requested resource. Try: check that it was created first.","details":null,"recovery":"Check that the resource exists and try again","timestamp":"2026-02-19T13:00:00Z"}}
  ```

---

## For Contributors

### Adding a New Error Code

When you need a new error code, follow this checklist:

1. **Define the constant** in `.aether/utils/error-handler.sh` at the top of the file:
   ```bash
   E_MY_NEW_CODE="E_MY_NEW_CODE"
   ```

2. **Add a recovery function** in `error-handler.sh`:
   ```bash
   _recovery_my_new_code() { echo '"Description of how to fix this"'; }
   ```

3. **Add a case entry** in the `_get_recovery` function in `error-handler.sh`:
   ```bash
   "$E_MY_NEW_CODE") _recovery_my_new_code ;;
   ```

4. **Add a fallback definition** at the top of `aether-utils.sh` (in the fallback constants block):
   ```bash
   : "${E_MY_NEW_CODE:=E_MY_NEW_CODE}"
   ```

5. **Export the constant and function** at the bottom of `error-handler.sh`:
   ```bash
   export E_MY_NEW_CODE
   export -f _recovery_my_new_code
   ```

6. **Document the code** in this file (`docs/error-codes.md`) following the existing format.

### Naming Convention

- **Prefix:** Always start with `E_`
- **Case:** SCREAMING_SNAKE_CASE (e.g., `E_FILE_NOT_FOUND`, `E_LOCK_FAILED`)
- **Be specific:** Prefer `E_FILE_NOT_FOUND` over `E_ERROR` — the code should communicate the failure category

### Category Selection Guide

| Category | Use when... |
|----------|-------------|
| File Errors | A file or directory path doesn't exist |
| Lock Errors | Lock acquisition fails |
| Tool/Dependency Errors | A required tool or script is missing |
| JSON/Data Errors | File content is malformed or invalid |
| Validation Errors | Arguments are wrong or missing |
| System Errors | Unexpected bash command failure |
| Hub/Repository Errors | Aether infrastructure is missing or not set up |
| Resource Errors | A named runtime object doesn't exist |

### Message Style Requirements

Every `json_err` call **must** include a "Try:" suggestion in the message:

```bash
# Correct — includes Try: suggestion
json_err "$E_FILE_NOT_FOUND" "Couldn't find flags.json. Try: run flag-add first to create it."

# Wrong — no actionable suggestion
json_err "$E_FILE_NOT_FOUND" "flags.json not found"
```

**Tone:** Use plain, friendly language. "Couldn't find..." is better than "File not found". Users are not experts in bash internals.

---

*Last updated: 2026-02-19*
