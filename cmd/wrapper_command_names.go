package cmd

// wrapperCommandNames is the set of runtime verbs that have a platform slash
// wrapper (`/ant-<verb>` in Claude Code and OpenCode). It exists so next-step
// hints can name the command the user actually types.
//
// It is deliberately an allowlist, not a blanket `aether ` → `/ant-` rewrite:
// plenty of real commands have no wrapper (publish, install, host, spawn-log,
// build-finalize, print-next-up, flag-resolve, version). Rewriting those would
// invent commands that do not exist — worse than showing the raw CLI form.
//
// TestWrapperCommandNamesMatchCanonicalCorpus keeps this in lockstep with
// .claude/commands/ant/*.md, so adding or removing a wrapper without updating
// this map fails the build.
var wrapperCommandNames = map[string]bool{
	"abandon":         true,
	"archaeology":     true,
	"assumptions":     true,
	"build":           true,
	"bump-version":    true,
	"chaos":           true,
	"colonize":        true,
	"continue":        true,
	"council":         true,
	"data-clean":      true,
	"discuss":         true,
	"dream":           true,
	"entomb":          true,
	"export-signals":  true,
	"feedback":        true,
	"flag":            true,
	"flags":           true,
	"focus":           true,
	"help":            true,
	"history":         true,
	"import-signals":  true,
	"init":            true,
	"insert-phase":    true,
	"interpret":       true,
	"lay-eggs":        true,
	"maturity":        true,
	"medic":           true,
	"memory-details":  true,
	"migrate-state":   true,
	"oracle":          true,
	"organize":        true,
	"patrol":          true,
	"pause-colony":    true,
	"phase":           true,
	"pheromones":      true,
	"plan":            true,
	"porter":          true,
	"preferences":     true,
	"profile":         true,
	"queen-compose":   true,
	"quick":           true,
	"recover":         true,
	"redirect":        true,
	"reference-index": true,
	"reference-list":  true,
	"reference-match": true,
	"resume":          true,
	"resume-colony":   true,
	"run":             true,
	"seal":            true,
	"shelf":           true,
	"shelf-add":       true,
	"shelf-dismiss":   true,
	"shelf-list":      true,
	"shelf-promote":   true,
	"skill-create":    true,
	"status":          true,
	"swarm":           true,
	"tunnels":         true,
	"unblock":         true,
	"update":          true,
	"verify-castes":   true,
	"watch":           true,
}
