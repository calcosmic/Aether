package colony

// AssumptionConfidence expresses how solid an assumption is before execution.
type AssumptionConfidence string

const (
	AssumptionConfidenceConfident AssumptionConfidence = "confident"
	AssumptionConfidenceLikely    AssumptionConfidence = "likely"
	AssumptionConfidenceUnclear   AssumptionConfidence = "unclear"
)

// Valid reports whether the confidence value is recognized.
func (c AssumptionConfidence) Valid() bool {
	switch c {
	case AssumptionConfidenceConfident, AssumptionConfidenceLikely, AssumptionConfidenceUnclear:
		return true
	default:
		return false
	}
}

// Assumption captures an implicit premise in the current plan.
type Assumption struct {
	ID             string               `json:"id"`
	Phase          int                  `json:"phase"`
	Category       string               `json:"category"`
	AssumptionText string               `json:"assumption_text"`
	Evidence       []string             `json:"evidence,omitempty"`
	FilePath       string               `json:"file_path,omitempty"`
	Confidence     AssumptionConfidence `json:"confidence"`
	IfWrong        string               `json:"if_wrong,omitempty"`
	ResearchFlag   bool                 `json:"research_flag,omitempty"`
	Validated      bool                 `json:"validated,omitempty"`
	ValidationNote string               `json:"validation_note,omitempty"`
	ValidatedAt    string               `json:"validated_at,omitempty"`
	CreatedAt      string               `json:"created_at"`
}

// AssumptionsFile is the persisted assumptions.json payload.
type AssumptionsFile struct {
	Version     string       `json:"version"`
	GeneratedAt string       `json:"generated_at"`
	Goal        string       `json:"goal"`
	Assumptions []Assumption `json:"assumptions"`
}
