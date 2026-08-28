# OpenCode spend fixture

Redacted, hand-written approximation of the on-disk layout observed on this
machine on 2026-08-13 (see 174-RESEARCH.md § "Pattern 2"). OpenCode's storage
layout is undocumented by the vendor (174-RESEARCH.md Assumption A3), so the
parser reading it must fail soft if the real layout changes.

`__REPO_ROOT__` in `project/prj_fixture.json` is rewritten by the test to its
temp repo root at run time, so worktree discovery is exercised for real
rather than stubbed. No real session ids, paths, or user content appear here.

This file is owned by plan 174-05 and is separate from `cmd/testdata/spend/README.md`
(plan 174-04's Claude Code fixture, owned independently).
