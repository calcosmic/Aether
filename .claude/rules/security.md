# Security

## Protected Paths

Never edit these paths programmatically:

| Path | Reason |
|------|--------|
| `.aether/data/` | Colony state is precious |
| `.aether/checkpoints/` | Session checkpoints |
| `.aether/locks/` | File locks |
| `.env*` | Environment secrets |
| `.claude/settings.json` | Hook configuration |
| `.github/workflows/` | CI configuration |

## High-Risk Operations

Require explicit user approval:

- `rm -rf` and variants
- `sudo` commands
- `git push --force` or `--force-with-lease`
- `curl ... | bash` patterns
- Reading keychains or credential files
- Bulk `chmod -R` or `chown -R`

## Secret Handling

- Never log API keys or tokens
- Use environment variables for secrets
- Never commit `.env` files
- Redact secrets in error messages

## Checkpoint Safety

The build checkpoint system uses `git stash`. To prevent data loss:
- Only stash system files (allowlisted paths)
- Never stash user data or uncommitted work
- Always restore stash after operation
