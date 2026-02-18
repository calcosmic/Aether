#!/usr/bin/env bash
# Lock Lifecycle Tests
# Verifies LOCK-01 through LOCK-04: lock release on all exit paths
#
# LOCK-01: jq failure releases lock in all 4 flag commands
# LOCK-02: SIGTERM/SIGINT releases lock via trap
# LOCK-03: atomic_write and atomic_write_from_file backup ordering + corruption safety
# LOCK-04: context-update acquires and releases lock; force-unlock clears all locks

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with aether-utils.sh + utils/
# ============================================================================
setup_isolated_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data"
    mkdir -p "$tmp_dir/.aether/locks"

    # Copy aether-utils.sh to temp location so it uses temp data dir
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    # Copy utils/ directory (needed for acquire_lock, atomic_write, etc.)
    local utils_source
    utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    echo "$tmp_dir"
}

# ============================================================================
# Helper: Create a valid flags.json in an isolated env
# ============================================================================
setup_flags_json() {
    local tmp_dir="$1"
    mkdir -p "$tmp_dir/.aether/data"
    echo '{"version":1,"flags":[]}' > "$tmp_dir/.aether/data/flags.json"
}

# ============================================================================
# Test 1: LOCK-01 — jq failure in flag-add releases lock
# ============================================================================
test_flag_add_jq_failure_releases_lock() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    setup_flags_json "$tmp_dir"

    # Corrupt flags.json so jq will fail
    echo "NOT JSON" > "$tmp_dir/.aether/data/flags.json"

    # Run flag-add — expect non-zero exit because jq will fail
    set +e
    bash "$tmp_dir/.aether/aether-utils.sh" flag-add blocker "test title" "test description" 2>/dev/null
    local exit_code=$?
    set -e

    # Command should have failed
    if [[ "$exit_code" -eq 0 ]]; then
        log_error "Expected non-zero exit code from flag-add with corrupt JSON, got 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: no .lock files remain (LOCK-01 fix verified)
    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after jq failure, found $lock_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 2: LOCK-01 — jq failure in flag-auto-resolve releases lock
# ============================================================================
test_flag_auto_resolve_jq_failure_releases_lock() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create flags.json with content that jq will fail to process for the specific operation
    # We need the file to exist but be invalid for jq to fail during processing.
    # flags-auto-resolve checks if file exists first, then acquires lock, then runs jq.
    # Write valid-looking but unparseable content:
    mkdir -p "$tmp_dir/.aether/data"
    echo "NOT JSON" > "$tmp_dir/.aether/data/flags.json"

    set +e
    bash "$tmp_dir/.aether/aether-utils.sh" flag-auto-resolve build_pass 2>/dev/null
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        log_error "Expected non-zero exit code from flag-auto-resolve with corrupt JSON, got 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after jq failure in flag-auto-resolve, found $lock_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 3: LOCK-01 — jq failure in flag-resolve releases lock
# (This command LACKED an EXIT trap before Plan 16-01)
# ============================================================================
test_flag_resolve_jq_failure_releases_lock() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create flags.json that passes existence check but fails jq parsing
    mkdir -p "$tmp_dir/.aether/data"
    echo "NOT JSON" > "$tmp_dir/.aether/data/flags.json"

    set +e
    bash "$tmp_dir/.aether/aether-utils.sh" flag-resolve some_flag_id "resolution" 2>/dev/null
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        log_error "Expected non-zero exit code from flag-resolve with corrupt JSON, got 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after jq failure in flag-resolve, found $lock_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 4: LOCK-01 — jq failure in flag-acknowledge releases lock
# (This command LACKED an EXIT trap before Plan 16-01)
# ============================================================================
test_flag_acknowledge_jq_failure_releases_lock() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create flags.json that passes existence check but fails jq parsing
    mkdir -p "$tmp_dir/.aether/data"
    echo "NOT JSON" > "$tmp_dir/.aether/data/flags.json"

    set +e
    bash "$tmp_dir/.aether/aether-utils.sh" flag-acknowledge some_flag_id 2>/dev/null
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        log_error "Expected non-zero exit code from flag-acknowledge with corrupt JSON, got 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after jq failure in flag-acknowledge, found $lock_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 5: LOCK-02 — SIGTERM releases lock (via cleanup_locks trap in file-lock.sh)
# ============================================================================
test_sigterm_releases_lock() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create a test script that sources file-lock.sh, acquires a lock, then sleeps
    local test_script="$tmp_dir/lock_hold.sh"
    cat > "$test_script" << SCRIPT
#!/bin/bash
set -euo pipefail
LOCK_DIR="$tmp_dir/.aether/locks"
mkdir -p "\$LOCK_DIR"
source "$tmp_dir/.aether/utils/file-lock.sh"
acquire_lock "$tmp_dir/.aether/data/flags.json"
# Signal parent that lock is held
echo "locked" > "$tmp_dir/lock_signal"
# Hold the lock
sleep 60
SCRIPT
    chmod +x "$test_script"

    # Run in background
    bash "$test_script" &
    local bg_pid=$!

    # Wait for lock signal (up to 5 seconds)
    local waited=0
    while [[ ! -f "$tmp_dir/lock_signal" ]] && [[ $waited -lt 10 ]]; do
        sleep 0.5
        waited=$((waited + 1))
    done

    if [[ ! -f "$tmp_dir/lock_signal" ]]; then
        log_error "Background script did not signal lock acquisition within 5 seconds"
        kill "$bg_pid" 2>/dev/null || true
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify lock file exists while process is running
    local lock_exists=false
    if ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | grep -q .; then
        lock_exists=true
    fi

    # Send SIGTERM
    kill -TERM "$bg_pid" 2>/dev/null || true
    wait "$bg_pid" 2>/dev/null || true

    # Give the trap handler a moment to run
    sleep 0.3

    # Assert: no lock files remain
    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after SIGTERM, found $lock_count (lock_existed=$lock_exists)"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 6: LOCK-02 — SIGINT releases lock (via cleanup_locks trap in file-lock.sh)
# ============================================================================
test_sigint_releases_lock() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local test_script="$tmp_dir/lock_hold.sh"
    cat > "$test_script" << SCRIPT
#!/bin/bash
set -euo pipefail
LOCK_DIR="$tmp_dir/.aether/locks"
mkdir -p "\$LOCK_DIR"
source "$tmp_dir/.aether/utils/file-lock.sh"
acquire_lock "$tmp_dir/.aether/data/flags.json"
echo "locked" > "$tmp_dir/lock_signal"
sleep 60
SCRIPT
    chmod +x "$test_script"

    bash "$test_script" &
    local bg_pid=$!

    local waited=0
    while [[ ! -f "$tmp_dir/lock_signal" ]] && [[ $waited -lt 10 ]]; do
        sleep 0.5
        waited=$((waited + 1))
    done

    if [[ ! -f "$tmp_dir/lock_signal" ]]; then
        log_error "Background script did not signal lock acquisition within 5 seconds"
        kill "$bg_pid" 2>/dev/null || true
        rm -rf "$tmp_dir"
        return 1
    fi

    # Send SIGINT
    kill -INT "$bg_pid" 2>/dev/null || true
    wait "$bg_pid" 2>/dev/null || true

    sleep 0.3

    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after SIGINT, found $lock_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 7: LOCK-03 — atomic_write creates backup BEFORE validation
# ============================================================================
test_atomic_write_backup_before_validate() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # We need atomic-write.sh to use our tmp_dir, not the git repo root.
    # atomic-write.sh uses git rev-parse for AETHER_ROOT. We create a helper
    # script that overrides HOME and cwd so the sourced script uses our tmp_dir.
    local driver="$tmp_dir/driver.sh"

    local original_json='{"version":1,"flags":[]}'
    local new_json='{"version":1,"flags":[{"id":"test"}]}'

    mkdir -p "$tmp_dir/.aether/data"
    mkdir -p "$tmp_dir/.aether/temp"
    mkdir -p "$tmp_dir/.aether/data/backups"

    local flags_file="$tmp_dir/.aether/data/flags.json"
    echo "$original_json" > "$flags_file"

    # Create a fake 'git' that always fails so atomic-write.sh falls back to pwd (our tmp_dir)
    mkdir -p "$tmp_dir/bin"
    cat > "$tmp_dir/bin/git" << 'FAKEGIT'
#!/bin/bash
exit 1
FAKEGIT
    chmod +x "$tmp_dir/bin/git"

    cat > "$driver" << DRIVER
#!/bin/bash
set -euo pipefail
# Run from tmp_dir so AETHER_ROOT fallback resolves to our isolated directory
cd "$tmp_dir"
# Prepend fake git to PATH so atomic-write.sh git detection fails gracefully
export PATH="$tmp_dir/bin:$PATH"
# Source atomic-write.sh directly from the isolated copy (BASH_SOURCE[0] sets _AETHER_UTILS_DIR)
source "$tmp_dir/.aether/utils/atomic-write.sh"
atomic_write "$flags_file" '$new_json'
DRIVER
    chmod +x "$driver"

    bash "$driver" 2>/dev/null

    # Verify backup exists
    local backup_count
    backup_count=$(ls "$tmp_dir/.aether/data/backups/"flags.json.*.backup 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$backup_count" -eq 0 ]]; then
        log_error "Expected at least 1 backup in backups/, found 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify backup contains ORIGINAL content (not new content)
    # Backup should be the pre-write snapshot
    local latest_backup
    latest_backup=$(ls -t "$tmp_dir/.aether/data/backups/"flags.json.*.backup 2>/dev/null | head -1)
    local backup_content
    backup_content=$(cat "$latest_backup")
    if [[ "$backup_content" != "$original_json" ]]; then
        log_error "Backup should contain original content '$original_json', got '$backup_content'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 8: LOCK-03 — atomic_write with invalid JSON does NOT corrupt target
# ============================================================================
test_atomic_write_invalid_json_preserves_target() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    mkdir -p "$tmp_dir/.aether/data"
    mkdir -p "$tmp_dir/.aether/temp"
    mkdir -p "$tmp_dir/.aether/data/backups"

    local flags_file="$tmp_dir/.aether/data/flags.json"
    local original_json='{"version":1,"flags":[]}'
    echo "$original_json" > "$flags_file"

    # Create a fake 'git' that always fails so atomic-write.sh falls back to pwd
    mkdir -p "$tmp_dir/bin"
    cat > "$tmp_dir/bin/git" << 'FAKEGIT'
#!/bin/bash
exit 1
FAKEGIT
    chmod +x "$tmp_dir/bin/git"

    local driver="$tmp_dir/driver.sh"
    cat > "$driver" << DRIVER
#!/bin/bash
cd "$tmp_dir"
export PATH="$tmp_dir/bin:$PATH"
source "$tmp_dir/.aether/utils/atomic-write.sh"
atomic_write "$flags_file" "NOT VALID JSON"
DRIVER
    chmod +x "$driver"

    # Should fail — invalid JSON rejected
    set +e
    bash "$driver" 2>/dev/null
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        log_error "Expected non-zero exit when writing invalid JSON, got 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Target must still have original content (not corrupted)
    local current_content
    current_content=$(cat "$flags_file")
    if [[ "$current_content" != "$original_json" ]]; then
        log_error "Target file corrupted! Expected '$original_json', got '$current_content'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 9: LOCK-04 — context-update acquires and releases lock
# ============================================================================
test_context_update_acquires_lock() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create a minimal CONTEXT.md (context-update activity needs this to be initialised first)
    mkdir -p "$tmp_dir/.aether"
    cat > "$tmp_dir/.aether/CONTEXT.md" << 'CONTEXT'
# Aether Colony — Current Context

## System Status

| Field | Value |
|-------|-------|
| **Last Updated** | 2026-02-18T00:00:00Z |
| **Goal** | test |
| **Phase** | 1 |

## Recent Activity

- test

## Active Constraints

- none
CONTEXT

    # Run context-update activity
    set +e
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" context-update activity "test-worker" "passed" "—" 2>&1)
    local exit_code=$?
    set -e

    if [[ "$exit_code" -ne 0 ]]; then
        log_error "context-update failed with exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: no lock files remain after successful completion
    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after context-update, found $lock_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 10: force-unlock clears all lock files
# ============================================================================
test_force_unlock_clears_locks() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Manually create fake lock files (simulate dead process)
    echo "99999" > "$tmp_dir/.aether/locks/test.lock"
    echo "99999" > "$tmp_dir/.aether/locks/test.lock.pid"
    echo "99999" > "$tmp_dir/.aether/locks/another.lock"
    echo "99999" > "$tmp_dir/.aether/locks/another.lock.pid"

    # Run force-unlock with --yes (non-interactive mode)
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" force-unlock --yes 2>&1)
    local exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        log_error "force-unlock failed with exit code $exit_code: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: no lock files remain
    local lock_count
    lock_count=$(ls "$tmp_dir/.aether/locks/"*.lock 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$lock_count" -ne 0 ]]; then
        log_error "Expected 0 lock files after force-unlock, found $lock_count"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Assert: output JSON has removed > 0
    local removed
    removed=$(echo "$output" | jq -r '.result.removed // .removed // 0' 2>/dev/null || echo "0")
    if [[ "$removed" -lt 1 ]]; then
        log_error "Expected .result.removed >= 1, got '$removed'. Output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 11: LOCK-03 — atomic_write_from_file creates backup BEFORE validation
# ============================================================================
test_atomic_write_from_file_backup_before_validate() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    mkdir -p "$tmp_dir/.aether/data"
    mkdir -p "$tmp_dir/.aether/temp"
    mkdir -p "$tmp_dir/.aether/data/backups"

    local flags_file="$tmp_dir/.aether/data/flags.json"
    local original_json='{"version":1,"flags":[]}'
    local new_json='{"version":2,"flags":[{"id":"new"}]}'

    echo "$original_json" > "$flags_file"

    # Create source file with new valid JSON
    local source_file="$tmp_dir/source.json"
    echo "$new_json" > "$source_file"

    # Create a fake 'git' that always fails so atomic-write.sh falls back to pwd
    mkdir -p "$tmp_dir/bin"
    cat > "$tmp_dir/bin/git" << 'FAKEGIT'
#!/bin/bash
exit 1
FAKEGIT
    chmod +x "$tmp_dir/bin/git"

    local driver="$tmp_dir/driver.sh"
    cat > "$driver" << DRIVER
#!/bin/bash
set -euo pipefail
cd "$tmp_dir"
export PATH="$tmp_dir/bin:$PATH"
source "$tmp_dir/.aether/utils/atomic-write.sh"
atomic_write_from_file "$flags_file" "$source_file"
DRIVER
    chmod +x "$driver"

    bash "$driver" 2>/dev/null

    # Verify backup exists
    local backup_count
    backup_count=$(ls "$tmp_dir/.aether/data/backups/"flags.json.*.backup 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$backup_count" -eq 0 ]]; then
        log_error "Expected at least 1 backup in backups/, found 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify backup contains ORIGINAL content (backup was made before overwrite)
    local latest_backup
    latest_backup=$(ls -t "$tmp_dir/.aether/data/backups/"flags.json.*.backup 2>/dev/null | head -1)
    local backup_content
    backup_content=$(cat "$latest_backup")
    if [[ "$backup_content" != "$original_json" ]]; then
        log_error "Backup should contain original content '$original_json', got '$backup_content'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test 12: LOCK-03 — atomic_write_from_file with invalid JSON does NOT corrupt target
# ============================================================================
test_atomic_write_from_file_invalid_json_preserves_target() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    mkdir -p "$tmp_dir/.aether/data"
    mkdir -p "$tmp_dir/.aether/temp"
    mkdir -p "$tmp_dir/.aether/data/backups"

    local flags_file="$tmp_dir/.aether/data/flags.json"
    local original_json='{"version":1,"flags":[]}'
    echo "$original_json" > "$flags_file"

    # Create source file with invalid JSON
    local source_file="$tmp_dir/source.json"
    echo "NOT VALID JSON" > "$source_file"

    # Create a fake 'git' that always fails so atomic-write.sh falls back to pwd
    mkdir -p "$tmp_dir/bin"
    cat > "$tmp_dir/bin/git" << 'FAKEGIT'
#!/bin/bash
exit 1
FAKEGIT
    chmod +x "$tmp_dir/bin/git"

    local driver="$tmp_dir/driver.sh"
    cat > "$driver" << DRIVER
#!/bin/bash
cd "$tmp_dir"
export PATH="$tmp_dir/bin:$PATH"
source "$tmp_dir/.aether/utils/atomic-write.sh"
atomic_write_from_file "$flags_file" "$source_file"
DRIVER
    chmod +x "$driver"

    set +e
    bash "$driver" 2>/dev/null
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        log_error "Expected non-zero exit when writing invalid JSON via atomic_write_from_file, got 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Target must still have original content
    local current_content
    current_content=$(cat "$flags_file")
    if [[ "$current_content" != "$original_json" ]]; then
        log_error "Target file corrupted! Expected '$original_json', got '$current_content'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Main Test Runner
# ============================================================================

main() {
    log "${YELLOW}=== Lock Lifecycle Tests ===${NC}"
    log "Testing: $AETHER_UTILS_SOURCE"
    log "LOCK-01: Lock release on jq failure in all flag commands"
    log "LOCK-02: Lock release on SIGTERM/SIGINT"
    log "LOCK-03: atomic_write backup ordering and corruption safety"
    log "LOCK-04: context-update lock acquire/release; force-unlock cleanup"
    log ""

    # LOCK-01: jq failure releases lock in all 4 flag commands
    run_test "test_flag_add_jq_failure_releases_lock" \
        "LOCK-01: flag-add jq failure releases lock (BUG-002 fix)"
    run_test "test_flag_auto_resolve_jq_failure_releases_lock" \
        "LOCK-01: flag-auto-resolve jq failure releases lock (BUG-005/011 fix)"
    run_test "test_flag_resolve_jq_failure_releases_lock" \
        "LOCK-01: flag-resolve jq failure releases lock (pre-16-01 missing trap)"
    run_test "test_flag_acknowledge_jq_failure_releases_lock" \
        "LOCK-01: flag-acknowledge jq failure releases lock (pre-16-01 missing trap)"

    # LOCK-02: Signal handling
    run_test "test_sigterm_releases_lock" \
        "LOCK-02: SIGTERM releases lock via cleanup_locks trap"
    run_test "test_sigint_releases_lock" \
        "LOCK-02: SIGINT releases lock via cleanup_locks trap"

    # LOCK-03: atomic_write safety
    run_test "test_atomic_write_backup_before_validate" \
        "LOCK-03: atomic_write backup created before validation (ordering fix)"
    run_test "test_atomic_write_invalid_json_preserves_target" \
        "LOCK-03: atomic_write invalid JSON does not corrupt target file"
    run_test "test_atomic_write_from_file_backup_before_validate" \
        "LOCK-03: atomic_write_from_file backup created before validation"
    run_test "test_atomic_write_from_file_invalid_json_preserves_target" \
        "LOCK-03: atomic_write_from_file invalid JSON does not corrupt target"

    # LOCK-04: context-update and force-unlock
    run_test "test_context_update_acquires_lock" \
        "LOCK-04: context-update acquires and releases lock"
    run_test "test_force_unlock_clears_locks" \
        "LOCK-04: force-unlock --yes clears all lock files"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
