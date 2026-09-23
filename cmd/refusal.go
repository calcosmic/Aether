package cmd

import (
	"fmt"
	"strings"
)

// refusal is a typed error that always carries the one way past it. It
// implements the error interface so it flows through every existing `error`
// return unchanged -- a caller that only handles `error` still gets a usable
// message; a caller that checks `errors.As` gets the structured fields a
// screen needs to render the full block.
//
// This is the field this project's own field reports and the Phase 207
// journey both dead-ended on missing: a refusal that names no way forward,
// leaving a chat to narrate a guess instead of typing a real command
// (WINDOWS.md row 53).
type refusal struct {
	ID           string
	What         string
	Why          string
	NextCommand  string
	ProtectsWork bool
	ExtraSteps   []string
}

// Error satisfies the error interface with one line: what stopped, and the
// one command that gets past it. A caller that only has a string -- a log
// line, an error wrapped three layers up -- still carries the way out.
//
// 208-09-PLAN.md (D-01): when sessionHasNoOneToAsk is true and this refusal
// both protects work and names a next command, the shared guidance
// sentence is inserted between "what stopped" and the " — next: " clause,
// so the next command stays the LAST thing on the line with nothing after
// it. This ordering is load-bearing: the TypeScript host relays this exact
// string to the chat unchanged (sanitizeBridgeMessage,
// .aether/ts-host/src/go-bridge.ts, redacts but never truncates), and
// journeyPrintedRefusals' relayed-line extractor (cmd/journey.go) depends
// on the command being everything after the marker.
func (r refusal) Error() string {
	what := strings.TrimSpace(r.What)
	next := strings.TrimSpace(r.NextCommand)
	if r.ProtectsWork && next != "" && sessionHasNoOneToAsk() {
		return fmt.Sprintf("%s %s — next: %s", what, unattendedGuidanceSentence, next)
	}
	return fmt.Sprintf("%s — next: %s", what, next)
}

// refuse looks up a registered refusal row by id and returns a refusal
// carrying its fields. Any detail strings are joined and appended to Why as
// one extra sentence, so a call site can name the concrete path, field, or
// value involved without inventing a new row for every variation.
//
// An id with no matching row still returns a refusal -- one whose What names
// the unregistered id explicitly -- never a silent empty screen. That is a
// bug in the refusal it names, not something the caller did.
func refuse(id string, detail ...string) refusal {
	row, ok := refusalForID(id)
	if !ok {
		return refusal{
			ID:          id,
			What:        fmt.Sprintf("Aether tried to refuse with an unregistered id (%q). This is a mistake in Aether itself.", id),
			Why:         "Every refusal is supposed to come from one checked-in table; this one names an id that table does not have.",
			NextCommand: "aether report",
		}
	}
	r := refusal{
		ID:           row.ID,
		What:         row.What,
		Why:          row.Why,
		NextCommand:  row.NextCommand,
		ProtectsWork: row.ProtectsWork,
		ExtraSteps:   row.ExtraSteps,
	}
	if extra := strings.TrimSpace(strings.Join(detail, " ")); extra != "" {
		if r.Why != "" {
			r.Why = r.Why + " " + extra
		} else {
			r.Why = extra
		}
	}
	return r
}

// warnAndCarryOn is the "warn" half of the refusal contract (208-06-PLAN.md
// Task 2): a refusal whose row classifies ProtectsWork=false and
// Disposition="warn" no longer stops the command -- the call site renders
// this block, headed as something Aether noticed rather than something
// that stopped, and then carries on using the recovery its NextCommand
// describes where Aether can perform it itself (the same pattern jj uses
// for stale snapshot state and git uses for a colliding checkout,
// .planning/research/2026-09-21-reliability-and-delivery.md section 2).
// It renders through renderWarning -- the same block shape renderRefusal
// produces, just headed differently -- writes it through the same
// shouldRenderVisualOutput/writeVisualOutput boundary every other
// human-facing screen uses, records it in the refusal log exactly like a
// stopped refusal, and returns nothing: there is no error to propagate,
// because the work after it is expected to run.
func warnAndCarryOn(r refusal) {
	appendRefusalToLog(r)
	if shouldRenderVisualOutput(stdout) {
		writeVisualOutput(stdout, renderWarning(r))
		return
	}
	visualFprintln(stdout, "Noticed:", strings.TrimSpace(r.What), "— continuing. Next:", strings.TrimSpace(r.NextCommand))
}

// renderWarning renders the same block renderRefusal renders -- what
// Aether noticed, why, and the one command that follows up on it -- headed
// as something noticed rather than something that stopped, and without the
// closing "do not edit Aether's own program files" line (there is nothing
// to work around; the command already carried on).
func renderWarning(r refusal) string {
	var b strings.Builder
	b.WriteString(renderBanner("👀", "Noticed"))
	b.WriteString(visualDividerStr())
	b.WriteString(strings.TrimSpace(r.What))
	b.WriteString("\n")
	if why := strings.TrimSpace(r.Why); why != "" {
		b.WriteString("Why: ")
		b.WriteString(why)
		b.WriteString("\n")
	}
	b.WriteString("Aether is carrying on. Next: `")
	b.WriteString(strings.TrimSpace(r.NextCommand))
	b.WriteString("`\n")
	for _, step := range r.ExtraSteps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		b.WriteString("  - ")
		b.WriteString(step)
		b.WriteString("\n")
	}
	return b.String()
}

// renderRefusal produces the owner-facing refusal block, drawn through the
// same banner/divider helpers renderVisualError already uses so a refusal
// screen looks like every other Aether screen, not a special case. The block
// carries, in order: what stopped, why, the one command that gets past it,
// and a closing line naming this as a limit in Aether itself -- never
// something the reader should try to patch around by hand.
//
// This is the ONE renderer both exit lanes call (ExitWithError's plain-text
// path and outputRefusal's drawn-screen path) -- the reason the two lanes
// cannot print a different next command for the same refusal.
func renderRefusal(r refusal) string {
	var b strings.Builder
	b.WriteString(renderBanner("⛔", "Refused"))
	b.WriteString(visualDividerStr())
	// The What, Why, and closing lines each carry a leading glyph from the
	// one shared table (voiceGlyphMap, cmd/codex_visuals.go;
	// TestVoiceGlyphsHaveOneTable forbids a second table) so this screen
	// meets the corpus-wide voice density gate
	// (TestEveryVoicedScreenMeetsTheReferenceDensity). The "Next: `...`"
	// line is deliberately left with NO leading glyph or other prefix:
	// journeyPrintedRefusalNextLineRe (cmd/journey.go) matches it anchored
	// at the start of the line, and 208-07's printed-refusal extractor
	// depends on that exact shape to find and re-run the command.
	b.WriteString(voiceLine("blocked", strings.TrimSpace(r.What)))
	b.WriteString("\n")
	if why := strings.TrimSpace(r.Why); why != "" {
		b.WriteString(voiceLine("evidence", "Why: "+why))
		b.WriteString("\n")
	}
	next := strings.TrimSpace(r.NextCommand)
	b.WriteString("Next: `")
	b.WriteString(next)
	b.WriteString("`\n")
	// 208-09-PLAN.md (D-01): the same shared guidance sentence Error()
	// carries, drawn through voiceLine like every other line in this
	// block -- never a bare line, which would drag this screen's symbol
	// density below TestEveryVoicedScreenMeetsTheReferenceDensity's floor.
	// The "Next: `...`" line above is left untouched: two extractors
	// (journeyPrintedRefusalNextLineRe, cmd/journey.go, and the printed
	// command runner it feeds) anchor on its exact shape.
	if r.ProtectsWork && next != "" && sessionHasNoOneToAsk() {
		b.WriteString(voiceLine("next", unattendedGuidanceSentence))
		b.WriteString("\n")
	}
	for _, step := range r.ExtraSteps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		b.WriteString("  - ")
		b.WriteString(step)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(voiceLine("warning", "This is a limit in Aether itself, not a mistake you made. Run `aether report` and send the file it writes to whoever maintains Aether -- please do not edit Aether's own program files to work around this."))
	b.WriteString("\n")
	return b.String()
}
