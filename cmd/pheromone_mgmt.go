package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var pheromonePrimeCmd = &cobra.Command{
	Use:   "pheromone-prime",
	Short: "Format active pheromone signals for prompt injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		pf := loadPheromones()
		if pf == nil {
			outputOK(map[string]interface{}{
				"section":      "",
				"signal_count": 0,
			})
			return nil
		}

		now := time.Now().UTC()
		var focus, redirect, feedback []colony.PheromoneSignal
		for _, sig := range filterSignalsForPrompt(pf.Signals, now) {
			switch sig.Type {
			case "FOCUS":
				focus = append(focus, sig)
			case "REDIRECT":
				redirect = append(redirect, sig)
			case "FEEDBACK":
				feedback = append(feedback, sig)
			}
		}

		var sb strings.Builder
		total := 0

		if len(redirect) > 0 {
			sb.WriteString("## ACTIVE REDIRECT SIGNALS (Hard Constraints)\n\n")
			for _, sig := range redirect {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [REDIRECT] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(redirect)
		}

		if len(focus) > 0 {
			sb.WriteString("## ACTIVE FOCUS SIGNALS\n\n")
			for _, sig := range focus {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [FOCUS] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(focus)
		}

		if len(feedback) > 0 {
			sb.WriteString("## ACTIVE FEEDBACK SIGNALS\n\n")
			for _, sig := range feedback {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [FEEDBACK] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(feedback)
		}

		section := strings.TrimSpace(sb.String())
		if section == "" {
			section = "No active pheromone signals."
		}

		outputOK(map[string]interface{}{
			"section":        section,
			"signal_count":   total,
			"focus_count":    len(focus),
			"redirect_count": len(redirect),
			"feedback_count": len(feedback),
		})
		return nil
	},
}

var colonyPrimeCmd = &cobra.Command{
	Use:   "colony-prime",
	Short: "Assemble full colony context for worker prompt injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		compact, _ := cmd.Flags().GetBool("compact")
		question, _ := cmd.Flags().GetString("question")
		result := buildColonyPrimeOutputOpts(colonyPrimeOptions{Compact: compact, Question: question})
		if strings.TrimSpace(question) == "" {
			emitPromptIntegrityEvents("colony-prime", colonyPrimeIntegrityRecords(result))
		}
		// Ask mode (--question) is a pure inspection: no event emission, no
		// writes — the briefing is read, ranked, and returned. Locked by
		// TestColonyPrimeQuestionIsReadOnly.
		outputOK(result)
		return nil
	},
}

// pheromoneDisplayCmd renders active pheromone signals in a formatted table.
var pheromoneDisplayCmd = &cobra.Command{
	Use:          "pheromone-display",
	Short:        "Display active pheromone signals in formatted table",
	Aliases:      []string{"pheromones"},
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// BIO-08 (plan 203-11): the five influence actions that are not
		// accept/edit/reject are exposed as flags on this existing
		// pheromone management command rather than five new commands.
		// Checked before the read-only display path below so a mutating
		// invocation never also renders the listing.
		if result, handled := runPheromoneInfluenceFlags(cmd); handled {
			return result
		}

		pf := loadPheromones()
		if pf == nil {
			outputWorkflow(map[string]interface{}{
				"signals": []interface{}{},
				"count":   0,
				"display": "No pheromone signals found.",
			}, renderPheromoneDisplayVisual("", 0, pendingSteeringSection()))
			return nil
		}

		filterType, _ := cmd.Flags().GetString("type")
		activeOnly, _ := cmd.Flags().GetBool("active-only")

		var filtered []colony.PheromoneSignal
		for _, sig := range pf.Signals {
			if activeOnly && !sig.Active {
				continue
			}
			if filterType != "" && sig.Type != filterType {
				continue
			}
			filtered = append(filtered, sig)
		}

		if len(filtered) == 0 {
			outputWorkflow(map[string]interface{}{
				"signals": []interface{}{},
				"count":   0,
				"display": "No pheromone signals found.",
			}, renderPheromoneDisplayVisual("", 0, pendingSteeringSection()))
			return nil
		}

		now := time.Now().UTC()
		sort.Slice(filtered, func(i, j int) bool {
			if signalPriority(filtered[i].Type) != signalPriority(filtered[j].Type) {
				return signalPriority(filtered[i].Type) < signalPriority(filtered[j].Type)
			}
			iStrength := computeEffectiveStrength(filtered[i], now)
			jStrength := computeEffectiveStrength(filtered[j], now)
			if iStrength != jStrength {
				return iStrength > jStrength
			}
			return extractText(filtered[i].Content) < extractText(filtered[j].Content)
		})

		// Classic sectioned view (v5.4.0 house style): emoji-headed groups
		// with a plain-English explanation, one signal per line with its
		// nested lifetime detail — never a fixed-width machine table.
		display := renderClassicPheromoneSections(filtered, now)

		// Build serializable signals list
		signals := make([]map[string]interface{}, len(filtered))
		for i, sig := range filtered {
			entry := map[string]interface{}{
				"id":                 sig.ID,
				"type":               sig.Type,
				"priority":           sig.Priority,
				"active":             sig.Active,
				"effective_strength": computeEffectiveStrength(sig, now),
				"life":               signalLifetimeSummary(sig, now),
			}
			if sig.Strength != nil {
				entry["strength"] = *sig.Strength
			}
			signals[i] = entry
		}

		outputWorkflow(map[string]interface{}{
			"signals": signals,
			"count":   len(filtered),
			"display": display,
		}, renderPheromoneDisplayVisual(display, len(filtered), pendingSteeringSection()))
		return nil
	},
}

// renderPheromoneDisplayVisual renders the signal table for a human reader.
//
// This command used to print the bare table to stdout and then a JSON envelope
// on top of it, in every output mode — so `/ant-pheromones` showed a user a
// table followed by a wall of JSON, with no banner and no next step. It was the
// only lifecycle command with no Next Up block.
func renderPheromoneDisplayVisual(display string, count int, suggestions string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("pheromones"), "Pheromone Signals"))
	b.WriteString(visualDividerStr())

	if count == 0 {
		b.WriteString("No active signals. Workers are running on colony context alone.\n")
		if suggestions != "" {
			b.WriteString(suggestions)
		}
		b.WriteString(renderNextUp(
			"Run `aether focus \"<area>\"` to point the colony at something specific.",
			"Run `aether redirect \"<pattern>\"` to set a hard constraint workers must not break.",
			"Run `aether status` to see where the colony is before steering it.",
		))
		return b.String()
	}

	b.WriteString(strings.TrimRight(display, "\n"))
	b.WriteString("\n")
	if suggestions != "" {
		b.WriteString(suggestions)
	}
	b.WriteString(renderNextUp(
		"Run `aether build <phase>` — these signals are injected into every worker prompt.",
		"Run `aether feedback \"<note>\"` to adjust behaviour without adding a hard constraint.",
		"Run `aether redirect \"<pattern>\"` if something here needs to become a hard constraint.",
	))
	return b.String()
}

var pheromoneSnapshotInjectCmd = &cobra.Command{
	Use:   "pheromone-snapshot-inject",
	Short: "Copy active pheromone signals from one repo/worktree root into another",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		sourceRoot, _ := cmd.Flags().GetString("source-root")
		targetRoot, _ := cmd.Flags().GetString("target-root")
		if strings.TrimSpace(targetRoot) == "" {
			outputError(1, "--target-root is required", nil)
			return nil
		}

		result, err := syncPheromoneStores(sourceRoot, targetRoot, pheromoneSyncOptions{ActiveOnly: true})
		if err != nil {
			outputError(2, fmt.Sprintf("failed to inject pheromone snapshot: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"injected": true,
			"result":   result,
		})
		return nil
	},
}

var pheromoneMergeBackCmd = &cobra.Command{
	Use:   "pheromone-merge-back",
	Short: "Merge pheromone changes from one repo/worktree root back into another",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceRoot, _ := cmd.Flags().GetString("source-root")
		targetRoot, _ := cmd.Flags().GetString("target-root")
		if strings.TrimSpace(sourceRoot) == "" {
			outputError(1, "--source-root is required", nil)
			return nil
		}

		result, err := syncPheromoneStores(sourceRoot, targetRoot, pheromoneSyncOptions{})
		if err != nil {
			outputError(2, fmt.Sprintf("failed to merge pheromones: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"merged": true,
			"result": result,
		})
		return nil
	},
}

func init() {
	colonyPrimeCmd.Flags().Bool("compact", false, "Use 4000 char budget instead of 8000")
	colonyPrimeCmd.Flags().String("question", "", "Ask mode: boost sections relevant to this question and include recent activity (read-only)")

	pheromoneDisplayCmd.Flags().String("type", "", "Filter by signal type (FOCUS/REDIRECT/FEEDBACK)")
	pheromoneDisplayCmd.Flags().Bool("active-only", true, "Only show active signals")
	pheromoneDisplayCmd.Flags().String("reinforce", "", "Reinforce a note by ID: raise its strength to the ceiling and record the action")
	pheromoneDisplayCmd.Flags().String("defer", "", "Defer a note by ID until --defer-until, recording the action")
	pheromoneDisplayCmd.Flags().String("defer-until", "", "RFC3339 timestamp the note named by --defer returns to effect at (required with --defer)")
	pheromoneDisplayCmd.Flags().String("expire", "", "Expire a note by ID now, recording the previous expiry")
	pheromoneDisplayCmd.Flags().String("revoke", "", "Revoke a note by ID permanently -- owner only")
	pheromoneDisplayCmd.Flags().String("appeal", "", "Appeal a rejected note by ID, surfacing it for reconsideration -- owner only")
	pheromoneDisplayCmd.Flags().String("reason", "", "Reason recorded alongside --reinforce/--defer/--expire/--revoke/--appeal")
	pheromoneDisplayCmd.Flags().String("actor", "", "Actor performing --reinforce/--defer/--expire/--revoke/--appeal: owner (default), runtime, or learning")
	pheromoneDisplayCmd.Flags().String("actor-name", "", "Name of the actor performing the action (optional, recorded in the history)")
	pheromoneDisplayCmd.Flags().Bool("dry-run", false, "Preview --reinforce/--defer/--expire/--revoke/--appeal without persisting")
	pheromoneSnapshotInjectCmd.Flags().String("source-root", "", "Repo or worktree root to copy active pheromones from (default current AETHER_ROOT)")
	pheromoneSnapshotInjectCmd.Flags().String("target-root", "", "Repo or worktree root to inject active pheromones into")
	pheromoneMergeBackCmd.Flags().String("source-root", "", "Repo or worktree root to merge pheromones from")
	pheromoneMergeBackCmd.Flags().String("target-root", "", "Repo or worktree root to merge pheromones into (default current AETHER_ROOT)")
	pheromoneMergeBackCmd.Flags().String("export-file", "", "Export merged pheromones to this file path")

	rootCmd.AddCommand(pheromonePrimeCmd)
	rootCmd.AddCommand(colonyPrimeCmd)
	rootCmd.AddCommand(pheromoneSnapshotInjectCmd)
	rootCmd.AddCommand(pheromoneMergeBackCmd)
	rootCmd.AddCommand(pheromoneDisplayCmd)
}

func colonyLifecycleSignalContext(state colony.ColonyState) string {
	lifecycleLine := fmt.Sprintf("Colony is %s. ", state.State)
	switch state.State {
	case colony.StateREADY:
		if len(state.Plan.Phases) == 0 {
			lifecycleLine += "Signals should guide planning scope and approach."
		} else {
			lifecycleLine += "Signals are pre-build guidance for upcoming execution."
		}
	case colony.StateEXECUTING:
		lifecycleLine += "Signals are active implementation constraints."
	case colony.StateBUILT:
		lifecycleLine += "Signals guide verification and learning extraction."
	default:
		lifecycleLine += "Signals provide ongoing context."
	}
	return lifecycleLine
}

// extractText extracts the text field from JSON content like {"text":"..."}.
func extractText(raw json.RawMessage) string {
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err == nil {
		if text, ok := m["text"]; ok {
			return text
		}
	}
	return strings.TrimSpace(string(raw))
}

// pendingSteeringSection loads colony state quietly and renders its active
// pending suggestions — empty when there is no colony or nothing pending.
func pendingSteeringSection() string {
	state, err := loadActiveColonyState()
	if err != nil {
		return ""
	}
	return renderSuggestedSteering(state)
}

// runPheromoneInfluenceFlags checks pheromoneDisplayCmd's --reinforce,
// --defer, --expire, --revoke and --appeal flags (BIO-08, plan 203-11) and,
// if exactly one names a note, performs that action and returns handled=true
// so the caller returns immediately rather than falling through to the
// read-only listing below. handled=false means none of the five flags were
// set and the caller should proceed with its normal display behaviour.
func runPheromoneInfluenceFlags(cmd *cobra.Command) (error, bool) {
	reinforceID, _ := cmd.Flags().GetString("reinforce")
	deferID, _ := cmd.Flags().GetString("defer")
	deferUntil, _ := cmd.Flags().GetString("defer-until")
	expireID, _ := cmd.Flags().GetString("expire")
	revokeID, _ := cmd.Flags().GetString("revoke")
	appealID, _ := cmd.Flags().GetString("appeal")
	reason, _ := cmd.Flags().GetString("reason")
	actorFlag, _ := cmd.Flags().GetString("actor")
	actorName, _ := cmd.Flags().GetString("actor-name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	actor := pheromoneActorOwner
	if strings.TrimSpace(actorFlag) != "" {
		actor = actorFlag
	}

	switch {
	case reinforceID != "":
		return renderPheromoneInfluenceResult(reinforceNote(reinforceID, actor, actorName, reason, dryRun)), true
	case deferID != "":
		return renderPheromoneInfluenceResult(deferNote(deferID, actor, actorName, deferUntil, reason, dryRun)), true
	case expireID != "":
		return renderPheromoneInfluenceResult(expireNote(expireID, actor, actorName, reason, dryRun)), true
	case revokeID != "":
		return renderPheromoneInfluenceResult(revokeNote(revokeID, actor, actorName, reason, dryRun)), true
	case appealID != "":
		return renderPheromoneInfluenceResult(appealNote(appealID, actor, actorName, reason, dryRun)), true
	}
	return nil, false
}

// renderPheromoneInfluenceResult writes the CLI envelope for one of the five
// influence actions. RunE functions in this package always return nil after
// calling outputError/outputOK -- errors are rendered as a JSON/visual error
// envelope, not returned as a Go error, matching every other command in this
// file.
func renderPheromoneInfluenceResult(outcome pheromoneInfluenceOutcome, err error) error {
	if err != nil {
		outputError(1, err.Error(), nil)
		return nil
	}
	if !outcome.Found {
		outputError(1, "note not found", nil)
		return nil
	}
	result := map[string]interface{}{
		"applied":     !outcome.WouldApply,
		"would_apply": outcome.WouldApply,
	}
	if outcome.Signal != nil {
		result["signal"] = *outcome.Signal
	}
	if outcome.Entry != nil {
		result["entry"] = *outcome.Entry
	}
	outputOK(result)
	return nil
}
