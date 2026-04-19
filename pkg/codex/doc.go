// Package codex provides the Codex worker invocation runtime for Aether.
//
// This package bridges the gap between Aether's Go-based colony orchestration
// and OpenAI's Codex CLI (`codex` on PATH). It enables
// Aether to spawn real Codex CLI subprocesses as colony workers, achieving
// true worker-spawning parity with Claude Code's agent system.
//
// # Overview
//
// In Claude Code, Aether uses Claude's native agent spawning to dispatch
// workers with full context injection. Codex CLI lacks a direct equivalent --
// it uses TOML agent definitions in `.codex/agents/*.toml` and communicates
// via subprocess invocation. This package provides the Go-side machinery to:
//
//  1. Resolve a worker's TOML definition (name, description, developer_instructions)
//  2. Assemble a complete prompt from TOML instructions + compact colony-prime
//     context + task brief
//  3. Invoke the `codex` CLI binary as a subprocess with the assembled prompt
//  4. Capture and validate structured JSON output (claims) from the worker
//  5. Handle timeouts, fallbacks, and environment configuration
//
// # WorkerInvoker Interface
//
// The WorkerInvoker interface defines the contract for invoking a Codex worker:
//
//	type WorkerInvoker interface {
//	    // Invoke spawns a codex CLI subprocess for the given worker configuration.
//	    // It assembles the prompt, runs the process, and returns a parsed result.
//	    Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error)
//
//	    // IsAvailable checks whether the codex CLI binary is installed and reachable.
//	    IsAvailable(ctx context.Context) bool
//
//	    // ValidateAgent checks that a TOML agent file is parseable and contains
//	    // all required fields (name, description, developer_instructions).
//	    ValidateAgent(path string) error
//	}
//
// Implementations include:
//   - RealInvoker: Invokes the actual `codex` CLI binary (default outside tests)
//   - FakeInvoker: Returns deterministic results for testing and explicit fake mode
//
// # Codex CLI Invocation
//
// The codex CLI (v0.121.0+) is invoked as a subprocess using the `exec` subcommand:
//
//	codex exec --full-auto "<prompt>"
//
// Where:
//   - exec runs codex in non-interactive execution mode
//   - --full-auto approves all tool calls automatically (no human approval),
//     and silently overrides any --sandbox value, hardcoding WorkspaceWrite
//   - <prompt> is the assembled task prompt (see Prompt Assembly below)
//
// NOTE: Do not include --sandbox alongside --full-auto. The --full-auto flag
// silently overrides the sandbox level to WorkspaceWrite regardless of what
// value is passed to --sandbox. Including both is misleading.
//
// IMPORTANT: There is NO --agent flag on the codex CLI. Agent TOML files are not
// selected directly by the CLI; Aether reads developer_instructions from the TOML
// file and injects them into the prompt before invoking `codex exec`.
//
// The agent TOML developer_instructions are read by Go at dispatch time and injected
// into the prompt text so the codex worker receives the full agent role definition.
//
// # Additional Codex CLI Flags
//
// Beyond the core invocation flags above, these flags are useful for Aether's
// dispatch pipeline:
//
//   - --json / --experimental-json: Enables JSONL event output, with one JSON
//     object per line on stdout. Each line represents a session event (tool call,
//     message, result). Essential for programmatic output parsing.
//
//   - --output-schema: Accepts a path to a JSON Schema file that enforces
//     structured output from the agent. The agent's final response must conform
//     to the schema, making claims validation more reliable.
//
//   - --ephemeral: Runs the session without persisting session files to disk.
//     Prevents workspace pollution from accumulated codex session state.
//
//   - --output-last-message / -o: Writes the agent's final message content
//     to the specified file path, separate from stdout event output.
//
// Recommended RealInvoker invocation combining these flags:
//
//	codex exec --full-auto --json --ephemeral --output-last-message <file> --output-schema <file>
//
// This gives us JSONL event output for logs, a deterministic final-message file
// for claims parsing, and a schema gate on the final response.
//
// # Prompt Assembly
//
// The worker prompt is assembled from up to five parts, concatenated in order.
// Empty sections are omitted to keep the prompt concise, and the final assembled
// prompt is trimmed against a global character budget so the overall worker
// prompt stays bounded.
//
//  1. TOML developer_instructions: The full developer_instructions field from the
//     agent's TOML file (e.g., .codex/agents/aether-builder.toml). This contains
//     the agent's role definition, coding standards, output format, and boundary
//     declarations. These instructions are injected by Codex CLI as a "developer"
//     role message (per the Codex CLI format specification).
//
//  2. Colony-prime context: Generated by the Codex-facing compact colony-prime
//     path, this provides the worker with colony state, active non-expired
//     pheromone signals, instincts, key decisions, phase learnings, hive wisdom,
//     user preferences, and blockers. This section is bounded by the
//     colony-prime character budget (8000 normal, 4000 compact) before it is
//     handed to the final prompt assembler.
//
//  3. Skill section (optional): Matched skill guidance assembled by skill-inject
//     based on the worker's role, active pheromone signals, and detected codebase
//     patterns. Provides reusable behavioral patterns and domain knowledge (e.g.,
//     TDD discipline, framework idioms). Omitted when no skills match.
//
//  4. Pheromone section (optional): Reserved for additional signal injection.
//     In the current Codex workflow, pheromone guidance is already folded into
//     colony-prime, so this section is typically empty to avoid duplicate prompt
//     content.
//
//  5. Task brief: Generated by renderCodexBuildWorkerBrief, this is a markdown
//     document containing the worker's specific assignment, phase objective,
//     dependencies, constraints, hints, success criteria, and relevant playbooks.
//     Written to .aether/data/build/phase-N/worker-briefs/<worker-name>.md.
//
// # Worker Output Format (Claims JSON)
//
// Each Codex worker must return a single JSON object as its final output.
// The invoker reads that final message from --output-last-message and validates
// it against the supplied JSON schema.
//
// The claims JSON schema:
//
//	{
//	  "ant_name": "string",           // Worker's assigned name (e.g., "Hammer-23")
//	  "caste": "string",              // Worker caste (builder, watcher, scout, etc.)
//	  "task_id": "string",            // Task identifier from the dispatch
//	  "status": "string",             // One of: "completed", "code_written", "failed", "blocked"
//	  "summary": "string",            // Plain-English summary of what was accomplished
//	  "files_created": ["string"],    // Absolute paths of files the worker created
//	  "files_modified": ["string"],   // Absolute paths of files the worker modified
//	  "tests_written": ["string"],    // Paths of test files the worker created
//	  "tool_count": 0,                // Number of tool calls the worker made
//	  "blockers": ["string"],         // Descriptions of any blocking issues
//	  "spawns": ["string"]            // Names of any sub-workers spawned (if applicable)
//	}
//
// For builder workers, an additional TDD report may be included:
//
//	"tdd": {
//	  "cycles_completed": 3,
//	  "tests_added": 3,
//	  "coverage_percent": 85,
//	  "all_passing": true
//	}
//
// If the worker's final message does not contain valid JSON, the invocation is
// considered failed.
//
// # Claims Capture and Validation
//
// After a successful invocation, claims are captured to
// .aether/data/last-build-claims.json for verification during `aether continue`:
//
//	{
//	  "files_created": ["path/to/file1.go"],
//	  "files_modified": ["path/to/file2.go"],
//	  "build_phase": 1,
//	  "timestamp": "2026-04-16T10:30:00Z"
//	}
//
// The verification pipeline (aether verify-claims, aether continue) cross-references
// these claims against the actual filesystem to detect discrepancies between what
// the worker claimed and what actually changed.
//
// # Timeout Handling
//
// Worker invocations are bounded by a configurable timeout:
//
//   - Default: 10 minutes (600 seconds)
//   - Configurable via WorkerConfig.Timeout
//   - When the timeout is exceeded, the subprocess is killed and a
//     WorkerResult with status "failed" is returned
//   - The context.Context passed to Invoke should carry the deadline
//
// # Environment Variables
//
//   - AETHER_CODEX_REAL_DISPATCH:
//
//   - `1`, `true`, `real` => force real codex subprocess invocation
//
//   - `0`, `false`, `fake` => force FakeInvoker
//
//   - unset => RealInvoker in the normal binary, FakeInvoker under `go test`
//
//   - AETHER_CODEX_TIMEOUT: Override the default worker timeout in seconds.
//     Ignored if WorkerConfig.Timeout is explicitly set.
//
//   - AETHER_CODEX_PATH: Override the path to the codex CLI binary.
//     Defaults to "codex" (resolved via $PATH).
//
//   - AETHER_OUTPUT_MODE: Controls output rendering. Set to "json" for
//     machine-readable output, "visual" for human-friendly terminal output.
//
// # Fallback Behavior
//
// When the codex CLI is not installed (IsAvailable returns false), RealInvoker
// returns an error. FakeInvoker is only used in tests or when explicitly forced.
//
// # Worker Configuration (codexWorkerConfig)
//
// WorkerConfig specifies all parameters needed to invoke a single worker:
//
//	type codexWorkerConfig struct {
//	    AgentName      string        // TOML agent name (e.g., "aether-builder")
//	    AgentTOMLPath  string        // Absolute path to the agent's TOML file
//	    Caste          string        // Worker caste (builder, watcher, scout, etc.)
//	    WorkerName     string        // Deterministic ant name (e.g., "Hammer-23")
//	    TaskID         string        // Task identifier from the build dispatch
//	    TaskBrief      string        // The markdown task brief content
//	    ContextCapsule string        // The assembled compact colony-prime context
//	    Root           string        // Repository root directory (working dir for subprocess)
//	    Timeout        time.Duration // Per-worker timeout (default: 10 minutes)
//	    ResponsePath   string        // Optional controller-managed response file path
//	}
//
// # Worker Result (codexWorkerResult)
//
// WorkerResult captures the outcome of a worker invocation:
//
//	type codexWorkerResult struct {
//	    WorkerName    string        // The worker's assigned name
//	    Caste         string        // Worker caste
//	    TaskID        string        // Task identifier
//	    Status        string        // "completed", "failed", or "blocked" ("code_written" normalizes to "completed")
//	    Summary       string        // Worker's self-reported summary
//	    FilesCreated  []string      // Files the worker claims to have created
//	    FilesModified []string      // Files the worker claims to have modified
//	    TestsWritten  []string      // Test files the worker created
//	    ToolCount     int           // Number of tool calls reported
//	    Blockers      []string      // Blocking issues reported
//	    Spawns        []string      // Sub-workers spawned
//	    Duration      time.Duration // Wall-clock time of the invocation
//	    RawOutput     string        // Full stdout from the subprocess
//	    Error         error         // Invocation error (if any)
//	}
//
// # TOML Agent Format
//
// Agent definitions live in .codex/agents/*.toml with this schema:
//
//	name = "aether-builder"
//	description = "Use this agent for code implementation..."
//	nickname_candidates = ["builder", "hammer"]
//	developer_instructions = '''
//	You are a Builder Ant in the Aether Colony...
//	'''
//
// Required fields: name, description, developer_instructions.
// Optional fields: nickname_candidates (used by Codex CLI for agent matching).
//
// Per the Codex CLI format specification, agent role files are TOML with
// ConfigToml keys. They are declared in config.toml via [agents.<name>.config_file]
// entries, and also read directly by Aether's Go dispatch layer for prompt injection.
// The developer_instructions field is injected as a "developer" role message
// into the agent's context.
//
// # Dispatch Flow
//
// The full dispatch flow from build command to worker completion:
//
//  1. aether build <phase> computes dispatches via plannedBuildDispatches()
//  2. Each dispatch gets a unique worker name via deterministicAntName()
//  3. Worker briefs are written to .aether/data/build/phase-N/worker-briefs/
//  4. For each dispatch, the invoker:
//     a. Loads the agent's TOML file
//     b. Generates compact colony-prime context for the worker
//     c. Assembles the prompt (TOML instructions + context + brief)
//     d. Invokes the codex CLI subprocess
//     e. Parses the trailing JSON output
//     f. Returns a WorkerResult
//  5. Results are aggregated into last-build-claims.json
//  6. aether continue validates claims and advances the phase
//
// # Integration with Existing Commands
//
// This package integrates with:
//   - cmd/codex_build.go: plannedBuildDispatches(), codexBuildManifest, codexBuildClaims
//   - cmd/colony_prime_context.go: compact colony-prime assembly for Codex workers
//   - cmd/context.go: context-capsule subcommand and fallback capsule assembly
//   - cmd/codex_visuals.go: renderBuildVisualWithDispatches(), casteEmoji()
//   - cmd/verify_claims.go: verify-claims subcommand for claims validation
//   - cmd/codex_continue.go: codexClaimVerification for continue-time verification
package codex
