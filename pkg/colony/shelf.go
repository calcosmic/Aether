package colony

import "time"

type ShelfStatus string

const (
	ShelfShelved   ShelfStatus = "shelved"
	ShelfPromoted  ShelfStatus = "promoted"
	ShelfDismissed ShelfStatus = "dismissed"
)

type ShelfCategory string

const (
	ShelfCategoryInstinct   ShelfCategory = "instinct"
	ShelfCategoryPheromone  ShelfCategory = "pheromone"
	ShelfCategoryUserNote   ShelfCategory = "user-note"
	ShelfCategoryRedirect   ShelfCategory = "redirect"
)

type ShelfEntry struct {
	ID           string        `json:"id"`
	Text         string        `json:"text"`
	Source       string        `json:"source"`        // "phase", "user", "colony"
	CreatedAt    string        `json:"created_at"`
	Category     ShelfCategory `json:"category"`
	Confidence   float64       `json:"confidence"`
	Tags         []string      `json:"tags"`
	PromotedTo   string        `json:"promoted_to"`   // colony goal or phase string
	Status       ShelfStatus   `json:"status"`
	AutoDetected bool          `json:"auto_detected"`
	SourcePhase  int           `json:"source_phase"`
	SourceColony string        `json:"source_colony"` // repo path or goal text
}

type ShelfFile struct {
	Version   string       `json:"version"`
	UpdatedAt string       `json:"updated_at"`
	Entries   []ShelfEntry `json:"entries"`
}

func NewShelfFile() ShelfFile {
	return ShelfFile{
		Version:   "1.0",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:   []ShelfEntry{},
	}
}
