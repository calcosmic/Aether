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
func (r refusal) Error() string {
	return fmt.Sprintf("%s — next: %s", strings.TrimSpace(r.What), strings.TrimSpace(r.NextCommand))
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
	b.WriteString(strings.TrimSpace(r.What))
	b.WriteString("\n")
	if why := strings.TrimSpace(r.Why); why != "" {
		b.WriteString("Why: ")
		b.WriteString(why)
		b.WriteString("\n")
	}
	b.WriteString("Next: `")
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
	b.WriteString("\nThis is a limit in Aether itself, not a mistake you made. Run `aether report` and send the file it writes to whoever maintains Aether -- please do not edit Aether's own program files to work around this.\n")
	return b.String()
}
