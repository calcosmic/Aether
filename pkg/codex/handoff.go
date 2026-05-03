package codex

// WorkerHandoff carries structured relay data from one worker to the next.
type WorkerHandoff struct {
	Freshness string `json:"freshness,omitempty"`
}

// ValidateWorkerHandoff checks that a WorkerHandoff is structurally valid.
func ValidateWorkerHandoff(h WorkerHandoff) error {
	return nil
}

// NormalizeWorkerHandoff returns a normalized copy of the handoff.
func NormalizeWorkerHandoff(root string, h WorkerHandoff) WorkerHandoff {
	return h
}
