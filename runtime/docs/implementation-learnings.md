# Implementation Learnings

Valuable findings from Aether v3.1 development that apply to other projects.

## Workflow Patterns

1. **Claude Code global sync** works by copying commands from `.claude/commands/` to `~/.claude/commands/`
   - This enables sharing commands across all projects
   - Source of truth is the repo, hub is the distribution mechanism

2. **OpenCode requires repo-local setup**
   - Each repo that wants ant commands must set them up locally
   - Unlike Claude Code, there's no global command directory

3. **Hash comparison prevents unnecessary file writes**
   - Comparing SHA-1 hashes before writes preserves file timestamps
   - Reduces git noise from unchanged files

4. **Namespace isolation via 'ant:' prefix**
   - Prevents collisions with other agents
   - Creates clear command ownership

5. **CLI sync verification catches content drift**
   - `generate-commands.sh check` uses SHA-1 checksums
   - Detects when .claude/ and .opencode/ mirrors diverge

## Error Handling Patterns

- Use `json_err "$E_*" "message"` consistently (not bare strings)
- Always release locks in error paths (use trap or explicit release)
- Validate JSON before atomic writes
- Use fallback json_err for backward compatibility

## File Operation Patterns

- Use `atomic_write()` for all state modifications
- Acquire locks before reading/writing shared state
- Create backups BEFORE validation for true atomicity
- Use temp file + mv pattern for atomic operations

## Model Routing Patterns

- Validate LiteLLM proxy health before spawning
- Support CLI --model override for one-time changes
- Log actual model used per spawn for telemetry
- Use task-based routing keywords for automatic selection

## Codebase Patterns Discovered

### Pattern 1: JSON Error Response Standard
All commands output JSON with `{"ok": true/false, "result": ...}` or `{"ok": false, "error": ...}`

### Pattern 2: Feature Degradation Pattern
```bash
if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
  json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
else
  acquire_lock ...
fi
```

### Pattern 3: Atomic Write Pattern
All state modifications use `atomic_write()` from atomic-write.sh (temp file + mv)

### Pattern 4: Trap-based Cleanup
file-lock.sh uses `trap cleanup_locks EXIT TERM INT` for lock cleanup

### Pattern 5: Inconsistent Error Code Evolution
Commands added early use hardcoded strings; commands added later use `$E_VALIDATION_FAILED` constant. This creates inconsistency that should be standardized.

## Connections Between Issues

1. **BUG-005** (flag-auto-resolve lock leak) and **BUG-002** (flag-add lock leak) share the same root cause: inconsistent error handling patterns
2. **BUG-004** and **ISSUE-001** are the same issue: missing error code constants
3. **ISSUE-004** (template path) affects **GAP-006** (missing docs) - both relate to queen-* command usability
4. **BUG-001** (awk apostrophes) affects the same lines that use Pattern 2 (feature degradation)

## Fix Priority Matrix

| Priority | Issues | Effort | Impact |
|----------|--------|--------|--------|
| Critical | BUG-005, BUG-011 | Low | Deadlock prevention |
| High | BUG-002, BUG-008 | Low | Lock safety |
| Medium | BUG-003, BUG-006, BUG-007 | Medium | Code quality |
| Low | ISSUE-001 through ISSUE-007 | Medium | Consistency |

---

*Preserved from Phase 0 cleanup - 2026-02-15*
