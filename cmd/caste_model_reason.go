package cmd

// Model reasons (Phase 196, D-02 — 196-CONTEXT.md).
//
// Every role kept on the expensive model carries a written reason, and that
// reason is shown to the owner on the pre-build team card before anything is
// spawned. Nothing here selects or overrides a model: automatic model
// routing was declined by the owner on 2026-07-28 and stays out of scope.
// The platform routes each role natively from the single `model:` line in
// `.claude/agents/ant/aether-<role>.md`; this file only explains a choice
// already made there.
//
// Writing rule: each sentence is read by someone who has never opened a file
// in this repository. So it says what the role actually judges and what a
// cheaper model would get wrong — never the role's own name as though its
// meaning were obvious, never a word this repository invented, never a file
// path. TestOpusRequiresRecordedReason enforces all of that, and fails by
// name if an expensive role has no reason or a reason outlives its role
// moving to the cheaper model.

// casteModelReasons maps every role on the expensive model to the one plain
// sentence justifying that expense. A role on the cheaper model has no
// entry — there is nothing to justify, and an entry for one fails the test
// in the other direction so the table cannot go stale either way.
var casteModelReasons = map[string]string{
	"archaeologist": "reads years of change history to work out why something was done the way it was; a cheaper model reports what changed and misses the why, which is the only part worth having",
	"architect":     "decides how the parts of a system fit together before any code exists, where a wrong choice is expensive to undo months later; a cheaper model produces a plausible shape that falls apart under the first real requirement",
	"auditor":       "judges whether finished code is genuinely good rather than merely working, which takes taste and a sense of what will hurt later; a cheaper model checks the obvious rules and waves through work a careful reviewer would send back",
	"gatekeeper":    "hunts for security holes — leaked passwords, a missing permission check — where one miss is the whole point of the check; a cheaper model finds the textbook cases and misses the ones that matter",
	"measurer":      "works out why something is slow, which means reasoning about what the machine really does rather than reading the code as written; a cheaper model guesses at the bottleneck and usually blames the wrong thing",
	"oracle":        "does open-ended research where the question itself has to be refined as answers arrive; a cheaper model answers the question as first asked and stops, which is how research goes confidently nowhere",
	"queen":         "decides which helpers to send at a piece of work and in what order, and every later cost follows from that one judgement; a cheaper model sends a safe, expensive crowd instead of the two people the job needs",
	"route_setter":  "breaks a large goal into an ordered plan, where a badly cut plan wastes every hour later spent executing it; a cheaper model produces steps that read well and cannot actually be built in that order",
	"sage":          "draws the general lesson out of what went right and wrong so it is not relearned next time; a cheaper model restates what happened instead of finding the lesson inside it",
	"tracker":       "chases a fault to its actual cause rather than the place it happened to surface; a cheaper model fixes the symptom and the same failure returns in a different shape",
}

// casteModelReason returns the written reason a role is kept on the
// expensive model, or "" for a role on the cheaper model (which needs no
// justification). The pre-build team card is its only consumer.
func casteModelReason(caste string) string {
	return casteModelReasons[normalizeCasteKey(caste)]
}
