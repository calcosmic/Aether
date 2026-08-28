package cmd

import (
	"fmt"
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
//
// Name is the deterministic per-worker name ("Mason-67") and it is the name
// the platform's own session title carries -- OpenCode writes titles shaped
// "🔨 Builder Mason-67: ...", and it is what the usage resolver matches on.
// AgentName is the agent DEFINITION the worker ran as ("aether-builder"),
// which several workers in one build share, so it cannot identify a row.
// It is only the fallback for a dispatch that never got a worker name.
func spendWorkerNameForDispatch(dispatch codexBuildDispatch) string {
	if name := strings.TrimSpace(dispatch.Name); name != "" {
		return name
	}
	return strings.TrimSpace(dispatch.AgentName)
}

// spendRunStartFromManifest reads the run's own start time from the dispatch
// manifest. A manifest that never recorded one yields the zero time, which the
// writer reports rather than replacing with a made-up window.
func spendRunStartFromManifest(generatedAt string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(generatedAt))
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
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
	outcome := spendWriteOutcome{}

	if req.StartedAt.IsZero() {
		outcome.Notes = append(outcome.Notes,
			"this run's start time was not recorded, so no worker's usage could be matched to it by time")
	}

	workerNames := make([]string, 0, len(req.Dispatches))
	for _, dispatch := range req.Dispatches {
		if name := spendWorkerNameForDispatch(dispatch); name != "" {
			workerNames = append(workerNames, name)
		}
	}

	resolution := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:    req.Platform,
		RepoRoot:    req.RepoRoot,
		StartedAt:   req.StartedAt,
		EndedAt:     req.EndedAt,
		WorkerNames: workerNames,
	})
	outcome.Notes = append(outcome.Notes, resolution.Diagnostics...)

	usageByWorker := make(map[string]wrapperWorkerUsage, len(resolution.Workers))
	for _, worker := range resolution.Workers {
		usageByWorker[worker.WorkerName] = worker
	}

	rows := make([]spendRow, 0, len(req.Dispatches))
	for _, dispatch := range req.Dispatches {
		name := spendWorkerNameForDispatch(dispatch)
		if name == "" {
			outcome.Notes = append(outcome.Notes,
				"one worker had no name at all, so its cost could not be filed against anything")
			continue
		}
		// A row whose status is outside the dispatch vocabulary cannot be
		// saved, and saveSpendLedger refuses the WHOLE ledger over one bad
		// row. Dropping that one row with a note is the lesser loss: the rest
		// of the run's accounting still gets filed, and nothing invents a
		// status the dispatch never stated.
		if _, ok := normalizeSpendRowStatus(dispatch.Status); !ok {
			outcome.Notes = append(outcome.Notes, fmt.Sprintf(
				"worker %s did not state an outcome this ledger recognises (%q), so its cost was left unfiled rather than recorded under a made-up one",
				name, dispatch.Status))
			continue
		}

		worker := usageByWorker[name]
		rows = append(rows, spendRow{
			AgentName: name,
			Caste:     dispatch.Caste,
			Task:      dispatch.Task,
			JobName:   dispatch.JobName,
			Status:    dispatch.Status,
			// When the platform reported nothing this is the zero value: no
			// columns, no source tag. That absence IS the record.
			Usage: worker.Usage,
		})
		if worker.Reported {
			outcome.Reported++
		}
	}

	if err := saveSpendLedger(spendLedger{
		Phase:      req.Phase,
		PhaseName:  req.PhaseName,
		Workflow:   req.Workflow,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Rows:       rows,
	}); err != nil {
		return outcome, err
	}

	outcome.RowsWritten = len(rows)
	return outcome, nil
}
