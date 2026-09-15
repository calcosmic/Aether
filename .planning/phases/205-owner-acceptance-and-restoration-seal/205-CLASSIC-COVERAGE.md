# Phase 205 Classic Coverage Audit

These two records connect the old system's documentation and platform-readiness
promises to files and checks that exist today. Each named check was run separately
and passed before its record was added. The JSON companion is the source checked
by the program; this table is its readable version.

## GOAL

| ID | Disposition | Modern home | Plan/task | Proof |
|---|---|---|---|---|
| CAP-023 | restore-modern | .aether/commands/seal.yaml and cmd/seal_wrapper_accuracy_test.go | 205-04 Tasks 1-3 | TestSealWrapperReviewClaimMatchesTheRuntime; TestSealWrapperTripletStaysIdentical; TestSealAndEntombWrappersRelayTheCard; TestEntombWrapperTripletStaysIdentical |
| CAP-054 | replace-better | pkg/codex/platform_contract.go | 205-10 Tasks 1-3 | TestOpenCodeCardClaimsResolveToTheCapturedRun; TestCodexCardClaimsResolveToTheCapturedRun; TestClaudeCardClaimsResolveToTheCapturedRun; TestEveryContractedPlatformHasACardAndARun; TestPlatformCardsDoNotClaimUngovernedAbilityAsGoverned; TestPlatformCardsPromiseNoFutureWork |

## What the evidence establishes

- **Documentation accuracy:** the finish-and-archive instructions describe the
  program's conditional review team and tell the assistant to show the program's
  output. The source and platform copies agree. Whether that output reaches the
  owner in a live chat still belongs to Journey 1, as recorded in
  `205-04-SUMMARY.md`.
- **Platform readiness:** each platform card cites a recorded run. Claude Code
  and Codex reached the status command; OpenCode stopped at a provider connection
  error. Those captures do not prove helper dispatch or an entire project
  lifecycle. `205-10-SUMMARY.md` and the versioned support contract retain those
  limits.

Both dispositions match the frozen ledger. Neither row needs a change to the
recorded disposition or an exception to the existing check.
