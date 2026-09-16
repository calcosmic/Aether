# Aether Runtime Update Architecture

> Distribution note: the Go `aether` binary is the only runtime. The npm package
> at `npm/` is a thin bootstrapper for `npx --yes aether-colony@latest`; it
> downloads the published Go release and then runs the normal install flow.

Version rule:
- `.aether/version.json` is the source-checkout release version.
- `npm/package.json` must use the exact same version for public releases.
- The public npm `latest` dist-tag should match the current stable GitHub
  release version.

## Current Model

For dummies: the hub is the cupboard, not every project folder. Aether keeps
commands, agents, shipped skills, docs, templates, workers, exchange files, and
runtime references in the shared hub and platform home directories. Target repos
keep only their local colony state, local guidance files, Claude settings/rules,
and cleanup markers.

```text
+---------------------------+       aether publish        +---------------------------+
| Aether source checkout     | --------------------------> | ~/.aether/ or ~/.aether-dev/ |
|                           |                             |                           |
| cmd/, pkg/                | builds aether/aether-dev    | system/                   |
| .aether/                  | syncs companion files       |   workers.md              |
| .claude/                  | syncs wrapper surfaces      |   skills/ docs/ templates |
| .opencode/                | syncs agent surfaces        |   commands/ agents/ codex |
| .codex/                   | verifies version agreement  | version.json              |
+---------------------------+                             +-------------+-------------+
                                                                        |
                                                                        | aether update
                                                                        v
                                                          +---------------------------+
                                                          | target repo                |
                                                          | .aether/data/              |
                                                          | .aether/QUEEN.md           |
                                                          | AGENTS.md/.codex/.opencode |
                                                          | .claude/settings/rules     |
                                                          +---------------------------+
```

Stable `aether publish` also refreshes platform home assets:

| Platform | Destination |
|----------|-------------|
| Claude commands | `~/.claude/commands/ant-*.md` |
| Claude agents | `~/.claude/agents/ant/` |
| OpenCode commands | `~/.config/opencode/commands/ant/` |
| OpenCode agents | `~/.config/opencode/agents/` |
| Codex agents | `~/.codex/agents/` |
| Codex public skills and private support | `~/.codex/skills/aether/` |

Dev-channel publish skips platform homes by default. Explicit home-sync opt-in
and update refusal are distinct; see the Codex distribution contract below.

## What Goes Where

| Category | Source | Hub | Target repos |
|----------|--------|-----|--------------|
| Go runtime | `cmd/`, `pkg/` | built binary metadata | not copied by update |
| System docs/templates/skills/workers/exchange | `.aether/` | `system/` | global only; stale repo copies pruned |
| Claude commands | `.claude/commands/ant/` | `system/commands/claude/` and platform home | global/platform home only |
| Claude agents | `.claude/agents/ant/` | `system/agents-claude/` and platform home | global/platform home only |
| OpenCode commands | `.opencode/commands/ant/` | `system/commands/opencode/` and platform home | global/platform home only |
| OpenCode agents | `.opencode/agents/` | `system/agents/` and platform home | global/platform home only |
| Codex agents | `.codex/agents/` | `system/codex/` and platform home | global/platform home only |
| Shipped skills | `.aether/skills/` | `system/skills/` | hub only; runtime matches and injects in-process |
| Codex public skills/private support | validated runtime generation + three source support bodies | `system/codex-skills/` with versioned manifest | shared `~/.codex/skills/aether/`, consumed by install/publish/update |
| Claude settings/rules | `.claude/settings.json`, `.aether/rules/` | `system/settings/claude`, `system/rules/` | synced to `.claude/` |
| Project guidance | templates under `.aether/templates/` | `system/templates/` | synced as `AGENTS.md`, `.codex/CODEX.md`, `.opencode/OPENCODE.md` when managed |
| Colony state | generated in target repo | never | stays repo-local |

## Codex Skill Distribution Contract

An update keeps the nine ant actions and their instructions together. It leaves
custom files alone and asks for a fresh chat when the available skills change.
This route requires a runtime with Phase 204.1 payload support; an older installed
binary cannot acquire new update logic merely by copying instructions.

The versioned published payload lives at `system/codex-skills/` in the selected
hub, separate from the worker-skill library at `system/skills/`. It contains:

- `manifest.json` (`codex-skill-payload/v1`): source version, executing generator
  identity, minimum runtime version, command inventory, relative paths, modes,
  and SHA-256 content digests.
- Nine public `ant-*/SKILL.md` files: init, discuss, oracle, colonize, plan, build,
  continue, swarm, and seal.
- Three ordinary private support files: `support/aether-colony-creation.md`,
  `support/aether-colony-research.md`, and `support/aether-colony-build-cycle.md`.
  Each public skill resolves support relative to its installed file.

The selected discovery destination is `~/.codex/skills/aether/`. Plan 01's
[fresh-client receipt](.planning/phases/204.1-codex-ant-skill-surface/evidence/discovery-tracer.json)
qualifies that root on Codex CLI 0.154.0 for ant-plan entry routing. It is not proof
of all nine workflows, other client versions, or native-worker parity. Plan 07
owns fresh-client discovery and entry-routing proof for all nine names after
installation and upgrade.

Install, publish and ordinary stable update consume the same validated payload
and exact-file ownership planner. Publish/install generate and validate the hub
payload before home synchronization. Normal update reads those published bytes;
it does not regenerate skills from an older executable's command list. Compatible
later inventories are retained; missing, empty, partial, corrupt, unknown-schema,
or runtime-incompatible payloads refuse skill synchronization. The skill set and
its ownership record commit together. Normal update adds them to its existing
transaction; unrelated install/publish work is not a whole-operation transaction.

Installed `.aether-owned.json` (`codex-skill-ownership/v1`) records the exact
payload identity, file digests and modes. A name or ownership-looking header is
not permission to overwrite a file. Only matching recorded bytes and modes may
be replaced or retired. The frozen v1.0.79 digest inventory also recognizes the
fourteen original unmodified public/helper `SKILL.md` files. Edited or unknown
legacy files are preserved and reported; they can remain visible as custom names.
Unowned canonical-name collisions and edited owned support stop the skill-set
transaction, including under `--force`. Unrelated custom files survive. There is
no recursive skill-directory deletion and no silent repair of missing/corrupt
ownership over existing files. A lower-version payload cannot overwrite a newer
installed ownership record. Changes after planning, including new files or chmod,
refuse the original plan before writes rather than authorizing a refreshed view.

Managed project instructions keep their existing sentinel-based refresh policy.
Custom `AGENTS.md` and `.codex/CODEX.md` files retain their bytes and are identified
as preserved in results. This document policy is distinct from the stronger
content-and-mode ownership policy for skills; keeping a managed document sentinel
still opts that document into the existing refresh behavior.

One Codex refresh notice covers copied or removed skill assets and changed managed
Codex guidance/agents. A removal-only skill update also needs a fresh session;
unchanged or preserved files alone do not. The internal receipt label remains
`Skills (codex shims)` for compatibility.

Dev install/publish skip shared platform homes by default. Explicit
`--sync-platform-homes` on those commands opts into the existing **stable home**
destinations and reports that scope. Dev update keeps homes isolated and refuses
that flag. It does not silently enable the install/publish opt-in.

### Recovery Boundary

A failed or interrupted, unverified transaction can restore its prior bytes and
modes through the existing transaction recovery mechanism. Preserve its journal
and follow the returned recovery instructions; intervening owner edits are kept
and reported as recovery-required. Do not delete custom files, manifests, or
journals to force an update through. A verified successful receipt cannot be
undone with `LifecycleTransaction.Rollback`; post-success downgrade is unsupported.

For a missing/incompatible hub payload, bootstrap a capable runtime and republish
the matching source or install the matching supported release. Ordinary update
refreshes companions and supported home assets; it does not invent an unreleased
binary. See the [publish/update runbook](.aether/docs/publish-update-runbook.md#codex-skill-upgrade-and-recovery)
for bootstrap commands, proof commands and the merge verification boundary.

## Commands

### `aether publish` in the Aether repo

This is the preferred source-checkout publish path.

For dummies: `aether publish --channel stable` updates your local Aether
cupboard and local binary. It is not the same thing as a public GitHub/npm
release. A public release also needs the committed version bump, a pushed
`vX.Y.Z` tag, GitHub release assets, and npm `latest` pointing at the same
version.

What it does:
1. Builds the channel binary (`aether` or `aether-dev`) unless
   `--skip-build-binary` is used.
2. Syncs source companion files into `~/.aether/system/` or
   `~/.aether-dev/system/`.
3. On stable, refreshes Claude/OpenCode/Codex platform home assets.
4. Verifies the built binary version and hub version agree.

Use:

```bash
# Stable/public channel
aether publish --channel stable --binary-dest "$HOME/.local/bin"

# Dev/isolated channel
aether publish --channel dev --binary-dest "$HOME/.local/bin"
```

### `aether install`

`install` remains the public bootstrap command and backward-compatible source
publish path. New users run it after installing the binary. Maintainers should
prefer `aether publish` because publish performs version agreement verification.

Use `go run ./cmd/aether publish ...` if the installed binary is too stale to
run the new publish logic. Use `go run ./cmd/aether install ...` only as a
fallback when publish itself is the broken code path.

### `aether update` in a target repo

Update pulls from the hub and prepares the target repo. It does not build or
publish the binary.

What it does now:
1. Verifies a hub version exists.
2. Ensures repo-local state scaffolding exists.
3. Prunes stale generated repo-local commands, agents, shipped skills, and old
   `.aether/` system copies.
4. Syncs `.claude/settings.json`, `.claude/rules/`, and managed project
   guidance docs.
5. On stable, plans the validated `system/codex-skills` payload, exact legacy
   retirements and ownership record in the same maintenance transaction.
6. Preserves `.aether/data/`, dreams, oracle, locks, local `QUEEN.md`, and other
   local colony state.

Use:

```bash
# In a dirty target repo, preview first
aether update --force --dry-run

# Apply hub-backed cleanup and scaffolding refresh
aether update --force

# Published release binary refresh path
aether update --force --download-binary
```

### `aether setup`

Setup is the first-time target-repo preparation flow. It uses the same current
hub model as update: local state directories are created, but shared agents,
commands, skills, docs, templates, workers, exchange files, and references stay
global.

## Protected Paths

These are never overwritten by update/setup:

| Path | Reason |
|------|--------|
| `.aether/data/` | Colony state, pheromones, midden |
| `.aether/dreams/` | Session notes |
| `.aether/oracle/` | Research artifacts |
| `.aether/checkpoints/` | Session checkpoints |
| `.aether/locks/` | Runtime locks |
| `.aether/QUEEN.md` | Repo-local wisdom/preferences |
| `CROWNED-ANTHILL.md` | Seal marker |

## One Clean Workflow

```bash
# In the Aether source repo
aether publish --channel stable --binary-dest "$HOME/.local/bin"
aether integrity --source --channel stable

# In a target repo
aether update --force --dry-run
aether update --force
```

For isolated development:

```bash
# In the Aether source repo
aether publish --channel dev --binary-dest "$HOME/.local/bin"
aether-dev integrity --source --channel dev

# In a target repo
aether-dev update --force --dry-run
aether-dev update --force
```

## Failure Signatures

| Symptom | Meaning | Fix |
|---------|---------|-----|
| `Aether hub not installed` | The selected channel hub has no readable `version.json` or `system/version.json` | Run `aether publish` or `aether install`; for dev use `aether publish --channel dev` |
| Hub version behind binary version | The binary was updated but the hub was not republished | Run `aether publish` in the Aether repo |
| Hub companion counts below expected | Publish did not populate commands/agents/skills into the hub | Run `aether publish`, then `aether integrity --source` |
| Target repo still has old `.codex/agents` or `.claude/commands/ant` copies | Old repo-local generated assets were left behind | Run `aether update --force` after a fresh publish |

## Expected Hub Counts

Current source inventory (counts alone do not prove payload validity):

| Surface | Expected |
|---------|----------|
| Claude commands | 64 |
| OpenCode commands | 64 |
| OpenCode agents | 27 |
| Codex agents | 27 |
| Hub shipped skills | 86 |
| Codex public skill payload | 9 public skills + 3 private support files + manifest |

## Agent Parity Model

Agent definitions are canonical platform sources:

| Platform | Source |
|----------|--------|
| Claude | `.claude/agents/ant/*.md` |
| OpenCode | `.opencode/agents/*` |
| Codex | `.codex/agents/*.toml` |

There are no repo-local packaging mirrors. `aether publish` installs these into
the hub and stable platform homes; target repo `aether update --force` removes
old generated copies rather than recreating them.
