package codex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

const (
	envWorkerColony = "AETHER_WORKER_COLONY"
	envWorkerName   = "AETHER_WORKER_NAME"
	envWorkerCaste  = "AETHER_WORKER_CASTE"

	workerProcessRegistryRel = ".aether/data/worker-processes.json"
	workerCleanupGrace       = 2 * time.Second
)

// TrackedProcess records a worker subprocess that Aether can clean up.
type TrackedProcess struct {
	PID           int               `json:"pid"`
	WorkerName    string            `json:"worker_name,omitempty"`
	TaskID        string            `json:"task_id,omitempty"`
	Caste         string            `json:"caste,omitempty"`
	Platform      string            `json:"platform,omitempty"`
	Root          string            `json:"root,omitempty"`
	ProviderRunID string            `json:"provider_run_id,omitempty"`
	Binding       *ExecutionBinding `json:"execution_binding,omitempty"`
	SpawnedAt     time.Time         `json:"spawned_at"`
}

// CleanupResult summarizes worker cleanup activity.
type CleanupResult struct {
	Stale      []TrackedProcess `json:"stale,omitempty"`
	Terminated []int            `json:"terminated,omitempty"`
	Killed     []int            `json:"killed,omitempty"`
	Failures   []string         `json:"failures,omitempty"`
}

type trackedProcessFile struct {
	Processes []TrackedProcess `json:"processes"`
}

// ProcessTracker tracks live worker subprocesses for cleanup.
type ProcessTracker struct {
	mu        sync.Mutex
	processes map[int]TrackedProcess
}

var (
	globalProcessTracker     = newProcessTracker()
	workerProcessExistsFunc  = workerProcessExists
	workerProcessCommandFunc = workerProcessCommandLine
	terminateWorkerFunc      = terminateWorkerProcess
	killWorkerFunc           = killWorkerProcess
)

func newProcessTracker() *ProcessTracker {
	return &ProcessTracker{processes: map[int]TrackedProcess{}}
}

// GlobalProcessTracker returns the singleton worker process tracker.
func GlobalProcessTracker() *ProcessTracker {
	return globalProcessTracker
}

// TrackProcess registers a subprocess and persists the record for stale cleanup.
func (p *ProcessTracker) TrackProcess(pid int, process TrackedProcess) {
	if p == nil || pid <= 0 {
		return
	}
	process.PID = pid
	process.Root = normalizeProcessRoot(process.Root)
	process.WorkerName = strings.TrimSpace(process.WorkerName)
	process.TaskID = strings.TrimSpace(process.TaskID)
	process.Caste = strings.TrimSpace(process.Caste)
	process.Platform = strings.TrimSpace(process.Platform)
	process.ProviderRunID = strings.TrimSpace(process.ProviderRunID)
	if process.SpawnedAt.IsZero() {
		process.SpawnedAt = time.Now().UTC()
	}

	p.mu.Lock()
	p.processes[pid] = process
	p.mu.Unlock()

	if process.Root != "" {
		_ = upsertTrackedProcess(process.Root, process)
	}
}

// UntrackProcess unregisters a subprocess after it exits.
func (p *ProcessTracker) UntrackProcess(pid int) {
	if p == nil || pid <= 0 {
		return
	}
	var root string
	p.mu.Lock()
	if process, ok := p.processes[pid]; ok {
		root = process.Root
	}
	delete(p.processes, pid)
	p.mu.Unlock()

	if strings.TrimSpace(root) != "" {
		_ = removeTrackedProcess(root, pid)
	}
}

// KillProcess terminates a tracked process group by PID.
func (p *ProcessTracker) KillProcess(pid int) CleanupResult {
	if p == nil || pid <= 0 {
		return CleanupResult{}
	}
	p.mu.Lock()
	process, ok := p.processes[pid]
	p.mu.Unlock()
	if !ok {
		process = TrackedProcess{PID: pid}
	}
	result := killTrackedProcess(process)
	if ok {
		p.UntrackProcess(pid)
	}
	return result
}

// KillAll terminates tracked process groups. When root is non-empty, cleanup is
// restricted to that repo root.
func (p *ProcessTracker) KillAll(root string) CleanupResult {
	if p == nil {
		return CleanupResult{}
	}
	root = normalizeProcessRoot(root)
	processes := p.snapshot(root)
	var result CleanupResult
	for _, process := range processes {
		mergeCleanupResult(&result, killTrackedProcess(process))
		p.UntrackProcess(process.PID)
	}
	return result
}

// ProcessesForRun returns live, known provider processes bound to one durable
// execution run. Persisted records are included so a new Aether process can
// diagnose work left behind after its predecessor exited.
func (p *ProcessTracker) ProcessesForRun(root, runID string) ([]TrackedProcess, error) {
	if p == nil {
		return nil, nil
	}
	root = normalizeProcessRoot(root)
	runID = strings.TrimSpace(runID)
	if root == "" || runID == "" {
		return nil, nil
	}
	persisted, err := readTrackedProcesses(root)
	if err != nil {
		return nil, err
	}
	byPID := map[int]TrackedProcess{}
	for _, process := range append(persisted, p.snapshot(root)...) {
		if process.PID <= 0 || process.Binding == nil || strings.TrimSpace(process.Binding.RunID) != runID {
			continue
		}
		if !workerProcessExistsFunc(process.PID) || !isKnownWorkerProcess(process.PID) {
			continue
		}
		byPID[process.PID] = process
	}
	result := make([]TrackedProcess, 0, len(byPID))
	for _, process := range byPID {
		result = append(result, process)
	}
	return result, nil
}

// KillRun cancels only provider processes carrying the requested durable run
// identity. It never falls back to a repository-wide kill.
func (p *ProcessTracker) KillRun(root, runID string) (CleanupResult, error) {
	processes, err := p.ProcessesForRun(root, runID)
	if err != nil {
		return CleanupResult{}, err
	}
	result := CleanupResult{Stale: append([]TrackedProcess(nil), processes...)}
	for _, process := range processes {
		mergeCleanupResult(&result, killTrackedProcess(process))
		p.UntrackProcess(process.PID)
		if process.Root != "" {
			_ = removeTrackedProcess(process.Root, process.PID)
		}
	}
	return result, nil
}

// DetectStaleWorkers returns persisted same-root worker processes that are no
// longer tracked by this process but still appear to be running.
func (p *ProcessTracker) DetectStaleWorkers(root string) ([]TrackedProcess, error) {
	if p == nil {
		return nil, nil
	}
	root = normalizeProcessRoot(root)
	if root == "" {
		return nil, nil
	}
	persisted, err := readTrackedProcesses(root)
	if err != nil {
		return nil, err
	}

	tracked := p.snapshotMap("")
	stale := make([]TrackedProcess, 0, len(persisted))
	keep := make([]TrackedProcess, 0, len(persisted))
	for _, process := range persisted {
		process.Root = normalizeProcessRoot(process.Root)
		if process.Root != root || process.PID <= 0 {
			continue
		}
		if !workerProcessExistsFunc(process.PID) {
			continue
		}
		if !isKnownWorkerProcess(process.PID) {
			continue
		}
		if _, ok := tracked[process.PID]; ok {
			keep = append(keep, process)
			continue
		}
		stale = append(stale, process)
		keep = append(keep, process)
	}
	if len(keep) != len(persisted) {
		_ = writeTrackedProcesses(root, keep)
	}
	return stale, nil
}

// CleanupStaleWorkers detects and kills stale same-root worker processes.
func CleanupStaleWorkers(root string) (CleanupResult, error) {
	tracker := GlobalProcessTracker()
	stale, err := tracker.DetectStaleWorkers(root)
	if err != nil {
		return CleanupResult{}, err
	}
	result := CleanupResult{Stale: stale}
	for _, process := range stale {
		mergeCleanupResult(&result, killTrackedProcess(process))
		if process.Root != "" {
			_ = removeTrackedProcess(process.Root, process.PID)
		}
	}
	return result, nil
}

func (p *ProcessTracker) snapshot(root string) []TrackedProcess {
	root = normalizeProcessRoot(root)
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]TrackedProcess, 0, len(p.processes))
	for _, process := range p.processes {
		if root != "" && normalizeProcessRoot(process.Root) != root {
			continue
		}
		out = append(out, process)
	}
	return out
}

func (p *ProcessTracker) snapshotMap(root string) map[int]TrackedProcess {
	processes := p.snapshot(root)
	out := make(map[int]TrackedProcess, len(processes))
	for _, process := range processes {
		out[process.PID] = process
	}
	return out
}

func killTrackedProcess(process TrackedProcess) CleanupResult {
	if process.PID <= 0 {
		return CleanupResult{}
	}
	result := CleanupResult{}
	if !workerProcessExistsFunc(process.PID) {
		return result
	}
	if err := terminateWorkerFunc(process.PID); err != nil {
		result.Failures = append(result.Failures, fmt.Sprintf("terminate %d: %v", process.PID, err))
	} else {
		result.Terminated = append(result.Terminated, process.PID)
	}

	deadline := time.Now().Add(workerCleanupGrace)
	for time.Now().Before(deadline) {
		if !workerProcessExistsFunc(process.PID) {
			return result
		}
		time.Sleep(100 * time.Millisecond)
	}
	if workerProcessExistsFunc(process.PID) {
		if err := killWorkerFunc(process.PID); err != nil {
			result.Failures = append(result.Failures, fmt.Sprintf("kill %d: %v", process.PID, err))
		} else {
			result.Killed = append(result.Killed, process.PID)
		}
	}
	return result
}

func mergeCleanupResult(target *CleanupResult, next CleanupResult) {
	if target == nil {
		return
	}
	target.Stale = append(target.Stale, next.Stale...)
	target.Terminated = append(target.Terminated, next.Terminated...)
	target.Killed = append(target.Killed, next.Killed...)
	target.Failures = append(target.Failures, next.Failures...)
}

func workerProcessEnv(base []string, config WorkerConfig, platform Platform) []string {
	env := append([]string{}, base...)
	root := normalizeProcessRoot(config.Root)
	if root != "" {
		env = setEnvValue(env, envWorkerColony, root)
	}
	if name := strings.TrimSpace(config.WorkerName); name != "" {
		env = setEnvValue(env, envWorkerName, name)
	}
	if caste := strings.TrimSpace(config.Caste); caste != "" {
		env = setEnvValue(env, envWorkerCaste, caste)
	}
	if platform != "" {
		env = setEnvValue(env, envWorkerPlatform, string(platform))
	}
	return env
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	entry := prefix + value
	for i, existing := range env {
		if strings.HasPrefix(existing, prefix) {
			env[i] = entry
			return env
		}
	}
	return append(env, entry)
}

func isKnownWorkerProcess(pid int) bool {
	command := strings.TrimSpace(workerProcessCommandFunc(pid))
	if command == "" {
		return false
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	binary := strings.ToLower(filepath.Base(fields[0]))
	return strings.Contains(binary, "codex") ||
		strings.Contains(binary, "claude") ||
		strings.Contains(binary, "opencode")
}

func normalizeProcessRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return filepath.Clean(abs)
}

func trackedProcessRegistryPath(root string) string {
	root = normalizeProcessRoot(root)
	if root == "" {
		return ""
	}
	return filepath.Join(root, workerProcessRegistryRel)
}

func readTrackedProcesses(root string) ([]TrackedProcess, error) {
	path := trackedProcessRegistryPath(root)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read worker process registry: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var file trackedProcessFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse worker process registry: %w", err)
	}
	return file.Processes, nil
}

func writeTrackedProcesses(root string, processes []TrackedProcess) error {
	registryStore, err := trackedProcessStore(root)
	if err != nil || registryStore == nil {
		return err
	}
	payload, err := json.MarshalIndent(trackedProcessFile{Processes: processes}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode worker process registry: %w", err)
	}
	if err := registryStore.AtomicWrite(filepath.Base(workerProcessRegistryRel), payload); err != nil {
		return fmt.Errorf("replace worker process registry: %w", err)
	}
	return nil
}

func trackedProcessStore(root string) (*storage.Store, error) {
	root = normalizeProcessRoot(root)
	if root == "" {
		return nil, nil
	}
	registryStore, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		return nil, fmt.Errorf("open worker process registry store: %w", err)
	}
	return registryStore, nil
}

func updateTrackedProcesses(root string, mutate func(*trackedProcessFile)) error {
	registryStore, err := trackedProcessStore(root)
	if err != nil || registryStore == nil {
		return err
	}
	var file trackedProcessFile
	if err := registryStore.UpdateJSONAtomically(filepath.Base(workerProcessRegistryRel), &file, func() error {
		mutate(&file)
		return nil
	}); err != nil {
		return fmt.Errorf("update worker process registry: %w", err)
	}
	return nil
}

func upsertTrackedProcess(root string, process TrackedProcess) error {
	return updateTrackedProcesses(root, func(file *trackedProcessFile) {
		next := make([]TrackedProcess, 0, len(file.Processes)+1)
		for _, existing := range file.Processes {
			if existing.PID == process.PID {
				continue
			}
			next = append(next, existing)
		}
		file.Processes = append(next, process)
	})
}

func removeTrackedProcess(root string, pid int) error {
	return updateTrackedProcesses(root, func(file *trackedProcessFile) {
		next := make([]TrackedProcess, 0, len(file.Processes))
		for _, existing := range file.Processes {
			if existing.PID == pid {
				continue
			}
			next = append(next, existing)
		}
		file.Processes = next
	})
}

// Test helper functions -- exported for use by cmd tests.

// WriteTrackedProcessesForTest writes tracked processes to the registry file for testing.
func WriteTrackedProcessesForTest(root string, processes []TrackedProcess) error {
	return writeTrackedProcesses(root, processes)
}

// SetWorkerProcessExistsFunc sets the process existence check function (for testing).
func SetWorkerProcessExistsFunc(f func(int) bool) { workerProcessExistsFunc = f }

// WorkerProcessExistsFunc returns the current process existence check function.
func WorkerProcessExistsFunc() func(int) bool { return workerProcessExistsFunc }

// SetWorkerProcessCommandFunc sets the process command line check function (for testing).
func SetWorkerProcessCommandFunc(f func(int) string) { workerProcessCommandFunc = f }

// WorkerProcessCommandFunc returns the current process command line check function.
func WorkerProcessCommandFunc() func(int) string { return workerProcessCommandFunc }

// SetWorkerTerminateFunc sets the terminate function (for testing).
func SetWorkerTerminateFunc(f func(int) error) { terminateWorkerFunc = f }

// TerminateWorkerFunc returns the current terminate function.
func TerminateWorkerFunc() func(int) error { return terminateWorkerFunc }

// SetWorkerKillFunc sets the kill function (for testing).
func SetWorkerKillFunc(f func(int) error) { killWorkerFunc = f }

// KillWorkerFunc returns the current kill function.
func KillWorkerFunc() func(int) error { return killWorkerFunc }
