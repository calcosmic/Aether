// Package learn provides the learning store foundation for Aether's hive
// learning system. It defines the LearnStore interface, core types (Entry,
// Evidence, Classification), and the ColonyStore implementation for
// repo-isolated JSON persistence.
package learn

// Classification enum for learning entries (D-10, D-11).
type Classification string

const (
	ClassBlocked       Classification = "blocked"
	ClassRepoLocal     Classification = "repo-local"
	ClassHiveShareable Classification = "hive-shareable"
	ClassNeedsApproval Classification = "needs-user-approval"
)

// WorkerEvidence records a single worker's contribution (D-09).
type WorkerEvidence struct {
	Name   string `json:"name"`
	Caste  string `json:"caste"`
	Status string `json:"status"`
}

// Evidence carries full structured provenance for a learning entry (D-09).
type Evidence struct {
	RunID        string           `json:"run_id"`
	Phase        int              `json:"phase"`
	Workers      []WorkerEvidence `json:"workers"`
	FilesTouched []string         `json:"files_touched,omitempty"`
	GatesPassed  int              `json:"gates_passed"`
	GatesTotal   int              `json:"gates_total"`
	Confidence   float64          `json:"confidence"`
	Timestamp    string           `json:"timestamp"`
	Scope        string           `json:"scope"`
}

// Entry is a single durable learning record.
type Entry struct {
	ID             string        `json:"id"`
	Content        string        `json:"content"`
	Evidence       Evidence      `json:"evidence"`
	Classification Classification `json:"classification"`
	CreatedAt      string        `json:"created_at"`
	Phase          int           `json:"phase"`
	Caste          string        `json:"caste,omitempty"`
	FilePath       string        `json:"file_path,omitempty"`
	Confidence     float64       `json:"confidence"`
	Redacted       bool          `json:"redacted,omitempty"`
}

// EntryFilter for List queries.
type EntryFilter struct {
	Phase          int            `json:"phase,omitempty"`
	Classification Classification `json:"classification,omitempty"`
	MinConfidence  float64        `json:"min_confidence,omitempty"`
	Limit          int            `json:"limit,omitempty"`
}

// LearnStore interface (D-07) -- ColonyStore and HiveStore implement this.
type LearnStore interface {
	Add(entry Entry) error
	Get(id string) (*Entry, error)
	List(filter EntryFilter) ([]Entry, error)
	Replace(id string, entry Entry) error
	Remove(id string) error
	Compact(budget int) error
}
