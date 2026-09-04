package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// Registry types.

const registryMutationSchemaVersion = "maintenance-registry/v1"

type registryMutationAction string

const (
	registryMutationUpsert   registryMutationAction = "upsert"
	registryMutationRemove   registryMutationAction = "remove"
	registryMutationFinalize registryMutationAction = "finalize"
)

// registryFinalStats holds colony lifecycle statistics recorded at entomb time.
type registryFinalStats struct {
	PhaseCount    int    `json:"phase_count,omitempty"`
	PlanCount     int    `json:"plan_count,omitempty"`
	LearningCount int    `json:"learning_count,omitempty"`
	InstinctCount int    `json:"instinct_count,omitempty"`
	SealDate      string `json:"seal_date,omitempty"`
	Duration      string `json:"duration,omitempty"`
}

type registryEntry struct {
	RepoID       string              `json:"repo_id,omitempty"`
	RepoPath     string              `json:"repo_path"`
	Domains      []string            `json:"domains"`
	Active       bool                `json:"active"`
	RegisteredAt string              `json:"registered_at"`
	LastGoal     string              `json:"last_goal,omitempty"`
	FinalStats   *registryFinalStats `json:"final_stats,omitempty"`
}

type registryData struct {
	Colonies []registryEntry `json:"colonies"`
}

type registryMutationRequest struct {
	SchemaVersion     string
	Operation         registryMutationAction
	TransactionID     string
	RepositoryRoot    string
	LifecycleDataRoot string
	HubRoot           string
	Channel           runtimeChannel
	RepoPath          string
	RepoIdentity      string
	ExpectedBaseline  string
	Domains           []string
	Goal              string
	Active            bool
	FinalStats        *registryFinalStats
	RegisteredAt      time.Time
	Rename            func(oldPath, newPath string) error
	Fault             lifecycleTransactionFaultHook
}

type registryMutationPreview struct {
	SchemaVersion      string                     `json:"schema_version"`
	Operation          registryMutationAction     `json:"operation"`
	RepositoryPath     string                     `json:"repository_path"`
	RepositoryIdentity string                     `json:"repository_identity"`
	BaselineDigest     string                     `json:"baseline_digest"`
	DesiredDigest      string                     `json:"desired_digest"`
	Updated            bool                       `json:"updated"`
	Total              int                        `json:"total"`
	Mutation           maintenanceMutationPreview `json:"mutation"`
}

type registryMutationPlan struct {
	Request  registryMutationRequest
	Preview  registryMutationPreview
	mutation maintenanceMutationPlan
}

type registryMutationResult struct {
	SchemaVersion string                         `json:"schema_version"`
	Operation     registryMutationAction         `json:"operation"`
	Preview       registryMutationPreview        `json:"preview"`
	StateEffect   colony.LifecycleStateEffect    `json:"state_effect"`
	Receipt       *colony.LifecycleReceipt       `json:"receipt,omitempty"`
	Verification  []colony.LifecycleVerification `json:"verification,omitempty"`
	Recovery      string                         `json:"recovery"`
}

func prepareRegistryMutation(request registryMutationRequest) (registryMutationPlan, error) {
	plan := registryMutationPlan{}
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	request.TransactionID = strings.TrimSpace(request.TransactionID)
	request.RepositoryRoot = filepath.Clean(request.RepositoryRoot)
	request.LifecycleDataRoot = filepath.Clean(request.LifecycleDataRoot)
	request.HubRoot = filepath.Clean(request.HubRoot)
	request.RepoPath = filepath.Clean(strings.TrimSpace(request.RepoPath))
	request.RepoIdentity = strings.TrimSpace(request.RepoIdentity)
	request.ExpectedBaseline = strings.TrimSpace(request.ExpectedBaseline)
	request.Domains = normalizeRegistryDomains(request.Domains)
	request.Goal = strings.TrimSpace(request.Goal)
	if request.Channel == "" {
		request.Channel = channelStable
	}
	plan.Request = request

	preview := registryMutationPreview{
		SchemaVersion:      registryMutationSchemaVersion,
		Operation:          request.Operation,
		RepositoryPath:     request.RepoPath,
		RepositoryIdentity: request.RepoIdentity,
		BaselineDigest:     request.ExpectedBaseline,
	}
	plan.Preview = preview
	if request.SchemaVersion != registryMutationSchemaVersion {
		return plan, fmt.Errorf("registry mutation: schema_version must be %s", registryMutationSchemaVersion)
	}
	switch request.Operation {
	case registryMutationUpsert, registryMutationRemove, registryMutationFinalize:
	default:
		return plan, fmt.Errorf("registry mutation: unsupported operation %q", request.Operation)
	}
	if request.TransactionID == "" {
		return plan, fmt.Errorf("registry mutation: transaction id is required")
	}
	if !filepath.IsAbs(request.RepoPath) || request.RepoPath == "." {
		return plan, fmt.Errorf("registry mutation: repository path must be a canonical absolute path")
	}
	wantedIdentity := stableRepoIdentity(request.RepoPath)
	if wantedIdentity == "" || request.RepoIdentity != wantedIdentity {
		return plan, fmt.Errorf("registry mutation: repository identity does not match canonical repository path")
	}
	if request.ExpectedBaseline == "" {
		return plan, fmt.Errorf("registry mutation: expected baseline digest is required")
	}
	if request.Active && request.FinalStats != nil {
		return plan, fmt.Errorf("registry mutation: an active colony cannot carry final stats")
	}

	registryPath := filepath.Join(request.HubRoot, "registry", "registry.json")
	current, err := readLifecycleFileState(registryPath)
	if err != nil {
		return plan, fmt.Errorf("registry mutation: read registry baseline: %w", err)
	}
	if current.Digest != request.ExpectedBaseline {
		return plan, fmt.Errorf("registry mutation: baseline changed (expected %s, found %s)", request.ExpectedBaseline, current.Digest)
	}
	registry := registryData{Colonies: []registryEntry{}}
	if current.Exists {
		if err := decodeLifecycleJSON(current.Bytes, &registry); err != nil {
			return plan, fmt.Errorf("registry mutation: decode current registry: %w", err)
		}
	}
	if err := validateAndNormalizeRegistry(&registry); err != nil {
		return plan, err
	}

	match := -1
	for index, entry := range registry.Colonies {
		if entry.RepoID == request.RepoIdentity || entry.RepoPath == request.RepoPath {
			if match >= 0 {
				return plan, fmt.Errorf("registry mutation: duplicate repository conflict for %s", request.RepoIdentity)
			}
			match = index
		}
	}
	updated := match >= 0
	switch request.Operation {
	case registryMutationUpsert:
		if match < 0 {
			if request.RegisteredAt.IsZero() {
				return plan, fmt.Errorf("registry mutation: registered_at is required for a new entry")
			}
			registry.Colonies = append(registry.Colonies, registryEntry{
				RepoID:       request.RepoIdentity,
				RepoPath:     request.RepoPath,
				Domains:      append([]string(nil), request.Domains...),
				Active:       request.Active,
				RegisteredAt: request.RegisteredAt.UTC().Format(time.RFC3339),
				LastGoal:     request.Goal,
				FinalStats:   cloneRegistryFinalStats(request.FinalStats),
			})
		} else {
			entry := &registry.Colonies[match]
			entry.RepoID = request.RepoIdentity
			entry.RepoPath = request.RepoPath
			if len(request.Domains) > 0 {
				entry.Domains = append([]string(nil), request.Domains...)
			}
			if request.Goal != "" {
				entry.LastGoal = request.Goal
			}
			entry.Active = request.Active
			if request.Active {
				entry.FinalStats = nil
			} else if request.FinalStats != nil {
				entry.FinalStats = cloneRegistryFinalStats(request.FinalStats)
			}
		}
	case registryMutationFinalize:
		if match < 0 {
			return plan, fmt.Errorf("registry mutation: cannot finalize an unregistered repository")
		}
		if request.FinalStats == nil {
			return plan, fmt.Errorf("registry mutation: final stats are required for finalize")
		}
		registry.Colonies[match].Active = false
		registry.Colonies[match].FinalStats = cloneRegistryFinalStats(request.FinalStats)
	case registryMutationRemove:
		if match >= 0 {
			registry.Colonies = append(registry.Colonies[:match], registry.Colonies[match+1:]...)
		}
	}
	if err := validateAndNormalizeRegistry(&registry); err != nil {
		return plan, err
	}
	desired, err := encodeRegistry(registry)
	if err != nil {
		return plan, err
	}
	targetAction := lifecycleTransactionWrite
	if request.Operation == registryMutationRemove && !updated {
		if current.Exists {
			desired = append([]byte(nil), current.Bytes...)
		} else {
			targetAction = lifecycleTransactionRemove
			desired = nil
		}
	}
	preview.DesiredDigest = lifecycleTransactionMissingDigest
	if targetAction == lifecycleTransactionWrite {
		preview.DesiredDigest = lifecycleDigest(desired)
	}
	preview.Updated = updated
	preview.Total = len(registry.Colonies)

	hubChannel := lifecycleTransactionHubStable
	rootKind := lifecycleTransactionRootHubStable
	if request.Channel == channelDev {
		hubChannel = lifecycleTransactionHubDev
		rootKind = lifecycleTransactionRootHubDev
	} else if request.Channel != channelStable {
		return plan, fmt.Errorf("registry mutation: unsupported channel %q", request.Channel)
	}
	mutation := maintenanceMutationPlan{
		SchemaVersion:   maintenanceMutationSchemaVersion,
		Operation:       "registry-" + string(request.Operation),
		TransactionID:   request.TransactionID,
		SourceRoot:      request.RepositoryRoot,
		DestinationRoot: request.HubRoot,
		Channel:         request.Channel,
		Checkpoint:      "maintenance:registry:validated",
		Recovery:        "Preserve the registry transaction journal and run `aether resume`; after a confirmed rollback, rerun the registry command against the new baseline.",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    request.RepositoryRoot,
			LifecycleDataRoot: request.LifecycleDataRoot,
			Hub:               lifecycleTransactionHubRoot{Channel: hubChannel, Path: request.HubRoot},
		},
		Targets: []maintenanceMutationTarget{{
			Root:           rootKind,
			RelativeTarget: filepath.Join("registry", "registry.json"),
			Source:         request.RepoPath,
			Action:         targetAction,
			Content:        desired,
			ExpectedDigest: request.ExpectedBaseline,
			Managed:        true,
		}},
		Rename: request.Rename,
		Fault:  request.Fault,
	}
	mutationPreview, err := prepareMaintenanceMutation(mutation)
	if err != nil {
		return plan, err
	}
	preview.Mutation = mutationPreview
	plan.Preview = preview
	plan.mutation = mutation
	return plan, nil
}

func commitRegistryMutation(plan registryMutationPlan) (registryMutationResult, error) {
	result := registryMutationResult{
		SchemaVersion: registryMutationSchemaVersion,
		Operation:     plan.Request.Operation,
		Preview:       plan.Preview,
		StateEffect:   colony.LifecycleStateEffectNone,
		Recovery:      plan.mutation.Recovery,
	}
	prepared, err := prepareRegistryMutation(plan.Request)
	if err != nil {
		return result, err
	}
	mutation, err := commitMaintenanceMutation(prepared.mutation)
	result.Preview = prepared.Preview
	result.StateEffect = mutation.StateEffect
	result.Receipt = mutation.Receipt
	result.Verification = append([]colony.LifecycleVerification(nil), mutation.Verification...)
	result.Recovery = mutation.Recovery
	return result, err
}

func validateAndNormalizeRegistry(registry *registryData) error {
	if registry == nil {
		return fmt.Errorf("registry mutation: registry is required")
	}
	if registry.Colonies == nil {
		registry.Colonies = []registryEntry{}
	}
	identities := make(map[string]struct{}, len(registry.Colonies))
	paths := make(map[string]struct{}, len(registry.Colonies))
	for index := range registry.Colonies {
		entry := &registry.Colonies[index]
		entry.RepoPath = filepath.Clean(strings.TrimSpace(entry.RepoPath))
		if entry.RepoPath == "." || !filepath.IsAbs(entry.RepoPath) {
			return fmt.Errorf("registry mutation: entry %d has a non-canonical repository path", index)
		}
		identity := stableRepoIdentity(entry.RepoPath)
		if entry.RepoID == "" {
			entry.RepoID = identity
		} else if entry.RepoID != identity {
			return fmt.Errorf("registry mutation: entry %d repository identity conflicts with its path", index)
		}
		if entry.RepoID == "" {
			return fmt.Errorf("registry mutation: entry %d has no stable repository identity", index)
		}
		if _, duplicate := identities[entry.RepoID]; duplicate {
			return fmt.Errorf("registry mutation: duplicate repository identity %s", entry.RepoID)
		}
		if _, duplicate := paths[entry.RepoPath]; duplicate {
			return fmt.Errorf("registry mutation: duplicate repository path %s", entry.RepoPath)
		}
		identities[entry.RepoID] = struct{}{}
		paths[entry.RepoPath] = struct{}{}
		entry.Domains = normalizeRegistryDomains(entry.Domains)
		if entry.Active && entry.FinalStats != nil {
			return fmt.Errorf("registry mutation: active entry %s cannot carry final stats", entry.RepoPath)
		}
	}
	return nil
}

func normalizeRegistryDomains(domains []string) []string {
	seen := make(map[string]struct{}, len(domains))
	normalized := make([]string, 0, len(domains))
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if _, duplicate := seen[domain]; duplicate {
			continue
		}
		seen[domain] = struct{}{}
		normalized = append(normalized, domain)
	}
	return normalized
}

func cloneRegistryFinalStats(stats *registryFinalStats) *registryFinalStats {
	if stats == nil {
		return nil
	}
	clone := *stats
	return &clone
}

func encodeRegistry(registry registryData) ([]byte, error) {
	encoded, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal registry: %w", err)
	}
	return append(encoded, '\n'), nil
}

// --- registry-add ---

var registryAddCmd = &cobra.Command{
	Use:   "registry-add [version]",
	Short: "Register a colony repository",
	Args:  cobra.MaximumNArgs(1), // accept optional positional version arg (ignored)
	RunE:  runRegistryAdd,
}

func runRegistryAdd(cmd *cobra.Command, _ []string) error {
	repo, _ := cmd.Flags().GetString("repo")
	if path, _ := cmd.Flags().GetString("path"); path != "" {
		repo = path
	}
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return nil
	}
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return fmt.Errorf("registry-add: resolve repository path: %w", err)
	}
	absRepo = filepath.Clean(absRepo)

	domainsText, _ := cmd.Flags().GetString("domain")
	if tags, _ := cmd.Flags().GetString("tags"); tags != "" {
		domainsText = tags
	}
	domains := []string{}
	if domainsText != "" {
		domains = normalizeRegistryDomains(strings.Split(domainsText, ","))
	}
	goal, _ := cmd.Flags().GetString("goal")
	active, _ := cmd.Flags().GetBool("active")
	remove, _ := cmd.Flags().GetBool("remove")
	action := registryMutationUpsert
	if remove {
		action = registryMutationRemove
	}

	hub := filepath.Clean(resolveHubPath())
	repositoryRoot, dataRoot, err := registryCoordinatorRoots(absRepo, hub)
	if err != nil {
		return err
	}
	baseline, err := registryBaseline(filepath.Join(hub, "registry", "registry.json"))
	if err != nil {
		return err
	}
	request := registryMutationRequest{
		SchemaVersion:     registryMutationSchemaVersion,
		Operation:         action,
		TransactionID:     fmt.Sprintf("registry-%s-%d", action, time.Now().UTC().UnixNano()),
		RepositoryRoot:    repositoryRoot,
		LifecycleDataRoot: dataRoot,
		HubRoot:           hub,
		Channel:           resolveRuntimeChannel(),
		RepoPath:          absRepo,
		RepoIdentity:      stableRepoIdentity(absRepo),
		ExpectedBaseline:  baseline,
		Domains:           domains,
		Goal:              goal,
		Active:            active,
		RegisteredAt:      time.Now().UTC(),
	}
	plan, err := prepareRegistryMutation(request)
	if err != nil {
		outputError(2, err.Error(), map[string]interface{}{
			"operation": action, "repo": absRepo, "repo_id": request.RepoIdentity,
			"baseline": baseline, "state_effect": colony.LifecycleStateEffectNone,
			"recovery": "Resolve the named registry validation or baseline conflict, then rerun registry-add.",
		})
		return nil
	}
	result, err := commitRegistryMutation(plan)
	payload := map[string]interface{}{
		"registered":   action != registryMutationRemove,
		"removed":      action == registryMutationRemove && plan.Preview.Updated,
		"updated":      plan.Preview.Updated,
		"repo":         absRepo,
		"repo_id":      request.RepoIdentity,
		"total":        plan.Preview.Total,
		"operation":    result.Operation,
		"preview":      result.Preview,
		"transaction":  result.Preview.Mutation.TransactionID,
		"receipt":      result.Receipt,
		"state_effect": result.StateEffect,
		"verification": result.Verification,
		"recovery":     result.Recovery,
	}
	if err != nil {
		outputError(3, err.Error(), payload)
		return nil
	}
	outputOK(payload)
	return nil
}

// --- registry-list ---

var registryListCmd = &cobra.Command{
	Use:   "registry-list",
	Short: "List all registered colonies",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		registryPath := filepath.Join(hub, "registry", "registry.json")

		var rd registryData
		if raw, err := os.ReadFile(registryPath); err != nil {
			outputOK(map[string]interface{}{"colonies": []registryEntry{}, "total": 0})
			return nil
		} else {
			json.Unmarshal(raw, &rd)
		}

		outputOK(map[string]interface{}{"colonies": rd.Colonies, "total": len(rd.Colonies)})
		return nil
	},
}

func writeRegistry(path string, rd registryData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	encoded, err := encodeRegistry(rd)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0644)
}

// updateRegistryFinalStats updates a colony's registry entry with final lifecycle
// stats and marks it inactive. Best-effort: returns silently if no registry or no
// matching entry (colony may not have been registered).
func updateRegistryFinalStats(repoPath string, stats registryFinalStats) {
	hub := resolveHubPath()
	registryPath := filepath.Join(hub, "registry", "registry.json")
	baseline, err := registryBaseline(registryPath)
	if err != nil || baseline == lifecycleTransactionMissingDigest {
		return // no registry file -- best effort
	}
	repositoryRoot, dataRoot, err := registryCoordinatorRoots(repoPath, hub)
	if err != nil {
		return
	}
	absRepo, err := filepath.Abs(strings.TrimSpace(repoPath))
	if err != nil {
		return
	}
	request := registryMutationRequest{
		SchemaVersion: registryMutationSchemaVersion, Operation: registryMutationFinalize,
		TransactionID:  fmt.Sprintf("registry-finalize-%d", time.Now().UTC().UnixNano()),
		RepositoryRoot: repositoryRoot, LifecycleDataRoot: dataRoot, HubRoot: filepath.Clean(hub), Channel: resolveRuntimeChannel(),
		RepoPath: filepath.Clean(absRepo), RepoIdentity: stableRepoIdentity(absRepo), ExpectedBaseline: baseline,
		Active: false, FinalStats: &stats,
	}
	plan, err := prepareRegistryMutation(request)
	if err != nil {
		return
	}
	_, _ = commitRegistryMutation(plan)
}

func init() {
	registryAddCmd.Flags().String("repo", "", "Repository path")
	registryAddCmd.Flags().String("path", "", "Repository path (alias for --repo)")
	registryAddCmd.Flags().String("domain", "", "Comma-separated domain tags")
	registryAddCmd.Flags().String("tags", "", "Comma-separated domain tags (alias for --domain)")
	registryAddCmd.Flags().String("goal", "", "Colony goal")
	registryAddCmd.Flags().Bool("active", true, "Set colony as active (default: true)")
	registryAddCmd.Flags().Bool("remove", false, "Remove the exact matching repository entry")

	rootCmd.AddCommand(registryAddCmd)
	rootCmd.AddCommand(registryListCmd)
}

// resolveHubPathQuiet is resolveHubPath without the outputError side effect,
// for non-blocking callers (init/seal bookkeeping must never fail a lifecycle
// command).
func resolveHubPathQuiet() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return resolveHubPathForHome(home, resolveRuntimeChannel())
}

// upsertColonyRegistryEntry registers or updates this repo in the hub-level
// colony registry. Wired into init (active, with detected domains) and seal
// (inactive) — RECLAIM-02: the registry supplies the domain tags that scope
// hive wisdom retrieval, and until this wiring nothing populated it.
func upsertColonyRegistryEntry(repoPath, goal string, domains []string, active bool) (updated bool, err error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return false, fmt.Errorf("empty repo path")
	}
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return false, fmt.Errorf("resolve repo path: %w", err)
	}
	absRepo = filepath.Clean(absRepo)
	hub := resolveHubPathQuiet()
	if hub == "" {
		return false, fmt.Errorf("hub path unavailable")
	}
	hub = filepath.Clean(hub)
	repositoryRoot, dataRoot, err := registryCoordinatorRoots(absRepo, hub)
	if err != nil {
		return false, err
	}
	baseline, err := registryBaseline(filepath.Join(hub, "registry", "registry.json"))
	if err != nil {
		return false, err
	}
	request := registryMutationRequest{
		SchemaVersion: registryMutationSchemaVersion, Operation: registryMutationUpsert,
		TransactionID:  fmt.Sprintf("registry-upsert-%d", time.Now().UTC().UnixNano()),
		RepositoryRoot: repositoryRoot, LifecycleDataRoot: dataRoot, HubRoot: hub, Channel: resolveRuntimeChannel(),
		RepoPath: absRepo, RepoIdentity: stableRepoIdentity(absRepo), ExpectedBaseline: baseline,
		Domains: domains, Goal: goal, Active: active, RegisteredAt: time.Now().UTC(),
	}
	plan, err := prepareRegistryMutation(request)
	if err != nil {
		return false, err
	}
	if _, err := commitRegistryMutation(plan); err != nil {
		return false, err
	}
	return plan.Preview.Updated, nil
}

func registryBaseline(path string) (string, error) {
	state, err := readLifecycleFileState(path)
	if err != nil {
		return "", fmt.Errorf("registry mutation: read baseline: %w", err)
	}
	return state.Digest, nil
}

func registryCoordinatorRoots(preferredRepo, hub string) (string, string, error) {
	type candidate struct {
		repository string
		data       string
	}
	candidates := []candidate{}
	if store != nil {
		candidates = append(candidates, candidate{repository: repoRootFromStore(store), data: store.BasePath()})
	}
	preferredRepo = filepath.Clean(preferredRepo)
	candidates = append(candidates, candidate{repository: preferredRepo, data: filepath.Join(preferredRepo, ".aether", "data")})
	if cwdRoot := filepath.Clean(resolveAetherRootPath()); cwdRoot != "." {
		candidates = append(candidates, candidate{repository: cwdRoot, data: filepath.Join(cwdRoot, ".aether", "data")})
	}
	for _, candidate := range candidates {
		repository := filepath.Clean(strings.TrimSpace(candidate.repository))
		data := filepath.Clean(strings.TrimSpace(candidate.data))
		if repository == "." || data == "." || !filepath.IsAbs(repository) || !filepath.IsAbs(data) || repository == filepath.Clean(hub) || data == filepath.Clean(hub) {
			continue
		}
		repositoryInfo, repositoryErr := os.Lstat(repository)
		dataInfo, dataErr := os.Lstat(data)
		if repositoryErr == nil && dataErr == nil && repositoryInfo.IsDir() && dataInfo.IsDir() && repositoryInfo.Mode()&os.ModeSymlink == 0 && dataInfo.Mode()&os.ModeSymlink == 0 {
			return repository, data, nil
		}
	}
	return "", "", fmt.Errorf("registry mutation: no existing repository/lifecycle-data coordinator roots are available")
}
