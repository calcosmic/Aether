package cmd

import (
	"fmt"
	"io"
)

// buildVerbose controls whether verbose output (raw worker output) is printed.
// When false (default), filteredPrintln/filteredFprintf/filteredFprintln silently discard.
var buildVerbose bool

// setBuildVerbose sets the package-level verbose flag.
func setBuildVerbose(v bool) {
	buildVerbose = v
}

// filteredPrintln writes a newline-terminated message to stdout only when verbose is true.
// This targets raw worker output -- ceremony events and stage markers use emitVisualProgress directly.
func filteredPrintln(a ...interface{}) {
	if !buildVerbose {
		return
	}
	fmt.Fprintln(stdout, a...)
}

// filteredFprintf writes a formatted message to stdout only when verbose is true.
func filteredFprintf(format string, a ...interface{}) {
	if !buildVerbose {
		return
	}
	fmt.Fprintf(stdout, format, a...)
}

// filteredFprintln writes a newline-terminated message to the given writer only when verbose is true.
func filteredFprintln(w io.Writer, a ...interface{}) {
	if !buildVerbose {
		return
	}
	fmt.Fprintln(w, a...)
}
