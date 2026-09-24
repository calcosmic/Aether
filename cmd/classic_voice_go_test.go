package cmd

// Phase 209 plan 01, Task 3 -- `/ant-go`'s screen joins the shared voice
// corpus (classic_voice_corpus_test.go), one init in this screen's own
// file so no two plans edit the same registration file.

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// renderVoiceGoSmallScreen builds the result map the way runGoJob genuinely
// produces it for a completed small job -- the small route's own keys
// (route/route_reason) merged with runQuickJob's own result keys -- and
// passes it to the real renderGoVisual. TestGoSmallRouteRunsTheJobEndToEnd
// (cmd/go_cmd_test.go) is the test proving those are the keys production
// actually sets.
func renderVoiceGoSmallScreen(t *testing.T) string {
	t.Helper()
	result := map[string]interface{}{
		"mode":              "quick-job",
		"job":               "fix the typo in README.md",
		"route":             string(jobSizeRouteSmall),
		"route_reason":      "the sentence names 1 file(s) already in this project, small enough for one helper to finish in one pass",
		"worker_name":       "Forge-11",
		"status":            "completed",
		"summary":           "Fixed the spelling mistake in README.md.",
		"files":             []string{"README.md"},
		"work_outcome":      colony.WorkOutcomeSuccess,
		"checks_status":     quickChecksPassed,
		"changes_confirmed": true,
		"duration_ms":       int64(1400),
	}
	return renderGoVisual(result)
}

func init() {
	registerVoiceScreen("go-small", renderVoiceGoSmallScreen)
}
