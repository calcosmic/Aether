package cmd

import (
	"fmt"
	"strings"
)

// partial_work_label.go is the one standing vocabulary every subsystem's
// half-finished work resolves through (LIVE-05, LIVE-07, D-12, CAP-072).
//
// Oracle already solved "useful but unproven" honestly for its own research
// (oracleResearchPartialLabel/oracleResearchStandingLabel,
// cmd/oracle_research_doc.go, Phase 198.2). The failure mode this file exists
// to prevent is solving the same problem three more times -- once per
// subsystem, with three different words for the same idea -- which is
// exactly the drift 202-CLASSIC-SYNTHESIS.md's SYN-202-12 row names as this
// repository's own recurring cost. Swarm's unproven repair ideas
// (cmd/swarm_episode.go), Oracle's partial research, plan research artifacts,
// and local reflections all resolve their standing through workStandingLabel
// below -- nothing renders a standing phrase locally.

// workStanding is the one standing vocabulary. Exactly three values exist on
// purpose: a fourth would be a second word for a standing this vocabulary
// already names.
type workStanding string

const (
	// workStandingVerified is a finished, checked result.
	workStandingVerified workStanding = "verified"
	// workStandingUsefulNotes is content with no verification behind it yet
	// -- kept, never discarded, and never presented as a checked result.
	workStandingUsefulNotes workStanding = "useful notes"
	// workStandingUnknown is an item whose standing could not be determined
	// -- e.g. a record that failed to parse. It is NEVER promoted to
	// verified.
	workStandingUnknown workStanding = "standing unknown"
)

// declaredWorkStandings is the full declared set, read by tests that assert
// the count rather than typing the number in.
func declaredWorkStandings() []workStanding {
	return []workStanding{workStandingVerified, workStandingUsefulNotes, workStandingUnknown}
}

// workStandingLabel renders a standing into an owner-facing phrase in plain
// words. whatWouldVerify names, in ordinary words, what would raise a
// useful-notes item to verified; every useful-notes label includes it. Both
// Swarm (cmd/swarm_episode.go) and Oracle (cmd/oracle_research_doc.go)
// render every standing phrase through this one function -- neither renders
// a standing phrase locally, so the same input standing always produces the
// same label regardless of which subsystem it came from.
func workStandingLabel(standing workStanding, whatWouldVerify string) string {
	switch standing {
	case workStandingVerified:
		return "verified"
	case workStandingUsefulNotes:
		reason := strings.TrimSpace(whatWouldVerify)
		if reason == "" {
			reason = "being checked and confirmed"
		}
		return fmt.Sprintf("useful notes, not verified — would be verified by %s", reason)
	default:
		return "standing unknown"
	}
}

// resolveMalformedItemStanding is the fallback every source below uses when
// an item cannot be parsed cleanly: it always resolves to workStandingUnknown
// paired with the reason it could not be determined. It never resolves to
// verified -- a record this repository could not read is never presented as
// a checked result.
func resolveMalformedItemStanding(reason string) (workStanding, string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "the item could not be parsed"
	}
	return workStandingUnknown, reason
}
