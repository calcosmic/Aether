# Playbook: Seal

## Overview

Final review, audit, documentation, and archive. Seal marks the colony as complete (Crowned Anthill) and prepares all artifacts for long-term storage.

## Stage 1: Review

### Step 1: Load Colony State

Run `aether load-state`. Verify all phases are completed.

If any phase is not completed:
- Display incomplete phases
- Offer to run `/ant-continue` for each
- Do not proceed until all phases complete

### Step 2: Final Review

Generate summary of all completed work:
- Total phases completed
- Files created/modified
- Tests added
- Coverage achieved
- Key decisions made
- Learnings extracted

Display:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👑 F I N A L   R E V I E W
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Goal: {colony_goal}
Phases: {completed}/{total}
Files: {created} created, {modified} modified
Tests: {total} total, {coverage}% coverage

Key decisions:
  {list top decisions}

Learnings:
  {list top learnings}
```

## Stage 2: Audit

### Step 3: Auditor Gate

Spawn Auditor for final quality audit:
- Review all modified files
- Multi-lens analysis (security, performance, quality, maintainability)
- Overall quality score

If critical findings exist:
- Display findings
- Offer to fix now or document as known issues
- Do not seal with unacknowledged critical issues

### Step 4: Security Review

Run Gatekeeper for final supply chain audit:
- Check for new CVEs since last audit
- Verify license compliance
- Flag any critical security issues

If critical CVEs found, block seal until resolved.

## Stage 3: Document

### Step 5: Chronicler Documentation

Spawn Chronicler to produce final documentation:
- Update README with project overview
- Document architecture decisions
- Write deployment guide
- Document known issues and workarounds

Store in `.aether/docs/final/` and root-level docs.

### Step 6: Wisdom Synthesis

Promote high-confidence instincts to QUEEN.md:
- Sweep all instincts with confidence >= 0.8
- Run `aether queen-promote-instinct` for each
- Run `aether hive-promote` for cross-colony sharing

Display:
```
📚 Wisdom promoted: {count} instincts to QUEEN.md
🌐 Hive shared: {count} instincts to cross-colony hive
```

## Stage 4: Archive

### Step 7: Create Archive

Prepare archive of colony artifacts:
- `COLONY_STATE.json` (final)
- `CONTEXT.md`
- `CHANGELOG.md`
- Survey documents
- Research findings
- Handoff history

Store in `.aether/chambers/{colony_name}-{timestamp}/`.

### Step 8: Mark as Sealed

Update `COLONY_STATE.json`:
- Set `milestone` to `"Crowned Anthill"`
- Set `state` to `"SEALED"`
- Set `sealed_at` timestamp
- Append event: `"<timestamp>|sealed|seal|Colony sealed"`

### Step 9: Display Completion

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   👑 C R O W N E D   A N T H I L L 👑
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎉 Colony sealed successfully!

Goal achieved: {colony_goal}
Phases completed: {total}
Archive: .aether/chambers/{colony_name}-{timestamp}/

🐜 The colony rests. Well done!

Next:
  /ant-init "new goal" — Start a new colony
```

### Step 10: Update Session

Run `aether session-update --command "/ant-seal" --suggested-next "/ant-init" --summary "Colony sealed: {colony_goal}"`
