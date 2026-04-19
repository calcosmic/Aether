package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const (
	behaviorObservationsFile = "behavior-observations.jsonl"
	profileFileName          = "profile.json"
)

var behavioralDimensionNames = []string{
	"communication_style",
	"decision_speed",
	"explanation_depth",
	"debugging_approach",
	"ux_philosophy",
	"vendor_philosophy",
	"frustration_triggers",
	"learning_style",
}

var behaviorObserveCmd = &cobra.Command{
	Use:   "behavior-observe",
	Short: "Append a behavioral observation for the active colony",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dimension := mustGetString(cmd, "dimension")
		if dimension == "" {
			return nil
		}
		dimension = strings.TrimSpace(dimension)
		if !isBehavioralDimension(dimension) {
			outputError(1, fmt.Sprintf("invalid dimension %q", dimension), map[string]interface{}{"valid_dimensions": behavioralDimensionNames})
			return nil
		}

		signal := mustGetString(cmd, "signal")
		if signal == "" {
			return nil
		}
		evidence := mustGetString(cmd, "evidence")
		if evidence == "" {
			return nil
		}
		strength, _ := cmd.Flags().GetFloat64("strength")
		if strength < 0 || strength > 1 {
			outputError(1, "--strength must be between 0.0 and 1.0", nil)
			return nil
		}
		commandName, _ := cmd.Flags().GetString("command")
		commandName = firstNonEmpty(strings.TrimSpace(commandName), "behavior-observe")

		observation := colony.BehaviorObservation{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			ColonyGoal: currentColonyGoal(),
			Command:    commandName,
			Dimension:  dimension,
			Signal:     strings.TrimSpace(signal),
			Strength:   strength,
			Evidence:   strings.TrimSpace(evidence),
		}

		if err := store.AppendJSONL(behaviorObservationsFile, observation); err != nil {
			outputError(2, fmt.Sprintf("failed to append behavior observation: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"logged":    true,
			"dimension": dimension,
			"signal":    observation.Signal,
			"strength":  strength,
			"path":      filepath.Join(store.BasePath(), behaviorObservationsFile),
		}
		outputWorkflow(result, renderBehaviorObserveVisual(result))
		return nil
	},
}

var profileReadCmd = &cobra.Command{
	Use:   "profile-read",
	Short: "Read the hub-level behavioral profile",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, path, err := loadUserProfile()
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		result := map[string]interface{}{
			"profile": profile,
			"path":    path,
		}
		outputWorkflow(result, renderProfileReadVisual(result))
		return nil
	},
}

var profileUpdateCmd = &cobra.Command{
	Use:   "profile-update",
	Short: "Consolidate observations into the hub profile and promote top profiled directives",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		result, err := runProfileUpdate()
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderProfileUpdateVisual(result))
		return nil
	},
}

func init() {
	behaviorObserveCmd.Flags().String("dimension", "", "Behavioral dimension to observe")
	behaviorObserveCmd.Flags().String("signal", "", "Observed signal text")
	behaviorObserveCmd.Flags().Float64("strength", 1.0, "Observation strength between 0.0 and 1.0")
	behaviorObserveCmd.Flags().String("evidence", "", "Concrete evidence for the observation")
	behaviorObserveCmd.Flags().String("command", "", "Optional command or workflow that produced the observation")

	rootCmd.AddCommand(behaviorObserveCmd)
	rootCmd.AddCommand(profileReadCmd)
	rootCmd.AddCommand(profileUpdateCmd)
}

func runProfileUpdate() (map[string]interface{}, error) {
	profile, profilePath, err := loadUserProfile()
	if err != nil {
		return nil, err
	}

	observations, err := loadBehaviorObservations()
	if err != nil {
		return nil, err
	}
	if len(observations) == 0 {
		return map[string]interface{}{
			"updated":           false,
			"profile":           profile,
			"path":              profilePath,
			"observation_count": 0,
			"promoted_count":    0,
			"promoted":          []string{},
			"next":              "Run `aether behavior-observe --dimension ... --signal ... --strength ... --evidence \"...\"` before updating the profile.",
		}, nil
	}

	updatedProfile := consolidateUserProfile(profile, observations)
	hub := hubStore()
	if hub == nil {
		return nil, fmt.Errorf("failed to initialize hub store")
	}
	if err := hub.SaveJSON(profileFileName, updatedProfile); err != nil {
		return nil, fmt.Errorf("failed to save profile.json: %w", err)
	}

	promoted, err := promoteProfileDirectives(updatedProfile.Directives)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"updated":           true,
		"profile":           updatedProfile,
		"path":              profilePath,
		"observation_count": len(observations),
		"promoted_count":    len(promoted),
		"promoted":          promoted,
		"next":              "Run `aether profile-read` to inspect the hub profile and `[profiled]` preferences now flowing through QUEEN.md.",
	}, nil
}

func loadUserProfile() (colony.UserProfile, string, error) {
	profile := defaultUserProfile()
	path := filepath.Join(resolveHubPath(), profileFileName)

	hub := hubStore()
	if hub == nil {
		return profile, path, fmt.Errorf("failed to initialize hub store")
	}
	if err := hub.LoadJSON(profileFileName, &profile); err != nil {
		return profile, path, nil
	}

	normalized := normalizeUserProfile(profile)
	if normalized.GeneratedAt == "" {
		normalized.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return normalized, path, nil
}

func loadBehaviorObservations() ([]colony.BehaviorObservation, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	lines, err := store.ReadJSONL(behaviorObservationsFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return []colony.BehaviorObservation{}, nil
		}
		return nil, fmt.Errorf("failed to read behavior observations: %w", err)
	}

	observations := make([]colony.BehaviorObservation, 0, len(lines))
	for _, line := range lines {
		var obs colony.BehaviorObservation
		if err := json.Unmarshal(line, &obs); err != nil {
			continue
		}
		if !isBehavioralDimension(obs.Dimension) {
			continue
		}
		if obs.Strength < 0 || obs.Strength > 1 {
			continue
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

func defaultUserProfile() colony.UserProfile {
	dimensions := make([]colony.BehavioralDimension, 0, len(behavioralDimensionNames))
	for _, name := range behavioralDimensionNames {
		dimensions = append(dimensions, colony.BehavioralDimension{Name: name, Score: 0, Evidence: []string{}, SampleCount: 0})
	}
	return colony.UserProfile{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ColonyCount: 0,
		Dimensions:  dimensions,
		Directives:  []string{},
	}
}

func normalizeUserProfile(profile colony.UserProfile) colony.UserProfile {
	if profile.Version == "" {
		profile.Version = "1.0"
	}
	if profile.Directives == nil {
		profile.Directives = []string{}
	}
	byName := map[string]colony.BehavioralDimension{}
	for _, dimension := range profile.Dimensions {
		if isBehavioralDimension(dimension.Name) {
			if dimension.Evidence == nil {
				dimension.Evidence = []string{}
			}
			byName[dimension.Name] = dimension
		}
	}

	dimensions := make([]colony.BehavioralDimension, 0, len(behavioralDimensionNames))
	for _, name := range behavioralDimensionNames {
		dimension, ok := byName[name]
		if !ok {
			dimension = colony.BehavioralDimension{Name: name, Score: 0, Evidence: []string{}, SampleCount: 0}
		}
		dimensions = append(dimensions, dimension)
	}
	profile.Dimensions = dimensions
	return profile
}

func consolidateUserProfile(existing colony.UserProfile, observations []colony.BehaviorObservation) colony.UserProfile {
	existing = normalizeUserProfile(existing)
	existingByName := map[string]colony.BehavioralDimension{}
	for _, dimension := range existing.Dimensions {
		existingByName[dimension.Name] = dimension
	}

	type dimensionAggregate struct {
		totalStrength float64
		sampleCount   int
		evidence      []string
		updatedAt     string
	}
	aggregates := map[string]*dimensionAggregate{}
	directiveScores := map[string]float64{}
	colonyGoals := map[string]struct{}{}

	for _, obs := range observations {
		if !isBehavioralDimension(obs.Dimension) {
			continue
		}
		aggregate := aggregates[obs.Dimension]
		if aggregate == nil {
			aggregate = &dimensionAggregate{evidence: []string{}}
			aggregates[obs.Dimension] = aggregate
		}
		aggregate.totalStrength += obs.Strength
		aggregate.sampleCount++
		aggregate.evidence = appendUniqueStrings(aggregate.evidence, formatObservationEvidence(obs))
		if isLaterProfileTimestamp(obs.Timestamp, aggregate.updatedAt) {
			aggregate.updatedAt = obs.Timestamp
		}

		if goal := strings.TrimSpace(obs.ColonyGoal); goal != "" {
			colonyGoals[goal] = struct{}{}
		}
		directive := ensureProfiledDirective(obs.Signal)
		directiveScores[directive] += obs.Strength
	}

	dimensions := make([]colony.BehavioralDimension, 0, len(behavioralDimensionNames))
	for _, name := range behavioralDimensionNames {
		current := existingByName[name]
		aggregate := aggregates[name]
		if aggregate == nil {
			dimensions = append(dimensions, current)
			continue
		}

		currentAverage := aggregate.totalStrength / float64(aggregate.sampleCount)
		totalSamples := current.SampleCount + aggregate.sampleCount
		score := currentAverage
		if current.SampleCount > 0 && totalSamples > 0 {
			score = ((current.Score * float64(current.SampleCount)) + aggregate.totalStrength) / float64(totalSamples)
		}
		dimensions = append(dimensions, colony.BehavioralDimension{
			Name:        name,
			Score:       clampUnit(score),
			Evidence:    mergeEvidence(current.Evidence, aggregate.evidence),
			UpdatedAt:   latestProfileTimestamp(current.UpdatedAt, aggregate.updatedAt),
			SampleCount: totalSamples,
		})
	}

	directives := topProfiledDirectives(directiveScores, existing.Directives, 3)
	colonyCount := existing.ColonyCount
	if len(colonyGoals) > colonyCount {
		colonyCount = len(colonyGoals)
	}

	return colony.UserProfile{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ColonyCount: colonyCount,
		Dimensions:  dimensions,
		Directives:  directives,
	}
}

func promoteProfileDirectives(directives []string) ([]string, error) {
	if len(directives) == 0 {
		return nil, nil
	}
	hub := hubStore()
	if hub == nil {
		return nil, fmt.Errorf("failed to initialize hub store")
	}

	text, _, err := loadQueenText(hub)
	if err != nil {
		return nil, fmt.Errorf("failed to load QUEEN.md: %w", err)
	}

	promoted := []string{}
	for _, directive := range directives {
		directive = ensureProfiledDirective(directive)
		if strings.Contains(text, sanitizeQueenInline(directive)) {
			continue
		}
		if err := promoteQueenPreferenceDirective(directive); err != nil {
			return promoted, err
		}
		promoted = append(promoted, directive)

		text, _, err = loadQueenText(hub)
		if err != nil {
			return promoted, fmt.Errorf("failed to reload QUEEN.md: %w", err)
		}
	}
	return promoted, nil
}

func promoteQueenPreferenceDirective(directive string) error {
	tmpCmd := &cobra.Command{}
	tmpCmd.Flags().String("section", "User Preferences", "")
	tmpCmd.Flags().String("content", directive, "")

	var outBuf strings.Builder
	var errBuf strings.Builder
	oldStdout := stdout
	oldStderr := stderr
	stdout = &outBuf
	stderr = &errBuf
	defer func() {
		stdout = oldStdout
		stderr = oldStderr
	}()

	if err := queenPromoteCmd.RunE(tmpCmd, nil); err != nil {
		return err
	}
	if strings.TrimSpace(errBuf.String()) != "" {
		var envelope map[string]interface{}
		if json.Unmarshal([]byte(strings.TrimSpace(errBuf.String())), &envelope) == nil {
			return fmt.Errorf("%s", stringValue(envelope["error"]))
		}
		return fmt.Errorf("%s", strings.TrimSpace(errBuf.String()))
	}
	if strings.TrimSpace(outBuf.String()) == "" {
		return fmt.Errorf("queen-promote produced no output")
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(outBuf.String())), &envelope); err != nil {
		return fmt.Errorf("failed to parse queen-promote output: %w", err)
	}
	if ok, _ := envelope["ok"].(bool); !ok {
		return fmt.Errorf("queen-promote failed")
	}
	return nil
}

func topProfiledDirectives(current map[string]float64, existing []string, limit int) []string {
	type candidate struct {
		text  string
		score float64
	}
	candidates := make([]candidate, 0, len(current))
	for directive, score := range current {
		candidates = append(candidates, candidate{text: ensureProfiledDirective(directive), score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].text < candidates[j].text
	})

	directives := []string{}
	for _, candidate := range candidates {
		if len(directives) >= limit {
			break
		}
		directives = appendUniqueStrings(directives, candidate.text)
	}
	for _, directive := range existing {
		if len(directives) >= limit {
			break
		}
		directives = appendUniqueStrings(directives, ensureProfiledDirective(directive))
	}
	return directives
}

func currentColonyGoal() string {
	if store == nil {
		return ""
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return ""
	}
	if state.Goal == nil {
		return ""
	}
	return strings.TrimSpace(*state.Goal)
}

func isBehavioralDimension(name string) bool {
	return slices.Contains(behavioralDimensionNames, strings.TrimSpace(name))
}

func ensureProfiledDirective(signal string) string {
	signal = sanitizeQueenInline(signal)
	if signal == "" {
		return "[profiled]"
	}
	if strings.HasPrefix(strings.ToLower(signal), "[profiled]") {
		return signal
	}
	return "[profiled] " + signal
}

func formatObservationEvidence(obs colony.BehaviorObservation) string {
	parts := []string{}
	if command := strings.TrimSpace(obs.Command); command != "" {
		parts = append(parts, command)
	}
	if evidence := strings.TrimSpace(obs.Evidence); evidence != "" {
		parts = append(parts, evidence)
	}
	return strings.Join(parts, ": ")
}

func appendUniqueStrings(existing []string, values ...string) []string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		found := false
		for _, current := range existing {
			if strings.EqualFold(strings.TrimSpace(current), value) {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, value)
		}
	}
	return existing
}

func mergeEvidence(existing, current []string) []string {
	merged := appendUniqueStrings([]string{}, existing...)
	merged = appendUniqueStrings(merged, current...)
	if len(merged) > 5 {
		return merged[:5]
	}
	return merged
}

func clampUnit(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func latestProfileTimestamp(left, right string) string {
	if isLaterProfileTimestamp(right, left) {
		return right
	}
	return left
}

func isLaterProfileTimestamp(candidate, baseline string) bool {
	candidateTime, candidateErr := time.Parse(time.RFC3339, strings.TrimSpace(candidate))
	baselineTime, baselineErr := time.Parse(time.RFC3339, strings.TrimSpace(baseline))
	switch {
	case candidateErr != nil:
		return false
	case baselineErr != nil:
		return true
	default:
		return candidateTime.After(baselineTime)
	}
}

func renderBehaviorObserveVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧠", "Behavior Observe"))
	b.WriteString(visualDivider)
	b.WriteString("Dimension: ")
	b.WriteString(stringValue(result["dimension"]))
	b.WriteString("\nSignal: ")
	b.WriteString(stringValue(result["signal"]))
	b.WriteString("\nStrength: ")
	if strength, ok := result["strength"].(float64); ok {
		b.WriteString(fmt.Sprintf("%.2f", strength))
	} else {
		b.WriteString("0.00")
	}
	b.WriteString("\n")
	b.WriteString(renderNextUp("Run `aether profile-update` to consolidate the latest observations into the hub profile."))
	return b.String()
}

func renderProfileReadVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧠", "Profile"))
	b.WriteString(visualDivider)
	profile, _ := result["profile"].(colony.UserProfile)
	b.WriteString(fmt.Sprintf("Colonies observed: %d\n", profile.ColonyCount))
	b.WriteString(fmt.Sprintf("Directives: %d\n\n", len(profile.Directives)))
	for _, dimension := range profile.Dimensions {
		if dimension.SampleCount == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s: %.2f (%d samples)\n", dimension.Name, dimension.Score, dimension.SampleCount))
	}
	if len(profile.Directives) > 0 {
		b.WriteString("\nDirectives\n")
		b.WriteString(renderIndentedList(profile.Directives))
	}
	b.WriteString(renderNextUp("Run `aether profile-update` after new observations if you want to refresh the hub profile."))
	return b.String()
}

func renderProfileUpdateVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧠", "Profile Update"))
	b.WriteString(visualDivider)
	b.WriteString(fmt.Sprintf("Observations consolidated: %d\n", intValue(result["observation_count"])))
	b.WriteString(fmt.Sprintf("Directives promoted: %d\n", intValue(result["promoted_count"])))
	if promoted := stringSliceValue(result["promoted"]); len(promoted) > 0 {
		b.WriteString("\nPromoted\n")
		b.WriteString(renderIndentedList(promoted))
	}
	b.WriteString(renderNextUp(stringValue(result["next"])))
	return b.String()
}
