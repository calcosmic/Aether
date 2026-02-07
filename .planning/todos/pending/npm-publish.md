# Publish aether-colony to npm

**Captured:** 2026-02-05
**Updated:** 2026-02-07

## Task

Publish `aether-colony@1.0.0` to the public npm registry so anyone can `npm install -g aether-colony`.

## Pre-publish checklist

- [ ] Test the system end-to-end in a fresh repo (not the Aether source repo)
- [ ] Fix repository URL in `package.json` (`callumcowie` → `calcosmic` or whatever the canonical org is)
- [ ] Run `npm adduser` to authenticate
- [ ] Decide if `aether-colony` is the final package name
- [ ] `npm publish`
- [ ] Verify: `npm install -g aether-colony` on a clean machine → `aether version` → 1.0.0

## Notes

- Package name `aether-colony` is available on npm (checked 2026-02-05)
- Tarball is 78.8 kB (29 files) — well under npm norms
- Postinstall copies to `~/.claude/commands/ant/` and `~/.aether/` — some users run `--ignore-scripts` and would need manual `aether install`
- Once a version is published, it's permanent — can't re-publish same version number
