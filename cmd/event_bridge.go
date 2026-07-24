package cmd

import (
	"fmt"
	"time"
)

// SyncEventsWithTS watches the shared NDJSON event stream and forwards
// every new event to the provided callback.
//
// In stub mode it tails the in-memory buffer; in production it would
// poll or use fsnotify on .aether/events/current.ndjson.
func SyncEventsWithTS(streamPath string, onEvent func(EventLine) error) (stop func(), err error) {
	done := make(chan struct{})
	ch, err := TailEvents(streamPath, done)
	if err != nil {
		return nil, fmt.Errorf("tail events: %w", err)
	}

	go func() {
		for ev := range ch {
			if onEvent != nil {
				_ = onEvent(ev) // best-effort in stub mode
			}
		}
	}()

	stop = func() {
		close(done)
	}
	return stop, nil
}

// ReadTSEvents yields events written by the TypeScript side.
// It first returns all existing events, then continues tailing for new ones.
func ReadTSEvents(streamPath string, done <-chan struct{}) (<-chan EventLine, error) {
	// Pre-read existing events so the consumer sees the full history.
	existing, err := ReadEvents(streamPath)
	if err != nil {
		return nil, fmt.Errorf("read events: %w", err)
	}

	out := make(chan EventLine)
	go func() {
		defer close(out)

		// Emit everything that already exists.
		for _, ev := range existing {
			select {
			case out <- ev:
			case <-done:
				return
			}
		}

		// Tail for new events.
		ch, err := TailEvents(streamPath, done)
		if err != nil {
			return
		}
		for ev := range ch {
			select {
			case out <- ev:
			case <-done:
				return
			}
		}
	}()

	return out, nil
}

// DemonstrateBridgeCompatibility writes a Go-origin event, reads back
// all events (including any written by TS), and confirms the NDJSON
// shape is compatible on both sides.
func DemonstrateBridgeCompatibility(streamPath string) (goEventWritten EventLine, allEvents []EventLine, formatOk bool, err error) {
	// 1. Write a Go event.
	goEventWritten = NewEventLine(EventTypeCustom, EventPayload{
		"source":  "go-bridge",
		"message": "Hello from Go",
	})
	if err := WriteEvent(streamPath, goEventWritten); err != nil {
		return EventLine{}, nil, false, fmt.Errorf("write go event: %w", err)
	}

	// 2. Read all events.
	allEvents, err = ReadEvents(streamPath)
	if err != nil {
		return EventLine{}, nil, false, fmt.Errorf("read events: %w", err)
	}

	// 3. Verify format compatibility.
	formatOk = true
	for _, ev := range allEvents {
		if ev.Type == "" || ev.Timestamp == "" || ev.Payload == nil {
			formatOk = false
			break
		}
		// Timestamp must be parseable as RFC3339Nano.
		if _, parseErr := time.Parse(time.RFC3339Nano, ev.Timestamp); parseErr != nil {
			formatOk = false
			break
		}
	}

	return goEventWritten, allEvents, formatOk, nil
}
