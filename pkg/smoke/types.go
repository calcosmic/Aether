package smoke

// FlagMismatch records a discrepancy between documented and registered flags.
type FlagMismatch struct {
	Command  string `json:"command"`
	Flag     string `json:"flag"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Severity string `json:"severity"` // "blocking" or "warning"
}
