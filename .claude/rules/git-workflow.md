# Git Workflow

## Commit Policy

- **Do not commit unless explicitly asked**
- **Do not push unless explicitly asked**
- Use concise commit messages: imperative mood, under 72 characters

## Commit Message Format

```
<type>: <short description>

# Types:
# feat:     New feature
# fix:      Bug fix
# docs:     Documentation only
# refactor: Code change without behavior change
# test:     Adding/modifying tests
# chore:    Maintenance tasks
```

## Branch Strategy

- `main` is protected - no direct commits
- Create feature branches from `main`
- PRs require passing CI checks

## Pre-commit Hooks

The repository uses pre-commit hooks to:
1. Block direct edits to `runtime/` (edit `.aether/` instead)
2. Run sync script before commit
3. Stage synced changes automatically

## Sync Workflow

```bash
# After editing .aether/ files:
git add .
git commit -m "docs: update workers.md"
npm install -g .   # Auto-syncs .aether/ → runtime/, pushes to hub
```
