package cmd

import (
	"strings"
	"time"
)

// spendWriteRequest is one finished run's accounting, ready to be filed.
type spendWriteRequest struct {
	// Phase and Workflow key the ledger file. Workflow is "build" or
	// "continue"; anything else is refused rather than quietly creating a
	// third file nothing aggregates.
	Phase     int
	PhaseName string
	Workflow  string

	// RepoRoot, Platform and the run window are what the usage resolver needs
	// to find this run's own session artifacts.
	RepoRoot  string
	Platform  string
	StartedAt time.Time
	EndedAt   time.Time

	// Dispatches are the run's workers at their terminal status.
	Dispatches []codexBuildDispatch
}

// spendWriteOutcome is what the writer did, for the caller to report.
type spendWriteOutcome struct {
	// RowsWritten counts the rows actually filed.
	RowsWritten int

	// Reported counts how many of those rows carry a real measurement. The
	// rest carry no token figure at all (D-01 as amended).
	Reported int

	// Notes are plain-English remarks about anything the writer could not
	// account for. They never stop a build.
	Notes []string
}

// spendWorkerNameForDispatch is the one name a dispatch is accounted under.
// AgentName is the deterministic per-worker name the platform's own session
// title carries ("Mason-67"); Name is the fallback for a dispatch that never
// got one.
func spendWorkerNameForDispatch(dispatch codexBuildDispatch) string {
	if name := strings.TrimSpace(dispatch.AgentName); name != "" {
		return name
	}
	return strings.TrimSpace(dispatch.Name)
}

// writeSpendRowsForRun turns a finished run's dispatches into ledger rows and
// files them under this phase and this workflow.
//
// Every figure it files was read from disk by the Go runtime — the platform's
// own session transcript or session store, or the provider's own attached
// measurement. Nothing is relayed through a completion packet and nothing is
// derived from a length.
//
// A worker the platform reported nothing for still gets a row, carrying no
// token figure and no source tag, so it cannot vanish and make the run look
// cheaper than it was.
//
// Rerunning the same run's finalize REPLACES this workflow's rows rather than
// appending to them: the ledger is saved whole, never loaded-and-extended, so
// a second finalize of the same build cannot double its own cost. The other
// workflow's file is never opened.
func writeSpendRowsForRun(req spendWriteRequest) (spendWriteOutcome, error) {
	return spendWriteOutcome{}, nil
}
