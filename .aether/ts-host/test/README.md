# TS Host Test Suite

## Snapshot Tests

Snapshot tests verify that ceremony rendering (banners, spawn frames, stage separators, boxen output) produces deterministic, expected output. They catch unintended visual regressions.

### Running Tests

```bash
npm test
```

### Updating Snapshots

After intentional changes to ceremony rendering (e.g., new figlet font, border style):

```bash
npm run test:update-snapshots
```

This regenerates all `.txt` snapshot files in `__snapshots__/`. Review the diff before committing.

### What to Do If a Snapshot Test Fails

1. Check if the change was intentional (did you modify renderer code, templates, or config?).
2. If intentional: run `npm run test:update-snapshots` and commit the updated snapshots.
3. If unintentional: the failure caught a regression — fix the source code, do not update snapshots.

### Golden Workflow Normalization

The golden workflow snapshot strips non-deterministic values:
- Timestamps (ISO-8601 patterns) → `<TIMESTAMP>`
- Temporary directory paths → `<TMPDIR>`
- Process IDs → `<PID>`
- Durations → `<DURATION>`

This ensures the snapshot is stable across runs and machines.
