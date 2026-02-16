# Aether-Specific Rules

## Source of Truth

```
.aether/           → SOURCE OF TRUTH (edit this)
runtime/           → STAGING (auto-populated, DO NOT EDIT)
```

## Directory Purpose

| Directory | Purpose |
|-----------|---------|
| `.aether/workers.md` | Worker definitions |
| `.aether/aether-utils.sh` | Utility layer |
| `.aether/utils/` | Helper scripts |
| `.aether/docs/` | Distributed documentation |
| `.aether/data/` | LOCAL - colony state |
| `.aether/dreams/` | LOCAL - session notes |
| `.aether/chambers/` | Archived colonies |

## Distribution Flow

```
Aether Repo → Hub (~/.aether/) → Target Repos
```

1. Edit `.aether/` in this repo
2. Run `npm install -g .` to sync to `runtime/` and push to hub
3. Run `aether update` in target repos to receive updates

## Pheromone Signals

| Signal | Command | Use When |
|--------|---------|----------|
| FOCUS | `/ant:focus "area"` | Steering attention |
| REDIRECT | `/ant:redirect "avoid"` | Hard constraint |
| FEEDBACK | `/ant:feedback "note"` | Gentle adjustment |

## Milestone Progression

```
First Mound → Open Chambers → Brood Stable → Ventilated Nest → Sealed Chambers → Crowned Anthill
```

## Key Commands

```bash
npm run lint:sync    # Verify command sync
npm test             # Run tests
aether update        # Pull latest from hub
```
