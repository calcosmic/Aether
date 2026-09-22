package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// reportBundleNow is a clock seam so a test can pin "now" and prove two
// bundles written in the same instant still get distinct filenames --
// follows the same pattern planCandidateNow (cmd/plan_candidate.go) uses.
var reportBundleNow = func() time.Time { return time.Now().UTC() }

// reportBundleRecentLimit bounds how many refusals and failures the bundle
// lists -- enough to see a pattern, small enough to stay paste-ready.
const reportBundleRecentLimit = 20

func init() {
	reportCmd.Flags().String("output", "", "directory to write the report bundle into (default: .aether/reports/)")
	rootCmd.AddCommand(reportCmd)
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Write a bundle describing what Aether refused, what went wrong, and where this project is -- send it to whoever maintains Aether",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := skillWorkspaceRoot()
		path, err := runReportBundle(root)
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		if outputDir, _ := cmd.Flags().GetString("output"); strings.TrimSpace(outputDir) != "" {
			target := outputDir
			if !filepath.IsAbs(target) {
				target = filepath.Join(root, target)
			}
			if mkErr := os.MkdirAll(target, 0755); mkErr == nil {
				dest := filepath.Join(target, filepath.Base(path))
				if renameErr := os.Rename(path, dest); renameErr == nil {
					path = dest
				}
			}
		}
		visual := renderBanner("\U0001F4C4", "Report") + visualDividerStr() +
			"Wrote a bundle you can send to whoever maintains Aether:\n" +
			"  " + path + "\n"
		outputWorkflow(map[string]interface{}{"path": path}, visual)
		return nil
	},
}

// runReportBundle assembles a paste-ready markdown bundle describing this
// run of Aether -- version, where the project is, what it refused, what
// went wrong, and what is still open -- and writes it under
// <store>/../reports/ (or <root>/.aether/reports/ when there is no
// project), returning the path it wrote. It always succeeds when there is
// no project in this folder -- the bundle itself says so -- rather than
// refusing.
func runReportBundle(root string) (string, error) {
	now := reportBundleNow()
	body := buildReportBundleBody()

	dir := reportBundleOutputDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create report directory %q: %w", dir, err)
	}
	path, err := reportBundleUniquePath(dir, now)
	if err != nil {
		return "", fmt.Errorf("choose report filename: %w", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return "", fmt.Errorf("write report bundle: %w", err)
	}
	return path, nil
}

func reportBundleOutputDir(root string) string {
	if store != nil {
		return filepath.Join(filepath.Dir(store.BasePath()), "reports")
	}
	return filepath.Join(root, ".aether", "reports")
}

// reportBundleUniquePath names the bundle report-<UTC timestamp>.md, with
// colons replaced by hyphens (a colon is not a safe filename character on
// every filesystem), and appends -2, -3, ... when that name is already
// taken -- so two bundles written in the same second never collide or
// overwrite each other.
func reportBundleUniquePath(dir string, now time.Time) (string, error) {
	stamp := strings.ReplaceAll(now.Format(time.RFC3339), ":", "-")
	candidate := filepath.Join(dir, fmt.Sprintf("report-%s.md", stamp))
	for n := 2; ; n++ {
		_, err := os.Stat(candidate)
		if os.IsNotExist(err) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		candidate = filepath.Join(dir, fmt.Sprintf("report-%s-%d.md", stamp, n))
	}
}

// buildReportBundleBody assembles the six sections, in order, then repairs
// the whole result to valid UTF-8 -- the same discipline
// TestDirectRouteMessageIsValidJSONForEveryScreen already requires of the
// hook payload -- so a recorded refusal, failure, or screen carrying an
// invalid byte sequence still produces a bundle a reader can open.
func buildReportBundleBody() string {
	var b strings.Builder
	b.WriteString("# Aether report bundle\n\n")

	b.WriteString("## What this is\n\n")
	b.WriteString("This file is for sending to whoever maintains Aether. Nothing in it changes this project.\n\n")

	b.WriteString("## Version\n\n")
	b.WriteString(resolveVersion())
	b.WriteString("\n\n")

	b.WriteString("## Where the project is\n\n")
	b.WriteString(reportBundleProjectSection())
	b.WriteString("\n")

	b.WriteString("## What Aether refused\n\n")
	b.WriteString(reportBundleRefusalsSection())
	b.WriteString("\n")

	b.WriteString("## What went wrong\n\n")
	b.WriteString(reportBundleFailuresSection())
	b.WriteString("\n")

	b.WriteString("## What is still open\n\n")
	b.WriteString(reportBundleOpenItemsSection())

	return strings.ToValidUTF8(b.String(), string(utf8.RuneError))
}

// reportBundleProjectSection reports the project's own goal and current
// phase and the one next command -- read from the same shared
// resolveNextAction (cmd/next_action.go) every other closing screen uses,
// never a second what-next decision -- or, with no project set up here,
// says so plainly and names `aether init`.
func reportBundleProjectSection() string {
	answer := resolveNextAction(loadNextActionInputForCommand("report"))
	if answer.Standing.State == "none" || answer.Standing.State == "" {
		return "There is no project set up in this folder. Run `aether init \"your goal\"` to start one.\n"
	}
	goal := strings.TrimSpace(answer.Standing.Goal)
	if goal == "" {
		goal = "(no goal recorded)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "- Goal: %s\n", goal)
	fmt.Fprintf(&b, "- Phase: %d of %d\n", answer.Standing.CurrentPhase, answer.Standing.TotalPhases)
	command := strings.TrimSpace(answer.Command)
	if command == "" {
		command = "aether status"
	}
	fmt.Fprintf(&b, "- Next command: `%s`\n", command)
	return b.String()
}

// reportBundleRefusalsSection lists the last reportBundleRecentLimit
// recorded refusals, newest first, each naming when, what stopped, and the
// one command that got past it.
func reportBundleRefusalsSection() string {
	entries := refusalLogEntries(reportBundleRecentLimit)
	if len(entries) == 0 {
		return "No refusals recorded yet.\n"
	}
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "- [%s] %s — next: `%s`\n", e.RecordedAt, e.What, e.NextCommand)
	}
	return b.String()
}

// reportBundleFailuresSection lists the last reportBundleRecentLimit
// recorded failures, newest first, read through the same failure-log store
// (cmd/midden_shared.go) `aether midden-recent-failures` reads.
func reportBundleFailuresSection() string {
	if store == nil {
		return "No project set up, so nothing has been recorded.\n"
	}
	mf, err := loadMiddenFile(store)
	if err != nil || len(mf.Entries) == 0 {
		return "No failures recorded yet.\n"
	}
	entries := append([]colony.MiddenEntry(nil), mf.Entries...)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})
	if len(entries) > reportBundleRecentLimit {
		entries = entries[:reportBundleRecentLimit]
	}
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "- [%s] %s: %s\n", e.Timestamp, e.Category, e.Message)
	}
	return b.String()
}

// reportBundleOpenItemsSection reports the blocker/issue/note counts from
// the one shared counting rule, classifyOpenFlags (cmd/open_flags.go),
// never a second re-derived count.
func reportBundleOpenItemsSection() string {
	if store == nil {
		return "No project set up, so nothing is open.\n"
	}
	flagsFile, ok := loadFlagsFile(store)
	if !ok {
		return "Nothing open.\n"
	}
	c := classifyOpenFlags(flagsFile.Decisions)
	return fmt.Sprintf("- Blocking work: %d\n- Issues: %d\n- Notes for later: %d\n", len(c.Blockers), len(c.Issues), len(c.Notes))
}
