package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// sharedFileTasksTooBigForOneBrief builds tasks that are grouped ONLY because
// they all edit the same implementation file -- no dependency edges at all --
// and whose combined text is far past what one worker brief may carry, so the
// planner has to split them.
func sharedFileTasksTooBigForOneBrief(count int) (colony.Phase, []coherentJobTask) {
	sharedFile := "cmd/shared_target.go"
	filler := strings.Repeat("rewrite the shared target file carefully and completely. ", 60)

	tasks := make([]colony.Task, 0, count)
	seeds := make([]coherentJobTask, 0, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("1.%d", i+1)
		idCopy := id
		task := colony.Task{
			ID:     &idCopy,
			Goal:   fmt.Sprintf("Step %s: %s", id, filler),
			Status: colony.TaskPending,
			Hints:  []string{sharedFile},
		}
		tasks = append(tasks, task)
		seeds = append(seeds, coherentJobTask{
			Task:          task,
			ID:            id,
			TaskIndex:     i,
			Caste:         "builder",
			DeclaredPaths: []string{sharedFile},
		})
	}
	return colony.Phase{ID: 1, Name: "One file, many big steps", Tasks: tasks}, seeds
}

// TestSplitJobsNeverShareAFileInTheSameWave is the permanent regression lock for
// the 195 review's seventh warning (WR-07).
//
// Tasks that share an implementation file with no dependency between them are
// grouped into one job precisely so one worker owns that file. When the group
// is too big for a single worker brief the planner splits it into parts -- but
// the split parts had no ordering edge between them, so they landed in the same
// round while still both declaring the file they were grouped over. In isolated
// mode the runtime then refused the whole build, telling the owner to "declare
// disjoint paths or move the work into different waves" for a clash the runtime
// itself had just created; in shared mode two workers wrote the same file at
// once.
func TestSplitJobsNeverShareAFileInTheSameWave(t *testing.T) {
	phase, seeds := sharedFileTasksTooBigForOneBrief(8)

	plan, err := planCoherentJobs(phase, seeds, nil)
	if err != nil {
		t.Fatalf("planCoherentJobs: %v", err)
	}
	if len(plan.Jobs) < 2 {
		t.Fatalf("fixture no longer splits: %d job(s) planned, so this test proves nothing", len(plan.Jobs))
	}

	pathsByJob := map[string]map[string]struct{}{}
	for _, job := range plan.Jobs {
		set := map[string]struct{}{}
		for _, seed := range job.Tasks {
			for _, p := range seed.DeclaredPaths {
				set[p] = struct{}{}
			}
		}
		pathsByJob[job.Name] = set
	}

	for waveIndex, wave := range plan.Waves {
		for i := 0; i < len(wave); i++ {
			for j := i + 1; j < len(wave); j++ {
				for p := range pathsByJob[wave[i]] {
					if _, clash := pathsByJob[wave[j]][p]; clash {
						t.Fatalf("in round %d, %s and %s both claim %s -- they were split apart by the runtime and then scheduled to run at the same time on the very file they were grouped over", waveIndex+1, wave[i], wave[j], p)
					}
				}
			}
		}
	}

	// Every task must still be planned exactly once, and the split parts must
	// stay in their original order.
	seen := map[string]int{}
	for _, job := range plan.Jobs {
		for _, id := range job.TaskIDs {
			seen[id]++
		}
	}
	for _, seed := range seeds {
		if seen[seed.ID] != 1 {
			t.Fatalf("task %s is planned %d times, want exactly once", seed.ID, seen[seed.ID])
		}
	}
}
