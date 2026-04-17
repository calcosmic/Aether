package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var entombCmd = &cobra.Command{
	Use:   "entomb",
	Short: "Archive a sealed colony into chambers and reset the active state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Backfill the legacy top-level session mirror when upgrading a repo that
		// still carries only the older colony-scoped session path.
		if _, err := ensureLegacySessionMirror(store); err != nil {
			outputError(2, fmt.Sprintf("failed to prepare session mirror: %v", err), nil)
			return nil
		}

		state, err := loadActiveColonyState()
		if err != nil {
			outputError(1, colonyStateLoadMessage(err), nil)
			return nil
		}
		if strings.TrimSpace(state.Milestone) != "Crowned Anthill" {
			outputError(1, fmt.Sprintf("Colony has not been sealed. Current milestone: %s. Run `aether seal` first.", emptyFallback(strings.TrimSpace(state.Milestone), "(none)")), nil)
			return nil
		}

		aetherRoot := resolveAetherRootPath()
		sealSummaryPath := filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md")
		if _, err := os.Stat(sealSummaryPath); err != nil {
			outputError(1, "CROWNED-ANTHILL.md not found. Run `aether seal` again before entombing.", nil)
			return nil
		}

		now := time.Now().UTC()
		goal := strings.TrimSpace(*state.Goal)
		chamberName := uniqueChamberName(filepath.Join(aetherRoot, ".aether", "chambers"), goal, now)
		chamberDir := filepath.Join(aetherRoot, ".aether", "chambers", chamberName)

		if err := os.MkdirAll(chamberDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create chamber directory: %v", err), nil)
			return nil
		}

		if err := writeEntombManifest(chamberDir, chamberName, state, now); err != nil {
			_ = os.RemoveAll(chamberDir)
			outputError(2, fmt.Sprintf("failed to write chamber manifest: %v", err), nil)
			return nil
		}

		if err := copyEntombArtifacts(aetherRoot, store.BasePath(), chamberDir); err != nil {
			_ = os.RemoveAll(chamberDir)
			outputError(2, fmt.Sprintf("failed to archive chamber artifacts: %v", err), nil)
			return nil
		}

		if err := exportArchiveXMLToFile(filepath.Join(chamberDir, "colony-archive.xml")); err != nil {
			_ = os.RemoveAll(chamberDir)
			outputError(2, fmt.Sprintf("failed to export colony archive XML: %v", err), nil)
			return nil
		}

		if err := verifyEntombedChamber(chamberDir); err != nil {
			outputError(2, fmt.Sprintf("chamber verification failed: %v", err), map[string]interface{}{"chamber": chamberDir})
			return nil
		}

		reset := resetColonyStateForEntomb(state)
		if err := store.SaveJSON("COLONY_STATE.json", reset); err != nil {
			outputError(2, fmt.Sprintf("failed to reset colony state: %v", err), nil)
			return nil
		}

		if err := clearActiveColonyRuntimeFiles(aetherRoot, store.BasePath()); err != nil {
			outputError(2, fmt.Sprintf("failed to clear active runtime files: %v", err), nil)
			return nil
		}

		if err := writeEntombRecoveryDocs(chamberName, goal, state, now); err != nil {
			outputError(2, fmt.Sprintf("failed to write entomb recovery docs: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"entombed":         true,
			"goal":             goal,
			"chamber":          chamberName,
			"chamber_path":     chamberDir,
			"phases_completed": completedPhaseCount(state),
			"total_phases":     len(state.Plan.Phases),
			"next":             `aether init "next goal"`,
		}
		outputWorkflow(result, renderEntombVisual(result))
		return nil
	},
}

var tunnelsCmd = &cobra.Command{
	Use:   "tunnels [chamber]",
	Short: "Browse archived chambers",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		aetherRoot := resolveAetherRootPath()
		chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")

		if len(args) == 0 {
			return chamberListCmd.RunE(cmd, args)
		}

		chamberName := strings.TrimSpace(args[0])
		chamberDir := filepath.Join(chambersDir, chamberName)
		manifestPath := filepath.Join(chamberDir, "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			outputError(1, fmt.Sprintf("chamber %q not found", chamberName), nil)
			return nil
		}

		var manifest map[string]interface{}
		if err := json.Unmarshal(data, &manifest); err != nil {
			outputError(1, fmt.Sprintf("chamber %q has invalid manifest", chamberName), nil)
			return nil
		}

		entries, _ := os.ReadDir(chamberDir)
		files := make([]string, 0, len(entries))
		for _, entry := range entries {
			files = append(files, entry.Name())
		}
		sort.Strings(files)

		sealSummary := ""
		if raw, err := os.ReadFile(filepath.Join(chamberDir, "CROWNED-ANTHILL.md")); err == nil {
			sealSummary = string(raw)
		}

		outputOK(map[string]interface{}{
			"chamber":      chamberName,
			"manifest":     manifest,
			"files":        files,
			"seal_summary": sealSummary,
		})
		return nil
	},
}

func uniqueChamberName(chambersRoot, goal string, now time.Time) string {
	prefix := now.Format("2006-01-02")
	slug := sanitizeChamberGoal(goal)
	name := prefix + "-" + slug
	candidate := name
	counter := 1
	for {
		if _, err := os.Stat(filepath.Join(chambersRoot, candidate)); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", name, counter)
		counter++
	}
}

func sanitizeChamberGoal(goal string) string {
	goal = strings.ToLower(strings.TrimSpace(goal))
	if goal == "" {
		return "colony"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range goal {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "colony"
	}
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	if slug == "" {
		return "colony"
	}
	return slug
}

func writeEntombManifest(chamberDir, chamberName string, state colony.ColonyState, now time.Time) error {
	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	manifest := map[string]interface{}{
		"name":             chamberName,
		"goal":             goal,
		"milestone":        state.Milestone,
		"phases_completed": completedPhaseCount(state),
		"total_phases":     len(state.Plan.Phases),
		"entombed_at":      now.Format(time.RFC3339),
		"colony_version":   state.ColonyVersion,
		"state":            state.State,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(chamberDir, "manifest.json"), append(data, '\n'), 0644)
}

func copyEntombArtifacts(aetherRoot, dataDir, chamberDir string) error {
	dataFiles := []string{
		"COLONY_STATE.json",
		"pheromones.json",
		"session.json",
		"activity.log",
		"flags.json",
		"constraints.json",
		"spawn-tree.txt",
		"timing.log",
		"view-state.json",
	}
	for _, name := range dataFiles {
		if err := copyIfExists(filepath.Join(dataDir, name), filepath.Join(chamberDir, name)); err != nil {
			return err
		}
	}

	rootFiles := []string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"),
		filepath.Join(aetherRoot, ".aether", "HANDOFF.md"),
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"),
	}
	for _, src := range rootFiles {
		if err := copyIfExists(src, filepath.Join(chamberDir, filepath.Base(src))); err != nil {
			return err
		}
	}

	if err := copyDirIfExists(filepath.Join(aetherRoot, ".aether", "dreams"), filepath.Join(chamberDir, "dreams")); err != nil {
		return err
	}
	if err := copyDirIfExists(filepath.Join(dataDir, "colonies"), filepath.Join(chamberDir, "colonies")); err != nil {
		return err
	}

	xmlMatches, _ := filepath.Glob(filepath.Join(aetherRoot, ".aether", "exchange", "*.xml"))
	for _, src := range xmlMatches {
		if err := copyIfExists(src, filepath.Join(chamberDir, filepath.Base(src))); err != nil {
			return err
		}
	}

	return nil
}

func copyIfExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDirIfExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyIfExists(path, target)
	})
}

func exportArchiveXMLToFile(outputPath string) error {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	oldStdout := stdout
	oldStderr := stderr
	stdout = &stdoutBuf
	stderr = &stderrBuf
	defer func() {
		stdout = oldStdout
		stderr = oldStderr
	}()

	tmpCmd := &cobra.Command{}
	tmpCmd.Flags().String("output", outputPath, "")
	if err := runExportArchive(tmpCmd, nil); err != nil {
		return err
	}
	if _, err := os.Stat(outputPath); err != nil {
		msg := strings.TrimSpace(stderrBuf.String())
		if msg == "" {
			msg = strings.TrimSpace(stdoutBuf.String())
		}
		if msg == "" {
			msg = errString(err)
		}
		return fmt.Errorf("%s", strings.TrimSpace(msg))
	}
	return nil
}

func verifyEntombedChamber(chamberDir string) error {
	required := []string{
		filepath.Join(chamberDir, "manifest.json"),
		filepath.Join(chamberDir, "COLONY_STATE.json"),
		filepath.Join(chamberDir, "CROWNED-ANTHILL.md"),
		filepath.Join(chamberDir, "colony-archive.xml"),
	}
	for _, requiredPath := range required {
		if _, err := os.Stat(requiredPath); err != nil {
			return fmt.Errorf("missing %s", filepath.Base(requiredPath))
		}
	}
	return nil
}

func resetColonyStateForEntomb(state colony.ColonyState) colony.ColonyState {
	state.Goal = nil
	state.ColonyVersion = 0
	state.State = colony.StateIDLE
	state.CurrentPhase = 0
	state.SessionID = nil
	state.InitializedAt = nil
	state.BuildStartedAt = nil
	state.Plan.GeneratedAt = nil
	state.Plan.Confidence = nil
	state.Plan.Phases = []colony.Phase{}
	state.Memory.PhaseLearnings = []colony.PhaseLearning{}
	state.Memory.Decisions = []colony.Decision{}
	state.Memory.Instincts = []colony.Instinct{}
	state.Errors.Records = []colony.ErrorRecord{}
	state.Errors.FlaggedPatterns = []colony.FlaggedPattern{}
	state.Signals = []colony.Signal{}
	state.Graveyards = []colony.Graveyard{}
	state.Events = []string{}
	state.Milestone = ""
	state.MilestoneUpdatedAt = nil
	state.Worktrees = nil
	state.TerritorySurveyed = nil
	return state
}

func clearActiveColonyRuntimeFiles(aetherRoot, dataDir string) error {
	toRemove := []string{
		filepath.Join(dataDir, "session.json"),
		filepath.Join(dataDir, "pheromones.json"),
		filepath.Join(dataDir, "activity.log"),
		filepath.Join(dataDir, "flags.json"),
		filepath.Join(dataDir, "constraints.json"),
		filepath.Join(dataDir, "spawn-tree.txt"),
		filepath.Join(dataDir, "timing.log"),
		filepath.Join(dataDir, "view-state.json"),
		filepath.Join(dataDir, ".version-check-cache"),
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"),
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"),
	}
	for _, path := range toRemove {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	xmlMatches, _ := filepath.Glob(filepath.Join(aetherRoot, ".aether", "exchange", "*.xml"))
	for _, path := range xmlMatches {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.RemoveAll(filepath.Join(dataDir, "colonies")); err != nil {
		return err
	}
	return nil
}

func writeEntombRecoveryDocs(chamberName, goal string, state colony.ColonyState, now time.Time) error {
	completed := completedPhaseCount(state)
	total := len(state.Plan.Phases)

	handoff := strings.Join([]string{
		"# Colony Session — " + chamberName,
		"",
		"## A Colony's Rest",
		"",
		"This colony has been entombed. Its work is complete and archived.",
		"",
		"**Chamber:** .aether/chambers/" + chamberName + "/",
		"",
		"## Colony Summary",
		"",
		"- Goal: \"" + goal + "\"",
		fmt.Sprintf("- Phases: %d completed of %d", completed, total),
		"- Milestone reached: Crowned Anthill",
		"- Entombed at: " + now.Format(time.RFC3339),
		"",
		"When you are ready to begin again:",
		"",
		"- Start fresh: `aether init \"new goal\"`",
		"- Browse archives: `aether tunnels`",
		"",
	}, "\n")
	if err := writeHandoffDocument(handoff); err != nil {
		return err
	}

	context := strings.Join([]string{
		"# Colony Context",
		"",
		"State: IDLE",
		"Latest chamber: .aether/chambers/" + chamberName + "/",
		"Previous goal: " + goal,
		"Recommended next action: `aether init \"new goal\"`",
		"",
	}, "\n")
	return writeContextDocument(context)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func renderEntombVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("⚰️", "Entomb"))
	b.WriteString(visualDivider)
	b.WriteString("Colony archived into chambers.\n")
	b.WriteString("Goal: ")
	b.WriteString(emptyFallback(stringValue(result["goal"]), "No goal recorded"))
	b.WriteString("\n")
	b.WriteString("Chamber: ")
	b.WriteString(emptyFallback(stringValue(result["chamber_path"]), stringValue(result["chamber"])))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Phases: %d/%d complete\n", intValue(result["phases_completed"]), intValue(result["total_phases"])))
	b.WriteString(renderNextUp(
		`Run `+"`aether init \"next goal\"`"+` to found the next colony.`,
		`Run `+"`aether tunnels`"+` to browse archived chambers.`,
	))
	return b.String()
}

func init() {
	rootCmd.AddCommand(entombCmd)
	rootCmd.AddCommand(tunnelsCmd)
}
