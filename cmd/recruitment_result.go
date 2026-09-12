package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// recruitmentResultsPath is the store-relative path bindRecruitmentResult
// persists to. New data file, per this plan's frontmatter.
const recruitmentResultsPath = "recruitment/results.json"

// recruitmentResult is the exactly-once completion record for one admitted
// recruitment, shaped like colony.PauseHandoff's own ID + Transaction +
// optional Receipt discipline (pkg/colony/lifecycle.go) -- the same pattern
// this codebase already trusts for replay-safe completion binding, reused
// verbatim rather than inventing a second idempotency mechanism (SYN-203-06).
type recruitmentResult struct {
	SchemaVersion  string                               `json:"schema_version"`
	RecruitmentID  string                               `json:"recruitment_id"`
	IntentID       string                               `json:"intent_id"`
	ChildName      string                               `json:"child_name"`
	ParentName     string                               `json:"parent_name"`
	TerminalStatus string                               `json:"terminal_status"`
	Summary        string                               `json:"summary,omitempty"`
	Transaction    colony.LifecycleTransactionReference `json:"transaction"`
	Receipt        *colony.LifecycleReceiptReference    `json:"receipt,omitempty"`
}

// recruitmentResultsFile is the on-disk container at recruitmentResultsPath.
type recruitmentResultsFile struct {
	Entries []recruitmentResult `json:"entries"`
}

// errRecruitmentResultAlreadyBound is the internal replay sentinel
// bindRecruitmentResult returns from its own UpdateJSONAtomically mutate
// closure to abort the write on a replay -- UpdateJSONAtomically's own
// contract is "if mutate returns an error, no write occurs" (pkg/storage),
// which is exactly the "mutates nothing" guarantee a replay requires.
var errRecruitmentResultAlreadyBound = errors.New("recruitment result already bound")

// bindRecruitmentResult stores result under recruitment/results.json, keyed
// on RecruitmentID. A second call carrying an already-stored RecruitmentID
// returns the STORED record and mutates nothing -- the exact exactly-once
// discipline colony.PauseHandoff's own replay path already establishes. Do
// not invent a second idempotency mechanism here.
func bindRecruitmentResult(result recruitmentResult) (recruitmentResult, error) {
	if store == nil {
		return recruitmentResult{}, fmt.Errorf("no store initialized")
	}
	if strings.TrimSpace(result.RecruitmentID) == "" {
		return recruitmentResult{}, fmt.Errorf("recruitment result requires a non-empty RecruitmentID")
	}

	var bound recruitmentResult
	var file recruitmentResultsFile
	err := store.UpdateJSONAtomically(recruitmentResultsPath, &file, func() error {
		for _, existing := range file.Entries {
			if existing.RecruitmentID == result.RecruitmentID {
				bound = existing
				return errRecruitmentResultAlreadyBound
			}
		}
		file.Entries = append(file.Entries, result)
		bound = result
		return nil
	})
	if err != nil && !errors.Is(err, errRecruitmentResultAlreadyBound) {
		return recruitmentResult{}, err
	}
	return bound, nil
}
