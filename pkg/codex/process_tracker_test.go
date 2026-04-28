package codex

import (
	"reflect"
	"testing"
	"time"
)

func resetProcessTestHooks(t *testing.T) {
	t.Helper()
	origExists := workerProcessExistsFunc
	origCommand := workerProcessCommandFunc
	origTerminate := terminateWorkerFunc
	origKill := killWorkerFunc
	t.Cleanup(func() {
		workerProcessExistsFunc = origExists
		workerProcessCommandFunc = origCommand
		terminateWorkerFunc = origTerminate
		killWorkerFunc = origKill
	})
}

func TestProcessTrackerTrackUntrackPersistsRegistry(t *testing.T) {
	root := t.TempDir()
	tracker := newProcessTracker()

	tracker.TrackProcess(12345, TrackedProcess{
		WorkerName: "Mason-12",
		Caste:      "builder",
		Platform:   string(PlatformCodex),
		Root:       root,
		SpawnedAt:  time.Now().UTC(),
	})

	processes, err := readTrackedProcesses(root)
	if err != nil {
		t.Fatalf("readTrackedProcesses: %v", err)
	}
	if len(processes) != 1 || processes[0].PID != 12345 {
		t.Fatalf("persisted processes = %+v, want pid 12345", processes)
	}

	tracker.UntrackProcess(12345)
	processes, err = readTrackedProcesses(root)
	if err != nil {
		t.Fatalf("readTrackedProcesses after untrack: %v", err)
	}
	if len(processes) != 0 {
		t.Fatalf("persisted processes after untrack = %+v, want none", processes)
	}
}

func TestProcessTrackerKillAllFiltersByRoot(t *testing.T) {
	resetProcessTestHooks(t)
	rootA := t.TempDir()
	rootB := t.TempDir()
	live := map[int]bool{101: true, 202: true}
	var terminated []int
	workerProcessExistsFunc = func(pid int) bool {
		return live[pid]
	}
	terminateWorkerFunc = func(pid int) error {
		terminated = append(terminated, pid)
		live[pid] = false
		return nil
	}
	killWorkerFunc = func(pid int) error {
		t.Fatalf("unexpected force kill for pid %d", pid)
		return nil
	}

	tracker := newProcessTracker()
	tracker.TrackProcess(101, TrackedProcess{WorkerName: "A", Root: rootA})
	tracker.TrackProcess(202, TrackedProcess{WorkerName: "B", Root: rootB})

	result := tracker.KillAll(rootA)
	if !reflect.DeepEqual(terminated, []int{101}) {
		t.Fatalf("terminated = %v, want [101]", terminated)
	}
	if !reflect.DeepEqual(result.Terminated, []int{101}) {
		t.Fatalf("result.Terminated = %v, want [101]", result.Terminated)
	}
	if len(tracker.snapshot(rootA)) != 0 {
		t.Fatalf("rootA processes still tracked: %+v", tracker.snapshot(rootA))
	}
	if len(tracker.snapshot(rootB)) != 1 {
		t.Fatalf("rootB processes not preserved: %+v", tracker.snapshot(rootB))
	}
}

func TestProcessTrackerDetectStaleWorkersSameRootOnly(t *testing.T) {
	resetProcessTestHooks(t)
	root := t.TempDir()
	otherRoot := t.TempDir()
	workerProcessExistsFunc = func(pid int) bool {
		return pid == 111 || pid == 222 || pid == 333
	}
	workerProcessCommandFunc = func(pid int) string {
		if pid == 333 {
			return "/usr/bin/ssh-agent"
		}
		return "/usr/bin/codex exec"
	}

	if err := writeTrackedProcesses(root, []TrackedProcess{
		{PID: 111, WorkerName: "stale", Root: root, SpawnedAt: time.Now().UTC()},
		{PID: 222, WorkerName: "current", Root: root, SpawnedAt: time.Now().UTC()},
		{PID: 333, WorkerName: "not-worker", Root: root, SpawnedAt: time.Now().UTC()},
		{PID: 444, WorkerName: "other", Root: otherRoot, SpawnedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatalf("writeTrackedProcesses: %v", err)
	}

	tracker := newProcessTracker()
	tracker.TrackProcess(222, TrackedProcess{WorkerName: "current", Root: root})
	stale, err := tracker.DetectStaleWorkers(root)
	if err != nil {
		t.Fatalf("DetectStaleWorkers: %v", err)
	}
	if len(stale) != 1 || stale[0].PID != 111 {
		t.Fatalf("stale = %+v, want only pid 111", stale)
	}
}
