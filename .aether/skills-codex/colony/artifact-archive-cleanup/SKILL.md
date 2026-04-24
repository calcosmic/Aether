---
name: artifact-archive-cleanup
description: Use when stale or completed colony artifacts need safe archival, cleanup, or retrieval without losing evidence
type: colony
domains: [maintenance, archival, workspace]
agent_roles: [medic, keeper, chronicler]
workflow_triggers: [medic, seal]
task_keywords: [cleanup, archive, stale, artifact, retrieve]
priority: normal
version: "1.0"
---

# Artifact Archive Cleanup

## Purpose

Colonies accumulate artifacts as they work. Completed phase directories, old planning files, and stale data clutter the active workspace. This skill archives completed work, keeps the workspace clean, and allows retrieval when archived context is needed again.

## When to Use

- Workspace is cluttered with completed phase directories
- User says "clean up" or "archive old phases"
- Milestone completed and old phase artifacts should be archived
- User wants to see what can be cleaned up before archiving
- Archived work needs to be retrieved for reference

## Instructions

### 1. Scan for Cleanup Candidates

```
1. Read ROADMAP.md to identify completed phases
2. Scan .aether/phases/ for directories matching completed phases
3. Check for stale data in .aether/data/ (older than decay threshold)
4. Identify orphaned artifacts (files not referenced by any active phase)
5. Calculate space savings estimate
```

### 2. Dry-Run Preview

Before any archival, always offer a preview:

```
 CLEANUP PREVIEW
   Phase directories to archive:
     - phase-1-requirements/     (completed 5 days ago, 12 files)
     - phase-2-design/           (completed 3 days ago, 8 files)
     - phase-3-implementation/   (completed 1 day ago, 23 files)
   
   Stale data to archive:
     - .aether/data/colonization/ (last updated 7 days ago)
   
   Total: 43 files, ~{size} -> will be archived to .aether/archive/milestone-{N}/
   
   [Run with --confirm to execute]
```

### 3. Selective Archival

Support granular control:

```
--phase {N}       Archive only phase N
--milestone {N}   Archive all completed phases in milestone N
--before {date}   Archive everything completed before date
--except {list}   Archive everything except listed phases
--data-only       Only archive stale data, keep phase directories
```

### 4. Archive Process

```
1. Create archive directory: .aether/archive/milestone-{N}/
2. Generate archive manifest: list of files, sizes, timestamps, checksums
3. Move phase directories to archive location
4. Create lightweight pointers in original locations:
   - phase-{N}/.archived -> points to archive location
   - Contains summary: phase name, status, artifact count, archive path
5. Update ROADMAP.md with archive references
6. Emit cleanup pheromone (strength 0.3, fast decay)
```

### 5. Retrieval

```
colony-cleanup retrieve --phase {N}
colony-cleanup retrieve --milestone {N}
colony-cleanup retrieve --search "{keyword}"
```

Retrieval process:
```
1. Check archive manifest for matching items
2. Copy archived files back to active workspace
3. Do NOT remove from archive (archive is append-only)
4. Log retrieval in archive manifest
```

### 6. Safety Guarantees

- **Never delete**: Archival moves to .aether/archive/, never deletes
- **Checksums**: Every archived file gets a checksum for integrity verification
- **Append-only archive**: Archived files persist even after retrieval
- **Archive manifest**: Complete audit trail of what was archived when

## Key Patterns

- **Preview before action**: Always show what will happen before it happens.
- **Pointers, not holes**: Archived locations get pointer files so nothing breaks.
- **Append-only archive**: The archive only grows, never shrinks.
- **Retrieval is copy**: Retrieving from archive copies, doesn't move.

## Output Format

```
 CLEANUP -- {action}
   Archived: {count} files ({size})
   Location: .aether/archive/milestone-{N}/
   Pointers: {count} archive pointers created
   Space recovered: {size} in active workspace
   Manifest: .aether/archive/milestone-{N}/MANIFEST.json
```

## Examples

**Preview and execute:**
> "Preview: 3 completed phase directories (43 files) ready for archival. Run `colony-cleanup --confirm --milestone 1` to archive."
> "Archived 43 files to .aether/archive/milestone-1/. Active workspace reduced by 180kb. 3 pointer files created."

**Retrieve for reference:**
> "Retrieving phase-2 artifacts from archive. 8 files copied to .aether/phases/phase-2-design/. Archive preserved."
