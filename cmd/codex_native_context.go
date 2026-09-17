package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/codex"
)

type codexNativePrompt struct {
	Prompt      string
	SHA256      string
	DecisionIDs []string
}

// renderCodexNativeContextAnswers is the scope hook for Plan 04's protected
// native-question renderer. Ordinary answers still use the one current-scope
// resolver and integrity/budget renderer. Filter delivered IDs BEFORE packing
// so older answers cannot crowd out newly applicable ones.
func renderCodexNativeContextAnswers(manifest codexBuildManifest, dispatch codexBuildDispatch, launch, child string, excluded []string) clarifiedIntentRenderResult {
	file, _ := loadScopedPendingDecisionFile(loadCurrentPendingDecisionScope())
	entries := resolvedClarifiedIntentEntries(file)
	seen := make(map[string]bool, len(excluded))
	for _, id := range excluded {
		seen[id] = true
	}
	pending := entries[:0]
	for _, entry := range entries {
		if !seen[entry.ID] {
			pending = append(pending, entry)
		}
	}
	source := pendingDecisionsFile
	if store != nil {
		source = filepath.Join(store.BasePath(), pendingDecisionsFile)
	}
	return renderClarifiedIntentPromptEntriesWithIntegrity(pending, source)
}

func codexNativeAnswerSection(result clarifiedIntentRenderResult) string {
	if len(result.Lines) == 0 {
		return ""
	}
	return "## CLARIFIED INTENT\n\n" + strings.Join(result.Lines, "\n")
}

func composeCodexNativePrompt(manifest codexBuildManifest, dispatch codexBuildDispatch, launch string) (codexNativePrompt, error) {
	brief := dispatch.Brief
	if dispatch.BriefPath != "" {
		path := filepath.Join(manifest.Root, filepath.FromSlash(dispatch.BriefPath))
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return codexNativePrompt{}, fmt.Errorf("read native brief: %w", err)
		}
		root, err := filepath.EvalSymlinks(manifest.Root)
		if err != nil {
			return codexNativePrompt{}, err
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return codexNativePrompt{}, fmt.Errorf("native brief is outside accepted workspace")
		}
		raw, err := os.ReadFile(resolved)
		if err != nil {
			return codexNativePrompt{}, err
		}
		brief = string(raw)
		if dispatch.BriefSHA256 == "" || lifecycleDigest(raw) != dispatch.BriefSHA256 || (dispatch.Brief != "" && dispatch.Brief != brief) {
			return codexNativePrompt{}, fmt.Errorf("native brief bytes do not match the accepted manifest identity")
		}
	}
	if strings.TrimSpace(brief) == "" {
		return codexNativePrompt{}, fmt.Errorf("native assignment has no required brief")
	}
	if dispatch.BriefSHA256 != "" && dispatch.BriefSHA256 != lifecycleDigest([]byte(brief)) {
		return codexNativePrompt{}, fmt.Errorf("native brief digest does not match accepted bytes")
	}
	prompt := fmt.Sprintf("You are %s (%s), assigned workspace %s. You are not alone; preserve others' edits.\nWAIT: do not read files, run checks or edit until the parent sends AETHER_NATIVE_RELEASE %s with your actual child ID and the runtime release token after binding. If no release arrives, remain waiting; do not do the job.\n", dispatch.Name, dispatch.Caste, manifest.Root, launch)
	prompt += manifest.ContextCapsule + "\n\n" + brief + "\n\n" + dispatch.SkillSection
	answers := renderCodexNativeContextAnswers(manifest, dispatch, launch, "", manifest.ContextDecisionIDs)
	if len(answers.Lines) > 0 {
		prompt += "\n\n" + codexNativeAnswerSection(answers)
	}
	prompt += fmt.Sprintf("\nReturn one JSON terminal result: name or ant_name=%q (if both are supplied they must agree), caste=%q, task_id=%q, status (code_written/completed/failed/blocked/timeout), summary, files_created, files_modified, tests_written, task_receipts, blockers, spawns, handoff; optional installed Builder tdd fields cycles_completed/tests_added/coverage_percent/all_passing are preserved without becoming provider telemetry. Each task receipt has task_id, status, summary, files_created, files_modified, tests_written, handoff. Every handoff uses %s. verification_status MUST be one enum value: pass, fail, partial, not_run, or unknown; put explanations in summary/known_failures, never in verification_status. Report only actual checks; omit usage. Do not stage, finalize, commit, recruit or launch helpers. Parent saves your terminal response.\n", dispatch.Name, dispatch.Caste, normalizedDispatchTaskID(dispatch), codex.HandoffFieldsSummary)
	if !utf8.ValidString(prompt) {
		return codexNativePrompt{}, fmt.Errorf("native prompt must contain valid UTF-8 bytes")
	}
	ids := append([]string(nil), manifest.ContextDecisionIDs...)
	ids = append(ids, answers.DecisionIDs...)
	return codexNativePrompt{Prompt: prompt, SHA256: lifecycleDigest([]byte(prompt)), DecisionIDs: ids}, nil
}
