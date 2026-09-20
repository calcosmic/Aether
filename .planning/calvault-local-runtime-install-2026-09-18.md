# CalVault runtime repair installed

Installed 2026-09-18T18:39:03.758896+00:00 as a binary-only local hotfix. CalVault now resolves
`/Users/callumcowie/.local/bin/aether` to the repaired executable. The version label stays **1.0.81** to
match the unchanged installed hub; the hash and embedded Git revision identify
this repair unambiguously.

- Previous SHA-256: `b37f56f6f5c9800483df8e6ea702b5002c9bee7eaff0300bc02830b8b96479f3`
- Installed SHA-256: `45663b7cc4ca018b5024ad78675f4255f894143b820484a8c2846c8270a7f816`
- Installed source revision: `6c615cf9d3856920abd4a22eb201639d010f9929` (`vcs.modified=false`).
- Source: `/Users/callumcowie/.aether-backups/calvault-field-hotfix-7475db1bda7c/source`.
- Stable baseline: `b37b22e39e012215a32423cf61354f2fb8f7ac20`; CalVault fix commit `5f0ee934e7963d5d27e15536ec5147388d8ebcc0`
  was cherry-picked as `5b37bf18`. The final commit makes its review-contract test
  independent of the unfinished native qualification test helper.
- Receipt: `/Users/callumcowie/.aether-backups/calvault-field-hotfix-7475db1bda7c/installation.json`.
- Previous executable backup: `/Users/callumcowie/.aether-backups/calvault-field-hotfix-7475db1bda7c/rollback/aether`.

## Validation and scope

The exact isolated candidate passed the focused command/producer regression
selection normally (cmd 20.882s; pkg/codex 2.169s) and with race detection
(cmd 24.133s; pkg/codex 3.771s). It includes the earlier installed bookkeeping
regressions. The initial stable test compile failed due to a test-only dependency
on nativeGapNoSchemaLoader; the local self-contained refusing loader fixed that.
Both initial failure and successful retry logs remain in the evidence directory.
The small test repair is also present in the main checkout for integration.

Candidate source integrity passed 5/5 checks. Installed version/hub agreement
and consumer integrity passed from CalVault. The replacement used a verified
adjacent temporary executable and atomic rename, with rollback on smoke failure.
All 1,092 recorded existing CalVault colony files and 747 hub files retained their
hashes. No hub or platform-home publish/update was needed for the Go-only fix.

This is focused qualification of a local repair, not a public release or a
passing full release suite. The full suite has not passed. No Phase 204.2 proof
credit, Phase 205 acceptance, CalVault phase retry, migration approval, or manifest
promotion follows from installing it. Already running processes retain their old
executable; new invocations at the recorded path use the repaired one.
