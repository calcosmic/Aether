package colony

import "encoding/json"

// FlagEntry represents a single pending decision in pending-decisions.json.
type FlagEntry struct {
	additionalFields map[string]json.RawMessage
	ID               string `json:"id"`
	Type             string `json:"type"`
	Description      string `json:"description"`
	Phase            *int   `json:"phase"`
	Source           string `json:"source"`
	CreatedAt        string `json:"created_at"`
	Resolved         bool   `json:"resolved"`
	ResolvedAt       string `json:"resolved_at,omitempty"`
	Resolution       string `json:"resolution,omitempty"`
	Acknowledged     bool   `json:"acknowledged,omitempty"`
	// AcknowledgedAt records WHEN a flag was parked (classic flag lifecycle:
	// acknowledge = "noted but continuing", issues and notes only — blockers
	// cannot be acknowledged, they must be resolved).
	AcknowledgedAt string `json:"acknowledged_at,omitempty"`
	// RecoveryCommand, when set, is the exact command the owner should run
	// to resolve this blocker -- used by entries that are not resolved
	// through `aether flag-resolve` (e.g. an owner-confirmation blocker,
	// which is computed live and never persisted to pending-decisions.json,
	// so `flag-resolve --id <its ID>` can never find it). Left empty for
	// ordinary persisted flags, which keep the ordinary `flag-resolve`
	// recovery path (WR-01, 193-REVIEW.md).
	RecoveryCommand string `json:"recovery_command,omitempty"`
	// AttemptID binds a worker-reported blocker to the exact build attempt
	// that produced it (cmd/memory_feed.go's recordDispatchWorkerOutcome,
	// CAP-003/CAP-004/CAP-051). Empty for every flag not originated by a
	// worker outcome -- ordinary manual flags, swarm escalations, and
	// autopilot checkpoints never set it. This is the ONLY place attempt
	// binding lives for a blocker: advancement (checkUnresolvedBlockerFlags),
	// status (readBlockerSnapshot), and closure (LifecycleFacts.Blockers)
	// all read pending-decisions.json directly, so a flag written here is
	// automatically visible to all three without any change to those
	// readers.
	AttemptID string `json:"attempt_id,omitempty"`
}

// FlagsFile represents the top-level pending-decisions.json file.
type FlagsFile struct {
	additionalFields map[string]json.RawMessage
	Version          string      `json:"version"`
	Decisions        []FlagEntry `json:"decisions"`
}

// Additional fields belong to the shared decision store, not the generic flag
// view. Preserve them for FlagsFile writes without exposing them when a caller
// serializes []FlagEntry for a status/list response.
func (f *FlagEntry) UnmarshalJSON(data []byte) error {
	type plain FlagEntry
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, key := range []string{"id", "type", "description", "phase", "source", "created_at", "resolved", "resolved_at", "resolution", "acknowledged", "acknowledged_at", "recovery_command", "attempt_id"} {
		delete(fields, key)
	}
	*f = FlagEntry(value)
	if len(fields) > 0 {
		f.additionalFields = fields
	}
	return nil
}

// HasAdditionalField allows a writer to recognize protected decision metadata
// without promoting that metadata into ordinary flag output.
func (f FlagEntry) HasAdditionalField(name string) bool {
	_, ok := f.additionalFields[name]
	return ok
}

func (f *FlagsFile) UnmarshalJSON(data []byte) error {
	type plain FlagsFile
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	delete(fields, "version")
	delete(fields, "decisions")
	*f = FlagsFile(value)
	if len(fields) > 0 {
		f.additionalFields = fields
	}
	return nil
}

func (f FlagsFile) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage, len(f.additionalFields)+2)
	for key, value := range f.additionalFields {
		fields[key] = value
	}
	version, err := json.Marshal(f.Version)
	if err != nil {
		return nil, err
	}
	fields["version"] = version
	var rows []json.RawMessage
	if f.Decisions != nil {
		rows = make([]json.RawMessage, 0, len(f.Decisions))
	}
	for _, flag := range f.Decisions {
		// Only this containing file restores unknown decision fields. The flag's
		// own JSON representation remains its historical public projection.
		raw, err := json.Marshal(flag)
		if err != nil {
			return nil, err
		}
		var row map[string]json.RawMessage
		if err = json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		for key, value := range flag.additionalFields {
			row[key] = value
		}
		raw, err = json.Marshal(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, raw)
	}
	encodedRows, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	fields["decisions"] = encodedRows
	return json.Marshal(fields)
}
