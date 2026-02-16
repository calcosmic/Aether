# Spawn Discipline

## Global Limits

| Metric | Limit | Reason |
|--------|-------|--------|
| Max spawn depth | 3 | Prevent runaway recursion |
| Max spawns at depth 1 | 4 | Parallelism cap |
| Max spawns at depth 2 | 2 | Secondary cap |
| Global workers per phase | 10 | Hard ceiling |

## Spawn Rules

1. **Never spawn beyond depth 3** - Workers at depth 3 cannot spawn
2. **Check before spawning** - Verify spawn budget is available
3. **Prefer sequential for dependencies** - If task B needs task A's output, don't parallelize
4. **Use appropriate castes** - Match worker type to task (see caste table in CLAUDE.md)

## Spawn Tree Tracking

All spawns are logged to `.aether/data/spawn-tree.txt`:
```
QUEEN (depth 0)
├── builder-1 (depth 1)
│   └── watcher-1 (depth 2)
└── scout-1 (depth 1)
```

## Model Routing

> **Note:** Model-per-caste routing is currently aspirational. All workers use the default model. See `TO-DOS.md` for verification status.
