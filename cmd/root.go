// Package cmd implements the Aether CLI commands using Cobra.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
)

// Version is set via -ldflags at build time.
var Version = "0.0.0-dev"

// resolveVersion returns the best available version in priority order:
// 1. ldflags Version (set by goreleaser for release builds)
// 2. Repo `.aether/version.json` when running from a source checkout
// 3. Nearest git tag from the given directory (for dev builds)
// 4. Installed hub version
// 5. Fallback "0.0.0-dev"
func resolveVersion(dir ...string) string {
	// If ldflags set a real version (not the dev default), use it.
	if Version != "0.0.0-dev" {
		return normalizeVersion(Version)
	}

	// Determine where to look for git tags.
	gitDir := ""
	if len(dir) > 0 && dir[0] != "" {
		gitDir = findAetherModuleRoot(dir[0])
	} else {
		// Walk up from the binary to find the Aether go.mod.
		exe, err := os.Executable()
		if err == nil {
			gitDir = findAetherModuleRoot(filepath.Dir(exe))
		}
		if gitDir == "" {
			if cwd, err := os.Getwd(); err == nil {
				gitDir = findAetherModuleRoot(cwd)
			}
		}
	}

	if gitDir != "" {
		if repoVersion := readRepoVersion(gitDir); repoVersion != "" {
			return repoVersion
		}

		args := []string{"-C", gitDir, "describe", "--tags", "--abbrev=0"}
		out, err := exec.Command("git", args...).Output()
		if err == nil {
			v := strings.TrimSpace(string(out))
			return normalizeVersion(v)
		}
	}

	if hubVersion := readInstalledHubVersion(); hubVersion != "" {
		return hubVersion
	}

	return Version
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func findAetherModuleRoot(start string) string {
	if start == "" {
		return ""
	}
	d, err := filepath.Abs(start)
	if err != nil {
		d = start
	}
	for {
		goMod := filepath.Join(d, "go.mod")
		data, err := os.ReadFile(goMod)
		if err == nil && strings.Contains(string(data), "github.com/calcosmic/Aether") {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

func readInstalledHubVersion() string {
	hubDir := resolveHubPath()
	if hubDir == "" {
		return ""
	}
	return readHubVersionAtPath(hubDir)
}

func readVersionJSONFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return normalizeVersion(v.Version)
}

func readRepoVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".aether", "version.json"))
	if err != nil {
		return ""
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return normalizeVersion(v.Version)
}

func resolveReleaseVersion(explicit string) (string, error) {
	version := normalizeVersion(explicit)
	if version == "" {
		version = normalizeVersion(resolveVersion())
	}
	if version == "" || version == "0.0.0-dev" {
		return "", fmt.Errorf("cannot infer a release version from this dev binary; pass --version/--binary-version explicitly or run from an installed Aether checkout")
	}
	return version, nil
}

func init() {
	// Override Cobra's default version template to print "aether v<version>"
	// instead of "aether version v<version>"
	rootCmd.SetVersionTemplate("aether {{ .Version }}\n")
	rootCmd.Version = "v" + resolveVersion()
	frontDoorDefaultHelpFunc = rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(renderFrontDoorHelp)
}

// store is the shared storage instance initialized by PersistentPreRunE.
// Commands that need data access should check this variable.
var store *storage.Store

// tracer is the shared trace logger initialized alongside store.
var tracer *trace.Tracer

// stdout and stderr are package-level writers that tests can override.
var stdout io.Writer = os.Stdout
var stderr io.Writer = os.Stderr

// renderedCommandExitCode bridges legacy handlers that render an error envelope
// but return nil to Cobra. Commands execute serially in the CLI process, so one
// atomic marker is sufficient and keeps the shell exit status aligned with JSON.
var renderedCommandExitCode atomic.Int64

// rootCmd is the root Cobra command for the aether CLI.
var rootCmd = &cobra.Command{
	Use:   "aether",
	Short: "Aether Colony Utility Layer",
	// SilenceUsage and SilenceErrors prevent Cobra from printing
	// usage/error text automatically -- we control output ourselves.
	SilenceUsage:  true,
	SilenceErrors: true,
	// Custom version printer to match expected format "aether v<version>"
	Version: "v" + Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Record the invoked command so streaming emitters can consult its
		// ceremony class (quiet commands never stream progress).
		currentStreamingCommand = cmd.Name()

		// Skip store initialization for commands that don't need it.
		if skipStoreInit(cmd) {
			return nil
		}

		dataDir := storage.ResolveDataDir(context.Background())
		s, err := storage.NewStore(dataDir)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		store = s
		tracer = trace.NewTracer(s)
		// Commands that promise a causally read-only result must not create the
		// first-run marker merely because they were inspected. The annotation is
		// owned by the command so future read-only expert surfaces can make the
		// same guarantee without growing a name switch here.
		if cmd.Annotations["aether.io/read-only"] != "true" {
			checkAndEmitFirstRun(dataDir)
		}
		return nil
	},
}

// skipStoreInit returns true for commands that don't require a store
// (completion, version, help).
func skipStoreInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations["aether.io/store-free"] == "true" {
			return true
		}
		switch c.Name() {
		case "command-guide", "completion", "version", "help", "init", "audit-catalog", "reconcile", "internal-worker-adapter":
			return true
		}
	}
	return false
}

const (
	frontDoorNormalGroupID  = "normal-journey"
	frontDoorInspectGroupID = "steer-and-inspect"
	frontDoorExpertGroupID  = "expert-maintenance"
)

type frontDoorHelpEntry struct {
	command     string
	description string
}

type frontDoorHelpGroup struct {
	id      string
	title   string
	entries []frontDoorHelpEntry
}

var (
	frontDoorHelpOnce        sync.Once
	frontDoorDefaultHelpFunc func(*cobra.Command, []string)
	frontDoorHelpGroups      = []frontDoorHelpGroup{
		{
			id: frontDoorNormalGroupID, title: "Normal journey",
			entries: []frontDoorHelpEntry{
				{`/ant-init "goal"`, "Start a guided colony for one goal."},
				{"/ant-plan", "Turn the accepted goal and territory evidence into an executable phase plan."},
				{"/ant-build", "Execute one accepted phase with guided checkpoints."},
				{"/ant-run", "Autopilot the remaining accepted phases within the displayed safety contract."},
				{"/ant-status", "Show the complete authoritative colony snapshot."},
				{"/ant-pause", "Stop at a safe boundary and save one resumable handoff."},
				{"/ant-resume", "Validate and restore the safest honest recovery point."},
				{"/ant-seal", "Close a verified colony, or explicitly record an owner-forced incomplete closure."},
				{"/ant-entomb", "Archive and clear the sealed colony."},
			},
		},
		{
			id: frontDoorInspectGroupID, title: "Steer and inspect",
			entries: []frontDoorHelpEntry{
				{"/ant-focus", "Guide colony attention toward one area."},
				{"/ant-feedback", "Add a gentle correction for future work."},
				{"/ant-redirect", "Record a hard constraint the colony must avoid."},
				{"/ant-watch", "Show live worker activity."},
				{"/ant-phase", "Inspect the current phase and its accepted work."},
				{"/ant-history", "Review recorded colony events."},
				{"/ant-swarm", "Route a problem or inspect the live swarm."},
				{"/ant-research", "Browse saved research and Dreams."},
				{"/ant-memory-details", "Inspect retained learning and memory."},
				{"/ant-review-ledger-summary", "Summarize retained findings."},
			},
		},
		{
			id: frontDoorExpertGroupID, title: "Expert maintenance",
			entries: []frontDoorHelpEntry{
				{"/ant-maintenance", "Inspect or repair Aether internals with preview and rollback."},
			},
		},
	}
)

// configureFrontDoorHelp runs lazily from Cobra's help hook. By then every
// file-level init function has registered its command, so assigning groups
// cannot depend on Go's cross-file initialization order.
func configureFrontDoorHelp() {
	frontDoorHelpOnce.Do(func() {
		for _, group := range frontDoorHelpGroups {
			rootCmd.AddGroup(&cobra.Group{ID: group.id, Title: group.title})
		}
		membership := map[string]string{
			"init": frontDoorNormalGroupID, "plan": frontDoorNormalGroupID, "build": frontDoorNormalGroupID,
			"run": frontDoorNormalGroupID, "status": frontDoorNormalGroupID, "pause": frontDoorNormalGroupID,
			"resume-colony": frontDoorNormalGroupID, "seal": frontDoorNormalGroupID, "entomb": frontDoorNormalGroupID,
			"focus": frontDoorInspectGroupID, "feedback": frontDoorInspectGroupID, "redirect": frontDoorInspectGroupID,
			"watch": frontDoorInspectGroupID, "phase": frontDoorInspectGroupID, "history": frontDoorInspectGroupID,
			"swarm": frontDoorInspectGroupID, "research": frontDoorInspectGroupID, "memory-details": frontDoorInspectGroupID,
			"review-ledger-summary": frontDoorInspectGroupID, "maintenance": frontDoorExpertGroupID,
		}
		for _, command := range rootCmd.Commands() {
			if groupID, ok := membership[command.Name()]; ok {
				command.GroupID = groupID
			}
		}
	})
}

func renderFrontDoorHelp(cmd *cobra.Command, args []string) {
	if cmd != rootCmd {
		frontDoorDefaultHelpFunc(cmd, args)
		return
	}
	configureFrontDoorHelp()
	width := lifecycleStatusOutputWidth()
	projection := frontDoorLifecycleProjection(resolveAetherRootPath())
	var lines []string
	if projection.Identity.Source.Provenance == LifecycleFactMissing || strings.TrimSpace(projection.Goal.Value) == "" {
		lines = append(lines,
			"No colony is active",
			`Start a guided colony for one goal with /ant-init "goal".`,
		)
	} else {
		lines = append(lines, renderFrontDoorStanding(projection))
	}
	lines = append(lines, "", "Usage: /ant-help [command]")
	for _, group := range frontDoorHelpGroups {
		lines = append(lines, "", group.title)
		for _, entry := range group.entries {
			if width < 64 {
				lines = append(lines, "  "+entry.command)
				for _, wrapped := range lifecycleStatusWrapLine(entry.description, width-4) {
					lines = append(lines, "    "+strings.TrimSpace(wrapped))
				}
				continue
			}
			row := fmt.Sprintf("  %-16s  %s", entry.command, entry.description)
			lines = append(lines, lifecycleStatusWrapLine(row, width)...)
		}
	}
	lines = append(lines, "", "Use /ant-help <command> for expert detail outside this journey map.")
	var rendered []string
	for _, line := range lines {
		rendered = append(rendered, lifecycleStatusWrapLine(line, width)...)
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.Join(rendered, "\n"))
}

// frontDoorLifecycleProjection reads only the facts needed by the compact
// standing line. It intentionally avoids storage.NewStore: merely asking for
// help must not create a data or lock directory.
func frontDoorLifecycleProjection(root string) LifecycleProjection {
	root = filepath.Clean(root)
	dataDir := filepath.Join(root, ".aether", "data")
	state, stateSource := readLifecycleState(filepath.Join(dataDir, "COLONY_STATE.json"))
	facts := lifecycleFactsFromStateSnapshot(state, stateSource.Provenance == LifecycleFactMissing, time.Now().UTC())
	facts.Root = root
	facts.State.Source = stateSource
	facts.Identity.Source = lifecycleDerivedSource("identity", stateSource)
	facts.Progress.Source = lifecycleDerivedSource("progress", stateSource)
	facts.Timing = lifecycleTiming(state, stateSource, facts.CapturedAt)
	facts.Actors.Value, facts.Actors.Source = readLifecycleActors(filepath.Join(dataDir, "spawn-tree.txt"))
	flags, blockerSource := readLifecycleJSON[struct {
		Decisions []colony.FlagEntry `json:"decisions"`
	}]("blockers", filepath.Join(dataDir, "pending-decisions.json"))
	facts.Blockers = LifecycleFact[[]colony.FlagEntry]{Value: flags.Decisions, Source: blockerSource}
	return projectLifecycle(facts, LifecycleViewCompact, "claude")
}

func renderFrontDoorStanding(projection LifecycleProjection) string {
	identity := projection.Identity.Value
	name := emptyFallback(strings.TrimSpace(identity.Name), "Unnamed colony")
	goal := emptyFallback(strings.TrimSpace(projection.Goal.Value), "Not recorded")
	episode := emptyFallback(strings.TrimSpace(identity.Episode), "Not recorded")
	standing := emptyFallback(strings.TrimSpace(projection.Standing.Value), "UNKNOWN")
	phase := projection.Phase.Value
	active, _ := lifecycleStatusActors(projection.Actors.Value)
	castes := make([]string, 0, len(active))
	seen := map[string]bool{}
	for _, actorFact := range active {
		caste := strings.TrimSpace(actorFact.Caste)
		if caste == "" || seen[strings.ToLower(caste)] {
			continue
		}
		seen[strings.ToLower(caste)] = true
		castes = append(castes, strings.ToUpper(caste[:1])+caste[1:])
	}
	sort.Strings(castes)
	actors := "No ants are active"
	if len(castes) > 0 {
		actors = strings.Join(castes, ", ")
	}
	next := lifecycleStatusActionCommand(projection.NextAction)
	if len(projection.NextAction.Choices) > 0 {
		choices := make([]string, 0, len(projection.NextAction.Choices))
		for _, choice := range projection.NextAction.Choices {
			command := choice.DisplayCommand
			if command == "" {
				command = choice.RuntimeCommand
			}
			choices = append(choices, command)
		}
		next = strings.Join(choices, " or ")
	}
	return fmt.Sprintf("Colony: %s | Goal: %s | Episode: %s | Phase: %d/%d | Standing: %s | Ants: %s | Blockers: %d | Next Up: %s",
		name, goal, episode, phase.CurrentNumber, phase.TotalPhases, standing, actors, len(projection.Blockers), emptyFallback(next, "Not available"))
}

// Execute runs the root command and returns any error.
func Execute() error {
	renderedCommandExitCode.Store(0)
	err := rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code > 0 {
		return renderedErrorExit(code)
	}
	return err
}

func markRenderedCommandError(code int) {
	if code <= 0 {
		code = 1
	}
	renderedCommandExitCode.Store(int64(code))
}

type renderedCommandError struct {
	code int
}

func (e renderedCommandError) Error() string {
	return fmt.Sprintf("command failed after rendering error output with code %d", e.code)
}

func renderedErrorExit(code int) error {
	if code <= 0 {
		code = 1
	}
	return renderedCommandError{code: code}
}

// ExitWithError prints the error to stderr and exits with code 1.
func ExitWithError(err error) {
	var renderedErr renderedCommandError
	if errors.As(err, &renderedErr) {
		os.Exit(renderedErr.code)
	}
	if err != nil {
		visualFprintln(stderr, "Error:", err.Error())
	}
	os.Exit(1)
}
