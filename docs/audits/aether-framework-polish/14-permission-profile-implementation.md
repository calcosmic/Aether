# Typed Permission Profile Implementation

Date: 2026-07-22

Status: implemented and locally verified against the full regression and staged
release-candidate matrix. No public release was published.

## Verdict

Aether now has one Go-owned permission decision per worker. A manifest cannot
silently broaden Scout into a writer, Claude no longer bypasses its permission
system, OpenCode no longer enters through the unrestricted default build agent,
and Codex no longer gives every caste the same sandbox.

For beginners: the worker's badge now comes with a real key issued by the engine.
Changing the words on the badge does not create a bigger key.

## Implemented Contract

`pkg/codex/permission_profile.go` defines the versioned request and selected-host
decision. Go recomputes the canonical profile from the caste. Adapter requests
with missing, stale, or broader enforcement fields fail before provider launch.

Current release profiles:

| Profile | Castes | Enforced behavior |
| --- | --- | --- |
| `repository_read_only` | Scout, Includer | Repository writes denied |
| `workspace_write` | All other current castes | Persistent writes confined to the project workspace |
| `scoped_write` | none | Recognized and rejected |
| `test_write` | none | Recognized and rejected |

The request also types shell, network, approval, and declared write scopes.
Current shell execution remains inside the selected filesystem boundary, approval
is `never`, and network is explicitly `provider_default`, not an Aether guarantee.

Probe test-only, survey-only, Chronicler documentation-only, and reviewer
ledger-only restrictions are emitted as `behavioral_restrictions`. The host still
enforces the workspace boundary, but Aether does not pretend those narrower paths
are sandboxed.

## Platform Enforcement

### Codex

- `repository_read_only` selects `--sandbox read-only`.
- `workspace_write` selects `--sandbox workspace-write`.
- `--ask-for-approval never` remains explicit.
- `CODEX_HOME` is no longer added as an extra writable directory.

### Claude Code

- Read-only workers use `--permission-mode plan`.
- Workspace writers use `--permission-mode acceptEdits` plus a temporary settings
  file enabling native sandboxing.
- `failIfUnavailable` is true and `allowUnsandboxedCommands` is false.
- `bypassPermissions` and redundant `--add-dir` widening were removed.

### OpenCode

- `.opencode/agents/aether-worker-router.md` is a primary infrastructure agent,
  not a new caste. It can use Task but cannot edit or run Bash.
- The Go adapter rejects any alternate primary-agent override.
- All 27 caste agents explicitly deny `external_directory`.
- Go parses and attests the selected target definition before launch. Scout and
  Includer must also explicitly deny write, edit, and Bash.
- OpenCode therefore ships 28 agent assets: 27 castes plus one router.

## Data Flow

```text
Go manifest caste
  -> canonical permission_profile
  -> TS/native wrapper preserves profile
  -> hidden Go adapter validates exact enforcement fields
  -> selected platform permission decision
  -> provider-specific sandbox/tool flags
  -> decision returned with terminal worker result
```

The TS host contains a minimal dynamic-child fallback, but Go remains
authoritative and rejects stale or broadened values.

## Evidence

| Invariant | Test |
| --- | --- |
| Caste maps to canonical profile | `pkg/codex/permission_profile_test.go` |
| Scout cannot request Builder access | profile mismatch and adapter tests |
| Unsupported narrow profiles fail closed | permission decision matrix |
| Codex Scout uses read-only sandbox | `TestCodexReadOnlyProfileSelectsReadOnlySandbox` |
| Claude sandbox cannot silently disable | `TestClaudeWorkspaceSettingsFailClosed` |
| Shipped OpenCode definitions enforce declared boundary | `TestShippedOpenCodeAgentsDeclarePermissionBoundary` |
| Build and plan manifests carry profiles | `cmd/permission_profile_integration_test.go` |
| TS forwards the profile to Go | `.aether/ts-host/test/worker-dispatch.test.ts` |

The first complete Go run also exposed stale parity fixtures that treated every
OpenCode agent file as a caste and a black-box adapter request that omitted the
new required profile. The fixtures now verify 27 cross-platform castes plus the
named OpenCode-only router, and the adapter fixture sends the canonical Builder
profile. Medic applies the same distinction when checking repository health.

## Verification

| Check | Result |
| --- | --- |
| `go test ./... -count=1` | Pass; `cmd` 232.947s |
| `go test ./... -race -count=1` | Pass; `cmd` 305.914s |
| `go vet ./...` | Pass |
| Native build and `version` smoke | Pass; `1.0.42` |
| Windows amd64 cross-build | Pass |
| TS host typecheck/build/tests | Pass; 514 tests |
| Retired control typecheck/tests | Pass; 133 tests |
| Protected colony-state hash after tests | Unchanged: `36182c8f...c39d5bf` |
| npm bootstrap tests/package dry-run | Pass; 11 tests, four intended files |
| `goreleaser check` | Pass |
| GoReleaser snapshot release | Pass; six archives plus checksums |
| Packed npm installs staged GoReleaser archive | Pass |
| `git diff --check` | Pass |

## Remaining Limitations

- Network policy is provider-managed.
- Narrow paths inside the workspace are not host-enforced.
- OpenCode retains access to its provider-owned temporary output directory.
- The supported TypeScript-host and native Go build paths are now journal-bound.
  Any advanced platform-native Task/subagent mode outside that adapter path remains
  behavioral and must not claim equivalent run recovery.
- Live authenticated provider smokes remain manual release evidence.

## Next Checkpoint

Prove iterative planning and research through real Scout, Oracle, and Route-Setter
provider execution. Preserve the current plan-revision store and add only the
typed assumption/decision impact links required by that journey.
