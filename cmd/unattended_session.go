package cmd

import "os"

// unattendedEnvVar is the one environment variable a caller sets,
// explicitly and deliberately, to tell Aether that no person is present to
// answer a question during this session. Accepted only when its value is
// exactly "1" -- see sessionHasNoOneToAsk's doc comment for why this fact
// is opt-in rather than inferred.
const unattendedEnvVar = "AETHER_UNATTENDED"

// sessionHasNoOneToAsk reports whether the current process has been told,
// via unattendedEnvVar, that no person is present to answer a question a
// stop-and-confirm refusal would otherwise raise (D-01,
// 208-CONTEXT.md "Gap-closure decisions": "act when alone, ask when you're
// there").
//
// This is deliberately an opt-in variable a caller sets, never anything
// inferred -- two reasons, written down here so a future change does not
// quietly try to infer it instead:
//
//  1. No terminal check can tell the two cases apart. A command run from a
//     chat's own shell tool (the Bash tool an AI assistant uses) is never
//     attached to a terminal either way -- an interactive chat's Bash calls
//     look identical, at the process level, to an unattended `claude -p`
//     chain's.
//  2. No Claude Code signal has been observed by this repository to
//     distinguish an unattended run from an interactive one. Acting on an
//     unobserved signal would be a guess dressed up as a fact, and D-01
//     requires an observable fact.
//
// Anything other than the one accepted value "1" -- including the variable
// being unset entirely -- means a person is present. That is the fail-safe
// direction on purpose: the worst an unset variable can do is make a chat
// ask a question nobody answers (annoying, recoverable); a wrong "nobody is
// here" reading could replace an owner's real work in a real project
// without ever checking first (D-01 rejects "always act" for exactly this
// reason).
func sessionHasNoOneToAsk() bool {
	// Deliberately spelled out as a literal here (not via unattendedEnvVar)
	// so this file's own os.Getenv("AETHER_UNATTENDED") call is the one
	// this repository ratchets against: TestTheIsAnyoneHereFactHasOneReader
	// (cmd/unattended_refusal_test.go) greps for that exact call shape and
	// fails naming any other non-test file under cmd/ that also performs
	// it directly, rather than going through this function.
	return os.Getenv("AETHER_UNATTENDED") == "1"
}

// unattendedGuidanceSentence is the one sentence both refusal lanes
// (Error()'s one-line form and renderRefusal's drawn block, cmd/refusal.go)
// carry when sessionHasNoOneToAsk is true and the refusal both protects
// work and names a next command -- one string, read by both lanes, so
// neither can drift from the other's wording.
const unattendedGuidanceSentence = "No one is here to answer, so run the command this refusal names and carry on; do not ask first."
