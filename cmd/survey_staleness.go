package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// surveyStaleCommitThreshold is the number of commits since the last
// colonize beyond which the staleness notice switches from a quiet age line
// to a loud warning. This is a judgement call, not a measurement — it may
// need to move once real usage data shows how fast a codebase map actually
// goes wrong.
const surveyStaleCommitThreshold = 25

// surveyStalenessNotice reports how old the territory survey is relative to
// the current git HEAD, so a worker grounding on the survey knows whether it
// is reading a fresh map or a stale one (D-10). It loads active colony state
// internally and returns "" on any failure — the same defensive posture
// resolveSurveySection already uses. A staleness helper that can break a
// build is worse than one that stays quiet. This never triggers a refresh;
// the map only refreshes when the user runs /ant-colonize.
func surveyStalenessNotice() string {
	if store == nil {
		return ""
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return ""
	}

	if state.TerritorySurveyed == nil || strings.TrimSpace(*state.TerritorySurveyed) == "" {
		return "_Territory has never been surveyed. Run `/ant-colonize` to map the codebase before grounding on file locations or structure claims._\n\n"
	}

	// The stored value comes from a state file a worker could have
	// influenced, and this is the only place in this phase where state text
	// reaches a subprocess argument (T-163-10). Validate it parses as
	// RFC3339 BEFORE touching git — a malformed value must never reach
	// exec.Command, so bail out here with no invocation at all.
	surveyedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(*state.TerritorySurveyed))
	if err != nil {
		return ""
	}

	root := filepath.Dir(filepath.Dir(store.BasePath()))
	count, ok := commitsSinceSurvey(root, surveyedAt)
	if !ok {
		return fmt.Sprintf("_Territory surveyed %s. Commit count since survey unavailable._\n\n", surveyedAt.Format("2006-01-02"))
	}

	if count >= surveyStaleCommitThreshold {
		return fmt.Sprintf(
			"_STALE MAP WARNING: territory surveyed %d commits ago (%s). The codebase map may no longer match the tree — run `/ant-colonize` to refresh before trusting file locations or structure claims._\n\n",
			count, surveyedAt.Format("2006-01-02"),
		)
	}
	return fmt.Sprintf("_Territory mapped %d commit(s) ago (%s)._\n\n", count, surveyedAt.Format("2006-01-02"))
}

// commitsSinceSurvey runs `git rev-list --count --since=<t> HEAD` in root and
// returns the count. ok is false whenever git cannot answer for any reason
// (not a git repo, git missing, timeout, malformed output) — the caller
// falls back to a quiet notice with no count rather than surfacing an error.
// The timestamp is always passed as a single, discrete argv element built
// with string concatenation, never interpolated into a shell string — there
// is no sh -c anywhere in this helper.
func commitsSinceSurvey(root string, since time.Time) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()

	sinceArg := "--since=" + since.UTC().Format(time.RFC3339)
	out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-list", "--count", sinceArg, "HEAD").Output()
	if err != nil {
		return 0, false
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, false
	}
	return count, true
}
