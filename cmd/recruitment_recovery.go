package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/spf13/cobra"
)

// recruitmentRecoveryClass is the closed, exhaustive vocabulary of ways a
// child's recruitment result can be found during recovery. Every member
// below MUST have a next action in recruitmentRecoveryNextAction; a class
// with no next action there returns "" and fails
// TestRecruitmentRecoveryClassesHaveNextActions by name.
type recruitmentRecoveryClass string

const (
	// recruitmentRecoveryMissing: an admitted recruitment (a recorded
	// spawn-tree amendment) has no bound result and no live process --
	// either the dispatch never ran to completion, or the whole `aether
	// recruit` invocation was killed before it could bind anything.
	recruitmentRecoveryMissing recruitmentRecoveryClass = "missing"
	// recruitmentRecoveryDuplicated: a second completion report for an
	// already-bound RecruitmentID disagrees with the stored one on content
	// -- an illegitimate duplicate, distinct from a safe replay.
	recruitmentRecoveryDuplicated recruitmentRecoveryClass = "duplicated"
	// recruitmentRecoveryAltered: a bound result's stored evidence digest no
	// longer matches the evidence file on disk -- tampering or corruption,
	// never treated as safe.
	recruitmentRecoveryAltered recruitmentRecoveryClass = "altered"
	// recruitmentRecoveryTimedOut: the child's dispatch exceeded its bounded
	// timeout before a result was bound. Partial evidence is preserved, not
	// discarded.
	recruitmentRecoveryTimedOut recruitmentRecoveryClass = "timed-out"
	// recruitmentRecoveryReplayed: a bound result already exists and either
	// no new report is being compared, or the new report is byte-for-byte
	// the same content -- a safe, read-only replay.
	recruitmentRecoveryReplayed recruitmentRecoveryClass = "replayed"
)

// recruitmentRecoveryClasses returns every declared recovery class,
// mirroring the ColonyLiveTopics()/ColonyLiveEpisodeKinds() completeness
// convention (pkg/events/colony_live.go): a member added to the const block
// above must also be added here, and to recruitmentRecoveryNextAction's
// switch.
func recruitmentRecoveryClasses() []recruitmentRecoveryClass {
	return []recruitmentRecoveryClass{
		recruitmentRecoveryMissing,
		recruitmentRecoveryDuplicated,
		recruitmentRecoveryAltered,
		recruitmentRecoveryTimedOut,
		recruitmentRecoveryReplayed,
	}
}

// recruitmentRecoveryQuery is everything classifyRecruitmentRecovery needs
// to answer "what state is this recruitment in, and what should the owner
// run next?" Candidate is nil for a pure status inspection (the `aether
// recruit --status` path); it is set when a NEW completion report is being
// checked against whatever is already stored, e.g. from a future
// finalization path that calls this before calling bindRecruitmentResult.
type recruitmentRecoveryQuery struct {
	RecruitmentID string
	ChildName     string
	Candidate     *recruitmentResult
}

// recruitmentRecoveryNextAction names the ONE owner-facing next action for
// class, given the identifiers in q. A class with no case here (i.e. any
// class added to recruitmentRecoveryClasses() without a matching case)
// falls through to "", which TestRecruitmentRecoveryClassesHaveNextActions
// fails on by name -- this is the mechanical guard the plan's acceptance
// criteria describe as "adding a sixth member without a next action makes a
// named test fail."
func recruitmentRecoveryNextAction(class recruitmentRecoveryClass, q recruitmentRecoveryQuery) string {
	recruitmentID := strings.TrimSpace(q.RecruitmentID)
	child := strings.TrimSpace(q.ChildName)
	if child == "" {
		child = "the recruited helper"
	}
	switch class {
	case recruitmentRecoveryMissing:
		return fmt.Sprintf(
			"recruitment %s never returned a result and no live process is recorded for %s -- run `aether recruit --status %s` again to confirm nothing changed, then re-submit the original recruitment request if the work still needs doing",
			recruitmentID, child, recruitmentID,
		)
	case recruitmentRecoveryDuplicated:
		return fmt.Sprintf(
			"recruitment %s received a second, conflicting completion report -- the stored result is kept and nothing was rewritten; run `aether recruit --status %s` to inspect it before resolving the conflict by hand",
			recruitmentID, recruitmentID,
		)
	case recruitmentRecoveryAltered:
		return fmt.Sprintf(
			"recruitment %s's stored evidence no longer matches the file(s) on disk -- restore the original evidence file, or re-run the recruitment to produce fresh evidence, then run `aether recruit --status %s` again",
			recruitmentID, recruitmentID,
		)
	case recruitmentRecoveryTimedOut:
		return fmt.Sprintf(
			"%s exceeded its bounded timeout on recruitment %s -- its partial evidence is preserved, not discarded; inspect it, then run `aether recruit --status %s` before deciding whether to retry",
			child, recruitmentID, recruitmentID,
		)
	case recruitmentRecoveryReplayed:
		return fmt.Sprintf(
			"recruitment %s already completed -- `aether recruit --status %s` is safe to run again at any time; no action is needed",
			recruitmentID, recruitmentID,
		)
	default:
		return ""
	}
}

// loadRecruitmentResultByID reads recruitment/results.json and returns the
// entry matching recruitmentID, if any. A missing results file is "not
// found", never an error -- a recruitment that has never had ANY result
// bound is the ordinary "missing" case, not a storage failure.
func loadRecruitmentResultByID(recruitmentID string) (recruitmentResult, bool, error) {
	if store == nil {
		return recruitmentResult{}, false, fmt.Errorf("no store initialized")
	}
	var file recruitmentResultsFile
	if err := store.LoadJSON(recruitmentResultsPath, &file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return recruitmentResult{}, false, nil
		}
		return recruitmentResult{}, false, err
	}
	for _, entry := range file.Entries {
		if entry.RecruitmentID == recruitmentID {
			return entry, true, nil
		}
	}
	return recruitmentResult{}, false, nil
}

// recruitmentEvidenceAltered applies the Phase 199 evidence classification
// rule unchanged (cmd/lifecycle_transaction.go's lifecycleRecoveryProvenance:
// a missing file is unknown, never altered; only a digest MISMATCH on a
// file that actually exists and reads counts as tampering): every one of
// result's Evidence entries that names both a Source path and a Digest is
// re-hashed against the file on disk. A missing evidence file is skipped
// (unknown, not altered); a read error other than "not exist" is returned as
// an error so the caller can fail closed rather than silently pass.
func recruitmentEvidenceAltered(result recruitmentResult) (bool, error) {
	for _, evidence := range result.Evidence {
		source := strings.TrimSpace(evidence.Source)
		digest := strings.TrimSpace(evidence.Digest)
		if source == "" || digest == "" {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return false, fmt.Errorf("read recruitment evidence %q: %w", source, err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != digest {
			return true, nil
		}
	}
	return false, nil
}

// classifyRecruitmentRecovery is the single classification chokepoint BIO-04
// exposes through the existing recovery reader (`aether recruit --status`,
// wired below) -- do not add a second recovery command. It NEVER writes;
// every branch reads durable state only.
func classifyRecruitmentRecovery(q recruitmentRecoveryQuery) (recruitmentRecoveryClass, string, error) {
	recruitmentID := strings.TrimSpace(q.RecruitmentID)
	if recruitmentID == "" {
		return "", "", fmt.Errorf("recruitment recovery classification requires a non-empty RecruitmentID")
	}
	q.RecruitmentID = recruitmentID

	stored, found, err := loadRecruitmentResultByID(recruitmentID)
	if err != nil {
		return "", "", err
	}

	if found {
		altered, alteredErr := recruitmentEvidenceAltered(stored)
		if alteredErr != nil {
			return "", "", alteredErr
		}
		if altered {
			return recruitmentRecoveryAltered, recruitmentRecoveryNextAction(recruitmentRecoveryAltered, q), nil
		}
		if q.Candidate != nil {
			if field := recruitmentResultContentDiff(stored, *q.Candidate); field != "" {
				return recruitmentRecoveryDuplicated, recruitmentRecoveryNextAction(recruitmentRecoveryDuplicated, q), nil
			}
		}
		return recruitmentRecoveryReplayed, recruitmentRecoveryNextAction(recruitmentRecoveryReplayed, q), nil
	}

	// No durable result exists yet. D-19 fail-closed discipline
	// (cmd/spawn_ancestor.go:95-98): an unreadable or absent spawn-tree
	// entry is "missing", never assumed to be anything safer.
	childName := strings.TrimSpace(q.ChildName)
	if childName != "" && store != nil {
		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if entries, parseErr := st.Parse(); parseErr == nil {
			for i := range entries {
				if entries[i].AgentName != childName {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(entries[i].Status), "timeout") {
					return recruitmentRecoveryTimedOut, recruitmentRecoveryNextAction(recruitmentRecoveryTimedOut, q), nil
				}
				break
			}
		}
	}
	return recruitmentRecoveryMissing, recruitmentRecoveryNextAction(recruitmentRecoveryMissing, q), nil
}

// runRecruitmentStatusCommand implements `aether recruit --status
// <recruitment-id>`, optionally paired with `--child <name>` so the missing
// vs. timed-out distinction can be made from the spawn-tree. It never
// dispatches anything and never writes.
func runRecruitmentStatusCommand(recruitmentID, childName string) {
	if store == nil {
		outputErrorMessage("no store initialized")
		return
	}
	class, nextAction, err := classifyRecruitmentRecovery(recruitmentRecoveryQuery{
		RecruitmentID: recruitmentID,
		ChildName:     childName,
	})
	if err != nil {
		outputError(2, fmt.Sprintf("failed to classify recruitment recovery: %v", err), nil)
		return
	}
	outputOK(map[string]interface{}{
		"recruitment_id": recruitmentID,
		"class":          string(class),
		"next_action":    nextAction,
	})
}

// init wires the --status/--child flags onto the EXISTING recruitCmd
// (cmd/recruitment.go, owned by 203-02/plan wave 4's file split) rather than
// registering a second command, per this plan's own instruction: "Expose
// the classification through the existing recovery reader rather than a new
// command." recruitCmd.RunE is captured and wrapped here, in this file,
// because cmd/recruitment.go is outside this plan's declared file list --
// this composes the new behavior onto the existing command from a sibling
// file in the same package rather than editing that file directly. Var
// initialization (recruitCmd's own composite literal, including its
// original RunE) completes before ANY init() function runs, in every Go
// build, so this wrap is correct regardless of init() ordering between this
// file and cmd/recruitment.go.
func init() {
	recruitCmd.Flags().String("status", "", "Report the recovery class and next action for a recruitment id, without dispatching anything")
	recruitCmd.Flags().String("child", "", "Child worker's recorded name, to distinguish missing from timed-out via the spawn tree (used only with --status)")

	originalRecruitRunE := recruitCmd.RunE
	recruitCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if recruitmentID, _ := cmd.Flags().GetString("status"); strings.TrimSpace(recruitmentID) != "" {
			childName, _ := cmd.Flags().GetString("child")
			runRecruitmentStatusCommand(strings.TrimSpace(recruitmentID), strings.TrimSpace(childName))
			return nil
		}
		return originalRecruitRunE(cmd, args)
	}
}
