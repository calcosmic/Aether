#!/usr/bin/env node
// Clash Detection PreToolUse Hook
//
// Runs before Edit/Write tool calls in Claude Code.
// Checks if any other active git worktree has uncommitted changes to the
// same file, preventing agents from silently overwriting each other's work.
//
// Exit codes:
//   0 - Allow the operation
//   2 - Block the operation (conflict detected)
//
// Design principle: fail-open. If the hook errors for any reason,
// the operation is allowed. The goal is safety guidance, not blocking work.

const { execSync } = require('child_process');
const path = require('path');

// Files that are branch-local state and never clash across worktrees
const ALLOWLIST = [
  '.aether/data/',
];

// Timeout for clash-detect subprocess (ms)
const DETECT_TIMEOUT = 5000;

let input = '';
const stdinTimeout = setTimeout(() => process.exit(0), 3000);
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => input += chunk);
process.stdin.on('end', () => {
  clearTimeout(stdinTimeout);
  try {
    const data = JSON.parse(input);
    const toolName = data.tool_name;

    // Only check Edit and Write operations
    if (toolName !== 'Edit' && toolName !== 'Write') {
      process.exit(0);
    }

    // Extract file path
    const filePath = data.tool_input?.file_path || '';
    if (!filePath) {
      process.exit(0);
    }

    // Check allowlist (branch-local state files never clash)
    if (ALLOWLIST.some(pattern => filePath.includes(pattern))) {
      process.exit(0);
    }

    // Find the clash-detect script
    // It lives at .aether/utils/clash-detect.sh relative to the repo root
    const cwd = data.cwd || process.cwd();
    const repoClashDetect = path.join(cwd, '.aether', 'utils', 'clash-detect.sh');

    // Extract just the relative file path for clash-detect
    const relPath = path.relative(cwd, filePath);

    // Determine which clash-detect to use: repo-local or PATH
    const fs = require('fs');
    const clashDetect = fs.existsSync(repoClashDetect) ? repoClashDetect : 'clash-detect';

    // Run clash-detect
    try {
      const cmd = clashDetect === 'clash-detect'
        ? `clash-detect --file "${relPath}"`
        : `bash "${clashDetect}" --file "${relPath}"`;
      const result = execSync(cmd, {
        timeout: DETECT_TIMEOUT,
        encoding: 'utf8',
        cwd: cwd,
        stdio: ['pipe', 'pipe', 'pipe'],
      });

      const parsed = JSON.parse(result.trim());
      if (parsed.ok && parsed.result?.conflict) {
        const worktrees = (parsed.result.conflicting_worktrees || []).join(', ');
        const output = {
          decision: 'block',
          reason: `File "${relPath}" has uncommitted changes in worktree(s): ${worktrees}. ` +
            'Coordinate with the other agent or wait for their changes to be committed.',
        };
        process.stderr.write(JSON.stringify(output) + '\n');
        process.exit(2);
      }

      // No conflict -- allow
      process.exit(0);
    } catch (err) {
      // Fail-open: if clash-detect fails, allow the operation
      // This prevents the hook from blocking all work when something goes wrong
      process.exit(0);
    }
  } catch (e) {
    // Fail-open on any parsing or unexpected error
    process.exit(0);
  }
});
