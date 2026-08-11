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
