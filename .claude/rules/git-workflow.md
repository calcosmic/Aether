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
1. Run package validation on `.aether/` changes (non-blocking)
2. Verify required files exist in `.aether/`

## Development Workflow

```bash
# After editing .aether/ files:
git add .
git commit -m "docs: update workers.md"
npm install -g .   # Validates .aether/ and pushes to hub
```
