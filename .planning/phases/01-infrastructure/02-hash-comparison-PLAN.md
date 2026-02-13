---
phase: 01-infrastructure
plan: 02
type: execute
wave: 1
depends_on: []
files_modified:
  - bin/cli.js
autonomous: true

must_haves:
  truths:
    - "syncSystemFilesWithCleanup compares file hashes before copying"
    - "Files with identical content are skipped (not copied)"
    - "Only changed files trigger actual filesystem writes"
  artifacts:
    - path: "bin/cli.js"
      provides: "Hash comparison in syncSystemFilesWithCleanup function"
      contains: "crypto.createHash|fs.readFileSync|hash.digest"
  key_links:
    - from: "syncSystemFilesWithCleanup"
      to: "crypto module"
      via: "hash comparison before fs.copyFileSync"
---

<objective>
Add hash comparison to syncSystemFilesWithCleanup to prevent unnecessary file writes.

Purpose: The current implementation copies all system files on every update, even if they haven't changed. This causes unnecessary filesystem writes and potentially triggers unnecessary file watchers. Hash comparison ensures only actually changed files are written.
Output: Modified syncSystemFilesWithCleanup function that computes SHA-256 hashes and skips files with matching content.
</objective>

<execution_context>
@~/.claude/cosmic-dev-system/workflows/execute-plan.md
@~/.claude/cosmic-dev-system/templates/summary.md
</execution_context>

<context>
@/Users/callumcowie/repos/Aether/bin/cli.js

The syncSystemFilesWithCleanup function (lines 279-317) currently:
1. Iterates through SYSTEM_FILES array
2. Copies each file from src to dest if it exists
3. Removes files from dest that no longer exist in src

The function needs to be modified to:
1. Compute SHA-256 hash of source file content
2. Compute SHA-256 hash of destination file content (if exists)
3. Only copy if hashes differ or destination doesn't exist
4. Track skipped files separately from copied files

The crypto module is already imported at the top of the file (line 5).
</context>

<tasks>

<task type="auto">
  <name>Add hash comparison to syncSystemFilesWithCleanup</name>
  <files>bin/cli.js</files>
  <action>
    Modify the syncSystemFilesWithCleanup function in bin/cli.js to add hash comparison.

    Changes needed:
    1. Add a helper function computeFileHash(filePath) that returns SHA-256 hash of file content, or null if file doesn't exist. Use crypto.createHash('sha256').

    2. Modify syncSystemFilesWithCleanup to:
       - Initialize skipped counter: let skipped = 0;
       - For each file in SYSTEM_FILES:
         - Compute srcHash = computeFileHash(srcPath)
         - Compute destHash = computeFileHash(destPath) if dest exists
         - If srcHash === destHash, increment skipped and continue (don't copy)
         - Otherwise, copy as before and increment copied
       - Return { copied, removed, skipped } instead of just { copied, removed }

    3. Update all call sites of syncSystemFilesWithCleanup to handle the new skipped field:
       - Line 483: const systemResult = syncSystemFilesWithCleanup(...)
       - Check if skipped count is used anywhere and update accordingly

    The hash computation should handle errors gracefully (return null for unreadable files).

    DO NOT change the function signature or behavior beyond adding hash comparison and the skipped return value.
  </action>
  <verify>
    node -e "
      const fs = require('fs');
      const crypto = require('crypto');

      // Check that computeFileHash function exists
      const content = fs.readFileSync('bin/cli.js', 'utf8');
      if (!content.includes('computeFileHash')) {
        console.error('Missing computeFileHash function');
        process.exit(1);
      }

      // Check that skipped is returned
      if (!content.includes('skipped')) {
        console.error('Missing skipped tracking');
        process.exit(1);
      }

      // Check for hash comparison logic
      if (!content.includes('sha256') && !content.includes('createHash')) {
        console.error('Missing hash computation');
        process.exit(1);
      }

      console.log('Hash comparison implementation verified');
    "
  </verify>
  <done>
    syncSystemFilesWithCleanup computes SHA-256 hashes for source and destination files, skips copying when hashes match, and returns { copied, removed, skipped }.
  </done>
</task>

</tasks>

<verification>
- [ ] computeFileHash helper function exists and uses crypto.createHash('sha256')
- [ ] syncSystemFilesWithCleanup compares hashes before copying
- [ ] Identical files are skipped (not copied)
- [ ] Return value includes skipped count
- [ ] All call sites handle the new skipped field
- [ ] Function handles missing/unreadable files gracefully
</verification>

<success_criteria>
- Files with identical content are not re-copied
- Update operations are idempotent (running twice doesn't change files the second time)
- No errors introduced to existing functionality
- Skipped files are reported in output
</success_criteria>

<output>
After completion, create `.planning/phases/01-infrastructure/01-infrastructure-02-SUMMARY.md`
</output>
