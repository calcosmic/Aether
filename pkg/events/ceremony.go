package events

import "encoding/json"

const (
	CeremonyTopicBuildPrewave   = "ceremony.build.prewave"
	CeremonyTopicBuildWaveStart = "ceremony.build.wave.start"
	CeremonyTopicBuildSpawn     = "ceremony.build.spawn"
	CeremonyTopicBuildToolUse   = "ceremony.build.tool_use"
	CeremonyTopicBuildWaveEnd   = "ceremony.build.wave.end"
	CeremonyTopicPheromoneEmit  = "ceremony.pheromone.emit"
	CeremonyTopicSkillActivate  = "ceremony.skill.activate"
	CeremonyTopicChamberSeal    = "ceremony.chamber.seal"
)

// CeremonyPayload is the shared event shape consumed by the bundled narrator.
// Fields are optional because different ceremony moments expose different facts.
type CeremonyPayload struct {
	Phase           int      `json:"phase,omitempty"`
	PhaseName       string   `json:"phase_name,omitempty"`
	Wave            int      `json:"wave,omitempty"`
	SpawnID         string   `json:"spawn_id,omitempty"`
	Caste           string   `json:"caste,omitempty"`
	Name            string   `json:"name,omitempty"`
	TaskID          string   `json:"task_id,omitempty"`
	Task            string   `json:"task,omitempty"`
	Status          string   `json:"status,omitempty"`
	Message         string   `json:"message,omitempty"`
	Skill           string   `json:"skill,omitempty"`
	PheromoneType   string   `json:"pheromone_type,omitempty"`
	Strength        float64  `json:"strength,omitempty"`
	Completed       int      `json:"completed,omitempty"`
	Total           int      `json:"total,omitempty"`
	ToolCount       int      `json:"tool_count,omitempty"`
	TokenCount      int      `json:"token_count,omitempty"`
	FilesCreated    []string `json:"files_created,omitempty"`
	FilesModified   []string `json:"files_modified,omitempty"`
	TestsWritten    []string `json:"tests_written,omitempty"`
	Blockers        []string `json:"blockers,omitempty"`
	SuccessCriteria []string `json:"success_criteria,omitempty"`
}

func (p CeremonyPayload) RawMessage() (json.RawMessage, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func CeremonyTopics() []string {
	return []string{
		CeremonyTopicBuildPrewave,
		CeremonyTopicBuildWaveStart,
		CeremonyTopicBuildSpawn,
		CeremonyTopicBuildToolUse,
		CeremonyTopicBuildWaveEnd,
		CeremonyTopicPheromoneEmit,
		CeremonyTopicSkillActivate,
		CeremonyTopicChamberSeal,
	}
}
