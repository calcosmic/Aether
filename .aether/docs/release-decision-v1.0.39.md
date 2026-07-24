# Release Decision: v1.0.39 Hybrid Aether Smoke

Date: 2026-05-18 (Europe/Zurich workspace date)
Version checked: 1.0.39
Decision: Conditional go after commit; do not public-release from the dirty tree.

## For Dummies

The local stable publish now works like a real install path: Aether can publish
itself to the hub, install/update into a brand new repo, and run the TypeScript
host commands without secretly depending on this source checkout.

The only release blocker left in this check is bookkeeping: there are uncommitted
changes. That is a good blocker. It means the code is behaving, but the public
release should wait until the current work is intentionally committed and the
porter gate is rerun.

## What Phase 8 Proved

Fresh repo smoke:

- Published local stable with `go run ./cmd/aether publish --channel stable --binary-dest "$HOME/.local/bin"`.
- Created a clean temp repo at `/var/folders/pj/fn0nrs6s1zj_pm7s486lnnz40000gn/T//aether-fresh-smoke-u4NbRM`.
- Ran `aether lay-eggs`.
- Ran `aether update --force`.
- Verified `.aether/ts-host/node_modules` exists after update.
- Verified `aether host plan --depth fast --planning-depth light` emits a plan-only manifest with dispatches.
- Verified `aether host colonize --force-resurvey` emits a plan-only colonize manifest with dispatches.
- Verified `aether ceremony spawn-plan --workflow plan --manifest-file host-plan.json` produces non-empty visual ceremony output.

Verification matrix:

- `go test ./... -count=1`: pass.
- `go test ./... -race -count=1`: pass.
- `npm --prefix .aether/ts-host run typecheck`: pass.
- `npm --prefix .aether/ts-host run build`: pass.
- `npm --prefix .aether/ts-host test`: pass, 286 tests.
- `git diff --check`: pass.
- `aether source-check`: pass.
- `aether porter check`: 9/10 pass; blocked only by dirty git status.

## Release Blockers Found And Fixed

1. `aether update --force` could fail fresh installs because `npm ci --prefix`
   was run against the TypeScript host package in a way that confused npm's
   lockfile validation. The fix runs `npm ci` and `npm run build` with
   `cmd.Dir` set to `.aether/ts-host`.

2. Host commands could emit no output from a fresh temp repo on macOS because
   the TypeScript host main-module check compared a symlink path with a realpath
   (`/var/...` versus `/private/var/...`). The fix normalizes entrypoint paths
   through `realpathSync`, with a regression test for symlinked entrypoints.

## Ship Commands

Use this path only after reviewing the dirty tree and committing intentionally:

```bash
git status --short
git add -A
git commit -m "Harden hybrid Aether release path"
aether porter check
aether porter check --full-release
```

If this should become the next public release after v1.0.39, bump to the next
explicit semver and commit the bump:

```bash
aether bump-version 1.0.40
git add .aether/version.json npm/package.json
git commit -m "Release v1.0.40"
git tag v1.0.40
git push origin HEAD
git push origin v1.0.40
```

Local stable hub publish for same-machine testing:

```bash
aether publish --channel stable --binary-dest "$HOME/.local/bin"
```

Downstream repo test path:

```bash
aether update --force
aether status
aether host plan --depth fast --planning-depth light
```

## Defer Path

If `aether porter check --full-release` fails after commit, defer public release
and start a targeted cleanup colony:

```bash
aether init "v1.0.40 Full Release Gate Cleanup"
aether discuss
aether plan
```
