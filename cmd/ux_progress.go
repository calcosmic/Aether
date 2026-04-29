package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ceremonyProgress wraps ceremony step execution with a progress indicator.
// In TTY environments it renders a progress bar; in non-TTY environments
// it falls back to plain-text step markers with elapsed timing.
type ceremonyProgress struct {
	bar     *progressbar.ProgressBar
	steps   []string
	current int
	start   time.Time
	tty     bool
	out     io.Writer
}

// newCeremonyProgress creates a progress tracker for the given ceremony steps.
func newCeremonyProgress(steps []string, out io.Writer) *ceremonyProgress {
	p := &ceremonyProgress{
		steps: steps,
		start: time.Now(),
		tty:   isTerminalWriter(out),
		out:   out,
	}
	if p.tty {
		p.bar = progressbar.NewOptions(len(steps),
			progressbar.OptionSetDescription("Starting ceremony..."),
			progressbar.OptionSetWriter(out),
			progressbar.OptionSetWidth(40),
			progressbar.OptionSetRenderBlankState(true),
		)
	}
	return p
}

// Advance marks the completion of a ceremony step and updates the progress display.
// If stepName is empty, the name is resolved from the steps array.
func (p *ceremonyProgress) Advance(stepName string) {
	p.current++
	name := stepName
	if name == "" && p.current <= len(p.steps) {
		name = p.steps[p.current-1]
	}
	if p.tty && p.bar != nil {
		if name != "" {
			p.bar.Describe(name)
		}
		_ = p.bar.Set(p.current)
	} else {
		fmt.Fprintf(p.out, "  Step %d/%d: %s (%s)\n", p.current, len(p.steps), name, time.Since(p.start).Round(time.Second))
	}
}

// Finish marks the ceremony as complete and prints the total elapsed time.
func (p *ceremonyProgress) Finish() {
	if p.tty && p.bar != nil {
		_ = p.bar.Finish()
		fmt.Fprintln(p.out)
	}
	fmt.Fprintf(p.out, "Ceremony complete in %s\n", time.Since(p.start).Round(time.Second))
}

// Steps returns the step names for the ceremony.
func (p *ceremonyProgress) Steps() []string {
	return p.steps
}

// NewCeremonyProgress is the exported constructor for ceremonyProgress.
func NewCeremonyProgress(steps []string, out io.Writer) *ceremonyProgress {
	return newCeremonyProgress(steps, out)
}
