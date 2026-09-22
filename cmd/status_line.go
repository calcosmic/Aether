package cmd

// Phase 206 plan 02 -- the permanent line.
//
// Claude Code (and any platform honouring the same settings shape) can be told
// to redraw one line at the bottom of the chat window on every turn via the
// `statusLine` settings key. This file is that line: where the owner is in
// the project and the one command to run next, redrawn constantly and costing
// nothing to read.
//
// Two rules bind it, load-bearing and asserted rather than promised:
//
//  1. It DECIDES nothing. Every command it can print comes from
//     resolveNextAction (cmd/next_action.go), the one shared what-next
//     decision every other lifecycle surface already renders from. A command
//     spelled here would be a rival decider, exactly the drift Phase 197
//     exists to end. TestStatusLineComesFromTheSharedDecision's AST guard
//     locks this structurally, not by convention.
//
//  2. It WRITES nothing. It is redrawn on every turn, so a write here would
//     run constantly; TestStatusLineChangesNothingAndRepeatsItself asserts
//     the project's saved data is byte-identical before and after a hundred
//     runs over one unchanged project.
//
// Silence is the correct output where there is no project -- the same rule
// the session-start greeting already follows -- and this command never calls
// loadNextActionInputForGreeting: that reader also reaches the shared
// cross-project hive and the owner's saved preferences, cost this line (read
// every second) has no reason to pay.

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/calcosmic/Aether/pkg/colony"
)

// statusLineMaxTaskRunes is where a long open-task goal is shortened. Chosen
// to keep the whole line readable on an ordinary terminal width alongside the
// phase segment and the command segment.
const statusLineMaxTaskRunes = 40

// statusLineTaskLabel renders the open task in ordinary words: the first
// task in the current phase that is not yet completed, or "all tasks done"
// once every task in the phase is, or the empty string when the phase has no
// tasks at all. It never prints a task's raw stored status value -- printing
// "in_progress" is exactly the bookkeeping-token leak the plain-English rule
// forbids.
func statusLineTaskLabel(tasks []colony.Task) string {
	if len(tasks) == 0 {
		return ""
	}
	for _, task := range tasks {
		if task.Status == colony.TaskCompleted {
			continue
		}
		return shortenStatusLineGoal(task.Goal)
	}
	return "all tasks done"
}

// shortenStatusLineGoal collapses whitespace and, when the result is longer
// than statusLineMaxTaskRunes, cuts back to the last whole word that fits and
// appends a single-character ellipsis. Only when the very first word is itself
// longer than the limit (one long identifier or path as a goal) is that word
// cut, so the line can never grow past the limit.
func shortenStatusLineGoal(goal string) string {
	collapsed := strings.Join(strings.Fields(goal), " ")
	runes := []rune(collapsed)
	if len(runes) <= statusLineMaxTaskRunes {
		return collapsed
	}
	window := string(runes[:statusLineMaxTaskRunes])
	if idx := strings.LastIndex(window, " "); idx > 0 {
		window = window[:idx]
	}
	return strings.TrimRight(window, " ") + "…"
}

// statusLineText assembles the whole line, or the empty string when there is
// no lifecycle projection to render (loadNextActionInput reports NoColony,
// so resolveNextAction was never even asked). Segments are joined with a
// middle dot and omitted individually when they carry nothing to say.
//
// The command segment always resolves through translateHintCommandsForPlatform
// -- the one existing command translator every other rendered surface in this
// package already uses (cardText, renderNextUp) -- so the line always names
// the form the owner can actually type on this platform, never the bare
// runtime spelling. Do not write a second translator here.
func statusLineText(answer nextAction, platform string) string {
	if answer.Projection == nil {
		return ""
	}
	proj := answer.Projection

	segments := []string{"Aether"}

	phase := proj.Phase.Value
	if phase.CurrentNumber != 0 {
		seg := fmt.Sprintf("phase %d of %d", phase.CurrentNumber, phase.TotalPhases)
		if phase.Current != nil {
			if name := strings.TrimSpace(phase.Current.Name); name != "" {
				seg += ": " + name
			}
		}
		segments = append(segments, seg)
	}

	if label := statusLineTaskLabel(proj.Tasks.Value); label != "" {
		segments = append(segments, label)
	}

	command := strings.TrimSpace(proj.NextAction.DisplayCommand)
	if command == "" {
		command = strings.TrimSpace(proj.NextAction.RuntimeCommand)
	}
	if command != "" {
		segments = append(segments, "next: "+translateHintCommandsForPlatform(command, platform))
	}

	return strings.Join(segments, " · ")
}

// statusLineCmd is the hidden command Claude Code's `statusLine` settings key
// names. It is redrawn on every turn, so it must stay cheap and read-only:
// loadNextActionInput (not the greeting variant) is the same read-only loader
// every other ordinary closing card already uses.
var statusLineCmd = &cobra.Command{
	Use:    "status-line",
	Short:  "Claude status line: where things stand and the one command to run next",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Drain stdin the way the sibling hooks do -- nothing the platform
		// says about the session is needed; where things stand is read from
		// the project, not from the payload.
		_, _ = readClaudeHookInput()

		in := loadNextActionInput()
		if in.NoColony {
			return nil
		}

		text := statusLineText(resolveNextAction(in), detectPlatform())
		if text != "" {
			fmt.Fprintln(stdout, text)
		}
		return nil
	},
}
