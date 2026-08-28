package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// openCodeStorageMaxFileBytes bounds every individual file this parser
// reads under OpenCode's local storage tree (T-174-02). Mirrors the
// precedent set by internalWorkerRequestMaxBytes = 2 << 20
// (cmd/internal_worker_adapter.go:22), which already bounds an untrusted
// input file's size the same way.
const openCodeStorageMaxFileBytes = 2 << 20

// openCodeStorageMaxSessionFiles bounds how many session or message files a
// single directory scan will read, so a pathological or corrupted store
// cannot turn discovery into an unbounded directory walk (T-174-02).
const openCodeStorageMaxSessionFiles = 2000

// openCodeSessionUsage is one resolved worker's usage, read from OpenCode's
// own local session store rather than relayed by an orchestrating LLM
// (D-06 -- a measurement the Go runtime read itself, not an assertion).
type openCodeSessionUsage struct {
	WorkerName      string
	SessionID       string
	ParentSessionID string
	Usage           codex.WorkerUsage
}

// openCodeStorageRoot returns OpenCode's canonical local storage root,
// $HOME/.local/share/opencode/storage. It is the expected root every
// candidate root argument to openCodeSessionUsageForRun is validated
// against (T-174-01).
func openCodeStorageRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "opencode", "storage"), nil
}

// validateOpenCodeStoragePath is the T-174-01 boundary check for this
// platform's different root. Mirrors validateSpendTranscriptPath
// (cmd/spend_session_capture.go) structurally: reject null bytes, require
// an absolute path, filepath.Clean, filepath.EvalSymlinks both root and
// candidate with a fallback to the un-evaluated absolute path on failure,
// and decide containment with filepath.Rel plus a leading ".." rejection
// -- never a bare lexical prefix match on the raw strings, which a crafted
// sibling directory name could defeat (e.g. "$HOME/.local/share/opencode/storage-evil").
func validateOpenCodeStoragePath(root, candidate string) error {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return fmt.Errorf("opencode storage root is empty")
	}
	if strings.ContainsRune(candidate, 0) {
		return fmt.Errorf("opencode storage root contains a null byte")
	}
	if !filepath.IsAbs(candidate) {
		return fmt.Errorf("opencode storage root %q must be absolute", candidate)
	}
	cleanCandidate := filepath.Clean(candidate)

	rootEval := evalSymlinksOrSelf(root)
	candidateEval := evalSymlinksOrSelf(cleanCandidate)

	if candidateEval == rootEval {
		return nil
	}
	rel, err := filepath.Rel(rootEval, candidateEval)
	if err != nil {
		return fmt.Errorf("opencode storage root %q does not resolve under %q", candidate, root)
	}
	if strings.SplitN(rel, string(filepath.Separator), 2)[0] == ".." {
		return fmt.Errorf("opencode storage root %q escapes %q", candidate, root)
	}
	return nil
}

// evalSymlinksOrSelf resolves symlinks in p, falling back to the
// unresolved, cleaned path when resolution fails (e.g. the path does not
// exist yet) -- the same fallback validateSpendTranscriptPath uses for its
// own root.
func evalSymlinksOrSelf(p string) string {
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return resolved
}

// readBoundedFile reads path, refusing anything larger than maxBytes
// without ever opening the file into memory unbounded (T-174-02 precedent:
// internalWorkerRequestMaxBytes).
func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, maxBytes)
	}
	return os.ReadFile(path)
}

// openCodeProjectRaw is the subset of an OpenCode project record this
// parser reads. The `worktree` field is the only structural signal that
// binds a session store to a specific repository (T-174-20).
type openCodeProjectRaw struct {
	ID       string `json:"id"`
	Worktree string `json:"worktree"`
}

// openCodeSessionRaw is the subset of an OpenCode session record this
// parser reads. Created/updated are decoded as json.Number so both an
// integer and a float64-shaped JSON literal parse without loss.
type openCodeSessionRaw struct {
	ID       string `json:"id"`
	ParentID string `json:"parentID"`
	Title    string `json:"title"`
	Time     struct {
		Created json.Number `json:"created"`
		Updated json.Number `json:"updated"`
	} `json:"time"`
}

type openCodeSessionRecord struct {
	ID        string
	ParentID  string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// openCodeMessageRaw is the subset of an OpenCode message record this
// parser reads. Deliberately has no field for `tokens.reasoning`:
// WorkerUsage has no column for it, and folding it into OutputTokens would
// invent arithmetic the provider did not report. This is a known, recorded
// limitation (174-RESEARCH.md Pattern 2), not an oversight.
type openCodeMessageRaw struct {
	Role   string `json:"role"`
	Tokens struct {
		Total json.Number `json:"total"`
		Input json.Number `json:"input"`
		// Output intentionally excludes reasoning -- see the struct
		// doc comment above.
		Output json.Number `json:"output"`
		Cache  struct {
			Read  json.Number `json:"read"`
			Write json.Number `json:"write"`
		} `json:"cache"`
	} `json:"tokens"`
}

// openCodeSessionUsageForRun discovers OpenCode workers for the current
// build/continue run and returns their usage, read directly from OpenCode's
// own local session store. root is validated to lie under
// openCodeStorageRoot() before anything is opened (T-174-01). repoRoot
// bounds discovery to sessions whose project record's worktree matches this
// repository (T-174-20). [startedAt, endedAt] bounds discovery to this
// run's own time window (T-174-19, Pitfall 4) -- when endedAt is the zero
// time the window is open-ended forward from startedAt.
//
// The second return value is a slice of plain-English diagnostics for
// anything discovery declined to resolve; it is never an error type because
// a partial, honestly-reported result is the correct outcome here, not a
// failure.
func openCodeSessionUsageForRun(root, repoRoot string, startedAt, endedAt time.Time, workerNames []string) ([]openCodeSessionUsage, []string) {
	canonicalRoot, err := openCodeStorageRoot()
	if err != nil {
		return nil, []string{fmt.Sprintf("resolve opencode storage root: %v", err)}
	}
	if err := validateOpenCodeStoragePath(canonicalRoot, root); err != nil {
		return nil, []string{fmt.Sprintf("opencode storage root rejected: %v", err)}
	}
	root = filepath.Clean(root)

	repoRootEval := evalSymlinksOrSelf(filepath.Clean(repoRoot))

	projectIDs := matchingOpenCodeProjects(root, repoRootEval)
	if len(projectIDs) == 0 {
		return nil, []string{fmt.Sprintf("no opencode project matches worktree %q", repoRoot)}
	}

	nameMatches := make(map[string][]openCodeSessionRecord)
	for _, projectID := range projectIDs {
		for _, sess := range readOpenCodeSessions(root, projectID) {
			if !openCodeSessionInWindow(sess, startedAt, endedAt) {
				continue
			}
			for _, name := range workerNames {
				if openCodeTitleMatchesWorker(sess.Title, name) {
					nameMatches[name] = append(nameMatches[name], sess)
				}
			}
		}
	}

	var results []openCodeSessionUsage
	var reasons []string
	for _, name := range workerNames {
		matches := nameMatches[name]
		switch len(matches) {
		case 0:
			continue
		case 1:
			sess := matches[0]
			results = append(results, openCodeSessionUsage{
				WorkerName:      name,
				SessionID:       sess.ID,
				ParentSessionID: sess.ParentID,
				Usage:           readOpenCodeUsage(root, sess.ID),
			})
		default:
			// Pitfall 4: when more than one in-window session matches the
			// same worker name, resolve NOTHING for that worker. An
			// ambiguous match must fall through to plan 174-06's estimate
			// rather than pick the most recent -- a plausible wrong number
			// is worse than an honest estimate.
			reasons = append(reasons, fmt.Sprintf("worker %q matched %d in-window opencode sessions; refusing to guess", name, len(matches)))
		}
	}

	return results, reasons
}

// openCodeSessionUsageForRunOrNone is the convenience entry point plan
// 174-06 calls. It resolves the storage root itself and reports false when
// it is absent or unreadable -- a missing OpenCode store on a Claude-Code-
// only machine is the normal case, not an error, so failing soft here is
// deliberate.
func openCodeSessionUsageForRunOrNone(repoRoot string, startedAt, endedAt time.Time, workerNames []string) ([]openCodeSessionUsage, bool) {
	root, err := openCodeStorageRoot()
	if err != nil {
		return nil, false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, false
	}
	entries, _ := openCodeSessionUsageForRun(root, repoRoot, startedAt, endedAt, workerNames)
	return entries, true
}

// matchingOpenCodeProjects reads every project record under
// <root>/project/, bounded by openCodeStorageMaxFileBytes per file, and
// returns the ids whose worktree (symlink-evaluated) equals repoRootEval.
// Decode failures are skipped silently -- an untrusted local store must
// never crash discovery.
func matchingOpenCodeProjects(root, repoRootEval string) []string {
	dir := filepath.Join(root, "project")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := readBoundedFile(filepath.Join(dir, entry.Name()), openCodeStorageMaxFileBytes)
		if err != nil {
			continue
		}
		var proj openCodeProjectRaw
		if err := json.Unmarshal(data, &proj); err != nil {
			continue
		}
		if proj.ID == "" || strings.TrimSpace(proj.Worktree) == "" {
			continue
		}
		if evalSymlinksOrSelf(filepath.Clean(proj.Worktree)) == repoRootEval {
			ids = append(ids, proj.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

// readOpenCodeSessions reads every session record under
// <root>/session/<projectID>/, bounded by openCodeStorageMaxSessionFiles
// entries and openCodeStorageMaxFileBytes per file. A session directory
// containing a malformed JSON file is skipped for that file only -- the
// remaining sessions still parse.
func readOpenCodeSessions(root, projectID string) []openCodeSessionRecord {
	dir := filepath.Join(root, "session", projectID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var sessions []openCodeSessionRecord
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if count >= openCodeStorageMaxSessionFiles {
			break
		}
		count++
		data, err := readBoundedFile(filepath.Join(dir, entry.Name()), openCodeStorageMaxFileBytes)
		if err != nil {
			continue
		}
		var raw openCodeSessionRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		if raw.ID == "" {
			continue
		}
		sessions = append(sessions, openCodeSessionRecord{
			ID:        raw.ID,
			ParentID:  raw.ParentID,
			Title:     raw.Title,
			CreatedAt: epochMillisToTime(raw.Time.Created),
			UpdatedAt: epochMillisToTime(raw.Time.Updated),
		})
	}
	return sessions
}

// epochMillisToTime converts an OpenCode epoch-millisecond field to a
// time.Time, returning the zero time for an absent or unparseable value.
func epochMillisToTime(n json.Number) time.Time {
	if strings.TrimSpace(string(n)) == "" {
		return time.Time{}
	}
	f, err := n.Float64()
	if err != nil {
		return time.Time{}
	}
	return time.UnixMilli(int64(f))
}

// openCodeSessionInWindow keeps a session only when its created or updated
// timestamp falls within the inclusive [startedAt, endedAt] window --
// mirroring filterEntriesForRun's bounding shape (pkg/agent/spawn_tree.go).
// When endedAt is the zero time the window is open-ended forward from
// startedAt.
func openCodeSessionInWindow(sess openCodeSessionRecord, startedAt, endedAt time.Time) bool {
	return timeInWindow(sess.CreatedAt, startedAt, endedAt) || timeInWindow(sess.UpdatedAt, startedAt, endedAt)
}

func timeInWindow(ts, startedAt, endedAt time.Time) bool {
	if ts.IsZero() || startedAt.IsZero() {
		return false
	}
	if ts.Before(startedAt) {
		return false
	}
	if !endedAt.IsZero() && ts.After(endedAt) {
		return false
	}
	return true
}

var openCodeWorkerNamePatternCache = map[string]*regexp.Regexp{}

// openCodeTitleMatchesWorker reports whether title contains name as an
// exact match bounded by non-word characters, so "Mason-6" never matches
// inside "Mason-67".
func openCodeTitleMatchesWorker(title, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	pattern, ok := openCodeWorkerNamePatternCache[name]
	if !ok {
		pattern = regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
		openCodeWorkerNamePatternCache[name] = pattern
	}
	return pattern.MatchString(title)
}

// readOpenCodeUsage reads every message record under
// <root>/message/<sessionID>/, bounded by openCodeStorageMaxSessionFiles
// entries and openCodeStorageMaxFileBytes per file, and accumulates the
// assistant-role records' disjoint token columns into a codex.WorkerUsage.
// Source is always UsageSourceSessionTranscript -- never UsageSourceProvider,
// which only ParseUsage may set.
func readOpenCodeUsage(root, sessionID string) codex.WorkerUsage {
	usage := codex.WorkerUsage{Source: codex.UsageSourceSessionTranscript}

	dir := filepath.Join(root, "message", sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return usage
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if count >= openCodeStorageMaxSessionFiles {
			break
		}
		count++
		data, err := readBoundedFile(filepath.Join(dir, entry.Name()), openCodeStorageMaxFileBytes)
		if err != nil {
			continue
		}
		var msg openCodeMessageRaw
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Role != "assistant" {
			continue
		}
		usage.InputTokens += jsonNumberOrZero(msg.Tokens.Input)
		usage.OutputTokens += jsonNumberOrZero(msg.Tokens.Output)
		usage.CachedInputTokens += jsonNumberOrZero(msg.Tokens.Cache.Read)
		usage.CacheCreationTokens += jsonNumberOrZero(msg.Tokens.Cache.Write)
		usage.TotalTokens += jsonNumberOrZero(msg.Tokens.Total)
	}
	return usage
}

func jsonNumberOrZero(n json.Number) int64 {
	if strings.TrimSpace(string(n)) == "" {
		return 0
	}
	f, err := n.Float64()
	if err != nil {
		return 0
	}
	return int64(f)
}
