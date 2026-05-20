# Command Catalog Classification Policy

> Source of truth: `cmd/testdata/command_catalog.json`
> Validation: `python3 scripts/verify_catalog_classified.py --strict`

## Why This Exists

Every command in the Aether runtime must carry classification metadata so that
documentation, parity matrices, and deprecation planning can be generated
automatically.  This policy prevents "invisible" commands from sneaking into
releases.

## When You Add a New Command

1. **Update the catalog** — add an entry to `cmd/testdata/command_catalog.json`
   with at least:
   - `name` — the command string
   - `short_description` — one-line summary
   - `classification` — see tiers below
   - `since_version` — e.g. `"v1.22"` or `"current"` if not yet released
   - `historical_presence` — map of version -> boolean presence (minimum 5 keys)

2. **Auto-classify** — run the helper to fill in missing fields:
   ```bash
   python3 scripts/classify_commands.py
   ```

3. **Validate** — make sure the catalog passes the gate:
   ```bash
   python3 scripts/verify_catalog_classified.py --strict
   ```
   CI will run the same check; a failing PR cannot merge.

## Classification Tiers

| Tier | Meaning | Examples |
|------|---------|----------|
| **public_lifecycle** | Core colony lifecycle commands users run regularly | `init`, `plan`, `build`, `continue`, `seal`, `colonize`, `run` |
| **public_utility** | Day-to-day utility commands | `status`, `focus`, `redirect`, `feedback`, `pheromones`, `watch` |
| **internal_runtime** | Runtime plumbing, not for end users | `autofix-checkpoint`, `error-pattern-check` |
| **alias** | Shorthand or re-export of another command | `watch` (alias for `pheromone-display`), `pheromone-export-xml` |
| **deprecated** | Scheduled for removal | any command whose description contains "deprecated" |

### How classification is assigned

The `scripts/classify_commands.py` script uses these rules in order:

1. If the description contains "deprecated" → `deprecated`
2. If the name is in the internal list or starts with `hook-` / `autofix-` → `internal_runtime`
3. If the name is in the alias list or description contains "alias" → `alias`
4. If the command is a `host` subcommand, map by host category
5. If the name matches lifecycle roots or known finalize variants → `public_lifecycle`
6. Everything else → `public_utility`

You can override the auto-assigned value by editing the JSON directly; the
verification script only checks that the value is valid, not how it got there.

## Validation Rules

`verify_catalog_classified.py` enforces:

- Every entry has a `classification` field with a valid tier value.
- Every entry has a non-empty `since_version` string.
- Every entry has an `historical_presence` object with at least 5 version keys.

With `--strict` it also checks that counts stay within expected ranges:

| Tier | Expected Range |
|------|----------------|
| public_lifecycle | 10 – 25 |
| alias | 5 – 15 |
| internal_runtime | 0 – 5 |
| deprecated | 0 – 10 |

`public_utility` is intentionally unbounded.

## CI Gate

The `verify-catalog` step in `.github/workflows/ci.yml` runs:

```bash
python3 scripts/verify_catalog_classified.py --strict
```

If this exits non-zero, the build fails.  Treat it as a required check.

## Source of Truth Chain

1. `cmd/testdata/command_catalog.json` — canonical catalog
2. `aether audit-catalog` — runtime command that reads the same file
3. `scripts/verify_catalog_classified.py` — CI / local gate
4. `scripts/classify_commands.py` — bulk auto-classification helper

When the catalog changes, rerun both helper and gate before committing.
