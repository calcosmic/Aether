package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const planningMutationHelperEnvironment = "AETHER_PLANNING_MUTATION_SESSION_HELPER"

type planningMutationProcessResult struct {
	Observed string                         `json:"observed,omitempty"`
	Receipt  *planningTimelineAppendReceipt `json:"receipt,omitempty"`
	Error    string                         `json:"error,omitempty"`
}

type planningMutationChildProcess struct {
	command     *exec.Cmd
	output      *bytes.Buffer
	ready       *os.File
	release     *os.File
	result      *os.File
	holdReady   *os.File
	holdRelease *os.File
	cancel      context.CancelFunc
}

func TestPlanningMutationSession200(t *testing.T) {
	if os.Getenv(planningMutationHelperEnvironment) != "" {
		runPlanningMutationSessionHelper(t)
		return
	}

	t.Run("simultaneous processes derive beneath one root lock", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, ".aether", "data", "planning", "session-state.txt")
		mustWritePlanningMutationFile(t, target, []byte("base"))
		outside := t.TempDir()
		mustWritePlanningMutationFile(t, filepath.Join(outside, "sentinel.txt"), []byte("outside-stays"))
		outsideBefore := snapshotPlanningMutationTree(t, outside)

		first := startPlanningMutationChild(t, "session-append", root, "A", "", false)
		second := startPlanningMutationChild(t, "session-append", root, "B", "", false)
		awaitPlanningMutationPipe(t, first.ready, "first child ready")
		awaitPlanningMutationPipe(t, second.ready, "second child ready")
		releasePlanningMutationPipe(t, first.release)
		releasePlanningMutationPipe(t, second.release)

		firstResult := awaitPlanningMutationChild(t, first)
		secondResult := awaitPlanningMutationChild(t, second)
		if firstResult.Error != "" || secondResult.Error != "" {
			t.Fatalf("session children failed: first=%q second=%q", firstResult.Error, secondResult.Error)
		}
		final := string(mustReadPlanningMutationFile(t, target))
		observed := map[string]string{"A": firstResult.Observed, "B": secondResult.Observed}
		switch final {
		case "baseAB":
			if observed["A"] != "base" || observed["B"] != "baseA" {
				t.Fatalf("serialized observations = %#v for final %q", observed, final)
			}
		case "baseBA":
			if observed["B"] != "base" || observed["A"] != "baseB" {
				t.Fatalf("serialized observations = %#v for final %q", observed, final)
			}
		default:
			t.Fatalf("final state = %q, want both process updates exactly once", final)
		}
		if after := snapshotPlanningMutationTree(t, outside); !equalPlanningMutationSnapshot(outsideBefore, after) {
			t.Fatalf("session race changed outside tree\nbefore: %#v\n after: %#v", outsideBefore, after)
		}
	})

	t.Run("present absent write and delete baselines are exact", func(t *testing.T) {
		root := t.TempDir()
		dataRoot := filepath.Join(root, ".aether", "data")
		mustWritePlanningMutationFile(t, filepath.Join(dataRoot, "planning", "present.txt"), []byte("present-before"))
		mustWritePlanningMutationFile(t, filepath.Join(dataRoot, "planning", "delete.txt"), []byte("delete-before"))

		err := withPlanningMutationSession(root, "test-baseline-matrix", func(session *planningMutationSession) error {
			present, exists, err := session.ReadFile(lifecycleTransactionRootData, "planning/present.txt")
			if err != nil || !exists || string(present) != "present-before" {
				return fmt.Errorf("present baseline = %q/%v: %w", present, exists, err)
			}
			missing, exists, err := session.ReadFile(lifecycleTransactionRootData, "planning/create.txt")
			if err != nil || exists || missing != nil {
				return fmt.Errorf("absent baseline = %q/%v: %w", missing, exists, err)
			}
			removed, exists, err := session.ReadFile(lifecycleTransactionRootData, "planning/delete.txt")
			if err != nil || !exists || string(removed) != "delete-before" {
				return fmt.Errorf("delete baseline = %q/%v: %w", removed, exists, err)
			}

			undeclared, err := beginLifecycleTransaction(planningMutationTestConfig(root, "session-undeclared", session))
			if err != nil {
				return err
			}
			if err := undeclared.DeclareWrite(lifecycleTransactionRootData, "planning/not-read.txt", []byte("no")); err == nil || !strings.Contains(err.Error(), "session baseline") {
				return fmt.Errorf("undeclared target error = %v, want session baseline refusal", err)
			}

			tx, err := beginLifecycleTransaction(planningMutationTestConfig(root, "session-baseline-matrix", session))
			if err != nil {
				return err
			}
			if err := tx.DeclareWrite(lifecycleTransactionRootData, "planning/present.txt", []byte("present-after")); err != nil {
				return err
			}
			if err := tx.DeclareWrite(lifecycleTransactionRootData, "planning/create.txt", []byte("created")); err != nil {
				return err
			}
			if err := tx.DeclareRemoval(lifecycleTransactionRootData, "planning/delete.txt"); err != nil {
				return err
			}
			_, err = tx.Commit()
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := string(mustReadPlanningMutationFile(t, filepath.Join(dataRoot, "planning", "present.txt"))); got != "present-after" {
			t.Fatalf("present target = %q", got)
		}
		if got := string(mustReadPlanningMutationFile(t, filepath.Join(dataRoot, "planning", "create.txt"))); got != "created" {
			t.Fatalf("created target = %q", got)
		}
		if _, err := os.Lstat(filepath.Join(dataRoot, "planning", "delete.txt")); !os.IsNotExist(err) {
			t.Fatalf("deleted target still exists: %v", err)
		}
	})

	t.Run("stale deletion is mutation free", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, ".aether", "data", "planning", "stale-delete.txt")
		mustWritePlanningMutationFile(t, target, []byte("captured"))
		err := withPlanningMutationSession(root, "test-stale-delete", func(session *planningMutationSession) error {
			if _, exists, err := session.ReadFile(lifecycleTransactionRootData, "planning/stale-delete.txt"); err != nil || !exists {
				return fmt.Errorf("capture stale target: exists=%v err=%w", exists, err)
			}
			if err := os.WriteFile(target, []byte("concurrent-owner-data"), 0o644); err != nil {
				return err
			}
			tx, err := beginLifecycleTransaction(planningMutationTestConfig(root, "session-stale-delete", session))
			if err != nil {
				return err
			}
			if err := tx.DeclareRemoval(lifecycleTransactionRootData, "planning/stale-delete.txt"); err != nil {
				return err
			}
			if _, err := tx.Commit(); err == nil || !strings.Contains(err.Error(), "baseline changed") {
				return fmt.Errorf("stale delete error = %v, want deterministic baseline refusal", err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := string(mustReadPlanningMutationFile(t, target)); got != "concurrent-owner-data" {
			t.Fatalf("stale refusal changed target to %q", got)
		}
	})

	t.Run("every committed prefix rolls back without touching unrelated data", func(t *testing.T) {
		for _, faultPoint := range []string{
			"after_target_commit:target-0001",
			"after_target_commit:target-0002",
			"after_target_commit:target-0003",
		} {
			t.Run(strings.ReplaceAll(faultPoint, ":", "_"), func(t *testing.T) {
				root := t.TempDir()
				dataRoot := filepath.Join(root, ".aether", "data")
				first := filepath.Join(dataRoot, "planning", "first.txt")
				third := filepath.Join(dataRoot, "planning", "third.txt")
				unrelated := filepath.Join(dataRoot, "planning", "unrelated.txt")
				mustWritePlanningMutationFile(t, first, []byte("first-before"))
				mustWritePlanningMutationFile(t, third, []byte("third-before"))
				mustWritePlanningMutationFile(t, unrelated, []byte("unrelated-before"))
				injected := errors.New("planning session target fault")
				err := withPlanningMutationSession(root, "test-rollback-prefix", func(session *planningMutationSession) error {
					for _, target := range []string{"planning/first.txt", "planning/second.txt", "planning/third.txt"} {
						if _, _, err := session.ReadFile(lifecycleTransactionRootData, target); err != nil {
							return err
						}
					}
					config := planningMutationTestConfig(root, "session-rollback-"+strings.NewReplacer(":", "-", "_", "-").Replace(faultPoint), session)
					config.Fault = func(point string) error {
						if point == faultPoint {
							return injected
						}
						return nil
					}
					tx, err := beginLifecycleTransaction(config)
					if err != nil {
						return err
					}
					if err := tx.DeclareWrite(lifecycleTransactionRootData, "planning/first.txt", []byte("first-after")); err != nil {
						return err
					}
					if err := tx.DeclareWrite(lifecycleTransactionRootData, "planning/second.txt", []byte("second-after")); err != nil {
						return err
					}
					if err := tx.DeclareRemoval(lifecycleTransactionRootData, "planning/third.txt"); err != nil {
						return err
					}
					if _, err := tx.Commit(); !errors.Is(err, injected) {
						return fmt.Errorf("commit error = %v, want injected fault", err)
					}
					if err := os.WriteFile(unrelated, []byte("unrelated-concurrent"), 0o644); err != nil {
						return err
					}
					config.Fault = nil
					tx.config.Fault = nil
					return tx.Rollback()
				})
				if err != nil {
					t.Fatal(err)
				}
				if got := string(mustReadPlanningMutationFile(t, first)); got != "first-before" {
					t.Fatalf("first target after rollback = %q", got)
				}
				if _, err := os.Lstat(filepath.Join(dataRoot, "planning", "second.txt")); !os.IsNotExist(err) {
					t.Fatalf("created target survived rollback: %v", err)
				}
				if got := string(mustReadPlanningMutationFile(t, third)); got != "third-before" {
					t.Fatalf("removed target after rollback = %q", got)
				}
				if got := string(mustReadPlanningMutationFile(t, unrelated)); got != "unrelated-concurrent" {
					t.Fatalf("rollback replaced unrelated data with %q", got)
				}
			})
		}
	})

	t.Run("terminated process releases root lock and leaves recoverable rollback", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, ".aether", "data", "planning", "killed.txt")
		mustWritePlanningMutationFile(t, target, []byte("before-kill"))
		child := startPlanningMutationChild(t, "session-crash", root, "K", "", true)
		awaitPlanningMutationPipe(t, child.ready, "crash child ready")
		releasePlanningMutationPipe(t, child.release)
		awaitPlanningMutationPipe(t, child.holdReady, "crash child committed prefix")
		if err := child.command.Process.Kill(); err != nil {
			t.Fatalf("kill helper process: %v", err)
		}
		_ = child.holdRelease.Close()
		_, _ = io.ReadAll(child.result)
		_ = child.result.Close()
		if err := child.command.Wait(); err == nil {
			t.Fatal("killed helper exited successfully")
		}
		child.cancel()

		err := withPlanningMutationSession(root, "test-crash-rollback", func(session *planningMutationSession) error {
			config := planningMutationTestConfig(root, "session-crash-K", session)
			tx, err := beginLifecycleTransaction(config)
			if err != nil {
				return err
			}
			return tx.Rollback()
		})
		if err != nil {
			t.Fatalf("rollback killed transaction: %v", err)
		}
		if got := string(mustReadPlanningMutationFile(t, target)); got != "before-kill" {
			t.Fatalf("killed transaction rollback = %q", got)
		}
	})

	t.Run("linked data root is refused without outside mutation", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()
		mustWritePlanningMutationFile(t, filepath.Join(outside, "sentinel.txt"), []byte("outside"))
		before := snapshotPlanningMutationTree(t, outside)
		if err := os.Symlink(outside, filepath.Join(root, ".aether")); err != nil {
			t.Fatal(err)
		}
		err := withPlanningMutationSession(root, "test-linked-data", func(*planningMutationSession) error {
			return errors.New("callback must not run")
		})
		if err == nil || !strings.Contains(err.Error(), "containment refused") {
			t.Fatalf("linked data root error = %v", err)
		}
		if after := snapshotPlanningMutationTree(t, outside); !equalPlanningMutationSnapshot(before, after) {
			t.Fatalf("linked root changed outside tree\nbefore: %#v\n after: %#v", before, after)
		}
	})
}

func TestPlanningTimelineConcurrentProcesses200(t *testing.T) {
	if os.Getenv(planningMutationHelperEnvironment) != "" {
		runPlanningMutationSessionHelper(t)
		return
	}

	root := t.TempDir()
	outside := t.TempDir()
	mustWritePlanningMutationFile(t, filepath.Join(outside, "sentinel.txt"), []byte("outside-timeline"))
	outsideBefore := snapshotPlanningMutationTree(t, outside)
	firstInput := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 8, 18, 0, 0, 0, time.UTC))
	firstInput.RunID = "planning-session-process-race"
	firstCanonical, _, err := canonicalPlanningTimelineCard(firstInput)
	if err != nil {
		t.Fatal(err)
	}
	secondInput := validPlanningIterationCardForTest(t, 2, time.Date(2026, time.September, 8, 18, 1, 0, 0, time.UTC))
	secondInput.RunID = firstInput.RunID
	firstJSON, _ := json.Marshal(firstInput)
	secondJSON, _ := json.Marshal(secondInput)

	first := startPlanningMutationChild(t, "timeline-append", root, "route-process-one", string(firstJSON), true)
	second := startPlanningMutationChild(t, "timeline-append", root, "route-process-two", string(secondJSON), false)
	awaitPlanningMutationPipe(t, first.ready, "first timeline child ready")
	awaitPlanningMutationPipe(t, second.ready, "second timeline child ready")
	releasePlanningMutationPipe(t, first.release)
	awaitPlanningMutationPipe(t, first.holdReady, "first timeline child holds session")
	releasePlanningMutationPipe(t, second.release)
	releasePlanningMutationPipe(t, first.holdRelease)

	firstResult := awaitPlanningMutationChild(t, first)
	secondResult := awaitPlanningMutationChild(t, second)
	if firstResult.Error != "" || secondResult.Error != "" {
		t.Fatalf("timeline children failed: first=%q second=%q", firstResult.Error, secondResult.Error)
	}
	loaded, err := loadPlanningTimeline(root, firstInput.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Cards) != 2 || loaded.Index == nil {
		t.Fatalf("timeline = %#v, want two serialized cards", loaded)
	}
	if firstResult.Receipt == nil || secondResult.Receipt == nil {
		t.Fatalf("timeline children returned no receipts: first=%#v second=%#v", firstResult, secondResult)
	}
	if loaded.Cards[0].ID != firstResult.Receipt.CardID || loaded.Cards[1].ID != secondResult.Receipt.CardID {
		t.Fatalf("timeline order = %q then %q, receipts = %q then %q", loaded.Cards[0].ID, loaded.Cards[1].ID, firstResult.Receipt.CardID, secondResult.Receipt.CardID)
	}
	if loaded.Index.Entries[0].PreviousCardHash != "" || loaded.Index.Entries[1].PreviousCardHash != firstCanonical.ContentHash {
		t.Fatalf("timeline predecessor chain = %#v", loaded.Index.Entries)
	}
	if loaded.Index.LastCardHash != secondResult.Receipt.CardHash || loaded.Index.TimelineDigest != secondResult.Receipt.TimelineDigest {
		t.Fatalf("timeline terminal hashes = %#v, second receipt = %#v", loaded.Index, secondResult.Receipt)
	}
	if after := snapshotPlanningMutationTree(t, outside); !equalPlanningMutationSnapshot(outsideBefore, after) {
		t.Fatalf("timeline race changed outside tree\nbefore: %#v\n after: %#v", outsideBefore, after)
	}
}

func runPlanningMutationSessionHelper(t *testing.T) {
	t.Helper()
	ready := os.NewFile(3, "planning-mutation-ready")
	release := os.NewFile(4, "planning-mutation-release")
	result := os.NewFile(5, "planning-mutation-result")
	holdReady := os.NewFile(6, "planning-mutation-hold-ready")
	holdRelease := os.NewFile(7, "planning-mutation-hold-release")
	defer ready.Close()
	defer release.Close()
	defer result.Close()
	defer holdReady.Close()
	defer holdRelease.Close()
	if _, err := ready.WriteString("ready\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(release, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}

	mode := os.Getenv("AETHER_PLANNING_MUTATION_MODE")
	root := os.Getenv("AETHER_PLANNING_MUTATION_ROOT")
	token := os.Getenv("AETHER_PLANNING_MUTATION_TOKEN")
	hold := os.Getenv("AETHER_PLANNING_MUTATION_HOLD") == "1"
	response := planningMutationProcessResult{}
	switch mode {
	case "session-append":
		response.Error = planningMutationSessionAppendHelper(root, token)
		if response.Error == "" {
			content, err := os.ReadFile(filepath.Join(root, ".aether", "data", "planning", "session-observed-"+token+".txt"))
			if err != nil {
				response.Error = err.Error()
			} else {
				response.Observed = string(content)
			}
		}
	case "session-crash":
		response.Error = planningMutationSessionCrashHelper(root, token, holdReady, holdRelease)
	case "timeline-append":
		var card colony.PlanningIterationCard
		if err := json.Unmarshal([]byte(os.Getenv("AETHER_PLANNING_MUTATION_CARD")), &card); err != nil {
			response.Error = err.Error()
			break
		}
		opts := planningTimelineAppendOptions{ReceiptID: token}
		if card.Iteration > 1 {
			var predecessor colony.PlanningIterationCard
			if err := json.Unmarshal([]byte(os.Getenv("AETHER_PLANNING_MUTATION_PREDECESSOR")), &predecessor); err != nil {
				response.Error = err.Error()
				break
			}
			canonical, _, err := canonicalPlanningTimelineCard(predecessor)
			if err != nil {
				response.Error = err.Error()
				break
			}
			opts.PreviousCardHash = canonical.ContentHash
		}
		if hold {
			opts.Fault = func(point string) error {
				if point != "after_validation" {
					return nil
				}
				if _, err := holdReady.WriteString("held\n"); err != nil {
					return err
				}
				_, err := io.ReadFull(holdRelease, make([]byte, 1))
				return err
			}
		}
		receipt, err := appendPlanningIterationCard(root, card, opts)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Receipt = &receipt
		}
	default:
		response.Error = "unknown helper mode " + mode
	}
	if err := json.NewEncoder(result).Encode(response); err != nil {
		t.Fatal(err)
	}
}

func planningMutationSessionAppendHelper(root, token string) string {
	return planningMutationErrorString(withPlanningMutationSession(root, "process-append-"+token, func(session *planningMutationSession) error {
		content, exists, err := session.ReadFile(lifecycleTransactionRootData, "planning/session-state.txt")
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("session state is absent")
		}
		observed := bytes.Clone(content)
		if err := os.WriteFile(filepath.Join(root, ".aether", "data", "planning", "session-observed-"+token+".txt"), observed, 0o644); err != nil {
			return err
		}
		tx, err := beginLifecycleTransaction(planningMutationTestConfig(root, "session-process-"+token, session))
		if err != nil {
			return err
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootData, "planning/session-state.txt", append(content, token...)); err != nil {
			return err
		}
		_, err = tx.Commit()
		return err
	}))
}

func planningMutationSessionCrashHelper(root, token string, holdReady, holdRelease *os.File) string {
	return planningMutationErrorString(withPlanningMutationSession(root, "process-crash-"+token, func(session *planningMutationSession) error {
		if _, exists, err := session.ReadFile(lifecycleTransactionRootData, "planning/killed.txt"); err != nil || !exists {
			return fmt.Errorf("read killed baseline: exists=%v err=%w", exists, err)
		}
		config := planningMutationTestConfig(root, "session-crash-"+token, session)
		config.Fault = func(point string) error {
			if point != "after_target_commit:target-0001" {
				return nil
			}
			if _, err := holdReady.WriteString("held\n"); err != nil {
				return err
			}
			_, err := io.ReadFull(holdRelease, make([]byte, 1))
			return err
		}
		tx, err := beginLifecycleTransaction(config)
		if err != nil {
			return err
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootData, "planning/killed.txt", []byte("after-kill")); err != nil {
			return err
		}
		_, err = tx.Commit()
		return err
	}))
}

func planningMutationTestConfig(root, id string, session *planningMutationSession) lifecycleTransactionConfig {
	return lifecycleTransactionConfig{
		TransactionID: id,
		Command:       "planning mutation session test",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    root,
			LifecycleDataRoot: filepath.Join(root, ".aether", "data"),
		},
		Session: session,
	}
}

func startPlanningMutationChild(t *testing.T, mode, root, token, card string, hold bool) *planningMutationChildProcess {
	t.Helper()
	readyRead, readyWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	releaseRead, releaseWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	resultRead, resultWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	holdReadyRead, holdReadyWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	holdReleaseRead, holdReleaseWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	testName := "^TestPlanningMutationSession200$"
	if mode == "timeline-append" {
		testName = "^TestPlanningTimelineConcurrentProcesses200$"
	}
	command := exec.CommandContext(ctx, os.Args[0], "-test.run="+testName, "-test.count=1")
	command.Env = append(os.Environ(),
		planningMutationHelperEnvironment+"=1",
		"AETHER_PLANNING_MUTATION_MODE="+mode,
		"AETHER_PLANNING_MUTATION_ROOT="+root,
		"AETHER_PLANNING_MUTATION_TOKEN="+token,
		"AETHER_PLANNING_MUTATION_CARD="+card,
		"AETHER_PLANNING_MUTATION_HOLD="+map[bool]string{false: "0", true: "1"}[hold],
	)
	if mode == "timeline-append" && strings.Contains(token, "two") {
		first := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 8, 18, 0, 0, 0, time.UTC))
		first.RunID = "planning-session-process-race"
		predecessor, _ := json.Marshal(first)
		command.Env = append(command.Env, "AETHER_PLANNING_MUTATION_PREDECESSOR="+string(predecessor))
	}
	command.ExtraFiles = []*os.File{readyWrite, releaseRead, resultWrite, holdReadyWrite, holdReleaseRead}
	output := &bytes.Buffer{}
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		cancel()
		t.Fatalf("start planning mutation helper: %v", err)
	}
	_ = readyWrite.Close()
	_ = releaseRead.Close()
	_ = resultWrite.Close()
	_ = holdReadyWrite.Close()
	_ = holdReleaseRead.Close()
	return &planningMutationChildProcess{
		command: command, output: output, ready: readyRead, release: releaseWrite, result: resultRead,
		holdReady: holdReadyRead, holdRelease: holdReleaseWrite, cancel: cancel,
	}
}

func awaitPlanningMutationChild(t *testing.T, child *planningMutationChildProcess) planningMutationProcessResult {
	t.Helper()
	_ = child.release.Close()
	_ = child.holdRelease.Close()
	content, readErr := io.ReadAll(child.result)
	_ = child.result.Close()
	waitErr := child.command.Wait()
	child.cancel()
	if readErr != nil {
		t.Fatalf("read planning mutation helper result: %v", readErr)
	}
	if waitErr != nil {
		t.Fatalf("planning mutation helper failed: %v; result=%s; output=%s", waitErr, content, child.output.String())
	}
	var result planningMutationProcessResult
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("decode planning mutation helper result %q: %v", content, err)
	}
	return result
}

func awaitPlanningMutationPipe(t *testing.T, pipe *os.File, description string) {
	t.Helper()
	content := make([]byte, 1)
	if _, err := io.ReadFull(pipe, content); err != nil {
		t.Fatalf("await %s: %v", description, err)
	}
	_ = pipe.Close()
}

func releasePlanningMutationPipe(t *testing.T, pipe *os.File) {
	t.Helper()
	if _, err := pipe.Write([]byte{1}); err != nil {
		t.Fatalf("release planning mutation barrier: %v", err)
	}
	_ = pipe.Close()
}

func planningMutationErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func mustWritePlanningMutationFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustReadPlanningMutationFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

type planningMutationSnapshotEntry struct {
	Path    string
	Mode    os.FileMode
	Content string
}

func snapshotPlanningMutationTree(t *testing.T, root string) []planningMutationSnapshotEntry {
	t.Helper()
	var snapshot []planningMutationSnapshotEntry
	err := filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		item := planningMutationSnapshotEntry{Path: filepath.ToSlash(relative), Mode: info.Mode()}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(current)
			if err != nil {
				return err
			}
			item.Content = target
		case info.Mode().IsRegular():
			content, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			item.Content = string(content)
		}
		snapshot = append(snapshot, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].Path < snapshot[j].Path })
	return snapshot
}

func equalPlanningMutationSnapshot(left, right []planningMutationSnapshotEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
