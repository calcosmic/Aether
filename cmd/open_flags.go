package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// classifiedFlags groups the unresolved rows of pending-decisions.json into
// the three kinds a flag can be: a blocker (stops advancement), an issue
// (tracked, does not stop advancement), or a note (a "deal with this later"
// reminder). Every other row type in the same file -- a clarification, a
// boundary answer, an autopilot checkpoint, or anything unrecognised -- is a
// different kind of pending-decision row and is never counted as a flag at
// all. This matches renderPhaseEndFlagsSection's skip list
// (cmd/phase_end_footer.go), the classic end-of-phase footer that already
// got this right.
type classifiedFlags struct {
	Blockers []colony.FlagEntry
	Issues   []colony.FlagEntry
	Notes    []colony.FlagEntry
}

// classifyOpenFlags is the one counting rule for open (unresolved) flags.
// Before this, three readers each re-derived their own switch statement over
// FlagEntry.Type and disagreed about what an unrecognised type counts as
// (cmd/status.go's countStatusNonBlockerFlags defaulted it to a note,
// cmd/flags.go's renderFlagsTable defaulted it to an issue, and
// cmd/reconcile.go's inspectReconcileFlags also defaulted it to a note) --
// so the same pending-decisions.json file could show a different flag count
// on every screen. Every reader below calls this function instead of its own
// switch, so they cannot drift apart again.
//
// cmd/flag_cmds.go's flag-check-blockers diagnostic is a deliberate
// exception: it keeps its own older counting rule (unknown non-blocker rows
// count as issues), preserved on purpose for backward compatibility and
// locked by name in TestFlagCheckBlockersAndStatusShareSnapshot.
func classifyOpenFlags(entries []colony.FlagEntry) classifiedFlags {
	var c classifiedFlags
	for _, entry := range entries {
		if entry.Resolved {
			continue
		}
		switch normalizedFlagType(entry.Type) {
		case "blocker":
			c.Blockers = append(c.Blockers, entry)
		case "issue":
			c.Issues = append(c.Issues, entry)
		case "note":
			c.Notes = append(c.Notes, entry)
		}
	}
	return c
}

// normalizedFlagType lower-cases and trims a flag's type so "Issue", " issue ",
// and "issue" all classify the same way.
func normalizedFlagType(flagType string) string {
	return strings.ToLower(strings.TrimSpace(flagType))
}

// maxOpenFlagsShownPerGroup caps each group at five titles before folding
// the rest into "and N more" -- the owner's own bar (plan 1.0.85, Part C2).
const maxOpenFlagsShownPerGroup = 5

// renderOpenFlagsSection is the headed "things to deal with" list shown
// beneath the status screen's flag counts: blocking work first (it stops
// progress), then issues, then notes for later -- up to five titles per
// group, then "and N more". Parked (acknowledged) items are marked. It
// returns "" when there is nothing open, so callers can skip the section
// entirely rather than print an empty heading.
//
// Both the live dashboard (renderDashboard) and the archived-project screen
// (renderArchivedProjectStatusVisual) call this one function, so a finished,
// archived project shows the exact same open items in the exact same words
// as an active one.
func renderOpenFlagsSection(s *storage.Store) string {
	flags, ok := loadFlagsFile(s)
	if !ok {
		return ""
	}
	c := classifyOpenFlags(flags.Decisions)
	if len(c.Blockers) == 0 && len(c.Issues) == 0 && len(c.Notes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderStageMarker("Things to deal with"))
	writeOpenFlagGroup(&b, "blocked", "Blocking work -- stops progress until it is resolved", c.Blockers)
	writeOpenFlagGroup(&b, "flag", "Issues", c.Issues)
	writeOpenFlagGroup(&b, "feedback", "For later", c.Notes)
	if command, _, ok := candidateCommand(candidateFlags); ok {
		fmt.Fprintf(&b, "%s\n", voiceLine("next", "See the full list: `"+command+"`"))
	}
	return b.String()
}

// writeOpenFlagGroup writes one heading plus up to maxOpenFlagsShownPerGroup
// titles for a single flag group. It writes nothing for an empty group.
func writeOpenFlagGroup(b *strings.Builder, glyphKind, heading string, entries []colony.FlagEntry) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(b, "%s\n", voiceLine(glyphKind, fmt.Sprintf("%s (%d)", heading, len(entries))))
	shown := entries
	if len(shown) > maxOpenFlagsShownPerGroup {
		shown = shown[:maxOpenFlagsShownPerGroup]
	}
	for _, entry := range shown {
		title := strings.TrimSpace(entry.Description)
		if title == "" {
			title = "(no title recorded)"
		}
		marker := ""
		if entry.Acknowledged {
			marker = " -- parked"
		}
		fmt.Fprintf(b, "   - %s%s\n", title, marker)
	}
	if extra := len(entries) - len(shown); extra > 0 {
		fmt.Fprintf(b, "   - and %d more\n", extra)
	}
}

// filterCarriedFlags is the ONE carry-forward rule, shared by `aether init`
// and `aether entomb` so an owner's open note or issue means exactly the
// same thing at both points and can never quietly drift apart. A blocker
// would stop the new project's very first check, and a clarification, an
// autopilot checkpoint, or any other protected decision belongs only to the
// conversation that produced it -- none of those survive. An unresolved row
// typed exactly "issue" or "note", and not a protected decision awaiting its
// own bound answer (flagRequiresBoundDecisionAnswer), carries forward with
// its phase number cleared, since that phase belonged to the finished
// project's plan.
func filterCarriedFlags(entries []colony.FlagEntry) []colony.FlagEntry {
	kept := make([]colony.FlagEntry, 0, len(entries))
	for _, flag := range entries {
		if flag.Resolved {
			continue
		}
		flagType := normalizedFlagType(flag.Type)
		if flagType != "issue" && flagType != "note" {
			continue
		}
		if flagRequiresBoundDecisionAnswer(flag) {
			continue
		}
		flag.Phase = nil
		kept = append(kept, flag)
	}
	return kept
}

// writeCarriedFlagsFile is the ONE writer for pending-decisions.json after
// filtering: both `aether init` and `aether entomb` (via
// restoreCarriedFlagsAfterEntomb) call it. When nothing survives the filter
// it removes the file entirely, matching the old unconditional-delete
// behaviour for the empty case. Otherwise it writes atomically -- a temp
// file in the same directory, then a rename -- so a reader can never observe
// a half-written file, and it refuses to write through anything that is not
// an ordinary file (the same guard writeProjectChangelogEntry uses in
// cmd/project_changelog.go): a symlink could point outside the project, and
// a directory is not ours to replace.
func writeCarriedFlagsFile(dataDir string, ff colony.FlagsFile, kept []colony.FlagEntry) error {
	path := filepath.Join(dataDir, pendingDecisionsFile)
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not an ordinary file, refusing to write through it", pendingDecisionsFile)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(kept) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	ff.Decisions = kept
	encoded, err := json.MarshalIndent(ff, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	temp, err := os.CreateTemp(dataDir, ".pending-decisions-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	_, writeErr := temp.Write(encoded)
	closeErr := temp.Close()
	if writeErr == nil && closeErr == nil {
		writeErr = os.Chmod(tempPath, 0o644)
	}
	if writeErr == nil && closeErr == nil {
		writeErr = os.Rename(tempPath, path)
	}
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tempPath)
		if writeErr == nil {
			writeErr = closeErr
		}
		return writeErr
	}
	return nil
}

// carryForwardOpenFlagsAcrossInit narrows what used to be an unconditional
// delete of pending-decisions.json at the start of `aether init`
// (RUNTIME-01, TestInitClearsPriorColonyDecisionResidue). See
// filterCarriedFlags for the rule and writeCarriedFlagsFile for how the
// result is written.
//
// Init's own mandatory pre-init backup (a few lines below this call in
// init_cmd.go) already holds a full, unfiltered copy of the file, and a
// successful `aether entomb` archives a full copy into its chamber before
// init is ever reachable in the ordinary flow -- so narrowing this delete
// rather than keeping it unconditional loses nothing recoverable.
func carryForwardOpenFlagsAcrossInit(dataDir string) error {
	path := filepath.Join(dataDir, pendingDecisionsFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var ff colony.FlagsFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		// Malformed content carries nothing forward -- clear it exactly as
		// the old unconditional delete did, so nothing broken leaks into the
		// new colony either.
		return os.Remove(path)
	}
	return writeCarriedFlagsFile(dataDir, ff, filterCarriedFlags(ff.Decisions))
}
