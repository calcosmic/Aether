package cmd

// Platform honesty cards (PROOF-04 / CAP-054).
//
// Task 1 of Phase 205 Plan 10 requires one plain-English card per platform
// (Claude Code, OpenCode, Codex) where every claim is bound, by an explicit
// transcript-line citation, to a real capture of a real platform run made in
// this phase. This file holds the parser/validator that enforces that
// binding, plus the tests that run it against the three shipped cards.
//
// Card format (see the three 205-PLATFORM-CARD-*.md files):
//
//	## What worked
//	- Claim text. [transcript: opencode-run.txt:2 | "verbatim quoted line"]
//
//	## What stopped early with a truthful message
//	- Claim text. [transcript: opencode-run.txt:6 | "verbatim quoted line"]
//
//	## What is not supported
//	- Claim text. [transcript: opencode-run.txt:6 | "verbatim quoted line"]
//
// A claim may additionally carry a [ledger: <episode-id>] token when it
// describes an ability as run, tracked, or accounted for by this program
// (see TestPlatformCardsDoNotClaimUngovernedAbilityAsGoverned below) --
// that token must resolve to a real entry in this program's own episode
// ledger, never a bare platform-native ability.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// platformHonestyCardEntry is one bullet on a card: a plain-English claim
// bound to a specific, verbatim-checked line of a captured transcript.
type platformHonestyCardEntry struct {
	Claim           string
	TranscriptFile  string
	TranscriptLine  int
	TranscriptQuote string
	LedgerEpisodeID string // set only when the claim carries a [ledger: ...] token
	Raw             string
}

// platformHonestyCard is one platform's card, parsed into its three
// required lists.
type platformHonestyCard struct {
	Platform     string
	Worked       []platformHonestyCardEntry
	StoppedEarly []platformHonestyCardEntry
	NotSupported []platformHonestyCardEntry
}

func (c platformHonestyCard) allEntries() []platformHonestyCardEntry {
	out := make([]platformHonestyCardEntry, 0, len(c.Worked)+len(c.StoppedEarly)+len(c.NotSupported))
	out = append(out, c.Worked...)
	out = append(out, c.StoppedEarly...)
	out = append(out, c.NotSupported...)
	return out
}

// platformHonestyCardPath returns the repo-root-relative path to the given
// platform's card. platform is matched case-insensitively (e.g. "opencode",
// "OpenCode", "OPENCODE" all resolve to the same file).
func platformHonestyCardPath(platform string) string {
	return filepath.Join(
		".planning", "phases", "205-owner-acceptance-and-restoration-seal",
		fmt.Sprintf("205-PLATFORM-CARD-%s.md", strings.ToUpper(platform)),
	)
}

// platformRunTranscriptPath returns the repo-root-relative path to the
// given platform's captured run transcript.
func platformRunTranscriptPath(platform string) string {
	return filepath.Join(
		".planning", "phases", "205-owner-acceptance-and-restoration-seal",
		"evidence", "platform-runs",
		fmt.Sprintf("%s-run.txt", strings.ToLower(platform)),
	)
}

var (
	platformCardTranscriptRefPattern = regexp.MustCompile(`\[transcript:\s*([\w.\-]+):(\d+)\s*\|\s*"((?:[^"\\]|\\.)*)"\]`)
	platformCardLedgerRefPattern     = regexp.MustCompile(`\[ledger:\s*([\w.\-]+)\]`)
)

// loadPlatformHonestyCard reads and parses a card at an absolute or
// cwd-relative path into its three lists. It fails if any of the three
// required section headings is missing, or if a bullet under a recognized
// section carries no parseable [transcript: ...] reference.
func loadPlatformHonestyCard(path string) (platformHonestyCard, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return platformHonestyCard{}, fmt.Errorf("reading card %s: %w", path, err)
	}

	card := platformHonestyCard{}
	sawWorked, sawStoppedEarly, sawNotSupported := false, false, false

	type section int
	const (
		sectionNone section = iota
		sectionWorked
		sectionStoppedEarly
		sectionNotSupported
	)
	current := sectionNone

	lines := strings.Split(string(raw), "\n")
	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			heading := strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			switch {
			case strings.Contains(heading, "what worked"):
				current = sectionWorked
				sawWorked = true
			case strings.Contains(heading, "stopped early"):
				current = sectionStoppedEarly
				sawStoppedEarly = true
			case strings.Contains(heading, "not supported"):
				current = sectionNotSupported
				sawNotSupported = true
			default:
				current = sectionNone
			}
			continue
		}

		if current == sectionNone || !strings.HasPrefix(trimmed, "- ") {
			continue
		}

		entry, perr := parsePlatformHonestyCardBullet(trimmed)
		if perr != nil {
			return platformHonestyCard{}, fmt.Errorf("%s line %d: %w", path, lineNum+1, perr)
		}

		switch current {
		case sectionWorked:
			card.Worked = append(card.Worked, entry)
		case sectionStoppedEarly:
			card.StoppedEarly = append(card.StoppedEarly, entry)
		case sectionNotSupported:
			card.NotSupported = append(card.NotSupported, entry)
		}
	}

	var missing []string
	if !sawWorked {
		missing = append(missing, "## What worked")
	}
	if !sawStoppedEarly {
		missing = append(missing, "## What stopped early with a truthful message")
	}
	if !sawNotSupported {
		missing = append(missing, "## What is not supported")
	}
	if len(missing) > 0 {
		return platformHonestyCard{}, fmt.Errorf("%s missing required section(s): %s", path, strings.Join(missing, ", "))
	}

	return card, nil
}

func parsePlatformHonestyCardBullet(bulletLine string) (platformHonestyCardEntry, error) {
	body := strings.TrimPrefix(bulletLine, "- ")

	match := platformCardTranscriptRefPattern.FindStringSubmatch(body)
	if match == nil {
		return platformHonestyCardEntry{}, fmt.Errorf("claim %q has no transcript reference (expected a trailing [transcript: file:line | \"quote\"] token)", body)
	}

	lineNum, err := strconv.Atoi(match[2])
	if err != nil {
		return platformHonestyCardEntry{}, fmt.Errorf("claim %q has a non-numeric transcript line %q", body, match[2])
	}

	claim := strings.TrimSpace(body[:strings.Index(body, match[0])])

	entry := platformHonestyCardEntry{
		Claim:           claim,
		TranscriptFile:  match[1],
		TranscriptLine:  lineNum,
		TranscriptQuote: match[3],
		Raw:             body,
	}

	if lm := platformCardLedgerRefPattern.FindStringSubmatch(body); lm != nil {
		entry.LedgerEpisodeID = lm[1]
	}

	return entry, nil
}

// validatePlatformHonestyCard checks every entry on the card against the
// captured transcript: the cited file must match, the cited line must
// exist, and the cited line's actual content must contain the entry's
// quoted text verbatim. A claim with no reference, or a reference that does
// not resolve, fails naming the claim.
func validatePlatformHonestyCard(card platformHonestyCard, transcript string) error {
	transcriptBytes, err := os.ReadFile(transcript)
	if err != nil {
		return fmt.Errorf("reading transcript %s: %w", transcript, err)
	}
	transcriptLines := strings.Split(string(transcriptBytes), "\n")
	transcriptBase := filepath.Base(transcript)

	for _, entry := range card.allEntries() {
		if entry.TranscriptFile == "" || entry.TranscriptLine == 0 {
			return fmt.Errorf("claim %q carries no transcript reference", entry.Claim)
		}
		if entry.TranscriptFile != transcriptBase {
			return fmt.Errorf("claim %q references transcript file %q but this card's transcript is %q", entry.Claim, entry.TranscriptFile, transcriptBase)
		}
		if entry.TranscriptLine < 1 || entry.TranscriptLine > len(transcriptLines) {
			return fmt.Errorf("claim %q references line %d, but %s only has %d lines", entry.Claim, entry.TranscriptLine, transcriptBase, len(transcriptLines))
		}
		actual := transcriptLines[entry.TranscriptLine-1]
		if !strings.Contains(actual, entry.TranscriptQuote) {
			return fmt.Errorf("claim %q quotes %q for %s:%d, but that line actually reads %q", entry.Claim, entry.TranscriptQuote, transcriptBase, entry.TranscriptLine, actual)
		}
	}

	return nil
}

func platformHonestyCardRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}
	return root
}

func loadAndValidatePlatformCard(t *testing.T, platform string) platformHonestyCard {
	t.Helper()
	root := platformHonestyCardRepoRoot(t)
	cardPath := filepath.Join(root, platformHonestyCardPath(platform))
	transcriptPath := filepath.Join(root, platformRunTranscriptPath(platform))

	if info, err := os.Stat(transcriptPath); err != nil || info.Size() == 0 {
		t.Fatalf("captured transcript %s must exist and be non-empty: %v", transcriptPath, err)
	}

	card, err := loadPlatformHonestyCard(cardPath)
	if err != nil {
		t.Fatalf("loading %s card: %v", platform, err)
	}

	if err := validatePlatformHonestyCard(card, transcriptPath); err != nil {
		t.Fatalf("%s card claim does not resolve to the captured run: %v", platform, err)
	}

	return card
}

func TestOpenCodeCardClaimsResolveToTheCapturedRun(t *testing.T) {
	card := loadAndValidatePlatformCard(t, "opencode")
	if len(card.allEntries()) == 0 {
		t.Fatalf("OpenCode card has no claims at all")
	}
}

func TestCodexCardClaimsResolveToTheCapturedRun(t *testing.T) {
	card := loadAndValidatePlatformCard(t, "codex")
	if len(card.allEntries()) == 0 {
		t.Fatalf("Codex card has no claims at all")
	}
}

func TestClaudeCardClaimsResolveToTheCapturedRun(t *testing.T) {
	card := loadAndValidatePlatformCard(t, "claude")
	if len(card.allEntries()) == 0 {
		t.Fatalf("Claude card has no claims at all")
	}
}

// TestEveryContractedPlatformHasACardAndARun iterates the real,
// user-facing platforms the versioned support contract knows about
// (PlatformFake is support_tier "test_only" and is not a user platform, so
// it is deliberately excluded) and fails naming any platform missing
// either its card or its captured run.
func TestEveryContractedPlatformHasACardAndARun(t *testing.T) {
	root := platformHonestyCardRepoRoot(t)

	platforms := []string{"claude", "opencode", "codex"}
	var problems []string

	for _, platform := range platforms {
		cardPath := filepath.Join(root, platformHonestyCardPath(platform))
		if info, err := os.Stat(cardPath); err != nil || info.Size() == 0 {
			problems = append(problems, fmt.Sprintf("%s: missing or empty card at %s", platform, cardPath))
		}

		transcriptPath := filepath.Join(root, platformRunTranscriptPath(platform))
		if info, err := os.Stat(transcriptPath); err != nil || info.Size() == 0 {
			problems = append(problems, fmt.Sprintf("%s: missing or empty captured run at %s", platform, transcriptPath))
		}
	}

	if len(problems) > 0 {
		t.Fatalf("platform(s) missing a card and/or a run:\n%s", strings.Join(problems, "\n"))
	}
}

// --- Governance and future-work checks (Task 3) ---

// platformCardGovernancePhrases names the wording a card claim must never
// use unless it can point at a real record in this program's own episode
// ledger. Aether's own dispatcher recording, tracking, or accounting for a
// platform ability is a fact about THIS PROGRAM's ledger, never a fact a
// card is free to assert on the strength of a platform simply having run.
var platformCardGovernancePhrases = []string{
	"the program tracked",
	"the program recorded",
	"the program logged",
	"the program accounted for",
	"aether tracked",
	"aether recorded",
	"aether's ledger",
	"the program's own ledger",
	"governed by aether",
	"governed by the program",
}

func minimalEpisodeLedgerRecordIDs(root string) (map[string]bool, error) {
	ledgerPath := filepath.Join(root, ".aether", "data", "episodes", "ledger.json")
	raw, err := os.ReadFile(ledgerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}

	var file episodeLedgerFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ledgerPath, err)
	}

	ids := make(map[string]bool, len(file.Entries))
	for _, entry := range file.Entries {
		ids[entry.EpisodeID] = true
		ids[entry.RecordID] = true
	}
	return ids, nil
}

// validatePlatformCardGovernanceClaims fails naming the first claim that
// uses program-governance wording without a [ledger: <id>] token that
// resolves to a real entry in ledgerIDs.
func validatePlatformCardGovernanceClaims(card platformHonestyCard, ledgerIDs map[string]bool) error {
	for _, entry := range card.allEntries() {
		lower := strings.ToLower(entry.Claim)
		usesGovernanceWording := false
		for _, phrase := range platformCardGovernancePhrases {
			if strings.Contains(lower, phrase) {
				usesGovernanceWording = true
				break
			}
		}
		if !usesGovernanceWording {
			continue
		}
		if entry.LedgerEpisodeID == "" {
			return fmt.Errorf("claim %q describes an ability as governed by this program but carries no [ledger: <id>] token", entry.Claim)
		}
		if !ledgerIDs[entry.LedgerEpisodeID] {
			return fmt.Errorf("claim %q cites ledger id %q, which does not exist in this program's own episode ledger", entry.Claim, entry.LedgerEpisodeID)
		}
	}
	return nil
}

func TestPlatformCardsDoNotClaimUngovernedAbilityAsGoverned(t *testing.T) {
	root := platformHonestyCardRepoRoot(t)
	ledgerIDs, err := minimalEpisodeLedgerRecordIDs(root)
	if err != nil {
		t.Fatalf("reading episode ledger: %v", err)
	}

	for _, platform := range []string{"claude", "opencode", "codex"} {
		cardPath := filepath.Join(root, platformHonestyCardPath(platform))
		card, err := loadPlatformHonestyCard(cardPath)
		if err != nil {
			t.Fatalf("loading %s card: %v", platform, err)
		}
		if err := validatePlatformCardGovernanceClaims(card, ledgerIDs); err != nil {
			t.Errorf("%s card: %v", platform, err)
		}
	}

	t.Run("synthetic_ungoverned_claim_is_rejected", func(t *testing.T) {
		synthetic := platformHonestyCard{
			Platform: "synthetic",
			Worked: []platformHonestyCardEntry{
				{Claim: "the program tracked this OpenCode run end to end", TranscriptFile: "x.txt", TranscriptLine: 1, TranscriptQuote: "x"},
			},
		}
		if err := validatePlatformCardGovernanceClaims(synthetic, map[string]bool{}); err == nil {
			t.Fatalf("expected a governance claim with no ledger token to be rejected, got nil error")
		}
	})
}

// platformCardFutureWorkPhrases names forward-looking wording that is
// never allowed on the two secondary platforms' cards this milestone (the
// owner explicitly excluded engineering effort on OpenCode and Codex --
// 205-CONTEXT.md decision D-13). A card describes what happened, not what
// could be made to happen.
var platformCardFutureWorkPhrases = []string{
	"will be added",
	"will be supported",
	"will support",
	"will work",
	"coming soon",
	"planned for",
	"future work",
	"to be implemented",
	"we will add",
	"we plan to",
	"in a future release",
	"is expected to",
}

func validatePlatformCardPromisesNoFutureWork(card platformHonestyCard) error {
	for _, entry := range card.allEntries() {
		lower := strings.ToLower(entry.Claim)
		for _, phrase := range platformCardFutureWorkPhrases {
			if strings.Contains(lower, phrase) {
				return fmt.Errorf("claim %q makes a forward-looking promise (%q) on a secondary platform with no engineering effort this milestone", entry.Claim, phrase)
			}
		}
	}
	return nil
}

func TestPlatformCardsPromiseNoFutureWork(t *testing.T) {
	root := platformHonestyCardRepoRoot(t)

	// Only the two secondary platforms (OpenCode, Codex) are checked --
	// Claude Code is the platform the owner decided has to work, and this
	// milestone's D-13 no-engineering-effort ruling names OpenCode and
	// Codex specifically.
	for _, platform := range []string{"opencode", "codex"} {
		cardPath := filepath.Join(root, platformHonestyCardPath(platform))
		card, err := loadPlatformHonestyCard(cardPath)
		if err != nil {
			t.Fatalf("loading %s card: %v", platform, err)
		}
		if err := validatePlatformCardPromisesNoFutureWork(card); err != nil {
			t.Errorf("%s card: %v", platform, err)
		}
	}

	t.Run("synthetic_future_promise_is_rejected", func(t *testing.T) {
		synthetic := platformHonestyCard{
			Platform: "synthetic",
			NotSupported: []platformHonestyCardEntry{
				{Claim: "named-caste routing will be supported in a future release", TranscriptFile: "x.txt", TranscriptLine: 1, TranscriptQuote: "x"},
			},
		}
		if err := validatePlatformCardPromisesNoFutureWork(synthetic); err == nil {
			t.Fatalf("expected a forward-looking promise to be rejected, got nil error")
		}
	})
}
