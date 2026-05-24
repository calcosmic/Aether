package cmd

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"time"
)

// ReadEvents reads all NDJSON events from a file path.
// It returns a slice of EventLine or an error if the file cannot be read
// or any line fails to parse.
func ReadEvents(streamPath string) ([]EventLine, error) {
	f, err := os.Open(streamPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return readEventLines(f)
}

// readEventLines parses NDJSON from an io.Reader.
func readEventLines(r io.Reader) ([]EventLine, error) {
	var events []EventLine
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev EventLine
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// TailEvents yields new events as they arrive on the stream.
// In stub mode it polls the file at a fixed interval and yields only
// events that have not been seen yet.
//
// The caller should range over the returned channel. To stop tailing,
// close the done channel. The returned channel will be closed when
// done is closed or the file is removed.
func TailEvents(streamPath string, done <-chan struct{}) (<-chan EventLine, error) {
	// Ensure the file exists so we have a starting size.
	info, err := os.Stat(streamPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var startSize int64
	if info != nil {
		startSize = info.Size()
	}

	out := make(chan EventLine)
	go tailEventStream(streamPath, startSize, out, done)
	return out, nil
}

// tailEventStream is the goroutine behind TailEvents.
func tailEventStream(streamPath string, lastSize int64, out chan<- EventLine, done <-chan struct{}) {
	defer close(out)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			info, err := os.Stat(streamPath)
			if err != nil {
				if os.IsNotExist(err) {
					// File removed; stop tailing.
					return
				}
				continue
			}

			newSize := info.Size()
			if newSize <= lastSize {
				continue
			}

			f, err := os.Open(streamPath)
			if err != nil {
				continue
			}

			_, err = f.Seek(lastSize, io.SeekStart)
			if err != nil {
				f.Close()
				continue
			}

			events, err := readEventLines(f)
			f.Close()
			if err != nil {
				continue
			}

			for _, ev := range events {
				select {
				case out <- ev:
				case <-done:
					return
				}
			}

			lastSize = newSize
		}
	}
}

// ReadEventsFromMemory reads all events from the in-memory buffer for a stream path.
// This is the stub-mode equivalent of ReadEvents.
func ReadEventsFromMemory(streamPath string) []EventLine {
	return GetMemoryBuffer(streamPath)
}
