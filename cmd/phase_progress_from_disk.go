package cmd

import (
	"github.com/calcosmic/Aether/pkg/colony"
)

// phaseProgressRankOrder is the ONE ordered least-to-most-finished status
// scale resolvePhaseProgressFromDisk ranks both records against -- not
// started, in progress, built but unverified, verified, complete. It is
// package-level and singular by design, mirroring the "one ranking rule"
// discipline TestStrongestHabitsHaveOneRankingRule already enforces for
// habit selection elsewhere in this program (198.2 plan 03). A structural
// test, TestPhaseProgressRankOrderHasOneSource, fails by name if a second
// such scale is ever introduced anywhere in cmd/*.go.
var phaseProgressRankOrder = []string{
	"not started",
	"in progress",
	"built but unverified",
	"verified",
	"complete",
}

// phaseProgressRankIndex returns status's position in phaseProgressRankOrder
// (0 = least finished), or -1 for a status outside the vocabulary.
func phaseProgressRankIndex(status string) int {
	for i, s := range phaseProgressRankOrder {
		if s == status {
			return i
		}
	}
	return -1
}

// phaseProgressFromDisk is the resolved answer to "how finished is this
// phase, really": Status is always one of phaseProgressRankOrder's five
// plain-English phrases, TasksTotal/TasksDone count from whichever record
// Source names, and Disagreed is true when the stored phase and the
// durable check record ranked differently -- in which case Status is
// always the LESS finished of the two (208-04 must_have: "the less-finished
// of the two is what the owner is shown").
type phaseProgressFromDisk struct {
	Status     string
	TasksTotal int
	TasksDone  int
	Source     string
	Disagreed  bool
}

// phaseProgressRankFromStoredPhase ranks phase using only what is stored on
// the phase itself: its own Status field plus its own Tasks' Status
// values. It never reads any other file.
func phaseProgressRankFromStoredPhase(phase colony.Phase) string {
	tasksTotal := len(phase.Tasks)
	tasksDone := 0
	for _, task := range phase.Tasks {
		if task.Status == colony.TaskCompleted {
			tasksDone++
		}
	}
	switch phase.Status {
	case colony.PhaseCompleted:
		return "complete"
	case colony.PhaseInProgress:
		if tasksTotal > 0 && tasksDone == tasksTotal {
			return "built but unverified"
		}
		return "in progress"
	default: // pending, ready, or an unrecognised value
		return "not started"
	}
}

func countStoredPhaseTasks(phase colony.Phase) (total, done int) {
	total = len(phase.Tasks)
	for _, task := range phase.Tasks {
		if task.Status == colony.TaskCompleted {
			done++
		}
	}
	return total, done
}

// phaseProgressRankFromCheckRecord ranks the durable check record
// (build/phase-<id>/continue.json) the same way -- Completed outranks
// Advanced, which outranks how many of the check's own recorded tasks it
// marked Verified.
func phaseProgressRankFromCheckRecord(report codexContinueReport) string {
	if report.Completed {
		return "complete"
	}
	if report.Advanced {
		return "verified"
	}
	total, done := countCheckRecordTasks(report)
	if total > 0 && done == total {
		return "built but unverified"
	}
	if done > 0 {
		return "in progress"
	}
	return "not started"
}

func countCheckRecordTasks(report codexContinueReport) (total, done int) {
	total = len(report.Tasks)
	for _, task := range report.Tasks {
		if task.Verified {
			done++
		}
	}
	return total, done
}

// resolvePhaseProgressFromDisk works out how finished phase really is from
// two independently-recorded sources on disk -- the stored phase (its own
// Status field plus its own Tasks) and the durable check record at
// build/phase-<id>/continue.json (Advanced/Completed/Tasks) -- and returns
// the LESS finished of the two whenever they disagree. It opens no file
// outside the store, writes nothing, and returns the stored phase's own
// answer unchanged when no check record exists yet or the store is
// unavailable (a phase never checked is not "disagreeing" with anything).
func resolvePhaseProgressFromDisk(phase colony.Phase) phaseProgressFromDisk {
	storedRank := phaseProgressRankFromStoredPhase(phase)
	storedTasksTotal, storedTasksDone := countStoredPhaseTasks(phase)
	stored := phaseProgressFromDisk{
		Status:     storedRank,
		TasksTotal: storedTasksTotal,
		TasksDone:  storedTasksDone,
		Source:     "the saved project record",
	}

	if store == nil {
		return stored
	}
	var report codexContinueReport
	if err := store.LoadJSON(continuePlanArtifactsPath(phase.ID, "continue.json"), &report); err != nil {
		return stored
	}

	checkRank := phaseProgressRankFromCheckRecord(report)
	storedIdx := phaseProgressRankIndex(storedRank)
	checkIdx := phaseProgressRankIndex(checkRank)
	if checkIdx < 0 || checkIdx == storedIdx {
		return stored
	}
	if checkIdx > storedIdx {
		// The stored phase is the less-finished record; keep it, but say
		// the two disagreed.
		stored.Disagreed = true
		return stored
	}
	checkTasksTotal, checkTasksDone := countCheckRecordTasks(report)
	return phaseProgressFromDisk{
		Status:     checkRank,
		TasksTotal: checkTasksTotal,
		TasksDone:  checkTasksDone,
		Source:     "the last check's own record",
		Disagreed:  true,
	}
}
