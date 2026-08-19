package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/storage"
)

// recordAutopilotRetryExhaustion writes one midden (failure-record) entry
// naming the phase whose build failed twice in a row -- once, then again
// after the autopilot's own single retry -- and both attempts' error text.
//
// It writes through the shared canonical midden writer (cmd/midden_shared.go's
// appendMiddenEntry, 188-01's output): the same file colony-prime, the
// autopilot's own pause-check, immune, and memory-health all now read, so a
// value written here is visible through every one of them without further
// plumbing.
//
// This is a directly unit-testable extraction of the "second failure -> give
// up" branch inside runCompatibilityAutopilot (cmd/compatibility_cmds.go),
// which previously discarded both attempts' error text silently. It does not
// swallow appendMiddenEntry's own error -- the caller decides whether a
// failed write should additionally matter; at this function's one call site,
// it must not (a failing write must never block the pause itself -- see
// T-188-15 and the call site's own comment).
func recordAutopilotRetryExhaustion(s *storage.Store, phaseID int, firstErr, secondErr error) error {
	firstText := autopilotRetryErrorText(firstErr)
	secondText := autopilotRetryErrorText(secondErr)

	var message string
	if firstErr != nil && secondErr != nil && firstText == secondText {
		// Both attempts failed the same way -- name the phase and the shared
		// cause once, but say plainly that two attempts happened so this
		// entry is never mistaken for a single failed try.
		message = fmt.Sprintf(
			"aether run: phase %d's build failed on both the first attempt and the automatic retry, the same way both times: %s",
			phaseID, secondText,
		)
	} else {
		message = fmt.Sprintf(
			"aether run: phase %d's build failed twice in a row, so autopilot paused instead of retrying again. First attempt: %s | Retry attempt: %s",
			phaseID, firstText, secondText,
		)
	}

	return appendMiddenEntry(s, "autopilot_retry_exhausted", "aether run", message, nil)
}

// autopilotRetryErrorText returns err's message, or a plain placeholder if
// err is nil. Defensive only: every real caller passes two non-nil errors
// (both branches of runCompatibilityAutopilot's retry only reach this
// function from inside their own `if err != nil` check), but a record-writer
// should never panic on a nil error slipping through.
func autopilotRetryErrorText(err error) string {
	if err == nil {
		return "(no error text recorded)"
	}
	return err.Error()
}
