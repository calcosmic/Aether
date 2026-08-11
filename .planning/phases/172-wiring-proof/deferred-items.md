# Deferred Items — Phase 172

Out-of-scope discoveries found while executing this phase's plans. Not fixed
because they are unrelated to the task that found them (Scope Boundary rule).

## 172-00 (Task 1)

- **`.aether/docs/command-playbooks/continue-advance.md` — malformed fence
  closer near line 503.** The line reads `--ttl "30d"` immediately followed
  by a closing triple-backtick fence marker on the SAME line, with no
  newline between them (` --ttl "30d"` glued directly to the fence). Because
  `extractDocumentedCalls`/`collectDocumentedCalls` detect fence boundaries
  via `strings.HasPrefix(strings.TrimSpace(line), "```")`, a fence marker
  that isn't on its own line is invisible to the toggle, which desyncs
  in-fence/out-of-fence parity for the rest of the file. One concrete effect:
  a duplicate `midden_result=$(aether midden-recent-failures 50
  2>/dev/null || echo '{"count":0,"failures":[]}')` call at line 553 is
  never extracted or audited at all (fixed anyway for consistency with the
  four sibling `--limit`-flag fixes in the same commit, but the audit still
  cannot see it because of this fence bug).
  Unrelated to Task 1's extractor changes — this fence has been malformed
  independently of command-substitution recognition, and was already
  invisible to the pre-fix extractor too. Not fixed here because repairing
  it touches unrelated markdown formatting, not the tokenizer.

  **Closed by plan 172-06.** `continue-advance.md`'s glued `--ttl "30d"`
  marker (and thirteen others of the same shape across six sibling
  playbook files) is repaired: the marker now sits on its own line.
  `extractDocumentedCalls` also gained fence-marker tolerance so a future
  glued marker no longer desyncs fence parity, and
  `TestAuditedCorpusHasNoGluedFenceMarkers` fails, naming the file and
  line, if the glued shape ever returns anywhere in the audited corpus.

## 172-07 (both tasks)

- **`pkg/codex` `TestCodexReadOnlyProfileSelectsReadOnlySandbox` fails only
  under full-suite load, not in isolation.** Running `go test ./... -count=1
  -timeout 900s` twice in this worktree produced the same failure both
  times: `permission_profile_test.go:111: worker startup failed: codex
  login status failed: timed out; sensitive details omitted`. Running the
  same test alone (`go test ./pkg/codex -run
  TestCodexReadOnlyProfileSelectsReadOnlySandbox -count=1 -v`) passes in
  under a second. The test shells out to check `codex` CLI login status,
  which times out when run concurrently with the ~300s `cmd` package under
  this sandboxed worktree's resource/network constraints — not something
  this plan's changes could cause. `pkg/codex/permission_profile_test.go`
  is untouched by this plan (last touched in an unrelated 163-03 commit,
  confirmed via `git log`), and this plan's files (`cmd/ci_wiring_gate_test.go`,
  `cmd/subcommand_reachability_ratchet_test.go`,
  `.github/workflows/ci.yml`) are all in the `cmd` package, which was fully
  green (298-332s, 0 failures) in both full-suite runs. Not fixed here —
  out of scope per the Scope Boundary rule.
