# Same-host routing dev hotfix delivered

Installed locally as `aether-dev`, version `1.0.80-dev.routing.1`, with hub `~/.aether-dev`. This is a narrow hotfix based on the exact source revision of the installed stable binary, plus routing tests/fixes, aligned guidance and unique dev version metadata. It excludes later unfinished Phase204.2 work. No public release or stable upgrade occurred.

Codex sessions select Codex workers, Claude sessions select Claude workers, and OpenCode sessions select OpenCode workers. A matching runtime that is unavailable stops dispatch. Explicit provider overrides remain supported. Empty provider environment variables no longer establish a host. Missing phase completion evidence remains a separate runtime requirement.

## Testing in another repository

Run `aether-dev update --force`, then use direct `aether-dev` lifecycle commands such as `aether-dev continue`. New repositories first use `aether-dev lay-eggs`. Use the dev executable explicitly: existing global ant skills and slash commands may still invoke stable `aether`. Dev publish does not replace those global surfaces.

## Verification

- Full `pkg/codex` tests passed normally and with race detection on the isolated candidate.
- Focused command-guide/docs/shim tests passed.
- Disposable publish, binary/hub version agreement, setup and update passed.
- Installed dev binary/hub version agreement and a second disposable consumer setup/update passed.
- Installed release integrity passed all five checks.
- 2,996 recorded stable binary/hub/platform files remained byte-identical.

The first disposable publish could not use an uncached npm tarball offline; retry with an isolated cache downloaded the pinned dependencies and passed. Its online npm audit reported one high-severity finding in the unchanged dependency tree; the later offline audit's zero is not evidence that the finding was resolved. No dependencies were upgraded. Raw output is retained.

## Identity and rollback

See `routing-hotfix-2026-09-18.json` for exact source and executable hashes, source branch, raw evidence and rollback paths. The previous dev binary and full dev hub were copied to the recorded rollback directory before publication. The stable executable remains version1.0.79. Phase204.2 qualification is still blocked and no ANT requirement was marked complete by this hotfix.
