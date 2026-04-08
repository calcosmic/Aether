## Contributing

Aether is shaped by its community. Whether you are fixing a bug, adding a command, or improving documentation, every contribution strengthens the colony. Here is how to get started.

### Prerequisites

- **Go 1.22+** -- [Install Go](https://go.dev/dl/) if you don't have it
- **Git** -- For cloning and branching

### Development Setup

```bash
git clone https://github.com/calcosmic/Aether.git
cd Aether
make build
```

That's it. The `make build` target compiles the binary with version injection from `package.json`. You will find the `aether` binary in the project root.

### Build, Test, Lint

| Command | What it does |
|---------|-------------|
| `make build` | Compile the binary (`go build` with version ldflags) |
| `make test` | Run all tests with race detection (`go test -race -count=1 ./...`) |
| `make lint` | Static analysis with `go vet ./...` |
| `make clean` | Remove the compiled binary |
| `make install` | Build and install the binary to `$GOPATH/bin` |

Run `make test` and `make lint` before every commit. CI will do the same.

### Project Structure

```
cmd/aether/          CLI entry point (main.go)
internal/            Core logic -- commands, pheromones, state, curation, and more
  commands/          Go implementations of slash commands
commands/            YAML source definitions for agent commands (consumed by setup)
.aether/             Colony system files -- templates, skills, agent definitions, docs
```

The Go code lives under `cmd/` and `internal/`. The colony's agent definitions, skills, templates, and command YAML files live under `.aether/`.

### Contributing Workflow

1. **Fork the repo** and clone your fork locally
2. **Create a feature branch** -- `git checkout -b my-feature`
3. **Make your changes** with tests covering new behavior
4. **Run `make test` and `make lint`** -- fix anything that breaks
5. **Submit a pull request** against `main` with a clear description of the change

Keep pull requests focused. One feature or fix per PR makes review easier and history cleaner.

### Adding Commands

Aether commands are defined as YAML files in the `commands/` directory at the repo root. Each YAML file describes the command name, description, agent caste, and prompt template. The Go implementation lives in `internal/` as a matching command file.

To add a new command:

1. Create a YAML definition in `commands/`
2. Implement the Go handler in `internal/`
3. Register the command in the root command registry
4. Add tests in a `_test.go` file alongside the implementation
5. Run `make test` and `make lint`

### Code of Conduct

Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). We are committed to providing a welcoming and inclusive experience for everyone.

---

*The colony grows when new ants join. Welcome to the swarm.*
