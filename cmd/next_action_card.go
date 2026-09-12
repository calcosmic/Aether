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
	if list := renderIndentedList(voicedLines("history", answer.Changed)); list != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("What changed"))
		b.WriteString(cardText(list, platform))
	}

	// 3. Anything open that is waiting on the owner.
	if list := renderIndentedList(voicedLines("flag", nextActionFlagLines(answer.Open.Flags))); list != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Waiting on you"))
		b.WriteString(cardText(list, platform))
	}
	// The signal-type glyph would require knowing each signal's FOCUS/
	// REDIRECT/FEEDBACK type, but answer.Open.Signals is already flattened
	// to plain text by the resolver (nextActionInputForState -> Signals
	// []string) -- there is no type left to key a glyph by without a second
	// read, which CEC-05/SYN-VOICE-05 forbids at this pure-rendering layer.
	// `feedback` is the closest single category every standing instruction
	// shares: it is steering guidance, the same concept FEEDBACK signals name.
	if list := renderIndentedList(voicedLines("feedback", answer.Open.Signals)); list != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Your standing instructions"))
		b.WriteString(cardText(list, platform))
	}

	// 3.5. What the colony remembers about the owner -- preferences, the
	// strongest learned habits, the last helper's note (198.2 plan 03, D-12).
	// This is populated ONLY by loadNextActionInputForGreeting, so it renders
	// only on the session-start greeting; every other closing card leaves
	// answer.Memory zero and this block disappears entirely -- the same
	// if-non-empty pattern every other part on this card already uses, so an
	// empty part is never a heading with nothing under it (D-14).
	if section := renderNextActionMemory(answer.Memory); section != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("What I remember about you"))
		b.WriteString(cardText(section, platform))
	}

	// 4-6. Render the projected action, its coequal choice set, evidence, and
	// secondary safe choices without re-deciding any of them here. Older
	// hand-constructed callers retain the compatibility rendering until their
	// own migration plan supplies a projection.
	if answer.Projection != nil {
		b.WriteString(renderLifecycleProjectionNextUp(*answer.Projection, platform))
	} else {
		b.WriteString(renderNextUp(
			nextActionPrimarySuggestion(answer),
			nextActionAlternativeSuggestions(answer)...,
		))
	}

	// Any place the availability check substituted a command. The owner is
	// never silently redirected to something other than what was decided.
	if list := renderIndentedList(voicedLines("warning", answer.Notes)); list != "" {
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

func renderLifecycleProjectionNextUp(projection LifecycleProjection, platform string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderBanner(commandEmoji("next-up"), "Next Up"))
	action := projection.NextAction

	if len(action.Choices) > 0 {
		if reason := strings.TrimSpace(action.Reason); reason != "" {
			b.WriteString(reason)
			b.WriteString("\n")
		}
		for _, choice := range action.Choices {
			command := lifecycleProjectionCommand(choice.RuntimeCommand, platform)
			if command == "" {
				continue
			}
			b.WriteString(voiceLine("next", "Choice: "+nextActionSuggestionBody(command, choice.Reason)))
			b.WriteString("\n")
		}
	} else if line := nextActionSuggestionLine(
		lifecycleProjectionCommand(action.RuntimeCommand, platform),
		action.Reason,
	); line != "" {
		b.WriteString(line)
		b.WriteString("\n")
	}

	for _, evidence := range action.Evidence {
		label := strings.TrimSpace(evidence.Summary)
		if label == "" {
			label = strings.TrimSpace(evidence.Source)
		}
		if label == "" {
			label = strings.TrimSpace(evidence.ID)
		}
		if label == "" {
			continue
		}
		b.WriteString("Evidence: ")
		b.WriteString(label)
		b.WriteString("\n")
	}

	for _, alternative := range projection.Alternatives {
		body := nextActionSuggestionBody(
			lifecycleProjectionCommand(alternative.RuntimeCommand, platform),
			alternative.Reason,
		)
		if body == "" {
			continue
		}
		b.WriteString(voiceLine("alternative", "Alternative: "+body))
		b.WriteString("\n")
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

// voicedLines applies voiceLine(kind, ...) to every non-blank entry of lines,
// dropping blanks, so the result can be handed straight to renderIndentedList
// -- which stays untouched, since many other screens share it (RESEARCH.md's
// "Pitfall 1": call the existing funnel, never hand-roll a second one).
func voicedLines(kind string, lines []string) []string {
	voiced := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		voiced = append(voiced, voiceLine(kind, line))
	}
	return voiced
}

// renderNextActionStanding is the "where things stand" block: the goal in the
// owner's own words, how far along the work is, and how far through the plan.
// Every line is glyph-led with `phase` -- this whole section is "where the
// project stands right now", the phase-shaped fact.
func renderNextActionStanding(standing nextActionStanding) string {
	var b strings.Builder
	if goal := strings.TrimSpace(standing.Goal); goal != "" {
		b.WriteString(voiceLine("goal", "Goal: "+goal))
		b.WriteString("\n")
	}
	if explanation := strings.TrimSpace(standing.Explanation); explanation != "" {
		b.WriteString(voiceLine("phase", explanation))
		b.WriteString("\n")
	}
	if milestone := strings.TrimSpace(standing.Milestone); milestone != "" {
		// The milestone is this project's own name for how far along it is, so
		// it is labelled rather than dropped in bare.
		b.WriteString(voiceLine("phase", fmt.Sprintf("Stage reached: %s.", milestone)))
		b.WriteString("\n")
	}
	return b.String()
}

// renderNextActionMemory renders the memory block: what the colony remembers
// about the owner. Each of the three parts is content, not a heading -- a
// part with nothing to say (mem.Preferences == "", no habits, no relay note)
// is simply absent, never a "nothing learned yet" filler line (D-14). Every
// sentence is written for someone who has never opened a file here: "learned
// habit" and "the last helper left a note" stand in for this repo's own
// words for those things. Every line is glyph-led with `memory`.
func renderNextActionMemory(mem nextActionMemory) string {
	var b strings.Builder
	if pref := strings.TrimSpace(mem.Preferences); pref != "" {
		b.WriteString(voiceLine("memory", pref))
		b.WriteString("\n")
	}
	for _, habit := range mem.Habits {
		habit = strings.TrimSpace(habit)
		if habit == "" {
			continue
		}
		b.WriteString(voiceLine("memory", "Learned habit: "+habit))
		b.WriteString("\n")
	}
	if note := strings.TrimSpace(mem.RelayNote); note != "" {
		b.WriteString(voiceLine("memory", note))
		b.WriteString("\n")
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
		return voiceLine("checkpoint", "Everything needed to pick this back up is written down, so it is safe to close this chat.")
	case contextHealthClearRecommended:
		return voiceLine("checkpoint", "This is a natural break, and everything is written down -- it is safe to close this chat, "+
			"and starting a fresh one from here will work better than carrying this one on.")
	case contextHealthKeep:
		switch verdict.Reason {
		case contextReasonBuildInProgress:
			return voiceLine("checkpoint", "Don't close this chat yet -- work is still running, and closing now would lose what is in flight.")
		default:
			return voiceLine("checkpoint", "Don't close this chat yet -- the handover note that lets you pick up where you left off "+
				"has not been written to disk.")
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// The machine-readable answer beside the card
// ---------------------------------------------------------------------------

// The stable keys every command's result map carries. Naming them once is what
// lets a wrapper read the next step out of ANY command's envelope without
// knowing which command produced it -- today each one invents its own shape.
const (
	// nextActionResultKey holds the whole answer, so a wrapper that wants a
	// field this list does not flatten can still reach it.
	nextActionResultKey = "next_action"
	// nextActionCommandKey is the exact command to run, in the platform-neutral
	// runtime form. This value is EXECUTED; it is never a slash spelling (S-01).
	nextActionCommandKey = "next_command"
	// nextActionRecommendationKey is the plain-English reason for it.
	nextActionRecommendationKey = "next_recommendation"
	// nextActionAlternativesKey is the two-to-four other ways forward.
	nextActionAlternativesKey = "next_alternatives"
	// nextActionContextHealthKey is the verdict on closing the chat, as an
	// enumeration plus a reason code -- the sentence belongs to the card.
	nextActionContextHealthKey = "next_context_health"
	// nextActionChoicesKey carries a coequal set when there is deliberately no
	// single next command (the accepted-plan build/run decision).
	nextActionChoicesKey = "next_choices"
	// lifecycleProjectionKey exposes the same semantic result the card renders.
	lifecycleProjectionKey = "lifecycle_projection"
)

// applyNextActionToResult folds the resolved answer into a command's result map
// under the stable keys above and returns the same map.
//
// It ADDS; it never removes. A command's existing keys -- including a `next`
// string something downstream is already reading -- survive untouched, because
// migrating readers is not the same job as breaking them.
func applyNextActionToResult(result map[string]interface{}, answer nextAction) map[string]interface{} {
	if result == nil {
		result = map[string]interface{}{}
	}
	result[nextActionResultKey] = answer
	result[nextActionCommandKey] = answer.Command
	result[nextActionRecommendationKey] = answer.Recommendation
	result[nextActionAlternativesKey] = answer.Alternatives
	result[nextActionContextHealthKey] = answer.ContextHealth
	if answer.Projection != nil {
		result[nextActionChoicesKey] = answer.Projection.NextAction.Choices
		result[lifecycleProjectionKey] = answer.Projection
	}
	return result
}

// nextActionFromResult recovers the answer a command folded into its result map
// with applyNextActionToResult. It is how a renderer that is handed only the
// result map still renders the SAME answer the command put in the envelope,
// rather than resolving a second one that could differ.
func nextActionFromResult(result map[string]interface{}) (nextAction, bool) {
	if result == nil {
		return nextAction{}, false
	}
	answer, ok := result[nextActionResultKey].(nextAction)
	return answer, ok
}

// renderNextActionRecovery is the paused-or-blocked block. Every line is
// glyph-led with `blocked`.
func renderNextActionRecovery(recovery nextActionRecovery) string {
	if !recovery.Paused && !recovery.Blocked {
		return ""
	}
	var b strings.Builder
	if explanation := strings.TrimSpace(recovery.Explanation); explanation != "" {
		b.WriteString(voiceLine("blocked", explanation))
		b.WriteString("\n")
	}
	if pausedAt := strings.TrimSpace(recovery.PausedAt); pausedAt != "" {
		b.WriteString(voiceLine("blocked", "Paused at: "+pausedAt))
		b.WriteString("\n")
	}
	if summary := strings.TrimSpace(recovery.Summary); summary != "" {
		b.WriteString(voiceLine("blocked", summary))
		b.WriteString("\n")
	}
	if path := strings.TrimSpace(recovery.ReportPath); path != "" {
		b.WriteString(voiceLine("blocked", "The report is saved at: "+path))
		b.WriteString("\n")
	}
	return b.String()
}
