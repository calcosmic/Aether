---
schema_version: "1.0"
id: stale-publish-diagnosis-playbook
kind: playbook
category: playbooks
title: Stale Publish Diagnosis Playbook
description: "Detecting and recovering from stale hub versions, channel mismatches, and outdated publishes."
output_types: [diagnosis-report, distribution-review, recovery-plan]
agent_roles: [builder, watcher, medic, queen, porter]
task_types: [stale, publish, diagnosis, update, recovery]
task_keywords: [stale, publish, version, hub, update, mismatch, recovery, channel, block, drift, integrity, corruption]
workflow_triggers: [build, update]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---

# Stale Publish Diagnosis Playbook

This playbook describes how to detect, diagnose, and recover from stale hub
publishes, version mismatches, and channel conflicts.

## For Beginners

When you publish Aether changes, they go to a "hub" directory on your machine.
Other projects pull from that hub when they run `aether update`. If the hub is
out of date -- perhaps you forgot to publish after making changes, or the
publish was incomplete -- projects get stale files. This playbook helps you
figure out what went wrong and fix it.

## Symptoms of a Stale Publish

Watch for these indicators:

| Symptom | Likely Cause |
|---------|-------------|
| `aether update --force` shows "0 copied, 0 unchanged" for commands | Hub publish is incomplete or missing |
| `aether version --check` returns non-zero exit code | Binary and hub versions disagree |
| New command not available after update | Command was added to source but not published |
| Agent definition missing in target repo | Agent mirror not published to hub |
| `aether update` blocks with stale publish warning | Critical stale publish detected |

## Diagnosis Steps

### Step 1: Check Version Agreement

```bash
aether version --check
```

Exit code 0 means binary and hub agree. Non-zero means they disagree.

If versions disagree:
- The binary may be newer than the hub (publish needed)
- The hub may be newer than the binary (update needed with `--download-binary`)
- The hub may be corrupted (integrity check needed)

### Step 2: Check Integrity

```bash
aether integrity
```

This validates the full release pipeline chain:
- Source files match hub files
- Binary version matches hub version
- Companion files are complete (no gaps)
- Downstream simulation passes

The integrity command reports which specific files are out of sync.

### Step 3: Inspect Hub Contents

Check what is actually in the hub:

```bash
ls ~/.aether/system/commands/
ls ~/.aether/system/agents/
ls ~/.aether/system/skills/
```

Compare with source:
```bash
ls .aether/commands/
ls .claude/agents/ant/
ls .aether/skills/
```

If hub directories are missing files that exist in source, the publish was
incomplete.

### Step 4: Check Channel Alignment

Verify you are checking the correct channel:

```bash
echo $AETHER_CHANNEL    # Should be empty or "stable" for production
which aether             # Should resolve to the stable binary
aether version           # Shows version and channel
```

A common mistake is publishing to dev but checking the stable hub, or
vice versa.

## Recovery Procedures

### Case 1: Hub Is Stale (Most Common)

**Scenario:** You made changes in the Aether source repo but did not publish,
or the publish did not complete.

**Recovery:**

1. Go to the Aether source repo
2. Ensure all changes are committed
3. Publish to the correct channel:
   ```bash
   aether publish                              # Stable channel
   aether publish --channel dev --binary-dest "$HOME/.local/bin"  # Dev channel
   ```
4. Verify the publish succeeded:
   ```bash
   aether version --check
   aether source-check
   ```
5. In the target repo, pull the update:
   ```bash
   aether update --force
   ```

### Case 2: Binary Is Stale

**Scenario:** The hub has been updated but the binary is an older version.

**Recovery:**

1. Update with binary download:
   ```bash
   aether update --force --download-binary
   ```
2. Or rebuild from source:
   ```bash
   go build ./cmd/aether
   ```
3. Verify:
   ```bash
   aether version --check
   ```

### Case 3: Stale Publish Blocks Update

**Scenario:** Running `aether update` shows a critical stale publish warning
and refuses to proceed.

**Recovery:**

The warning message includes a recovery command. Follow it. Typically:

1. Go to the Aether source repo
2. Republish to the hub:
   ```bash
   aether publish
   ```
3. Return to the target repo and retry:
   ```bash
   aether update --force
   ```

Critical stale publishes are detected automatically during update. The update
command refuses to proceed because the hub version is significantly behind
what the binary expects, which could cause data format issues.

### Case 4: Hub Corruption

**Scenario:** Hub files are corrupted or partially written (e.g., from an
interrupted publish).

**Recovery:**

1. Check integrity:
   ```bash
   aether integrity
   ```
2. If corruption is detected, republish from source:
   ```bash
   aether publish
   ```
3. If the source repo is not available, reinstall from the package:
   using the install command as a bootstrap
4. Verify after recovery:
   ```bash
   aether integrity
   ```

### Case 5: Channel Cross-Contamination

**Scenario:** Dev files ended up in the stable hub or vice versa.

**Recovery:**

1. Identify which channel is contaminated:
   ```bash
   aether integrity
   aether-dev integrity
   ```
2. Republish the correct channel from the Aether source repo
3. Verify isolation:
   ```bash
   aether version --check
   aether-dev version --check
   ```

## Prevention

- **Always publish after source changes.** Make it a habit: edit, commit,
  publish.
- **Run `aether integrity` before sealing.** Catch drift before it propagates.
- **Use `--force` with updates.** Without it, stale Aether-managed files may
  be left behind.
- **Keep channels isolated.** Never copy files between `~/.aether/` and
  `~/.aether-dev/` manually.
- **Verify after every publish.** Run `aether version --check` to confirm
  binary and hub agree.

## Quick Reference

| Command | Purpose |
|---------|---------|
| `aether version --check` | Verify binary and hub versions agree |
| `aether integrity` | Full pipeline validation |
| `aether source-check` | Source-to-mirror parity check |
| `aether publish` | Publish stable channel |
| `aether publish --channel dev` | Publish dev channel |
| `aether update --force` | Pull all updates from hub |
| `aether update --force --download-binary` | Pull updates including binary |
