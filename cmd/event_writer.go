package cmd

import (
	"os"
	"sync"
)

// EventStream manages atomic NDJSON writes to a file.
type EventStream struct {
	path   string
	file   *os.File
	mu     sync.Mutex
	closed bool
}

// WriteEvent appends a single EventLine as NDJSON to the given file path.
// It opens the file with O_APPEND|O_CREATE|O_WRONLY, writes the line, and closes.
// For high-frequency writes, use CreateEventStream instead.
func WriteEvent(streamPath string, eventLine EventLine) error {
	f, err := os.OpenFile(streamPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := eventLine.ToNDJSON()
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}

// CreateEventStream opens a persistent event stream for atomic writes.
// The returned EventStream must be closed with Close() when done.
func CreateEventStream(streamPath string) (*EventStream, error) {
	f, err := os.OpenFile(streamPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &EventStream{
		path: streamPath,
		file: f,
	}, nil
}

// Write appends an event line to the stream. Thread-safe.
func (es *EventStream) Write(eventLine EventLine) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.closed {
		return os.ErrClosed
	}

	data, err := eventLine.ToNDJSON()
	if err != nil {
		return err
	}

	_, err = es.file.Write(data)
	if err != nil {
		return err
	}
	// Ensure the line is on disk for crash safety.
	return es.file.Sync()
}

// Close closes the underlying file. Thread-safe.
func (es *EventStream) Close() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.closed {
		return nil
	}
	es.closed = true
	return es.file.Close()
}

// In-memory stub helpers for testing and stub mode.

var (
	memoryBuffers   = make(map[string][]EventLine)
	memoryBuffersMu sync.RWMutex
)

// WriteEventToMemory writes an event line to an in-memory buffer keyed by streamPath.
// This is the stub-mode equivalent of WriteEvent, useful when file I/O is not desired.
func WriteEventToMemory(streamPath string, eventLine EventLine) {
	memoryBuffersMu.Lock()
	defer memoryBuffersMu.Unlock()
	memoryBuffers[streamPath] = append(memoryBuffers[streamPath], eventLine)
}

// GetMemoryBuffer returns the in-memory buffer for a stream path.
func GetMemoryBuffer(streamPath string) []EventLine {
	memoryBuffersMu.RLock()
	defer memoryBuffersMu.RUnlock()
	return memoryBuffers[streamPath]
}

// ClearMemoryBuffer removes the in-memory buffer for a stream path.
func ClearMemoryBuffer(streamPath string) {
	memoryBuffersMu.Lock()
	defer memoryBuffersMu.Unlock()
	delete(memoryBuffers, streamPath)
}

// WriteEventAsJSON is a convenience helper that marshals a raw map to an EventLine
// and writes it to the given path. Useful for quick instrumentation.
func WriteEventAsJSON(streamPath string, eventType EventType, payload map[string]interface{}) error {
	line := NewEventLine(eventType, payload)
	return WriteEvent(streamPath, line)
}

// WriteEventToMemoryAsJSON marshals a raw map to an EventLine and stores it in memory.
func WriteEventToMemoryAsJSON(streamPath string, eventType EventType, payload map[string]interface{}) {
	line := NewEventLine(eventType, payload)
	WriteEventToMemory(streamPath, line)
}
