package cmd

import (
	"fmt"
	"os"

	"github.com/calcosmic/Aether/pkg/colony"
)

// resolveAuthoredPhaseMode decides a phase's mode ONCE, at authoring time, and
// the result is written durably into COLONY_STATE.json.
//
// Precedence: an explicit, valid mode declared by the planner always wins.
// Only when the planner declared nothing does keyword inference run — as an
// authoring-time default whose result is visible, durable, and revisable via
// plan revision, never as runtime control flow. When inference is used, it
// says so on stderr, because a phase silently classified as "discovery" by the
// word "research" in its prose once dispatched an Oracle instead of a Builder.
func resolveAuthoredPhaseMode(declared, name, description string) colony.PhaseMode {
	if mode := colony.PhaseMode(declared); mode.Valid() {
		return mode
	}
	inferred := colony.InferPhaseMode(name, description)
	fmt.Fprintf(os.Stderr, "note: phase %q declared no mode; defaulting to %q from its wording — declare `mode` explicitly in the plan to override\n", name, inferred)
	return inferred
}
