package cmd

import "strings"

// refusalSelfRecoveryRow is one opt-in entry in refusalSelfRecoveryTable.
// Both fields are printed VERBATIM to the owner
// (renderRefusalSelfRecoveryNotice below) and must each stay a short,
// plain-English clause an owner can read with no file open -- nothing else.
// A developer's rationale or a planning-decision citation belongs in a Go
// comment beside the row, never inside either string: 208-13-PLAN.md
// (WR-03, 208-REVIEW-GAP2.md) found a planning-decision id and a
// `.planning/` filename embedded in this string and printed straight to the
// owner's screen. A future row must not reintroduce that by accident.
//
//   - Reason answers "why is Aether going ahead instead of stopping to
//     ask" -- the clause the "question" line prints.
//   - Action answers "what is Aether going ahead and doing" -- the clause
//     the closing "next" line prints. Before 208-18-PLAN.md (WR-03 /
//     208-REVIEW-GAP3.md) this clause was a fixed phrase baked into
//     renderRefusalSelfRecoveryNotice ("rebuilding the map of your code"),
//     so a future second row would have silently kept that same wording no
//     matter what its own recovery actually did. No clause on this screen
//     is written into the screen any more -- a future row must supply its
//     own accurate Action, and refusalSelfRecoveryContractProblems (this
//     package's test file) refuses to accept one that does not.
type refusalSelfRecoveryRow struct {
	Reason string
	Action string
}

// refusalSelfRecoveryTable is the checked-in, opt-in list of refusals Aether
// is allowed to recover from itself when nobody is present to answer
// (sessionHasNoOneToAsk, cmd/unattended_session.go). It holds exactly one
// entry this round -- the colonize existing-survey stop.
//
// Developer rationale for the one row below (not printed to the owner):
// nobody is present to ask, the refusal's own next command is safe for
// Aether to run on its own, and the alternative -- printing an instruction
// and hoping an unattended chat follows it -- is a rehearsal that can never
// finish (owner ruling D-03, recorded in this phase's second gap-closure
// round).
//
// The finalize sibling ("colonize-finalize-existing-survey-found")
// deliberately stays out. Its own NextCommand
// ("aether colonize --plan-only --force-resurvey") names work a chat
// performs -- dispatching surveyor workers and posting a new completion
// packet -- not work this runtime process can perform for itself at the
// moment that refusal fires. Do not add it here by symmetry with the row
// above; a future reader who wants to recover it needs a mechanism that can
// actually dispatch workers from inside colonize-finalize, which does not
// exist yet.
var refusalSelfRecoveryTable = map[string]refusalSelfRecoveryRow{
	"colonize-existing-survey-found": {
		Reason: "nobody is here to answer, and rebuilding the map of your code is something " +
			"Aether can safely do on its own rather than leaving the project stuck waiting for a reply.",
		Action: "rebuilding the map of your code",
	},
}

// attemptRefusalSelfRecovery is the ONE place this repository decides
// whether Aether carries out a refusal's own recovery itself rather than
// stopping and waiting for an owner to answer. Every call site that reaches
// an existing-survey refusal must offer it to this function rather than
// re-deriving the same conditions for itself (TestSelfRecoveryHasOneDecision,
// 208-11-PLAN.md Task 2, widened by 208-13-PLAN.md Task 1).
//
// It returns false -- changing nothing, printing nothing, recording nothing
// -- unless all of these hold:
//
//  1. sessionHasNoOneToAsk() is true (cmd/unattended_session.go) -- read
//     only through that one function, never by reaching the environment
//     variable directly here.
//  2. r.ID is a key in refusalSelfRecoveryTable above.
//  3. r.ProtectsWork is true (the caller's own copy) -- a refusal that does
//     not protect work is already a "warn" row (warnAndCarryOn,
//     cmd/refusal.go) and has no business being offered here at all.
//  4. strings.TrimSpace(r.NextCommand) is non-empty (the caller's own copy)
//     -- there must be an actual command to run on the owner's behalf.
//  5. The AUTHORITATIVE registered row for r.ID (read back through
//     refusalForID, never trusted from the caller's own struct a second
//     time) also protects work and also names a non-empty command
//     (208-13-PLAN.md Task 1, WR-08 / 208-REVIEW-GAP2.md). This is what
//     makes the build-time contract checker
//     (refusalSelfRecoveryContractProblems, in this package's test file) and
//     this runtime gate agree on one source instead of two that could
//     silently drift apart -- and it is also why the command named in the
//     notice below is read from this registered row, not from the caller's
//     own field.
//
// These are deliberately not joined by any freshness or staleness check of
// their own. The refusal itself consults no such classification before
// firing -- surveyDocsExist alone decides whether
// "colonize-existing-survey-found" fires -- and adding a second condition
// here would make the exact same refusal behave two different ways for
// reasons the refusal itself never states.
//
// When it returns true, it has already done both of the things that make
// the recovery visible and auditable, in this order: announced the recovery
// to a person through emitVisualProgress (cmd/codex_visuals.go), which is
// silent in machine-output mode -- this is the link that keeps a host's JSON
// parse of a plan-only manifest intact; warnAndCarryOn's own stdout write is
// a different link and must not be reused here, because it writes
// unconditionally rather than through the machine-output gate -- and
// appended a refusal-log record marked recovered rather than refused
// (appendRecoveredRefusalToLog, cmd/refusal_log.go). A caller that receives
// true from this function should fall through into the recovery path
// unconditionally; there is nothing left for it to announce or record
// itself.
func attemptRefusalSelfRecovery(r refusal) bool {
	if !sessionHasNoOneToAsk() {
		return false
	}
	entry, listed := refusalSelfRecoveryTable[r.ID]
	if !listed {
		return false
	}
	if !r.ProtectsWork {
		return false
	}
	if strings.TrimSpace(r.NextCommand) == "" {
		return false
	}
	row, ok := refusalForID(r.ID)
	if !ok || !row.ProtectsWork || strings.TrimSpace(row.NextCommand) == "" {
		return false
	}
	emitVisualProgress(renderRefusalSelfRecoveryNotice(r, entry.Reason, entry.Action, row.NextCommand))
	appendRecoveredRefusalToLog(r)
	return true
}

// renderRefusalSelfRecoveryNotice draws the person-facing notice for a
// self-recovered refusal, through the same banner/divider/voiceLine
// machinery every other Aether screen uses rather than a bare line. It
// names, in order: what Aether found, that nobody was there to ask, and that
// Aether is going ahead instead of stopping to ask.
//
// It deliberately never claims, in the past tense, that the named command
// has already run (208-13-PLAN.md Task 2, WR-02 / 208-REVIEW-GAP2.md): on
// colonize's plan-only lane, at the moment this notice prints, nothing has
// actually re-surveyed the project yet -- runCodexColonizePlanOnly only sets
// ForceResurvey and goes on to build a manifest a host will later dispatch
// surveyors from. The command name (next) is kept visible as supporting
// evidence, not as the claim itself, and is read from the registered refusal
// row (attemptRefusalSelfRecovery's own authoritative lookup) rather than a
// caller-supplied copy.
//
// action (208-18-PLAN.md Task 2, WR-03 / 208-REVIEW-GAP3.md) is the row's
// own description of what Aether is going ahead and doing, composed into
// the closing "next" line exactly as before -- this clause used to be a
// fixed phrase baked into this function, so a second table row would have
// silently kept the first row's wording regardless of what it actually did.
// It is now supplied by the caller (attemptRefusalSelfRecovery, reading it
// off the table row that fired), and every clause on this screen traces
// back to that row.
func renderRefusalSelfRecoveryNotice(r refusal, reason string, action string, next string) string {
	next = strings.TrimSpace(next)
	action = strings.TrimSpace(action)
	var b strings.Builder
	b.WriteString(renderBanner("🙅", "Carrying On Without You"))
	b.WriteString(visualDividerStr())
	b.WriteString(voiceLine("blocked", strings.TrimSpace(r.What)))
	b.WriteString("\n")
	b.WriteString(voiceLine("question", "No one is here to answer, so Aether is going ahead on its own: "+reason))
	b.WriteString("\n")
	b.WriteString(voiceLine("next", "Aether is going ahead and "+action+" instead of stopping to ask (`"+next+"`)."))
	b.WriteString("\n")
	return b.String()
}
