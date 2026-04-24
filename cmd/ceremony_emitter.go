package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

const (
	ceremonyTextLimit     = 500
	ceremonyListLimit     = 20
	ceremonyListItemLimit = 240
)

type ceremonyNarrator interface {
	EmitEvent(events.Event)
	Close()
}

type buildCeremonyEmitter struct {
	bus      *events.Bus
	narrator ceremonyNarrator
	source   string

	phaseID   int
	phaseName string
}

var (
	activeBuildCeremonyMu sync.RWMutex
	activeBuildCeremony   *buildCeremonyEmitter
)

func newBuildCeremonyEmitter(ctx context.Context, root string, phase colony.Phase) *buildCeremonyEmitter {
	var bus *events.Bus
	if store != nil {
		bus = events.NewBus(store, events.DefaultConfig())
	}
	return &buildCeremonyEmitter{
		bus:       bus,
		narrator:  maybeLaunchNarrator(ctx, root),
		source:    "aether-build",
		phaseID:   phase.ID,
		phaseName: strings.TrimSpace(phase.Name),
	}
}

func setActiveBuildCeremony(emitter *buildCeremonyEmitter) func() {
	activeBuildCeremonyMu.Lock()
	previous := activeBuildCeremony
	activeBuildCeremony = emitter
	activeBuildCeremonyMu.Unlock()

	return func() {
		activeBuildCeremonyMu.Lock()
		activeBuildCeremony = previous
		activeBuildCeremonyMu.Unlock()
	}
}

func currentBuildCeremony() *buildCeremonyEmitter {
	activeBuildCeremonyMu.RLock()
	defer activeBuildCeremonyMu.RUnlock()
	return activeBuildCeremony
}

func (e *buildCeremonyEmitter) Close() {
	if e == nil || e.narrator == nil {
		return
	}
	e.narrator.Close()
}

func (e *buildCeremonyEmitter) Emit(topic string, payload events.CeremonyPayload) {
	if e == nil || strings.TrimSpace(topic) == "" {
		return
	}
	if payload.Phase == 0 {
		payload.Phase = e.phaseID
	}
	if strings.TrimSpace(payload.PhaseName) == "" {
		payload.PhaseName = e.phaseName
	}
	payload = trimCeremonyPayload(payload)

	raw, err := payload.RawMessage()
	if err != nil {
		return
	}

	if e.bus != nil {
		if evt, err := e.bus.Publish(context.Background(), topic, raw, e.source); err == nil {
			e.emitToNarrator(*evt)
			return
		}
	}

	e.emitToNarrator(syntheticCeremonyEvent(topic, raw, e.source))
}

func (e *buildCeremonyEmitter) emitToNarrator(evt events.Event) {
	if e == nil || e.narrator == nil {
		return
	}
	e.narrator.EmitEvent(evt)
}

func emitLifecycleCeremony(topic string, payload events.CeremonyPayload, source string) {
	if store == nil || strings.TrimSpace(topic) == "" {
		return
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "aether"
	}
	payload = trimCeremonyPayload(payload)
	raw, err := payload.RawMessage()
	if err != nil {
		return
	}
	bus := events.NewBus(store, events.DefaultConfig())
	_, _ = bus.Publish(context.Background(), topic, raw, source)
}

type lifecycleCeremonyTopics struct {
	WaveStart string
	Spawn     string
	WaveEnd   string
}

type lifecycleCeremonyStep struct {
	Wave     int
	SpawnID  string
	Caste    string
	Name     string
	TaskID   string
	Task     string
	Status   string
	Message  string
	Blockers []string
}

func emitLifecycleCeremonySequence(topics lifecycleCeremonyTopics, source, lifecycle string, phase int, phaseName string, steps []lifecycleCeremonyStep) {
	if len(steps) == 0 || strings.TrimSpace(topics.WaveStart) == "" || strings.TrimSpace(topics.Spawn) == "" || strings.TrimSpace(topics.WaveEnd) == "" {
		return
	}
	lifecycle = strings.TrimSpace(lifecycle)
	if lifecycle == "" {
		lifecycle = "lifecycle"
	}

	waves := map[int][]lifecycleCeremonyStep{}
	waveOrder := []int{}
	for _, step := range steps {
		wave := step.Wave
		if wave <= 0 {
			wave = 1
		}
		step.Wave = wave
		if _, seen := waves[wave]; !seen {
			waveOrder = append(waveOrder, wave)
		}
		waves[wave] = append(waves[wave], step)
	}

	for _, wave := range waveOrder {
		waveSteps := waves[wave]
		emitLifecycleCeremony(topics.WaveStart, events.CeremonyPayload{
			Phase:     phase,
			PhaseName: phaseName,
			Wave:      wave,
			Total:     len(waveSteps),
			Status:    "starting",
			Message:   fmt.Sprintf("%s wave %d with %d worker(s)", lifecycle, wave, len(waveSteps)),
		}, source)

		completed := 0
		blockers := []string{}
		for _, step := range waveSteps {
			status := ceremonyStepStatus(step.Status)
			if ceremonyStepCompleted(status) {
				completed++
			}
			blockers = append(blockers, step.Blockers...)
			if !ceremonyStepCompleted(status) && strings.TrimSpace(step.Message) != "" {
				blockers = append(blockers, step.Message)
			}
			emitLifecycleCeremony(topics.Spawn, events.CeremonyPayload{
				Phase:     phase,
				PhaseName: phaseName,
				Wave:      wave,
				SpawnID:   step.SpawnID,
				Caste:     step.Caste,
				Name:      step.Name,
				TaskID:    step.TaskID,
				Task:      step.Task,
				Status:    status,
				Message:   step.Message,
				Blockers:  step.Blockers,
			}, source)
		}

		waveStatus := "completed"
		if completed < len(waveSteps) || len(blockers) > 0 {
			waveStatus = "blocked"
		}
		emitLifecycleCeremony(topics.WaveEnd, events.CeremonyPayload{
			Phase:     phase,
			PhaseName: phaseName,
			Wave:      wave,
			Status:    waveStatus,
			Completed: completed,
			Total:     len(waveSteps),
			Blockers:  blockers,
			Message:   fmt.Sprintf("%s wave %d %s", lifecycle, wave, waveStatus),
		}, source)
	}
}

func ceremonyStepStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" || status == "spawned" || status == "planned" {
		return "completed"
	}
	return status
}

func ceremonyStepCompleted(status string) bool {
	switch strings.TrimSpace(status) {
	case "", "completed", "manually-reconciled", "skipped":
		return true
	default:
		return false
	}
}

func emitPlanCeremonyDispatchSequence(source string, dispatches []codexPlanningDispatch) {
	steps := make([]lifecycleCeremonyStep, 0, len(dispatches))
	for idx, dispatch := range dispatches {
		wave := dispatch.Wave
		if wave <= 0 {
			wave = idx + 1
		}
		taskID := strings.TrimSpace(dispatch.TaskID)
		if taskID == "" {
			taskID = fmt.Sprintf("plan-%d", idx)
		}
		message := strings.TrimSpace(dispatch.Summary)
		if message == "" {
			message = strings.Join(dispatch.Outputs, ", ")
		}
		steps = append(steps, lifecycleCeremonyStep{
			Wave:     wave,
			SpawnID:  taskID,
			Caste:    dispatch.Caste,
			Name:     dispatch.Name,
			TaskID:   taskID,
			Task:     dispatch.Task,
			Status:   dispatch.Status,
			Message:  message,
			Blockers: append([]string{}, dispatch.Blockers...),
		})
	}
	emitLifecycleCeremonySequence(lifecycleCeremonyTopics{
		WaveStart: events.CeremonyTopicPlanWaveStart,
		Spawn:     events.CeremonyTopicPlanSpawn,
		WaveEnd:   events.CeremonyTopicPlanWaveEnd,
	}, source, "plan", 0, "Planning", steps)
}

func emitColonizeCeremonyDispatchSequence(source string, dispatches []codexSurveyorDispatch) {
	steps := make([]lifecycleCeremonyStep, 0, len(dispatches))
	for idx, dispatch := range dispatches {
		taskID := fmt.Sprintf("survey-%d", idx)
		message := strings.TrimSpace(dispatch.Summary)
		if message == "" {
			message = strings.Join(dispatch.Outputs, ", ")
		}
		steps = append(steps, lifecycleCeremonyStep{
			Wave:    1,
			SpawnID: taskID,
			Caste:   dispatch.Caste,
			Name:    dispatch.Name,
			TaskID:  taskID,
			Task:    dispatch.Task,
			Status:  dispatch.Status,
			Message: message,
		})
	}
	emitLifecycleCeremonySequence(lifecycleCeremonyTopics{
		WaveStart: events.CeremonyTopicColonizeWaveStart,
		Spawn:     events.CeremonyTopicColonizeSpawn,
		WaveEnd:   events.CeremonyTopicColonizeWaveEnd,
	}, source, "colonize", 0, "Territory Survey", steps)
}

func emitContinueCeremonyFlowSequence(source string, phase colony.Phase, workerFlow []codexContinueWorkerFlowStep) {
	steps := make([]lifecycleCeremonyStep, 0, len(workerFlow))
	for idx, step := range workerFlow {
		taskID := fmt.Sprintf("continue-%d", idx)
		if stage := strings.TrimSpace(step.Stage); stage != "" {
			taskID = fmt.Sprintf("continue-%s-%d", stage, idx)
		}
		steps = append(steps, lifecycleCeremonyStep{
			Wave:    continueCeremonyWaveForStage(step.Stage),
			SpawnID: taskID,
			Caste:   step.Caste,
			Name:    step.Name,
			TaskID:  taskID,
			Task:    continueWorkerFlowTask(step),
			Status:  step.Status,
			Message: step.Summary,
		})
	}
	emitLifecycleCeremonySequence(lifecycleCeremonyTopics{
		WaveStart: events.CeremonyTopicContinueWaveStart,
		Spawn:     events.CeremonyTopicContinueSpawn,
		WaveEnd:   events.CeremonyTopicContinueWaveEnd,
	}, source, "continue", phase.ID, phase.Name, steps)
}

func continueCeremonyWaveForStage(stage string) int {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "verification":
		return 1
	case "review":
		return 2
	case "housekeeping":
		return 3
	default:
		return 1
	}
}

func syntheticCeremonyEvent(topic string, payload json.RawMessage, source string) events.Event {
	now := time.Now().UTC()
	return events.Event{
		ID:        fmt.Sprintf("evt_narrator_%d", now.UnixNano()),
		Topic:     topic,
		Payload:   payload,
		Source:    source,
		Timestamp: events.FormatTimestamp(now),
		TTLDays:   events.DefaultTTL,
		ExpiresAt: events.FormatTimestamp(events.ComputeExpiry(now, events.DefaultTTL)),
	}
}

func emitBuildCeremony(topic string, payload events.CeremonyPayload) {
	if emitter := currentBuildCeremony(); emitter != nil {
		emitter.Emit(topic, payload)
	}
}

func emitBuildCeremonyPrewave(phase colony.Phase, dispatches []codexBuildDispatch, waveCount int) {
	emitBuildCeremony(events.CeremonyTopicBuildPrewave, events.CeremonyPayload{
		Phase:           phase.ID,
		PhaseName:       phase.Name,
		Total:           len(dispatches),
		SuccessCriteria: append([]string{}, phase.SuccessCriteria...),
		Message:         fmt.Sprintf("%d workers planned across %d waves", len(dispatches), max(waveCount, 1)),
	})
}

func emitBuildCeremonyWaveStart(phase colony.Phase, wave int, dispatches []codex.WorkerDispatch, parallelMode colony.ParallelMode) {
	policy := buildWaveExecutionPlan(wave, len(dispatches), parallelMode)
	emitBuildCeremony(events.CeremonyTopicBuildWaveStart, events.CeremonyPayload{
		Phase:     phase.ID,
		PhaseName: phase.Name,
		Wave:      wave,
		Total:     len(dispatches),
		Status:    "starting",
		Message:   fmt.Sprintf("%s wave with %d worker(s)", policy.Strategy, len(dispatches)),
	})
}

func emitBuildCeremonyWorkerStarting(dispatch codex.WorkerDispatch, wave int) {
	emitBuildCeremony(events.CeremonyTopicBuildSpawn, ceremonyPayloadForDispatch(dispatch, wave, "starting", ""))
}

func emitBuildCeremonyWorkerRunning(dispatch codex.WorkerDispatch, wave int, message string) {
	emitBuildCeremony(events.CeremonyTopicBuildToolUse, ceremonyPayloadForDispatch(dispatch, wave, "running", message))
}

func emitBuildCeremonyWorkerTimeout(dispatch codex.WorkerDispatch, wave int, err error) {
	message := "worker timed out before start"
	if err != nil {
		message = err.Error()
	}
	payload := ceremonyPayloadForDispatch(dispatch, wave, "timeout", message)
	if err != nil {
		payload.Blockers = []string{err.Error()}
	}
	emitBuildCeremony(events.CeremonyTopicBuildSpawn, payload)
}

func emitBuildCeremonyWorkerFailed(dispatch codex.WorkerDispatch, wave int, err error) {
	message := "worker failed"
	if err != nil {
		message = err.Error()
	}
	payload := ceremonyPayloadForDispatch(dispatch, wave, "failed", message)
	if err != nil {
		payload.Blockers = []string{err.Error()}
	}
	emitBuildCeremony(events.CeremonyTopicBuildSpawn, payload)
}

func emitBuildCeremonyWorkerFinished(dispatch codex.WorkerDispatch, result codex.DispatchResult) {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = "failed"
	}
	payload := ceremonyPayloadForDispatch(dispatch, dispatch.Wave, status, dispatchResultSummary(dispatch, result))
	if result.WorkerResult != nil {
		payload.FilesCreated = append([]string{}, result.WorkerResult.FilesCreated...)
		payload.FilesModified = append([]string{}, result.WorkerResult.FilesModified...)
		payload.TestsWritten = append([]string{}, result.WorkerResult.TestsWritten...)
		payload.ToolCount = result.WorkerResult.ToolCount
		payload.Blockers = append(payload.Blockers, result.WorkerResult.Blockers...)
	}
	if result.Error != nil {
		payload.Blockers = append(payload.Blockers, result.Error.Error())
	}
	emitBuildCeremony(events.CeremonyTopicBuildSpawn, payload)
}

func emitBuildCeremonyWaveEnd(phase colony.Phase, wave int, results []codex.DispatchResult) {
	completed := 0
	blockers := []string{}
	for _, result := range results {
		if result.Status == "completed" {
			completed++
		}
		if result.Error != nil {
			blockers = append(blockers, result.Error.Error())
		}
		if result.WorkerResult != nil {
			blockers = append(blockers, result.WorkerResult.Blockers...)
		}
	}
	emitBuildCeremony(events.CeremonyTopicBuildWaveEnd, events.CeremonyPayload{
		Phase:     phase.ID,
		PhaseName: phase.Name,
		Wave:      wave,
		Status:    "completed",
		Completed: completed,
		Total:     len(results),
		Blockers:  blockers,
		Message:   fmt.Sprintf("wave %d completed", wave),
	})
}

func ceremonyPayloadForDispatch(dispatch codex.WorkerDispatch, wave int, status, message string) events.CeremonyPayload {
	return events.CeremonyPayload{
		Wave:    wave,
		SpawnID: dispatch.ID,
		Caste:   dispatch.Caste,
		Name:    dispatch.WorkerName,
		TaskID:  dispatch.TaskID,
		Task:    workerDispatchSummary(dispatch),
		Status:  status,
		Message: message,
	}
}

func trimCeremonyPayload(payload events.CeremonyPayload) events.CeremonyPayload {
	payload.PhaseName = trimCeremonyText(payload.PhaseName, ceremonyTextLimit)
	payload.SpawnID = trimCeremonyText(payload.SpawnID, ceremonyTextLimit)
	payload.Caste = trimCeremonyText(payload.Caste, ceremonyTextLimit)
	payload.Name = trimCeremonyText(payload.Name, ceremonyTextLimit)
	payload.TaskID = trimCeremonyText(payload.TaskID, ceremonyTextLimit)
	payload.Task = trimCeremonyText(payload.Task, ceremonyTextLimit)
	payload.Status = trimCeremonyText(payload.Status, ceremonyTextLimit)
	payload.Message = trimCeremonyText(payload.Message, ceremonyTextLimit)
	payload.Skill = trimCeremonyText(payload.Skill, ceremonyTextLimit)
	payload.PheromoneType = trimCeremonyText(payload.PheromoneType, ceremonyTextLimit)
	payload.FilesCreated = trimCeremonyList(payload.FilesCreated)
	payload.FilesModified = trimCeremonyList(payload.FilesModified)
	payload.TestsWritten = trimCeremonyList(payload.TestsWritten)
	payload.Blockers = trimCeremonyList(payload.Blockers)
	payload.SuccessCriteria = trimCeremonyList(payload.SuccessCriteria)
	return payload
}

func trimCeremonyList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	limit := len(values)
	if limit > ceremonyListLimit {
		limit = ceremonyListLimit
	}
	out := make([]string, 0, limit)
	for _, value := range values[:limit] {
		value = trimCeremonyText(value, ceremonyListItemLimit)
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func trimCeremonyText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 1 {
		return value[:limit]
	}
	if limit <= 3 {
		return value[:limit]
	}
	return strings.TrimSpace(value[:limit-3]) + "..."
}
