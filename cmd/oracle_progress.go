package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Oracle round-by-round progress.
//
// The loop already rendered a per-iteration panel, and nobody ever saw it. The
// wrapper always starts Oracle in the background; a detached controller is
// launched with AETHER_OUTPUT_MODE=json (oracleBackgroundEnv), and
// emitVisualProgress returns early in json mode. So the one progress surface
// the loop had was, in practice, dead on every real run -- the live oracle.log
// on this repo is 35KB of JSON envelopes with no rounds in it.
//
// The fix is a log that does not care about output mode. Every round appends
// one line here regardless of how the process was started, and `aether oracle
// status --follow` streams those lines back to whoever asked.

const (
	oracleProgressEventRunStart        = "run_start"
	oracleProgressEventPhaseTransition = "phase_transition"
	oracleProgressEventIterationStart  = "iteration_start"
	oracleProgressEventAttemptStart    = "attempt_start"
	oracleProgressEventIterationEnd    = "iteration_end"
	oracleProgressEventRunEnd          = "run_end"

	// oracleProgressQuestionChars keeps a streamed line to roughly one
	// terminal row.
	oracleProgressQuestionChars = 160
)

type oracleProgressEvent struct {
	Timestamp        string `json:"ts"`
	Event            string `json:"event"`
	Iteration        int    `json:"iteration"`
	MaxIterations    int    `json:"max_iterations"`
	Remaining        int    `json:"remaining"`
	Phase            string `json:"phase,omitempty"`
	PreviousPhase    string `json:"previous_phase,omitempty"`
	QuestionID       string `json:"question_id,omitempty"`
	Question         string `json:"question,omitempty"`
	Confidence       int    `json:"confidence"`
	TargetConfidence int    `json:"target_confidence"`
	Reasoning        string `json:"reasoning,omitempty"`
	Attempt          int    `json:"attempt,omitempty"`
	Status           string `json:"status,omitempty"`
	StopReason       string `json:"stop_reason,omitempty"`
	// ConsecutiveLow (202-11, LIVE-06) mirrors
	// oracleStateFile.Novelty.ConsecutiveLow -- how many rounds in a row
	// have now added no new ground. It is never a new counter: only the
	// existing value the loop already tracks, carried onto the round line
	// and the live event payload so diminishing returns is visible, not
	// just measured.
	ConsecutiveLow int `json:"consecutive_low,omitempty"`
}

// newOracleProgressEvent snapshots the fields every event shares so callers
// only supply what is specific to their moment.
func newOracleProgressEvent(event string, state oracleStateFile) oracleProgressEvent {
	remaining := state.MaxIterations - state.Iteration
	if remaining < 0 {
		remaining = 0
	}
	return oracleProgressEvent{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Event:            event,
		Iteration:        state.Iteration,
		MaxIterations:    state.MaxIterations,
		Remaining:        remaining,
		Phase:            strings.TrimSpace(state.Phase),
		QuestionID:       strings.TrimSpace(state.ActiveQuestionID),
		Question:         truncateString(strings.Join(strings.Fields(state.ActiveQuestionText), " "), oracleProgressQuestionChars),
		Confidence:       state.OverallConfidence,
		TargetConfidence: state.TargetConfidence,
		Reasoning:        strings.TrimSpace(state.ActiveReasoning),
		Status:           strings.TrimSpace(state.Status),
		StopReason:       strings.TrimSpace(state.StopReason),
		ConsecutiveLow:   state.Novelty.ConsecutiveLow,
	}
}

// appendOracleProgressEvent writes one line and never fails the run. Progress
// reporting that can abort research would be worse than no reporting.
func appendOracleProgressEvent(path string, event oracleProgressEvent) {
	if strings.TrimSpace(path) == "" {
		return
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(stderr, "oracle progress: encode %s: %v\n", event.Event, err)
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(stderr, "oracle progress: open %s: %v\n", path, err)
		return
	}
	defer file.Close()
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		fmt.Fprintf(stderr, "oracle progress: write %s: %v\n", path, err)
	}
}

// emitOracleProgress records the round and, when the caller can actually see
// output, prints it. The recording is unconditional; only the printing is not.
func emitOracleProgress(path string, event oracleProgressEvent) {
	appendOracleProgressEvent(path, event)
	if line := renderOracleProgressLine(event); line != "" {
		emitVisualLine(line)
	}
}

// renderOracleProgressLine renders one round as a single line, so a long run
// reads as scrollback the operator can watch climb.
//
//	🔮 [ 5/30] survey  q6  conf 35%→90%  medium  Which files matter most to …
func renderOracleProgressLine(event oracleProgressEvent) string {
	switch event.Event {
	case oracleProgressEventIterationStart:
	case oracleProgressEventRunStart:
		return fmt.Sprintf("🔮 starting research — up to %d rounds, target %d%%", event.MaxIterations, event.TargetConfidence)
	case oracleProgressEventRunEnd:
		reason := event.StopReason
		if reason == "" {
			reason = event.Status
		}
		if reason == "diminishing_returns" {
			return fmt.Sprintf("🔮 finished after %d rounds at %d%% — stopped: the last %d answers in a row added no new ground", event.Iteration, event.Confidence, event.ConsecutiveLow)
		}
		return fmt.Sprintf("🔮 finished after %d rounds at %d%% (%s)", event.Iteration, event.Confidence, emptyFallback(reason, "done"))
	default:
		// Phase transitions, attempts and iteration ends are recorded for
		// `--follow` and diagnostics but would make the live view noisy.
		return ""
	}

	width := len(fmt.Sprintf("%d", event.MaxIterations))
	line := fmt.Sprintf("🔮 [%*d/%d] %-11s", width, event.Iteration, event.MaxIterations, emptyFallback(event.Phase, "survey"))
	if event.QuestionID != "" {
		line += fmt.Sprintf(" %-4s", event.QuestionID)
	}
	line += fmt.Sprintf(" conf %3d%%→%d%%", event.Confidence, event.TargetConfidence)
	if event.Reasoning != "" {
		line += fmt.Sprintf(" %-6s", event.Reasoning)
	}
	if event.Question != "" {
		line += "  " + truncateString(event.Question, 72)
	}
	if event.ConsecutiveLow > 0 {
		line += fmt.Sprintf("  (%d in a row added no new ground)", event.ConsecutiveLow)
	}
	return line
}

// --- follow ------------------------------------------------------------

const (
	defaultOracleFollowInterval = 2 * time.Second
	// oracleFollowGracePolls tolerates the gap between a run being started and
	// its controller writing the first line.
	oracleFollowGracePolls = 5
)

// followOracleProgress streams the round log until the run ends. It replays
// what has already happened first, so attaching late still shows the whole run.
//
// It never calls oracleStatusResult: that function repairs state when it finds
// a dead controller, and a poll loop must not rewrite state underneath a live
// run. Inspection does not mutate.
func followOracleProgress(root string, interval time.Duration) error {
	if interval <= 0 {
		interval = defaultOracleFollowInterval
	}
	paths := oracleWorkspacePaths(root)

	offset := int64(0)
	emptyPolls := 0
	for {
		lines, next, err := readOracleProgressFrom(paths.ProgressPath, offset)
		if err != nil {
			return err
		}
		offset = next

		if len(lines) == 0 {
			emptyPolls++
		} else {
			emptyPolls = 0
		}

		for _, event := range lines {
			if rendered := renderOracleProgressLine(event); rendered != "" {
				emitVisualLine(rendered)
			}
			if event.Event == oracleProgressEventRunEnd {
				return nil
			}
		}

		// Nothing is writing and no controller is alive: say so rather than
		// waiting forever on a run that already died.
		if emptyPolls >= oracleFollowGracePolls {
			state, stateErr := loadOracleStateFile(paths.StatePath)
			if stateErr != nil {
				return fmt.Errorf("no Oracle run to follow: %s is unavailable; start one with `aether oracle --from-brief`", paths.StatePath)
			}
			if state.ControllerPID > 0 && !oracleProcessExists(state.ControllerPID) {
				emitVisualLine(fmt.Sprintf("🔮 controller %d is gone; last seen at %d%% after %d rounds. Run `aether oracle recover` to clear the stale run.", state.ControllerPID, state.OverallConfidence, state.Iteration))
				return nil
			}
			if !strings.EqualFold(strings.TrimSpace(state.Status), "active") {
				emitVisualLine(fmt.Sprintf("🔮 run is %s at %d%% after %d rounds.", emptyFallback(state.Status, "idle"), state.OverallConfidence, state.Iteration))
				return nil
			}
		}

		time.Sleep(interval)
	}
}

// readOracleProgressFrom reads whole lines appended since offset. A partially
// written trailing line is left for the next poll rather than parsed.
func readOracleProgressFrom(path string, offset int64) ([]oracleProgressEvent, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, offset, nil
		}
		return nil, offset, fmt.Errorf("read oracle progress: %w", err)
	}
	if int64(len(data)) <= offset {
		// The workspace was archived and a new run started underneath us.
		if int64(len(data)) < offset {
			offset = 0
		} else {
			return nil, offset, nil
		}
	}

	chunk := data[offset:]
	lastNewline := strings.LastIndexByte(string(chunk), '\n')
	if lastNewline < 0 {
		return nil, offset, nil
	}
	complete := string(chunk[:lastNewline+1])
	next := offset + int64(lastNewline) + 1

	events := make([]oracleProgressEvent, 0, 8)
	for _, line := range strings.Split(complete, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event oracleProgressEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// A corrupt line is not worth killing the stream over.
			continue
		}
		events = append(events, event)
	}
	return events, next, nil
}
