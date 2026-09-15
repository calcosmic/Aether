# Phase 205: Restoration Parity Record

This is PROOF-03's four-dimension parity ledger. PROOF-03 says a restored
piece of old ("Classic") behaviour only counts as genuinely brought back when
it can show real evidence on four separate things: that the owner gets a
useful result, that the mechanism genuinely runs (not just looks like it
runs), that the owner can understand what happened, and that the modern
program's safety rules still hold. A capability row ("CAP row" -- one
numbered entry in this project's master list of old features that needed a
home in the new program) is a "slice" here: one restored piece of behaviour
this record must carry a row for.

Each row below names one or more slices and, for each of the four
dimensions, either real evidence naming a public command a person can
actually run, or an honest "not evidenced" statement with the reason. A test
count, a percentage, or a combined score is never offered as completion
evidence anywhere in this document -- `go test ./cmd/ -run TestClassicParity`
is the command that checks this ledger holds together; it does not itself
prove any one piece of evidence is true.

Where two slices are genuinely delivered by the same mechanism and share the
same real evidence, they appear as one row naming both -- never as two rows
each independently crediting the same proof.

## Phase 203

### CAP-002, CAP-014

- **Outcome**: Evidenced: Every worker brief and the owner-facing pheromone display read FOCUS/REDIRECT/FEEDBACK signals through the same rule for what counts as "still active," so a signal the owner set once behaves the same wherever it is read back. | Path: aether pheromone-display
- **Behavior**: Evidenced: TestOneEffectivePheromonePredicate and TestEveryBriefReaderUsesTheResolver both pass against the single canonical resolver in cmd/pheromone_resolver.go (203-05 Task 3); the resolver is the only place strength/expiry/revocation logic lives. | Path: aether pheromone-display
- **Experience**: Evidenced: A note's live strength, decay, and status render in one formatted table an owner can read directly, rather than being separately inferred by each reader. | Path: aether pheromone-display
- **Safety**: Evidenced: A revoked or expired note is excluded by the same resolver every reader calls, so a stale or withdrawn note can never silently reappear in one reader while correctly excluded in another (SYN-203-10). | Path: aether pheromone-display

### CAP-009

- **Outcome**: Evidenced: The end-of-run family tree shows the owner who asked another helper for backup, whether the request was granted or refused, and what it cost -- a record drawn from the real recruitment and credit decision, not a guess. | Path: aether watch
- **Behavior**: Evidenced: TestAgencyEvidenceFromTrophallaxisDecision passes against cmd/agency_contract.go's real trophallaxis-decision/recruitment-credit join (203-12 Task 2) rather than a placeholder. | Path: aether watch
- **Experience**: Evidenced: The inline recruit/refusal lines and the closing family tree render through the caste-identity system Phase 202 already built, so a helper joining or being refused looks and reads like every other worker line. | Path: aether watch
- **Safety**: Evidenced: A refusal never stalls the requesting worker -- it continues its task alone and the command still reports success, never a failure. | Path: aether recruit

### CAP-030

- **Outcome**: Evidenced: A hard "never do this" REDIRECT note the owner set is honoured the same way during planning as everywhere else it is read, instead of planning silently using a looser rule that let a withdrawn note keep influencing plans. | Path: aether plan
- **Behavior**: Evidenced: TestActiveStrongExpiredExcludedByEveryReader confirms codex_plan.go's REDIRECT scan calls the same resolver as every other reader (203-05 Task 1), closing the exact defect 203-CLASSIC-SYNTHESIS.md names as its clearest single row. | Path: aether plan
- **Experience**: Evidenced: Planning output honours REDIRECT constraints the same way the owner already sees them rendered through the shared signal display. | Path: aether plan
- **Safety**: Evidenced: An expired or revoked REDIRECT note can no longer keep shaping plans after it should have stopped mattering -- the single-resolver rule applies to planning exactly as it applies to every other reader. | Path: aether plan

### CAP-031

- **Outcome**: Evidenced: The owner can permanently withdraw a "never do this" note with one command, and it stays withdrawn -- it cannot come back through any other path in the program. | Path: aether pheromone-display
- **Behavior**: Evidenced: TestRevokedNoteStaysOut proves the real BIO-08 revoke verb in cmd/pheromone_influence.go (203-11 Task 2) is a permanent state with no runtime path back, not a soft, reversible flag. | Path: aether pheromone-display
- **Experience**: Evidenced: A revoked note is shown as revoked in the same signal table every other note appears in, with a recorded reason and actor. | Path: aether pheromone-display
- **Safety**: Evidenced: No runtime path anywhere in the program can bring a revoked note back into effect; only the owner can revoke, and the revoke is written down. | Path: aether pheromone-display

### CAP-058

- **Outcome**: Evidenced: The owner reviews and dismisses a suggested note through one shared tick-to-approve queue, instead of four separate, narrower legacy helpers each doing a piece of the same job. | Path: aether suggest-approve
- **Behavior**: Evidenced: TestSuggestApprove_DismissSuggestion passes against the real, scope-narrowed dismiss verb in cmd/suggest_approve.go (203-08 Task 2); the other four named legacy helpers this phase considered stay out of this phase's boundary, recorded in 203-CLASSIC-SYNTHESIS.md's own CAP-058 routing note. | Path: aether suggest-approve
- **Experience**: Evidenced: A dismissal is confirmed back to the owner through the same command that showed the suggestion in the first place, with no second, unexplained surface. | Path: aether suggest-approve
- **Safety**: Evidenced: Dismissing a suggestion only ever affects that one suggestion's own record; the shared queue's other approve/reinforce/pin actions are unaffected. | Path: aether suggest-approve
