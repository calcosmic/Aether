package cmd

// Phase 197 plan 02 -- the one closing card.
//
// Eleven lifecycle commands each hand-write their own closing message today, so
// a wording fix has to land eleven times and usually lands in one or two. This
// is the single card they will all render from, built from the one answer
// resolveNextAction gives (cmd/next_action.go).
//
// Three rules bind everything below, and each is asserted by a test rather than
// promised by this comment:
//
//  1. It DECIDES nothing. Whether it is safe to close the chat, which command to
//     recommend, which alternatives to offer -- all of that is already resolved.
//     A second implementation of any of those rules is how they drift apart, so
//     TestNextActionCardRefusesUnsafeClear fails if this file ever looks for the
//     handover file itself.
//
//  2. Platform translation happens on the way OUT, never in the answer (S-01).
//     The recommendation and the alternatives go through renderNextUp, which is
//     the single funnel where `aether continue` becomes `/ant-continue` on the
//     wrapper platforms. The value the card was handed is unchanged, because
//     wrappers and the TS host EXECUTE it.
//
//  3. Every sentence is written for someone who has never opened a file in this
//     repository. A word this repo invented -- colony, seal, entomb, pheromone
//     -- is explained in the same sentence it appears in, or it does not appear.
//     TestNextActionCardSpeaksPlainEnglish reads the rendered card and fails on
//     any that is not.

import (
	"fmt"
	"strings"
)

// renderNextActionCard turns the one resolved answer into the closing block a
// command shows the owner. The caller hands the result to the existing output
// path; this function writes nothing itself.
func renderNextActionCard(answer nextAction) string {
	return renderNextActionCardForPlatform(answer, detectPlatform())
}

// renderNextActionCardForPlatform is renderNextActionCard with the platform
// supplied, so both spellings can be asserted without touching the environment.
func renderNextActionCardForPlatform(answer nextAction, platform string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("status"), "What Next"))
	b.WriteString(visualDividerStr())

	// 1. Where things stand.
	if section := renderNextActionStanding(answer.Standing); section != "" {
		b.WriteString(renderStageMarker("Where things stand"))
		b.WriteString(cardText(section, platform))
	}

	// 2. What changed.
	if list := renderIndentedList(answer.Changed); list != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("What changed"))
		b.WriteString(cardText(list, platform))
	}

	// 3. Anything open that is waiting on the owner.
	if list := renderIndentedList(nextActionFlagLines(answer.Open.Flags)); list != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Waiting on you"))
		b.WriteString(cardText(list, platform))
	}
	if list := renderIndentedList(answer.Open.Signals); list != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Your standing instructions"))
		b.WriteString(cardText(list, platform))
	}

	// 4-6. The recommendation, the exact command, and the alternatives. This is
	// the single Next Up funnel, which is where platform translation happens.
	b.WriteString(renderNextUp(
		nextActionPrimarySuggestion(answer),
		nextActionAlternativeSuggestions(answer)...,
	))

	// Any place the availability check substituted a command. The owner is
	// never silently redirected to something other than what was decided.
	if list := renderIndentedList(answer.Notes); list != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Heads up"))
		b.WriteString(cardText(list, platform))
	}

	// 7. Whether it is safe to close the chat. Rendered from the resolved
	// verdict; the rule itself lives in the resolver.
	if sentence := renderNextActionContextHealth(answer.ContextHealth); sentence != "" {
		b.WriteString("\n")
		b.WriteString(cardText(sentence, platform))
	}

	// 8. Anything paused or blocked.
	if section := renderNextActionRecovery(answer.Recovery); section != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Paused or blocked"))
		b.WriteString(cardText(section, platform))
	}

	return b.String()
}

// cardText applies the platform's command spelling to a block of prose. The
// funnel in renderNextUp covers the recommendation and the alternatives; this
// covers the surrounding sentences, which can also quote a command.
func cardText(text, platform string) string {
	if text == "" {
		return ""
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return translateHintCommandsForPlatform(text, platform)
}

// renderNextActionStanding is the "where things stand" block: the goal in the
// owner's own words, how far along the work is, and how far through the plan.
func renderNextActionStanding(standing nextActionStanding) string {
	var b strings.Builder
	if goal := strings.TrimSpace(standing.Goal); goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	if explanation := strings.TrimSpace(standing.Explanation); explanation != "" {
		b.WriteString(explanation)
		b.WriteString("\n")
	}
	if milestone := strings.TrimSpace(standing.Milestone); milestone != "" {
		// The milestone is this project's own name for how far along it is, so
		// it is labelled rather than dropped in bare.
		b.WriteString(fmt.Sprintf("Stage reached: %s.\n", milestone))
	}
	return b.String()
}

// nextActionFlagLines turns open flags into one plain line each.
func nextActionFlagLines(flags []nextActionFlag) []string {
	lines := make([]string, 0, len(flags))
	for _, flag := range flags {
		description := strings.TrimSpace(flag.Description)
		if description == "" {
			continue
		}
		if kind := strings.TrimSpace(flag.Type); kind != "" {
			description = "[" + kind + "] " + description
		}
		lines = append(lines, description)
	}
	return lines
}

// renderNextActionContextHealth turns the resolved verdict into the sentence
// the owner reads. It is a RENDERER: it maps an enumeration and a reason code to
// words, and makes no judgement of its own. Deliberately it names no command,
// so the card offers exactly one set of choices -- the recommendation and its
// alternatives -- rather than quietly repeating one of them here.
func renderNextActionContextHealth(verdict nextActionContextVerdict) string {
	switch verdict.Health {
	case contextHealthSafe:
		return "Everything needed to pick this back up is written down, so it is safe to close this chat."
	case contextHealthClearRecommended:
		return "This is a natural break, and everything is written down -- it is safe to close this chat, " +
			"and starting a fresh one from here will work better than carrying this one on."
	case contextHealthKeep:
		switch verdict.Reason {
		case contextReasonBuildInProgress:
			return "Don't close this chat yet -- work is still running, and closing now would lose what is in flight."
		default:
			return "Don't close this chat yet -- the handover note that lets you pick up where you left off " +
				"has not been written to disk."
		}
	}
	return ""
}

// renderNextActionRecovery is the paused-or-blocked block.
func renderNextActionRecovery(recovery nextActionRecovery) string {
	if !recovery.Paused && !recovery.Blocked {
		return ""
	}
	var b strings.Builder
	if explanation := strings.TrimSpace(recovery.Explanation); explanation != "" {
		b.WriteString(explanation)
		b.WriteString("\n")
	}
	if pausedAt := strings.TrimSpace(recovery.PausedAt); pausedAt != "" {
		b.WriteString("Paused at: ")
		b.WriteString(pausedAt)
		b.WriteString("\n")
	}
	if summary := strings.TrimSpace(recovery.Summary); summary != "" {
		b.WriteString(summary)
		b.WriteString("\n")
	}
	if path := strings.TrimSpace(recovery.ReportPath); path != "" {
		b.WriteString("The report is saved at: ")
		b.WriteString(path)
		b.WriteString("\n")
	}
	return b.String()
}
