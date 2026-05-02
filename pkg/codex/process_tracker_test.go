package codex

import (
	"os"
	"path/filepath"
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

// TestProcessTrackerKillAllEmptyRoot kills all tracked processes when root is empty.
func TestProcessTrackerKillAllEmptyRoot(t *testing.T) {
	resetProcessTestHooks(t)

	var terminated []int
	live := map[int]bool{101: true, 202: true}
	workerProcessExistsFunc = func(pid int) bool {
		return live[pid]
	}
	terminateWorkerFunc = func(pid int) error {
		terminated = append(terminated, pid)
		live[pid] = false
		return nil
	}
	killWorkerFunc = func(pid int) error {
		return nil
	}

	tracker := newProcessTracker()
	tracker.TrackProcess(101, TrackedProcess{WorkerName: "A"})
	tracker.TrackProcess(202, TrackedProcess{WorkerName: "B"})

	result := tracker.KillAll("")
	if len(result.Terminated) != 2 {
		t.Fatalf("result.Terminated = %v, want 2 terminated processes", result.Terminated)
	}
	terminatedSet := map[int]bool{}
	for _, pid := range result.Terminated {
		terminatedSet[pid] = true
	}
	if !terminatedSet[101] || !terminatedSet[202] {
		t.Fatalf("result.Terminated = %v, want both 101 and 202", result.Terminated)
	}
	if len(tracker.snapshot("")) != 0 {
		t.Fatalf("processes still tracked after KillAll: %+v", tracker.snapshot(""))
	}
}

// TestProcessTrackerPersistRead verifies that tracked processes are written to
// the JSON file and can be read back with matching data.
func TestProcessTrackerPersistRead(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	process := TrackedProcess{
		PID:        54321,
		WorkerName: "Mason-99",
		Caste:      "builder",
		Platform:   string(PlatformCodex),
		Root:       root,
		SpawnedAt:  now,
	}

	if err := writeTrackedProcesses(root, []TrackedProcess{process}); err != nil {
		t.Fatalf("writeTrackedProcesses: %v", err)
	}

	// Verify the file exists at the expected path
	registryPath := filepath.Join(root, ".aether", "data", "worker-processes.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Fatalf("registry file not created at %s", registryPath)
	}

	// Read back and verify data matches
	processes, err := readTrackedProcesses(root)
	if err != nil {
		t.Fatalf("readTrackedProcesses: %v", err)
	}
	if len(processes) != 1 {
		t.Fatalf("len(processes) = %d, want 1", len(processes))
	}
	got := processes[0]
	if got.PID != 54321 {
		t.Errorf("PID = %d, want 54321", got.PID)
	}
	if got.WorkerName != "Mason-99" {
		t.Errorf("WorkerName = %q, want %q", got.WorkerName, "Mason-99")
	}
	if got.Caste != "builder" {
		t.Errorf("Caste = %q, want %q", got.Caste, "builder")
	}
	if got.Platform != string(PlatformCodex) {
		t.Errorf("Platform = %q, want %q", got.Platform, PlatformCodex)
	}
}

// TestProcessTrackerCleanupStaleWorkers verifies the package-level
// CleanupStaleWorkers function detects and kills stale workers.
func TestProcessTrackerCleanupStaleWorkers(t *testing.T) {
	resetProcessTestHooks(t)
	root := t.TempDir()

	// Seed stale worker processes in the registry file
	staleTime := time.Now().UTC().Add(-1 * time.Hour)
	if err := writeTrackedProcesses(root, []TrackedProcess{
		{PID: 999, WorkerName: "stale-worker", Caste: "builder", Root: root, SpawnedAt: staleTime},
	}); err != nil {
		t.Fatalf("writeTrackedProcesses: %v", err)
	}

	// Mock process existence and command
	workerProcessExistsFunc = func(pid int) bool {
		return pid == 999
	}
	workerProcessCommandFunc = func(pid int) string {
		return "/usr/local/bin/codex exec"
	}

	var terminated []int
	terminateWorkerFunc = func(pid int) error {
		terminated = append(terminated, pid)
		return nil
	}
	killWorkerFunc = func(pid int) error {
		return nil
	}

	// Track a different process in the current tracker to avoid it being considered stale
	tracker := GlobalProcessTracker()
	tracker.TrackProcess(888, TrackedProcess{WorkerName: "active-worker", Root: root})
	defer tracker.UntrackProcess(888)

	result, err := CleanupStaleWorkers(root)
	if err != nil {
		t.Fatalf("CleanupStaleWorkers: %v", err)
	}

	if len(result.Stale) != 1 || result.Stale[0].PID != 999 {
		t.Fatalf("result.Stale = %+v, want stale pid 999", result.Stale)
	}
	if !reflect.DeepEqual(terminated, []int{999}) {
		t.Fatalf("terminated = %v, want [999]", terminated)
	}
}

// TestProcessTrackerNilGuards verifies nil and zero-PID operations don't panic.
func TestProcessTrackerNilGuards(t *testing.T) {
	var nilTracker *ProcessTracker

	// None of these should panic
	nilTracker.TrackProcess(1, TrackedProcess{})
	nilTracker.UntrackProcess(1)
	_ = nilTracker.KillProcess(1)
	_ = nilTracker.KillAll("")
	stale, err := nilTracker.DetectStaleWorkers("/some/path")
	if err != nil {
		t.Fatalf("DetectStaleWorkers on nil tracker returned error: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("nil tracker should return empty stale, got %v", stale)
	}

	// Zero PID should be a no-op
	tracker := newProcessTracker()
	tracker.TrackProcess(0, TrackedProcess{})
	tracker.UntrackProcess(0)
	_ = tracker.KillProcess(0)
	if len(tracker.snapshot("")) != 0 {
		t.Fatal("zero PID operations should not add to tracker")
	}
}
