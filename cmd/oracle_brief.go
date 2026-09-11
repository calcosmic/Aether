package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The Oracle setup ritual.
//
// The wrapper has always described this ritual -- propose the settings, ask
// clarifying questions, gate on an approved brief -- but none of it had runtime
// existence. Nothing proposed the settings, nothing recorded the approved brief,
// and nothing stopped the loop starting without one, so whether the ritual
// happened at all depended on the model reading its own instructions. These
// commands give the ritual a runtime the wrapper calls and a test can fail.

const (
	// oracleCoreQuestionMaxChars keeps the approved question readable. The
	// generated questions this replaces ran past 300 characters by splicing a
	// truncated topic into a template, which is unanswerable as a question.
	oracleCoreQuestionMaxChars = 240
	oraclePendingBriefVersion  = "1.0"
	oraclePendingBriefFileName = "pending-brief.json"
)

type oraclePendingBrief struct {
	Version          string   `json:"version"`
	Topic            string   `json:"topic"`
	CoreQuestion     string   `json:"core_question"`
	Context          string   `json:"context,omitempty"`
	SuccessCriteria  []string `json:"success_criteria,omitempty"`
	Template         string   `json:"template"`
	Depth            string   `json:"depth"`
	Scope            string   `json:"scope"`
	TargetConfidence int      `json:"target_confidence"`
	MaxIterations    int      `json:"max_iterations,omitempty"`
	ApprovedAt       string   `json:"approved_at"`
}

type oracleBriefOptions struct {
	Topic            string
	CoreQuestion     string
	Context          string
	SuccessCriteria  []string
	Template         string
	Depth            string
	Scope            string
	TargetConfidence int
	MaxIterations    int
}

func oraclePendingBriefPath(root string) string {
	return filepath.Join(oracleWorkspacePaths(root).Dir, oraclePendingBriefFileName)
}

// validateOracleCoreQuestion is what makes the brief gate an actual gate. A
// core question that is empty, is not a question, or runs past the readable
// limit is exactly the malformed input the loop then spends iterations on.
func validateOracleCoreQuestion(question string) error {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(question)), " ")
	if trimmed == "" {
		return fmt.Errorf("--core-question is required: the brief gate exists to stop the loop running on an unscoped topic")
	}
	if !strings.Contains(trimmed, "?") {
		return fmt.Errorf("--core-question must be a genuine question (no question mark found): %q", truncateString(trimmed, 80))
	}
	if len([]rune(trimmed)) > oracleCoreQuestionMaxChars {
		return fmt.Errorf("--core-question is %d characters; keep it under %d so the loop targets one decision", len([]rune(trimmed)), oracleCoreQuestionMaxChars)
	}
	return nil
}

// --- propose -----------------------------------------------------------

type oracleDepthOption struct {
	Value            string `json:"value"`
	Label            string `json:"label"`
	MaxIterations    int    `json:"max_iterations"`
	TargetConfidence int    `json:"default_target_confidence"`
	Description      string `json:"description"`
	TypicalMinutes   int    `json:"typical_minutes"`
	MaxMinutes       int    `json:"max_minutes"`
	Recommended      bool   `json:"recommended,omitempty"`
}

type oracleConfidenceOption struct {
	Value       int    `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended,omitempty"`
}

type oracleScopingOption struct {
	Value       int    `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended,omitempty"`
}

// oracleDepthOptionOrder keeps the presented order stable and meaningful --
// cheapest first -- matching the four shared preset names planning already
// established (Fast, Balanced, Deep, Exhaustive), rather than Oracle's own
// legacy depth words or map iteration order.
var oracleDepthOptionOrder = []string{
	string(planningStagePresetFast),
	string(planningStagePresetBalanced),
	string(planningStagePresetDeep),
	string(planningStagePresetExhaustive),
}

// oracleDepthOptions renders the four shared presets, each carrying its own
// confidence target and round cap -- Oracle's own numbers, never planning's.
// `recommended` may be a shared name or any legacy word oracleSuggestedDepth
// returns; both resolve through resolveOraclePreset onto the row they mark.
func oracleDepthOptions(recommended string) []oracleDepthOption {
	recommendedPreset, recErr := resolveOraclePreset(recommended)
	options := make([]oracleDepthOption, 0, len(oracleDepthOptionOrder))
	for _, id := range oracleDepthOptionOrder {
		policy, err := resolveOraclePreset(id)
		if err != nil {
			continue
		}
		options = append(options, oracleDepthOption{
			Value:            string(policy.ID),
			Label:            policy.Label,
			MaxIterations:    policy.RoundCap,
			TargetConfidence: policy.TargetConfidence,
			Description:      fmt.Sprintf("%s research, up to %d rounds", policy.Label, policy.RoundCap),
			// Estimates come from the real per-attempt watchdogs in
			// defaultOracleAttemptPolicy: 3-6 minutes each, and workers
			// usually return well before the ceiling.
			TypicalMinutes: policy.RoundCap * 2,
			MaxMinutes:     policy.RoundCap * 5,
			Recommended:    recErr == nil && policy.ID == recommendedPreset.ID,
		})
	}
	return options
}

func oracleConfidenceOptions() []oracleConfidenceOption {
	return []oracleConfidenceOption{
		{Value: 80, Label: "80% -- first pass", Description: "Good enough to decide a direction."},
		{Value: 90, Label: "90% -- solid", Description: "Solid understanding, some gaps remain."},
		{Value: 95, Label: "95% -- thorough", Description: "Few gaps remaining.", Recommended: true},
		{Value: 99, Label: "99% -- near-exhaustive", Description: "Converges only when almost nothing is left open."},
	}
}

func oracleScopingOptions(recommended int) []oracleScopingOption {
	options := []oracleScopingOption{
		{Value: 0, Label: "Skip -- I know exactly what I want", Description: "Go straight to the brief."},
		{Value: 3, Label: "Quick", Description: "Three questions on the decision, the output shape, and what done looks like."},
		{Value: 6, Label: "Standard", Description: "Six questions, also covering constraints, non-goals, and evidence sources."},
		{Value: 10, Label: "Thorough", Description: "Ten questions, for a broad or unfamiliar topic."},
	}
	for i := range options {
		options[i].Recommended = options[i].Value == recommended
	}
	return options
}

// oracleVaguenessScore rates how much scoping a raw topic needs, 0 (precise)
// to 100 (needs a real conversation first). It is a heuristic and is always
// surfaced as a labelled suggestion, never applied silently.
func oracleVaguenessScore(topic string) (int, []string) {
	text := strings.Join(strings.Fields(strings.TrimSpace(topic)), " ")
	lower := strings.ToLower(text)
	reasons := make([]string, 0, 4)
	if text == "" {
		return 100, []string{"no topic given"}
	}

	score := 0
	words := len(strings.Fields(text))
	switch {
	case words <= 4:
		score += 40
		reasons = append(reasons, "very short topic")
	case words <= 10:
		score += 25
		reasons = append(reasons, "short topic")
	case words >= 120:
		// A wall of text is its own kind of unscoped: it usually bundles
		// several unrelated asks into one run.
		score += 30
		reasons = append(reasons, "very long topic, likely several questions bundled together")
	}

	if !strings.Contains(text, "?") {
		score += 15
		reasons = append(reasons, "no explicit question")
	}

	broad := []string{"everything", "all of", "overall", "in general", "look into", "have a look",
		"review", "improve", "better", "modernize", "clean up", "tidy", "sort out", "figure out",
		"comprehensive", "full audit", "anything"}
	for _, needle := range broad {
		if strings.Contains(lower, needle) {
			score += 20
			reasons = append(reasons, fmt.Sprintf("broad wording (%q)", needle))
			break
		}
	}

	// Several distinct asks joined together rarely converge as one run.
	conjunctions := strings.Count(lower, " and ") + strings.Count(text, ";") + strings.Count(text, ",")
	if conjunctions >= 6 {
		score += 20
		reasons = append(reasons, "many separate asks in one topic")
	}

	if score > 100 {
		score = 100
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "topic reads as a single scoped question")
	}
	return score, reasons
}

func oracleSuggestedQuestionCount(score int) int {
	switch {
	case score >= 60:
		return 10
	case score >= 35:
		return 6
	case score >= 15:
		return 3
	default:
		return 0
	}
}

func oracleSuggestedDepth(score int, template string) string {
	// A bug hunt converges faster than an architecture review; a vague topic
	// needs more rounds regardless.
	switch strings.ToLower(strings.TrimSpace(template)) {
	case "bug-investigation":
		if score >= 60 {
			return "balanced"
		}
		return "quick"
	case "architecture-review", "prd":
		return "deep"
	}
	if score >= 50 {
		return "deep"
	}
	return "balanced"
}

// runOraclePropose is read-only. It never touches the workspace, so it is safe
// to call before the user has committed to anything.
func runOraclePropose(root, topic string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(topic)
	if trimmed == "" {
		return nil, fmt.Errorf("--topic is required: pass the rough topic and Oracle will propose how to scope it")
	}

	template := inferOracleTemplate(trimmed)
	scope := inferOracleAutoScope(trimmed)
	score, reasons := oracleVaguenessScore(trimmed)
	questionCount := oracleSuggestedQuestionCount(score)
	// oracleSuggestedDepth's recommendation logic is unchanged -- it still
	// returns a legacy word -- but the result is translated through
	// resolveOraclePreset so the recommended row in depth_options is marked
	// under its shared name rather than the legacy word.
	depth := oracleSuggestedDepth(score, template)
	preset, presetErr := resolveOraclePreset(depth)
	if presetErr != nil {
		preset, _ = resolveOraclePreset(string(planningStagePresetBalanced))
	}

	return map[string]interface{}{
		"topic":          trimmed,
		"topic_headline": oracleTopicHeadline(trimmed),
		"suggested": map[string]interface{}{
			// Every value here is a suggestion derived from keywords. The
			// wrapper must present them as pre-selected choices the user can
			// override, never apply them silently.
			"template":                   template,
			"scope":                      scope,
			"depth":                      string(preset.ID),
			"target_confidence":          preset.TargetConfidence,
			"clarifying_questions":       questionCount,
			"basis":                      "keyword heuristics over the raw topic",
			"vagueness_score":            score,
			"vagueness_reasons":          reasons,
			"present_as_suggestion_only": true,
		},
		"depth_options":      oracleDepthOptions(string(preset.ID)),
		"confidence_options": oracleConfidenceOptions(),
		"scoping_options":    oracleScopingOptions(questionCount),
		"next":               "aether oracle brief --core-question \"...\" --depth <depth> --confidence <percent>",
	}, nil
}

func renderOraclePropose(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🔮🐜", "Research Setup"))
	b.WriteString(visualDividerStr())

	if headline, _ := result["topic_headline"].(string); headline != "" {
		fmt.Fprintf(&b, "Topic: %s\n\n", headline)
	}

	suggested, _ := result["suggested"].(map[string]interface{})
	if suggested != nil {
		b.WriteString("Suggested (from keywords -- confirm or change):\n")
		fmt.Fprintf(&b, "  Output shape:        %v\n", suggested["template"])
		fmt.Fprintf(&b, "  Evidence sources:    %v\n", suggested["scope"])
		depthLabel := fmt.Sprintf("%v", suggested["depth"])
		if preset, err := resolveOraclePreset(depthLabel); err == nil {
			depthLabel = preset.Label
		}
		fmt.Fprintf(&b, "  Depth:               %s\n", depthLabel)
		fmt.Fprintf(&b, "  Target accuracy:     %v%%\n", suggested["target_confidence"])
		fmt.Fprintf(&b, "  Scoping questions:   %v\n", suggested["clarifying_questions"])
		if reasons, ok := suggested["vagueness_reasons"].([]string); ok && len(reasons) > 0 {
			fmt.Fprintf(&b, "  Because:             %s\n", strings.Join(reasons, "; "))
		}
		b.WriteString("\n")
	}

	if options, ok := result["depth_options"].([]oracleDepthOption); ok {
		b.WriteString("Depth options:\n")
		for _, option := range options {
			marker := " "
			if option.Recommended {
				marker = "*"
			}
			fmt.Fprintf(&b, "  %s %-11s target %d%%, up to %2d rounds, ~%d min (max %d min)\n",
				marker, option.Label, option.TargetConfidence, option.MaxIterations, option.TypicalMinutes, option.MaxMinutes)
		}
	}
	return strings.TrimSpace(b.String())
}

// --- brief -------------------------------------------------------------

func runOracleBriefApprove(root string, opts oracleBriefOptions, dryRun bool) (map[string]interface{}, error) {
	if err := validateOracleCoreQuestion(opts.CoreQuestion); err != nil {
		return nil, err
	}

	topic := strings.TrimSpace(opts.Topic)
	if topic == "" {
		// The core question is a legitimate topic on its own; requiring both
		// would just make the wrapper repeat itself.
		topic = strings.TrimSpace(opts.CoreQuestion)
	}

	// Infer from the topic and the core question together. The core question is
	// the sharper signal -- a topic of "cache storage" says nothing, while
	// "Should the local cache use SQLite or Postgres?" is plainly a comparison.
	inferenceText := strings.TrimSpace(topic + " " + strings.TrimSpace(opts.CoreQuestion))
	template, err := resolveOracleTemplate(inferenceText, opts.Template)
	if err != nil {
		return nil, err
	}
	scopeProfile, err := resolveOracleScope(inferenceText, opts.Scope)
	if err != nil {
		return nil, err
	}

	depthInput := strings.TrimSpace(opts.Depth)
	if depthInput == "" {
		depthInput = string(planningStagePresetBalanced)
	}
	// The picker this brief follows (oracleDepthOptions) now offers the
	// shared Fast/Balanced/Deep/Exhaustive names as its Value, and the
	// --depth flag's own help text advertises them too -- so the brief must
	// accept exactly what it was just shown, not only the legacy words.
	preset, err := resolveOraclePreset(depthInput)
	if err != nil {
		return nil, err
	}
	depth := string(preset.ID)

	target := opts.TargetConfidence
	if target == 0 {
		target = preset.TargetConfidence
	}
	if target < 1 || target > 100 {
		return nil, fmt.Errorf("--confidence must be between 1 and 100, got %d", target)
	}
	if opts.MaxIterations < 0 || opts.MaxIterations > 50 {
		return nil, fmt.Errorf("--max-iterations must be between 1 and 50, got %d", opts.MaxIterations)
	}

	criteria := make([]string, 0, len(opts.SuccessCriteria))
	for _, item := range opts.SuccessCriteria {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			criteria = append(criteria, trimmed)
		}
	}

	brief := oraclePendingBrief{
		Version:          oraclePendingBriefVersion,
		Topic:            topic,
		CoreQuestion:     strings.Join(strings.Fields(strings.TrimSpace(opts.CoreQuestion)), " "),
		Context:          strings.TrimSpace(opts.Context),
		SuccessCriteria:  criteria,
		Template:         template,
		Depth:            depth,
		Scope:            scopeProfile.Scope,
		TargetConfidence: target,
		MaxIterations:    opts.MaxIterations,
		ApprovedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	result := map[string]interface{}{
		"brief":    brief,
		"dry_run":  dryRun,
		"panel":    renderOracleBriefPanel(brief),
		"next":     "aether oracle --from-brief --background --follow",
		"approved": !dryRun,
	}
	if dryRun {
		result["brief_path"] = ""
		return result, nil
	}

	paths := oracleWorkspacePaths(root)
	if err := os.MkdirAll(paths.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create oracle workspace: %w", err)
	}
	encoded, err := json.MarshalIndent(brief, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal oracle brief: %w", err)
	}
	briefPath := oraclePendingBriefPath(root)
	if err := os.WriteFile(briefPath, append(encoded, '\n'), 0644); err != nil {
		return nil, fmt.Errorf("write oracle brief: %w", err)
	}
	rel, relErr := filepath.Rel(root, briefPath)
	if relErr != nil {
		rel = briefPath
	}
	result["brief_path"] = rel
	return result, nil
}

func renderOracleBriefPanel(brief oraclePendingBrief) string {
	var b strings.Builder
	b.WriteString(renderBanner("🔮", "Research Brief"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Core Question:    %s\n", brief.CoreQuestion)
	if brief.Context != "" {
		fmt.Fprintf(&b, "Context:          %s\n", brief.Context)
	}
	if len(brief.SuccessCriteria) > 0 {
		b.WriteString("Success Criteria:\n")
		for _, item := range brief.SuccessCriteria {
			fmt.Fprintf(&b, "  - %s\n", item)
		}
	}
	b.WriteString("\n")
	depthLabel := brief.Depth
	iterations := brief.MaxIterations
	if preset, err := resolveOraclePreset(brief.Depth); err == nil {
		depthLabel = preset.Label
		if iterations <= 0 {
			iterations = preset.RoundCap
		}
	} else if iterations <= 0 {
		// Defensive fallback for a brief written to disk before this preset
		// layer existed, or hand-edited to carry an unrecognized depth.
		iterations = resolveOracleDepth(brief.Depth).MaxIterations
	}
	fmt.Fprintf(&b, "Output shape: %s   Sources: %s\n", brief.Template, brief.Scope)
	fmt.Fprintf(&b, "Depth: %s (up to %d rounds)   Target: %d%%\n", depthLabel, iterations, brief.TargetConfidence)
	return strings.TrimSpace(b.String())
}

// loadOraclePendingBrief reads an approved brief without removing it.
func loadOraclePendingBrief(root string) (*oraclePendingBrief, error) {
	data, err := os.ReadFile(oraclePendingBriefPath(root))
	if err != nil {
		return nil, err
	}
	var brief oraclePendingBrief
	if err := json.Unmarshal(data, &brief); err != nil {
		return nil, fmt.Errorf("parse oracle brief: %w", err)
	}
	return &brief, nil
}

// approvedBriefForTopic returns the pending brief only when it was approved for
// this exact topic. Briefs live in the workspace until the next run archives
// them, so without this check a brief approved for one question would silently
// attach itself to whatever ran next.
func approvedBriefForTopic(root, topic string) *oraclePendingBrief {
	brief, err := loadOraclePendingBrief(root)
	if err != nil || brief == nil {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(brief.Topic), strings.TrimSpace(topic)) {
		return nil
	}
	return brief
}

// resolveOracleBriefRun backs `aether oracle --from-brief`. It is the gate that
// makes the setup ritual enforceable: without an approved brief on disk there is
// nothing to run, and saying so is the failure the Definition of Done asks for.
func resolveOracleBriefRun(root string) (*oraclePendingBrief, error) {
	brief, err := loadOraclePendingBrief(root)
	if err != nil {
		return nil, fmt.Errorf("no approved research brief found: run `aether oracle propose` and `aether oracle brief` first, or pass the topic directly to `aether oracle \"<topic>\"`")
	}
	if strings.TrimSpace(brief.CoreQuestion) == "" {
		return nil, fmt.Errorf("the approved research brief has no core question: re-run `aether oracle brief --core-question \"...\"`")
	}
	return brief, nil
}
