export type HostCommandRunner =
  | "go-json"
  | "dispatched"
  | "oracle-lifecycle"
  | "lifecycle"
  | "watch-display"
  | "swarm-display";

export type HostCommandCategory = "orchestrated" | "display" | "lifecycle";

export interface ParsedHostArgs {
  command: string;
  cwd: string;
  simulate: boolean;
  synthetic: boolean;
  noDashboard: boolean;
  skipMiddenCheck: boolean;
  skipWatchers: boolean;
  refresh: boolean;
  force: boolean;
  forceResurvey: boolean;
  tasks: string[];
  depth: string | undefined;
  planningDepth: string | undefined;
  verificationDepth: string | undefined;
  targetConfidence: string | undefined;
  maxIterations: string | undefined;
  accept: boolean;
  verificationTimeout: string | undefined;
  light: boolean;
  heavy: boolean;
  workerTimeout: string | undefined;
  circuitBreakerThreshold: string | undefined;
  noSuggest: boolean;
  verbose: boolean;
  reconcileTasks: string[];
  noLearn: boolean;
  classicCeremony: boolean;
  help: boolean;
  positional: string[];
  unknownFlags: string[];
}

export interface HostCommandDefinition {
  command: string;
  usage: string;
  description: string;
  category: HostCommandCategory;
  runner: HostCommandRunner;
  goPlanCommand?: string;
  finalizerCommand?: string;
  ceremonyWorkflow?: string;
  supportsDashboard: boolean;
  literalPassthrough: boolean;
  buildGoArgs?: (parsed: ParsedHostArgs) => string[];
}

function rejectUnsupportedFlags(parsed: ParsedHostArgs): void {
  if (parsed.unknownFlags.length > 0) {
    throw new Error(`Unsupported host flag(s): ${parsed.unknownFlags.join(", ")}`);
  }
}

function pushRepeatedFlag(args: string[], flag: string, values: readonly string[]): void {
  for (const value of values) {
    args.push(flag, value);
  }
}

function planArgs(parsed: ParsedHostArgs): string[] {
  rejectUnsupportedFlags(parsed);
  const args = ["plan", "--plan-only"];
  if (parsed.refresh) args.push("--refresh");
  if (parsed.force) args.push("--force");
  if (parsed.depth) args.push("--depth", parsed.depth);
  if (parsed.planningDepth) args.push("--planning-depth", parsed.planningDepth);
  if (parsed.verificationDepth) args.push("--verification-depth", parsed.verificationDepth);
  if (parsed.targetConfidence) args.push("--target", parsed.targetConfidence);
  if (parsed.maxIterations) args.push("--max-iterations", parsed.maxIterations);
  if (parsed.accept) args.push("--accept");
  if (parsed.synthetic || parsed.simulate) args.push("--synthetic");
  if (parsed.workerTimeout) args.push("--worker-timeout", parsed.workerTimeout);
  return args;
}

function buildArgs(parsed: ParsedHostArgs): string[] {
  rejectUnsupportedFlags(parsed);
  const phase = parsed.positional[0];
  if (!phase) {
    throw new Error("build requires a phase number");
  }
  const args = ["build", phase, "--plan-only"];
  pushRepeatedFlag(args, "--task", parsed.tasks);
  if (parsed.force) args.push("--force");
  if (parsed.synthetic || parsed.simulate) args.push("--synthetic");
  if (parsed.light) args.push("--light");
  if (parsed.heavy) args.push("--heavy");
  if (parsed.verificationDepth) args.push("--verification-depth", parsed.verificationDepth);
  if (parsed.workerTimeout) args.push("--worker-timeout", parsed.workerTimeout);
  if (parsed.circuitBreakerThreshold) args.push("--circuit-breaker-threshold", parsed.circuitBreakerThreshold);
  if (parsed.noSuggest) args.push("--no-suggest");
  if (parsed.verbose) args.push("--verbose");
  return args;
}

function colonizeArgs(parsed: ParsedHostArgs): string[] {
  rejectUnsupportedFlags(parsed);
  const args = ["colonize", "--plan-only"];
  if (parsed.force || parsed.forceResurvey) args.push("--force-resurvey");
  if (parsed.workerTimeout) args.push("--worker-timeout", parsed.workerTimeout);
  return args;
}

function continueArgs(parsed: ParsedHostArgs): string[] {
  rejectUnsupportedFlags(parsed);
  const args = ["continue", "--plan-only"];
  pushRepeatedFlag(args, "--reconcile-task", parsed.reconcileTasks);
  if (parsed.classicCeremony) args.push("--classic-ceremony");
  if (parsed.verificationDepth) args.push("--verification-depth", parsed.verificationDepth);
  if (parsed.verificationTimeout) args.push("--verification-timeout", parsed.verificationTimeout);
  if (parsed.light) args.push("--light");
  if (parsed.heavy) args.push("--heavy");
  if (parsed.skipWatchers) args.push("--skip-watchers");
  if (parsed.synthetic || parsed.simulate) args.push("--synthetic");
  if (parsed.workerTimeout) args.push("--worker-timeout", parsed.workerTimeout);
  if (parsed.noLearn) args.push("--no-learn");
  return args;
}

function sealArgs(parsed: ParsedHostArgs): string[] {
  rejectUnsupportedFlags(parsed);
  const args = ["seal", "--plan-only"];
  if (parsed.force) args.push("--force");
  return args;
}

export const HOST_COMMANDS: readonly HostCommandDefinition[] = [
  {
    command: "colonize",
    usage: "colonize",
    description: "Call aether colonize --plan-only",
    category: "orchestrated",
    runner: "go-json",
    goPlanCommand: "aether colonize --plan-only",
    finalizerCommand: "aether colonize-finalize --completion-file <file>",
    ceremonyWorkflow: "colonize",
    supportsDashboard: false,
    literalPassthrough: false,
    buildGoArgs: colonizeArgs,
  },
  {
    command: "plan",
    usage: "plan",
    description: "Dispatch real Scout/Route-Setter planning workers with ceremony",
    category: "orchestrated",
    runner: "dispatched",
    goPlanCommand: "aether plan --plan-only",
    finalizerCommand: "aether plan-finalize --completion-file <file>",
    ceremonyWorkflow: "plan",
    supportsDashboard: false,
    literalPassthrough: false,
    buildGoArgs: planArgs,
  },
  {
    command: "build",
    usage: "build <N>",
    description: "Dispatch real build workers in parallel waves with ceremony",
    category: "orchestrated",
    runner: "dispatched",
    goPlanCommand: "aether build <phase> --plan-only",
    finalizerCommand: "aether build-finalize <phase> --completion-file <file>",
    ceremonyWorkflow: "build",
    supportsDashboard: false,
    literalPassthrough: false,
    buildGoArgs: buildArgs,
  },
  {
    command: "continue",
    usage: "continue",
    description: "Dispatch real review workers with ceremony and continue-finalizer",
    category: "orchestrated",
    runner: "dispatched",
    goPlanCommand: "aether continue --plan-only",
    finalizerCommand: "aether continue-finalize --completion-file <file>",
    ceremonyWorkflow: "continue",
    supportsDashboard: false,
    literalPassthrough: false,
    buildGoArgs: continueArgs,
  },
  {
    command: "seal",
    usage: "seal",
    description: "Call aether seal --plan-only",
    category: "orchestrated",
    runner: "go-json",
    goPlanCommand: "aether seal --plan-only",
    finalizerCommand: "aether seal-finalize --completion-file <file>",
    ceremonyWorkflow: "seal",
    supportsDashboard: false,
    literalPassthrough: false,
    buildGoArgs: sealArgs,
  },
  {
    command: "oracle",
    usage: "oracle [topic]",
    description: "Run Oracle RALF lifecycle loop",
    category: "orchestrated",
    runner: "oracle-lifecycle",
    goPlanCommand: "aether oracle-iterate --plan-only",
    finalizerCommand: "aether oracle-iterate-finalize --completion-file <file>",
    ceremonyWorkflow: "oracle",
    supportsDashboard: true,
    literalPassthrough: false,
  },
  {
    command: "lifecycle",
    usage: "lifecycle [N] [topic]",
    description: "Run experimental simulate-only plan->build->continue smoke",
    category: "lifecycle",
    runner: "lifecycle",
    ceremonyWorkflow: "lifecycle",
    supportsDashboard: true,
    literalPassthrough: false,
  },
  {
    command: "watch",
    usage: "watch",
    description: "Show colony status, optionally with live dashboard",
    category: "display",
    runner: "watch-display",
    supportsDashboard: true,
    literalPassthrough: false,
  },
  {
    command: "swarm",
    usage: "swarm [target]",
    description: "Show swarm plan for a target problem",
    category: "display",
    runner: "swarm-display",
    goPlanCommand: "aether swarm --plan-only",
    finalizerCommand: "aether swarm-finalize --completion-file <file>",
    ceremonyWorkflow: "swarm",
    supportsDashboard: true,
    literalPassthrough: false,
  },
];

export const HOST_COMMAND_NAMES = HOST_COMMANDS.map((definition) => definition.command);

export function listHostCommandDefinitions(): readonly HostCommandDefinition[] {
  return HOST_COMMANDS;
}

export function getHostCommandDefinition(command: string): HostCommandDefinition | undefined {
  return HOST_COMMANDS.find((definition) => definition.command === command);
}

export function buildHostGoArgs(parsed: ParsedHostArgs): string[] | undefined {
  return getHostCommandDefinition(parsed.command)?.buildGoArgs?.(parsed);
}
