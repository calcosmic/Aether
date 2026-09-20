package cmd

// autopilotState is the durable state shared by the supported run and status
// commands. The retired autopilot-* adapters are deliberately absent; keeping
// this schema here preserves compatibility with state written by earlier
// releases and with the current invocation report.
type autopilotPhaseStatus struct {
	Phase  int    `json:"phase"`
	Status string `json:"status"`
	At     string `json:"at,omitempty"`
}

type autopilotState struct {
	SchemaVersion  int                        `json:"schema_version,omitempty"`
	InitializedAt  string                     `json:"initialized_at"`
	TotalPhases    int                        `json:"total_phases"`
	CurrentPhase   int                        `json:"current_phase"`
	Status         string                     `json:"status"`
	Reason         string                     `json:"reason"`
	Headless       bool                       `json:"headless"`
	ReplanInterval int                        `json:"replan_interval"`
	Phases         []autopilotPhaseStatus     `json:"phases"`
	LastUpdated    string                     `json:"last_updated"`
	LastReport     *autopilotInvocationReport `json:"last_report,omitempty"`
}

const (
	autopilotStatePath          = "autopilot/state.json"
	autopilotStateSchemaVersion = 2
)
